package blockchain

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	blockProcessTimer           = metrics.NewRegisteredTimer("blockchain/process", nil)
	blockConnectedNotifications = metrics.NewRegisteredTimer("blockchain/process/connectnotifications", nil)

	duplicateTxsGauge = metrics.NewRegisteredGauge("blockchain/duplicatetxs", nil)
)
