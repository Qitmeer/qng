package blockchain

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	blockProcessTimer  = metrics.NewRegisteredTimer("blockchain/process", nil)
)
