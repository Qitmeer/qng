package config

import "github.com/Qitmeer/qng/v2/consensus/engine"

type PoAConfig struct {
	Period uint64 // Number of seconds between blocks to enforce
	Epoch  uint64 // Epoch length to reset votes and checkpoint
}

func (c *PoAConfig) Type() engine.EngineType {
	return engine.PoAEngineType
}

func (c *PoAConfig) Check() error {
	return nil
}
