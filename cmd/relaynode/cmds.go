package main

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/p2p"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
)

func commands() []*cli.Command {
	cmds := []*cli.Command{}
	cmds = append(cmds, qitCmd())
	return cmds
}

func qitCmd() *cli.Command {
	return &cli.Command{
		Name:        "qit",
		Aliases:     []string{"qi"},
		Category:    "qit",
		Usage:       "qit manager",
		Description: "qit manager",
		Subcommands: []*cli.Command{
			&cli.Command{
				Name:        "writeaddress",
				Aliases:     []string{"wa"},
				Usage:       "QitSubnet writeaddress",
				Description: "QitSubnet writeaddress",
				Action: func(ctx *cli.Context) error {
					cfg := conf
					pk, err := p2p.PrivateKey(cfg.DataDir, cfg.PrivateKey, 0600)
					if err != nil {
						return err
					}
					pkb, err := pk.Raw()
					if err != nil {
						return err
					}
					nk, err := crypto.HexToECDSA(hex.EncodeToString(pkb))
					if err != nil {
						return err
					}
					fmt.Printf("%x\n", crypto.FromECDSAPub(&nk.PublicKey)[1:])
					return nil
				},
			},
		},
		Before: func(context *cli.Context) error {
			return conf.load()
		},
	}
}
