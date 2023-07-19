package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database/common"
	l "github.com/Qitmeer/qng/log"
)

// update db to new version
func (b *BlockChain) upgradeDB(interrupt <-chan struct{}) error {
	version8 := uint32(8)
	version9 := uint32(9)
	version10 := uint32(10)
	version11 := uint32(11)
	if b.dbInfo.Version() == currentDatabaseVersion {
		return nil
	} else if b.dbInfo.Version() == version8 ||
		b.dbInfo.Version() == version9 ||
		b.dbInfo.Version() == version10 ||
		b.dbInfo.Version() == version11 {
		return fmt.Errorf("Your database version(%d) is too old and can only use old qng (release-v1.0.20)\n", b.dbInfo.Version())
	}
	//
	if onEnd := l.LogAndMeasureExecutionTime(log, "BlockChain.upgradeDB"); onEnd != nil {
		defer onEnd()
	}
	log.Info(fmt.Sprintf("Update cur db to new version: version(%d) -> version(%d) ...", b.dbInfo.Version(), currentDatabaseVersion))

	bidxStart := roughtime.Now()

	serializedData, err := b.consensus.DatabaseContext().GetBestChainState()
	if err != nil {
		return err
	}
	if serializedData == nil {
		return fmt.Errorf("No chain state")
	}
	bs, err := DeserializeBestChainState(serializedData)
	if err != nil {
		return err
	}

	err = b.bd.UpgradeDB(b.db, &bs.hash, bs.total, b.params.GenesisHash, interrupt, dbFetchBlockByHash,
		b.indexManager.IsDuplicateTx, b.meerChain.ETHChain().Ether().BlockChain(), b.meerChain.ETHChain().Ether().ChainDb())
	if err != nil {
		return err
	}

	b.dbInfo = common.NewDatabaseInfo(currentDatabaseVersion, serialization.CurrentCompressionVersion, currentBlockIndexVersion, roughtime.Now())
	err = b.consensus.DatabaseContext().PutInfo(b.dbInfo)
	if err != nil {
		return fmt.Errorf("Upgrade failed:%s. You can cleanup your block data base by '--cleanup'.\n", err)
	}
	log.Info(fmt.Sprintf("Finish update db version:time=%v", roughtime.Since(bidxStart)))
	return nil
}
