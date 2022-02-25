package consensus

import (
	"github.com/Qitmeer/qng-core/common/hash"
	"github.com/Qitmeer/qng-core/consensus"
	"github.com/Qitmeer/qng-core/core/types"
)

type VMI interface {
	VerifyTx(tx consensus.Tx) (int64, error)
	GetVM(id string) (consensus.ChainVM, error)
	CheckConnectBlock(block *types.SerializedBlock) error
	ConnectBlock(block *types.SerializedBlock) error
	DisconnectBlock(block *types.SerializedBlock) error
	RemoveTxFromMempool(h *hash.Hash) error
}
