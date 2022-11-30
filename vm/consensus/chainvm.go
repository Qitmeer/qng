/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

type ChainVM interface {
	VM

	GetBlock(*hash.Hash) (Block, error)

	BuildBlock([]Tx) (Block, error)

	ParseBlock([]byte) (Block, error)

	LastAccepted() (*hash.Hash, error)

	GetBalance(string) (int64, error)

	VerifyTx(tx Tx) (int64, error)
	VerifyTxSanity(tx Tx) error

	AddTxToMempool(tx *types.Transaction, local bool) (int64, error)

	GetTxsFromMempool() ([]*types.Transaction,[]*hash.Hash, error)

	GetMempoolSize() int64

	RemoveTxFromMempool(tx *types.Transaction) error

	CheckConnectBlock(block Block) error

	ConnectBlock(block Block) (uint64,error)

	DisconnectBlock(block Block) (uint64,error)

	ResetTemplate() error

	Genesis() *hash.Hash

	GetBlockID(bh *hash.Hash) uint64
	GetBlockIDByTxHash(txhash *hash.Hash) uint64
}
