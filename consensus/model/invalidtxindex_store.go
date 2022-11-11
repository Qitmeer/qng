package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

type InvalidTxIndexStore interface {
	Store
	Stage(stagingArea *StagingArea, bid uint64, block *types.SerializedBlock)
	StageTip(stagingArea *StagingArea, bhash *hash.Hash, order uint64)
	IsStaged(stagingArea *StagingArea) bool
	Get(stagingArea *StagingArea, txid *hash.Hash) (*types.Transaction, error)
	GetIdByHash(stagingArea *StagingArea,h *hash.Hash) (*hash.Hash, error)
	Delete(stagingArea *StagingArea, bid uint64,block *types.SerializedBlock)
	Tip(stagingArea *StagingArea) (uint64, *hash.Hash, error)
	IsEmpty() bool
	Clean() error
}
