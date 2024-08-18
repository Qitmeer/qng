package forks

import (
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/params"
	"math"
)

const (
	// TODO:Future decision on whether to start
	// Support MeerChange system contract at height
	MeerChangeForkHeight = math.MaxInt64
)

func IsMeerChangeForkHeight(mainHeight int64) bool {
	if params.ActiveNetParams.Net != protocol.MainNet {
		return true
	}
	return mainHeight >= MeerChangeForkHeight
}
