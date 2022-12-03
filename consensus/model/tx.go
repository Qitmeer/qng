package model

import "github.com/Qitmeer/qng/core/types"

type Tx interface {
	GetTxType() types.TxType
	GetFrom() string
	GetTo() string
	GetValue() uint64
	GetData() []byte
}
