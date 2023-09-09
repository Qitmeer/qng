// Copyright (c) 2017-2018 The qitmeer developers

package index

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
)

// Manager defines an index manager that manages multiple optional indexes and
// implements the blockchain.IndexManager interface so it can be seamlessly
// plugged into normal chain processing.
type Manager struct {
	consensus      model.Consensus
	cfg            *Config
	db             legacydb.DB
	enabledIndexes []Indexer
	invalidtxIndex *InvalidTxIndex
	txIndex        *TxIndex
	txhashIndex    *TxHashIndex
}

// Ensure the Manager type implements the blockchain.IndexManager interface.
var _ model.IndexManager = (*Manager)(nil)

// NewManager returns a new index manager with the provided indexes enabled.
//
// The manager returned satisfies the blockchain.IndexManager interface and thus
// cleanly plugs into the normal blockchain processing path.
func NewManager(cfg *Config, consensus model.Consensus) *Manager {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	// Create the transaction and address indexes if needed.
	var indexers []Indexer
	if cfg.AddrIndex {
		addrIndex := NewAddrIndex(consensus)
		indexers = append(indexers, addrIndex)
	}
	for _, indexer := range indexers {
		log.Info(fmt.Sprintf("%s is enabled", indexer.Name()))
	}
	im := &Manager{
		cfg:            cfg,
		db:             consensus.LegacyDB(),
		enabledIndexes: indexers,
		consensus:      consensus,
		txIndex:        NewTxIndex(consensus),
	}
	if cfg.InvalidTxIndex {
		im.invalidtxIndex = NewInvalidTxIndex(consensus)
	}
	if cfg.TxhashIndex {
		im.txhashIndex = NewTxHashIndex(consensus)
	}
	return im
}

// Init initializes the enabled indexes.  This is called during chain
// initialization and primarily consists of catching up all indexes to the
// current best chain tip.  This is necessary since each index can be disabled
// and re-enabled at any time and attempting to catch-up indexes at the same
// time new blocks are being downloaded would lead to an overall longer time to
// catch up due to the I/O contention.
//
// This is part of the blockchain.IndexManager interface.
func (m *Manager) Init() error {
	interrupt := m.consensus.Interrupt()
	m.txIndex.Init()
	if m.invalidtxIndex != nil {
		err := m.invalidtxIndex.Init()
		if err != nil {
			return err
		}
	}
	if m.txhashIndex != nil {
		err := m.txhashIndex.Init()
		if err != nil {
			return err
		}
	}
	if system.InterruptRequested(interrupt) {
		return errInterruptRequested
	}

	// Initialize each of the enabled indexes.
	for _, indexer := range m.enabledIndexes {
		if err := indexer.Init(); err != nil {
			return err
		}
	}
	return nil
}

// ConnectBlock must be invoked when a block is extending the main chain.  It
// keeps track of the state of each index it is managing, performs some sanity
// checks, and invokes each indexer.
//
// This is part of the blockchain.IndexManager interface.
func (m *Manager) ConnectBlock(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	for _, index := range m.enabledIndexes {
		err := index.ConnectBlock(sblock, block, stxos)
		if err != nil {
			return err
		}
	}

	if m.txhashIndex != nil {
		err := m.txhashIndex.ConnectBlock(sblock, block)
		if err != nil {
			return err
		}
	}
	if block.GetState().GetStatus().KnownInvalid() {
		if m.invalidtxIndex != nil {
			return m.invalidtxIndex.ConnectBlock(sblock, block)
		}
	} else {
		return m.txIndex.ConnectBlock(sblock, block)
	}
	return nil
}

// DisconnectBlock must be invoked when a block is being disconnected from the
// end of the main chain.  It keeps track of the state of each index it is
// managing, performs some sanity checks, and invokes each indexer to remove
// the index entries associated with the block.
//
// This is part of the blockchain.IndexManager interface.
func (m *Manager) DisconnectBlock(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	// Call each of the currently active optional indexes with the block
	// being disconnected so they can update accordingly.
	for _, index := range m.enabledIndexes {
		err := index.DisconnectBlock(sblock, block, stxos)
		if err != nil {
			return err
		}
	}

	// TODO: Future optimization points is the attribute of KnownInvalid
	err := m.txIndex.DisconnectBlock(sblock)
	if err != nil {
		return err
	}
	if m.txhashIndex != nil {
		err := m.txhashIndex.DisconnectBlock(sblock)
		if err != nil {
			return err
		}
	}
	if m.invalidtxIndex != nil {
		return m.invalidtxIndex.DisconnectBlock(sblock, block)
	}
	return nil
}

func (m *Manager) UpdateMainTip(bh *hash.Hash, order uint64) error {
	if m.invalidtxIndex != nil {
		return m.invalidtxIndex.UpdateMainTip(bh, order)
	}
	return nil
}

// HasTransaction
func (m *Manager) IsDuplicateTx(txid *hash.Hash, blockHash *hash.Hash) bool {
	_, bh, err := m.consensus.DatabaseContext().GetTxIdxEntry(txid, false)
	if err != nil {
		return false
	}
	if bh == nil {
		return false
	}
	if bh.IsEqual(blockHash) {
		return false
	}
	return true
}

func (m *Manager) HasTx(txid *hash.Hash) bool {
	_, blockhash, err := m.consensus.DatabaseContext().GetTxIdxEntry(txid, false)
	if err == nil && blockhash != nil {
		return true
	}
	return false
}

func (m *Manager) TxIndex() *TxIndex {
	return m.txIndex
}

func (m *Manager) AddrIndex() *AddrIndex {
	indexer := m.GetIndex(AddrIndexName)
	if indexer != nil {
		return indexer.(*AddrIndex)
	}
	return nil
}

func (m *Manager) InvalidTxIndex() *InvalidTxIndex {
	return m.invalidtxIndex
}

func (m *Manager) TxHashIndex() *TxHashIndex {
	return m.txhashIndex
}

func (m *Manager) GetIndex(name string) Indexer {
	for _, index := range m.enabledIndexes {
		if index.Name() == name {
			return index
		}
	}
	return nil
}
