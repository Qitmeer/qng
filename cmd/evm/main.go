// (c) 2021, the Qitmeer developers. All rights reserved.
// license that can be found in the LICENSE file.
package main

import (
	"github.com/Qitmeer/meerevm/cmd/evm/vm"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/vm/chainvm"
	"github.com/hashicorp/go-plugin"
	"runtime"
)

var (
	// Version is the version of MeerEvm
	Version = "meerevm-v0.0.0"
)

func main() {
	log.Info("System info", "ETH VM Version", Version, "Go version", runtime.Version())

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: chainvm.Handshake,
		Plugins: map[string]plugin.Plugin{
			"vm": chainvm.New(&vm.VM{}),
		},

		GRPCServer: plugin.DefaultGRPCServer,
	})
}
