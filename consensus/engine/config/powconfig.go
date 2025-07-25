package config

import (
	"github.com/Qitmeer/qng/v2/consensus/engine"
	"github.com/Qitmeer/qng/v2/consensus/engine/pow"
	"time"
)

type PoWConfig struct {
	// PowConfig defines the highest allowed proof of work value for a block or lowest difficulty for a block
	PowConfig *pow.PowConfig
	// WorkDiffAlpha is the stake difficulty EMA calculation alpha (smoothing)
	// value. It is different from a normal EMA alpha. Closer to 1 --> smoother.
	WorkDiffAlpha int64

	// WorkDiffWindowSize is the number of windows (intervals) used for calculation
	// of the exponentially weighted average.
	WorkDiffWindowSize int64

	// WorkDiffWindows is the number of windows (intervals) used for calculation
	// of the exponentially weighted average.
	WorkDiffWindows int64

	// RetargetAdjustmentFactor is the adjustment factor used to limit
	// the minimum and maximum amount of adjustment that can occur between
	// difficulty retargets.
	RetargetAdjustmentFactor int64

	// ReduceMinDifficulty defines whether the network should reduce the
	// minimum required difficulty after a long enough period of time has
	// passed without finding a block.  This is really only useful for test
	// networks and should not be set on a main network.
	ReduceMinDifficulty bool

	// MinDiffReductionTime is the amount of time after which the minimum
	// required difficulty should be reduced when a block hasn't been found.
	//
	// NOTE: This only applies if ReduceMinDifficulty is true.
	MinDiffReductionTime time.Duration
}

func (c *PoWConfig) Type() engine.EngineType {
	return engine.PoWEngineType
}

func (c *PoWConfig) Check() error {
	return c.PowConfig.Check()
}
