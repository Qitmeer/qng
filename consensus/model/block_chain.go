package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

type BlockChain interface {
	GetMainOrder() uint
	DBFetchBlockByOrder(order uint64) (*types.SerializedBlock, Block, error)
	FetchSpendJournalPKS(targetBlock *types.SerializedBlock) ([][]byte, error)
	CalculateDAGDuplicateTxs(block *types.SerializedBlock)
	GetBlockHashByOrder(order uint) *hash.Hash
	BlockByOrder(blockOrder uint64) (*types.SerializedBlock, error)
}
