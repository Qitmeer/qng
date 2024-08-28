package forks

import (
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	"math"
	"math/big"
)

const (
	// TODO:Future decision on whether to start
	// Support cancun height
	CancunForkEvmHeight = math.MaxInt64
)

func IsCancunForkHeight(height int64) bool {
	if params.ActiveNetParams.Net == protocol.PrivNet {
		return true
	}
	return height >= CancunForkEvmHeight
}

func GetCancunForkDifficulty(height int64) *big.Int {
	if IsCancunForkHeight(height) {
		return common.Big0
	}
	return common.Big1
}
