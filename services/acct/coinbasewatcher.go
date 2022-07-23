package acct

import (
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
)

type CoinbaseWatcher struct {
	au             *AcctUTXO
	unlocked       bool
	fee            uint64
	target         meerdag.IBlock
	targetMainFork meerdag.IBlock
}

func (cw *CoinbaseWatcher) Update(am *AccountManager) error {
	if cw.target == nil {
		return nil
	}
	if cw.IsUnlocked() {
		return nil
	}
	var ret bool
	ret, cw.targetMainFork = am.chain.BlockDAG().CheckMainBlueAndMature(cw.target, cw.targetMainFork, uint(params.ActiveNetParams.CoinbaseMaturity))
	if !ret {
		return nil
	}
	cw.unlocked = true

	if !cw.target.GetHash().IsEqual(params.ActiveNetParams.GenesisHash) {
		cw.fee = uint64(am.chain.GetFeeByCoinID(cw.target.GetHash(), types.MEERID))
	}
	return nil
}

func (cw *CoinbaseWatcher) GetBalance() uint64 {
	if cw.unlocked {
		return cw.au.balance + cw.fee
	}
	return 0
}

func (cw *CoinbaseWatcher) Lock() {
	cw.unlocked = false
	cw.fee = 0
}

func (cw *CoinbaseWatcher) IsUnlocked() bool {
	return cw.unlocked
}

func NewCoinbaseWatcher(au *AcctUTXO, target meerdag.IBlock) *CoinbaseWatcher {
	cw := &CoinbaseWatcher{au: au, target: target}
	return cw
}
