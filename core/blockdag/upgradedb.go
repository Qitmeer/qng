package blockdag

import (
	"github.com/Qitmeer/qng/database"
)

// update db to new version
func (bd *BlockDAG) UpgradeDB(dbTx database.Tx) error {
	return nil
}
