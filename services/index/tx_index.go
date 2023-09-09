// Copyright (c) 2016 The btcsuite developers
// Copyright (c) 2016-2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package index

import (
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
)

const (
	// txIndexName is the human-readable name for the index.
	txIndexName = "transaction index"
)

// TxIndex implements a transaction by hash index.  That is to say, it supports
// querying all transactions by their hash.
type TxIndex struct {
	consensus model.Consensus
}

// Init initializes the hash-based transaction index.  In particular, it finds
// the highest used block ID and stores it for later use when connecting or
// disconnecting blocks.
//
// This is part of the Indexer interface.
func (idx *TxIndex) Init() error {
	log.Info("Init", "index", idx.Name(), "block order", idx.consensus.BlockChain().GetMainOrder())
	return nil
}

// Name returns the human-readable name of the index.
//
// This is part of the Indexer interface.
func (idx *TxIndex) Name() string {
	return txIndexName
}

// ConnectBlock is invoked by the index manager when a new block has been
// connected to the main chain.  This indexer adds a hash-to-transaction mapping
// for every transaction in the passed block.
//
// This is part of the Indexer interface.
func (idx *TxIndex) ConnectBlock(sblock *types.SerializedBlock, block model.Block) error {
	return idx.consensus.DatabaseContext().PutTxIdxEntrys(sblock, block)
}

// DisconnectBlock is invoked by the index manager when a block has been
// disconnected from the main chain.  This indexer removes the
// hash-to-transaction mapping for every transaction in the block.
//
// This is part of the Indexer interface.
func (idx *TxIndex) DisconnectBlock(block *types.SerializedBlock) error {
	return idx.consensus.DatabaseContext().DeleteTxIdxEntrys(block)
}

// NewTxIndex returns a new instance of an indexer that is used to create a
// mapping of the hashes of all transactions in the blockchain to the respective
// block, location within the block, and size of the transaction.
//
// It implements the Indexer interface which plugs into the IndexManager that in
// turn is used by the blockchain package.  This allows the index to be
// seamlessly maintained along with the chain.
func NewTxIndex(consensus model.Consensus) *TxIndex {
	return &TxIndex{consensus: consensus}
}
