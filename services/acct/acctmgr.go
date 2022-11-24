package acct

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/dbnamespace"
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
	chain    *blockchain.BlockChain
	cfg      *config.Config
	db       database.DB
	info     *AcctInfo
	utxoops  []*UTXOOP
	watchers map[string]*AcctBalanceWatcher
}

func (a *AccountManager) Start() error {
	if err := a.Service.Start(); err != nil {
		return err
	}
	if a.cfg.AcctMode {
		err := a.initDB(true)
		if err != nil {
			log.Error(fmt.Sprintf("Serious error, you can try to delete the data file(%s):%s", getDBPath(a.cfg.DataDir), err.Error()))
		}
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
		if err := a.db.Close(); err != nil {
			log.Error(err.Error())
		}
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
			if !a.info.IsCurrentVersion() {
				log.Warn(fmt.Sprintf("The account database version is not current(%d != %d). It will be rebuilt", a.info.version, CurrentAcctInfoVersion))
				rebuilddb = true
				return nil
			} else if curDAGID != a.info.updateDAGID {
				log.Warn(fmt.Sprintf("DAG is not consistent with account manager state"))
				if first {
					rebuilddb = true
					return nil
				} else {
					return fmt.Errorf("update dag id is inconformity:%d != %d", curDAGID, a.info.updateDAGID)
				}
			}
			return a.initWatchers(dbTx)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if rebuilddb {
		info := NewAcctInfo()
		if a.info != nil {
			info.addrs = a.info.addrs
		}
		a.info = info
		a.cleanDB()
		return a.initDB(false)
	} else if rebuildidx {
		if a.info.IsEmpty() {
			log.Info("There is no account address for the moment. You can add it later through (RPC:addBalance)")
			return nil
		}
		err = a.rebuild(nil)
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

func (a *AccountManager) rebuild(addrs []string) error {
	if len(addrs) > 0 {
		log.Trace(fmt.Sprintf("Try to rebuild account index for (%v)", addrs))
	} else {
		log.Trace("Try to rebuild account index")
	}
	ops := []*types.TxOutPoint{}
	entrys := []*blockchain.UtxoEntry{}
	err := a.chain.DB().View(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		utxoBucket := meta.Bucket(dbnamespace.UtxoSetBucketName)
		cursor := utxoBucket.Cursor()
		for ok := cursor.First(); ok; ok = cursor.Next() {
			op, err := parseOutpoint(cursor.Key())
			if err != nil {
				return err
			}
			serializedUtxo := cursor.Value()
			// Deserialize the utxo entry and return it.
			entry, err := blockchain.DeserializeUtxoEntry(serializedUtxo)
			if err != nil {
				return err
			}
			if entry.IsSpent() {
				continue
			}
			if len(addrs) > 0 {
				addr, _, err := a.checkUtxoEntry(entry, addrs)
				if err != nil {
					return err
				}
				if len(addr) <= 0 {
					continue
				}
			}
			ops = append(ops, op)
			entrys = append(entrys, entry)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(ops) > 0 {
		for i := 0; i < len(ops); i++ {
			err = a.apply(true, ops[i], entrys[i])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *AccountManager) checkUtxoEntry(entry *blockchain.UtxoEntry, tracks []string) (string, txscript.ScriptClass, error) {
	if entry.Amount().Id != types.MEERA {
		return "", txscript.NonStandardTy, nil
	}
	scriptClass, addrs, _, err := txscript.ExtractPkScriptAddrs(entry.PkScript(), params.ActiveNetParams.Params)
	if err != nil {
		return "", txscript.NonStandardTy, err
	}
	if len(addrs) <= 0 {
		return "", txscript.NonStandardTy, nil
	}
	addrStr := addrs[0].String()

	isHas := func(addr string) bool {
		if len(tracks) <= 0 {
			return false
		}
		for _, ad := range tracks {
			if ad == addr {
				return true
			}
		}
		return false
	}
	if !isHas(addrStr) {
		return "", txscript.NonStandardTy, nil
	}
	if scriptClass != txscript.PubKeyHashTy &&
		scriptClass != txscript.PubKeyTy &&
		scriptClass != txscript.CLTVPubKeyHashTy {
		return "", txscript.NonStandardTy, nil
	}
	return addrStr, scriptClass, nil
}

func (a *AccountManager) apply(add bool, op *types.TxOutPoint, entry *blockchain.UtxoEntry) error {
	addrStr, scriptClass, err := a.checkUtxoEntry(entry, a.info.addrs)
	if err != nil {
		return err
	}
	if len(addrStr) <= 0 {
		return nil
	}
	if add {
		if entry.Amount().Value == 0 && !entry.IsCoinBase() {
			return nil
		}
		if entry.IsCoinBase() && op.OutIndex != blockchain.CoinbaseOutput_subsidy {
			return nil
		}
		var balance *AcctBalance
		err = a.db.View(func(dbTx database.Tx) error {
			balance, err = DBGetACCTBalance(dbTx, addrStr)
			return err
		})
		if err != nil {
			return err
		}

		if balance == nil {
			if entry.IsCoinBase() ||
				scriptClass == txscript.CLTVPubKeyHashTy {
				balance = NewAcctBalance(0, 0, uint64(entry.Amount().Value), 1)
			} else {
				balance = NewAcctBalance(uint64(entry.Amount().Value), 1, 0, 0)
			}
			a.info.total++
			err = a.db.Update(func(tx database.Tx) error {
				return DBPutACCTInfo(tx, a.info)
			})
			if err != nil {
				return err
			}
		} else {
			if entry.IsCoinBase() ||
				scriptClass == txscript.CLTVPubKeyHashTy {
				balance.locked += uint64(entry.Amount().Value)
				balance.locUTXONum++
			} else {
				balance.normal += uint64(entry.Amount().Value)
				balance.norUTXONum++
			}

		}
		err = a.db.Update(func(tx database.Tx) error {
			return DBPutACCTBalance(tx, addrStr, balance)
		})
		if err != nil {
			return err
		}
		au := NewAcctUTXO()
		au.SetBalance(uint64(entry.Amount().Value))

		wb, exist := a.watchers[addrStr]
		if entry.IsCoinBase() {
			au.SetCoinbase()
			//
			if !exist {
				wb = NewAcctBalanceWatcher(addrStr, balance)
				a.watchers[addrStr] = wb
			}
			opk := OutpointKey(op)
			uw := BuildUTXOWatcher(opk, au, entry, a)
			if uw != nil {
				wb.Add(opk, uw)
			}
		} else if scriptClass == txscript.CLTVPubKeyHashTy {
			au.SetCLTV()
			if !exist {
				wb = NewAcctBalanceWatcher(addrStr, balance)
				a.watchers[addrStr] = wb
			}
			opk := OutpointKey(op)
			uw := BuildUTXOWatcher(opk, au, entry, a)
			if uw != nil {
				wb.Add(opk, uw)
			}
		} else {
			if exist {
				wb.ab = balance
			}
		}
		log.Trace(fmt.Sprintf("Add balance: %s (%s)", addrStr, au.String()))
		return a.db.Update(func(tx database.Tx) error {
			return DBPutACCTUTXO(tx, addrStr, op, au)
		})
	} else {
		err = a.db.Update(func(dbTx database.Tx) error {
			balance, er := DBGetACCTBalance(dbTx, addrStr)
			if er != nil {
				return er
			}
			if balance == nil {
				a.DelWatcher(addrStr, nil)
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
						balance.normal = 0
					} else {
						balance.normal -= amount
					}
					if balance.norUTXONum > 0 {
						balance.norUTXONum--
					}
				}
			}
			log.Trace(fmt.Sprintf("Del balance: %s (%s:%d)", addrStr, op.Hash.String(), op.OutIndex))
			if balance.IsEmpty() {
				er = a.cleanBalanceDB(dbTx, addrStr)
				if er != nil {
					return er
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
			if balance.locUTXONum <= 0 {
				a.DelWatcher(addrStr, nil)
			} else if entry.IsCoinBase() {
				a.DelWatcher(addrStr, op)
			}
			return nil
		})
		return err
	}
}

func (a *AccountManager) DelWatcher(addr string, op *types.TxOutPoint) {
	if op != nil {
		wb, exist := a.watchers[addr]
		if !exist {
			return
		}
		wb.Del(OutpointKey(op))
	} else {
		delete(a.watchers, addr)
	}
}

func (a *AccountManager) initWatchers(dbTx database.Tx) error {
	meta := dbTx.Metadata()
	balBucket := meta.Bucket(BalanceBucketName)
	if balBucket == nil {
		return nil
	}
	kus := [][]byte{}
	aus := []*AcctUTXO{}
	wbs := []*AcctBalanceWatcher{}
	err := balBucket.ForEach(func(k, v []byte) error {
		balance := &AcctBalance{}
		err := balance.Decode(bytes.NewReader(v))
		if err != nil {
			return err
		}
		if balance.locUTXONum <= 0 {
			return nil
		}
		balUTXOBucket := balBucket.Bucket(GetACCTUTXOKey(string(k)))
		if balUTXOBucket == nil {
			return nil
		}
		balUTXOBucket.ForEach(func(ku, vu []byte) error {
			au := NewAcctUTXO()
			err := au.Decode(bytes.NewReader(vu))
			if err != nil {
				return err
			}
			if !au.IsCoinbase() &&
				!au.IsCLTV() {
				return nil
			}
			addrStr := string(k)
			wb, exist := a.watchers[addrStr]
			if !exist {
				wb = NewAcctBalanceWatcher(addrStr, balance)
				a.watchers[addrStr] = wb
			}
			kus = append(kus, ku)
			aus = append(aus, au)
			wbs = append(wbs, wb)
			return nil
		})
		return nil
	})
	if err != nil {
		return err
	}
	if len(aus) > 0 {
		for i := 0; i < len(aus); i++ {
			uw := BuildUTXOWatcher(kus[i], aus[i], nil, a)
			if uw != nil {
				wbs[i].Add(kus[i], uw)
			}
		}
	}
	if len(a.watchers) > 0 {
		for _, w := range a.watchers {
			err = w.Update(a)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *AccountManager) Apply(add bool, op *types.TxOutPoint, entry *blockchain.UtxoEntry) error {
	if !a.cfg.AcctMode {
		return nil
	}
	a.utxoops = append(a.utxoops, &UTXOOP{add: add, op: op, entry: entry})
	return nil
}

func (a *AccountManager) Commit() error {
	if !a.cfg.AcctMode {
		return nil
	}
	defer func() {
		a.utxoops = []*UTXOOP{}
	}()

	curDAGID := uint32(a.chain.BlockDAG().GetBlockTotal())
	a.info.updateDAGID = curDAGID
	err := a.db.Update(func(dbTx database.Tx) error {
		return DBPutACCTInfo(dbTx, a.info)
	})
	if err != nil {
		return err
	}

	for _, op := range a.utxoops {
		err := a.apply(op.add, op.op, op.entry)
		if err != nil {
			return err
		}
	}

	if len(a.watchers) > 0 {
		for _, w := range a.watchers {
			err = w.Update(a)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *AccountManager) GetBalance(addr string) (uint64, error) {
	if !a.cfg.AcctMode {
		return 0, fmt.Errorf("Please enable --acctmode")
	}
	if !address.IsForCurNetwork(addr) {
		return 0, fmt.Errorf("network error:%s", addr)
	}
	result := uint64(0)
	wb, exist := a.watchers[addr]
	if exist {
		return wb.GetBalance(), nil
	}

	err := a.db.Update(func(dbTx database.Tx) error {
		balance, err := DBGetACCTBalance(dbTx, addr)
		if err != nil {
			return err
		}
		if balance != nil {
			result = balance.normal
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (a *AccountManager) GetUTXOs(addr string) ([]UTXOResult, error) {
	utxos := []UTXOResult{}
	err := a.db.Update(func(dbTx database.Tx) error {
		us := DBGetACCTUTXOs(dbTx, addr)
		if len(us) > 0 {
			for k, v := range us {
				ur := UTXOResult{Type: v.TypeStr(), Amount: v.balance, Status: "valid"}
				wb, exist := a.watchers[addr]
				if exist {
					wu := wb.GetByOPS(k)
					if wu != nil {
						if wu.IsUnlocked() {
							ur.Status = "unlocked"
						} else {
							ur.Status = "locked"
						}
					}
				}

				opk, err := hex.DecodeString(k)
				if err != nil {
					return err
				}
				op, err := parseOutpoint(opk)
				if err != nil {
					return err
				}
				ur.PreTxHash = op.Hash.String()
				ur.PreOutIdx = op.OutIndex
				utxos = append(utxos, ur)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return utxos, nil
}

func (a *AccountManager) AddAddress(addr string) error {
	if !a.cfg.AcctMode {
		return fmt.Errorf("Please enable --acctmode")
	}
	if !address.IsForCurNetwork(addr) {
		return fmt.Errorf("network error:%s", addr)
	}
	if a.info.Has(addr) {
		return fmt.Errorf(fmt.Sprintf("Already exists:%s", addr))
	}
	_, exist := a.watchers[addr]
	if exist {
		return fmt.Errorf(fmt.Sprintf("Already exists watcher:%s", addr))
	}
	a.info.Add(addr)
	err := a.db.Update(func(dbTx database.Tx) error {
		return a.cleanBalanceDB(dbTx, addr)
	})
	if err != nil {
		return err
	}
	return a.rebuild([]string{addr})
}

func (a *AccountManager) cleanBalanceDB(dbTx database.Tx, addr string) error {
	er := DBDelACCTBalance(dbTx, addr)
	if er != nil {
		return er
	}
	er = DBDelACCTUTXOs(dbTx, addr)
	if er != nil {
		return er
	}
	if a.info.total > 0 {
		a.info.total--
		er = DBPutACCTInfo(dbTx, a.info)
		if er != nil {
			return er
		}
	}
	return nil
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
		chain:    chain,
		cfg:      cfg,
		info:     NewAcctInfo(),
		utxoops:  []*UTXOOP{},
		watchers: map[string]*AcctBalanceWatcher{},
	}
	return &a, nil
}
