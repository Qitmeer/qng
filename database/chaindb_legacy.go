package database

import (
	"github.com/Qitmeer/qng/database/legacydb"
)

// TODO: It will soon be discarded in the near future
type LegacyChainDB struct {
	db legacydb.DB
}

func (cdb *LegacyChainDB) Name() string {
	return "Legacy Chain DB"
}
