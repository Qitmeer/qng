package eth

import (
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/eth/downloader"
)

type dnsConfig struct {
	prefix string
	domain string
}

var (
	dnscfgs = []dnsConfig{
		{prefix: "enrtree://APY42GPXO2Q6I7BOXMYAELS5JU6WJXJLN64RFZOKG2MUCN6SLNGRM@", domain: "qitmeer.top"},
	}
)

func SetDNSDiscoveryDefaults(cfg *Config) {
	if cfg.Eth.EthDiscoveryURLs != nil {
		return // already set through flags/config
	}
	if len(dnscfgs) <= 0 {
		return
	}
	protocol := "all"
	if cfg.Eth.SyncMode == downloader.LightSync {
		protocol = "les"
	}
	cfg.Eth.EthDiscoveryURLs = []string{}
	for _, dc := range dnscfgs {
		url := dc.prefix + protocol + "." + cfg.Node.Name + "." + params.ActiveNetParams.Name + "." + dc.domain
		cfg.Eth.EthDiscoveryURLs = append(cfg.Eth.EthDiscoveryURLs, url)
	}
	cfg.Eth.SnapDiscoveryURLs = cfg.Eth.EthDiscoveryURLs
}
