package acct

import (
	"fmt"
	"github.com/Qitmeer/qng/v2/engine/txscript"
	"github.com/Qitmeer/qng/v2/params"
)

type CLTVWatcher struct {
	au            *AcctUTXO
	lockTime      int64
	isForkGenUTXO bool // MeerEVM fork
}

func (cw *CLTVWatcher) Update(am *AccountManager) error {
	if cw.IsUnlocked() {
		return nil
	}
	mainTip := am.chain.BlockDAG().GetMainChainTip()
	if mainTip == nil {
		return fmt.Errorf("No main tip")
	}
	if params.ActiveNetParams.IsMeerUTXOFork(int64(mainTip.GetHeight())) && cw.isForkGenUTXO {
		cw.au.FinalizeBalanceByAmount()
		return nil
	}
	lockTime := int64(0)
	if cw.lockTime < txscript.LockTimeThreshold {
		lockTime = int64(mainTip.GetHeight())
	} else {
		lockTime = am.chain.BlockDAG().GetBlockData(mainTip).GetTimestamp()
	}
	err := txscript.VerifyLockTime(lockTime, txscript.LockTimeThreshold, cw.lockTime)
	if err != nil {
		return nil
	}

	cw.au.FinalizeBalanceByAmount()
	return nil
}

func (cw *CLTVWatcher) GetBalance() uint64 {
	return cw.au.balance
}

func (cw *CLTVWatcher) Lock() {
	cw.au.Lock()
}

func (cw *CLTVWatcher) IsUnlocked() bool {
	return cw.au.IsFinal()
}

func (cw *CLTVWatcher) GetName() string {
	return cw.au.TypeStr()
}

func (cw *CLTVWatcher) GetUTXO() *AcctUTXO {
	return cw.au
}

func NewCLTVWatcher(au *AcctUTXO, lockTime int64, isForkGenUTXO bool) *CLTVWatcher {
	cw := &CLTVWatcher{au: au, lockTime: lockTime, isForkGenUTXO: isForkGenUTXO}
	return cw
}
