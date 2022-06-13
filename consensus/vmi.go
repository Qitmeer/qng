package consensus

import (
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/vm/consensus"
)

type VMI interface {
	VerifyTx(tx consensus.Tx) (int64, error)
	VerifyTxSanity(tx consensus.Tx) error
	GetVM(id string) (consensus.ChainVM, error)
	CheckConnectBlock(block *types.SerializedBlock) error
	ConnectBlock(block *types.SerializedBlock) error
	DisconnectBlock(block *types.SerializedBlock) error
	AddTxToMempool(tx *types.Transaction, local bool) (int64, error)
	RemoveTxFromMempool(tx *types.Transaction) error
	GetTxsFromMempool() ([]*types.Transaction, error)
	GetMempoolSize() int64
}
