package cmd

import (
	"errors"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/meerevm/eth"
	cli "github.com/urfave/cli/v2"
	"strconv"
)

var (
	setHeadCommand = &cli.Command{
		Action:    setHead,
		Name:      "sethead",
		Usage:     "Set up a new head for block chain",
		ArgsUsage: "<block number>",
		Description: `
Set up a new head for block chain`,
	}
)

func setHead(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		return errors.New("This command requires an argument.")
	}
	if len(ctx.Args().First()) <= 0 {
		return errors.New("No block number")
	}
	number, err := strconv.ParseUint(ctx.Args().First(), 10, 64)
	if err != nil {
		return err
	}
	if !config.Cfg.Amana {
		return errors.New("Currently only supported amana")
	}
	stack, cfg := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()

	chain, db, err := eth.MakeChain(ctx, stack, false, cfg)
	defer db.Close()
	if err != nil {
		return err
	}
	return chain.SetHead(number)
}
