package main

import (
	"fmt"
	"github.com/Qitmeer/qng/cmd/relaynode/amanacrawl"
	"github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
)

func commands() []*cli.Command {
	cmds := []*cli.Command{}
	cmds = append(cmds, amanaWriteAddressCmd())
	cmds = append(cmds, amanacrawl.Cmd())
	return cmds
}

func amanaWriteAddressCmd() *cli.Command {
	return &cli.Command{
		Name:        "amanawriteaddress",
		Aliases:     []string{"qw"},
		Category:    "Amana",
		Usage:       "Amana writeaddress",
		Description: "Amana manager",
		Before: func(context *cli.Context) error {
			return config.Conf.Load()
		},
		Action: func(ctx *cli.Context) error {
			cfg := config.Conf
			pk, err := common.PrivateKey(cfg.DataDir, cfg.PrivateKey, 0600)
			if err != nil {
				return err
			}
			nk, err := common.ToECDSAPrivKey(pk)
			if err != nil {
				return err
			}
			fmt.Printf("%x\n", crypto.FromECDSAPub(&nk.PublicKey)[1:])
			return nil
		},
	}
}
