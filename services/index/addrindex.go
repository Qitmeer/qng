// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2016-2017 The Decred developers
// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package index

import (
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/services/progresslog"
	"math"
	"sync"

	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

const (
	// addrIndexName is the human-readable name for the index.
	AddrIndexName = "address index"

	// addrKeySize is the number of bytes an address key consumes in the
	// index.  It consists of 1 byte address type + 20 bytes hash160.
	AddrKeySize = 1 + 20

	// addrKeyTypePubKeyHash is the address type in an address key which
	// represents both a pay-to-pubkey-hash and a pay-to-pubkey address.
	// This is done because both are identical for the purposes of the
	// address index.
	addrKeyTypePubKeyHash = 0

	// addrKeyTypePubKeyHashEdwards is the address type in an address key
	// which represents both a pay-to-pubkey-hash and a pay-to-pubkey-alt
	// address using Schnorr signatures over the Ed25519 curve.  This is
	// done because both are identical for the purposes of the address
	// index.
	addrKeyTypePubKeyHashEdwards = 1

	// addrKeyTypePubKeyHashSchnorr is the address type in an address key
	// which represents both a pay-to-pubkey-hash and a pay-to-pubkey-alt
	// address using Schnorr signatures over the secp256k1 curve.  This is
	// done because both are identical for the purposes of the address
	// index.
	addrKeyTypePubKeyHashSchnorr = 2

	// addrKeyTypeScriptHash is the address type in an address key which
	// represents a pay-to-script-hash address.  This is necessary because
	// the hash of a pubkey address might be the same as that of a script
	// hash.
	addrKeyTypeScriptHash = 3
)

var (
	// addrIndexKey is the key of the address index and the db bucket used
	// to house it.
	addrIndexKey = []byte("txbyaddridx")

	// errUnsupportedAddressType is an error that is used to signal an
	// unsupported address type has been used.
	errUnsupportedAddressType = errors.New("address type is not supported " +
		"by the address index")
)

// addrToKey converts known address types to an addrindex key.  An error is
// returned for unsupported types.
func AddrToKey(addr types.Address, params *params.Params) ([AddrKeySize]byte, error) {
	switch addr := addr.(type) {
	case *address.PubKeyHashAddress:
		switch addr.EcType() {
		case ecc.ECDSA_Secp256k1:
			var result [AddrKeySize]byte
			result[0] = addrKeyTypePubKeyHash
			copy(result[1:], addr.Hash160()[:])
			return result, nil
		case ecc.EdDSA_Ed25519:
			var result [AddrKeySize]byte
			result[0] = addrKeyTypePubKeyHashEdwards
			copy(result[1:], addr.Hash160()[:])
			return result, nil
		case ecc.ECDSA_SecpSchnorr:
			var result [AddrKeySize]byte
			result[0] = addrKeyTypePubKeyHashSchnorr
			copy(result[1:], addr.Hash160()[:])
			return result, nil
		}

	case *address.ScriptHashAddress:
		var result [AddrKeySize]byte
		result[0] = addrKeyTypeScriptHash
		copy(result[1:], addr.Hash160()[:])
		return result, nil

	case *address.SecpPubKeyAddress:
		var result [AddrKeySize]byte
		result[0] = addrKeyTypePubKeyHash
		copy(result[1:], addr.PKHAddress().Hash160()[:])
		return result, nil

	case *address.EdwardsPubKeyAddress:
		var result [AddrKeySize]byte
		result[0] = addrKeyTypePubKeyHashEdwards
		copy(result[1:], addr.PKHAddress().Hash160()[:])
		return result, nil

	case *address.SecSchnorrPubKeyAddress:
		var result [AddrKeySize]byte
		result[0] = addrKeyTypePubKeyHashSchnorr
		copy(result[1:], addr.PKHAddress().Hash160()[:])
		return result, nil
	}

	return [AddrKeySize]byte{}, errUnsupportedAddressType
}

// AddrIndex implements a transaction by address index.  That is to say, it
// supports querying all transactions that reference a given address because
// they are either crediting or debiting the address.  The returned transactions
// are ordered according to their order of appearance in the blockchain.  In
// other words, first by block height and then by offset inside the block.
//
// In addition, support is provided for a memory-only index of unconfirmed
// transactions such as those which are kept in the memory pool before inclusion
// in a block.
type AddrIndex struct {
	consensus model.Consensus

	// The following fields are set when the instance is created and can't
	// be changed afterwards, so there is no need to protect them with a
	// separate mutex.
	db          legacydb.DB
	chainParams *params.Params

	// The following fields are used to quickly link transactions and
	// addresses that have not been included into a block yet when an
	// address index is being maintained.  The are protected by the
	// unconfirmedLock field.
	//
	// The txnsByAddr field is used to keep an index of all transactions
	// which either create an output to a given address or spend from a
	// previous output to it keyed by the address.
	//
	// The addrsByTx field is essentially the reverse and is used to
	// keep an index of all addresses which a given transaction involves.
	// This allows fairly efficient updates when transactions are removed
	// once they are included into a block.
	unconfirmedLock sync.RWMutex
	txnsByAddr      map[[AddrKeySize]byte]map[hash.Hash]*types.Tx
	addrsByTx       map[hash.Hash]map[[AddrKeySize]byte]struct{}
}

// Ensure the AddrIndex type implements the Indexer interface.
var _ Indexer = (*AddrIndex)(nil)

// Ensure the AddrIndex type implements the NeedsInputser interface.
var _ NeedsInputser = (*AddrIndex)(nil)

// NeedsInputs signals that the index requires the referenced inputs in order
// to properly create the index.
//
// This implements the NeedsInputser interface.
func (idx *AddrIndex) NeedsInputs() bool {
	return true
}

// Init is only provided to satisfy the Indexer interface as there is nothing to
// initialize for this index.
//
// This is part of the Indexer interface.
func (idx *AddrIndex) Init() error {
	err := idx.DB().CleanAddrIdx(true)
	// Finish any drops that were previously interrupted.
	if err != nil {
		return err
	}
	tipHash, tiporder, err := idx.DB().GetAddrIdxTip()
	if err != nil {
		return err
	}
	log.Info("Init", "index", idx.Name(), "tipHash", tipHash.String(), "tipOrder", tiporder)

	//
	chain := idx.consensus.BlockChain()
	bestOrder := chain.GetMainOrder()
	backorder := tiporder
	// Rollback indexes to the main chain if their tip is an orphaned fork.
	// This is fairly unlikely, but it can happen if the chain is
	// reorganized while the index is disabled.  This has to be done in
	// reverse order because later indexes can depend on earlier ones.
	var spentTxos [][]byte

	// Nothing to do if the index does not have any entries yet.
	if backorder != math.MaxUint32 {
		var block *types.SerializedBlock
		for backorder > bestOrder {
			// Load the block for the height since it is required to index
			// it.
			block, err = chain.BlockByOrder(uint64(backorder))
			if err != nil {
				return err
			}
			spentTxos = nil
			if idx.NeedsInputs() {
				spentTxos, err = chain.FetchSpendJournalPKS(block)
				if err != nil {
					return err
				}
			}
			err = idx.DisconnectBlock(block, nil, spentTxos)
			if err != nil {
				return err
			}
			log.Trace(fmt.Sprintf("%s rollback order= %d", idx.Name(), backorder))
			backorder--
			if system.InterruptRequested(idx.consensus.Interrupt()) {
				return errInterruptRequested
			}
		}
	}

	lowestOrder := int64(bestOrder)
	if tiporder == math.MaxUint32 {
		lowestOrder = -1
	} else if int64(tiporder) < lowestOrder {
		lowestOrder = int64(tiporder)
	}

	// Nothing to index if all of the indexes are caught up.
	if lowestOrder == int64(bestOrder) {
		return nil
	}

	// Create a progress logger for the indexing process below.
	progressLogger := progresslog.NewBlockProgressLogger("Indexed", log)

	// At this point, one or more indexes are behind the current best chain
	// tip and need to be caught up, so log the details and loop through
	// each block that needs to be indexed.
	log.Info(fmt.Sprintf("Catching up indexes from order %d to %d", lowestOrder,
		bestOrder))

	for order := lowestOrder + 1; order <= int64(bestOrder); order++ {
		if system.InterruptRequested(idx.consensus.Interrupt()) {
			return errInterruptRequested
		}

		var block *types.SerializedBlock
		var blk model.Block
		// Load the block for the height since it is required to index
		// it.
		block, blk, err = chain.FetchBlockByOrder(uint64(order))
		if err != nil {
			return err
		}

		if system.InterruptRequested(idx.consensus.Interrupt()) {
			return errInterruptRequested
		}
		chain.SetDAGDuplicateTxs(block, blk)
		// Connect the block for all indexes that need it.
		spentTxos = nil

		if spentTxos == nil && idx.NeedsInputs() {
			spentTxos, err = chain.FetchSpendJournalPKS(block)
			if err != nil {
				return err
			}
		}
		err = idx.ConnectBlock(block, blk, spentTxos)
		if err != nil {
			return err
		}
		progressLogger.LogBlockOrder(blk.GetOrder(), block)
	}

	log.Info(fmt.Sprintf("Indexes caught up to order %d", bestOrder))
	return nil
}

// Name returns the human-readable name of the index.
//
// This is part of the Indexer interface.
func (idx *AddrIndex) Name() string {
	return AddrIndexName
}

// ConnectBlock is invoked by the index manager when a new block has been
// connected to the main chain.  This indexer adds a mapping for each address
// the transactions in the block involve.
//
// This is part of the Indexer interface.
func (idx *AddrIndex) ConnectBlock(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	_, order, err := idx.DB().GetAddrIdxTip()
	if err != nil {
		return err
	}
	if order != math.MaxUint32 && order+1 != block.GetOrder() ||
		order == math.MaxUint32 && block.GetOrder() != 0 {
		return fmt.Errorf("dbIndexConnectBlock must be "+
			"called with a block that extends the current index "+
			"tip (%s, tip %d, block %d)", idx.Name(),
			order, block.GetOrder())
	}
	if !block.GetState().GetStatus().KnownInvalid() {
		err = idx.DB().PutAddrIdx(sblock, block, stxos)
		if err != nil {
			return err
		}
	}
	// Update the current index tip.
	return idx.DB().PutAddrIdxTip(sblock.Hash(), block.GetOrder())
}

// DisconnectBlock is invoked by the index manager when a block has been
// disconnected from the main chain.  This indexer removes the address mappings
// each transaction in the block involve.
//
// This is part of the Indexer interface.
func (idx *AddrIndex) DisconnectBlock(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	// Assert that the block being disconnected is the current tip of the
	// index.
	curTipHash, order, err := idx.DB().GetAddrIdxTip()
	if err != nil {
		return err
	}
	if !curTipHash.IsEqual(sblock.Hash()) {
		return fmt.Errorf("dbIndexDisconnectBlock must "+
			"be called with the block at the current index tip "+
			"(%s, tip %s, block %s)", idx.Name(),
			curTipHash, sblock.Hash())
	}
	if order == math.MaxUint32 {
		return fmt.Errorf("Can't disconnect root index tip")
	}
	// Notify the indexer with the disconnected block so it can remove all
	// of the appropriate entries.
	if err := idx.DB().DeleteAddrIdx(sblock, stxos); err != nil {
		return err
	}

	// Update the current index tip.
	var prevHash *hash.Hash
	var preOrder uint
	if order == 0 {
		prevHash = &hash.ZeroHash
		preOrder = math.MaxUint32
	} else {
		preOrder = order - 1

		pblock := idx.consensus.BlockChain().GetBlockByOrder(uint64(preOrder))
		if pblock == nil {
			return fmt.Errorf("No block:%d", preOrder)
		}
		prevHash = pblock.GetHash()
	}

	return idx.DB().PutAddrIdxTip(prevHash, preOrder)
}

// TxRegionsForAddress returns a slice of block regions which identify each
// transaction that involves the passed address according to the specified
// number to skip, number requested, and whether or not the results should be
// reversed.  It also returns the number actually skipped since it could be less
// in the case where there are not enough entries.
//
// NOTE: These results only include transactions confirmed in blocks.  See the
// UnconfirmedTxnsForAddress method for obtaining unconfirmed transactions
// that involve a given address.
//
// This function is safe for concurrent access.
func (idx *AddrIndex) TxRegionsForAddress(addr types.Address, numToSkip, numRequested uint32, reverse bool) ([]*common.RetrievedTx, uint32, error) {
	return idx.DB().GetTxForAddress(addr, numToSkip, numRequested, reverse)
}

// indexUnconfirmedAddresses modifies the unconfirmed (memory-only) address
// index to include mappings for the addresses encoded by the passed public key
// script to the transaction.
//
// This function is safe for concurrent access.
func (idx *AddrIndex) indexUnconfirmedAddresses(pkScript []byte, tx *types.Tx) {
	// The error is ignored here since the only reason it can fail is if the
	// script fails to parse and it was already validated before being
	// admitted to the mempool.
	_, addresses, _, _ := txscript.ExtractPkScriptAddrs(pkScript, idx.chainParams)

	for _, addr := range addresses {
		// Ignore unsupported address types.
		addrKey, err := AddrToKey(addr, idx.chainParams)
		if err != nil {
			continue
		}

		// Add a mapping from the address to the transaction.
		idx.unconfirmedLock.Lock()
		addrIndexEntry := idx.txnsByAddr[addrKey]
		if addrIndexEntry == nil {
			addrIndexEntry = make(map[hash.Hash]*types.Tx)
			idx.txnsByAddr[addrKey] = addrIndexEntry
		}
		addrIndexEntry[*tx.Hash()] = tx

		// Add a mapping from the transaction to the address.
		addrsByTxEntry := idx.addrsByTx[*tx.Hash()]
		if addrsByTxEntry == nil {
			addrsByTxEntry = make(map[[AddrKeySize]byte]struct{})
			idx.addrsByTx[*tx.Hash()] = addrsByTxEntry
		}
		addrsByTxEntry[addrKey] = struct{}{}
		idx.unconfirmedLock.Unlock()
	}
}

// AddUnconfirmedTx adds all addresses related to the transaction to the
// unconfirmed (memory-only) address index.
//
// NOTE: This transaction MUST have already been validated by the memory pool
// before calling this function with it and have all of the inputs available in
// the provided utxo view.  Failure to do so could result in some or all
// addresses not being indexed.
//
// This function is safe for concurrent access.
func (idx *AddrIndex) AddUnconfirmedTx(tx *types.Tx, pkScripts [][]byte) {
	for _, pks := range pkScripts {
		idx.indexUnconfirmedAddresses(pks, tx)
	}
}

// RemoveUnconfirmedTx removes the passed transaction from the unconfirmed
// (memory-only) address index.
//
// This function is safe for concurrent access.
func (idx *AddrIndex) RemoveUnconfirmedTx(hash *hash.Hash) {
	idx.unconfirmedLock.Lock()
	defer idx.unconfirmedLock.Unlock()

	// Remove all address references to the transaction from the address
	// index and remove the entry for the address altogether if it no longer
	// references any transactions.
	for addrKey := range idx.addrsByTx[*hash] {
		delete(idx.txnsByAddr[addrKey], *hash)
		if len(idx.txnsByAddr[addrKey]) == 0 {
			delete(idx.txnsByAddr, addrKey)
		}
	}

	// Remove the entry from the transaction to address lookup map as well.
	delete(idx.addrsByTx, *hash)
}

// UnconfirmedTxnsForAddress returns all transactions currently in the
// unconfirmed (memory-only) address index that involve the passed address.
// Unsupported address types are ignored and will result in no results.
//
// This function is safe for concurrent access.
func (idx *AddrIndex) UnconfirmedTxnsForAddress(addr types.Address) []*types.Tx {
	// Ignore unsupported address types.
	addrKey, err := AddrToKey(addr, idx.chainParams)
	if err != nil {
		return nil
	}

	// Protect concurrent access.
	idx.unconfirmedLock.RLock()
	defer idx.unconfirmedLock.RUnlock()

	// Return a new slice with the results if there are any.  This ensures
	// safe concurrency.
	if txns, exists := idx.txnsByAddr[addrKey]; exists {
		addressTxns := make([]*types.Tx, 0, len(txns))
		for _, tx := range txns {
			addressTxns = append(addressTxns, tx)
		}
		return addressTxns
	}

	return nil
}

func (idx *AddrIndex) DB() model.DataBase {
	return idx.consensus.DatabaseContext()
}

// NewAddrIndex returns a new instance of an indexer that is used to create a
// mapping of all addresses in the blockchain to the respective transactions
// that involve them.
//
// It implements the Indexer interface which plugs into the IndexManager that in
// turn is used by the blockchain package.  This allows the index to be
// seamlessly maintained along with the chain.
func NewAddrIndex(consensus model.Consensus) *AddrIndex {
	return &AddrIndex{
		consensus:   consensus,
		chainParams: params.ActiveNetParams.Params,
		txnsByAddr:  make(map[[AddrKeySize]byte]map[hash.Hash]*types.Tx),
		addrsByTx:   make(map[hash.Hash]map[[AddrKeySize]byte]struct{}),
	}
}
