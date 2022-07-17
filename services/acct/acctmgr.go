package acct

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
)

// account manager communicate with various backends for signing transactions.
type AccountManager struct {
	service.Service
	chain *blockchain.BlockChain
	cfg   *config.Config
	db    database.DB
	info  *AcctInfo
}

func (a *AccountManager) Start() error {
	if err := a.Service.Start(); err != nil {
		return err
	}
	if a.cfg.AcctMode {
		return a.initDB(true)
	} else {
		a.cleanDB()
	}
	return nil
}

func (a *AccountManager) initDB(first bool) error {
	log.Info("AccountManager enable account mode")
	var err error
	a.db, err = loadDB(a.cfg.DbType, a.cfg.DataDir, true)
	if err != nil {
		return err
	}
	curDAGID := uint32(a.chain.BlockDAG().GetBlockTotal())
	rebuild := false
	err = a.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		infoData := meta.Get(InfoBucketName)
		if infoData == nil {
			a.info.updateDAGID = curDAGID
			err := DBPutACCTInfo(dbTx, a.info)
			if err != nil {
				return err
			}
			log.Info("Init account manager info")
			err = a.rebuild()
			if err != nil {
				return err
			}
		} else {
			err := a.info.Decode(bytes.NewReader(infoData))
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Load account manager info:%s", a.info.String()))
			if curDAGID != a.info.updateDAGID {
				log.Warn(fmt.Sprintf("DAG is not consistent with account manager state"))
				if first {
					rebuild = true
				} else {
					return fmt.Errorf("update dag id is inconformity:%d != %d", curDAGID, a.info.updateDAGID)
				}
			}
		}
		if rebuild {
			return nil
		}
		//
		_, err := meta.CreateBucketIfNotExists(BalanceBucketName)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if rebuild {
		a.info = NewAcctInfo()
		a.cleanDB()
		return a.initDB(false)
	}
	return nil
}

func (a *AccountManager) cleanDB() {
	err := a.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		infoData := meta.Get(InfoBucketName)
		if infoData == nil {
			return nil
		} else {
			err := meta.Delete(InfoBucketName)
			if err != nil {
				return err
			}
			if meta.Bucket(BalanceBucketName) != nil {
				err := meta.DeleteBucket(BalanceBucketName)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Error(err.Error())
	}
	err = removeDB(getDBPath(a.cfg.DataDir))
	if err != nil {
		log.Error(err.Error())
	}
}

func (a *AccountManager) rebuild() error {

}

func (a *AccountManager) apply(add bool, op *types.TxOutPoint, entry *blockchain.UtxoEntry) {
	if add {
		err := a.db.Update(func(dbTx database.Tx) error {
			meta := dbTx.Metadata()
			infoData := meta.Get(InfoBucketName)
			if infoData == nil {
				return nil
			} else {
				err := meta.Delete(InfoBucketName)
				if err != nil {
					return err
				}
				if meta.Bucket(BalanceBucketName) != nil {
					err := meta.DeleteBucket(BalanceBucketName)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Error(err.Error())
		}
	}
}

func (a *AccountManager) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicAccountManagerAPI(a),
			Public:    true,
		},
	}
}

func New(chain *blockchain.BlockChain, cfg *config.Config) (*AccountManager, error) {
	a := AccountManager{
		chain: chain,
		cfg:   cfg,
		info:  NewAcctInfo(),
	}
	return &a, nil
}
