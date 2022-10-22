package model

import "github.com/Qitmeer/qng/common/hash"

// BlockHeaderStore represents a store of block headers
type BlockHeaderStore interface {
	Store
	Stage(stagingArea *StagingArea, blockHash *hash.Hash, blockHeader BlockHeader)
	IsStaged(stagingArea *StagingArea) bool
	BlockHeader(dbContext DBReader, stagingArea *StagingArea, blockHash *hash.Hash) (BlockHeader, error)
	HasBlockHeader(dbContext DBReader, stagingArea *StagingArea, blockHash *hash.Hash) (bool, error)
	BlockHeaders(dbContext DBReader, stagingArea *StagingArea, blockHashes []*hash.Hash) ([]BlockHeader, error)
	Delete(stagingArea *StagingArea, blockHash *hash.Hash)
	Count(stagingArea *StagingArea) uint64
}
