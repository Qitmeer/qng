package forks

import (
	"github.com/Qitmeer/qng/common/math"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/params"
)

const (
	// TODO:Future decision on whether to start
	// Should we abolish the consensus restriction strategy when empty blocks appear
	EmptyBlockForkHeight = math.MaxInt64
)

func IsEmptyBlockForkHeight(mainHeight int64) bool {
	if params.ActiveNetParams.Net != protocol.MainNet {
		return true
	}
	return mainHeight >= EmptyBlockForkHeight
}
