// Copyright (c) 2017-2018 The qitmeer developers
package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
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
	return NewBlockNode(block)
}

func (b *BlockChain) GetBlock(h *hash.Hash) meerdag.IBlock {
	return b.bd.GetBlock(h)
}

func (b *BlockChain) GetBlockByOrder(order uint64) model.Block {
	return b.bd.GetBlockByOrder(uint(order))
}

func (b *BlockChain) GetBlockById(id uint) model.Block {
	return b.bd.GetBlockById(id)
}

// BlockOrderByHash returns the order of the block with the given hash in the
// chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) GetBlockOrderByHash(hash *hash.Hash) (uint, error) {
	ib := b.bd.GetBlock(hash)
	if ib == nil {
		return meerdag.MaxBlockOrder, fmt.Errorf("No block order:%s", hash.String())
	}
	return ib.GetOrder(), nil
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
		return nil, nil, fmt.Errorf("No block:order=%d\n", order)
	}
	bn := b.GetBlockNode(ib)
	if bn == nil {
		return nil, nil, fmt.Errorf("No block node:hash=%s\n", ib.GetHash().String())
	}
	return bn.GetBody(), ib, nil
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

// Get Meer DAG block through the EVM block number
// TODO: Will continue to optimize in the future due to insufficient performance
func (b *BlockChain) GetBlockByNumber(number uint64) meerdag.IBlock {
	if number == 0 {
		return b.BlockDAG().GetBlockByOrder(0)
	}
	b.ChainRLock()
	defer b.ChainRUnlock()

	mainTip := b.BlockDAG().GetMainChainTip()
	var section meerdag.IBlock
	if number > uint64(mainTip.GetOrder()) ||
		number > mainTip.GetState().GetEVMNumber() {
		return nil
	} else if number == uint64(mainTip.GetOrder()) ||
		number == mainTip.GetState().GetEVMNumber() {
		section = mainTip
	} else {
		start := number
		end := uint64(mainTip.GetOrder())
		mid := uint64(0)
		for start <= end {
			mid = start + (end-start)/2
			cur := b.BlockDAG().GetBlockByOrder(uint(mid))
			if cur == nil {
				return nil
			}
			if cur.GetState().GetEVMNumber() == number {
				section = cur
				break
			} else if cur.GetState().GetEVMNumber() > number {
				end = mid - 1
			} else {
				start = mid + 1
			}
		}
	}
	if section == nil {
		return nil
	}
	if section.GetOrder() == 0 {
		return section
	}
	for i := section.GetOrder() - 1; i >= 0; i-- {
		prev := b.BlockDAG().GetBlockByOrder(i)
		if prev == nil {
			return nil
		}
		if prev.GetState().GetEVMNumber()+1 == number {
			return section
		}
		section = prev
	}
	return nil
}

// BlockByHash returns the block from the main chain with the given hash.
//
// This function is safe for concurrent access.
func (b *BlockChain) BlockByHash(hash *hash.Hash) (*types.SerializedBlock, error) {
	b.ChainRLock()
	defer b.ChainRUnlock()

	return b.fetchMainChainBlockByHash(hash)
}

// fetchMainChainBlockByHash returns the block from the main chain with the
// given hash.  It first attempts to use cache and then falls back to loading it
// from the database.
//
// An error is returned if the block is either not found or not in the main
// chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) fetchMainChainBlockByHash(hash *hash.Hash) (*types.SerializedBlock, error) {
	if !b.MainChainHasBlock(hash) {
		return nil, fmt.Errorf("No block in main chain")
	}
	block, err := b.fetchBlockByHash(hash)
	return block, err
}

// HeaderByHash returns the block header identified by the given hash or an
// error if it doesn't exist.  Note that this will return headers from both the
// main chain and any side chains.
//
// This function is safe for concurrent access.
func (b *BlockChain) HeaderByHash(hash *hash.Hash) (types.BlockHeader, error) {
	block, err := b.fetchBlockByHash(hash)
	if err != nil || block == nil {
		return types.BlockHeader{}, fmt.Errorf("block %s is not known", hash)
	}

	return block.Block().Header, nil
}

// FetchBlockByHash searches the internal chain block stores and the database
// in an attempt to find the requested block.
//
// This function differs from BlockByHash in that this one also returns blocks
// that are not part of the main chain (if they are known).
//
// This function is safe for concurrent access.
func (b *BlockChain) FetchBlockByHash(hash *hash.Hash) (*types.SerializedBlock, error) {
	return b.fetchBlockByHash(hash)
}

func (b *BlockChain) FetchBlockBytesByHash(hash *hash.Hash) ([]byte, error) {
	return b.fetchBlockBytesByHash(hash)
}

// fetchBlockByHash returns the block with the given hash from all known sources
// such as the internal caches and the database.
//
// This function is safe for concurrent access.
func (b *BlockChain) fetchBlockByHash(hash *hash.Hash) (*types.SerializedBlock, error) {
	// Check orphan cache.
	block := b.GetOrphan(hash)
	if block != nil {
		return block, nil
	}

	// Load the block from the database.
	block, err := dbFetchBlockByHash(b.consensus.DatabaseContext(), hash)
	if err == nil && block != nil {
		return block, nil
	}
	return nil, fmt.Errorf("unable to find block %v db", hash)
}

func (b *BlockChain) fetchBlockBytesByHash(hash *hash.Hash) ([]byte, error) {
	// Check orphan cache.
	block := b.GetOrphan(hash)
	if block != nil {
		return block.Bytes()
	}
	return b.consensus.DatabaseContext().GetBlockBytes(hash)
}

func (b *BlockChain) fetchHeaderByHash(hash *hash.Hash) (*types.BlockHeader, error) {
	// Check orphan cache.
	block := b.GetOrphan(hash)
	if block != nil {
		return &block.Block().Header, nil
	}

	header, err := dbFetchHeaderByHash(b.consensus.DatabaseContext(), hash)
	if err == nil && header != nil {
		return header, nil
	}
	return nil, fmt.Errorf("unable to find block header %v db %v", hash, err)
}

func (b *BlockChain) GetBlockHeader(ib meerdag.IBlock) *types.BlockHeader {
	if ib == nil {
		return nil
	}
	if ib.GetData() != nil {
		bn, ok := ib.GetData().(*BlockNode)
		if !ok {
			log.Error("block data type error", "hash", ib.GetHash().String())
			return nil
		}
		return bn.GetHeader()
	}
	header, err := b.fetchHeaderByHash(ib.GetHash())
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return header
}
