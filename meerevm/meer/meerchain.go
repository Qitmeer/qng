/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/consensus/model"
	qtypes "github.com/Qitmeer/qng/core/types"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	qparams "github.com/Qitmeer/qng/params"
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

const BLOCK_GASLIMIT = 0x10000000000000

type MeerChain struct {
	chain    *ETHChain
	meerpool *MeerPool
}

func (b *MeerChain) CheckConnectBlock(block qconsensus.Block) error {
	_, _, _, err := b.buildBlock(block.Transactions(), block.Timestamp().Unix())
	if err != nil {
		return err
	}
	return nil
}

func (b *MeerChain) ConnectBlock(block qconsensus.Block) (uint64, error) {
	var err error
	mblock, _, _, err := b.buildBlock(block.Transactions(), block.Timestamp().Unix())
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
	WriteBlockNumber(b.chain.Ether().ChainDb(), mbh, mblock.NumberU64())
	//
	log.Debug(fmt.Sprintf("MeerEVM Block:number=%d hash=%s txs=%d  => blockHash(%s) txs=%d", mblock.Number().Uint64(), mblock.Hash().String(), len(mblock.Transactions()), mbh.String(), len(block.Transactions())))

	return mblock.NumberU64(), nil
}

func (b *MeerChain) DisconnectBlock(block qconsensus.Block) (uint64, error) {
	curBlock := b.chain.Ether().BlockChain().CurrentBlock()
	if curBlock == nil {
		log.Error("Can't find current block")
		return 0, nil
	}

	mbh := qcommon.ToEVMHash(block.ID())

	bn := ReadBlockNumber(b.chain.Ether().ChainDb(), mbh)
	if bn == nil {
		return 0, nil
	}
	defer func() {
		DeleteBlockNumber(b.chain.Ether().ChainDb(), mbh)
	}()

	if *bn > curBlock.NumberU64() {
		return *bn, nil
	}
	parentNumber := *bn - 1
	err := b.chain.Ether().BlockChain().SetHead(parentNumber)
	if err != nil {
		log.Error(err.Error())
		return *bn, nil
	}
	newParent := b.chain.Ether().BlockChain().CurrentBlock()
	if newParent == nil {
		log.Error("Can't find current block")
		return *bn, nil
	}
	log.Debug(fmt.Sprintf("Reorganize:%s(%d) => %s(%d)", curBlock.Hash().String(), curBlock.NumberU64(), newParent.Hash().String(), newParent.NumberU64()))
	return *bn, nil
}

func (b *MeerChain) buildBlock(qtxs []model.Tx, timestamp int64) (*types.Block, types.Receipts, *state.StateDB, error) {
	config := b.chain.Config().Eth.Genesis.Config
	engine := b.chain.Ether().Engine()
	parent := b.chain.Ether().BlockChain().CurrentBlock()

	uncles := []*types.Header{}

	chainreader := &fakeChainReader{config: config}

	statedb, err := b.chain.Ether().BlockChain().StateAt(parent.Root())
	if err != nil {
		return nil, nil, nil, err
	}

	header := makeHeader(&b.chain.Config().Eth, parent, statedb, timestamp, qparams.ActiveNetParams.MeerEVMCfg.GasLimit)

	if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(header.Number) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	txs, receipts, err := b.fillBlock(qtxs, header, statedb)
	if err != nil {
		return nil, nil, nil, err
	}

	block, err := engine.FinalizeAndAssemble(chainreader, header, statedb, txs, uncles, receipts)
	if err != nil {
		return nil, nil, nil, err
	}
	return block, receipts, statedb, nil
}

func (b *MeerChain) fillBlock(qtxs []model.Tx, header *types.Header, statedb *state.StateDB) ([]*types.Transaction, []*types.Receipt, error) {
	txs := []*types.Transaction{}
	receipts := []*types.Receipt{}

	header.Coinbase = b.chain.config.Eth.Miner.Etherbase
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
	b.meerpool.Start(b.chain.config.Eth.Miner.Etherbase)
}

func (b *MeerChain) Stop() {
	b.meerpool.Stop()
}

func (b *MeerChain) MeerPool() *MeerPool {
	return b.meerpool
}

func NewMeerChain(chain *ETHChain, ctx qconsensus.Context) *MeerChain {
	mc := &MeerChain{
		chain:    chain,
		meerpool: chain.config.Eth.Miner.External.(*MeerPool),
	}
	mc.meerpool.init(&chain.config.Eth.Miner, chain.config.Eth.Genesis.Config, chain.ether.Engine(), chain.ether, chain.ether.EventMux(), ctx)
	return mc
}

func makeHeader(cfg *ethconfig.Config, parent *types.Block, state *state.StateDB, timestamp int64, gaslimit uint64) *types.Header {
	ptt := int64(parent.Time())
	if timestamp <= ptt {
		timestamp = ptt + 1
	}
	if gaslimit <= 0 {
		gaslimit = BLOCK_GASLIMIT
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
func (cr *fakeChainReader) GetTd(hash common.Hash, number uint64) *big.Int          { return nil }
