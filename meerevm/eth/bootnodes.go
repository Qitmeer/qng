package eth

import (
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"net"
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
	cfg.Eth.EthDiscoveryURLs = []string{}
	for _, dc := range dnscfgs {
		url := dc.prefix + protocol + "." + cfg.Node.Name + "." + params.ActiveNetParams.Name + "." + dc.domain
		cfg.Eth.EthDiscoveryURLs = append(cfg.Eth.EthDiscoveryURLs, url)
	}
	cfg.Eth.SnapDiscoveryURLs = cfg.Eth.EthDiscoveryURLs
}

func mustParseBootnodes(urls []string) []*enode.Node {
	nodes := make([]*enode.Node, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			node, err := enode.Parse(enode.ValidSchemes, url)
			if err != nil {
				log.Crit("Bootstrap URL invalid", "enode", url, "err", err)
				return nil
			}
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func defaultBootNodeUrl(port int) string {
	db, _ := enode.OpenDB("")
	key, _ := crypto.GenerateKey()
	ln := enode.NewLocalNode(db, key)
	ln.SetFallbackIP(net.IP{127, 0, 0, 1})
	ln.SetFallbackUDP(port)
	return ln.Node().String()
}

func GetBootstrapNodes(port int, urls []string) []*enode.Node {
	if len(urls) <= 0 {
		urls = append(urls, defaultBootNodeUrl(port))
	}
	return mustParseBootnodes(urls)
}
