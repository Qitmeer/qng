package meer

import (
	"github.com/Qitmeer/qng/consensus/model"
	mmeer "github.com/Qitmeer/qng/consensus/model/meer"
	qtypes "github.com/Qitmeer/qng/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

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

func BuildEVMBlock(block *qtypes.SerializedBlock) (*mmeer.Block, error) {
	result := &mmeer.Block{Id: block.Hash(), Txs: []model.Tx{}, Time: block.Block().Header.Timestamp}

	for idx, tx := range block.Transactions() {
		if idx == 0 {
			continue
		}
		if tx.IsDuplicate {
			continue
		}

		if qtypes.IsCrossChainExportTx(tx.Tx) {
			if tx.Object != nil {
				result.Txs = append(result.Txs, tx.Object.(*mmeer.ExportTx))
				continue
			}
			ctx, err := mmeer.NewExportTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		} else if qtypes.IsCrossChainImportTx(tx.Tx) {
			ctx, err := mmeer.NewImportTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			err = ctx.SetCoinbaseTx(block.Transactions()[0].Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		} else if qtypes.IsCrossChainVMTx(tx.Tx) {
			if tx.Object != nil {
				vt := tx.Object.(*mmeer.VMTx)
				if vt.Coinbase.IsEqual(block.Transactions()[0].Hash()) {
					result.Txs = append(result.Txs, vt)
					continue
				}
			}
			ctx, err := mmeer.NewVMTx(tx.Tx, block.Transactions()[0].Tx)
			if err != nil {
				return nil, err
			}
			tx.Object = ctx
			result.Txs = append(result.Txs, ctx)
		}
	}
	return result, nil
}
