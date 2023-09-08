package legacychaindb

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/staging"
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
)

func (cdb *LegacyChainDB) PutTxIndexEntrys(sblock *types.SerializedBlock, block model.Block) error {
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

func (cdb *LegacyChainDB) GetTxIndexEntry(id *hash.Hash, verbose bool) (*types.Tx, *hash.Hash, error) {
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

func (cdb *LegacyChainDB) DeleteTxIndexEntrys(block *types.SerializedBlock) error {
	for _, tx := range block.Transactions() {
		_, blockHash, _ := cdb.GetTxIndexEntry(tx.Hash(), false)
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

func (cdb *LegacyChainDB) IsInvalidTxEmpty() bool {
	return cdb.InvalidtxindexStore().IsEmpty()
}

func (cdb *LegacyChainDB) GetInvalidTxTip() (uint64, *hash.Hash, error) {
	return cdb.InvalidtxindexStore().Tip(model.NewStagingArea())
}

func (cdb *LegacyChainDB) PutInvalidTxTip(order uint64, bh *hash.Hash) error {
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

func (cdb *LegacyChainDB) CleanInvalidTxs() error {
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
