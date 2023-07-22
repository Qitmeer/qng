package invalid_tx_index

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/staging"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
)

var bucketName = []byte("invalid_tx_index")
var itxidByTxhashBucketName = []byte("invalid_txidbytxhash")
var tipOrderKeyName = []byte("itxi_tip_order")
var tipHashKeyName = []byte("itxi_tip_hash")

type invalidtxindexStore struct {
	shardID model.StagingShardID
	ldb     legacydb.DB
	db      model.DataBase
}

func (itxis *invalidtxindexStore) Stage(stagingArea *model.StagingArea, bid uint64, block *types.SerializedBlock) {
	stagingShard := itxis.stagingShard(stagingArea)
	if _, ok := stagingShard.toDelete[bid]; ok {
		delete(stagingShard.toDelete, bid)
	}
	stagingShard.toAdd[bid] = block
}

func (itxis *invalidtxindexStore) StageTip(stagingArea *model.StagingArea, bhash *hash.Hash, order uint64) {
	stagingShard := itxis.stagingShard(stagingArea)
	stagingShard.tipOrder = order
	stagingShard.tipHash = bhash
}

func (itxis *invalidtxindexStore) IsStaged(stagingArea *model.StagingArea) bool {
	return itxis.stagingShard(stagingArea).isStaged()
}

func (itxis *invalidtxindexStore) Get(stagingArea *model.StagingArea, txid *hash.Hash) (*types.Transaction, error) {
	stagingShard := itxis.stagingShard(stagingArea)
	for _, add := range stagingShard.toAdd {
		for _, tx := range add.Transactions() {
			if tx.Hash().IsEqual(txid) {
				return tx.Tx, nil
			}
		}
	}

	for _, add := range stagingShard.toDelete {
		for _, tx := range add.Transactions() {
			if tx.Hash().IsEqual(txid) {
				return nil, nil
			}
		}
	}
	blockRegion, err := dbFetchTxIndexEntry(itxis.ldb, itxis.db, txid)
	if err != nil {
		return nil, err
	}
	if blockRegion == nil {
		return nil, nil
	}
	var tx *types.Transaction
	err = itxis.ldb.View(func(dbTx legacydb.Tx) error {
		txBytes, err := dbTx.FetchBlockRegion(blockRegion)
		if err != nil {
			return err
		}
		dtx := types.Transaction{}
		err = dtx.Deserialize(bytes.NewReader(txBytes))
		if err != nil {
			return err
		}
		tx = &dtx
		return err
	})
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (itxis *invalidtxindexStore) GetIdByHash(stagingArea *model.StagingArea, h *hash.Hash) (*hash.Hash, error) {
	stagingShard := itxis.stagingShard(stagingArea)
	for _, add := range stagingShard.toAdd {
		for _, tx := range add.Transactions() {
			th := tx.Tx.TxHashFull()
			if th.IsEqual(h) {
				return tx.Hash(), nil
			}
		}
	}

	for _, add := range stagingShard.toDelete {
		for _, tx := range add.Transactions() {
			th := tx.Tx.TxHashFull()
			if th.IsEqual(h) {
				return nil, nil
			}
		}
	}

	var txid *hash.Hash
	err := itxis.ldb.View(func(dbTx legacydb.Tx) error {
		id, er := dbFetchTxIdByHash(dbTx, h)
		if er != nil {
			return er
		}
		txid = id
		return nil
	})
	if err != nil {
		return nil, err
	}
	return txid, nil
}

func (itxis *invalidtxindexStore) Delete(stagingArea *model.StagingArea, bid uint64, block *types.SerializedBlock) {
	stagingShard := itxis.stagingShard(stagingArea)
	if _, ok := stagingShard.toAdd[bid]; ok {
		delete(stagingShard.toAdd, bid)
	}
	stagingShard.toDelete[bid] = block
}

func (itxis *invalidtxindexStore) Tip(stagingArea *model.StagingArea) (uint64, *hash.Hash, error) {
	stagingShard := itxis.stagingShard(stagingArea)
	if stagingShard.tipHash != nil {
		return stagingShard.tipOrder, stagingShard.tipHash, nil
	}
	var tipHash *hash.Hash
	var tipOrder uint64
	err := itxis.ldb.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(bucketName)
		if bucket == nil {
			return fmt.Errorf("No vm block index:%s", bucketName)
		}
		tiphashValue := bucket.Get(tipHashKeyName)
		if len(tiphashValue) <= 0 {
			return fmt.Errorf("No vm block index tip hash")
		}
		th, err := hash.NewHash(tiphashValue)
		if err != nil {
			return err
		}
		tipHash = th
		//
		tiporderValue := bucket.Get(tipOrderKeyName)
		if len(tiporderValue) <= 0 {
			return fmt.Errorf("No vm block index tip order")
		}
		to, err := serialization.DeserializeUint64(tiporderValue)
		if err != nil {
			return err
		}
		tipOrder = to
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return tipOrder, tipHash, nil
}

func (itxis *invalidtxindexStore) IsEmpty() bool {
	has := false
	itxis.ldb.View(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(bucketName)
		if bucket == nil {
			return nil
		}
		cursor := bucket.Cursor()
		has = cursor.First()
		return nil
	})
	return !has
}

func (itxis *invalidtxindexStore) Clean() error {
	return itxis.ldb.Update(func(dbTx legacydb.Tx) error {
		bucket := dbTx.Metadata().Bucket(bucketName)
		if bucket != nil {
			return dbTx.Metadata().DeleteBucket(bucketName)
		}
		return nil
	})
}

func (itxis *invalidtxindexStore) stagingShard(stagingArea *model.StagingArea) *invalidtxindexStagingShard {
	return stagingArea.GetOrCreateShard(itxis.shardID, func() model.StagingShard {
		return &invalidtxindexStagingShard{
			store:    itxis,
			toAdd:    make(map[uint64]*types.SerializedBlock),
			toDelete: make(map[uint64]*types.SerializedBlock),
		}
	}).(*invalidtxindexStagingShard)
}

func New(ldb legacydb.DB, db model.DataBase, cacheSize int, preallocate bool) (model.InvalidTxIndexStore, error) {
	store := &invalidtxindexStore{
		shardID: staging.GenerateShardingID(),
		db:      db,
		ldb:     ldb,
	}
	return store, nil
}

func dbGetTx(dbTx legacydb.Tx, br *legacydb.BlockRegion) (*types.Transaction, error) {
	txBytes, err := dbTx.FetchBlockRegion(br)
	if err != nil {
		return nil, err
	}
	// Deserialize the transaction
	var msgTx types.Transaction
	err = msgTx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return nil, err
	}
	return &msgTx, nil
}
