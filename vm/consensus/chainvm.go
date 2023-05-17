/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

type ChainVM interface {
	VM

	GetBlock(*hash.Hash) (Block, error)
	GetBlockByNumber(num uint64) (Block, error)

	BuildBlock([]model.Tx) (Block, error)

	ParseBlock([]byte) (Block, error)

	LastAccepted() (*hash.Hash, error)

	GetBalance(string) (int64, error)

	VerifyTx(tx model.Tx) (int64, error)
	VerifyTxSanity(tx model.Tx) error

	AddTxToMempool(tx *types.Transaction, local bool) (int64, error)

	GetTxsFromMempool() ([]*types.Transaction, []*hash.Hash, error)

	GetMempoolSize() int64

	RemoveTxFromMempool(tx *types.Transaction) error

	CheckConnectBlock(block Block) error

	ConnectBlock(block Block) (uint64, error)

	DisconnectBlock(block Block) (uint64, error)
	RewindTo(state model.BlockState) error

	ResetTemplate() error

	Genesis() *hash.Hash

	GetBlockIDByTxHash(txhash *hash.Hash) uint64

	GetCurStateRoot() common.Hash
	GetCurHeader() *etypes.Header
	BlockChain() *core.BlockChain
	ChainDatabase() ethdb.Database
}
