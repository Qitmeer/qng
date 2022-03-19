/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package main

import (
	"github.com/Qitmeer/qng/common/roughtime"
	_ "github.com/Qitmeer/qng/database/ffldb"
	_ "github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/version"
	"github.com/urfave/cli/v2"
	"os"
	"runtime"
	"runtime/debug"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	debug.SetGCPercent(20)
	if err := relayNode(); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func relayNode() error {
	node := &Node{}
	app := &cli.App{
		Name:     "RelayNode",
		Version:  version.String(),
		Compiled: roughtime.Now(),
		Authors: []*cli.Author{
			&cli.Author{
				Name: "QNG",
			},
		},
		Copyright:            "(c) 2020 QNG",
		Usage:                "Relay Node",
		Flags:                AppFlags,
		EnableBashCompletion: true,
		Before: func(c *cli.Context) error {
			return node.init(conf)
		},
		After: func(c *cli.Context) error {
			return node.Stop()
		},
		Action: func(c *cli.Context) error {
			return node.Start()
		},
	}

	return app.Run(os.Args)
}
