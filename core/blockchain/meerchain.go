package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	mmeer "github.com/Qitmeer/qng/consensus/model/meer"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerevm/meer"
)

func (b *BlockChain) MeerChain() model.MeerChain {
	return b.meerChain
}

func (b *BlockChain) calcMeerGenesis(txs []*types.Tx) *hash.Hash {
	has := false
	for idx, tx := range txs {
		if idx == 0 {
			continue
		}
		if tx.IsDuplicate {
			continue
		}
		if types.IsCrossChainExportTx(tx.Tx) {
			has = true
			break
		} else if types.IsCrossChainImportTx(tx.Tx) {
			has = true
			break
		} else if types.IsCrossChainVMTx(tx.Tx) {
			has = true
			break
		}
	}
	if !has {
		return nil
	}
	return b.meerChain.Genesis()
}

func (b *BlockChain) meerCheckConnectBlock(block *BlockNode) error {
	eb, err := meer.BuildEVMBlock(block.GetBody())
	if err != nil {
		return err
	}
	block.SetMeerBlock(nil)
	if len(eb.Transactions()) <= 0 {
		return nil
	}
	block.SetMeerBlock(eb)
	return b.meerChain.CheckConnectBlock(eb)
}

func (b *BlockChain) meerConnectBlock(block *BlockNode) (uint64, error) {
	eb := block.GetMeerBlock()
	if eb == nil || len(eb.Transactions()) <= 0 {
		return 0, nil
	}
	return b.meerChain.ConnectBlock(eb)
}

func (b *BlockChain) MeerVerifyTx(tx model.Tx) (int64, error) {
	if tx.GetTxType() == types.TxTypeCrossChainVM {
		return b.meerChain.VerifyTx(tx)
	}

	if tx.GetTxType() != types.TxTypeCrossChainImport {
		return 0, fmt.Errorf("Not support:%s\n", tx.GetTxType().String())
	}

	itx, ok := tx.(*mmeer.ImportTx)
	if !ok {
		return 0, fmt.Errorf("Not support tx:%s\n", tx.GetTxType().String())
	}

	pka, err := itx.GetPKAddress()
	if err != nil {
		return 0, err
	}
	ba, err := b.meerChain.GetBalance(pka.String())
	if err != nil {
		return 0, err
	}
	if ba <= 0 {
		return 0, fmt.Errorf("Balance (%s) is %d\n", pka.String(), ba)
	}
	if ba < itx.Transaction.TxOut[0].Amount.Value {
		return 0, fmt.Errorf("Balance (%s)  %d < output %d", pka.String(), ba, itx.Transaction.TxOut[0].Amount.Value)
	}
	return ba - itx.Transaction.TxOut[0].Amount.Value, nil
}
