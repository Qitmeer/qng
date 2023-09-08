package amana

import (
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/protocol"
	mconsensus "github.com/Qitmeer/qng/meerevm/amana/consensus"
	mcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/eth"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"net"
	"path/filepath"
)

var (
	ClientIdentifier = mconsensus.Identifier
)

func MakeConfig(datadir string) (*eth.Config, error) {
	genesis := Genesis()

	econfig := ethconfig.Defaults

	econfig.NetworkId = genesis.Config.ChainID.Uint64()
	econfig.Genesis = genesis
	econfig.ConsensusEngine = createConsensusEngine

	nodeConf := node.DefaultConfig
	nodeConf.DataDir = datadir
	nodeConf.Name = ClientIdentifier
	nodeConf.Version = params.VersionWithMeta
	nodeConf.HTTPModules = append(nodeConf.HTTPModules, "eth")
	nodeConf.WSModules = append(nodeConf.WSModules, "eth")
	nodeConf.IPCPath = ClientIdentifier + ".ipc"
	nodeConf.KeyStoreDir = filepath.Join(datadir, "keystore")
	var p2pPort int
	nodeConf.HTTPPort, nodeConf.WSPort, nodeConf.AuthPort, p2pPort = getDefaultPort()
	nodeConf.P2P.ListenAddr = fmt.Sprintf(":%d", p2pPort)
	nodeConf.P2P.BootstrapNodes = getBootstrapNodes(p2pPort)

	cfg := &eth.Config{
		Eth:     econfig,
		Node:    nodeConf,
		Metrics: metrics.DefaultConfig,
	}
	eth.SetDNSDiscoveryDefaults(cfg)
	return cfg, nil
}

func MakeParams(cfg *config.Config) (*eth.Config, []string, error) {
	ecfg, err := MakeConfig(cfg.DataDir)
	if err != nil {
		return ecfg, nil, err
	}
	args, err := mcommon.ProcessEnv(cfg.AmanaEnv, ecfg.Node.Name, nil)
	return ecfg, args, err
}

func getDefaultPort() (int, int, int, int) {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		return 8525, 8526, 8527, 8528
	case protocol.TestNet:
		return 18525, 18526, 18527, 18528
	case protocol.MixNet:
		return 28525, 28526, 28527, 28528
	default:
		return 38525, 38526, 38527, 38528
	}
}

func createConsensusEngine(config *params.ChainConfig, db ethdb.Database) (consensus.Engine, error) {
	return mconsensus.New(config.Clique, db), nil
}

func getBootstrapNodes(port int) []*enode.Node {
	db, _ := enode.OpenDB("")
	key, _ := crypto.GenerateKey()
	ln := enode.NewLocalNode(db, key)
	ln.SetFallbackIP(net.IP{127, 0, 0, 1})
	ln.SetFallbackUDP(port)

	urls := []string{}
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		urls = append(urls, ln.Node().String())
	case protocol.TestNet:
		urls = append(urls, "enr:-KO4QMjjjBd9NKfXaf2LCEmEVcl7q2m9ArnKOJxGaIvkC5L3Fn6zqJxi1ON4nFkH3Go4fE5oDa7uUxLWeBgsSHbWb3SGAYdv5iIng2V0aMfGhE5aFCGAgmlkgnY0gmlwhC0gC3eJc2VjcDI1NmsxoQK6ou0UT10hfVO8-b-2zWawj1_l8xflj764RASTmM8f64RzbmFwwIN0Y3CCSGCDdWRwgkhg")
	case protocol.MixNet:
		urls = append(urls, ln.Node().String())
	default:
		urls = append(urls, ln.Node().String())
	}
	bootstrapNodes := []*enode.Node{}
	for _, url := range urls {
		if url != "" {
			node, err := enode.Parse(enode.ValidSchemes, url)
			if err != nil {
				log.Crit("Bootstrap URL invalid", "enode", url, "err", err)
				continue
			}
			bootstrapNodes = append(bootstrapNodes, node)
		}
	}
	return bootstrapNodes
}

func Genesis() *core.Genesis {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		return AmanaGenesis()
	case protocol.TestNet:
		return AmanaTestnetGenesis()
	case protocol.MixNet:
		return AmanaMixnetGenesis()
	case protocol.PrivNet:
		return AmanaPrivnetGenesis()
	}
	return nil
}
