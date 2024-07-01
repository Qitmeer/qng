package crawl

import (
	"fmt"
	"github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/urfave/cli/v2"
	"time"
)

var (
	bootnodesFlag = &cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated nodes used for bootstrapping",
	}
	nodedbFlag = &cli.StringFlag{
		Name:  "nodedb",
		Usage: "Nodes database location",
	}
	listenAddrFlag = &cli.StringFlag{
		Name:  "addr",
		Usage: "Listening address",
	}
	crawlTimeoutFlag = &cli.DurationFlag{
		Name:  "timeout",
		Usage: "Time limit for the Amana crawl.",
		Value: 30 * time.Minute,
	}
)

func Cmds() []*cli.Command {
	return []*cli.Command{amanaCmd(), amanaNodesCmd(), meerCmd(), meerNodesCmd(), bootNodesCmd()}
}

// commandHasFlag returns true if the current command supports the given flag.
func commandHasFlag(ctx *cli.Context, flag cli.Flag) bool {
	names := flag.Names()
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	for _, fn := range ctx.FlagNames() {
		if _, ok := set[fn]; ok {
			return true
		}
	}
	return false
}

func bootNodesCmd() *cli.Command {
	return &cli.Command{
		Name:        "bootnodes",
		Aliases:     []string{"mn"},
		Category:    "crawl",
		Usage:       "Show boot nodes info",
		Description: "Show boot nodes info",
		Flags: []cli.Flag{
			bootnodesFlag,
		},
		Before: func(ctx *cli.Context) error {
			return config.Conf.Load()
		},
		Action: func(ctx *cli.Context) error {
			if !commandHasFlag(ctx, bootnodesFlag) {
				return fmt.Errorf("No bootnodes config")
			}
			_, err := parseBootnodes(ctx)
			return err
		},
		After: func(ctx *cli.Context) error {
			return nil
		},
	}
}