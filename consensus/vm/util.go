package vm

import (
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/vm/consensus"
)

func BuildEVMBlock(block *types.SerializedBlock, prevState model.BlockState) (consensus.Block, error) {
	result := &Block{Id: block.Hash(), Txs: []model.Tx{}, Time: block.Block().Header.Timestamp, ParentBlockState: prevState}

	for idx, tx := range block.Transactions() {
		if idx == 0 {
			continue
		}
		if tx.IsDuplicate {
			continue
		}

		if types.IsCrossChainExportTx(tx.Tx) {
			ctx, err := NewExportTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		} else if types.IsCrossChainImportTx(tx.Tx) {
			ctx, err := NewImportTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			err = ctx.SetCoinbaseTx(block.Transactions()[0].Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		} else if types.IsCrossChainVMTx(tx.Tx) {
			ctx, err := NewVMTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			err = ctx.SetCoinbaseTx(block.Transactions()[0].Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		}
	}
	return result, nil
}
