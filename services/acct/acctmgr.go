package acct

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
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

func (a *AccountManager) Stop() error {
	if err := a.Service.Stop(); err != nil {
		return err
	}
	if a.db != nil {
		return a.db.Close()
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
	rebuilddb := false
	rebuildidx := false
	err = a.db.Update(func(dbTx database.Tx) error {
		info, err := DBGetACCTInfo(dbTx)
		if err != nil {
			return err
		}
		if info == nil {
			a.info.updateDAGID = curDAGID
			err := DBPutACCTInfo(dbTx, a.info)
			if err != nil {
				return err
			}
			log.Info("Init account manager info")
			rebuildidx = true
		} else {
			a.info = info
			log.Info(fmt.Sprintf("Load account manager info:%s", a.info.String()))
			if curDAGID != a.info.updateDAGID {
				log.Warn(fmt.Sprintf("DAG is not consistent with account manager state"))
				if first {
					rebuilddb = true
				} else {
					return fmt.Errorf("update dag id is inconformity:%d != %d", curDAGID, a.info.updateDAGID)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if rebuilddb {
		a.info = NewAcctInfo()
		a.cleanDB()
		return a.initDB(false)
	} else if rebuildidx {
		err = a.rebuild()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *AccountManager) cleanDB() {
	if a.db == nil {
		db, err := loadDB(a.cfg.DbType, a.cfg.DataDir, false)
		if err != nil {
			return
		}
		a.db = db
	}

	if a.db != nil {
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

	err := removeDB(getDBPath(a.cfg.DataDir))
	if err != nil {
		log.Error(err.Error())
	}
}

func (a *AccountManager) rebuild() error {
	log.Trace("Try to rebuild account index")
	err := a.chain.DB().View(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		utxoBucket := meta.Bucket(dbnamespace.UtxoSetBucketName)
		cursor := utxoBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			serializedUtxo := utxoBucket.Get(cursor.Key())
			txhash, err := hash.NewHash(cursor.Key()[:hash.HashSize])
			if err != nil {
				return err
			}
			txOutIdex, size := serialization.DeserializeVLQ(cursor.Key()[hash.HashSize:])
			if size <= 0 {
				return fmt.Errorf("DeserializeVLQ:%s %v", txhash.String(), cursor.Key()[hash.HashSize:])
			}
			// Deserialize the utxo entry and return it.
			entry, err := blockchain.DeserializeUtxoEntry(serializedUtxo)
			if err != nil {
				return err
			}
			err = a.apply(true, types.NewOutPoint(txhash, uint32(txOutIdex)), entry)
			if err != nil {
				return err
			}
			return nil
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (a *AccountManager) apply(add bool, op *types.TxOutPoint, entry *blockchain.UtxoEntry) error {
	scriptClass, addrs, _, err := txscript.ExtractPkScriptAddrs(entry.PkScript(), params.ActiveNetParams.Params)
	if err != nil {
		return err
	}
	if len(addrs) <= 0 {
		return nil
	}
	if scriptClass != txscript.PubKeyHashTy &&
		scriptClass != txscript.PubKeyTy {
		return nil
	}

	if add {
		err = a.db.Update(func(dbTx database.Tx) error {
			addrStr := addrs[0].String()
			balance, er := DBGetACCTBalance(dbTx, addrStr)
			if er != nil {
				return er
			}
			if balance == nil {
				if entry.IsCoinBase() {
					balance = NewAcctBalance(0, 0, uint64(entry.Amount().Value), 1)
				} else {
					balance = NewAcctBalance(uint64(entry.Amount().Value), 1, 0, 0)
				}
				a.info.addrTotal++
				er = DBPutACCTInfo(dbTx, a.info)
				if er != nil {
					return er
				}
			} else {
				if entry.IsCoinBase() {
					balance.locked += uint64(entry.Amount().Value)
					balance.locUTXONum++
				} else {
					balance.normal += uint64(entry.Amount().Value)
					balance.norUTXONum++
				}

			}
			er = DBPutACCTBalance(dbTx, addrStr, balance)
			if er != nil {
				return er
			}
			au := NewAcctUTXO()
			au.SetBalance(uint64(entry.Amount().Value))
			er = DBPutACCTUTXO(dbTx, addrStr, op, au)
			if er != nil {
				return er
			}
			return nil
		})
		return err
	} else {
		err = a.db.Update(func(dbTx database.Tx) error {
			addrStr := addrs[0].String()
			balance, er := DBGetACCTBalance(dbTx, addrStr)
			if er != nil {
				return er
			}
			if balance == nil {
				return nil
			} else {
				amount := uint64(entry.Amount().Value)
				if entry.IsCoinBase() {
					if balance.locked <= amount {
						balance.locked = 0
					} else {
						balance.locked -= amount
					}
					if balance.locUTXONum > 0 {
						balance.locUTXONum--
					}
				} else {
					if balance.normal <= amount {
						balance.locked = 0
					} else {
						balance.normal -= amount
					}
					if balance.norUTXONum > 0 {
						balance.norUTXONum--
					}
				}
			}
			if balance.IsEmpty() {
				er = DBDelACCTBalance(dbTx, addrStr)
				if er != nil {
					return er
				}
				er = DBDelACCTUTXOs(dbTx, addrStr)
				if er != nil {
					return er
				}
				if a.info.addrTotal > 0 {
					a.info.addrTotal--
					er = DBPutACCTInfo(dbTx, a.info)
					if er != nil {
						return er
					}
				}
			} else {
				er = DBPutACCTBalance(dbTx, addrStr, balance)
				if er != nil {
					return er
				}
				er = DBDelACCTUTXO(dbTx, addrStr, op)
				if er != nil {
					return er
				}
			}
			return nil
		})
		return err
	}
}

func (a *AccountManager) Apply(add bool, op *types.TxOutPoint, entry *blockchain.UtxoEntry) error {
	return a.apply(add, op, entry)
}

func (a *AccountManager) GetBalance(address string) (uint64, error) {
	result := uint64(0)
	err := a.db.Update(func(dbTx database.Tx) error {
		balance, err := DBGetACCTBalance(dbTx, address)
		if err != nil {
			return err
		}
		result = balance.normal
		return nil
	})
	if err != nil {
		return 0, err
	}
	return result, nil
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
