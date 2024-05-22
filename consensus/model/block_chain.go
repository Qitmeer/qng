package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
)

type BlockChain interface {
	GetMainOrder() uint
	FetchBlockByOrder(order uint64) (*types.SerializedBlock, Block, error)
	FetchSpendJournalPKS(targetBlock *types.SerializedBlock) ([][]byte, error)
	SetDAGDuplicateTxs(sblock *types.SerializedBlock, block Block)
	GetBlockHashByOrder(order uint) *hash.Hash
	BlockByOrder(blockOrder uint64) (*types.SerializedBlock, error)
	Rebuild() error
	GetMiningTips(expectPriority int) []*hash.Hash
	GetBlockState(order uint64) BlockState
	MeerChain() MeerChain
	Start() error
	Stop() error
	GetBlockByOrder(order uint64) Block
	GetBlockById(id uint) Block
	GetMainChainTip() Block
	FetchBlockByHash(hash *hash.Hash) (*types.SerializedBlock, error)
	GetBlockOrderByHash(hash *hash.Hash) (uint, error)
	GetBlockHeader(block Block) *types.BlockHeader
	ForeachBlueBlocks(start Block, depth uint, powType pow.PowType, fn func(block Block, header *types.BlockHeader) error) error
}
