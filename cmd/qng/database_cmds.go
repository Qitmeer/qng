package main

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/database/chaindb"
	"github.com/Qitmeer/qng/database/rawdb"
	"github.com/urfave/cli/v2"
)

func dbCmd() *cli.Command {
	var (
		prefix string
		start  string
	)
	return &cli.Command{
		Name:        "DB",
		Aliases:     []string{"db"},
		Category:    "DataBase",
		Usage:       "Low level database operations",
		Description: "Low level database operations",
		Subcommands: []*cli.Command{
			&cli.Command{
				Name:        "inspect",
				Aliases:     []string{"i"},
				Usage:       "Inspect the storage size for each type of data in the database",
				Description: "This commands iterates the entire database. If the optional 'prefix' and 'start' arguments are provided, then the iteration is limited to the given subset of data.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "prefix",
						Aliases:     []string{"p"},
						Usage:       "prefix",
						Destination: &prefix,
					},
					&cli.StringFlag{
						Name:        "start",
						Aliases:     []string{"e"},
						Usage:       "start",
						Destination: &start,
					},
				},
				Action: func(ctx *cli.Context) error {
					cfg := config.Cfg
					db, err := chaindb.New(cfg)
					if err != nil {
						return err
					}
					defer db.Close()

					return rawdb.InspectDatabase(db.DB(), []byte(prefix), []byte(start))
				},
			},
		},
	}
}
