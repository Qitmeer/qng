/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chainvm

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/consensus"
	"os/exec"

	qlog "github.com/Qitmeer/qng/log"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

type Factory struct {
	Path          string
	arg           string
	LogLevel      string
	LogIncludeLoc bool
	vm            *VMClient
}

func (f *Factory) New(ctx context.Context) (consensus.ChainVM, error) {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:            "VM",
		Output:          qlog.LogWrite(),
		Level:           hclog.LevelFromString(f.LogLevel),
		IncludeLocation: false,
		TimeFormat:      qlog.TermTimeFormat,
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
	f.vm = vm
	return vm, nil
}

func (f *Factory) GetVM() consensus.ChainVM {
	return f.vm
}
