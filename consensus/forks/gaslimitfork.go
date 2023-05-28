package forks

import (
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/params"
)

const (
	// New gas limit for evm block
	GasLimitForkEVMHeight = 606567
)

func IsGasLimitForkHeight(number int64) bool {
	if params.ActiveNetParams.Net != protocol.MainNet {
		return false
	}
	return number >= GasLimitForkEVMHeight
}
