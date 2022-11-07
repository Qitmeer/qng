package index

import "github.com/Qitmeer/qng/config"

type Config struct {
	TxIndex        bool
	AddrIndex      bool
	VMBlockIndex   bool
	InvalidTxIndex bool
}

func DefaultConfig() *Config {
	return &Config{
		TxIndex:        true,
		AddrIndex:      false,
		VMBlockIndex:   false,
		InvalidTxIndex: false,
	}
}

func ToConfig(cfg *config.Config) *Config {
	return &Config{
		TxIndex:        true,
		AddrIndex:      cfg.AddrIndex,
		VMBlockIndex:   cfg.VMBlockIndex,
		InvalidTxIndex: cfg.CacheInvalidTx,
	}
}
