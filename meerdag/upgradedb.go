package meerdag

import (
	"github.com/Qitmeer/qng-core/database"
)

// update db to new version
func (bd *MeerDAG) UpgradeDB(dbTx database.Tx) error {
	return nil
}
