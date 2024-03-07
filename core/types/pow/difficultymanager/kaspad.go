package difficultymanager

import (
	"math"
	"math/big"
	"time"

	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/params"
)

type DifficultyBlock struct {
	TimeInMilliseconds int64
	Bits               uint32
	Hash               hash.Hash
	BlueWork           bool
}
type blockWindow []DifficultyBlock

func ghostdagLess(blockA *DifficultyBlock, blockB *DifficultyBlock) bool {
	return blockA.BlueWork == blockB.BlueWork
}

func (window blockWindow) MinMaxTimestamps() (min, max int64, minIndex int) {
	min = math.MaxInt64
	minIndex = 0
	max = 0
	for i, block := range window {
		// If timestamps are equal we ghostdag compare in order to reach consensus on `minIndex`
		if block.TimeInMilliseconds < min ||
			(block.TimeInMilliseconds == min && ghostdagLess(&block, &window[minIndex])) {
			min = block.TimeInMilliseconds
			minIndex = i
		}
		if block.TimeInMilliseconds > max {
			max = block.TimeInMilliseconds
		}
	}
	return
}

func (window *blockWindow) Remove(n int) {
	(*window)[n] = (*window)[len(*window)-1]
	*window = (*window)[:len(*window)-1]
}

func (window blockWindow) AverageTarget() *big.Int {
	averageTarget := new(big.Int)
	targetTmp := new(big.Int)
	for _, block := range window {
		pow.CompactToBigWithDestination(block.Bits, targetTmp)
		averageTarget.Add(averageTarget, targetTmp)
	}
	return averageTarget.Div(averageTarget, big.NewInt(int64(len(window))))
}

// DifficultyManager provides a method to resolve the
// difficulty value of a block
type kaspadDiff struct {
	powMax                         *big.Int
	difficultyAdjustmentWindowSize int
	disableDifficultyAdjustment    bool
	targetTimePerBlock             time.Duration
	genesisBits                    uint32
	b                              model.BlockChain
	cfg                            *params.Params
	con                            model.Consensus
}

// CalcEasiestDifficulty calculates the easiest possible difficulty that a block
// can have given starting difficulty bits and a duration.  It is mainly used to
// verify that claimed proof of work by a block is sane as compared to a
// known good checkpoint.
func (m *kaspadDiff) CalcEasiestDifficulty(bits uint32, duration time.Duration, powInstance pow.IPow) uint32 {
	// Convert types used in the calculations below.
	durationVal := int64(duration)
	adjustmentFactor := big.NewInt(m.cfg.RetargetAdjustmentFactor)
	maxRetargetTimespan := int64(m.cfg.TargetTimespan) *
		m.cfg.RetargetAdjustmentFactor
	target := powInstance.GetSafeDiff(0)
	// The test network rules allow minimum difficulty blocks once too much
	// time has elapsed without mining a block.
	if m.cfg.ReduceMinDifficulty {
		if durationVal > int64(m.cfg.MinDiffReductionTime) {
			return pow.BigToCompact(target)
		}
	}

	// Since easier difficulty equates to higher numbers, the easiest
	// difficulty for a given duration is the largest value possible given
	// the number of retargets for the duration and starting difficulty
	// multiplied by the max adjustment factor.
	newTarget := pow.CompactToBig(bits)

	for durationVal > 0 && powInstance.CompareDiff(newTarget, target) {
		newTarget.Mul(newTarget, adjustmentFactor)
		newTarget = powInstance.GetNextDiffBig(adjustmentFactor, newTarget, big.NewInt(0))
		durationVal -= maxRetargetTimespan
	}

	// Limit new value to the proof of work limit.
	if !powInstance.CompareDiff(newTarget, target) {
		newTarget.Set(target)
	}

	return pow.BigToCompact(newTarget)
}

func (m *kaspadDiff) RequiredDifficulty(block model.Block, newBlockTime time.Time, powInstance pow.IPow) (uint32, error) {
	return m.RequiredDifficultyByWindows(m.getblockWindows(block, powInstance.GetPowType(), int(m.cfg.WorkDiffWindowSize)))
}

// RequiredDifficultyByWindows returns the difficulty required for some block
func (dm *kaspadDiff) RequiredDifficultyByWindows(targetsWindow blockWindow) (uint32, error) {
	if len(targetsWindow) < 1 {
		return dm.genesisBits, nil
	}
	return dm.requiredDifficultyFromTargetsWindow(targetsWindow)
}

func (dm *kaspadDiff) requiredDifficultyFromTargetsWindow(targetsWindow blockWindow) (uint32, error) {
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
		Mul(newTarget, div.SetInt64(int64(math.Max(float64(windowMaxTimeStamp-windowMinTimestamp), 1)))).
		Div(newTarget, div.SetInt64(dm.targetTimePerBlock.Milliseconds())).
		Div(newTarget, div.SetUint64(uint64(len(targetsWindow))))
	if newTarget.Cmp(dm.powMax) > 0 {
		return pow.BigToCompact(dm.powMax), nil
	}
	newTargetBits := pow.BigToCompact(newTarget)
	return newTargetBits, nil
}

// blockWindow returns a blockWindow of the given size that contains the
// blocks in the past of startingNode, the sorting is unspecified.
// If the number of blocks in the past of startingNode is less then windowSize,
// the window will be padded by genesis blocks to achieve a size of windowSize.
func (dm *kaspadDiff) getblockWindows(oldBlock model.Block, powType pow.PowType, windowSize int) blockWindow {
	windows := make(blockWindow, 0, windowSize)
	dm.b.ForeachBlueBlocks(oldBlock, uint(windowSize), powType, func(block model.Block, header *types.BlockHeader) error {
		windows = append(windows, DifficultyBlock{
			TimeInMilliseconds: header.Timestamp.UnixMilli(),
			Bits:               header.Difficulty,
			Hash:               header.BlockHash(),
			BlueWork:           true,
		})
		return nil
	})

	return windows
}

// find block node by pow type
func (m *kaspadDiff) GetCurrentPowDiff(ib model.Block, powType pow.PowType) *big.Int {
	instance := pow.GetInstance(powType, 0, []byte{})
	instance.SetParams(m.cfg.PowConfig)
	safeBigDiff := instance.GetSafeDiff(0)
	for {
		curNode := m.b.GetBlockHeader(ib)
		if curNode == nil {
			return safeBigDiff
		}
		if curNode.Pow.GetPowType() == powType {
			return pow.CompactToBig(curNode.Difficulty)
		}

		if !ib.HasParents() {
			return safeBigDiff
		}

		ib = m.b.GetBlockById(ib.GetMainParent())
		if ib == nil {
			return safeBigDiff
		}
	}
}
