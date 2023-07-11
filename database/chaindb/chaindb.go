package chaindb

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/database/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"path/filepath"
	"sync"
	"sync/atomic"
)

var (
	DBDirectoryName = "chaindata"
)

type ChainDB struct {
	db ethdb.Database

	cfg  *config.Config
	lock sync.RWMutex
	// All open databases
	databases   map[*closeTrackingDB]struct{}
	closedState atomic.Bool
}

func (cdb *ChainDB) Name() string {
	return "Chain DB"
}

func (cdb *ChainDB) Close() {
	log.Info("Close", "name", cdb.Name())
	if cdb.closedState.Load() {
		log.Error("Already closed", "name", cdb.Name())
		return
	}
	cdb.closedState.Store(true)
	cdb.closeDatabases()
}

// wrapDatabase ensures the database will be auto-closed when Node is closed.
func (cdb *ChainDB) wrapDatabase(db ethdb.Database) ethdb.Database {
	wrapper := &closeTrackingDB{db, cdb}
	cdb.databases[wrapper] = struct{}{}
	return wrapper
}

// closeDatabases closes all open databases.
func (cdb *ChainDB) closeDatabases() (errors []error) {
	for db := range cdb.databases {
		delete(cdb.databases, db)
		if err := db.Database.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (cdb *ChainDB) CloseDatabases() (errors []error) {
	return cdb.closeDatabases()
}

func (cdb *ChainDB) OpenDatabaseWithFreezer(name string, cache, handles int, ancient string, namespace string, readonly bool) (ethdb.Database, error) {
	cdb.lock.Lock()
	defer cdb.lock.Unlock()
	if cdb.closedState.Load() {
		return nil, ErrDBClosed
	}

	var db ethdb.Database
	var err error
	if cdb.cfg.DataDir == "" {
		db = rawdb.NewMemoryDatabase()
	} else {
		db, err = rawdb.Open(rawdb.OpenOptions{
			Type:              node.DefaultConfig.DBEngine,
			Directory:         cdb.cfg.ResolveDataPath(name),
			AncientsDirectory: cdb.ResolveAncient(name, ancient),
			Namespace:         namespace,
			Cache:             cache,
			Handles:           handles,
			ReadOnly:          readonly,
		})
	}

	if err == nil {
		db = cdb.wrapDatabase(db)
	}
	return db, err
}

func (cdb *ChainDB) OpenDatabase(name string, cache, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	cdb.lock.Lock()
	defer cdb.lock.Unlock()
	if cdb.closedState.Load() {
		return nil, ErrDBClosed
	}

	var db ethdb.Database
	var err error
	if cdb.cfg.DataDir == "" {
		db = rawdb.NewMemoryDatabase()
	} else {
		db, err = rawdb.Open(rawdb.OpenOptions{
			Type:      node.DefaultConfig.DBEngine,
			Directory: cdb.cfg.ResolveDataPath(name),
			Namespace: namespace,
			Cache:     cache,
			Handles:   handles,
			ReadOnly:  readonly,
		})
	}

	if err == nil {
		db = cdb.wrapDatabase(db)
	}
	return db, err
}

func (cdb *ChainDB) ResolveAncient(name string, ancient string) string {
	switch {
	case ancient == "":
		ancient = filepath.Join(cdb.cfg.ResolveDataPath(name), "ancient")
	case !filepath.IsAbs(ancient):
		ancient = cdb.cfg.ResolveDataPath(ancient)
	}
	return ancient
}

func New(cfg *config.Config) (*ChainDB, error) {
	cdb := &ChainDB{
		cfg:       cfg,
		databases: make(map[*closeTrackingDB]struct{}),
	}
	cdb.closedState.Store(false)
	return cdb, nil
}
