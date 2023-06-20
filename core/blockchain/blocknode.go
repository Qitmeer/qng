// Copyright (c) 2017-2018 The qitmeer developers
package blockchain

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"time"
)

type BlockNode struct {
	block *types.SerializedBlock
}

// return the block node hash.
func (node *BlockNode) GetHash() *hash.Hash {
	return node.block.Hash()
}

// Include all parents for set
func (node *BlockNode) GetParents() []*hash.Hash {
	return node.block.Block().Parents
}

// return the timestamp of node
func (node *BlockNode) GetTimestamp() int64 {
	return node.block.Block().Header.Timestamp.Unix()
}

func (node *BlockNode) GetHeader() *types.BlockHeader {
	return &node.block.Block().Header
}

func (node *BlockNode) Difficulty() uint32 {
	return node.GetHeader().Difficulty
}

func (node *BlockNode) Pow() pow.IPow {
	return node.GetHeader().Pow
}

func (node *BlockNode) GetPowType() pow.PowType {
	return node.Pow().GetPowType()
}

func (node *BlockNode) Timestamp() time.Time {
	return node.GetHeader().Timestamp
}

func (node *BlockNode) GetPriority() int {
	return len(node.block.Block().Transactions)
}

func (node *BlockNode) GetMainParent() *hash.Hash {
	return node.block.Block().Parents[0]
}

func NewBlockNode(block *types.SerializedBlock) *BlockNode {
	bn := BlockNode{
		block: block,
	}
	return &bn
}
