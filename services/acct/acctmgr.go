package acct

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/v2/common/system"
	"github.com/Qitmeer/qng/v2/config"
	"github.com/Qitmeer/qng/v2/consensus/model"
	"github.com/Qitmeer/qng/v2/core/address"
	"github.com/Qitmeer/qng/v2/core/blockchain"
	"github.com/Qitmeer/qng/v2/core/blockchain/utxo"
	"github.com/Qitmeer/qng/v2/core/event"
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/database/legacydb"
	"github.com/Qitmeer/qng/v2/engine/txscript"
	qlog "github.com/Qitmeer/qng/v2/log"
	"github.com/Qitmeer/qng/v2/meerevm/eth"
	"github.com/Qitmeer/qng/v2/node/service"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/Qitmeer/qng/v2/rpc/api"
	"github.com/Qitmeer/qng/v2/rpc/client/cmds"
	"github.com/schollz/progressbar/v3"
	"sync"
)

// account manager communicate with various backends for signing transactions.
type AccountManager struct {
	service.Service
	chain     *blockchain.BlockChain
	cfg       *config.Config
	db        legacydb.DB
	info      *AcctInfo
	utxoops   []*UTXOOP
	watchLock sync.RWMutex
	watchers  map[string]*AcctBalanceWatcher
	events    *event.Feed
	statpoint model.Block
}

func (a *AccountManager) SetEvents(evs *event.Feed) {
	a.events = evs
}

func (a *AccountManager) Start() error {
	if err := a.Service.Start(); err != nil {
		return err
	}
	if a.cfg.AcctMode {
		err := a.initDB(true)
		if err != nil {
			return fmt.Errorf("Serious error, you can try to delete the data file(%s):%s", getDBPath(getDataDir(a.cfg)), err.Error())
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
	a.db, err = loadDB("ffldb", getDataDir(a.cfg), true)
	if err != nil {
		return err
	}
	curDAGID := uint32(a.chain.BlockDAG().GetBlockTotal())
	rebuilddb := false
	rebuildidx := false
	trackAddrs := false
	err = a.db.Update(func(dbTx legacydb.Tx) error {
		info, err := DBGetACCTInfo(dbTx)
		if err != nil {
			return err
		}
		if info == nil {
			a.info.updateDAGID = curDAGID
			a.info.all = isAllMode(a.cfg.AcctAddrs)
			if !a.info.all {
				err = a.trackCfgAddresses(true)
				if err != nil {
					return err
				}
			}
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
			} else if len(a.cfg.AcctAddrs) > 0 {
				if a.info.all && !isAllMode(a.cfg.AcctAddrs) {
					return errors.New("Already in Mode ALL, canceling it will damage the DB")
				} else if !a.info.all && isAllMode(a.cfg.AcctAddrs) {
					a.info.all = true
					rebuilddb = true
					log.Info("Attempting to convert to ALL mode requires resetting the database")
					return nil
				} else if !a.info.all && !isAllMode(a.cfg.AcctAddrs) {
					trackAddrs = true
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
			info.all = a.info.all
		}
		a.info = info
		a.cleanDB()
		return a.initDB(false)
	} else if rebuildidx {
		if a.info.IsEmpty() && !a.isAllMode() {
			log.Info("There is no account address for the moment. You can add it later through (RPC:addBalance)")
			return nil
		}

		err = a.rebuild(nil)
		if err != nil {
			return err
		}
	} else if trackAddrs {
		err = a.trackCfgAddresses(false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *AccountManager) cleanDB() {
	if a.db == nil {
		db, err := loadDB("ffldb", getDataDir(a.cfg), false)
		if err != nil {
			return
		}
		a.db = db
	}

	if a.db != nil {
		err := a.db.Update(func(dbTx legacydb.Tx) error {
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

	err := RemoveDB(a.cfg)
	if err != nil {
		log.Error(err.Error())
	}
}

func (a *AccountManager) rebuild(addrs []string) error {
	if len(addrs) > 0 {
		log.Info(fmt.Sprintf("Try to rebuild account index for (%v)", addrs))
	} else {
		if a.info.all {
			log.Info("Try to rebuild account index, the first time may be very slow, please be patient and wait")
		} else {
			log.Info("Try to rebuild account index")
		}
	}
	ops := []*types.TxOutPoint{}
	entrys := []*utxo.UtxoEntry{}
	err := a.chain.DB().ForeachUtxo(func(key []byte, data []byte) error {
		op, err := parseOutpoint(key)
		if err != nil {
			return err
		}
		serializedUtxo := data
		// Deserialize the utxo entry and return it.
		entry, err := utxo.DeserializeUtxoEntry(serializedUtxo)
		if err != nil {
			return err
		}
		if entry.IsSpent() {
			return nil
		}
		if len(addrs) > 0 {
			addr, _, err := a.checkUtxoEntry(entry, addrs)
			if err != nil {
				return err
			}
			if len(addr) <= 0 {
				return nil
			}
		}
		ops = append(ops, op)
		entrys = append(entrys, entry)
		return nil
	})
	if err != nil {
		return err
	}
	if len(ops) > 0 {
		logLvl := qlog.Glogger().GetVerbosity()
		bar := progressbar.Default(int64(len(ops)), "Build ACCT Index:")
		qlog.Glogger().Verbosity(qlog.LvlCrit)
		eth.InitLog(qlog.LvlCrit.String(), a.chain.Consensus().Config().DebugPrintOrigins)
		defer func() {
			fmt.Println()
			qlog.Glogger().Verbosity(logLvl)
			eth.InitLog(logLvl.String(), a.chain.Consensus().Config().DebugPrintOrigins)
			bar.Close()
		}()

		for i := 0; i < len(ops); i++ {
			if system.InterruptRequested(a.chain.Consensus().Interrupt()) {
				err := RemoveDB(a.cfg)
				if err != nil {
					return err
				}
				return fmt.Errorf("interrupt rebuild")
			}
			err = a.apply(true, ops[i], entrys[i])
			if err != nil {
				return err
			}
			bar.Add(1)
		}

	}

	return nil
}

func (a *AccountManager) checkUtxoEntry(entry *utxo.UtxoEntry, tracks []string) (string, txscript.ScriptClass, error) {
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
		if a.isAllMode() {
			return true
		}
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

func (a *AccountManager) apply(add bool, op *types.TxOutPoint, entry *utxo.UtxoEntry) error {

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
		err = a.db.View(func(dbTx legacydb.Tx) error {
			balance, err = DBGetACCTBalance(dbTx, addrStr)
			return err
		})
		if err != nil {
			return err
		}
		infoChange := false
		if balance == nil {
			if entry.IsCoinBase() ||
				scriptClass == txscript.CLTVPubKeyHashTy {
				balance = NewAcctBalance(0, 0, uint64(entry.Amount().Value), 1)
			} else {
				balance = NewAcctBalance(uint64(entry.Amount().Value), 1, 0, 0)
			}
			a.info.total++
			infoChange = true
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
		err = a.db.Update(func(tx legacydb.Tx) error {
			return DBPutACCTBalance(tx, addrStr, balance)
		})
		if err != nil {
			return err
		}
		au := NewAcctUTXO(uint64(entry.Amount().Value))
		if entry.IsCoinBase() {
			au.SetCoinbase()
			// update watcher
			a.addWatcher(op, entry, addrStr, balance, au)
		} else if scriptClass == txscript.CLTVPubKeyHashTy {
			au.SetCLTV()
			// update watcher
			a.addWatcher(op, entry, addrStr, balance, au)
		} else {
			au.FinalizeBalanceByAmount()
		}

		log.Trace(fmt.Sprintf("Add balance: %s (%s)", addrStr, au.String()))

		if a.isAllMode() {
			if !a.info.Has(addrStr) {
				a.info.Add(addrStr)
				infoChange = true
			}
		}
		if infoChange {
			err = a.db.Update(func(tx legacydb.Tx) error {
				return DBPutACCTInfo(tx, a.info)
			})
			if err != nil {
				return err
			}
		}
		return a.db.Update(func(tx legacydb.Tx) error {
			return DBPutACCTUTXO(tx, addrStr, op, au)
		})
	} else {
		err = a.db.Update(func(dbTx legacydb.Tx) error {
			balance, er := DBGetACCTBalance(dbTx, addrStr)
			if er != nil {
				return er
			}
			if balance == nil {
				a.DelWatcher(addrStr)
				return nil
			} else {
				amount := uint64(entry.Amount().Value)
				if entry.IsCoinBase() ||
					scriptClass == txscript.CLTVPubKeyHashTy {
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
			var au *AcctUTXO
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
				au, err = DBGetACCTUTXO(dbTx, addrStr, op)
				if er != nil {
					return er
				}
				er = DBDelACCTUTXO(dbTx, addrStr, op)
				if er != nil {
					return er
				}
			}
			if balance.locUTXONum <= 0 {
				a.DelWatcher(addrStr)
			} else {
				a.DelWatcherOP(addrStr, op, au)
			}
			return nil
		})
		return err
	}
}

func (a *AccountManager) addWatcher(op *types.TxOutPoint, entry *utxo.UtxoEntry, addrStr string, balance *AcctBalance, au *AcctUTXO) {
	opk := OutpointKey(op)
	uw := BuildUTXOWatcher(op, au, entry, a)
	if uw == nil {
		return
	}
	uw.Update(a)

	wb := a.getWatcher(addrStr)
	if wb == nil {
		wb = NewAcctBalanceWatcher(addrStr, balance)
		a.watchLock.Lock()
		a.watchers[addrStr] = wb
		a.watchLock.Unlock()
	}
	if uw.IsUnlocked() {
		wb.Unlock(uw)
	} else {
		wb.Add(opk, uw)
	}
}

func (a *AccountManager) hasWatcher(addr string) bool {
	a.watchLock.RLock()
	_, exist := a.watchers[addr]
	a.watchLock.RUnlock()
	return exist
}

func (a *AccountManager) getWatcher(addr string) *AcctBalanceWatcher {
	a.watchLock.RLock()
	wb, exist := a.watchers[addr]
	a.watchLock.RUnlock()
	if exist {
		return wb
	}
	return nil
}

func (a *AccountManager) DelWatcher(addr string) {
	a.watchLock.Lock()
	delete(a.watchers, addr)
	a.watchLock.Unlock()
}

func (a *AccountManager) DelWatcherOP(addr string, op *types.TxOutPoint, au *AcctUTXO) {
	a.watchLock.RLock()
	wb, exist := a.watchers[addr]
	a.watchLock.RUnlock()
	if !exist {
		return
	}
	opk := OutpointKey(op)
	if wb.Has(opk) {
		wb.Del(opk)
	} else {
		if au != nil && au.IsFinal() {
			if au.IsCoinbase() || au.IsCLTV() {
				wb.Drop(au)
			}
		}
	}
}

func (a *AccountManager) trackCfgAddresses(setinfo bool) error {
	addrs := a.cfg.AcctAddrs
	if len(addrs) <= 0 {
		return nil
	}
	ret := []string{}
	for _, addr := range addrs {
		err := a.checkAddress(addr)
		if err != nil {
			return err
		}
		if a.info.Has(addr) {
			log.Warn("Already exists", "addr", addr)
			continue
		}
		if a.hasWatcher(addr) {
			return fmt.Errorf("Already exists watcher:%s", addr)
		}
		a.info.Add(addr)
		if setinfo {
			continue
		}
		err = a.db.Update(func(dbTx legacydb.Tx) error {
			return a.cleanBalanceDB(dbTx, addr)
		})
		if err != nil {
			return err
		}
		ret = append(ret, addr)
	}
	if len(ret) <= 0 {
		return nil
	}
	return a.rebuild(ret)
}

func (a *AccountManager) initWatchers(dbTx legacydb.Tx) error {
	meta := dbTx.Metadata()
	balBucket := meta.Bucket(BalanceBucketName)
	if balBucket == nil {
		return nil
	}
	kus := [][]byte{}
	aus := []*AcctUTXO{}
	bas := []*AcctBalance{}
	ads := []string{}
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
			au := NewAcctUTXO(0)
			err := au.Decode(bytes.NewReader(vu))
			if err != nil {
				return err
			}
			if au.IsFinal() {
				return nil
			}
			addrStr := string(k)
			kus = append(kus, ku)
			aus = append(aus, au)
			bas = append(bas, balance)
			ads = append(ads, addrStr)
			return nil
		})
		return nil
	})
	if err != nil {
		return err
	}
	if len(aus) > 0 {
		for i := 0; i < len(aus); i++ {
			outpoint, err := parseOutpoint(kus[i])
			if err != nil {
				log.Error(err.Error())
				continue
			}
			entry, err := a.getEntry(outpoint)
			if err != nil {
				log.Error(err.Error())
				continue
			}
			a.addWatcher(outpoint, entry, ads[i], bas[i], aus[i])
		}
	}
	return nil
}

func (a *AccountManager) Apply(add bool, op *types.TxOutPoint, entry interface{}) error {
	if !a.cfg.AcctMode {
		return nil
	}
	a.utxoops = append(a.utxoops, &UTXOOP{add: add, op: op, entry: entry.(*utxo.UtxoEntry)})
	return nil
}

func (a *AccountManager) Commit(point model.Block) error {
	if !a.cfg.AcctMode {
		return nil
	}
	defer func() {
		a.utxoops = []*UTXOOP{}
	}()

	curDAGID := uint32(a.chain.BlockDAG().GetBlockTotal())
	a.info.updateDAGID = curDAGID
	err := a.db.Update(func(dbTx legacydb.Tx) error {
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
	a.watchLock.RLock()
	defer a.watchLock.RUnlock()
	if len(a.watchers) > 0 {
		for _, w := range a.watchers {
			err = w.Update(a)
			if err != nil {
				return err
			}
		}
	}
	a.statpoint = point
	return nil
}

func (a *AccountManager) GetBalance(addr string) (uint64, error) {
	err := a.checkAddress(addr)
	if err != nil {
		return 0, err
	}
	if !a.info.Has(addr) {
		return 0, fmt.Errorf("Please track this account:%s", addr)
	}
	result := uint64(0)

	wb := a.getWatcher(addr)
	if wb != nil {
		return wb.GetBalance(), nil
	}

	err = a.db.Update(func(dbTx legacydb.Tx) error {
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

func (a *AccountManager) GetUTXOs(addr string, limit *int, locked *bool, amount *uint64) ([]UTXOResult, uint64, error) {
	err := a.checkAddress(addr)
	if err != nil {
		return nil, 0, err
	}
	if !a.info.Has(addr) {
		return nil, 0, fmt.Errorf("Please track this account:%s", addr)
	}
	utxos := []UTXOResult{}
	totalAmount := uint64(0)
	err = a.db.Update(func(dbTx legacydb.Tx) error {
		us := DBGetACCTUTXOs(dbTx, addr)
		if len(us) > 0 {
			for k, v := range us {
				ur := UTXOResult{Type: v.TypeStr(), Amount: v.amount, Status: "valid"}
				if !v.IsFinal() {
					wb := a.getWatcher(addr)
					if wb != nil {
						wu := wb.GetByOPS(k)
						if wu != nil {
							if locked != nil && !(*locked) {
								continue
							}
							ur.Status = "locked"
						}
					}
				} else {
					ur.Amount = v.balance
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

				if limit != nil {
					if len(utxos) >= *limit {
						break
					}
				}
				totalAmount += ur.Amount

				if amount != nil {
					if totalAmount >= *amount {
						break
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return utxos, totalAmount, nil
}

func (a *AccountManager) HasUTXO(addr string, outPoint *types.TxOutPoint) bool {
	err := a.checkAddress(addr)
	if err != nil {
		return false
	}
	if !a.info.Has(addr) {
		return false
	}
	has := false
	err = a.db.Update(func(dbTx legacydb.Tx) error {
		us := DBGetACCTUTXOs(dbTx, addr)
		if len(us) > 0 {
			for k, v := range us {
				if !v.IsFinal() {
					continue
				}
				opk, err := hex.DecodeString(k)
				if err != nil {
					return err
				}
				op, err := parseOutpoint(opk)
				if err != nil {
					return err
				}
				if op.Hash.IsEqual(&outPoint.Hash) && op.OutIndex == outPoint.OutIndex {
					has = true
					return nil
				}
			}
		}
		return nil
	})
	if err != nil {
		return false
	}
	return has
}

func (a *AccountManager) AddAddress(addr string) error {
	if a.isAllMode() {
		return fmt.Errorf("Already exists:%s (Because of all mode)", addr)
	}
	err := a.checkAddress(addr)
	if err != nil {
		return err
	}
	if a.info.Has(addr) {
		return fmt.Errorf("Already exists:%s", addr)
	}
	if a.hasWatcher(addr) {
		return fmt.Errorf("Already exists watcher:%s", addr)
	}
	a.info.Add(addr)
	err = a.db.Update(func(dbTx legacydb.Tx) error {
		return a.cleanBalanceDB(dbTx, addr)
	})
	if err != nil {
		return err
	}
	return a.rebuild([]string{addr})
}

func (a *AccountManager) DelAddress(addr string) error {
	if a.isAllMode() {
		return fmt.Errorf("Prohibit deletion operation:%s (Because of all mode)", addr)
	}
	err := a.checkAddress(addr)
	if err != nil {
		return err
	}
	if !a.info.Has(addr) {
		return fmt.Errorf("Account does not exist:%s", addr)
	}
	a.DelWatcher(addr)
	a.info.Del(addr)
	return a.db.Update(func(dbTx legacydb.Tx) error {
		return a.cleanBalanceDB(dbTx, addr)
	})
}

func (a *AccountManager) GetChain() *blockchain.BlockChain {
	return a.chain
}

func (a *AccountManager) cleanBalanceDB(dbTx legacydb.Tx, addr string) error {
	del, er := DBDelACCTBalance(dbTx, addr)
	if er != nil {
		return er
	}
	if del {
		if a.info.total > 0 {
			a.info.total--
			er = DBPutACCTInfo(dbTx, a.info)
			if er != nil {
				return er
			}
		}
	}
	er = DBDelACCTUTXOs(dbTx, addr)
	if er != nil {
		return er
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

func (a *AccountManager) checkAddress(addr string) error {
	if !a.cfg.AcctMode {
		return fmt.Errorf("Please enable --acctmode")
	}
	if len(addr) <= 0 {
		return fmt.Errorf("The entered address cannot be empty")
	}
	if !address.IsForCurNetwork(addr) {
		return fmt.Errorf("network error:%s", addr)
	}
	return nil
}

func (a *AccountManager) getUtxoWatcherSize() int {
	if len(a.watchers) <= 0 {
		return 0
	}
	a.watchLock.RLock()
	defer a.watchLock.RUnlock()
	size := 0
	for _, w := range a.watchers {
		size += w.GetWatchersSize()
	}
	return size
}

func (a *AccountManager) isAllMode() bool {
	return a.info.all
}

func (a *AccountManager) getEntry(outpoint *types.TxOutPoint) (*utxo.UtxoEntry, error) {
	entry, err := utxo.DBFetchUtxoEntry(a.chain.Consensus().DatabaseContext(), *outpoint)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("No entry:%s", outpoint.Hash.String())
	}
	return entry, nil
}

func New(chain *blockchain.BlockChain, cfg *config.Config, _events *event.Feed) (*AccountManager, error) {
	a := AccountManager{
		chain:    chain,
		cfg:      cfg,
		info:     NewAcctInfo(),
		utxoops:  []*UTXOOP{},
		watchers: map[string]*AcctBalanceWatcher{},
		events:   _events,
	}
	return &a, nil
}
