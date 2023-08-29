package miner

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	totalGbtRequests    = metrics.NewRegisteredGauge("mining/totalGbtRequests", nil)
	totalGbts           = metrics.NewRegisteredGauge("mining/totalGbts", nil)
	totalSubmits        = metrics.NewRegisteredGauge("mining/totalSubmits", nil)
	totalTxEmptySubmits = metrics.NewRegisteredGauge("mining/totalTxEmptySubmits", nil)

	submitDuration       = metrics.NewRegisteredTimer("mining/submitDuration", nil)
	submitTxCount        = metrics.NewRegisteredGauge("mining/submitTxCount", nil)
	gbtRequestDuration   = metrics.NewRegisteredTimer("mining/gbtRequestDuration", nil)
	gbtDuration          = metrics.NewRegisteredTimer("mining/gbtDuration", nil)
	mempoolEmptyDuration = metrics.NewRegisteredTimer("mining/mempoolEmptyDuration", nil)
)
