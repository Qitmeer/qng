package vm_block_index

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database"
)

type vmblockindexStagingShard struct {
	store    *vmblockindexStore
	toAdd    map[uint64]*hash.Hash
	toDelete map[uint64]struct{}
}

func (biss *vmblockindexStagingShard) Commit(dbTx database.Tx) error {
	bucket := dbTx.Metadata().Bucket(bucketName)

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
	return nil
}

func (biss *vmblockindexStagingShard) isStaged() bool {
	return len(biss.toAdd) != 0 || len(biss.toDelete) != 0
}
