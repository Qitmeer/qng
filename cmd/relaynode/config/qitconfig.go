package config

type QitBootConfig struct {
	Enable      bool
	ListenAddr  string
	Natdesc     string
	Netrestrict string
	Runv5       bool
}
