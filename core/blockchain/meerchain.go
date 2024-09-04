package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	mmeer "github.com/Qitmeer/qng/consensus/model/meer"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerdag"
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

func (b *BlockChain) MeerVerifyTx(tx model.Tx, utxoView *utxo.UtxoViewpoint) (int64, error) {
	if tx.GetTxType() == types.TxTypeCrossChainVM {
		vmtx := tx.(*mmeer.VMTx)
		fee, err := b.meerChain.VerifyTx(vmtx, utxoView)
		if err != nil {
			return 0, err
		}

		var op *types.TxOutPoint
		if vmtx.ExportData != nil {
			op, err = vmtx.ExportData.GetOutPoint()
			if err != nil {
				return 0, err
			}
		} else if vmtx.Export4337Data != nil {
			op, err = vmtx.Export4337Data.GetOutPoint()
			if err != nil {
				return 0, err
			}
		} else {
			return fee, err
		}

		utxoEntry := utxoView.LookupEntry(*op)
		if utxoEntry == nil || utxoEntry.IsSpent() {
			str := fmt.Sprintf("output %v referenced from "+
				"meer transaction %s either does not exist or "+
				"has already been spent", op, vmtx.ETx.Hash())
			return 0, ruleError(ErrMissingTxOut, str)
		}

		// Ensure the coinId is known
		err = types.CheckCoinID(utxoEntry.Amount().Id)
		if err != nil {
			return 0, err
		}
		if utxoEntry.Amount().Id != types.MEERA {
			return 0, fmt.Errorf("meer tx %s has illegal inputs %s", vmtx.ETx.Hash(), utxoEntry.Amount().Id.Name())
		}

		originTxAtom := utxoEntry.Amount()

		var ubhIB meerdag.IBlock
		if utxoEntry.IsCoinBase() {
			ubhIB = b.bd.GetBlock(utxoEntry.BlockHash())
			if ubhIB == nil {
				str := fmt.Sprintf("utxoEntry blockhash error:%s", utxoEntry.BlockHash())
				return 0, ruleError(ErrNoViewpoint, str)
			}

			if !utxoEntry.BlockHash().IsEqual(b.params.GenesisHash) {
				if originTxAtom.Id == types.MEERA {
					if op.OutIndex == CoinbaseOutput_subsidy {
						originTxAtom.Value += b.GetFeeByCoinID(utxoEntry.BlockHash(), originTxAtom.Id)
					}
				}
			}
		}
		if originTxAtom.Value < 0 {
			str := fmt.Sprintf("meer transaction output has negative "+
				"value of %v", originTxAtom)
			return 0, ruleError(ErrInvalidTxOutValue, str)
		}
		if originTxAtom.Value == 0 {
			str := fmt.Sprintf("meer transaction output is empty "+
				"value of %v", originTxAtom)
			return 0, ruleError(ErrInvalidTxOutValue, str)
		}
		if originTxAtom.Value > types.MaxAmount {
			str := fmt.Sprintf("meer transaction output value of %v is "+
				"higher than max allowed value of %v",
				originTxAtom, types.MaxAmount)
			return 0, ruleError(ErrInvalidTxOutValue, str)
		}

		if ubhIB != nil {
			targets := []uint{ubhIB.GetID()}
			viewpoints := []uint{}
			for _, blockHash := range utxoView.Viewpoints() {
				vIB := b.bd.GetBlock(blockHash)
				if vIB != nil {
					viewpoints = append(viewpoints, vIB.GetID())
				}
			}
			if len(viewpoints) == 0 {
				str := fmt.Sprintf("meer transaction %s has no viewpoints", vmtx.ETx.Hash())
				return 0, ruleError(ErrNoViewpoint, str)
			}
			err := b.bd.CheckBlueAndMatureMT(targets, viewpoints, uint(b.params.CoinbaseMaturity))
			if err != nil {
				return 0, ruleError(ErrImmatureSpend, err.Error())
			}
		}
		return fee, err
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

func (b *BlockChain) VerifyMeerTx(tx model.Tx) error {
	view := utxo.NewUtxoViewpoint()
	view.SetViewpoints(b.GetMiningTips(meerdag.MaxPriority))
	_, err := b.MeerVerifyTx(tx, view)
	return err
}

func (bc *BlockChain) connectVMTransaction(tx *types.Tx, vmtx *mmeer.VMTx, stxos *[]utxo.SpentTxOut, view *utxo.UtxoViewpoint) error {
	var err error
	var op *types.TxOutPoint
	if vmtx.ExportData != nil {
		op, err = vmtx.ExportData.GetOutPoint()
		if err != nil {
			return err
		}
	} else if vmtx.Export4337Data != nil {
		op, err = vmtx.Export4337Data.GetOutPoint()
		if err != nil {
			return err
		}
	} else if vmtx.ImportData != nil {
		view.AddTxOutForMeerChange(*vmtx.ImportData.OutPoint, vmtx.ImportData.Output)
	} else {
		return nil
	}

	entry := view.Entries()[*op]
	if entry == nil {
		return model.AssertError(fmt.Sprintf("view missing input %v", *op))
	}
	entry.Spend()

	// Don't create the stxo details if not requested.
	if stxos == nil {
		return nil
	}
	var stxo = utxo.SpentTxOut{
		Amount:     entry.Amount(),
		Fees:       types.Amount{Value: 0, Id: entry.Amount().Id},
		PkScript:   entry.PkScript(),
		BlockHash:  *entry.BlockHash(),
		IsCoinBase: entry.IsCoinBase(),
		TxIndex:    uint32(tx.Index()),
		TxInIndex:  uint32(0),
	}
	if stxo.IsCoinBase && !entry.BlockHash().IsEqual(bc.params.GenesisHash) {
		if op.OutIndex == CoinbaseOutput_subsidy ||
			entry.Amount().Id != types.MEERA {
			stxo.Fees.Value = bc.GetFeeByCoinID(&stxo.BlockHash, stxo.Fees.Id)
		}
	}
	// Append the entry to the provided spent txouts slice.
	*stxos = append(*stxos, stxo)
	log.Debug("meer tx spend utxo by crosschain contract", "txhash", tx.Hash().String(), "meertxhash", vmtx.ETx.Hash().String(), "utxoTxid", op.Hash.String(), "utxoIdx", op.OutIndex)
	return nil
}
