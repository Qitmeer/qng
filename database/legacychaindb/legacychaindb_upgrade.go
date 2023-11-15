package legacychaindb

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/legacydb"
	l "github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"github.com/schollz/progressbar/v3"
	"math"
)

func (cdb *LegacyChainDB) TryUpgrade(di *common.DatabaseInfo, interrupt <-chan struct{}) error {
	if di.Version() == common.CurrentDatabaseVersion {
		// To fix old data, re index old genesis transaction data.
		txId := params.ActiveNetParams.GenesisBlock.Transactions()[0].Hash()
		_, blockHash, err := cdb.GetTxIdxEntry(txId, false)
		if err != nil {
			return err
		}
		if blockHash == nil {
			log.Info("Re index genesis transaction for legacy database")
			return cdb.doPutTxIndexEntrys(params.ActiveNetParams.GenesisBlock, 0)
		}
		return nil
	} else if di.Version() > 13 {
		return fmt.Errorf("The data is temporarily incompatible.")
	} else if di.Version() < 13 {
		return fmt.Errorf("Your database version(%d) is too old and can only use old qng (release-v1.0.x)\n", di.Version())
	}
	if onEnd := l.LogAndMeasureExecutionTime(log, "Upgrade Legacy DB"); onEnd != nil {
		defer onEnd()
	}
	log.Info(fmt.Sprintf("Update cur db to new version: version(%d) -> version(%d) ...", di.Version(), common.CurrentDatabaseVersion))
	//
	var ibID uint32
	var err error
	err = cdb.db.View(func(dbTx legacydb.Tx) error {
		// Create the bucket for the current tips as needed.
		meta := dbTx.Metadata()
		bucket := meta.Bucket(dbnamespace.IndexTipsBucketName)
		if bucket == nil {
			return fmt.Errorf("No index tips bucket")
		}
		// Fetch the current tip for the index.
		_, ibID, err = dbFetchIndexerTip(dbTx, txIndexKey)
		return err
	})
	if err != nil {
		return err
	}
	// Nothing to do if the index does not have any entries yet.
	if ibID == math.MaxUint32 {
		return nil
	}
	//
	var bar *progressbar.ProgressBar
	logLvl := l.Glogger().GetVerbosity()
	bar = progressbar.Default(int64(ibID), "LegacyDB:")
	l.Glogger().Verbosity(l.LvlCrit)
	defer func() {
		bar.Finish()
		l.Glogger().Verbosity(logLvl)
	}()
	//

	var blockhash *hash.Hash
	var sb *types.SerializedBlock
	var blockid uint
	for i := uint32(1); i <= ibID; i++ {
		bar.Add(1)
		if system.InterruptRequested(interrupt) {
			return fmt.Errorf("interrupt upgrade database")
		}
		blockid, err = cdb.GetBlockIdByOrder(uint(i))
		if err != nil {
			return err
		}
		err = cdb.db.View(func(dbTx legacydb.Tx) error {
			blockhash, err = dbFetchBlockHashByIID(dbTx, uint32(blockid))
			return err
		})
		if err != nil {
			return err
		}
		sb, err = cdb.GetBlock(blockhash)
		if err != nil {
			return err
		}
		dblock := &meerdag.Block{}
		dblock.SetID(blockid)
		ib := &meerdag.PhantomBlock{Block: dblock}

		err = meerdag.DBGetDAGBlock(cdb, ib)
		if err != nil {
			return err
		}
		txs := sb.Transactions()
		for _, txid := range ib.GetState().GetDuplicateTxs() {
			if txid >= len(txs) {
				continue
			}
			txs[txid].IsDuplicate = true
		}
		err = cdb.doPutTxIndexEntrys(sb, blockid)
		if err != nil {
			return err
		}
	}

	//
	newDBInfo := common.NewDatabaseInfo(common.CurrentDatabaseVersion, di.CompVer(), di.BidxVer(), roughtime.Now())
	err = cdb.PutInfo(newDBInfo)
	if err != nil {
		return fmt.Errorf("Upgrade failed:%s. You can cleanup your block data base by '--cleanup'.\n", err)
	}
	log.Info("Finish update db version", "num index", ibID)
	return nil
}
