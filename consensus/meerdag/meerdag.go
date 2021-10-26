// Package meerdag implements the MeerDAG consensus engine.
package main

import (
	"github.com/Qitmeer/qng/consensus/meerdag/meervm"
	"github.com/Qitmeer/qng/version"
	"github.com/Qitmeer/qng/vm/chainvm"
	"github.com/hashicorp/go-plugin"
	"runtime"
)

func main() {
	log.Info("System info", "ETH VM Version", version.String(), "Go version", runtime.Version())

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: chainvm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": chainvm.New(&meervm.VM{}),
		},

		GRPCServer: plugin.DefaultGRPCServer,
	})
}
