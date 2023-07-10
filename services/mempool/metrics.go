package mempool

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	mempoolMaybeAcceptTransaction   = metrics.NewRegisteredTimer("mempool/maybeAcceptTransaction", nil)
	mempoolCheckTransactionSanity   = metrics.NewRegisteredTimer("mempool/maybeAcceptTransaction/checkTransactionSanity", nil)
	mempoolCheckTransactionStandard = metrics.NewRegisteredTimer("mempool/maybeAcceptTransaction/checkTransactionStandard", nil)
	mempoolCheckPoolDoubleSpend     = metrics.NewRegisteredTimer("mempool/maybeAcceptTransaction/checkPoolDoubleSpends", nil)
	mempoolLookSpent                = metrics.NewRegisteredTimer("mempool/maybeAcceptTransaction/lookSpent", nil)
	mempoolCheckTransactionInputs   = metrics.NewRegisteredTimer("mempool/maybeAcceptTransaction/checkTransactionInputs", nil)
	mempoolCountSigOps              = metrics.NewRegisteredTimer("mempool/maybeAcceptTransaction/countSigOps", nil)
	mempoolHaveTransaction          = metrics.NewRegisteredTimer("mempool/haveTransaction", nil)
)
