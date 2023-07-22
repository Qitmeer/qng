package invalid_tx_index

import (
	"encoding/binary"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
)

const (
	// Size of a transaction entry.  It consists of 4 bytes block id + 4
	// bytes offset + 4 bytes length.
	txEntrySize = 4 + 4 + 4
)

var (
	// byteOrder is the preferred byte order used for serializing numeric
	// fields for storage in the database.
	byteOrder = binary.LittleEndian
)

type invalidtxindexStagingShard struct {
	store    *invalidtxindexStore
	toAdd    map[uint64]*types.SerializedBlock
	toDelete map[uint64]*types.SerializedBlock
	tipOrder uint64
	tipHash  *hash.Hash
}

func (biss *invalidtxindexStagingShard) Commit(dbTx legacydb.Tx) error {
	if !biss.isStaged() {
		return nil
	}
	var bucket legacydb.Bucket
	var itxidByTxhashBucket legacydb.Bucket
	bucket = dbTx.Metadata().Bucket(bucketName)
	if bucket == nil {
		var err error
		bucket, err = dbTx.Metadata().CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Create bucket:%s", bucketName))

		itxidByTxhashBucket, err = bucket.CreateBucketIfNotExists(itxidByTxhashBucketName)
		if err != nil {
			return err
		}
	}
	for blockid, block := range biss.toAdd {
		err := dbAddTxIndexEntries(bucket, itxidByTxhashBucket, block, blockid)
		if err != nil {
			return err
		}
	}
	for blockid, block := range biss.toDelete {
		err := dbRemoveTxIndexEntries(bucket, itxidByTxhashBucket, blockid, block)
		if err != nil {
			return err
		}
	}
	if biss.tipHash != nil {
		err := bucket.Put(tipOrderKeyName, serialization.SerializeUint64(biss.tipOrder))
		if err != nil {
			return err
		}
		err = bucket.Put(tipHashKeyName, biss.tipHash.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (biss *invalidtxindexStagingShard) isStaged() bool {
	return len(biss.toAdd) != 0 || len(biss.toDelete) != 0 || biss.tipHash != nil
}

func dbAddTxIndexEntries(itxIndex legacydb.Bucket, itxidByTxhash legacydb.Bucket, block *types.SerializedBlock, blockID uint64) error {
	addEntries := func(txns []*types.Tx, txLocs []types.TxLoc, blockID uint64) error {
		offset := 0
		serializedValues := make([]byte, len(txns)*txEntrySize)
		for i, tx := range txns {
			putTxIndexEntry(serializedValues[offset:], blockID, txLocs[i])
			endOffset := offset + txEntrySize

			if !tx.IsDuplicate {
				if err := dbPutTxIndexEntry(itxIndex, tx.Hash(),
					serializedValues[offset:endOffset:endOffset]); err != nil {
					return err
				}
				if err := dbPutTxIdByHash(itxidByTxhash, tx.Tx.TxHashFull(), tx.Hash()); err != nil {
					return err
				}
			}
			offset += txEntrySize
		}
		return nil
	}
	txLocs, err := block.TxLoc()
	if err != nil {
		return err
	}

	err = addEntries(block.Transactions(), txLocs, blockID)
	if err != nil {
		return err
	}

	return nil
}

func dbPutTxIndexEntry(itxIndex legacydb.Bucket, txHash *hash.Hash, serializedData []byte) error {
	return itxIndex.Put(txHash[:], serializedData)
}

func dbPutTxIdByHash(itxidByTxhash legacydb.Bucket, txHash hash.Hash, txId *hash.Hash) error {
	return itxidByTxhash.Put(txHash[:], txId[:])
}

// putTxIndexEntry serializes the provided values according to the format
// described about for a transaction index entry.  The target byte slice must
// be at least large enough to handle the number of bytes defined by the
// txEntrySize constant or it will panic.
func putTxIndexEntry(target []byte, blockID uint64, txLoc types.TxLoc) {
	byteOrder.PutUint64(target, blockID)
	byteOrder.PutUint32(target[8:], uint32(txLoc.TxStart))
	byteOrder.PutUint32(target[12:], uint32(txLoc.TxLen))
}

func dbFetchTxIndexEntry(ldb legacydb.DB, db model.DataBase, txid *hash.Hash) (*legacydb.BlockRegion, error) {
	var serializedData []byte
	ldb.View(func(dbTx legacydb.Tx) error {
		itxIndex := dbTx.Metadata().Bucket(bucketName)
		if itxIndex == nil {
			return nil
		}
		serializedData = itxIndex.Get(txid[:])
		return nil
	})
	if len(serializedData) <= 0 {
		return nil, nil
	}

	// Ensure the serialized data has enough bytes to properly deserialize.
	if len(serializedData) < 16 {
		return nil, legacydb.Error{
			ErrorCode: legacydb.ErrCorruption,
			Description: fmt.Sprintf("corrupt transaction index "+
				"entry for %s", txid),
		}
	}

	// Load the block hash associated with the block ID.
	blockID, err := serialization.DeserializeUint64(serializedData[0:4])
	if err != nil {
		return nil, err
	}

	h, err := meerdag.DBGetDAGBlockHashByID(db, blockID)
	if err != nil {
		return nil, legacydb.Error{
			ErrorCode: legacydb.ErrCorruption,
			Description: fmt.Sprintf("corrupt transaction index "+
				"entry for %s: %v", txid, err),
		}
	}

	// Deserialize the final entry.
	region := legacydb.BlockRegion{Hash: &hash.Hash{}}
	copy(region.Hash[:], h[:])
	region.Offset = byteOrder.Uint32(serializedData[4:8])
	region.Len = byteOrder.Uint32(serializedData[8:12])
	return &region, nil
}

func dbFetchBlockIDByTxID(itxIndex legacydb.Bucket, txid *hash.Hash) (uint64, error) {
	serializedData := itxIndex.Get(txid[:])
	if len(serializedData) == 0 {
		return 0, nil
	}

	// Ensure the serialized data has enough bytes to properly deserialize.
	if len(serializedData) < 16 {
		return 0, legacydb.Error{
			ErrorCode: legacydb.ErrCorruption,
			Description: fmt.Sprintf("corrupt transaction index "+
				"entry for %s", txid),
		}
	}
	return serialization.DeserializeUint64(serializedData[0:4])
}

func dbRemoveTxIndexEntries(itxIndex legacydb.Bucket, itxidByTxhash legacydb.Bucket, blockID uint64, block *types.SerializedBlock) error {
	removeEntries := func(txns []*types.Tx) error {
		for _, tx := range txns {
			bid, err := dbFetchBlockIDByTxID(itxIndex, tx.Hash())
			if err != nil {
				return err
			}
			if bid != blockID {
				continue
			}
			if err := dbRemoveTxIndexEntry(itxIndex, tx.Hash()); err != nil {
				return err
			}
			if err := dbRemoveTxIdByHash(itxidByTxhash, tx.Tx.TxHashFull()); err != nil {
				return err
			}
		}
		return nil
	}
	if err := removeEntries(block.Transactions()); err != nil {
		return err
	}
	return nil
}

func dbRemoveTxIndexEntry(itxIndex legacydb.Bucket, txHash *hash.Hash) error {
	serializedData := itxIndex.Get(txHash[:])
	if len(serializedData) == 0 {
		return nil
	}
	return itxIndex.Delete(txHash[:])
}

func dbRemoveTxIdByHash(itxidByTxhash legacydb.Bucket, txhash hash.Hash) error {
	serializedData := itxidByTxhash.Get(txhash[:])
	if len(serializedData) == 0 {
		return nil
	}
	return itxidByTxhash.Delete(txhash[:])
}

func dbFetchTxIdByHash(dbTx legacydb.Tx, h *hash.Hash) (*hash.Hash, error) {
	bucket := dbTx.Metadata().Bucket(bucketName)
	if bucket == nil {
		return nil, nil
	}
	itxidByTxhashBucket := bucket.Bucket(itxidByTxhashBucketName)
	if itxidByTxhashBucket == nil {
		return nil, nil
	}
	serializedData := itxidByTxhashBucket.Get(h[:])
	if serializedData == nil {
		return nil, nil
	}
	txId := hash.Hash{}
	txId.SetBytes(serializedData[:])
	return &txId, nil
}
