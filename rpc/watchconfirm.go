package rpc

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/Qitmeer/qng/services/index"
)

type TxConfirm struct {
	Confirms  uint64
	TxHash    string
	EndHeight uint64 // timeout
}

func (w *WatchTxConfirmServer) AddTxConfirms(confirm TxConfirm) {
	if w == nil {
		w = &WatchTxConfirmServer{}
	}
	if _, ok := (*w)[confirm.TxHash]; !ok {
		(*w)[confirm.TxHash] = TxConfirm{}
	}
	(*w)[confirm.TxHash] = confirm
}

func (w *WatchTxConfirmServer) RemoveTxConfirms(confirm TxConfirm) {
	if w == nil {
		w = &WatchTxConfirmServer{}
	}
	if _, ok := (*w)[confirm.TxHash]; !ok {
		return
	}
	delete((*w), confirm.TxHash)
}

type WatchTxConfirmServer map[string]TxConfirm

func (w *WatchTxConfirmServer) Handle(wsc *wsClient, currentHeight uint64) {
	if w == nil || len(*w) <= 0 {
		return
	}
	if wsc.server.consensus == nil {
		return
	}
	bc := wsc.server.BC
	indexMgr := wsc.server.consensus.IndexManager().(*index.Manager)

	for tx, txconf := range *w {
		txHash := hash.MustHexToDecodedHash(tx)
		mtx, blockHash, err := wsc.server.consensus.DatabaseContext().GetTxIdxEntry(&txHash, true)
		if err != nil {
			log.Error(err.Error(), "txhash", txHash)
			continue
		}
		// Deserialize the transaction
		var msgTx *types.Transaction

		if mtx == nil {
			if indexMgr.InvalidTxIndex() != nil {
				msgTx, err = indexMgr.InvalidTxIndex().Get(&txHash)
				if err != nil {
					log.Error(err.Error(), "txhash", txHash)
					continue
				}
			} else {
				// timeout
				if txconf.EndHeight > 0 && currentHeight >= txconf.EndHeight {
					log.Debug("coinbase tx long time not confirm ,will remove for watch", "txhash", txHash)
					delete(*w, tx)
				}
				continue
			}
			if msgTx == nil {
				log.Error("tx not found")
				continue
			}
			mtx = types.NewTx(msgTx)
		}

		ib := bc.BlockDAG().GetBlock(blockHash)
		if ib == nil {
			log.Error("block hash not exist", "hash", blockHash)
			w.SendTxNotification(tx, 0, wsc, false, false)
			continue
		}
		if mtx.IsDuplicate {
			w.SendTxNotification(tx, 0, wsc, false, false)
			continue
		}
		isBlue := true
		if mtx.Tx.IsCoinBase() {
			isBlue = bc.BlockDAG().IsBlue(ib.GetID())
		}
		if !isBlue {
			w.SendTxNotification(tx, 0, wsc, false, false)
			continue
		}
		InValid := ib.GetState().GetStatus().KnownInvalid()
		if InValid {
			w.SendTxNotification(tx, 0, wsc, false, false)
			continue
		}
		confirmations := bc.BlockDAG().GetConfirmations(ib.GetID())
		if uint64(confirmations) >= txconf.Confirms {
			w.SendTxNotification(tx, uint64(confirmations), wsc, isBlue, !InValid)
		}
	}
}

func (w *WatchTxConfirmServer) SendTxNotification(tx string, confirms uint64, wsc *wsClient, isBlue, isValid bool) {
	ntfn := &cmds.NotificationTxConfirmNtfn{
		ConfirmResult: cmds.TxConfirmResult{
			Tx:       tx,
			Confirms: confirms,
			IsBlue:   isBlue,
			IsValid:  isValid,
		},
	}
	marshalledJSON, err := cmds.MarshalCmd(nil, ntfn)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to marshal tx confirm notification: "+
			"%v", err))
		return
	}
	err = wsc.QueueNotification(marshalledJSON)
	if err != nil {
		log.Error("notify failed", "err", err)
		return
	}
	delete(*w, tx)
}
