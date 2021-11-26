package consensus

import "github.com/Qitmeer/qng-core/core/types"

type Tx interface {
	GetTxType() types.TxType
	GetFrom() string
	GetTo() string
	GetValue() uint64
	GetData() []byte
}
