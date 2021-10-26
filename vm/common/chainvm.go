package common

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

type ChainVM interface {
	VM

	GetBlock(*hash.Hash) (*types.Block, error)

	BuildBlock() (*types.Block, error)

	ParseBlock([]byte) (*types.Block, error)
}
