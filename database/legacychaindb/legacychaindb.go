package legacychaindb

import (
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/services/index"
)

// TODO: It will soon be discarded in the near future
type LegacyChainDB struct {
	db legacydb.DB

	cfg *config.Config
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

	// Cleanup the block database
	if cfg.Cleanup {
		db.Close()
		CleanupBlockDB(cfg)
		return nil, nil
	}
	cdb := &LegacyChainDB{
		cfg: cfg,
		db:  db,
	}
	return cdb, nil
}
