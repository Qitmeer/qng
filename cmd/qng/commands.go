package main

import (
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerevm/cmd"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/services/index"
	"github.com/Qitmeer/qng/version"
	"github.com/urfave/cli/v2"
	"runtime"
)

func commands() []*cli.Command {
	cmds := []*cli.Command{}
	cmds = append(cmds, indexCmd())
	cmds = append(cmds, cmd.Commands...)

	for _, cmd := range cmds {
		cmd.Before = loadConfig
	}
	return cmds
}

func indexCmd() *cli.Command {
	dropvmblock := false
	return &cli.Command{
		Name:        "index",
		Aliases:     []string{"i"},
		Category:    "index",
		Usage:       "index manager",
		Description: "index manager",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "dropvmblock",
				Aliases:     []string{"dv"},
				Usage:       "Deletes the vm block index from the database on start up and then exits",
				Value:       false,
				Destination: &dropvmblock,
			},
		},
		Action: func(ctx *cli.Context) error {
			if dropvmblock {
				cfg := config.Cfg
				defer func() {
					if log.LogWrite() != nil {
						log.LogWrite().Close()
					}
				}()
				interrupt := system.InterruptListener()
				log.Info("System info", "QNG Version", version.String(), "Go version", runtime.Version())
				log.Info("System info", "Home dir", cfg.HomeDir)
				if cfg.NoFileLogging {
					log.Info("File logging disabled")
				}
				db, err := common.LoadBlockDB(cfg)
				if err != nil {
					log.Error("load block database", "error", err)
					return err
				}
				defer db.Close()
				return index.DropVMBlockIndex(db, interrupt)
			}
			return cli.ShowAppHelp(ctx)
		},
		After: func(ctx *cli.Context) error {
			log.Info("Exit index command")
			return nil
		},
	}
}

func loadConfig(ctx *cli.Context) error {
	cfg, err := common.LoadConfig(ctx, false)
	if err != nil {
		return err
	}
	config.Cfg = cfg
	return nil
}
