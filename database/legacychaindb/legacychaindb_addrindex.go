package legacychaindb

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/services/index"
	"math"
)

const (

	// level0MaxEntries is the maximum number of transactions that are
	// stored in level 0 of an address index entry.  Subsequent levels store
	// 2^n * level0MaxEntries entries, or in words, double the maximum of
	// the previous level.
	level0MaxEntries = 8

	// levelKeySize is the number of bytes a level key in the address index
	// consumes.  It consists of the address key + 1 byte for the level.
	levelKeySize = index.AddrKeySize + 1

	// levelOffset is the offset in the level key which identifes the level.
	levelOffset = levelKeySize - 1
)

var (

	// addrIndexKey is the key of the address index and the db bucket used
	// to house it.
	addrIndexKey = []byte("txbyaddridx")
)

// writeIndexData represents the address index data to be written for one block.
// It consists of the address mapped to an ordered list of the transactions
// that involve the address in block.  It is ordered so the transactions can be
// stored in the order they appear in the block.
type writeIndexData map[[index.AddrKeySize]byte][]int

func (cdb *LegacyChainDB) GetAddrIdxTip() (*hash.Hash, uint, error) {
	err := cdb.db.Update(func(dbTx legacydb.Tx) error {
		// Create the bucket for the current tips as needed.
		meta := dbTx.Metadata()
		_, err := meta.CreateBucketIfNotExists(dbnamespace.IndexTipsBucketName)
		if err != nil {
			return err
		}
		indexesBucket := meta.Bucket(dbnamespace.IndexTipsBucketName)
		// Nothing to do if the index tip already exists.
		idxKey := addrIndexKey
		if indexesBucket.Get(idxKey) != nil {
			return nil
		}

		// The tip for the index does not exist, so create it and
		// invoke the create callback for the index so it can perform
		// any one-time initialization it requires.
		if _, err := meta.CreateBucket(idxKey); err != nil {
			return err
		}

		// Set the tip for the index to values which represent an
		// uninitialized index (the genesis block hash and height).
		return dbPutIndexerTip(dbTx, idxKey, &hash.ZeroHash, math.MaxUint32)
	})
	if err != nil {
		return nil, math.MaxUint32, err
	}
	var bh *hash.Hash
	var order uint32
	err = cdb.db.View(func(dbTx legacydb.Tx) error {
		bh, order, err = dbFetchIndexerTip(dbTx, addrIndexKey)
		return err
	})
	if err != nil {
		return nil, math.MaxUint32, err
	}
	return bh, uint(order), nil
}

func (cdb *LegacyChainDB) PutAddrIdxTip(bh *hash.Hash, order uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		return dbPutIndexerTip(dbTx, addrIndexKey, bh, uint32(order))
	})
}

func (cdb *LegacyChainDB) PutAddrIdx(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	// The offset and length of the transactions within the serialized
	// block.
	txLocs, err := sblock.TxLoc()
	if err != nil {
		return err
	}
	// Build all of the address to transaction mappings in a local map.
	addrsToTxns := make(writeIndexData)
	cdb.indexBlock(addrsToTxns, sblock, stxos)

	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		// Add all of the index entries for each address.
		addrIdxBucket := dbTx.Metadata().Bucket(addrIndexKey)
		for addrKey, txIdxs := range addrsToTxns {
			for _, txIdx := range txIdxs {
				// Switch to using the newest block ID for the stake transactions,
				// since these are not from the parent. Offset the index to be
				// correct for the location in this given block.
				err := dbPutAddrIndexEntry(addrIdxBucket, addrKey,
					uint32(block.GetID()), txLocs[txIdx])
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

}

func (cdb *LegacyChainDB) GetTxForAddress(addr types.Address, numToSkip, numRequested uint32, reverse bool) ([]*common.RetrievedTx, uint32, error) {
	addrKey, err := index.AddrToKey(addr, cdb.chainParams)
	if err != nil {
		return nil, 0, err
	}

	fetchBlockHash := func(id []byte) (*hash.Hash, error) {
		// Deserialize and populate the result.
		blockid := uint64(byteOrder.Uint32(id))
		return meerdag.DBGetDAGBlockHashByID(cdb, blockid)
	}

	fetchAddrLevelData := func(key []byte) []byte {
		var levelData []byte
		cdb.db.View(func(dbTx legacydb.Tx) error {
			addrIdxBucket := dbTx.Metadata().Bucket(addrIndexKey)
			levelData = addrIdxBucket.Get(key)
			return nil
		})
		return levelData
	}
	regions, dbSkipped, err := dbFetchAddrIndexEntries(fetchAddrLevelData, addrKey, numToSkip, numRequested, reverse, fetchBlockHash)
	if err != nil {
		return nil, 0, err
	}
	var serializedTxns [][]byte
	err = cdb.db.Update(func(dbTx legacydb.Tx) error {
		// Load the raw transaction bytes from the database.
		serializedTxns, err = dbTx.FetchBlockRegions(regions)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	addressTxns := []*common.RetrievedTx{}
	for i, serializedTx := range serializedTxns {
		addressTxns = append(addressTxns, &common.RetrievedTx{
			Bytes:   serializedTx,
			BlkHash: regions[i].Hash,
		})
	}
	return addressTxns, dbSkipped, nil
}

func (cdb *LegacyChainDB) DeleteAddrIdx(sblock *types.SerializedBlock, stxos [][]byte) error {
	// Build all of the address to transaction mappings in a local map.
	addrsToTxns := make(writeIndexData)
	cdb.indexBlock(addrsToTxns, sblock, stxos)

	// Remove all of the index entries for each address.
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(addrIndexKey)
		for addrKey, txIdxs := range addrsToTxns {
			err := dbRemoveAddrIndexEntries(bucket, addrKey, len(txIdxs))
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (cdb *LegacyChainDB) CleanAddrIdx(finish bool) error {
	if !finish {
		return dropIndex(cdb.db, addrIndexKey, index.AddrIndexName, cdb.interrupt)
	}
	indexNeedDrop := false
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		// None of the indexes needs to be dropped if the index tips
		// bucket hasn't been created yet.
		indexesBucket := dbTx.Metadata().Bucket(dbnamespace.IndexTipsBucketName)
		if indexesBucket == nil {
			return nil
		}

		// Mark the indexer as requiring a drop if one is already in
		// progress.
		dropKey := indexDropKey(addrIndexKey)
		if indexesBucket.Get(dropKey) != nil {
			indexNeedDrop = true
		}
		return nil
	})
	if err != nil {
		return err
	}

	if system.InterruptRequested(cdb.interrupt) {
		return errInterruptRequested
	}

	// Finish dropping any of the enabled indexes that are already in the
	// middle of being dropped.
	if indexNeedDrop {
		log.Info(fmt.Sprintf("Resuming %s drop", index.AddrIndexName))
		err := dropIndex(cdb.db, addrIndexKey, index.AddrIndexName, cdb.interrupt)
		if err != nil {
			return err
		}
	}

	return nil
}

// indexPkScript extracts all standard addresses from the passed public key
// script and maps each of them to the associated transaction using the passed
// map.
func (cdb *LegacyChainDB) indexPkScript(data writeIndexData, pkScript []byte, txIdx int) {
	// Nothing to index if the script is non-standard or otherwise doesn't
	// contain any addresses.
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript, cdb.chainParams)
	if err != nil {
		return
	}

	if len(addrs) == 0 {
		return
	}

	for _, addr := range addrs {
		addrKey, err := index.AddrToKey(addr, cdb.chainParams)
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

// indexBlock extract all of the standard addresses from all of the transactions
// in the parent of the passed block (if they were valid) and all of the stake
// transactions in the passed block, and maps each of them to the associated
// transaction using the passed map.
func (cdb *LegacyChainDB) indexBlock(data writeIndexData, block *types.SerializedBlock, stxos [][]byte) {
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
				cdb.indexPkScript(data, stxo, txIdx)
			}
		}

		for _, txOut := range tx.Transaction().TxOut {
			cdb.indexPkScript(data, txOut.PkScript, txIdx)
		}
	}

}

type fetchAddrLevelDataFunc func(key []byte) []byte

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

// fetchBlockHashFunc defines a callback function to use in order to convert a
// serialized block ID to an associated block hash.
type fetchBlockHashFunc func(serializedID []byte) (*hash.Hash, error)

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
func deserializeAddrIndexEntry(serialized []byte, region *legacydb.BlockRegion, fetchBlockHash fetchBlockHashFunc) error {
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

// keyForLevel returns the key for a specific address and level in the address
// index entry.
func keyForLevel(addrKey [index.AddrKeySize]byte, level uint8) [levelKeySize]byte {
	var key [levelKeySize]byte
	copy(key[:], addrKey[:])
	key[levelOffset] = level
	return key
}

// dbPutAddrIndexEntry updates the address index to include the provided entry
// according to the level-based scheme described in detail above.
func dbPutAddrIndexEntry(bucket internalBucket, addrKey [index.AddrKeySize]byte, blockID uint32, txLoc types.TxLoc) error {
	// Start with level 0 and its initial max number of entries.
	curLevel := uint8(0)
	maxLevelBytes := level0MaxEntries * txEntrySize

	// Simply append the new entry to level 0 and return now when it will
	// fit.  This is the most common path.
	newData := serializeAddrIndexEntry(blockID, txLoc)
	level0Key := keyForLevel(addrKey, 0)
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
		curLevelKey := keyForLevel(addrKey, curLevel)
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
			mergeLevelKey := keyForLevel(addrKey, mergeLevel)
			prevLevelKey := keyForLevel(addrKey, mergeLevel-1)
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

// dbFetchAddrIndexEntries returns block regions for transactions referenced by
// the given address key and the number of entries skipped since it could have
// been less in the case where there are less total entries than the requested
// number of entries to skip.
func dbFetchAddrIndexEntries(fetchAddrLevelData fetchAddrLevelDataFunc, addrKey [index.AddrKeySize]byte, numToSkip, numRequested uint32, reverse bool, fetchBlockHash fetchBlockHashFunc) ([]legacydb.BlockRegion, uint32, error) {
	// When the reverse flag is not set, all levels need to be fetched
	// because numToSkip and numRequested are counted from the oldest
	// transactions (highest level) and thus the total count is needed.
	// However, when the reverse flag is set, only enough records to satisfy
	// the requested amount are needed.
	var level uint8
	var serialized []byte
	for !reverse || len(serialized) < int(numToSkip+numRequested)*txEntrySize {
		curLevelKey := keyForLevel(addrKey, level)
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
	results := make([]legacydb.BlockRegion, numToLoad)
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

// dbRemoveAddrIndexEntries removes the specified number of entries from from
// the address index for the provided key.  An assertion error will be returned
// if the count exceeds the total number of entries in the index.
func dbRemoveAddrIndexEntries(bucket internalBucket, addrKey [index.AddrKeySize]byte, count int) error {
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
			curLevelKey := keyForLevel(addrKey, level)
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
		curLevelKey := keyForLevel(addrKey, level)
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
		curLevelKey := keyForLevel(addrKey, level)
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
