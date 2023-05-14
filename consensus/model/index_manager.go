package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
)

// IndexManager provides a generic interface that the is called when blocks are
// connected and disconnected to and from the tip of the main chain for the
// purpose of supporting optional indexes.
type IndexManager interface {
	// Init is invoked during chain initialize in order to allow the index
	// manager to initialize itself and any indexes it is managing.  The
	// channel parameter specifies a channel the caller can close to signal
	// that the process should be interrupted.  It can be nil if that
	// behavior is not desired.
	Init() error

	// ConnectBlock is invoked when a new block has been connected to the
	// main chain.
	ConnectBlock(block *types.SerializedBlock, stxos [][]byte, blk Block) error

	// DisconnectBlock is invoked when a block has been disconnected from
	// the main chain.
	DisconnectBlock(block *types.SerializedBlock, stxos [][]byte, blk Block) error

	UpdateMainTip(bh *hash.Hash, order uint64) error

	// IsDuplicateTx
	IsDuplicateTx(tx database.Tx, txid *hash.Hash, blockHash *hash.Hash) bool

	HasTx(txid *hash.Hash) bool

	Drop() error
}
