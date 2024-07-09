package main

import (
	"fmt"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerevm/amana"
	"github.com/Qitmeer/qng/meerevm/cmd"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/p2p"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/version"
	"github.com/urfave/cli/v2"
	"os"
	"runtime"
)

func commands() []*cli.Command {
	cmds := []*cli.Command{}
	cmds = append(cmds, indexCmd())
	cmds = append(cmds, consensusCmd())
	cmds = append(cmds, blockchainCmd())
	cmds = append(cmds, cmd.Commands...)
	cmds = append(cmds, dbCmd())
	cmds = append(cmds, cleanupCmd())

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
					db, err := database.New(cfg, interrupt)
					if err != nil {
						log.Error("load block database", "error", err)
						return err
					}
					defer db.Close()

					return db.CleanInvalidTxIdx()
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

func cleanupCmd() *cli.Command {
	removeAll := false
	return &cli.Command{
		Name:        "cleanup",
		Aliases:     []string{"cl"},
		Usage:       "Delete all status data, including cache",
		Description: "Delete all status data, including cache",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "all",
				Aliases:     []string{"a"},
				Usage:       "Delete all data including blockchain data, peerstore, logs",
				Destination: &removeAll,
			},
		},
		Action: func(ctx *cli.Context) error {
			cfg := config.Cfg
			cfg.Cleanup = true
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
			_, err := database.New(cfg, interrupt)
			if err != nil {
				log.Error("load block database", "error", err)
				return err
			}
			if !removeAll {
				return nil
			}
			p2p.Cleanup(cfg.DataDir)
			if len(cfg.LogDir) > 0 {
				err = os.RemoveAll(cfg.LogDir)
				if err != nil {
					log.Error(err.Error())
				}
				log.Info(fmt.Sprintf("Finished cleanup:%s", cfg.LogDir))
			}
			return nil
		},
	}
}

func loadConfig(ctx *cli.Context) error {
	_, err := common.LoadConfig(ctx, false)
	if err != nil {
		return err
	}
	return nil
}
