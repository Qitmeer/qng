package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/database"
)

type VMBlockIndexStore interface {
	Store
	Stage(stagingArea *StagingArea, bid uint64, bhash *hash.Hash)
	StageTip(stagingArea *StagingArea, bhash *hash.Hash, order uint64)
	IsStaged(stagingArea *StagingArea) bool
	Get(dbContext database.DB, stagingArea *StagingArea, bid uint64) (*hash.Hash, error)
	Has(dbContext database.DB, stagingArea *StagingArea, bid uint64) (bool, error)
	Delete(stagingArea *StagingArea, bid uint64)
}
