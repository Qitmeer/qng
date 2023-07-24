package database

import (
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/shutdown"
	"github.com/Qitmeer/qng/database/chaindb"
	"github.com/Qitmeer/qng/database/legacychaindb"
	_ "github.com/Qitmeer/qng/database/legacydb/ffldb"
	"github.com/Qitmeer/qng/meerevm/amana"
	"github.com/Qitmeer/qng/meerevm/meer"
	"os"
)

func New(cfg *config.Config, interrupt <-chan struct{}) (model.DataBase, error) {
	// Cleanup the block database
	if cfg.Cleanup {
		Cleanup(cfg)
		return nil, nil
	}
	if !cfg.DevNextGDB {
		return legacychaindb.New(cfg, interrupt)
	}
	return chaindb.New(cfg)
}

func Cleanup(cfg *config.Config) {
	var dbPath string
	if cfg.DevNextGDB {
		dbPath = cfg.ResolveDataPath(chaindb.DBDirectoryName)
	} else {
		dbPath = legacychaindb.BlockDbPath(cfg)
	}
	err := remove(dbPath)
	if err != nil {
		log.Error(err.Error())
	}
	meer.Cleanup(cfg)
	amana.Cleanup(cfg)
	err = shutdown.NewTracker(cfg.DataDir).Done()
	if err != nil {
		log.Error(err.Error())
	}
	log.Info("Finished cleanup")
}

// remove removes the existing database
func remove(dbPath string) error {
	// Remove the old database if it already exists.
	fi, err := os.Stat(dbPath)
	if err == nil {
		log.Info(fmt.Sprintf("Removing block database from '%s'", dbPath))
		if fi.IsDir() {
			err := os.RemoveAll(dbPath)
			if err != nil {
				return err
			}
		} else {
			err := os.Remove(dbPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
