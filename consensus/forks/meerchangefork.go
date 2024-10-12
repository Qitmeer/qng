package forks

import (
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/params"
	"math"
)

const (
	// TODO:Future decision on whether to start
	// Support MeerChange system contract at evm height
	MeerChangeForkEvmHeight = math.MaxInt64
)

func IsMeerChangeForkHeight(height int64) bool {
	if params.ActiveNetParams.Net == protocol.PrivNet {
		return true
	}
	return height >= MeerChangeForkEvmHeight
}

func GetMeerChangeForkHeight() uint64 {
	if params.ActiveNetParams.Net == protocol.PrivNet {
		return 0
	}
	return MeerChangeForkEvmHeight
}
