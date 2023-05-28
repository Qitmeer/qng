/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/consensus/vm"
	qtypes "github.com/Qitmeer/qng/core/types"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/eth"
	mparams "github.com/Qitmeer/qng/meerevm/params"
	"github.com/Qitmeer/qng/rpc/api"
	qconsensus "github.com/Qitmeer/qng/vm/consensus"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"reflect"
)

type MeerChain struct {
	chain     *eth.ETHChain
	meerpool  *MeerPool
	consensus model.Consensus
}

func (b *MeerChain) CheckConnectBlock(block qconsensus.Block) error {
	parent := b.chain.Ether().BlockChain().CurrentBlock()
	_, _, _, err := b.buildBlock(parent, block.Transactions(), block.Timestamp().Unix())
	if err != nil {
		return err
	}
	return nil
}

func (b *MeerChain) ConnectBlock(block qconsensus.Block) (uint64, error) {
	parent := b.chain.Ether().BlockChain().CurrentBlock()
	mblock, _, _, err := b.buildBlock(parent, block.Transactions(), block.Timestamp().Unix())
	if err != nil {
		return 0, err
	}
	var st int
	st, err = b.chain.Ether().BlockChain().InsertChain(types.Blocks{mblock})
	if err != nil {
		return 0, err
	}
	if st != 1 {
		return 0, fmt.Errorf("BuildBlock error")
	}
	//
	mbh := qcommon.ToEVMHash(block.ID())
	//
	log.Debug(fmt.Sprintf("MeerEVM Block:number=%d hash=%s txs=%d  => blockHash(%s) txs=%d", mblock.Number().Uint64(), mblock.Hash().String(), len(mblock.Transactions()), mbh.String(), len(block.Transactions())))

	return mblock.NumberU64(), nil
}

func (b *MeerChain) buildBlock(parent *types.Header, qtxs []model.Tx, timestamp int64) (*types.Block, types.Receipts, *state.StateDB, error) {
	config := b.chain.Config().Eth.Genesis.Config
	engine := b.chain.Ether().Engine()
	parentBlock := types.NewBlockWithHeader(parent)

	uncles := []*types.Header{}

	chainreader := &fakeChainReader{config: config}

	statedb, err := b.chain.Ether().BlockChain().StateAt(parentBlock.Root())
	if err != nil {
		return nil, nil, nil, err
	}
	gaslimit := core.CalcGasLimit(parentBlock.GasLimit(), b.meerpool.config.GasCeil)

	// --------Will be discard in the future --------------------
	if config.ChainID.Int64() == mparams.QngMainnetChainConfig.ChainID.Int64() &&
		!forks.IsGasLimitForkHeight(parent.Number.Int64()) {
		gaslimit = 0x10000000000000
	}
	// ----------------------------------------------------------

	header := makeHeader(&b.chain.Config().Eth, parentBlock, statedb, timestamp, gaslimit)

	if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(header.Number) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	txs, receipts, err := b.fillBlock(qtxs, header, statedb)
	if err != nil {
		return nil, nil, nil, err
	}

	block, err := engine.FinalizeAndAssemble(chainreader, header, statedb, txs, uncles, receipts, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	return block, receipts, statedb, nil
}

func (b *MeerChain) fillBlock(qtxs []model.Tx, header *types.Header, statedb *state.StateDB) ([]*types.Transaction, []*types.Receipt, error) {
	txs := []*types.Transaction{}
	receipts := []*types.Receipt{}

	header.Coinbase = b.chain.Config().Eth.Miner.Etherbase
	for _, tx := range qtxs {
		if tx.GetTxType() == qtypes.TxTypeCrossChainVM ||
			tx.GetTxType() == qtypes.TxTypeCrossChainImport {
			pubkBytes, err := hex.DecodeString(tx.GetTo())
			if err != nil {
				return nil, nil, err
			}
			publicKey, err := crypto.UnmarshalPubkey(pubkBytes)
			if err != nil {
				return nil, nil, err
			}
			toAddr := crypto.PubkeyToAddress(*publicKey)
			header.Coinbase = toAddr
			break
		}
	}

	gasPool := new(core.GasPool).AddGas(header.GasLimit)

	for _, tx := range qtxs {
		if tx.GetTxType() == qtypes.TxTypeCrossChainExport {
			pubkBytes, err := hex.DecodeString(tx.GetTo())
			if err != nil {
				return nil, nil, err
			}
			publicKey, err := crypto.UnmarshalPubkey(pubkBytes)
			if err != nil {
				return nil, nil, err
			}

			value := big.NewInt(int64(tx.GetValue()))
			value = value.Mul(value, qcommon.Precision)
			toAddr := crypto.PubkeyToAddress(*publicKey)
			txData := &types.AccessListTx{
				To:    &toAddr,
				Value: value,
				Nonce: uint64(tx.GetTxType()),
			}
			etx := types.NewTx(txData)
			txmb, err := etx.MarshalBinary()
			if err != nil {
				return nil, nil, err
			}
			if len(header.Extra) > 0 {
				return nil, nil, fmt.Errorf("import and export tx conflict")
			}
			header.Extra = txmb
		} else if tx.GetTxType() == qtypes.TxTypeCrossChainImport {
			pubkBytes, err := hex.DecodeString(tx.GetFrom())
			if err != nil {
				return nil, nil, err
			}
			publicKey, err := crypto.UnmarshalPubkey(pubkBytes)
			if err != nil {
				return nil, nil, err
			}

			toAddr := crypto.PubkeyToAddress(*publicKey)

			value := big.NewInt(int64(tx.GetValue()))
			value = value.Mul(value, qcommon.Precision)
			txData := &types.AccessListTx{
				To:    &toAddr,
				Value: value,
				Nonce: uint64(tx.GetTxType()),
			}
			etx := types.NewTx(txData)
			txmb, err := etx.MarshalBinary()
			if err != nil {
				return nil, nil, err
			}
			if len(header.Extra) > 0 {
				return nil, nil, fmt.Errorf("import and export tx conflict")
			}
			header.Extra = txmb
		} else if tx.GetTxType() == qtypes.TxTypeCrossChainVM {
			txb := tx.GetData()
			var txmb = &types.Transaction{}
			if err := txmb.UnmarshalBinary(txb); err != nil {
				return nil, nil, err
			}
			err := b.addTx(txmb, header, statedb, &txs, &receipts, gasPool)
			if err != nil {
				return nil, nil, err
			}
		}

	}

	return txs, receipts, nil
}

func (b *MeerChain) addTx(tx *types.Transaction, header *types.Header, statedb *state.StateDB, txs *[]*types.Transaction, receipts *[]*types.Receipt, gasPool *core.GasPool) error {
	config := b.chain.Config().Eth.Genesis.Config
	statedb.SetTxContext(tx.Hash(), len(*txs))

	bc := b.chain.Ether().BlockChain()
	snap := statedb.Snapshot()

	receipt, err := core.ApplyTransaction(config, bc, &header.Coinbase, gasPool, statedb, header, tx, &header.GasUsed, *bc.GetVMConfig())
	if err != nil {
		statedb.RevertToSnapshot(snap)
		return err
	}

	*txs = append(*txs, tx)
	*receipts = append(*receipts, receipt)

	return nil
}

func (b *MeerChain) RegisterAPIs(apis []api.API) {
	eapis := []rpc.API{}

	for _, api := range apis {
		eapi := rpc.API{
			Namespace: "qng",
			Version:   "1.0",
			Service:   api.Service,
			Public:    api.Public,
		}
		eapis = append(eapis, eapi)

		log.Trace(fmt.Sprintf("Bridging API:%s.%s in QNG => qng.%s in MeerEVM", api.NameSpace, reflect.TypeOf(api.Service).Elem(), reflect.TypeOf(api.Service).Elem()))
	}
	b.chain.Node().RegisterAPIs(eapis)
}

func (b *MeerChain) Start() {
	b.meerpool.Start()
}

func (b *MeerChain) Stop() {
	b.meerpool.Stop()
}

func (b *MeerChain) MeerPool() *MeerPool {
	return b.meerpool
}

func (b *MeerChain) ETHChain() *eth.ETHChain {
	return b.chain
}

func (b *MeerChain) prepareEnvironment(state model.BlockState) (*types.Header, error) {
	curBlockHeader := b.chain.Ether().BlockChain().CurrentBlock()
	if curBlockHeader.Number.Uint64() > state.GetEVMNumber() {
		err := b.RewindTo(state)
		if err != nil {
			return nil, err
		}
		curBlockHeader = b.chain.Ether().BlockChain().CurrentBlock()
	}
	if curBlockHeader.Hash() == state.GetEVMHash() &&
		curBlockHeader.Number.Uint64() == state.GetEVMNumber() {
		return curBlockHeader, nil
	}
	getError := func(msg string) error {
		return fmt.Errorf("meer chain env error:targetEVM.number=%d, targetEVM.hash=%s, targetState.order=%d, cur.number=%d, cur.hash=%s, %s", state.GetEVMNumber(), state.GetEVMHash().String(), state.GetOrder(), curBlockHeader.Number, curBlockHeader.Hash().String(), msg)
	}
	if state.GetOrder() <= 0 {
		return nil, getError("reach genesis")
	}
	log.Info("Start to find cur block state", "state.order", state.GetOrder(), "evm.Number", state.GetEVMNumber(), "cur.number", curBlockHeader.Number.Uint64())
	var curBlockState model.BlockState
	list := []model.BlockState{state}
	startState := b.consensus.BlockChain().GetBlockState(state.GetOrder() - 1)
	for startState != nil && startState.GetEVMNumber() >= curBlockHeader.Number.Uint64() {
		if startState.GetEVMNumber() == curBlockHeader.Number.Uint64() &&
			startState.GetEVMHash() == curBlockHeader.Hash() {
			curBlockState = startState
			break
		}
		list = append(list, startState)
		if startState.GetOrder() <= 0 {
			break
		}
		startState = b.consensus.BlockChain().GetBlockState(startState.GetOrder() - 1)
	}
	if curBlockState == nil {
		return nil, getError("Can't find cur block state")
	}
	log.Info("Find cur block state", "state.order", curBlockState.GetOrder(), "evm.Number", curBlockState.GetEVMNumber())
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].GetStatus().KnownInvalid() {
			continue
		}
		cur := b.chain.Ether().BlockChain().CurrentBlock()
		if list[i].GetEVMNumber() == cur.Number.Uint64() {
			continue
		}
		log.Info("Try to restore block state for EVM", "evm.hash", list[i].GetEVMHash().String(), "evm.number", list[i].GetEVMNumber(), "state.order", list[i].GetOrder())
		block := b.chain.Ether().BlockChain().GetBlock(list[i].GetEVMHash(), list[i].GetEVMNumber())
		if block != nil {
			log.Info("Try to rebuild evm block", "state.order", list[i].GetOrder())
			sb, err := b.consensus.BlockChain().BlockByOrder(list[i].GetOrder())
			if err != nil {
				return nil, getError(err.Error())
			}
			dtxs := list[i].GetDuplicateTxs()
			if len(dtxs) > 0 {
				for _, index := range dtxs {
					sb.Transactions()[index].IsDuplicate = true
				}
			}

			eb, err := vm.BuildEVMBlock(sb)
			if err != nil {
				return nil, getError(err.Error())
			}
			if len(eb.Transactions()) <= 0 {
				return nil, getError("transactions is empty")
			}
			block, _, _, err = b.buildBlock(cur, eb.Transactions(), eb.Timestamp().Unix())
			if err != nil {
				return nil, getError(err.Error())
			}
		}
		st, err := b.chain.Ether().BlockChain().InsertChain(types.Blocks{block})
		if err != nil {
			return nil, err
		}
		if st != 1 {
			return nil, getError("insert chain")
		}
	}
	cur := b.chain.Ether().BlockChain().CurrentBlock()
	if cur.Hash() == state.GetEVMHash() &&
		cur.Number.Uint64() == state.GetEVMNumber() {
		return cur, nil
	}
	return nil, getError("prepare environment")
}

func (b *MeerChain) PrepareEnvironment(state model.BlockState) (*types.Header, error) {
	return b.prepareEnvironment(state)
}

func (b *MeerChain) RewindTo(state model.BlockState) error {
	curBlockHeader := b.chain.Ether().BlockChain().CurrentBlock()
	if curBlockHeader.Number.Uint64() <= state.GetEVMNumber() {
		return nil
	}
	log.Info("Try to rewind", "cur.number", curBlockHeader.Number.Uint64(), "cur.hash", curBlockHeader.Hash().String(), "target.evm.root", state.GetEVMRoot(), "target.evm.number", state.GetEVMNumber(), "target.evm.hash", state.GetEVMHash())
	err := b.chain.Ether().BlockChain().SetHead(state.GetEVMNumber())
	if err != nil {
		return err
	}
	cur := b.chain.Ether().BlockChain().CurrentBlock()
	if cur.Number.Uint64() <= state.GetEVMNumber() {
		log.Info("Rewound", "cur.number", cur.Number.Uint64(), "cur.hash", cur.Hash().String(), "target.evm.root", state.GetEVMRoot(), "target.evm.number", state.GetEVMNumber(), "target.evm.hash", state.GetEVMHash())
		return nil
	}
	return fmt.Errorf("Rewind fail:cur.number=%d, cur.hash=%s, target.evm.root=%s, target.evm.number=%d, target.evm.hash=%s", cur.Number.Uint64(), cur.Hash().String(), state.GetEVMRoot(), state.GetEVMNumber(), state.GetEVMHash())
}

func NewMeerChain(ctx qconsensus.Context) (*MeerChain, error) {
	cfg := ctx.GetConfig()
	eth.InitLog(cfg.DebugLevel, cfg.DebugPrintOrigins)
	//
	ecfg, args, err := MakeParams(cfg)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	chain, err := eth.NewETHChain(ecfg, args)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	mc := &MeerChain{
		chain:     chain,
		meerpool:  chain.Config().Eth.Miner.External.(*MeerPool),
		consensus: ctx.GetConsensus(),
	}
	mc.meerpool.init(&chain.Config().Eth.Miner, chain.Config().Eth.Genesis.Config, chain.Ether().Engine(), chain.Ether(), chain.Ether().EventMux(), ctx)
	return mc, nil
}

func makeHeader(cfg *ethconfig.Config, parent *types.Block, state *state.StateDB, timestamp int64, gaslimit uint64) *types.Header {
	ptt := int64(parent.Time())
	if timestamp <= ptt {
		timestamp = ptt + 1
	}

	header := &types.Header{
		Root:       state.IntermediateRoot(cfg.Genesis.Config.IsEIP158(parent.Number())),
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: common.Big1,
		GasLimit:   gaslimit,
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Time:       uint64(timestamp),
	}
	if cfg.Genesis.Config.IsLondon(header.Number) {
		header.BaseFee = misc.CalcBaseFee(cfg.Genesis.Config, parent.Header())
		if !cfg.Genesis.Config.IsLondon(parent.Number()) {
			parentGasLimit := parent.GasLimit() * cfg.Genesis.Config.ElasticityMultiplier()
			header.GasLimit = core.CalcGasLimit(parentGasLimit, parentGasLimit)
		}
	}
	return header
}

type fakeChainReader struct {
	config *params.ChainConfig
}

// Config returns the chain configuration.
func (cr *fakeChainReader) Config() *params.ChainConfig {
	return cr.config
}

func (cr *fakeChainReader) CurrentHeader() *types.Header                            { return nil }
func (cr *fakeChainReader) GetHeaderByNumber(number uint64) *types.Header           { return nil }
func (cr *fakeChainReader) GetHeaderByHash(hash common.Hash) *types.Header          { return nil }
func (cr *fakeChainReader) GetHeader(hash common.Hash, number uint64) *types.Header { return nil }
func (cr *fakeChainReader) GetBlock(hash common.Hash, number uint64) *types.Block   { return nil }
func (cr *fakeChainReader) GetTd(hash common.Hash, number uint64) *big.Int          { return nil }
