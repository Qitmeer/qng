/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package chain

import (
	"encoding/hex"
	"fmt"
	qcommon "github.com/Qitmeer/meerevm/common"
	qconsensus "github.com/Qitmeer/qng-core/consensus"
	qtypes "github.com/Qitmeer/qng-core/core/types"
	"github.com/Qitmeer/qng-core/rpc/api"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"reflect"
)

type MeerChain struct {
	chain *ETHChain

	parent *types.Block
}

func (b *MeerChain) CheckConnectBlock(block qconsensus.Block) error {
	mblock, _, err := b.buildBlock(block.Transactions())

	if err != nil {
		return err
	}

	num, err := b.chain.Ether().BlockChain().InsertChainWithoutSealVerification(mblock)
	if err != nil {
		return err
	}
	if num != 1 {
		return fmt.Errorf("BuildBlock error")
	}
	return nil
}

func (b *MeerChain) ConnectBlock(block qconsensus.Block) error {

	mblock, _, err := b.buildBlock(block.Transactions())

	if err != nil {
		return err
	}

	num, err := b.chain.Ether().BlockChain().InsertChainWithoutSealVerification(mblock)
	if err != nil {
		return err
	}
	if num != 1 {
		return fmt.Errorf("BuildBlock error")
	}

	b.parent = mblock

	//
	mbhb := block.ID().Bytes()
	qcommon.ReverseBytes(&mbhb)
	mbh := common.BytesToHash(mbhb)
	//
	WriteBlockNumber(b.chain.Ether().ChainDb(), mbh, mblock.NumberU64())
	//
	log.Debug(fmt.Sprintf("MeerEVM Block:number=%d hash=%s txs=%d  => blockHash(%s) txs=%d", mblock.Number().Uint64(), mblock.Hash().String(), len(mblock.Transactions()), mbh.String(), len(block.Transactions())))

	return nil
}

func (b *MeerChain) DisconnectBlock(block qconsensus.Block) error {
	if b.parent == nil {
		return nil
	}
	mbhb := block.ID().Bytes()
	qcommon.ReverseBytes(&mbhb)
	mbh := common.BytesToHash(mbhb)

	bn := ReadBlockNumber(b.chain.Ether().ChainDb(), mbh)
	if bn == nil {
		return nil
	}
	defer func() {
		DeleteBlockNumber(b.chain.Ether().ChainDb(), mbh)
	}()

	if *bn > b.parent.NumberU64() {
		return nil
	}

	var newParent *types.Block
	if *bn == b.parent.NumberU64() {
		newParent = b.chain.Ether().BlockChain().GetBlockByHash(b.parent.ParentHash())

	} else {
		newParent = b.chain.Ether().BlockChain().GetBlockByNumber(*bn)
		newParent = b.chain.Ether().BlockChain().GetBlockByHash(newParent.ParentHash())
	}

	if newParent == nil {
		return fmt.Errorf("Can't find %v in meerevm", b.parent.ParentHash().String())
	}

	log.Debug(fmt.Sprintf("Reorganize:%s(%d) => %s(%d)", b.parent.Hash().String(), b.parent.NumberU64(), newParent.Hash().String(), newParent.NumberU64()))
	b.parent = newParent
	return nil
}

func (b *MeerChain) buildBlock(qtxs []qconsensus.Tx) (*types.Block, types.Receipts, error) {
	config := b.chain.Config().Eth.Genesis.Config
	engine := b.chain.Ether().Engine()
	db := b.chain.Ether().ChainDb()

	if b.parent == nil {
		b.parent = b.chain.Ether().BlockChain().CurrentBlock()
	}

	uncles := []*types.Header{}

	chainreader := &fakeChainReader{config: config}

	statedb, err := state.New(b.parent.Root(), state.NewDatabase(db), nil)
	if err != nil {
		return nil, nil, err
	}

	header := makeHeader(chainreader, b.parent, statedb, engine)

	if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(header.Number) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	txs, receipts, err := b.fillBlock(qtxs, header, statedb)
	if err != nil {
		return nil, nil, err
	}

	block, err := engine.FinalizeAndAssemble(chainreader, header, statedb, txs, uncles, receipts)
	if err != nil {
		return nil, nil, err
	}

	root, err := statedb.Commit(config.IsEIP158(header.Number))
	if err != nil {
		return nil, nil, fmt.Errorf(fmt.Sprintf("state write error: %v", err))
	}
	if err := statedb.Database().TrieDB().Commit(root, false, nil); err != nil {
		return nil, nil, fmt.Errorf(fmt.Sprintf("trie write error: %v", err))
	}
	return block, receipts, nil
}

func (b *MeerChain) fillBlock(qtxs []qconsensus.Tx, header *types.Header, statedb *state.StateDB) ([]*types.Transaction, []*types.Receipt, error) {
	txs := []*types.Transaction{}
	receipts := []*types.Receipt{}

	header.Coinbase = b.chain.config.Eth.Miner.Etherbase
	for _, tx := range qtxs {
		if tx.GetTxType() == qtypes.TxTypeCrossChainVM {
			txb := common.FromHex(string(tx.GetData()))
			var txmb = &types.Transaction{}
			if err := txmb.UnmarshalBinary(txb); err != nil {
				return nil, nil, err
			}
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
			txb := common.FromHex(string(tx.GetData()))
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
	statedb.Prepare(tx.Hash(), len(*txs))
	receipt, err := core.ApplyTransaction(config, nil, &header.Coinbase, gasPool, statedb, header, tx, &header.GasUsed, vm.Config{})
	if err != nil {
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

func NewMeerChain(chain *ETHChain) *MeerChain {
	mc := &MeerChain{chain: chain}
	return mc
}

func makeHeader(chain consensus.ChainReader, parent *types.Block, state *state.StateDB, engine consensus.Engine) *types.Header {
	var time uint64
	if parent.Time() == 0 {
		time = 10
	} else {
		time = parent.Time() + 10 // block time is fixed at 10 seconds
	}
	header := &types.Header{
		Root:       state.IntermediateRoot(chain.Config().IsEIP158(parent.Number())),
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: engine.CalcDifficulty(chain, time, &types.Header{
			Number:     parent.Number(),
			Time:       time - 10,
			Difficulty: parent.Difficulty(),
			UncleHash:  parent.UncleHash(),
		}),
		GasLimit: parent.GasLimit(),
		Number:   new(big.Int).Add(parent.Number(), common.Big1),
		Time:     time,
	}
	if chain.Config().IsLondon(header.Number) {
		header.BaseFee = misc.CalcBaseFee(chain.Config(), parent.Header())
		if !chain.Config().IsLondon(parent.Number()) {
			parentGasLimit := parent.GasLimit() * params.ElasticityMultiplier
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
