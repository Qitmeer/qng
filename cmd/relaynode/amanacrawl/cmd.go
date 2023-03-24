package amanacrawl

import (
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

func Cmd() *cli.Command {
	var qd *AmanaCrawlService
	return &cli.Command{
		Name:        "amanacrawl",
		Aliases:     []string{"qc"},
		Category:    "amana",
		Usage:       "Updates a nodes.json file with random nodes found in the DHT for Amana",
		Description: "Updates a nodes.json file with random nodes found in the DHT for Amana",
		Flags: []cli.Flag{
			bootnodesFlag,
			nodedbFlag,
			crawlTimeoutFlag,
		},
		Before: func(ctx *cli.Context) error {
			return config.Conf.Load()
		},
		Action: func(ctx *cli.Context) error {
			cfg := config.Conf
			qd = NewAmanaCrawlService(cfg, ctx)
			return qd.Start()
		},
		After: func(ctx *cli.Context) error {
			if qd != nil {
				return qd.Stop()
			}
			return nil
		},
	}
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
