package acct

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/services/acct"
)

type AcctBalanceWatcher struct {
	ab           *AcctBalance
	unlocked     uint64
	unlocUTXONum uint32

	watchers map[string]AcctUTXOIWatcher
}

func (aw *AcctBalanceWatcher) Add(op []byte, au AcctUTXOIWatcher) {
	if aw.Has(op) {
		return
	}
	aw.watchers[hex.EncodeToString(op)] = au
}

func (aw *AcctBalanceWatcher) Del(op []byte) {
	delete(aw.watchers, hex.EncodeToString(op))
}

func (aw *AcctBalanceWatcher) Has(op []byte) bool {
	ops := hex.EncodeToString(op)
	_, exist := aw.watchers[ops]
	return exist
}

func (aw *AcctBalanceWatcher) GetBalance() uint64 {
	return aw.ab.normal + aw.unlocked
}

func (aw *AcctBalanceWatcher) Update(am *AccountManager) error {
	aw.unlocked = 0
	aw.unlocUTXONum = 0
	for _, w := range aw.watchers {
		err := aw.Update(am)
		if err != nil {
			return err
		}
		aw.unlocked += w.GetBalance()
		if w.IsUnlocked() {
			aw.unlocUTXONum++
		}
	}
	return nil
}

func NewAcctBalanceWatcher(ab *AcctBalance) *AcctBalanceWatcher {
	return &AcctBalanceWatcher{
		ab:       ab,
		watchers: map[string]AcctUTXOIWatcher{},
	}
}

type AcctUTXOIWatcher interface {
	Update(am *AccountManager) error
	GetBalance() uint64
	IsUnlocked() bool
}

type CoinbaseWatcher struct {
	au       *AcctUTXO
	unlocked bool
	fee      uint64
	ib       meerdag.IBlock
}

func (cw *CoinbaseWatcher) Update(am *AccountManager) error {

}

func (cw *CoinbaseWatcher) GetBalance() uint64 {
	if cw.unlocked {
		return cw.au.balance + cw.fee
	}
	return 0
}

func (cw *CoinbaseWatcher) IsUnlocked() bool {
	return cw.unlocked
}
