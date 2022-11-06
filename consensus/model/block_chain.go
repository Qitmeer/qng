package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
)

type BlockChain interface {
	GetMainOrder() uint
	DBFetchBlockByOrder(dbTx database.Tx, order uint64) (*types.SerializedBlock, Block, error)
	FetchSpendJournalPKS(targetBlock *types.SerializedBlock) ([][]byte, error)
	CalculateDAGDuplicateTxs(block *types.SerializedBlock)
	IsCacheInvalidTx() bool
	GetBlockHashByOrder(order uint) *hash.Hash
}
