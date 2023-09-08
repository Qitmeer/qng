package index

import "github.com/Qitmeer/qng/config"

type Config struct {
	AddrIndex      bool
	InvalidTxIndex bool
	TxhashIndex    bool
}

func DefaultConfig() *Config {
	return &Config{
		AddrIndex:      false,
		InvalidTxIndex: false,
		TxhashIndex:    false,
	}
}

func ToConfig(cfg *config.Config) *Config {
	return &Config{
		AddrIndex:      cfg.AddrIndex,
		InvalidTxIndex: cfg.InvalidTxIndex,
		TxhashIndex:    cfg.TxHashIndex,
	}
}
