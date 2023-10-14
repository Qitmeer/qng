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

	// Size of a transaction entry.  It consists of 4 bytes block id + 4
	// bytes offset + 4 bytes length.
	txEntrySize = 4 + 4 + 4

	// level0MaxEntries is the maximum number of transactions that are
	// stored in level 0 of an address index entry.  Subsequent levels store
	// 2^n * level0MaxEntries entries, or in words, double the maximum of
	// the previous level.
	level0MaxEntries = 8

	// levelKeySize is the number of bytes a level key in the address index
	// consumes.  It consists of the address key + 1 byte for the level.
	levelKeySize = AddrKeySize + 1

	// levelOffset is the offset in the level key which identifes the level.
	levelOffset = levelKeySize - 1
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
func AddrToKey(addr types.Address) ([AddrKeySize]byte, error) {
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
		log.Error(err.Error())
		return nil
	}
	if order != math.MaxUint32 && order+1 != block.GetOrder() ||
		order == math.MaxUint32 && block.GetOrder() != 0 {
		log.Warn(fmt.Sprintf("dbIndexConnectBlock must be "+
			"called with a block that extends the current index "+
			"tip (%s, tip %d, block %d)", idx.Name(),
			order, block.GetOrder()))
		return nil
	}
	if !block.GetState().GetStatus().KnownInvalid() {
		err = idx.DB().PutAddrIdx(sblock, block, stxos)
		if err != nil {
			log.Error(err.Error())
			return nil
		}
	}
	// Update the current index tip.
	err = idx.DB().PutAddrIdxTip(sblock.Hash(), block.GetOrder())
	if err != nil {
		log.Error(err.Error())
	}
	return nil
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
		log.Error(err.Error())
		return nil
	}
	if !curTipHash.IsEqual(sblock.Hash()) {
		log.Warn(fmt.Sprintf("dbIndexDisconnectBlock must "+
			"be called with the block at the current index tip "+
			"(%s, tip %s, block %s)", idx.Name(),
			curTipHash, sblock.Hash()))
	}
	if order == math.MaxUint32 {
		log.Warn("Can't disconnect root index tip")
		return nil
	}
	// Notify the indexer with the disconnected block so it can remove all
	// of the appropriate entries.
	if err := idx.DB().DeleteAddrIdx(sblock, stxos); err != nil {
		log.Warn(err.Error())
		return nil
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
			log.Warn(fmt.Sprintf("No block:%d", preOrder))
			return nil
		}
		prevHash = pblock.GetHash()
	}

	err = idx.DB().PutAddrIdxTip(prevHash, preOrder)
	if err != nil {
		log.Error(err.Error())
	}
	return nil
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
		addrKey, err := AddrToKey(addr)
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
	addrKey, err := AddrToKey(addr)
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

// InternalBucket is an abstraction over a database bucket.  It is used to make
// the code easier to test since it allows mock objects in the tests to only
// implement these functions instead of everything a database.Bucket supports.
type InternalBucket interface {
	Get(key []byte) []byte
	Put(key []byte, value []byte) error
	Delete(key []byte) error
}

// WriteAddrIdxData represents the address index data to be written for one block.
// It consists of the address mapped to an ordered list of the transactions
// that involve the address in block.  It is ordered so the transactions can be
// stored in the order they appear in the block.
type WriteAddrIdxData map[[AddrKeySize]byte][]int

type FetchAddrLevelDataFunc func(key []byte) []byte

// fetchBlockHashFunc defines a callback function to use in order to convert a
// serialized block ID to an associated block hash.
type FetchBlockHashFunc func(serializedID []byte) (*hash.Hash, error)

type BlockRegion struct {
	Hash   *hash.Hash
	Offset uint32
	Len    uint32
}

// addrIndexPkScript extracts all standard addresses from the passed public key
// script and maps each of them to the associated transaction using the passed
// map.
func addrIndexPkScript(data WriteAddrIdxData, pkScript []byte, txIdx int) {
	// Nothing to index if the script is non-standard or otherwise doesn't
	// contain any addresses.
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript, params.ActiveNetParams.Params)
	if err != nil {
		return
	}

	if len(addrs) == 0 {
		return
	}

	for _, addr := range addrs {
		addrKey, err := AddrToKey(addr)
		if err != nil {
			// Ignore unsupported address types.
			continue
		}

		// Avoid inserting the transaction more than once.  Since the
		// transactions are indexed serially any duplicates will be
		// indexed in a row, so checking the most recent entry for the
		// address is enough to detect duplicates.
		indexedTxns := data[addrKey]
		numTxns := len(indexedTxns)
		if numTxns > 0 && indexedTxns[numTxns-1] == txIdx {
			continue
		}
		indexedTxns = append(indexedTxns, txIdx)
		data[addrKey] = indexedTxns
	}
}

// AddrIndexBlock extract all of the standard addresses from all of the transactions
// in the parent of the passed block (if they were valid) and all of the stake
// transactions in the passed block, and maps each of them to the associated
// transaction using the passed map.
func AddrIndexBlock(data WriteAddrIdxData, block *types.SerializedBlock, stxos [][]byte) {
	index := 0
	for txIdx, tx := range block.Transactions() {
		if tx.IsDuplicate {
			continue
		}
		// Coinbases do not reference any inputs.  Since the block is
		// required to have already gone through full validation, it has
		// already been proven on the first transaction in the block is
		// a coinbase.
		if txIdx != 0 {
			if len(stxos) == 0 {
				return
			}
			for range tx.Transaction().TxIn {
				if index >= len(stxos) {
					return
				}
				stxo := stxos[index]
				index++
				addrIndexPkScript(data, stxo, txIdx)
			}
		}

		for _, txOut := range tx.Transaction().TxOut {
			addrIndexPkScript(data, txOut.PkScript, txIdx)
		}
	}

}

// keyForLevel returns the key for a specific address and level in the address
// index entry.
func keyForLevel(addrKey [AddrKeySize]byte, level uint8, prefix []byte) []byte {
	if len(prefix) <= 0 {
		var key [levelKeySize]byte
		copy(key[:], addrKey[:])
		key[levelOffset] = level
		return key[:]
	}
	key := []byte{}
	key = append(key, prefix...)
	key = append(key, addrKey[:]...)
	key = append(key, level)
	return key
}

// serializeAddrIndexEntry serializes the provided block id and transaction
// location according to the format described in detail above.
func serializeAddrIndexEntry(blockID uint32, txLoc types.TxLoc) []byte {
	// Serialize the entry.
	serialized := make([]byte, 12)
	byteOrder.PutUint32(serialized, blockID)
	byteOrder.PutUint32(serialized[4:], uint32(txLoc.TxStart))
	byteOrder.PutUint32(serialized[8:], uint32(txLoc.TxLen))
	return serialized
}

// deserializeAddrIndexEntry decodes the passed serialized byte slice into the
// provided region struct according to the format described in detail above and
// uses the passed block hash fetching function in order to conver the block ID
// to the associated block hash.
func deserializeAddrIndexEntry(serialized []byte, region *BlockRegion, fetchBlockHash FetchBlockHashFunc) error {
	// Ensure there are enough bytes to decode.
	if len(serialized) < txEntrySize {
		return model.ErrDeserialize("unexpected end of data")
	}

	hash, err := fetchBlockHash(serialized[0:4])
	if err != nil {
		return err
	}
	region.Hash = hash
	region.Offset = byteOrder.Uint32(serialized[4:8])
	region.Len = byteOrder.Uint32(serialized[8:12])
	return nil
}

// DBPutAddrIndexEntry updates the address index to include the provided entry
// according to the level-based scheme described in detail above.
func DBPutAddrIndexEntry(bucket InternalBucket, addrKey [AddrKeySize]byte, blockID uint32, txLoc types.TxLoc, prefix []byte) error {
	// Start with level 0 and its initial max number of entries.
	curLevel := uint8(0)
	maxLevelBytes := level0MaxEntries * txEntrySize

	// Simply append the new entry to level 0 and return now when it will
	// fit.  This is the most common path.
	newData := serializeAddrIndexEntry(blockID, txLoc)
	level0Key := keyForLevel(addrKey, 0, prefix)
	level0Data := bucket.Get(level0Key[:])
	if len(level0Data)+len(newData) <= maxLevelBytes {
		mergedData := newData
		if len(level0Data) > 0 {
			mergedData = make([]byte, len(level0Data)+len(newData))
			copy(mergedData, level0Data)
			copy(mergedData[len(level0Data):], newData)
		}
		return bucket.Put(level0Key[:], mergedData)
	}

	// At this point, level 0 is full, so merge each level into higher
	// levels as many times as needed to free up level 0.
	prevLevelData := level0Data
	for {
		// Each new level holds twice as much as the previous one.
		curLevel++
		maxLevelBytes *= 2

		// Move to the next level as long as the current level is full.
		curLevelKey := keyForLevel(addrKey, curLevel, prefix)
		curLevelData := bucket.Get(curLevelKey[:])
		if len(curLevelData) == maxLevelBytes {
			prevLevelData = curLevelData
			continue
		}

		// The current level has room for the data in the previous one,
		// so merge the data from previous level into it.
		mergedData := prevLevelData
		if len(curLevelData) > 0 {
			mergedData = make([]byte, len(curLevelData)+
				len(prevLevelData))
			copy(mergedData, curLevelData)
			copy(mergedData[len(curLevelData):], prevLevelData)
		}
		err := bucket.Put(curLevelKey[:], mergedData)
		if err != nil {
			return err
		}

		// Move all of the levels before the previous one up a level.
		for mergeLevel := curLevel - 1; mergeLevel > 0; mergeLevel-- {
			mergeLevelKey := keyForLevel(addrKey, mergeLevel, prefix)
			prevLevelKey := keyForLevel(addrKey, mergeLevel-1, prefix)
			prevData := bucket.Get(prevLevelKey[:])
			err := bucket.Put(mergeLevelKey[:], prevData)
			if err != nil {
				return err
			}
		}
		break
	}

	// Finally, insert the new entry into level 0 now that it is empty.
	return bucket.Put(level0Key[:], newData)
}

// DBFetchAddrIndexEntries returns block regions for transactions referenced by
// the given address key and the number of entries skipped since it could have
// been less in the case where there are less total entries than the requested
// number of entries to skip.
func DBFetchAddrIndexEntries(fetchBlockHash FetchBlockHashFunc, fetchAddrLevelData FetchAddrLevelDataFunc, addrKey [AddrKeySize]byte, numToSkip, numRequested uint32, reverse bool, prefix []byte) ([]BlockRegion, uint32, error) {
	// When the reverse flag is not set, all levels need to be fetched
	// because numToSkip and numRequested are counted from the oldest
	// transactions (highest level) and thus the total count is needed.
	// However, when the reverse flag is set, only enough records to satisfy
	// the requested amount are needed.
	var level uint8
	var serialized []byte
	for !reverse || len(serialized) < int(numToSkip+numRequested)*txEntrySize {
		curLevelKey := keyForLevel(addrKey, level, prefix)
		levelData := fetchAddrLevelData(curLevelKey[:])
		if levelData == nil {
			// Stop when there are no more levels.
			break
		}

		// Higher levels contain older transactions, so prepend them.
		prepended := make([]byte, len(serialized)+len(levelData))
		copy(prepended, levelData)
		copy(prepended[len(levelData):], serialized)
		serialized = prepended
		level++
	}

	// When the requested number of entries to skip is larger than the
	// number available, skip them all and return now with the actual number
	// skipped.
	numEntries := uint32(len(serialized) / txEntrySize)
	if numToSkip >= numEntries {
		return nil, numEntries, nil
	}

	// Nothing more to do when there are no requested entries.
	if numRequested == 0 {
		return nil, numToSkip, nil
	}

	// Limit the number to load based on the number of available entries,
	// the number to skip, and the number requested.
	numToLoad := numEntries - numToSkip
	if numToLoad > numRequested {
		numToLoad = numRequested
	}

	// Start the offset after all skipped entries and load the calculated
	// number.
	results := make([]BlockRegion, numToLoad)
	for i := uint32(0); i < numToLoad; i++ {
		// Calculate the read offset according to the reverse flag.
		var offset uint32
		if reverse {
			offset = (numEntries - numToSkip - i - 1) * txEntrySize
		} else {
			offset = (numToSkip + i) * txEntrySize
		}

		// Deserialize and populate the result.
		err := deserializeAddrIndexEntry(serialized[offset:],
			&results[i], fetchBlockHash)
		if err != nil {
			// Ensure any deserialization errors are returned as
			// database corruption errors.
			if model.IsDeserializeErr(err) {
				err = legacydb.Error{
					ErrorCode: legacydb.ErrCorruption,
					Description: fmt.Sprintf("failed to "+
						"deserialized address index "+
						"for key %x: %v", addrKey, err),
				}
			}

			return nil, 0, err
		}
	}

	return results, numToSkip, nil
}

// -----------------------------------------------------------------------------
// The address index maps addresses referenced in the blockchain to a list of
// all the transactions involving that address.  Transactions are stored
// according to their order of appearance in the blockchain.  That is to say
// first by block height and then by offset inside the block.  It is also
// important to note that this implementation requires the transaction index
// since it is needed in order to catch up old blocks due to the fact the spent
// outputs will already be pruned from the utxo set.
//
// The approach used to store the index is similar to a log-structured merge
// tree (LSM tree) and is thus similar to how leveldb works internally.
//
// Every address consists of one or more entries identified by a level starting
// from 0 where each level holds a maximum number of entries such that each
// subsequent level holds double the maximum of the previous one.  In equation
// form, the number of entries each level holds is 2^n * firstLevelMaxSize.
//
// New transactions are appended to level 0 until it becomes full at which point
// the entire level 0 entry is appended to the level 1 entry and level 0 is
// cleared.  This process continues until level 1 becomes full at which point it
// will be appended to level 2 and cleared and so on.
//
// The result of this is the lower levels contain newer transactions and the
// transactions within each level are ordered from oldest to newest.
//
// The intent of this approach is to provide a balance between space efficiency
// and indexing cost.  Storing one entry per transaction would have the lowest
// indexing cost, but would waste a lot of space because the same address hash
// would be duplicated for every transaction key.  On the other hand, storing a
// single entry with all transactions would be the most space efficient, but
// would cause indexing cost to grow quadratically with the number of
// transactions involving the same address.  The approach used here provides
// logarithmic insertion and retrieval.
//
// The serialized key format is:
//
//   <addr type><addr hash><level>
//
//   Field           Type      Size
//   addr type       uint8     1 byte
//   addr hash       hash160   20 bytes
//   level           uint8     1 byte
//   -----
//   Total: 22 bytes
//
// The serialized value format is:
//
//   [<block id><start offset><tx length>,...]
//
//   Field           Type      Size
//   block id        uint32    4 bytes
//   start offset    uint32    4 bytes
//   tx length       uint32    4 bytes
//   -----
//   Total: 12 bytes per indexed tx
// -----------------------------------------------------------------------------

// minEntriesToReachLevel returns the minimum number of entries that are
// required to reach the given address index level.
func minEntriesToReachLevel(level uint8) int {
	maxEntriesForLevel := level0MaxEntries
	minRequired := 1
	for l := uint8(1); l <= level; l++ {
		minRequired += maxEntriesForLevel
		maxEntriesForLevel *= 2
	}
	return minRequired
}

// maxEntriesForLevel returns the maximum number of entries allowed for the
// given address index level.
func maxEntriesForLevel(level uint8) int {
	numEntries := level0MaxEntries
	for l := level; l > 0; l-- {
		numEntries *= 2
	}
	return numEntries
}

// DBRemoveAddrIndexEntries removes the specified number of entries from from
// the address index for the provided key.  An assertion error will be returned
// if the count exceeds the total number of entries in the index.
func DBRemoveAddrIndexEntries(bucket InternalBucket, addrKey [AddrKeySize]byte, count int, prefix []byte) error {
	// Nothing to do if no entries are being deleted.
	if count <= 0 {
		return nil
	}

	// Make use of a local map to track pending updates and define a closure
	// to apply it to the database.  This is done in order to reduce the
	// number of database reads and because there is more than one exit
	// path that needs to apply the updates.
	pendingUpdates := make(map[uint8][]byte)
	applyPending := func() error {
		for level, data := range pendingUpdates {
			curLevelKey := keyForLevel(addrKey, level, prefix)
			if len(data) == 0 {
				err := bucket.Delete(curLevelKey[:])
				if err != nil {
					return err
				}
				continue
			}
			err := bucket.Put(curLevelKey[:], data)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Loop forwards through the levels while removing entries until the
	// specified number has been removed.  This will potentially result in
	// entirely empty lower levels which will be backfilled below.
	var highestLoadedLevel uint8
	numRemaining := count
	for level := uint8(0); numRemaining > 0; level++ {
		// Load the data for the level from the database.
		curLevelKey := keyForLevel(addrKey, level, prefix)
		curLevelData := bucket.Get(curLevelKey[:])
		if len(curLevelData) == 0 && numRemaining > 0 {
			return model.AssertError(fmt.Sprintf("dbRemoveAddrIndexEntries "+
				"not enough entries for address key %x to "+
				"delete %d entries", addrKey, count))
		}
		pendingUpdates[level] = curLevelData
		highestLoadedLevel = level

		// Delete the entire level as needed.
		numEntries := len(curLevelData) / txEntrySize
		if numRemaining >= numEntries {
			pendingUpdates[level] = nil
			numRemaining -= numEntries
			continue
		}

		// Remove remaining entries to delete from the level.
		offsetEnd := len(curLevelData) - (numRemaining * txEntrySize)
		pendingUpdates[level] = curLevelData[:offsetEnd]
		break
	}

	// When all elements in level 0 were not removed there is nothing left
	// to do other than updating the database.
	if len(pendingUpdates[0]) != 0 {
		return applyPending()
	}

	// At this point there are one or more empty levels before the current
	// level which need to be backfilled and the current level might have
	// had some entries deleted from it as well.  Since all levels after
	// level 0 are required to either be empty, half full, or completely
	// full, the current level must be adjusted accordingly by backfilling
	// each previous levels in a way which satisfies the requirements.  Any
	// entries that are left are assigned to level 0 after the loop as they
	// are guaranteed to fit by the logic in the loop.  In other words, this
	// effectively squashes all remaining entries in the current level into
	// the lowest possible levels while following the level rules.
	//
	// Note that the level after the current level might also have entries
	// and gaps are not allowed, so this also keeps track of the lowest
	// empty level so the code below knows how far to backfill in case it is
	// required.
	lowestEmptyLevel := uint8(255)
	curLevelData := pendingUpdates[highestLoadedLevel]
	curLevelMaxEntries := maxEntriesForLevel(highestLoadedLevel)
	for level := highestLoadedLevel; level > 0; level-- {
		// When there are not enough entries left in the current level
		// for the number that would be required to reach it, clear the
		// the current level which effectively moves them all up to the
		// previous level on the next iteration.  Otherwise, there are
		// are sufficient entries, so update the current level to
		// contain as many entries as possible while still leaving
		// enough remaining entries required to reach the level.
		numEntries := len(curLevelData) / txEntrySize
		prevLevelMaxEntries := curLevelMaxEntries / 2
		minPrevRequired := minEntriesToReachLevel(level - 1)
		if numEntries < prevLevelMaxEntries+minPrevRequired {
			lowestEmptyLevel = level
			pendingUpdates[level] = nil
		} else {
			// This level can only be completely full or half full,
			// so choose the appropriate offset to ensure enough
			// entries remain to reach the level.
			var offset int
			if numEntries-curLevelMaxEntries >= minPrevRequired {
				offset = curLevelMaxEntries * txEntrySize
			} else {
				offset = prevLevelMaxEntries * txEntrySize
			}
			pendingUpdates[level] = curLevelData[:offset]
			curLevelData = curLevelData[offset:]
		}

		curLevelMaxEntries = prevLevelMaxEntries
	}
	pendingUpdates[0] = curLevelData
	if len(curLevelData) == 0 {
		lowestEmptyLevel = 0
	}

	// When the highest loaded level is empty, it's possible the level after
	// it still has data and thus that data needs to be backfilled as well.
	for len(pendingUpdates[highestLoadedLevel]) == 0 {
		// When the next level is empty too, the is no data left to
		// continue backfilling, so there is nothing left to do.
		// Otherwise, populate the pending updates map with the newly
		// loaded data and update the highest loaded level accordingly.
		level := highestLoadedLevel + 1
		curLevelKey := keyForLevel(addrKey, level, prefix)
		levelData := bucket.Get(curLevelKey[:])
		if len(levelData) == 0 {
			break
		}
		pendingUpdates[level] = levelData
		highestLoadedLevel = level

		// At this point the highest level is not empty, but it might
		// be half full.  When that is the case, move it up a level to
		// simplify the code below which backfills all lower levels that
		// are still empty.  This also means the current level will be
		// empty, so the loop will perform another another iteration to
		// potentially backfill this level with data from the next one.
		curLevelMaxEntries := maxEntriesForLevel(level)
		if len(levelData)/txEntrySize != curLevelMaxEntries {
			pendingUpdates[level] = nil
			pendingUpdates[level-1] = levelData
			level--
			curLevelMaxEntries /= 2
		}

		// Backfill all lower levels that are still empty by iteratively
		// halfing the data until the lowest empty level is filled.
		for level > lowestEmptyLevel {
			offset := (curLevelMaxEntries / 2) * txEntrySize
			pendingUpdates[level] = levelData[:offset]
			levelData = levelData[offset:]
			pendingUpdates[level-1] = levelData
			level--
			curLevelMaxEntries /= 2
		}

		// The lowest possible empty level is now the highest loaded
		// level.
		lowestEmptyLevel = highestLoadedLevel
	}

	// Apply the pending updates.
	return applyPending()
}
