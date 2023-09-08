package main

import (
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/database/legacychaindb"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerevm/amana"
	"github.com/Qitmeer/qng/meerevm/cmd"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/services/index"
	"github.com/Qitmeer/qng/version"
	"github.com/urfave/cli/v2"
	"runtime"
)

func commands() []*cli.Command {
	cmds := []*cli.Command{}
	cmds = append(cmds, indexCmd())
	cmds = append(cmds, consensusCmd())
	cmds = append(cmds, blockchainCmd())
	cmds = append(cmds, cmd.Commands...)

	for _, cmd := range cmds {
		cmd.Before = loadConfig
	}
	return cmds
}

func indexCmd() *cli.Command {
	return &cli.Command{
		Name:        "index",
		Aliases:     []string{"i"},
		Category:    "index",
		Usage:       "index manager",
		Description: "index manager",
		Subcommands: []*cli.Command{
			&cli.Command{
				Name:        "dropinvalidtxindex",
				Aliases:     []string{"di"},
				Usage:       "Deletes the invalid tx index from the database on start up and then exits",
				Description: "Deletes the invalid tx index from the database on start up and then exits",
				Action: func(ctx *cli.Context) error {
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
					db, err := legacychaindb.LoadBlockDB(cfg)
					if err != nil {
						log.Error("load block database", "error", err)
						return err
					}
					defer db.Close()
					return index.DropInvalidTxIndex(db, interrupt)
				},
			},
		},
		After: func(ctx *cli.Context) error {
			log.Info("Exit index command")
			return nil
		},
	}
}

func consensusCmd() *cli.Command {
	return &cli.Command{
		Name:        "consensus",
		Aliases:     []string{"c"},
		Category:    "consensus",
		Usage:       "consensus",
		Description: "consensus",
		Subcommands: []*cli.Command{
			&cli.Command{
				Name:        "rebuild",
				Aliases:     []string{"re"},
				Usage:       "rebuild consensus",
				Description: "rebuild consensus",
				Action: func(ctx *cli.Context) error {
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
					db, err := database.New(cfg, interrupt)
					if err != nil {
						log.Error("load block database", "error", err)
						return err
					}
					defer db.Close()

					meer.Cleanup(cfg)
					amana.Cleanup(cfg)
					//
					cfg.InvalidTxIndex = false
					cfg.AddrIndex = false
					cons := consensus.New(cfg, db, interrupt, make(chan struct{}))
					err = cons.Init()
					if err != nil {
						log.Error(err.Error())
						return err
					}
					err = cons.BlockChain().Start()
					if err != nil {
						return err
					}
					return cons.Rebuild()
				},
			},
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
