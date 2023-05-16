package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database"
	l "github.com/Qitmeer/qng/log"
)

// update db to new version
func (b *BlockChain) upgradeDB(interrupt <-chan struct{}) error {
	version8 := uint32(8)
	version9 := uint32(9)
	version10 := uint32(10)
	version11 := uint32(11)
	if b.dbInfo.version == currentDatabaseVersion {
		return nil
	} else if b.dbInfo.version == version8 ||
		b.dbInfo.version == version9 ||
		b.dbInfo.version == version10 ||
		b.dbInfo.version == version11 {
		return fmt.Errorf("Your database version(%d) is too old and can only use old qng (release-v1.0.20)\n", b.dbInfo.version)
	}
	//
	if onEnd := l.LogAndMeasureExecutionTime(log, "BlockChain.upgradeDB"); onEnd != nil {
		defer onEnd()
	}
	log.Info(fmt.Sprintf("Update cur db to new version: version(%d) -> version(%d) ...", b.dbInfo.version, currentDatabaseVersion))

	bidxStart := roughtime.Now()

	var bs *bestChainState
	err := b.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		serializedData := meta.Get(dbnamespace.ChainStateKeyName)
		if serializedData == nil {
			return fmt.Errorf("No chain state")
		}
		state, err := DeserializeBestChainState(serializedData)
		if err != nil {
			return err
		}
		bs = &state
		return nil
	})
	err = b.bd.UpgradeDB(b.db, &bs.hash, bs.total, b.params.GenesisHash, interrupt, dbFetchBlockByHash, b.indexManager.IsDuplicateTx)
	if err != nil {
		return err
	}

	err = b.db.Update(func(dbTx database.Tx) error {
		// save
		b.dbInfo = &databaseInfo{
			version: currentDatabaseVersion,
			compVer: serialization.CurrentCompressionVersion,
			bidxVer: currentBlockIndexVersion,
			created: roughtime.Now(),
		}
		e := dbPutDatabaseInfo(dbTx, b.dbInfo)
		if e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Upgrade failed:%s. You can cleanup your block data base by '--cleanup'.\n", err)
	}
	log.Info(fmt.Sprintf("Finish update db version:time=%v", roughtime.Since(bidxStart)))
	return nil
}
