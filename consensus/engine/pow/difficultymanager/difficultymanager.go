package difficultymanager

import (
	"github.com/Qitmeer/qng/v2/consensus/engine/pow"
	"github.com/Qitmeer/qng/v2/consensus/model"
	"github.com/Qitmeer/qng/v2/params"
)

func NewDiffManager(con model.Consensus, cfg *params.Params) model.DifficultyManager {
	switch cfg.ToPoWConfig().PowConfig.DifficultyMode {
	case pow.DIFFICULTY_MODE_GHOSTDAG:
		return &ghostdagDiff{
			con:                            con,
			b:                              con.BlockChain(),
			powMax:                         cfg.ToPoWConfig().PowConfig.MeerXKeccakV1PowLimit,
			difficultyAdjustmentWindowSize: int(cfg.ToPoWConfig().WorkDiffWindowSize),
			disableDifficultyAdjustment:    false,
			targetTimePerBlock:             cfg.TargetTimePerBlock,
			genesisBits:                    cfg.ToPoWConfig().PowConfig.MeerXKeccakV1PowLimitBits,
			cfg:                            cfg,
		}
	case pow.DIFFICULTY_MODE_DEVELOP:
		return &developDiff{
			b:   con.BlockChain(),
			cfg: cfg,
		}
	}
	return &meerDiff{
		con: con,
		b:   con.BlockChain(),
		cfg: cfg,
	}
}
