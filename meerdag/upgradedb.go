package meerdag

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/database"
)

// update db to new version
func (bd *MeerDAG) UpgradeDB(dbTx database.Tx, mainTip *hash.Hash, total uint64, genesis *hash.Hash) error {
	// TODO: Reserved for next upgrade
	return nil
}
