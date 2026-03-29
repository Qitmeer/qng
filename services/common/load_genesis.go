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
	"github.com/ethereum/go-ethereum/core"
	etypes "github.com/ethereum/go-ethereum/core/types"
	eparams "github.com/ethereum/go-ethereum/params"
)

type CustomGenesis struct {
	Name   string `json:"name"`
	Params *struct {
		TargetTimePerBlock int    `json:"target_time_per_block"`
		BaseSubsidy        int64  `json:"base_subsidy"`
		CoinbaseMaturity   uint16 `json:"coinbase_maturity"`
		Timestamp          int64  `json:"timestamp"`
	} `json:"params"`

	Consensus *struct {
		PoA *struct {
			Period  uint64   `json:"period"`
			Epoch   uint64   `json:"epoch"`
			Signers []string `json:"signers"`
		} `json:"poa"`
	} `json:"consensus"`

	EVMConfig *struct {
		Config *struct {
			ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection

			HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

			DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
			DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

			// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
			EIP150Block *big.Int `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
			EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
			EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

			ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
			ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
			PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`     // Petersburg switch block (nil = same as Constantinople)
			IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`       // Istanbul switch block (nil = no fork, 0 = already on istanbul)
			MuirGlacierBlock    *big.Int `json:"muirGlacierBlock,omitempty"`    // Eip-2384 (bomb delay) switch block (nil = no fork, 0 = already activated)
			BerlinBlock         *big.Int `json:"berlinBlock,omitempty"`         // Berlin switch block (nil = no fork, 0 = already on berlin)
			LondonBlock         *big.Int `json:"londonBlock,omitempty"`         // London switch block (nil = no fork, 0 = already on london)
			ArrowGlacierBlock   *big.Int `json:"arrowGlacierBlock,omitempty"`   // Eip-4345 (bomb delay) switch block (nil = no fork, 0 = already activated)
			GrayGlacierBlock    *big.Int `json:"grayGlacierBlock,omitempty"`    // Eip-5133 (bomb delay) switch block (nil = no fork, 0 = already activated)
			MergeNetsplitBlock  *big.Int `json:"mergeNetsplitBlock,omitempty"`  // Virtual fork after The Merge to use as a network splitter
		} `json:"config"`
		ExtraData []byte              `json:"extraData,omitempty"`
		Alloc     etypes.GenesisAlloc `json:"alloc,omitempty"`
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
	// name
	if len(f.Name) > 0 {
		params.AmanaNetParams.Name = f.Name
	}

	// Params
	if f.Params != nil {
		if f.Params.BaseSubsidy > 0 {
			params.AmanaNetParams.BaseSubsidy = f.Params.BaseSubsidy
		}
		if f.Params.CoinbaseMaturity > 0 {
			params.AmanaNetParams.CoinbaseMaturity = f.Params.CoinbaseMaturity
		}
		if f.Params.TargetTimePerBlock > 0 {
			tt := time.Second * time.Duration(f.Params.TargetTimePerBlock)
			params.AmanaNetParams.TargetTimePerBlock = tt
			params.AmanaNetParams.TargetTimespan = tt * 100
		}
	}

	// Consensus
	// poa
	if f.Consensus != nil {
		poaCfg, ok := params.AmanaNetParams.ConsensusConfig.(*cfgengine.PoAConfig)
		if !ok {
			return fmt.Errorf("applyCustomGenesis: Amana consensus is not PoA")
		}
		if f.Consensus.PoA.Period > 0 {
			poaCfg.Period = f.Consensus.PoA.Period
		}
		if f.Consensus.PoA.Epoch > 0 {
			poaCfg.Epoch = f.Consensus.PoA.Epoch
		}
		if len(f.Consensus.PoA.Signers) == 0 {
			return fmt.Errorf("applyCustomGenesis: signers must not be empty")
		}
		signers := make([]common.Address, 0, len(f.Consensus.PoA.Signers))
		for _, s := range f.Consensus.PoA.Signers {
			if !common.IsHexAddress(s) {
				return fmt.Errorf("applyCustomGenesis: invalid signer address %q", s)
			}
			signers = append(signers, common.HexToAddress(s))
		}

		gb := params.AmanaNetParams.GenesisBlock.Block()
		if f.Params != nil {
			gb.Header.Timestamp = time.Unix(f.Params.Timestamp, 0)
		}
		gb.Header.Engine.(*ptypes.PoA).Signers = signers

		params.AmanaNetParams.GenesisBlock = types.NewBlock(gb)
		params.AmanaNetParams.GenesisHash = params.AmanaNetParams.GenesisBlock.Hash()
	}

	// evm
	if f.EVMConfig != nil {
		if f.EVMConfig.Config != nil {
			if f.EVMConfig.Config.ChainID != nil {
				params.AmanaNetParams.MeerConfig.ChainID = f.EVMConfig.Config.ChainID
				err = eparams.AddMeerChainConfig(&eparams.MeerChainConfig{ChainID: params.AmanaNetParams.MeerConfig.ChainID, Name: params.AmanaNetParams.Name, Type: eparams.Amana})
				if err != nil {
					return err
				}
			}
		}
		params.AmanaNetParams.MeerGenesis = &core.Genesis{
			Config: &eparams.ChainConfig{
				ChainID:             f.EVMConfig.Config.ChainID,
				HomesteadBlock:      f.EVMConfig.Config.HomesteadBlock,
				DAOForkBlock:        f.EVMConfig.Config.DAOForkBlock,
				DAOForkSupport:      f.EVMConfig.Config.DAOForkSupport,
				EIP150Block:         f.EVMConfig.Config.EIP150Block,
				EIP155Block:         f.EVMConfig.Config.EIP155Block,
				EIP158Block:         f.EVMConfig.Config.EIP158Block,
				ByzantiumBlock:      f.EVMConfig.Config.ByzantiumBlock,
				ConstantinopleBlock: f.EVMConfig.Config.ConstantinopleBlock,
				PetersburgBlock:     f.EVMConfig.Config.PetersburgBlock,
				IstanbulBlock:       f.EVMConfig.Config.IstanbulBlock,
				MuirGlacierBlock:    f.EVMConfig.Config.MuirGlacierBlock,
				BerlinBlock:         f.EVMConfig.Config.BerlinBlock,
				LondonBlock:         f.EVMConfig.Config.LondonBlock,
				ArrowGlacierBlock:   f.EVMConfig.Config.ArrowGlacierBlock,
				GrayGlacierBlock:    f.EVMConfig.Config.GrayGlacierBlock,
				MergeNetsplitBlock:  f.EVMConfig.Config.MergeNetsplitBlock,
			},
			ExtraData: f.EVMConfig.ExtraData,
			Alloc:     f.EVMConfig.Alloc,
		}
	}
	return nil
}
