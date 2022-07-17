package blockchain

import "github.com/Qitmeer/qng/core/types"

type ACCTI interface {
	Apply(add bool, op *types.TxOutPoint, entry *UtxoEntry) error
}