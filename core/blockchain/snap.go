package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/v2/common/hash"
	mmeer "github.com/Qitmeer/qng/v2/consensus/model/meer"
	"github.com/Qitmeer/qng/v2/core/blockchain/token"
	"github.com/Qitmeer/qng/v2/core/blockchain/utxo"
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/meerdag"
	"time"
)

type SnapData struct {
	block      *types.SerializedBlock
	stxos      []utxo.SpentTxOut
	dagBlock   meerdag.IBlock
	dagBIDs    map[uint]*hash.Hash
	main       bool
	tokenState *token.TokenState
	prevTSHash *hash.Hash
}

func (s *SnapData) SetBlock(block *types.SerializedBlock) {
	s.block = block
}

func (s *SnapData) SetStxos(stxos []utxo.SpentTxOut) {
	s.stxos = stxos
}

func (s *SnapData) SetDAGBlock(block meerdag.IBlock, dagBIDs map[uint]*hash.Hash) {
	s.dagBlock = block
	s.dagBIDs = dagBIDs
}

func (s *SnapData) SetMain(main bool) {
	s.main = main
}

func (s *SnapData) SetTokenState(tokenState *token.TokenState) {
	s.tokenState = tokenState
}

func (s *SnapData) SetPrevTSHash(prevTSHash *hash.Hash) {
	s.prevTSHash = prevTSHash
}

func (s *SnapData) GetStxo(idx int, inIdx int) *utxo.SpentTxOut {
	if len(s.stxos) <= 0 {
		return nil
	}
	for _, stxo := range s.stxos {
		if stxo.TxIndex == uint32(idx) && stxo.TxInIndex == uint32(inIdx) {
			return &stxo
		}
	}
	return nil
}

func (s *SnapData) HasStxo(idx int, inIdx int) bool {
	return s.GetStxo(idx, inIdx) != nil
}

func NewSnapData() *SnapData {
	return &SnapData{}
}

func (b *BlockChain) IsSnapSyncing() bool {
	return b.snapSyncing.Load()
}

func (b *BlockChain) SetSnapSyncing(val bool) {
	b.snapSyncing.Store(val)
}

func (b *BlockChain) BeginSnapSyncing() error {
	if b.IsSnapSyncing() {
		return nil
	}
	b.SetSnapSyncing(true)
	for {
		size := b.ProcessQueueSize()
		if size <= 0 {
			break
		}
		select {
		case <-time.After(time.Second):
			log.Trace("Try to snap syncing", "size", size)
		case <-b.quit:
			b.SetSnapSyncing(false)
			return fmt.Errorf("blockchain shutdown")
		}
	}
	return nil
}

func (b *BlockChain) EndSnapSyncing() {
	b.bd.RecalDiffAnticone()
	b.SetSnapSyncing(false)
}

func (b *BlockChain) ProcessBlockBySnap(sds []*SnapData) (meerdag.IBlock, error) {
	if len(sds) <= 0 {
		return nil, nil
	}
	var latest meerdag.IBlock

	b.ChainLock()
	returnFun := func(err error) (meerdag.IBlock, error) {
		b.ChainUnlock()
		return latest, err
	}

	spendEntry := func(tx *types.Tx, view *utxo.UtxoViewpoint, sd *SnapData, node *BlockNode) {
		if !tx.Transaction().IsCoinBase() {
			for txInIndex, txIn := range tx.Transaction().TxIn {
				entry := view.Entries()[txIn.PreviousOut]
				if entry != nil {
					if sd.HasStxo(tx.Index(), txInIndex) {
						entry.Spend()
					}
				}
			}
		}
		view.AddTxOuts(tx, node.GetHash())
	}

	txNum := int64(0)
	for _, sd := range sds {
		if !b.DB().HasBlock(sd.block.Hash()) {
			err := dbMaybeStoreBlock(b.DB(), sd.block)
			if err != nil {
				return returnFun(err)
			}
		}
		node := NewBlockNode(sd.block)
		dblock, err := b.bd.AddDirectBlock(node, sd.dagBlock, sd.main, sd.dagBIDs)
		if err != nil {
			b.ChainUnlock()
			return latest, err
		}
		b.SetDAGDuplicateTxs(sd.block, dblock)
		//

		if !dblock.GetState().GetStatus().KnownInvalid() {

			view := utxo.NewUtxoViewpoint()
			view.SetViewpoints([]*hash.Hash{node.GetHash()})

			err = b.fetchInputUtxos(sd.block, view)
			if err != nil {
				return returnFun(err)
			}

			for _, tx := range sd.block.Transactions() {
				if tx.IsDuplicate {
					if tx.Tx.IsCoinBase() {
						return returnFun(ruleError(ErrDuplicateTx, fmt.Sprintf("Coinbase Tx(%s) is duplicate in block(%s)", tx.Hash().String(), node.GetHash().String())))
					}
					continue
				}

				if types.IsTokenTx(tx.Tx) {
					if types.IsTokenMintTx(tx.Tx) {
						spendEntry(tx, view, sd, node)
					}
				} else if types.IsCrossChainImportTx(tx.Tx) {
					view.AddTxOuts(tx, node.GetHash())
				} else if types.IsCrossChainVMTx(tx.Tx) {
					var vtx *mmeer.VMTx
					var err error
					if tx.Object != nil {
						vtx = tx.Object.(*mmeer.VMTx)
					} else {
						vtx, err = mmeer.NewVMTx(tx.Tx, sd.block.Transactions()[0].Tx)
						if err != nil {
							return returnFun(err)
						}
						tx.Object = vtx
					}
					if vtx.ExportData != nil {
						err = b.meerChain.TxPool().CheckMeerChangeExportTx(vtx.ETx, vtx.ExportData, view)
						if err != nil {
							return returnFun(err)
						}
					}

					err = b.connectVMTransaction(tx, vtx, nil, view)
					if err != nil {
						return returnFun(err)
					}
				} else {
					spendEntry(tx, view, sd, node)
				}
			}

			err = b.dbUpdateUtxoView(view)
			if err != nil {
				return returnFun(err)
			}
			view.Commit()

			err = utxo.DBPutSpendJournalEntry(b.DB(), sd.block.Hash(), sd.stxos)
			if err != nil {
				return returnFun(err)
			}
			pkss := [][]byte{}
			for _, stxo := range sd.stxos {
				pkss = append(pkss, stxo.PkScript)
			}
			err = b.indexManager.ConnectBlock(sd.block, dblock, pkss)
			if err != nil {
				return returnFun(err)
			}
			if sd.prevTSHash != nil {
				err = b.addTokenState(dblock.GetID(), sd.tokenState, sd.prevTSHash)
				if err != nil {
					return returnFun(err)
				}
			}
		} else {
			err := b.indexManager.ConnectBlock(sd.block, dblock, nil)
			if err != nil {
				return returnFun(err)
			}
		}

		//
		latest = dblock

		txNum += int64(len(sd.block.Transactions()))
	}
	err := b.updateBestState(sds[len(sds)-1].block, false)
	if err != nil {
		return returnFun(err)
	}
	b.ChainUnlock()

	if b.Acct != nil {
		err := b.Acct.Commit(latest)
		if err != nil {
			log.Error(err.Error())
		}
	}
	b.progressLogger.LogBlockOrders(latest, int64(len(sds)), txNum)
	return latest, nil
}
