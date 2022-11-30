package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
)

// BlockHeaderStore represents a store of block headers
type BlockHeaderStore interface {
	model.Store
	BlockHeader(dbContext DBReader, stagingArea *model.StagingArea, blockHash *hash.Hash) (BlockHeader, error)
	HasBlockHeader(dbContext DBReader, stagingArea *model.StagingArea, blockHash *hash.Hash) (bool, error)
	BlockHeaders(dbContext DBReader, stagingArea *model.StagingArea, blockHashes []*hash.Hash) ([]BlockHeader, error)
	Delete(stagingArea *model.StagingArea, blockHash *hash.Hash)
	Count(stagingArea *model.StagingArea) uint64
}
