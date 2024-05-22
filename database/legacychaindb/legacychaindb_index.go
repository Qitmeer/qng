package legacychaindb

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/staging"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/consensus/store/invalid_tx_index"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/meerdag"
)

const (
	// Size of a transaction entry.  It consists of 4 bytes block id + 4
	// bytes offset + 4 bytes length.
	txEntrySize = 4 + 4 + 4

	defaultPreallocateCaches = false
	defaultCacheSize         = 10
)

var (
	// txIndexKey is the key of the transaction index and the db bucket used
	// to house it.
	txIndexKey = []byte("txbyhashidx")

	// txidByTxhashBucketName is the name of the db bucket used to house
	// the tx hash -> tx id.
	txidByTxhashBucketName = []byte("txidbytxhash")

	// errNoTxHashEntry is an error that indicates a requested entry does
	// not exist in the tx hash
	errNoTxHashEntry = errors.New("no entry in the tx hash")

	// hashByOrderIndexBucketName is the name of the db bucket used to house
	// the block index ID -> block hash index.
	hashByIDIndexBucketName = []byte("hashbyididx")

	// errNoBlockOrderEntry is an error that indicates a requested entry does
	// not exist in the block order index.
	errNoBlockOrderEntry = errors.New("no entry in the block order index")

	// errInterruptRequested indicates that an operation was cancelled due
	// to a user-requested interrupt.
	errInterruptRequested = errors.New("interrupt requested")
)

func (cdb *LegacyChainDB) PutTxIdxEntrys(sblock *types.SerializedBlock, block model.Block) error {
	return cdb.doPutTxIndexEntrys(sblock, block.GetID())
}

func (cdb *LegacyChainDB) doPutTxIndexEntrys(sblock *types.SerializedBlock, blockid uint) error {
	addEntries := func(txns []*types.Tx, txLocs []types.TxLoc, blockID uint32) error {
		offset := 0
		serializedValues := make([]byte, len(txns)*txEntrySize)
		return cdb.db.Update(func(dbTx legacydb.Tx) error {
			for i, tx := range txns {
				putTxIndexEntry(serializedValues[offset:], blockID,
					txLocs[i])
				endOffset := offset + txEntrySize

				if !tx.IsDuplicate {
					if err := dbPutTxIndexEntry(dbTx, tx.Hash(),
						serializedValues[offset:endOffset:endOffset]); err != nil {
						return err
					}
				}
				offset += txEntrySize
			}
			return nil
		})
	}

	// Add the regular transactions.
	// The offset and length of the transactions within the
	// serialized parent block.
	txLocs, err := sblock.TxLoc()
	if err != nil {
		return err
	}
	return addEntries(sblock.Transactions(), txLocs, uint32(blockid))
}

func (cdb *LegacyChainDB) GetTxIdxEntry(id *hash.Hash, verbose bool) (*types.Tx, *hash.Hash, error) {
	var blockHash *hash.Hash
	var serializedData []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		txIndex := dbTx.Metadata().Bucket(txIndexKey)
		serializedData = txIndex.Get(id[:])
		if len(serializedData) == 0 {
			return nil
		}
		// Ensure the serialized data has enough bytes to properly deserialize.
		if len(serializedData) < 12 {
			return legacydb.Error{
				ErrorCode:   legacydb.ErrCorruption,
				Description: fmt.Sprintf("corrupt transaction index entry for %s", id),
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(serializedData) == 0 {
		return nil, nil, nil
	}
	blockid := uint(byteOrder.Uint32(serializedData[0:4]))
	blockHash, err = meerdag.DBGetDAGBlockHashByID(cdb, uint64(blockid))
	if err != nil {
		return nil, nil, err
	}
	if !verbose {
		return nil, blockHash, nil
	}
	// Deserialize the final entry.
	region := legacydb.BlockRegion{Hash: &hash.Hash{}}
	copy(region.Hash[:], blockHash[:])
	region.Offset = byteOrder.Uint32(serializedData[4:8])
	region.Len = byteOrder.Uint32(serializedData[8:12])

	var txBytes []byte
	err = cdb.db.View(func(dbTx legacydb.Tx) error {
		txBytes, err = dbTx.FetchBlockRegion(&region)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	var msgTx types.Transaction
	err = msgTx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, nil, err
	}
	return types.NewTx(&msgTx), blockHash, nil
}

func (cdb *LegacyChainDB) DeleteTxIdxEntrys(block *types.SerializedBlock) error {
	for _, tx := range block.Transactions() {
		_, blockHash, _ := cdb.GetTxIdxEntry(tx.Hash(), false)
		if blockHash != nil && !blockHash.IsEqual(block.Hash()) {
			continue
		}
		err := cdb.db.Update(func(dbTx legacydb.Tx) error {
			return dbRemoveTxIndexEntry(dbTx, tx.Hash())
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (cdb *LegacyChainDB) PutTxHashs(block *types.SerializedBlock) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		for _, tx := range block.Transactions() {
			if tx.IsDuplicate {
				continue
			}
			err := dbPutTxIdByHash(dbTx, tx.Tx.TxHashFull(), tx.Hash())
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (cdb *LegacyChainDB) GetTxIdByHash(fullHash *hash.Hash) (*hash.Hash, error) {
	var txid *hash.Hash
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		var err error
		id, err := dbFetchTxIdByHash(dbTx, *fullHash)
		if err != nil {
			return err
		}
		txid = id
		return nil
	})
	return txid, err
}

func (cdb *LegacyChainDB) DeleteTxHashs(block *types.SerializedBlock) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		for _, tx := range block.Transactions() {
			if err := dbRemoveTxIdByHash(dbTx, tx.Tx.TxHashFull()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (cdb *LegacyChainDB) InvalidtxindexStore() model.InvalidTxIndexStore {
	if cdb.invalidtxindexStore == nil {
		var err error
		cdb.invalidtxindexStore, err = invalid_tx_index.New(cdb.DB(), cdb, defaultCacheSize, defaultPreallocateCaches)
		if err != nil {
			log.Error(err.Error())
			return nil
		}
	}
	return cdb.invalidtxindexStore
}

func (cdb *LegacyChainDB) IsInvalidTxIdxEmpty() bool {
	return cdb.InvalidtxindexStore().IsEmpty()
}

func (cdb *LegacyChainDB) GetInvalidTxIdxTip() (uint64, *hash.Hash, error) {
	return cdb.InvalidtxindexStore().Tip(model.NewStagingArea())
}

func (cdb *LegacyChainDB) PutInvalidTxIdxTip(order uint64, bh *hash.Hash) error {
	stagingArea := model.NewStagingArea()
	cdb.InvalidtxindexStore().StageTip(stagingArea, bh, order)
	return staging.CommitAllChanges(cdb.DB(), stagingArea)
}

func (cdb *LegacyChainDB) PutInvalidTxs(sblock *types.SerializedBlock, block model.Block) error {
	stagingArea := model.NewStagingArea()
	cdb.InvalidtxindexStore().Stage(stagingArea, uint64(block.GetID()), sblock)
	return staging.CommitAllChanges(cdb.db, stagingArea)
}

func (cdb *LegacyChainDB) DeleteInvalidTxs(sblock *types.SerializedBlock, block model.Block) error {
	stagingArea := model.NewStagingArea()
	cdb.InvalidtxindexStore().Delete(stagingArea, uint64(block.GetID()), sblock)
	return staging.CommitAllChanges(cdb.DB(), stagingArea)
}

func (cdb *LegacyChainDB) GetInvalidTx(id *hash.Hash) (*types.Transaction, error) {
	stagingArea := model.NewStagingArea()
	return cdb.InvalidtxindexStore().Get(stagingArea, id)
}

func (cdb *LegacyChainDB) GetInvalidTxIdByHash(fullHash *hash.Hash) (*hash.Hash, error) {
	stagingArea := model.NewStagingArea()
	return cdb.InvalidtxindexStore().GetIdByHash(stagingArea, fullHash)
}

func (cdb *LegacyChainDB) CleanInvalidTxIdx() error {
	log.Info("Start clean invalidtx index")
	if cdb.InvalidtxindexStore().IsEmpty() {
		return fmt.Errorf("No data needs to be deleted")
	}
	tipOrder, tipHash, err := cdb.InvalidtxindexStore().Tip(model.NewStagingArea())
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("All invalidtx index at (%s,%d) will be deleted", tipHash, tipOrder))
	return cdb.InvalidtxindexStore().Clean()
}

// putTxIndexEntry serializes the provided values according to the format
// described about for a transaction index entry.  The target byte slice must
// be at least large enough to handle the number of bytes defined by the
// txEntrySize constant or it will panic.
func putTxIndexEntry(target []byte, blockID uint32, txLoc types.TxLoc) {
	byteOrder.PutUint32(target, blockID)
	byteOrder.PutUint32(target[4:], uint32(txLoc.TxStart))
	byteOrder.PutUint32(target[8:], uint32(txLoc.TxLen))
}

// dbRemoveTxIndexEntry uses an existing database transaction to remove the most
// recent transaction index entry for the given hash.
func dbRemoveTxIndexEntry(dbTx legacydb.Tx, txHash *hash.Hash) error {
	txIndex := dbTx.Metadata().Bucket(txIndexKey)
	serializedData := txIndex.Get(txHash[:])
	if len(serializedData) == 0 {
		return nil
	}
	return txIndex.Delete(txHash[:])
}

// dbPutTxIndexEntry uses an existing database transaction to update the
// transaction index given the provided serialized data that is expected to have
// been serialized putTxIndexEntry.
func dbPutTxIndexEntry(dbTx legacydb.Tx, txHash *hash.Hash, serializedData []byte) error {
	txIndex := dbTx.Metadata().Bucket(txIndexKey)
	return txIndex.Put(txHash[:], serializedData)
}

// dbPutTxIdByHash
func dbPutTxIdByHash(dbTx legacydb.Tx, txHash hash.Hash, txId *hash.Hash) error {
	txidByTxhash := dbTx.Metadata().Bucket(txidByTxhashBucketName)
	return txidByTxhash.Put(txHash[:], txId[:])
}

func dbFetchTxIdByHash(dbTx legacydb.Tx, txhash hash.Hash) (*hash.Hash, error) {
	txidByTxhash := dbTx.Metadata().Bucket(txidByTxhashBucketName)
	serializedData := txidByTxhash.Get(txhash[:])
	if serializedData == nil {
		return nil, errNoTxHashEntry
	}
	txId := hash.Hash{}
	txId.SetBytes(serializedData[:])

	return &txId, nil
}

func dbRemoveTxIdByHash(dbTx legacydb.Tx, txhash hash.Hash) error {
	txidByTxhash := dbTx.Metadata().Bucket(txidByTxhashBucketName)
	serializedData := txidByTxhash.Get(txhash[:])
	if len(serializedData) == 0 {
		return nil
	}
	return txidByTxhash.Delete(txhash[:])
}

func dbFetchIndexerTip(dbTx legacydb.Tx, idxKey []byte) (*hash.Hash, uint32, error) {
	indexesBucket := dbTx.Metadata().Bucket(dbnamespace.IndexTipsBucketName)
	serialized := indexesBucket.Get(idxKey)
	if len(serialized) < hash.HashSize+4 {
		return nil, 0, legacydb.Error{
			ErrorCode: legacydb.ErrCorruption,
			Description: fmt.Sprintf("unexpected end of data for "+
				"index %q tip", string(idxKey)),
		}
	}

	var h hash.Hash
	copy(h[:], serialized[:hash.HashSize])
	order := uint32(byteOrder.Uint32(serialized[hash.HashSize:]))
	return &h, order, nil
}

// -----------------------------------------------------------------------------
// The index manager tracks the current tip of each index by using a parent
// bucket that contains an entry for index.
//
// The serialized format for an index tip is:
//
//   [<block hash><block order>],...
//
//   Field           Type             Size
//   block hash      chainhash.Hash   chainhash.HashSize
//   block order    uint32           4 bytes
// -----------------------------------------------------------------------------

// dbPutIndexerTip uses an existing database transaction to update or add the
// current tip for the given index to the provided values.
func dbPutIndexerTip(dbTx legacydb.Tx, idxKey []byte, h *hash.Hash, order uint32) error {
	serialized := make([]byte, hash.HashSize+4)
	copy(serialized, h[:])
	byteOrder.PutUint32(serialized[hash.HashSize:], uint32(order))

	indexesBucket := dbTx.Metadata().Bucket(dbnamespace.IndexTipsBucketName)
	return indexesBucket.Put(idxKey, serialized)
}

func dbFetchBlockHashBySerializedID(dbTx legacydb.Tx, serializedID []byte) (*hash.Hash, error) {
	idIndex := dbTx.Metadata().Bucket(hashByIDIndexBucketName)
	hashBytes := idIndex.Get(serializedID)
	if hashBytes == nil {
		return nil, errNoBlockOrderEntry
	}

	var hash hash.Hash
	copy(hash[:], hashBytes)
	return &hash, nil
}

func dbFetchBlockHashByIID(dbTx legacydb.Tx, iid uint32) (*hash.Hash, error) {
	var serializedID [4]byte
	byteOrder.PutUint32(serializedID[:], iid)
	return dbFetchBlockHashBySerializedID(dbTx, serializedID[:])
}

// dropIndex drops the passed index from the database without using incremental
// deletion.  This should be used to drop indexes containing nested buckets,
// which can not be deleted with dropFlatIndex.
func dropIndex(db legacydb.DB, idxKey []byte, idxName string, interrupt <-chan struct{}) error {
	// Nothing to do if the index doesn't already exist.
	exists, err := existsIndex(db, idxKey, idxName)
	if err != nil {
		return err
	}
	if !exists {
		log.Info(fmt.Sprintf("Not dropping %s because it does not exist", idxName))
		return nil
	}

	log.Info(fmt.Sprintf("Dropping all %s entries.  This might take a while...",
		idxName))

	// Mark that the index is in the process of being dropped so that it
	// can be resumed on the next start if interrupted before the process is
	// complete.
	err = markIndexDeletion(db, idxKey)
	if err != nil {
		return err
	}

	// Since the indexes can be so large, attempting to simply delete
	// the bucket in a single database transaction would result in massive
	// memory usage and likely crash many systems due to ulimits.  In order
	// to avoid this, use a cursor to delete a maximum number of entries out
	// of the bucket at a time.
	err = incrementalFlatDrop(db, idxKey, idxName, interrupt)
	if err != nil {
		return err
	}

	// Remove the index tip, index bucket, and in-progress drop flag.  Removing
	// the index bucket also recursively removes all values saved to the index.
	err = dropIndexMetadata(db, idxKey, idxName)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Dropped %s", idxName))
	return nil
}

// existsIndex returns whether the index keyed by idxKey exists in the database.
func existsIndex(db legacydb.DB, idxKey []byte, idxName string) (bool, error) {
	var exists bool
	err := db.View(func(dbTx legacydb.Tx) error {
		indexesBucket := dbTx.Metadata().Bucket(dbnamespace.IndexTipsBucketName)
		if indexesBucket != nil && indexesBucket.Get(idxKey) != nil {
			exists = true
		}
		return nil
	})
	return exists, err
}

// markIndexDeletion marks the index identified by idxKey for deletion.  Marking
// an index for deletion allows deletion to resume next startup if an
// incremental deletion was interrupted.
func markIndexDeletion(db legacydb.DB, idxKey []byte) error {
	return db.Update(func(dbTx legacydb.Tx) error {
		indexesBucket := dbTx.Metadata().Bucket(dbnamespace.IndexTipsBucketName)
		return indexesBucket.Put(indexDropKey(idxKey), idxKey)
	})
}

// indexDropKey returns the key for an index which indicates it is in the
// process of being dropped.
func indexDropKey(idxKey []byte) []byte {
	dropKey := make([]byte, len(idxKey)+1)
	dropKey[0] = 'd'
	copy(dropKey[1:], idxKey)
	return dropKey
}

// incrementalFlatDrop uses multiple database updates to remove key/value pairs
// saved to a flat index.
func incrementalFlatDrop(db legacydb.DB, idxKey []byte, idxName string, interrupt <-chan struct{}) error {
	// Since the indexes can be so large, attempting to simply delete
	// the bucket in a single database transaction would result in massive
	// memory usage and likely crash many systems due to ulimits.  In order
	// to avoid this, use a cursor to delete a maximum number of entries out
	// of the bucket at a time. Recurse buckets depth-first to delete any
	// sub-buckets.
	const maxDeletions = 2000000
	var totalDeleted uint64

	// Recurse through all buckets in the index, cataloging each for
	// later deletion.
	var subBuckets [][][]byte
	var subBucketClosure func(legacydb.Tx, []byte, [][]byte) error
	subBucketClosure = func(dbTx legacydb.Tx,
		subBucket []byte, tlBucket [][]byte) error {
		// Get full bucket name and append to subBuckets for later
		// deletion.
		var bucketName [][]byte
		if (tlBucket == nil) || (len(tlBucket) == 0) {
			bucketName = append(bucketName, subBucket)
		} else {
			bucketName = append(tlBucket, subBucket)
		}
		subBuckets = append(subBuckets, bucketName)
		// Recurse sub-buckets to append to subBuckets slice.
		bucket := dbTx.Metadata()
		for _, subBucketName := range bucketName {
			bucket = bucket.Bucket(subBucketName)
			if bucket == nil {
				return legacydb.Error{
					ErrorCode:   legacydb.ErrBucketNotFound,
					Description: fmt.Sprintf("db bucket '%s' not found, your data is corrupted, please clean up your block database by using '--cleanup'", subBucketName),
					Err:         nil}
			}

		}
		return bucket.ForEachBucket(func(k []byte) error {
			return subBucketClosure(dbTx, k, bucketName)
		})
	}

	// Call subBucketClosure with top-level bucket.
	err := db.View(func(dbTx legacydb.Tx) error {
		return subBucketClosure(dbTx, idxKey, nil)
	})
	if err != nil {
		return err
	}

	// Iterate through each sub-bucket in reverse, deepest-first, deleting
	// all keys inside them and then dropping the buckets themselves.
	for i := range subBuckets {
		bucketName := subBuckets[len(subBuckets)-1-i]
		// Delete maxDeletions key/value pairs at a time.
		for numDeleted := maxDeletions; numDeleted == maxDeletions; {
			numDeleted = 0
			err := db.Update(func(dbTx legacydb.Tx) error {
				subBucket := dbTx.Metadata()
				for _, subBucketName := range bucketName {
					subBucket = subBucket.Bucket(subBucketName)
				}
				cursor := subBucket.Cursor()
				for ok := cursor.First(); ok; ok = cursor.Next() &&
					numDeleted < maxDeletions {

					if err := cursor.Delete(); err != nil {
						return err
					}
					numDeleted++
				}
				return nil
			})
			if err != nil {
				return err
			}

			if numDeleted > 0 {
				totalDeleted += uint64(numDeleted)
				log.Info(fmt.Sprintf("Deleted %d keys (%d total) from %s",
					numDeleted, totalDeleted, idxName))
			}
		}

		if system.InterruptRequested(interrupt) {
			return errInterruptRequested
		}

		// Drop the bucket itself.
		db.Update(func(dbTx legacydb.Tx) error {
			bucket := dbTx.Metadata()
			for j := 0; j < len(bucketName)-1; j++ {
				bucket = bucket.Bucket(bucketName[j])
			}
			return bucket.DeleteBucket(bucketName[len(bucketName)-1])
		})
	}
	return nil
}

// dropIndexMetadata drops the passed index from the database by removing the
// top level bucket for the index, the index tip, and any in-progress drop flag.
func dropIndexMetadata(db legacydb.DB, idxKey []byte, idxName string) error {
	return db.Update(func(dbTx legacydb.Tx) error {
		meta := dbTx.Metadata()
		indexesBucket := meta.Bucket(dbnamespace.IndexTipsBucketName)
		err := indexesBucket.Delete(idxKey)
		if err != nil {
			return err
		}

		err = meta.DeleteBucket(idxKey)
		if err != nil && !legacydb.IsError(err, legacydb.ErrBucketNotFound) {
			return err
		}

		return indexesBucket.Delete(indexDropKey(idxKey))
	})
}
