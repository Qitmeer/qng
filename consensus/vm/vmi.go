package vm

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/vm/consensus"
)

type VMI interface {
	VerifyTx(tx consensus.Tx) (int64, error)
	VerifyTxSanity(tx consensus.Tx) error
	GetVM(id string) (consensus.ChainVM, error)
	CheckConnectBlock(block *types.SerializedBlock) error
	ConnectBlock(block *types.SerializedBlock) (uint64,error)
	DisconnectBlock(block *types.SerializedBlock) (uint64,error)
	AddTxToMempool(tx *types.Transaction, local bool) (int64, error)
	RemoveTxFromMempool(tx *types.Transaction) error
	GetTxsFromMempool() ([]*types.Transaction,[]*hash.Hash, error)
	GetMempoolSize() int64
	ResetTemplate() error
	Genesis(txs []*types.Tx) *hash.Hash
	GetBlockID(bh *hash.Hash) uint64
	GetBlockIDByTxHash(txhash *hash.Hash) uint64
}
