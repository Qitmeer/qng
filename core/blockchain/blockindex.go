// Copyright (c) 2017-2018 The qitmeer developers
package blockchain

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerdag"
)

// LookupNode returns the block node identified by the provided hash.  It will
// return nil if there is no entry for the hash.
func (b *BlockChain) LookupNode(hash *hash.Hash) *BlockNode {
	ib := b.GetBlock(hash)
	if ib == nil {
		return nil
	}
	return b.bd.GetBlockData(ib).(*BlockNode)
}

func (b *BlockChain) LookupNodeById(id uint) *BlockNode {
	ib := b.bd.GetBlockById(id)
	if ib == nil {
		return nil
	}
	return b.bd.GetBlockData(ib).(*BlockNode)
}

func (b *BlockChain) GetBlockNode(ib meerdag.IBlock) *BlockNode {
	if ib == nil {
		return nil
	}
	return b.bd.GetBlockData(ib).(*BlockNode)
}

func (b *BlockChain) GetBlock(h *hash.Hash) meerdag.IBlock {
	return b.bd.GetBlock(h)
}
