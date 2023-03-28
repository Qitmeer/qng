package acct

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/engine/txscript"
)

type AcctBalanceWatcher struct {
	address      string
	ab           *AcctBalance
	unlocked     uint64
	unlocUTXONum uint32

	watchers map[string]AcctUTXOIWatcher
}

func (aw *AcctBalanceWatcher) Add(op []byte, au AcctUTXOIWatcher) {
	if aw.Has(op) || au == nil {
		return
	}
	key := hex.EncodeToString(op)
	aw.watchers[key] = au
	log.Trace(fmt.Sprintf("Balance (%s) add utxo watcher:%s %s", aw.address, key, au.GetName()))
}

func (aw *AcctBalanceWatcher) Del(op []byte) {
	key := hex.EncodeToString(op)
	delete(aw.watchers, key)
	log.Trace(fmt.Sprintf("Balance (%s) del utxo watcher:%s", aw.address, key))
}

func (aw *AcctBalanceWatcher) Has(op []byte) bool {
	ops := hex.EncodeToString(op)
	_, exist := aw.watchers[ops]
	return exist
}

func (aw *AcctBalanceWatcher) Get(op []byte) AcctUTXOIWatcher {
	ops := hex.EncodeToString(op)
	return aw.GetByOPS(ops)
}

func (aw *AcctBalanceWatcher) GetByOPS(ops string) AcctUTXOIWatcher {
	return aw.watchers[ops]
}

func (aw *AcctBalanceWatcher) GetBalance() uint64 {
	return aw.ab.normal + aw.unlocked
}

func (aw *AcctBalanceWatcher) Update(am *AccountManager) error {
	aw.unlocked = 0
	aw.unlocUTXONum = 0
	for k, w := range aw.watchers {
		err := w.Update(am)
		if err != nil {
			return err
		}
		aw.unlocked += w.GetBalance()
		if w.IsUnlocked() {
			aw.unlocUTXONum++
		}
		if am.cfg.AutoCollectEvm && w.IsUnlocked() {
			opk, err := hex.DecodeString(k)
			if err != nil {
				return err
			}
			op, err := parseOutpoint(opk)
			if err != nil {
				return err
			}
			am.autoCollectUtxo <- types.AutoCollectUtxo{
				Op:      *op,
				Address: aw.address,
				Amount:  w.GetBalance(),
			}
		}
	}
	return nil
}

func NewAcctBalanceWatcher(address string, ab *AcctBalance) *AcctBalanceWatcher {
	return &AcctBalanceWatcher{
		address:  address,
		ab:       ab,
		watchers: map[string]AcctUTXOIWatcher{},
	}
}

type AcctUTXOIWatcher interface {
	Update(am *AccountManager) error
	GetBalance() uint64
	IsUnlocked() bool
	GetName() string
}

func BuildUTXOWatcher(op []byte, au *AcctUTXO, entry *utxo.UtxoEntry, am *AccountManager) AcctUTXOIWatcher {
	if entry == nil {
		outpoint, err := parseOutpoint(op)
		if err != nil {
			return nil
		}
		err = am.chain.DB().View(func(dbTx database.Tx) error {
			entry, err = utxo.DBFetchUtxoEntry(dbTx, *outpoint)
			return err
		})
		if err != nil {
			log.Error(err.Error())
			return nil
		}
		if entry == nil {
			return nil
		}
	}
	if entry.BlockHash().IsEqual(&hash.ZeroHash) {
		return nil
	}
	ib := am.chain.BlockDAG().GetBlock(entry.BlockHash())
	if ib == nil {
		return nil
	}
	if ib.GetStatus().KnownInvalid() {
		return nil
	}
	if au.IsCoinbase() {
		return NewCoinbaseWatcher(au, ib)
	} else if au.IsCLTV() {
		ops, err := txscript.ParseScript(entry.PkScript())
		if err != nil {
			log.Error(err.Error())
			return nil
		}
		if len(ops) < 2 {
			return nil
		}
		lockTime := txscript.GetInt64FromOpcode(ops[0])
		outpoint, err := parseOutpoint(op)
		if err != nil {
			return nil
		}
		return NewCLTVWatcher(au, lockTime, forks.IsMaxLockUTXOInGenesis(outpoint))
	}
	return nil
}
