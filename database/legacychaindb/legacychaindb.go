package legacychaindb

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/services/index"
	"math"
)

// TODO: It will soon be discarded in the near future
type LegacyChainDB struct {
	db legacydb.DB

	cfg       *config.Config
	interrupt <-chan struct{}
}

func (cdb *LegacyChainDB) Name() string {
	return "Legacy Chain DB"
}

func (cdb *LegacyChainDB) Init() error {
	var err error
	err = cdb.db.Update(func(dbTx legacydb.Tx) error {
		meta := dbTx.Metadata()
		// Create the bucket that houses the spend journal data.
		_, err = meta.CreateBucket(dbnamespace.SpendJournalBucketName)
		if err != nil {
			return err
		}
		// Create the bucket that houses the utxo set.  Note that the
		// genesis block coinbase transaction is intentionally not
		// inserted here since it is not spendable by consensus rules.
		_, err = meta.CreateBucket(dbnamespace.UtxoSetBucketName)
		if err != nil {
			return err
		}
		// Create the bucket which house the token state
		_, err = meta.CreateBucket(dbnamespace.TokenBucketName)
		if err != nil {
			return err
		}

		// DAG
		// Create the bucket that houses the block index data.
		_, err := meta.CreateBucket(meerdag.BlockIndexBucketName)
		if err != nil {
			return err
		}

		// Create the bucket that houses the chain block order to hash
		// index.
		_, err = meta.CreateBucket(meerdag.OrderIdBucketName)
		if err != nil {
			return err
		}

		_, err = meta.CreateBucket(meerdag.DagMainChainBucketName)
		if err != nil {
			return err
		}

		_, err = meta.CreateBucket(meerdag.BlockIdBucketName)
		if err != nil {
			return err
		}
		_, err = meta.CreateBucketIfNotExists(meerdag.DAGTipsBucketName)
		if err != nil {
			return err
		}
		_, err = meta.CreateBucketIfNotExists(meerdag.DiffAnticoneBucketName)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

func (cdb *LegacyChainDB) Close() {
	log.Info("Close", "name", cdb.Name())
	cdb.db.Close()

}

func (cdb *LegacyChainDB) DB() legacydb.DB {
	return cdb.db
}

func (cdb *LegacyChainDB) Rebuild(mgr model.IndexManager) error {
	err := index.DropInvalidTxIndex(cdb.db, cdb.interrupt)
	if err != nil {
		log.Info(err.Error())
	}
	err = index.DropTxIndex(cdb.db, cdb.interrupt)
	if err != nil {
		log.Info(err.Error())
	}
	//
	err = cdb.db.Update(func(tx legacydb.Tx) error {
		meta := tx.Metadata()
		err = meta.DeleteBucket(dbnamespace.SpendJournalBucketName)
		if err != nil {
			return err
		}
		err = meta.DeleteBucket(dbnamespace.UtxoSetBucketName)
		if err != nil {
			return err
		}
		err = meta.DeleteBucket(dbnamespace.TokenBucketName)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	//
	err = cdb.db.Update(func(tx legacydb.Tx) error {
		meta := tx.Metadata()
		_, err = meta.CreateBucket(dbnamespace.SpendJournalBucketName)
		if err != nil {
			return err
		}
		_, err = meta.CreateBucket(dbnamespace.UtxoSetBucketName)
		if err != nil {
			return err
		}
		_, err = meta.CreateBucket(dbnamespace.TokenBucketName)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	txIndex := mgr.(*index.Manager).TxIndex()
	txIndex.SetCurOrder(-1)
	err = cdb.db.Update(func(tx legacydb.Tx) error {
		err = txIndex.Create(tx)
		if err != nil {
			return err
		}
		err = index.DBPutIndexerTip(tx, txIndex.Key(), &hash.ZeroHash, math.MaxUint32)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (cdb *LegacyChainDB) GetSpendJournal(bh *hash.Hash) ([]byte, error) {
	var data []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.SpendJournalBucketName)
		data = bucket.Get(bh[:])
		return nil
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (cdb *LegacyChainDB) PutSpendJournal(bh *hash.Hash, data []byte) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.SpendJournalBucketName)
		return bucket.Put(bh[:], data)
	})
}

func (cdb *LegacyChainDB) DeleteSpendJournal(bh *hash.Hash) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.SpendJournalBucketName)
		return bucket.Delete(bh[:])
	})
}

func (cdb *LegacyChainDB) GetUtxo(key []byte) ([]byte, error) {
	var data []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.UtxoSetBucketName)
		data = bucket.Get(key)
		return nil
	})
	return data, err
}

func (cdb *LegacyChainDB) PutUtxo(key []byte, data []byte) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.UtxoSetBucketName)
		return bucket.Put(key, data)
	})
}

func (cdb *LegacyChainDB) DeleteUtxo(key []byte) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.UtxoSetBucketName)
		return bucket.Delete(key)
	})
}

func (cdb *LegacyChainDB) GetTokenState(blockID uint) ([]byte, error) {
	var data []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.TokenBucketName)
		var serializedID [4]byte
		dbnamespace.ByteOrder.PutUint32(serializedID[:], uint32(blockID))

		data = bucket.Get(serializedID[:])
		if data == nil {
			return fmt.Errorf("tokenstate db can't find record from block id : %v", blockID)
		}
		return nil
	})
	return data, err
}

func (cdb *LegacyChainDB) PutTokenState(blockID uint, data []byte) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.TokenBucketName)
		// Store the current token balance record into the token state database.
		var serializedID [4]byte
		dbnamespace.ByteOrder.PutUint32(serializedID[:], uint32(blockID))
		return bucket.Put(serializedID[:], data)
	})
}

func (cdb *LegacyChainDB) DeleteTokenState(blockID uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(dbnamespace.TokenBucketName)
		var serializedID [4]byte
		dbnamespace.ByteOrder.PutUint32(serializedID[:], uint32(blockID))

		key := serializedID[:]
		return bucket.Delete(key)
	})
}

func (cdb *LegacyChainDB) GetBestChainState() ([]byte, error) {
	var data []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		data = dbTx.Metadata().Get(dbnamespace.ChainStateKeyName)
		return nil
	})
	return data, err
}

func (cdb *LegacyChainDB) PutBestChainState(data []byte) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		return dbTx.Metadata().Put(dbnamespace.ChainStateKeyName, data)
	})
}

func (cdb *LegacyChainDB) GetBlock(hash *hash.Hash) (*types.SerializedBlock, error) {
	var blockBytes []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		var err error
		// Load the raw block bytes from the database.
		blockBytes, err = dbTx.FetchBlock(hash)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Create the encapsulated block and set the height appropriately.
	block, err := types.NewBlockFromBytes(blockBytes)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (cdb *LegacyChainDB) GetBlockBytes(hash *hash.Hash) ([]byte, error) {
	var blockBytes []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		var err error
		// Load the raw block bytes from the database.
		blockBytes, err = dbTx.FetchBlock(hash)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return blockBytes, nil
}

func (cdb *LegacyChainDB) GetHeader(hash *hash.Hash) (*types.BlockHeader, error) {
	var headerBytes []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		var err error
		headerBytes, err = dbTx.FetchBlockHeader(hash)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var header types.BlockHeader
	err = header.Deserialize(bytes.NewReader(headerBytes))
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func (cdb *LegacyChainDB) PutBlock(block *types.SerializedBlock) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		return dbTx.StoreBlock(block)
	})
}

func (cdb *LegacyChainDB) HasBlock(hash *hash.Hash) bool {
	result := false
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		has, er := dbTx.HasBlock(hash)
		if er != nil {
			return er
		}
		result = has
		return nil
	})
	if err != nil {
		log.Error(err.Error())
		return false
	}
	return result
}

func (cdb *LegacyChainDB) GetDagInfo() ([]byte, error) {
	var data []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		serializedData := dbTx.Metadata().Get(meerdag.DagInfoBucketName)
		if serializedData == nil {
			return fmt.Errorf("dag load error")
		}
		data = serializedData
		return nil
	})
	return data, err
}

func (cdb *LegacyChainDB) PutDagInfo(data []byte) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		return dbTx.Metadata().Put(meerdag.DagInfoBucketName, data)
	})
}

func (cdb *LegacyChainDB) GetDAGBlock(blockID uint) ([]byte, error) {
	var data []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.BlockIndexBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(blockID))

		data = bucket.Get(serializedID[:])
		return nil
	})
	return data, err
}

func (cdb *LegacyChainDB) PutDAGBlock(blockID uint, data []byte) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.BlockIndexBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(blockID))
		return bucket.Put(serializedID[:], data)
	})
}

func (cdb *LegacyChainDB) DeleteDAGBlock(blockID uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.BlockIndexBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(blockID))
		return bucket.Delete(serializedID[:])
	})
}

func (cdb *LegacyChainDB) GetDAGBlockIdByHash(bh *hash.Hash) (uint, error) {
	var id uint
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.BlockIdBucketName)
		data := bucket.Get(bh[:])
		if data == nil {
			id = uint(meerdag.MaxId)
			return fmt.Errorf("get dag block error")
		}
		id = uint(meerdag.ByteOrder.Uint32(data))
		return nil
	})
	return id, err
}

func (cdb *LegacyChainDB) PutDAGBlockIdByHash(bh *hash.Hash, id uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.BlockIdBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(id))
		return bucket.Put(bh[:], serializedID[:])
	})
}

func (cdb *LegacyChainDB) DeleteDAGBlockIdByHash(bh *hash.Hash) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.BlockIdBucketName)
		return bucket.Delete(bh[:])
	})
}

func (cdb *LegacyChainDB) PutMainChainBlock(blockID uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.DagMainChainBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(blockID))
		return bucket.Put(serializedID[:], []byte{0})
	})
}

func (cdb *LegacyChainDB) HasMainChainBlock(blockID uint) bool {
	has := false
	cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.DagMainChainBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(blockID))

		data := bucket.Get(serializedID[:])
		if len(data) > 0 {
			has = true
		}
		return nil
	})
	return has
}

func (cdb *LegacyChainDB) DeleteMainChainBlock(blockID uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.DagMainChainBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(blockID))
		return bucket.Delete(serializedID[:])
	})
}

func (cdb *LegacyChainDB) PutBlockIdByOrder(order uint, id uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		// Serialize the order for use in the index entries.
		var serializedOrder [4]byte
		meerdag.ByteOrder.PutUint32(serializedOrder[:], uint32(order))

		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(id))

		// Add the block order to id mapping to the index.
		bucket := dbTx.Metadata().Bucket(meerdag.OrderIdBucketName)
		return bucket.Put(serializedOrder[:], serializedID[:])
	})
}

func (cdb *LegacyChainDB) GetBlockIdByOrder(order uint) (uint, error) {
	if order > math.MaxUint32 {
		str := fmt.Sprintf("order %d is overflow", order)
		return uint(meerdag.MaxId), meerdag.NewDAGErrorByStr(str)
	}
	var serializedOrder [4]byte
	meerdag.ByteOrder.PutUint32(serializedOrder[:], uint32(order))

	var id uint
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.OrderIdBucketName)
		idBytes := bucket.Get(serializedOrder[:])
		if idBytes == nil {
			str := fmt.Sprintf("no block at order %d exists", order)
			id = uint(meerdag.MaxId)
			return meerdag.NewDAGErrorByStr(str)
		}
		id = uint(meerdag.ByteOrder.Uint32(idBytes))
		return nil
	})
	return id, err
}

func (cdb *LegacyChainDB) PutDAGTip(id uint, isMain bool) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(id))

		bucket := dbTx.Metadata().Bucket(meerdag.DAGTipsBucketName)
		main := byte(0)
		if isMain {
			main = byte(1)
		}
		return bucket.Put(serializedID[:], []byte{main})
	})
}

func (cdb *LegacyChainDB) GetDAGTips() ([]uint, error) {
	result := []uint{}
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.DAGTipsBucketName)
		cursor := bucket.Cursor()
		mainTip := meerdag.MaxId
		tips := []uint{}
		for cok := cursor.First(); cok; cok = cursor.Next() {
			id := uint(meerdag.ByteOrder.Uint32(cursor.Key()))
			main := cursor.Value()
			if len(main) > 0 {
				if main[0] > 0 {
					if mainTip != meerdag.MaxId {
						return fmt.Errorf("Too many main tip:cur(%d) => next(%d)", mainTip, id)
					}
					mainTip = id
					continue
				}
			}
			tips = append(tips, id)
		}
		if mainTip == meerdag.MaxId {
			return fmt.Errorf("Can't find main tip")
		}
		result = append(result, mainTip)
		if len(tips) > 0 {
			result = append(result, tips...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, err
}

func (cdb *LegacyChainDB) DeleteDAGTip(id uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.DAGTipsBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(id))
		return bucket.Delete(serializedID[:])
	})
}

func (cdb *LegacyChainDB) PutDiffAnticone(id uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(id))
		bucket := dbTx.Metadata().Bucket(meerdag.DiffAnticoneBucketName)
		return bucket.Put(serializedID[:], []byte{byte(0)})
	})
}

func (cdb *LegacyChainDB) GetDiffAnticones() ([]uint, error) {
	diffs := []uint{}
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.DiffAnticoneBucketName)
		cursor := bucket.Cursor()
		for cok := cursor.First(); cok; cok = cursor.Next() {
			id := uint(meerdag.ByteOrder.Uint32(cursor.Key()))
			diffs = append(diffs, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return diffs, nil
}

func (cdb *LegacyChainDB) DeleteDiffAnticone(id uint) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(meerdag.DiffAnticoneBucketName)
		var serializedID [4]byte
		meerdag.ByteOrder.PutUint32(serializedID[:], uint32(id))
		return bucket.Delete(serializedID[:])
	})
}

func (cdb *LegacyChainDB) Get(key []byte) ([]byte, error) {
	var value []byte
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		value = dbTx.Metadata().Get(key)
		return nil
	})
	return value, err
}

func (cdb *LegacyChainDB) Put(key []byte, value []byte) error {
	return cdb.db.Update(func(dbTx legacydb.Tx) error {
		return dbTx.Metadata().Put(key, value)
	})
}

func New(cfg *config.Config, interrupt <-chan struct{}) (*LegacyChainDB, error) {
	// Load the block database.
	db, err := LoadBlockDB(cfg)
	if err != nil {
		log.Error("load block database", "error", err)
		return nil, err
	}
	// Return now if an interrupt signal was triggered.
	if system.InterruptRequested(interrupt) {
		return nil, nil
	}
	// Drop indexes and exit if requested.
	if cfg.DropAddrIndex {
		if err := index.DropAddrIndex(db, interrupt); err != nil {
			log.Error(err.Error())
			return nil, err
		}
		return nil, nil
	}
	if cfg.DropTxIndex {
		if err := index.DropTxIndex(db, interrupt); err != nil {
			log.Error(err.Error())
			return nil, err
		}
		return nil, nil
	}
	cdb := &LegacyChainDB{
		cfg:       cfg,
		db:        db,
		interrupt: interrupt,
	}
	return cdb, nil
}

func NewNaked(db legacydb.DB) *LegacyChainDB {
	return &LegacyChainDB{
		db: db,
	}
}
