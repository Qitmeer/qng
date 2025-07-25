package acct

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/consensus/forks"
	"github.com/Qitmeer/qng/v2/core/blockchain/utxo"
	"github.com/Qitmeer/qng/v2/core/event"
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/engine/txscript"
)

type AcctBalanceWatcher struct {
	address      string
	ab           *AcctBalance
	unlocked     uint64
	unlocUTXONum uint32

	watchers map[string]AcctUTXOIWatcher

	lock sync.RWMutex
}

func (aw *AcctBalanceWatcher) Add(op []byte, au AcctUTXOIWatcher) {
	if aw.Has(op) || au == nil {
		return
	}
	key := hex.EncodeToString(op)
	aw.lock.Lock()
	aw.watchers[key] = au
	aw.lock.Unlock()
	log.Trace(fmt.Sprintf("Balance (%s) add utxo watcher:%s %s", aw.address, key, au.GetName()))
}

func (aw *AcctBalanceWatcher) Del(op []byte) {
	aw.lock.Lock()
	defer aw.lock.Unlock()
	aw.del(op)
}

func (aw *AcctBalanceWatcher) del(op []byte) {
	key := hex.EncodeToString(op)
	delete(aw.watchers, key)
	log.Trace(fmt.Sprintf("Balance (%s) del utxo watcher:%s", aw.address, key))
}

func (aw *AcctBalanceWatcher) Has(op []byte) bool {
	ops := hex.EncodeToString(op)
	return aw.HasByOPS(ops)
}

func (aw *AcctBalanceWatcher) HasByOPS(ops string) bool {
	aw.lock.RLock()
	_, exist := aw.watchers[ops]
	aw.lock.RUnlock()
	return exist
}

func (aw *AcctBalanceWatcher) Get(op []byte) AcctUTXOIWatcher {
	ops := hex.EncodeToString(op)
	return aw.GetByOPS(ops)
}

func (aw *AcctBalanceWatcher) GetByOPS(ops string) AcctUTXOIWatcher {
	aw.lock.RLock()
	ret := aw.watchers[ops]
	aw.lock.RUnlock()
	return ret
}

func (aw *AcctBalanceWatcher) GetBalance() uint64 {
	return aw.ab.normal + aw.unlocked
}

func (aw *AcctBalanceWatcher) Unlock(uw AcctUTXOIWatcher) {
	aw.unlocked += uw.GetBalance()
	aw.unlocUTXONum++
}

func (aw *AcctBalanceWatcher) Drop(au *AcctUTXO) {
	if aw.unlocked > au.balance {
		aw.unlocked -= au.balance
	}
	if aw.unlocUTXONum > 0 {
		aw.unlocUTXONum--
	}
}

func (aw *AcctBalanceWatcher) Update(am *AccountManager) error {
	aw.lock.RLock()
	defer aw.lock.RUnlock()
	for k, w := range aw.watchers {
		err := w.Update(am)
		if err != nil {
			return err
		}
		if w.IsUnlocked() {
			aw.Unlock(w)
			delete(aw.watchers, k)
			log.Trace(fmt.Sprintf("Balance (%s) del utxo watcher:%s, because of final(%s)", aw.address, k, w.GetUTXO().String()))
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
			go am.events.Send(event.New(&types.AutoCollectUtxo{
				Op:      *op,
				Address: aw.address,
				Amount:  w.GetBalance(),
			}))
		}
	}
	return nil
}

func (aw *AcctBalanceWatcher) GetWatchersSize() int {
	aw.lock.RLock()
	defer aw.lock.RUnlock()
	return len(aw.watchers)
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
	GetUTXO() *AcctUTXO
}

func BuildUTXOWatcher(op *types.TxOutPoint, au *AcctUTXO, entry *utxo.UtxoEntry, am *AccountManager) AcctUTXOIWatcher {
	if entry.BlockHash().IsEqual(&hash.ZeroHash) {
		return nil
	}
	ib := am.chain.BlockDAG().GetBlock(entry.BlockHash())
	if ib == nil {
		return nil
	}
	if ib.GetState().GetStatus().KnownInvalid() {
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
		return NewCLTVWatcher(au, lockTime, forks.IsMaxLockUTXOInGenesis(op))
	}
	return nil
}
