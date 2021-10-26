package common

import "context"

type VM interface {
	Initialize(ctx context.Context) error
	Bootstrapping() error
	Bootstrapped() error
	Shutdown() error
	Version() (string, error)
}
