/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package model

import (
	"github.com/Qitmeer/qng/core/types"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Notify interface manage message announce & relay & notification between mempool, websocket, gbt long pull
// and rpc server.
type Notify interface {
	AnnounceNewTransactions(newTxs []*types.TxDesc, meerTxs []*etypes.Transaction, filters []peer.ID)
	RelayInventory(block *types.SerializedBlock, flags uint32, source *peer.ID)
	BroadcastMessage(data interface{})
	TransactionConfirmed(tx *types.Tx)
	TransactionsConfirmed(txs []*types.Tx)
	AddRebroadcastInventory(newTxs []*types.TxDesc)
}
