package common

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

type RetrievedTx struct {
	Bytes   []byte
	BlkHash *hash.Hash // Only set when transaction is in a block.
	Tx      *types.Tx
}
