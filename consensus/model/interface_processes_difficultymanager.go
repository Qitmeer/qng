package model

import (
	"math"
	"math/big"

	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types/pow"
)

type DifficultyBlock struct {
	TimeInMilliseconds int64
	Bits               uint32
	Hash               hash.Hash
	BlueWork           bool
}
type BlockWindow []DifficultyBlock

func ghostdagLess(blockA *DifficultyBlock, blockB *DifficultyBlock) bool {
	return blockA.BlueWork == blockB.BlueWork
}

func (window BlockWindow) MinMaxTimestamps() (min, max int64, minIndex int) {
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

func (window *BlockWindow) Remove(n int) {
	(*window)[n] = (*window)[len(*window)-1]
	*window = (*window)[:len(*window)-1]
}

func (window BlockWindow) AverageTarget() *big.Int {
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
type DifficultyManager interface {
	RequiredDifficulty(blocks BlockWindow, powInstance pow.IPow) (uint32, error)
}
