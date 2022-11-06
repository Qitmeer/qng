package model

import (
	"github.com/Qitmeer/qng/common/hash"
)

type VMBlockIndexStore interface {
	Store
	Stage(stagingArea *StagingArea, bid uint64, bhash *hash.Hash)
	StageTip(stagingArea *StagingArea, bhash *hash.Hash, order uint64)
	IsStaged(stagingArea *StagingArea) bool
	Get(stagingArea *StagingArea, bid uint64) (*hash.Hash, error)
	Has(stagingArea *StagingArea, bid uint64) (bool, error)
	Delete(stagingArea *StagingArea, bid uint64)
	Tip(stagingArea *StagingArea) (uint64, *hash.Hash, error)
	IsEmpty() bool
}
