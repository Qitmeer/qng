package config

import "github.com/Qitmeer/qng/v2/consensus/engine"

type Config interface {
	Type() engine.EngineType
	Check() error
}
