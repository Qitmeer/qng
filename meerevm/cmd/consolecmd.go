// Copyright (c) 2017-2018 The qitmeer developers

package cmd

import (
	"github.com/Qitmeer/qng/v2/config"
	"github.com/Qitmeer/qng/v2/meerevm/meer"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli/v2"
)

var (
	consoleFlags = []cli.Flag{utils.JSpathFlag, utils.ExecFlag, utils.PreloadJSFlag}

	attachCommand = &cli.Command{
		Action:    remoteConsole,
		Name:      "attach",
		Usage:     "Start an interactive JavaScript environment (connect to node)",
		ArgsUsage: "[endpoint]",
		Flags:     append(consoleFlags, utils.DataDirFlag, utils.HttpHeaderFlag),
		Category:  "CONSOLE COMMANDS",
		Description: `
The QNG console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://geth.ethereum.org/docs/interface/javascript-console.
This command allows to open a console on a running geth node.`,
	}
)

// remoteConsole will connect to a remote geth instance, attaching a JavaScript
// console to it.
func remoteConsole(ctx *cli.Context) error {
	// Attach to a remotely running geth instance and start the JavaScript console
	endpoint := ctx.Args().First()
	nconfig := &node.Config{DataDir: config.Cfg.DataDir}
	if ctx.IsSet(utils.DataDirFlag.Name) {
		nconfig.DataDir = ctx.String(utils.DataDirFlag.Name)
	}
	if endpoint == "" {
		nconfig.IPCPath = meer.ClientIdentifier + ".ipc"
		endpoint = nconfig.IPCEndpoint()

	}
	client, err := utils.DialRPCWithHeaders(endpoint, ctx.StringSlice(utils.HttpHeaderFlag.Name))
	if err != nil {
		utils.Fatalf("Unable to attach to remote qng: %v", err)
	}
	config := console.Config{
		DataDir: nconfig.DataDir,
		DocRoot: ctx.String(utils.JSpathFlag.Name),
		Client:  client,
		Preload: utils.MakeConsolePreloads(ctx),
	}

	console, err := console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	if script := ctx.String(utils.ExecFlag.Name); script != "" {
		console.Evaluate(script)
		return nil
	}

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

	return nil
}
