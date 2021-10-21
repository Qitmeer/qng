// Package meerdag implements the MeerDAG consensus engine.
package meerdag

import (
	"github.com/Qitmeer/qng/consensus/meerdag/meervm"
	"github.com/Qitmeer/qng/version"
	"github.com/Qitmeer/qng/vm"
	"github.com/hashicorp/go-plugin"
	"runtime"
)

func main() {
	log.Info("System info", "ETH VM Version", version.String(), "Go version", runtime.Version())

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: vm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": vm.New(&meervm.VM{}),
		},

		GRPCServer: plugin.DefaultGRPCServer,
	})
}
