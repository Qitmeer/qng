// Copyright (c) 2017-2018 The qitmeer developers

package cmd

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/meerevm/amana"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli/v2"
)

var (
	Commands = []*cli.Command{
		removedbCommand,
		// See accountcmd.go:
		accountCommand,
		walletCommand,
		// See consolecmd.go:
		attachCommand,
		// see dbcmd.go
		dbCommand,
		// See cmd/utils/flags_legacy.go
		utils.ShowDeprecated,
		// See snapshot.go
		snapshotCommand,
	}
)

func makeConfigNode(ctx *cli.Context, cfg *config.Config) (*node.Node, *eth.Config) {
	eth.InitLog(cfg.DebugLevel, cfg.DebugPrintOrigins)
	//
	var ecfg *eth.Config
	var args []string
	var err error
	if cfg.Amana {
		ecfg, args, err = amana.MakeParams(cfg)
		if err != nil {
			log.Error(err.Error())
			return nil, nil
		}
	} else {
		ecfg, args, err = meer.MakeParams(cfg)
		if err != nil {
			log.Error(err.Error())
			return nil, nil
		}
	}
	var n *node.Node
	n, _, err = eth.MakeNakedNode(ecfg, args)
	if err != nil {
		log.Error(err.Error())
		return nil, nil
	}
	return n, ecfg
}

func makeConfig(cfg *config.Config) (*eth.Config, error) {
	var ecfg *eth.Config
	var err error
	if cfg.Amana {
		ecfg, err = amana.MakeConfig(cfg)
		if err != nil {
			log.Error(err.Error())
			return nil, nil
		}
	} else {
		ecfg, err = meer.MakeConfig(cfg)
		if err != nil {
			log.Error(err.Error())
			return nil, nil
		}
	}
	return ecfg, nil
}
