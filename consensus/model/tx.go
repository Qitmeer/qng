package model

import "github.com/Qitmeer/qng/v2/core/types"

type Tx interface {
	GetTxType() types.TxType
	GetFrom() []byte
	GetTo() []byte
	GetValue() uint64
	GetData() []byte
}
