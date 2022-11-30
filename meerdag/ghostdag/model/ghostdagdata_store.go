package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
)

// GHOSTDAGDataStore represents a store of BlockGHOSTDAGData
type GHOSTDAGDataStore interface {
	model.Store
	Stage(stagingArea *model.StagingArea, blockHash *hash.Hash, blockGHOSTDAGData *BlockGHOSTDAGData, isTrustedData bool)
	IsStaged(stagingArea *model.StagingArea) bool
	Get(dbContext DBReader, stagingArea *model.StagingArea, blockHash *hash.Hash, isTrustedData bool) (*BlockGHOSTDAGData, error)
	UnstageAll(stagingArea *model.StagingArea)
}
