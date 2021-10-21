package vm

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

type ChainVM interface {
	Version() (string, error)

	GetBlock(*hash.Hash) (*types.Block, error)

	BuildBlock() (*types.Block, error)

	ParseBlock([]byte) (*types.Block, error)

	Shutdown() error
}
