package difficultymanager

import (
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/params"
)

func NewDiffManager(b model.BlockChain, cfg *params.Params) model.DifficultyManager {
	switch cfg.PowConfig.DifficultyMode {
	case pow.DIFFICULTY_MODE_KASPAD:
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
