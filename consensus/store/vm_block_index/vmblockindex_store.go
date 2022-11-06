package vm_block_index

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/lrucache"
	"github.com/Qitmeer/qng/common/staging"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database"
)

var bucketName = []byte("vm_block_index")
var tipOrderKeyName = []byte("vmbi_tip_order")
var tipHashKeyName = []byte("vmbi_tip_hash")

type vmblockindexStore struct {
	shardID model.StagingShardID
	cache   *lrucache.LRUIDCache
	db      database.DB
}

func (bis *vmblockindexStore) Stage(stagingArea *model.StagingArea, bid uint64, bhash *hash.Hash) {
	stagingShard := bis.stagingShard(stagingArea)
	if _, ok := stagingShard.toDelete[bid]; ok {
		delete(stagingShard.toDelete, bid)
	}
	stagingShard.toAdd[bid] = bhash
}

func (bis *vmblockindexStore) StageTip(stagingArea *model.StagingArea, bhash *hash.Hash, order uint64) {
	stagingShard := bis.stagingShard(stagingArea)
	stagingShard.tipOrder = order
	stagingShard.tipHash = bhash
}

func (bis *vmblockindexStore) IsStaged(stagingArea *model.StagingArea) bool {
	return bis.stagingShard(stagingArea).isStaged()
}

func (bis *vmblockindexStore) Get(stagingArea *model.StagingArea, bid uint64) (*hash.Hash, error) {
	stagingShard := bis.stagingShard(stagingArea)
	if bh, ok := stagingShard.toAdd[bid]; ok {
		return bh, nil
	}

	if _, ok := stagingShard.toDelete[bid]; ok {
		return nil, nil
	}

	if bh, ok := bis.cache.Get(bid); ok {
		return bh.(*hash.Hash), nil
	}
	var bh *hash.Hash
	err := bis.db.View(func(dbTx database.Tx) error {
		bucket := dbTx.Metadata().Bucket(bucketName)
		hb := bucket.Get(serialization.SerializeUint64(bid))
		if len(hb) <= 0 {
			return nil
		}
		h, err := hash.NewHash(hb)
		bh = h
		return err
	})
	if err != nil {
		return nil, err
	}
	if bh == nil {
		return nil, nil
	}
	bis.cache.Add(bid, bh)
	return bh, nil
}

func (bis *vmblockindexStore) Has(stagingArea *model.StagingArea, bid uint64) (bool, error) {
	stagingShard := bis.stagingShard(stagingArea)
	if _, ok := stagingShard.toAdd[bid]; ok {
		return true, nil
	}
	if _, ok := stagingShard.toDelete[bid]; ok {
		return false, nil
	}
	if bis.cache.Has(bid) {
		return true, nil
	}
	exists := false
	err := bis.db.View(func(dbTx database.Tx) error {
		bucket := dbTx.Metadata().Bucket(bucketName)
		hb := bucket.Get(serialization.SerializeUint64(bid))
		if len(hb) > 0 {
			exists = true
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (bis *vmblockindexStore) Delete(stagingArea *model.StagingArea, bid uint64) {
	stagingShard := bis.stagingShard(stagingArea)
	if _, ok := stagingShard.toAdd[bid]; ok {
		delete(stagingShard.toAdd, bid)
	}
	stagingShard.toDelete[bid] = struct{}{}
}

func (bis *vmblockindexStore) Tip(stagingArea *model.StagingArea) (uint64, *hash.Hash, error) {
	stagingShard := bis.stagingShard(stagingArea)
	if stagingShard.tipHash != nil {
		return stagingShard.tipOrder, stagingShard.tipHash, nil
	}
	var tipHash *hash.Hash
	var tipOrder uint64
	err := bis.db.View(func(dbTx database.Tx) error {
		bucket := dbTx.Metadata().Bucket(bucketName)
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
	stagingShard.tipOrder = tipOrder
	stagingShard.tipHash = tipHash
	return tipOrder, tipHash, nil
}

func (bis *vmblockindexStore) IsEmpty() bool {
	hasTip := false
	bis.db.View(func(dbTx database.Tx) error {
		bucket := dbTx.Metadata().Bucket(bucketName)
		tiphashValue := bucket.Get(tipHashKeyName)
		if len(tiphashValue) > 0 {
			hasTip = true
		}
		return nil
	})
	return hasTip
}

func (bis *vmblockindexStore) stagingShard(stagingArea *model.StagingArea) *vmblockindexStagingShard {
	return stagingArea.GetOrCreateShard(bis.shardID, func() model.StagingShard {
		return &vmblockindexStagingShard{
			store:    bis,
			toAdd:    make(map[uint64]*hash.Hash),
			toDelete: make(map[uint64]struct{}),
		}
	}).(*vmblockindexStagingShard)
}

func New(db database.DB, cacheSize int, preallocate bool) (model.VMBlockIndexStore, error) {
	store := &vmblockindexStore{
		shardID: staging.GenerateShardingID(),
		cache:   lrucache.NewLRUIDCache(cacheSize, preallocate),
		db:      db,
	}
	return store, nil
}
