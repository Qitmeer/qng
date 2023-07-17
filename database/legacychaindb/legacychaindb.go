package legacychaindb

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/database/legacydb"
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
