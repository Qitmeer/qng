package acct

import (
	"fmt"
	"github.com/Qitmeer/qng/engine/txscript"
)

type CLTVWatcher struct {
	au       *AcctUTXO
	unlocked bool
	lockTime int64
}

func (cw *CLTVWatcher) Update(am *AccountManager) error {
	if cw.IsUnlocked() {
		return nil
	}
	mainTip := am.chain.BlockDAG().GetMainChainTip()
	if mainTip == nil {
		return fmt.Errorf("No main tip")
	}
	lockTime := int64(0)
	if cw.lockTime < txscript.LockTimeThreshold {
		lockTime = int64(mainTip.GetHeight())
	} else {
		lockTime = mainTip.GetData().GetTimestamp()
	}
	err := txscript.VerifyLockTime(lockTime, txscript.LockTimeThreshold, cw.lockTime)
	if err != nil {
		return nil
	}

	cw.unlocked = true
	return nil
}

func (cw *CLTVWatcher) GetBalance() uint64 {
	if cw.unlocked {
		return cw.au.balance
	}
	return 0
}

func (cw *CLTVWatcher) Lock() {
	cw.unlocked = false
}

func (cw *CLTVWatcher) IsUnlocked() bool {
	return cw.unlocked
}

func (cw *CLTVWatcher) GetName() string {
	return cw.au.TypeStr()
}

func NewCLTVWatcher(au *AcctUTXO, lockTime int64) *CLTVWatcher {
	fmt.Println(au.String(), " lockTime=", lockTime)
	cw := &CLTVWatcher{au: au, lockTime: lockTime}
	return cw
}
