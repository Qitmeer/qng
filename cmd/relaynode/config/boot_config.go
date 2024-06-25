package config

type BootConfig struct {
	Enable      bool
	ListenAddr  string
	Natdesc     string
	Netrestrict string
	Runv5       bool
}
