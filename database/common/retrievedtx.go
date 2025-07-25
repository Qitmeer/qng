package common

import (
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/core/types"
)

type RetrievedTx struct {
	Bytes   []byte
	BlkHash *hash.Hash // Only set when transaction is in a block.
	Tx      *types.Tx
}
