package difficultymanager

import (
	"math/big"
	"time"

	"github.com/Qitmeer/qng/common/util/math"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/params"
)

// DifficultyManager provides a method to resolve the
// difficulty value of a block
type difficultyManager struct {
	powMax                         *big.Int
	difficultyAdjustmentWindowSize int
	disableDifficultyAdjustment    bool
	targetTimePerBlock             time.Duration
	genesisBits                    uint32
}

// New instantiates a new DifficultyManager
func New(cfg *params.Params) model.DifficultyManager {
	return &difficultyManager{
		powMax:                         cfg.PowConfig.MeerXKeccakV1PowLimit,
		difficultyAdjustmentWindowSize: int(cfg.WorkDiffWindowSize),
		disableDifficultyAdjustment:    false,
		targetTimePerBlock:             cfg.TargetTimePerBlock,
		genesisBits:                    cfg.PowConfig.MeerXKeccakV1PowLimitBits,
	}
}

// RequiredDifficulty returns the difficulty required for some block
func (dm *difficultyManager) RequiredDifficulty(targetsWindow model.BlockWindow, powInstance pow.IPow) (uint32, error) {
	if powInstance.GetPowType() != pow.MEERXKECCAKV1 || len(targetsWindow) < 1 {
		return dm.genesisBits, nil
	}
	return dm.requiredDifficultyFromTargetsWindow(targetsWindow)
}

func (dm *difficultyManager) requiredDifficultyFromTargetsWindow(targetsWindow model.BlockWindow) (uint32, error) {
	if dm.disableDifficultyAdjustment {
		return dm.genesisBits, nil
	}

	// in the past this was < 2 as the comment explains, we changed it to under the window size to
	// make the hashrate(which is ~1.5GH/s) constant in the first 2641 blocks so that we won't have a lot of tips

	// We need at least 2 blocks to get a timestamp interval
	// We could instead clamp the timestamp difference to `targetTimePerBlock`,
	// but then everything will cancel out and we'll get the target from the last block, which will be the same as genesis.
	// We add 64 as a safety margin
	if len(targetsWindow) < 2 || len(targetsWindow) < dm.difficultyAdjustmentWindowSize {
		return dm.genesisBits, nil
	}

	windowMinTimestamp, windowMaxTimeStamp, windowMinIndex := targetsWindow.MinMaxTimestamps()
	// Remove the last block from the window so to calculate the average target of dag.difficultyAdjustmentWindowSize blocks
	targetsWindow.Remove(windowMinIndex)

	// Calculate new target difficulty as:
	// averageWindowTarget * (windowMinTimestamp / (targetTimePerBlock * windowSize))
	// The result uses integer division which means it will be slightly
	// rounded down.
	div := new(big.Int)
	newTarget := targetsWindow.AverageTarget()
	newTarget.
		// We need to clamp the timestamp difference to 1 so that we'll never get a 0 target.
		Mul(newTarget, div.SetInt64(math.MaxInt64(windowMaxTimeStamp-windowMinTimestamp, 1))).
		Div(newTarget, div.SetInt64(dm.targetTimePerBlock.Milliseconds())).
		Div(newTarget, div.SetUint64(uint64(len(targetsWindow))))
	if newTarget.Cmp(dm.powMax) > 0 {
		return pow.BigToCompact(dm.powMax), nil
	}
	newTargetBits := pow.BigToCompact(newTarget)
	return newTargetBits, nil
}
