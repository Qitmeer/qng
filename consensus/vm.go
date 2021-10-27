/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import "context"

type VM interface {
	Initialize(ctx context.Context) error
	Bootstrapping() error
	Bootstrapped() error
	Shutdown() error
	Version() (string, error)
}
