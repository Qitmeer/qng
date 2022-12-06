// Copyright (c) 2017-2018 The qitmeer developers
package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
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

func (b *BlockChain) getBlockData(hash *hash.Hash) meerdag.IBlockData {
	if hash.String() == forks.BadBlockHashHex {
		panic(fmt.Sprintf("The dag data was damaged (Has bad block %s). you can cleanup your block data base by '--cleanup'.", hash.String()))
	}
	block, err := b.fetchBlockByHash(hash)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return NewBlockNode(block, block.Block().Parents)
}

func (b *BlockChain) GetBlock(h *hash.Hash) meerdag.IBlock {
	return b.bd.GetBlock(h)
}

// BlockOrderByHash returns the order of the block with the given hash in the
// chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) BlockOrderByHash(hash *hash.Hash) (uint64, error) {
	ib := b.bd.GetBlock(hash)
	if ib == nil {
		return uint64(meerdag.MaxBlockOrder), fmt.Errorf("No block\n")
	}
	return uint64(ib.GetOrder()), nil
}

// OrderRange returns a range of block hashes for the given start and end
// orders.  It is inclusive of the start order and exclusive of the end
// order.  The end order will be limited to the current main chain order.
//
// This function is safe for concurrent access.
func (b *BlockChain) OrderRange(startOrder, endOrder uint64) ([]hash.Hash, error) {
	// Ensure requested orders are sane.
	if startOrder < 0 {
		return nil, fmt.Errorf("start order of fetch range must not "+
			"be less than zero - got %d", startOrder)
	}
	if endOrder < startOrder {
		return nil, fmt.Errorf("end order of fetch range must not "+
			"be less than the start order - got start %d, end %d",
			startOrder, endOrder)
	}

	// There is nothing to do when the start and end orders are the same,
	// so return now to avoid the chain view lock.
	if startOrder == endOrder {
		return nil, nil
	}

	// Grab a lock on the chain view to prevent it from changing due to a
	// reorg while building the hashes.
	b.ChainLock()
	defer b.ChainUnlock()

	// When the requested start order is after the most recent best chain
	// order, there is nothing to do.
	latestOrder := b.BestSnapshot().GraphState.GetMainOrder()
	if startOrder > uint64(latestOrder) {
		return nil, nil
	}

	// Limit the ending order to the latest order of the chain.
	if endOrder > uint64(latestOrder+1) {
		endOrder = uint64(latestOrder + 1)
	}

	// Fetch as many as are available within the specified range.
	hashes := make([]hash.Hash, 0, endOrder-startOrder)
	for i := startOrder; i < endOrder; i++ {
		h, err := b.BlockHashByOrder(i)
		if err != nil {
			log.Error("order not exist", "order", i)
			return nil, err
		}
		hashes = append(hashes, *h)
	}
	return hashes, nil
}

func (b *BlockChain) GetBlockHashByOrder(order uint) *hash.Hash {
	return b.bd.GetBlockHashByOrder(order)
}

// dbFetchBlockByOrder uses an existing database transaction to retrieve the
// raw block for the provided order, deserialize it, and return a Block
// with the height set.
func (b *BlockChain) FetchBlockByOrder(order uint64) (*types.SerializedBlock, model.Block, error) {
	// First find the hash associated with the provided order in the index.
	ib := b.bd.GetBlockByOrder(uint(order))
	if ib == nil {
		return nil, nil, fmt.Errorf("No block\n")
	}

	var blockBytes []byte
	var err error
	err = b.db.View(func(dbTx database.Tx) error {
		// Load the raw block bytes from the database.
		blockBytes, err = dbTx.FetchBlock(ib.GetHash())
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	// Create the encapsulated block and set the order appropriately.
	block, err := types.NewBlockFromBytes(blockBytes)
	if err != nil {
		return nil, nil, err
	}
	block.SetOrder(order)
	block.SetHeight(ib.GetHeight())
	return block, ib, nil
}

// BlockByHeight returns the block at the given height in the main chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) BlockByOrder(blockOrder uint64) (*types.SerializedBlock, error) {
	block, _, err := b.FetchBlockByOrder(blockOrder)
	return block, err
}

// BlockHashByOrder returns the hash of the block at the given order in the
// main chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) BlockHashByOrder(blockOrder uint64) (*hash.Hash, error) {
	hash := b.bd.GetBlockHashByOrder(uint(blockOrder))
	if hash == nil {
		return nil, fmt.Errorf("Can't find block")
	}
	return hash, nil
}

// MainChainHasBlock returns whether or not the block with the given hash is in
// the main chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) MainChainHasBlock(hash *hash.Hash) bool {
	return b.bd.IsOnMainChain(b.bd.GetBlockId(hash))
}

func (b *BlockChain) GetMainOrder() uint {
	return b.BestSnapshot().GraphState.GetMainOrder()
}

// expect priority
func (b *BlockChain) GetMiningTips(expectPriority int) []*hash.Hash {
	return b.BlockDAG().GetValidTips(expectPriority)
}

func (b *BlockChain) HasTx(txid *hash.Hash) bool {
	return b.indexManager.HasTx(txid)
}
