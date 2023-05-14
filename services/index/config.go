package index

import "github.com/Qitmeer/qng/config"

type Config struct {
	TxIndex        bool
	AddrIndex      bool
	InvalidTxIndex bool
}

func DefaultConfig() *Config {
	return &Config{
		TxIndex:        true,
		AddrIndex:      false,
		InvalidTxIndex: false,
	}
}

func ToConfig(cfg *config.Config) *Config {
	return &Config{
		TxIndex:        true,
		AddrIndex:      cfg.AddrIndex,
		InvalidTxIndex: cfg.InvalidTxIndex,
	}
}
