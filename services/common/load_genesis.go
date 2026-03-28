// Copyright (c) 2017-2025 The qitmeer developers

package common

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Qitmeer/qng/common/util"
	"github.com/Qitmeer/qng/config"
	cfgengine "github.com/Qitmeer/qng/consensus/engine/config"
	ptypes "github.com/Qitmeer/qng/consensus/engine/poa/types"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	eparams "github.com/ethereum/go-ethereum/params"
)

type CustomGenesis struct {
	Name    string `json:"name"`
	ChainID uint64 `json:"chain_id"`
	PoA     struct {
		Period uint64 `json:"period"`
		Epoch  uint64 `json:"epoch"`
	} `json:"poa"`
	TargetTimePerBlock int                 `json:"target_time_per_block"`
	BaseSubsidy        int64               `json:"base_subsidy"`
	CoinbaseMaturity   uint16              `json:"coinbase_maturity"`
	Timestamp          string              `json:"timestamp"`
	Signers            []string            `json:"signers"`
	Alloc              etypes.GenesisAlloc `json:"alloc"`
}

func applyCustomGenesis(cfg *config.Config) error {
	path := util.CleanAndExpandPath(cfg.Genesis)
	if !filepath.IsAbs(path) {
		path = filepath.Join(cfg.HomeDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("applyCustomGenesis: read %s: %w", path, err)
	}

	var f CustomGenesis
	if err := json.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("applyCustomGenesis: parse json: %w", err)
	}
	if len(f.Signers) == 0 {
		return fmt.Errorf("applyCustomGenesis: signers must not be empty")
	}

	signers := make([]common.Address, 0, len(f.Signers))
	for _, s := range f.Signers {
		if !common.IsHexAddress(s) {
			return fmt.Errorf("applyCustomGenesis: invalid signer address %q", s)
		}
		signers = append(signers, common.HexToAddress(s))
	}

	ts, err := strconv.ParseInt(f.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("applyCustomGenesis: timestamp: %w", err)
	}

	poaCfg, ok := params.AmanaNetParams.ConsensusConfig.(*cfgengine.PoAConfig)
	if !ok {
		return fmt.Errorf("applyCustomGenesis: Amana consensus is not PoA")
	}
	if f.PoA.Period > 0 {
		poaCfg.Period = f.PoA.Period
	} else if f.TargetTimePerBlock > 0 {
		poaCfg.Period = uint64(f.TargetTimePerBlock)
	}
	if f.PoA.Epoch > 0 {
		poaCfg.Epoch = f.PoA.Epoch
	}

	if f.Name != "" {
		params.AmanaNetParams.Name = f.Name
	}
	if f.BaseSubsidy > 0 {
		params.AmanaNetParams.BaseSubsidy = f.BaseSubsidy
	}
	if f.CoinbaseMaturity > 0 {
		params.AmanaNetParams.CoinbaseMaturity = f.CoinbaseMaturity
	}
	if f.TargetTimePerBlock > 0 {
		tt := time.Second * time.Duration(f.TargetTimePerBlock)
		params.AmanaNetParams.TargetTimePerBlock = tt
		params.AmanaNetParams.TargetTimespan = tt * 100
	}

	if f.ChainID > 0 {
		params.AmanaNetParams.MeerConfig.ChainID = new(big.Int).SetUint64(f.ChainID)
		err = eparams.AddMeerChainConfig(&eparams.MeerChainConfig{ChainID: params.AmanaNetParams.MeerConfig.ChainID, Name: f.Name, Type: eparams.Amana})
		if err != nil {
			return err
		}
	}

	gb := params.AmanaNetParams.GenesisBlock.Block()
	gb.Header.Timestamp = time.Unix(ts, 0)
	gb.Header.Engine.(*ptypes.PoA).Signers = signers

	params.AmanaNetParams.GenesisBlock = types.NewBlock(gb)
	params.AmanaNetParams.GenesisHash = params.AmanaNetParams.GenesisBlock.Hash()

	if len(f.Alloc) <= 0 {
		params.AmanaNetParams.MeerAlloc = etypes.GenesisAlloc{}
	} else {
		params.AmanaNetParams.MeerAlloc = f.Alloc
	}
	return nil
}
