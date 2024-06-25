package crawl

import (
	"github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/Qitmeer/qng/meerevm/amana"
	"github.com/urfave/cli/v2"
)

func amanaCmd() *cli.Command {
	var qd *CrawlService
	return &cli.Command{
		Name:        "amanacrawl",
		Aliases:     []string{"qc"},
		Category:    "crawl",
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
			ecfg, err := amana.MakeConfig(".")
			if err != nil {
				return err
			}
			qd = NewCrawlService(cfg,ecfg, ctx)
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