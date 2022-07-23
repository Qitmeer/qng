package acct

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
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
	log.Trace(fmt.Sprintf("Balance (%s) add utxo watcher:%s", aw.address, key))
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

func (aw *AcctBalanceWatcher) GetBalance() uint64 {
	return aw.ab.normal + aw.unlocked
}

func (aw *AcctBalanceWatcher) Update(am *AccountManager) error {
	aw.unlocked = 0
	aw.unlocUTXONum = 0
	for _, w := range aw.watchers {
		err := w.Update(am)
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
}

func BuildUTXOWatcher(op []byte, au *AcctUTXO, entry *blockchain.UtxoEntry, am *AccountManager) AcctUTXOIWatcher {
	if entry == nil {
		txhash, err := hash.NewHash(op[:hash.HashSize])
		if err != nil {
			log.Error(err.Error())
			return nil
		}
		txOutIdex, size := serialization.DeserializeVLQ(op[hash.HashSize:])
		if size <= 0 {
			log.Error(fmt.Sprintf("DeserializeVLQ:%s %v", txhash.String(), op[hash.HashSize:]))
			return nil
		}
		err = am.chain.DB().View(func(dbTx database.Tx) error {
			entry, err = blockchain.DBFetchUtxoEntry(dbTx, *types.NewOutPoint(txhash, uint32(txOutIdex)))
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
	}
	return nil
}
