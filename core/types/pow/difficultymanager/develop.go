// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package difficultymanager

import (
	"math/big"
	"time"

	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/params"
)

type developDiff struct {
	b   model.BlockChain
	cfg *params.Params
}

// CalcEasiestDifficulty calculates the easiest possible difficulty that a block
// can have given starting difficulty bits and a duration.  It is mainly used to
// verify that claimed proof of work by a block is sane as compared to a
// known good checkpoint.
func (m *developDiff) CalcEasiestDifficulty(bits uint32, duration time.Duration, powInstance pow.IPow) uint32 {
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

// findPrevTestNetDifficulty returns the difficulty of the previous block which
// did not have the special testnet minimum difficulty rule applied.
//
// This function MUST be called with the chain state lock held (for writes).
func (m *developDiff) findPrevTestNetDifficulty(startBlock model.Block, powInstance pow.IPow) uint32 {
	// Search backwards through the chain for the last block without
	// the special rule applied.
	target := powInstance.GetSafeDiff(0)
	lastBits := pow.BigToCompact(target)
	blocksPerRetarget := uint64(m.cfg.WorkDiffWindowSize * m.cfg.WorkDiffWindows)
	iterBlock := startBlock
	if iterBlock == nil ||
		uint64(iterBlock.GetHeight())%blocksPerRetarget == 0 {
		return lastBits
	}
	var iterNode *types.BlockHeader
	iterNode = m.b.GetBlockHeader(iterBlock)
	if iterNode.Difficulty != pow.BigToCompact(target) {
		return lastBits
	}
	return iterNode.Difficulty
}

// RequiredDifficulty calculates the required difficulty for the block
// after the passed previous block node based on the difficulty retarget rules.
// This function differs from the exported RequiredDifficulty in that
// the exported version uses the current best chain as the previous block node
// while this function accepts any block node.
func (m *developDiff) RequiredDifficulty(block model.Block, newBlockTime time.Time, powInstance pow.IPow) (uint32, error) {
	baseTarget := powInstance.GetSafeDiff(0)
	return pow.BigToCompact(baseTarget), nil
}

// find block node by pow type
func (m *developDiff) GetCurrentPowDiff(ib model.Block, powType pow.PowType) *big.Int {
	instance := pow.GetInstance(powType, 0, []byte{})
	instance.SetParams(m.cfg.PowConfig)
	safeBigDiff := instance.GetSafeDiff(0)
	return safeBigDiff
}
