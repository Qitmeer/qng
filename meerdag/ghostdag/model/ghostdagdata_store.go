package model

import "github.com/Qitmeer/qng/common/hash"

// GHOSTDAGDataStore represents a store of BlockGHOSTDAGData
type GHOSTDAGDataStore interface {
	Store
	Stage(stagingArea *StagingArea, blockHash *hash.Hash, blockGHOSTDAGData *BlockGHOSTDAGData, isTrustedData bool)
	IsStaged(stagingArea *StagingArea) bool
	Get(dbContext DBReader, stagingArea *StagingArea, blockHash *hash.Hash, isTrustedData bool) (*BlockGHOSTDAGData, error)
	UnstageAll(stagingArea *StagingArea)
}
