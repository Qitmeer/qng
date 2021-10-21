/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meervm

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

type VM struct {
}

func (vm *VM) Version() (string, error) {
	return "", nil
}

func (vm *VM) GetBlock(*hash.Hash) (*types.Block, error) {
	return nil, nil
}

func (vm *VM) BuildBlock() (*types.Block, error) {
	return nil, nil
}

func (vm *VM) ParseBlock([]byte) (*types.Block, error) {
	return nil, nil
}

func (vm *VM) Shutdown() error {

	return nil
}
