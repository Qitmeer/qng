package database

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/database/chaindb"
	"github.com/Qitmeer/qng/database/legacychaindb"
)

func New(cfg *config.Config, interrupt <-chan struct{}) (model.DataBase, error) {
	if !cfg.DevNextGDB {
		return legacychaindb.New(cfg, interrupt)
	}
	return chaindb.New(cfg)
}
