package main

import (
	"fmt"
	"github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/Qitmeer/qng/cmd/relaynode/qitcrawl"
	"github.com/Qitmeer/qng/p2p"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
)

func commands() []*cli.Command {
	cmds := []*cli.Command{}
	cmds = append(cmds, qitWriteAddressCmd())
	cmds = append(cmds, qitcrawl.Cmd())
	return cmds
}

func qitWriteAddressCmd() *cli.Command {
	return &cli.Command{
		Name:        "qitwriteaddress",
		Aliases:     []string{"qw"},
		Category:    "qit",
		Usage:       "QitSubnet writeaddress",
		Description: "qit manager",
		Before: func(context *cli.Context) error {
			return config.Conf.Load()
		},
		Action: func(ctx *cli.Context) error {
			cfg := config.Conf
			pk, err := p2p.PrivateKey(cfg.DataDir, cfg.PrivateKey, 0600)
			if err != nil {
				return err
			}
			nk, err := p2p.ToECDSAPrivKey(pk)
			if err != nil {
				return err
			}
			fmt.Printf("%x\n", crypto.FromECDSAPub(&nk.PublicKey)[1:])
			return nil
		},
	}
}
