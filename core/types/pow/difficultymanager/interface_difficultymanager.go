package difficultymanager

import (
	"math/big"
	"time"

	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/params"
)

// DifficultyManager provides a method to resolve the
// difficulty value of a block
type DifficultyManager interface {
	CalcNextRequiredDifficulty(timestamp time.Time, powType pow.PowType) (uint32, error)
	RequiredDifficulty(block model.Block, newBlockTime time.Time, powInstance pow.IPow) (uint32, error)
	CalcEasiestDifficulty(bits uint32, duration time.Duration, powInstance pow.IPow) uint32
	GetCurrentPowDiff(ib model.Block, powType pow.PowType) *big.Int
}

func NewDiffManager(b model.BlockChain, cfg *params.Params) DifficultyManager {
	switch cfg.PowConfig.DifficultyMode {
	case types.DIFFICULTY_MODE_KASPAD:
		return &kaspadDiff{
			b:                              b,
			powMax:                         cfg.PowConfig.MeerXKeccakV1PowLimit,
			difficultyAdjustmentWindowSize: int(cfg.WorkDiffWindowSize),
			disableDifficultyAdjustment:    false,
			targetTimePerBlock:             cfg.TargetTimePerBlock,
			genesisBits:                    cfg.PowConfig.MeerXKeccakV1PowLimitBits,
			cfg:                            cfg,
		}
	}
	return &meerDiff{
		b:   b,
		cfg: cfg,
	}
}
