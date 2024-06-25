package crawl

import (
	"github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/services/common"
	"github.com/urfave/cli/v2"
)

func meerCmd() *cli.Command {
	var qd *CrawlService
	return &cli.Command{
		Name:        "meercrawl",
		Aliases:     []string{"mc"},
		Category:    "crawl",
		Usage:       "Updates a nodes.json file with random nodes found in the DHT for Meer",
		Description: "Updates a nodes.json file with random nodes found in the DHT for Meer",
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
			ecfg, err := meer.MakeConfig(common.DefaultConfig("."))
			if err != nil {
				return err
			}
			qd = NewCrawlService(cfg, ecfg, ctx)
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
