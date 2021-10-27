/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meervm

import (
	"context"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/version"
	"sync"
)

// ID of the platform VM
var (
	ID = "meerdag"
)

type VM struct {
	ctx          context.Context
	shutdownChan chan struct{}
	shutdownWg   sync.WaitGroup
}

func (vm *VM) Initialize(ctx context.Context) error {
	log.Debug("Initialize")

	vm.shutdownChan = make(chan struct{}, 1)
	vm.ctx = ctx

	return nil
}

func (vm *VM) Bootstrapping() error {
	log.Debug("Bootstrapping")
	return nil
}

func (vm *VM) Bootstrapped() error {
	log.Debug("Bootstrapped")
	return nil
}

func (vm *VM) Shutdown() error {
	log.Debug("Shutdown")
	if vm.ctx == nil {
		return nil
	}

	close(vm.shutdownChan)
	vm.shutdownWg.Wait()
	return nil
}

func (vm *VM) Version() (string, error) {
	return version.String(), nil
}

func (vm *VM) GetBlock(*hash.Hash) (consensus.Block, error) {
	return nil, nil
}

func (vm *VM) BuildBlock() (consensus.Block, error) {
	return nil, nil
}

func (vm *VM) ParseBlock([]byte) (consensus.Block, error) {
	return nil, nil
}

func (vm *VM) LastAccepted() (*hash.Hash, error) {
	return nil, nil
}
