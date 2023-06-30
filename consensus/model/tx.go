package model

import "github.com/Qitmeer/qng/core/types"

type Tx interface {
	GetTxType() types.TxType
	GetFrom() []byte
	GetTo() []byte
	GetValue() uint64
	GetData() []byte
}
