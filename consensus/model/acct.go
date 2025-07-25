package model

import (
	"github.com/Qitmeer/qng/v2/core/types"
)

type Acct interface {
	Apply(add bool, op *types.TxOutPoint, entry interface{}) error
	Commit(point Block) error
}
