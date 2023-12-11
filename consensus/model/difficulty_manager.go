package model

import (
	"math/big"
	"time"

	"github.com/Qitmeer/qng/core/types/pow"
)

// DifficultyManager provides a method to resolve the
// difficulty value of a block
type DifficultyManager interface {
	CalcNextRequiredDifficulty(timestamp time.Time, powType pow.PowType) (uint32, error)
	RequiredDifficulty(block Block, newBlockTime time.Time, powInstance pow.IPow) (uint32, error)
	CalcEasiestDifficulty(bits uint32, duration time.Duration, powInstance pow.IPow) uint32
	GetCurrentPowDiff(ib Block, powType pow.PowType) *big.Int
}
