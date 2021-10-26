/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainvm

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

type Factory struct {
	Path string
	arg  string
}

func (f *Factory) New(ctx context.Context) (interface{}, error) {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "VM",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	config := &plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins:         PluginMap,
		Cmd:             exec.Command(f.Path, f.arg),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
			plugin.ProtocolGRPC,
		},
		Managed: true,
		Logger:  logger,
	}

	client := plugin.NewClient(config)

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}

	raw, err := rpcClient.Dispense("vm")
	if err != nil {
		client.Kill()
		return nil, err
	}

	vm, ok := raw.(*VMClient)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("wrong vm type")
	}

	vm.SetProcess(client)
	vm.ctx = ctx
	return vm, nil
}
