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
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/services/index"
	"math"
)

var (

	// addrIndexKey is the key of the address index and the db bucket used
	// to house it.
	addrIndexKey = []byte("txbyaddridx")
)

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
	addrsToTxns := make(index.WriteAddrIdxData)
	index.AddrIndexBlock(addrsToTxns, sblock, stxos)
	b := &bucket{db: cdb.db}
	for addrKey, txIdxs := range addrsToTxns {
		for _, txIdx := range txIdxs {
			// Switch to using the newest block ID for the stake transactions,
			// since these are not from the parent. Offset the index to be
			// correct for the location in this given block.
			err := index.DBPutAddrIndexEntry(b, addrKey,
				uint32(block.GetID()), txLocs[txIdx], nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cdb *LegacyChainDB) GetTxForAddress(addr types.Address, numToSkip, numRequested uint32, reverse bool) ([]*common.RetrievedTx, uint32, error) {
	addrKey, err := index.AddrToKey(addr)
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
	regions, dbSkipped, err := index.DBFetchAddrIndexEntries(fetchBlockHash, fetchAddrLevelData,
		addrKey, numToSkip, numRequested, reverse, nil)
	if err != nil {
		return nil, 0, err
	}
	var serializedTxns [][]byte
	err = cdb.db.Update(func(dbTx legacydb.Tx) error {
		// Load the raw transaction bytes from the database.
		rs := []legacydb.BlockRegion{}
		for _, r := range regions {
			rs = append(rs, legacydb.BlockRegion{Hash: r.Hash, Offset: r.Offset, Len: r.Len})
		}
		serializedTxns, err = dbTx.FetchBlockRegions(rs)
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
	addrsToTxns := make(index.WriteAddrIdxData)
	index.AddrIndexBlock(addrsToTxns, sblock, stxos)

	// Remove all of the index entries for each address.
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(addrIndexKey)
		for addrKey, txIdxs := range addrsToTxns {
			err := index.DBRemoveAddrIndexEntries(bucket, addrKey, len(txIdxs), nil)
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

type bucket struct {
	db legacydb.DB
}

func (b *bucket) Get(key []byte) []byte {
	value := []byte{}
	b.db.View(func(dbTx legacydb.Tx) error {
		addrIdxBucket := dbTx.Metadata().Bucket(addrIndexKey)
		value = addrIdxBucket.Get(key)
		return nil
	})
	return value
}

func (b *bucket) Put(key []byte, value []byte) error {
	return b.db.Update(func(dbTx legacydb.Tx) error {
		addrIdxBucket := dbTx.Metadata().Bucket(addrIndexKey)
		return addrIdxBucket.Put(key, value)
	})
}

func (b *bucket) Delete(key []byte) error {
	return b.db.Update(func(dbTx legacydb.Tx) error {
		addrIdxBucket := dbTx.Metadata().Bucket(addrIndexKey)
		return addrIdxBucket.Delete(key)
	})
}
