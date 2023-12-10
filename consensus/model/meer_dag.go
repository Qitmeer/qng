package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/rpc/api"
)

type MeerDag interface {
	RegisterAPIs(apis []api.API)
	GetBlockIDByTxHash(txhash *hash.Hash) uint64
}
