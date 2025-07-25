package acct

import (
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/meerdag"
	"github.com/Qitmeer/qng/v2/params"
)

type CoinbaseWatcher struct {
	au             *AcctUTXO
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
	amount := cw.au.amount
	if !cw.target.GetHash().IsEqual(params.ActiveNetParams.GenesisHash) {
		fee := uint64(am.chain.GetFeeByCoinID(cw.target.GetHash(), types.MEERA))
		amount += fee
	}
	cw.au.FinalizeBalance(amount)
	return nil
}

func (cw *CoinbaseWatcher) GetBalance() uint64 {
	return cw.au.balance
}

func (cw *CoinbaseWatcher) Lock() {
	cw.au.Lock()
}

func (cw *CoinbaseWatcher) IsUnlocked() bool {
	return cw.au.IsFinal()
}

func (cw *CoinbaseWatcher) GetName() string {
	return cw.au.TypeStr()
}

func (cw *CoinbaseWatcher) GetUTXO() *AcctUTXO {
	return cw.au
}

func NewCoinbaseWatcher(au *AcctUTXO, target meerdag.IBlock) *CoinbaseWatcher {
	cw := &CoinbaseWatcher{au: au, target: target}
	return cw
}
