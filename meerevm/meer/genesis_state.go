package meer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"math/big"
)

var (
	SysContractDeployerAddress = common.Address{}
)

type Alloc map[common.Address]core.GenesisAccount

func (g Alloc) OnRoot(common.Hash) {}

func (g Alloc) OnAccount(addr *common.Address, dumpAccount state.DumpAccount) {
	balance, _ := new(big.Int).SetString(dumpAccount.Balance, 10)
	var storage map[common.Hash]common.Hash
	if dumpAccount.Storage != nil {
		storage = make(map[common.Hash]common.Hash)
		for k, v := range dumpAccount.Storage {
			storage[k] = common.HexToHash(v)
		}
	}
	genesisAccount := core.GenesisAccount{
		Code:    dumpAccount.Code,
		Storage: storage,
		Balance: balance,
		Nonce:   dumpAccount.Nonce,
	}
	g[*addr] = genesisAccount
}

type GenTransaction struct {
	*types.Transaction
	From common.Address
}

func Apply(genesis *core.Genesis, txs []*GenTransaction) (Alloc, error) {
	if genesis.Config.IsLondon(big.NewInt(int64(0))) {
		if genesis.BaseFee == nil {
			return nil, fmt.Errorf("EIP-1559 config but missing 'currentBaseFee' in env section")
		}
	}

	chainConfig := genesis.Config
	getHash := func(num uint64) common.Hash {
		return common.Hash{}
	}
	var (
		statedb     = MakePreState(rawdb.NewMemoryDatabase(), genesis.Alloc)
		gaspool     = new(core.GasPool)
		blockHash   = common.Hash{0x13, 0x37}
		includedTxs types.Transactions
		gasUsed     = uint64(0)
		receipts    = make(types.Receipts, 0)
		txIndex     = 0
		signer      = types.MakeSigner(chainConfig, new(big.Int).SetUint64(0), 0)
	)

	gaspool.AddGas(genesis.GasLimit)
	vmContext := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    genesis.Coinbase,
		BlockNumber: new(big.Int).SetUint64(0),
		Time:        genesis.Timestamp,
		Difficulty:  genesis.Difficulty,
		GasLimit:    genesis.GasLimit,
		GetHash:     getHash,
	}
	// If currentBaseFee is defined, add it to the vmContext.
	if genesis.BaseFee != nil {
		vmContext.BaseFee = new(big.Int).Set(genesis.BaseFee)
	}
	// If DAO is supported/enabled, we need to handle it here. In geth 'proper', it's
	// done in StateProcessor.Process(block, ...), right before transactions are applied.
	if chainConfig.DAOForkSupport &&
		chainConfig.DAOForkBlock != nil &&
		chainConfig.DAOForkBlock.Cmp(new(big.Int).SetUint64(0)) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	vmConfig := vm.Config{
		Tracer: nil,
	}
	for i, tx := range txs {
		msg, err := core.TransactionToMessage(tx.Transaction, signer, genesis.BaseFee)
		if err != nil {
			log.Warn("rejected tx", "index", i, "hash", tx.Hash(), "error", err)
			return nil, err
		}
		statedb.SetTxContext(tx.Hash(), txIndex)
		txContext := core.NewEVMTxContext(msg)
		snapshot := statedb.Snapshot()
		evm := vm.NewEVM(vmContext, txContext, statedb, chainConfig, vmConfig)

		// (ret []byte, usedGas uint64, failed bool, err error)
		msgResult, err := core.ApplyMessage(evm, msg, gaspool)
		if err != nil {
			statedb.RevertToSnapshot(snapshot)
			log.Error(fmt.Sprintf("rejected tx index:%d hash:%s from:%s error:%s", i, tx.Hash(), msg.From, err))
			return nil, err
		}
		includedTxs = append(includedTxs, tx.Transaction)

		gasUsed += msgResult.UsedGas

		// Receipt:
		{
			var root []byte
			if chainConfig.IsByzantium(vmContext.BlockNumber) {
				statedb.Finalise(true)
			} else {
				root = statedb.IntermediateRoot(chainConfig.IsEIP158(vmContext.BlockNumber)).Bytes()
			}

			// Create a new receipt for the transaction, storing the intermediate root and
			// gas used by the tx.
			receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: gasUsed}
			if msgResult.Failed() {
				receipt.Status = types.ReceiptStatusFailed
			} else {
				receipt.Status = types.ReceiptStatusSuccessful
			}
			receipt.TxHash = tx.Hash()
			receipt.GasUsed = msgResult.UsedGas

			// If the transaction created a contract, store the creation address in the receipt.
			if msg.To == nil {
				receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
			}

			// Set the receipt logs and create the bloom filter.
			receipt.Logs = statedb.GetLogs(tx.Hash(), vmContext.BlockNumber.Uint64(), blockHash)
			receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
			// These three are non-consensus fields:
			//receipt.BlockHash
			//receipt.BlockNumber
			receipt.TransactionIndex = uint(txIndex)
			receipts = append(receipts, receipt)
		}

		txIndex++
	}
	statedb.IntermediateRoot(chainConfig.IsEIP158(vmContext.BlockNumber))
	// Commit block
	_, err := statedb.Commit(vmContext.BlockNumber.Uint64(), chainConfig.IsEIP158(vmContext.BlockNumber))
	if err != nil {
		return nil, fmt.Errorf("could not commit state: %v", err)
	}

	collector := make(Alloc)
	statedb.DumpToCollector(collector, nil)
	return collector, nil
}

func MakePreState(db ethdb.Database, accounts core.GenesisAlloc) *state.StateDB {
	sdb := state.NewDatabase(db)
	statedb, _ := state.New(common.Hash{}, sdb, nil)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code)
		statedb.SetNonce(addr, a.Nonce)
		statedb.SetBalance(addr, uint256.MustFromBig(a.Balance), tracing.BalanceIncreaseGenesisBalance)
		for k, v := range a.Storage {
			statedb.SetState(addr, k, v)
		}
	}
	// Commit and re-open to start with a clean state.
	root, _ := statedb.Commit(0, false)
	statedb, _ = state.New(root, sdb, nil)
	return statedb
}

func UpdateAlloc(genesis *core.Genesis, contracts []Contract) error {
	// tx
	auth, err := NewTransactorWithChainID(SysContractDeployerAddress, genesis.Config.ChainID)
	if err != nil {
		return err
	}
	auth.Nonce = big.NewInt(int64(0))
	auth.Value = big.NewInt(0)                     // in wei
	auth.GasLimit = uint64(params.GenesisGasLimit) // in units
	auth.GasPrice = big.NewInt(0)

	txs := []*GenTransaction{}
	for _, con := range contracts {
		if len(con.BIN) <= 0 {
			continue
		}
		metaData := &bind.MetaData{
			ABI: con.ABI,
			Bin: con.BIN,
		}

		bytecode := common.FromHex(metaData.Bin)
		txData := bytecode

		if len(con.Input) > 0 {
			log.Info(fmt.Sprintf("input:%s", con.Input))
			input, err := hex.DecodeString(con.Input)
			if err != nil {
				return err
			}
			txData = append(txData, input...)
		}

		tx, err := transact(auth, txData)
		if err != nil {
			return err
		}
		txs = append(txs, tx)
		address := crypto.CreateAddress(auth.From, tx.Nonce())

		log.Info(fmt.Sprintf("Contract address:%s  tx hash:%s", address, tx.Hash().Hex()))
	}

	alloc, err := Apply(genesis, txs)
	if err != nil {
		return err
	}
	genesis.Alloc = core.GenesisAlloc(alloc)

	b, err := json.MarshalIndent(alloc, "", " ")
	if err != nil {
		return err
	}
	log.Info(string(b))

	return nil
}

func transact(opts *bind.TransactOpts, input []byte) (*GenTransaction, error) {
	value := opts.Value
	nonce := opts.Nonce.Uint64()
	if opts.GasPrice != nil && (opts.GasFeeCap != nil || opts.GasTipCap != nil) {
		return nil, errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}
	gasLimit := opts.GasLimit
	baseTx := &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: opts.GasPrice,
		Gas:      gasLimit,
		Value:    value,
		Data:     input,
	}
	return &GenTransaction{
		Transaction: types.NewTx(baseTx),
		From:        opts.From,
	}, nil
}

func NewTransactorWithChainID(addr common.Address, chainID *big.Int) (*bind.TransactOpts, error) {
	if chainID == nil {
		return nil, bind.ErrNoChainID
	}
	return &bind.TransactOpts{
		From:    addr,
		Context: context.Background(),
	}, nil
}
