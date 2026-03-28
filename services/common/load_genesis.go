// Copyright (c) 2017-2025 The qitmeer developers

package common

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
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
	Name string `json:"name"`
	PoA  struct {
		Period  uint64   `json:"period"`
		Epoch   uint64   `json:"epoch"`
		Signers []string `json:"signers"`
	} `json:"poa"`
	EVMConfig struct {
		ChainID   uint64              `json:"chain_id"`
		Timestamp int64               `json:"timestamp"`
		Alloc     etypes.GenesisAlloc `json:"alloc"`
	} `json:"evm"`
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
	if len(f.PoA.Signers) == 0 {
		return fmt.Errorf("applyCustomGenesis: signers must not be empty")
	}

	signers := make([]common.Address, 0, len(f.PoA.Signers))
	for _, s := range f.PoA.Signers {
		if !common.IsHexAddress(s) {
			return fmt.Errorf("applyCustomGenesis: invalid signer address %q", s)
		}
		signers = append(signers, common.HexToAddress(s))
	}

	poaCfg, ok := params.AmanaNetParams.ConsensusConfig.(*cfgengine.PoAConfig)
	if !ok {
		return fmt.Errorf("applyCustomGenesis: Amana consensus is not PoA")
	}
	if f.PoA.Period > 0 {
		poaCfg.Period = f.PoA.Period
	}
	if f.PoA.Epoch > 0 {
		poaCfg.Epoch = f.PoA.Epoch
	}

	if f.Name != "" {
		params.AmanaNetParams.Name = f.Name
	}

	if f.EVMConfig.ChainID > 0 {
		params.AmanaNetParams.MeerConfig.ChainID = new(big.Int).SetUint64(f.EVMConfig.ChainID)
		err = eparams.AddMeerChainConfig(&eparams.MeerChainConfig{ChainID: params.AmanaNetParams.MeerConfig.ChainID, Name: f.Name, Type: eparams.Amana})
		if err != nil {
			return err
		}
	}

	gb := params.AmanaNetParams.GenesisBlock.Block()
	gb.Header.Timestamp = time.Unix(f.EVMConfig.Timestamp, 0)
	gb.Header.Engine.(*ptypes.PoA).Signers = signers

	params.AmanaNetParams.GenesisBlock = types.NewBlock(gb)
	params.AmanaNetParams.GenesisHash = params.AmanaNetParams.GenesisBlock.Hash()

	if len(f.EVMConfig.Alloc) <= 0 {
		params.AmanaNetParams.MeerAlloc = etypes.GenesisAlloc{}
	} else {
		params.AmanaNetParams.MeerAlloc = f.EVMConfig.Alloc
	}
	return nil
}
