package config

type AmanaBootConfig struct {
	Enable      bool
	ListenAddr  string
	Natdesc     string
	Netrestrict string
	Runv5       bool
}
