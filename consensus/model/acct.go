package model

import (
	"github.com/Qitmeer/qng/core/types"
)

type Acct interface {
	Apply(add bool, op *types.TxOutPoint, entry interface{}) error
	Commit(point Block) error
}
