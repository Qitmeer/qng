package vm_block_index

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/log"
)

type vmblockindexStagingShard struct {
	store    *vmblockindexStore
	toAdd    map[uint64]*hash.Hash
	toDelete map[uint64]struct{}
	tipOrder uint64
	tipHash  *hash.Hash
}

func (biss *vmblockindexStagingShard) Commit(dbTx database.Tx) error {
	if !biss.isStaged() {
		return nil
	}
	bucket := dbTx.Metadata().Bucket(bucketName)
	if bucket == nil {
		var err error
		bucket,err=dbTx.Metadata().CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("Create bucket:%s",bucketName))
	}
	for vmbid, bhash := range biss.toAdd {
		err := bucket.Put(serialization.SerializeUint64(vmbid), bhash.Bytes())
		if err != nil {
			return err
		}
		biss.store.cache.Add(vmbid, bhash)
	}

	for vmbid := range biss.toDelete {
		err := bucket.Delete(serialization.SerializeUint64(vmbid))
		if err != nil {
			return err
		}
		biss.store.cache.Remove(vmbid)
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

func (biss *vmblockindexStagingShard) isStaged() bool {
	return len(biss.toAdd) != 0 || len(biss.toDelete) != 0 || biss.tipHash != nil
}
