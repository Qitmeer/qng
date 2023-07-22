package legacychaindb

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/params"
	"os"
	"path/filepath"
)

const (
	// blockDbNamePrefix is the prefix for the block database name.  The
	// database type is appended to this value to form the full block
	// database name.
	blockDbNamePrefix = "blocks"
)

var (
	CreateIfNoExist = true
)

// loadBlockDB loads (or creates when needed) the block database taking into
// account the selected database backend and returns a handle to it.  It also
// contains additional logic such warning the user if there are multiple
// databases which consume space on the file system and ensuring the regression
// test database is clean when in regression test mode.
func LoadBlockDB(cfg *config.Config) (legacydb.DB, error) {
	// The database name is based on the database type.
	dbPath := BlockDbPath(cfg)
	log.Info("Loading block database", "dbPath", dbPath)
	db, err := legacydb.Open(cfg.DbType, dbPath, params.ActiveNetParams.Net)
	if err != nil {
		if CreateIfNoExist {
			// Return the error if it's not because the database doesn't
			// exist.
			if dbErr, ok := err.(legacydb.Error); !ok || dbErr.ErrorCode !=
				legacydb.ErrDbDoesNotExist {

				return nil, err
			}
			// Create the db if it does not exist.
			err = os.MkdirAll(cfg.DataDir, 0700)
			if err != nil {
				return nil, err
			}
			db, err = legacydb.Create(cfg.DbType, dbPath, params.ActiveNetParams.Net)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	log.Info("Block database loaded")
	return db, nil
}

// blockDbPath returns the path to the block database given a database type.
func BlockDbPath(cfg *config.Config) string {
	// The database name is based on the database type.
	dbName := blockDbNamePrefix + "_" + cfg.DbType
	dbPath := filepath.Join(cfg.DataDir, dbName)
	return dbPath
}
