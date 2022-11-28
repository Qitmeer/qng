package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/core/blockchain/token"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database"
	l "github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
)

// update db to new version
func (b *BlockChain) upgradeDB(interrupt <-chan struct{}) error {
	version8 := uint32(8)
	version9 := uint32(9)
	version10 := uint32(10)
	version11 := uint32(11)
	if b.dbInfo.version == currentDatabaseVersion {
		return nil
	} else if b.dbInfo.version != version8 &&
		b.dbInfo.version != version9 &&
		b.dbInfo.version != version10 &&
		b.dbInfo.version != version11 {
		return fmt.Errorf("Only supported update version(%d or %d,%d,%d) -> version(%d), but cur db is version:%d\n", version8, version9, version10, version11, currentDatabaseVersion, b.dbInfo.version)
	}
	if onEnd := l.LogAndMeasureExecutionTime(log, "BlockChain.upgradeDB"); onEnd != nil {
		defer onEnd()
	}
	log.Info(fmt.Sprintf("Update cur db to new version: version(%d) -> version(%d) ...", b.dbInfo.version, currentDatabaseVersion))
	if b.dbInfo.version < version10 {
		err := b.indexManager.Drop()
		if err != nil {
			log.Debug(err.Error())
		}
	}
	err := b.db.Update(func(dbTx database.Tx) error {
		bidxStart := roughtime.Now()
		meta := dbTx.Metadata()
		serializedData := meta.Get(dbnamespace.ChainStateKeyName)
		if serializedData == nil {
			return nil
		}
		state, err := DeserializeBestChainState(serializedData)
		if err != nil {
			return err
		}

		if b.dbInfo.version == version8 {
			err = b.bd.UpgradeDB(dbTx, &state.hash, state.total, b.params.GenesisHash, true, interrupt)
			if err != nil {
				return err
			}
		} else {
			err = b.bd.UpgradeDB(dbTx, &state.hash, state.total, b.params.GenesisHash, false, interrupt)
			if err != nil {
				return err
			}
		}

		// token
		if b.dbInfo.version < version11 {
			tokenTipID := meerdag.MaxId
			bid, er := meerdag.DBGetBlockIdByHash(dbTx, &state.tokenTipHash)
			if er == nil {
				tokenTipID = uint(bid)
			}
			if tokenTipID != meerdag.MaxId {
				ts := b.GetTokenState(uint32(tokenTipID))
				if ts == nil {
					return fmt.Errorf("token state error:%d", tokenTipID)
				}
				oldIds := ts.Types.Ids()
				genTS := token.BuildGenesisTokenState()
				for _, ty := range genTS.Types {
					_, ok := ts.Types[ty.Id]
					if !ok || b.dbInfo.version == version10 {
						ts.Types[ty.Id] = ty
					}
				}
				err = token.DBPutTokenState(dbTx, uint32(tokenTipID), ts)
				if err != nil {
					return err
				}
				ts.Commit()
				log.Info(fmt.Sprintf("Upgrade token state(%s)(id=%d):%v => %v", state.tokenTipHash, tokenTipID, oldIds, ts.Types.Ids()))
			}
		}
		// save
		b.dbInfo = &databaseInfo{
			version: currentDatabaseVersion,
			compVer: serialization.CurrentCompressionVersion,
			bidxVer: currentBlockIndexVersion,
			created: roughtime.Now(),
		}
		err = dbPutDatabaseInfo(dbTx, b.dbInfo)
		if err != nil {
			return err
		}

		log.Info(fmt.Sprintf("Finish update db version:time=%v", roughtime.Since(bidxStart)))
		return nil
	})
	if err != nil {
		return fmt.Errorf("Upgrade failed:%s. You can cleanup your block data base by '--cleanup'.\n", err)
	}
	return nil
}
