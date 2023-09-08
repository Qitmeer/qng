package meer

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/protocol"
	mcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/eth"
	mconsensus "github.com/Qitmeer/qng/meerevm/meer/consensus"
	mparams "github.com/Qitmeer/qng/meerevm/params"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"net"
	"path/filepath"
)

var (
	// ClientIdentifier is a hard coded identifier to report into the network.
	ClientIdentifier = "meereth"

	exclusionFlags = append([]cli.Flag{
		utils.TxPoolLocalsFlag,
		utils.TxPoolNoLocalsFlag,
		utils.SyncTargetFlag,
		utils.DiscoveryPortFlag,
		utils.MiningEnabledFlag,
		utils.MinerEtherbaseFlag,
		utils.MinerNewPayloadTimeout,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.DNSDiscoveryFlag,
	}, utils.NetworkFlags...)
)

func MakeConfig(datadir string) (*eth.Config, error) {
	genesis := Genesis()

	etherbase := common.Address{}
	econfig := ethconfig.Defaults

	econfig.NetworkId = genesis.Config.ChainID.Uint64()
	econfig.Genesis = genesis
	econfig.NoPruning = false
	econfig.SkipBcVersionCheck = false
	econfig.ConsensusEngine = createConsensusEngine

	econfig.Miner.Etherbase = etherbase
	econfig.Miner.ExtraData = []byte{byte(0)}
	econfig.Miner.External = &MeerPool{}

	econfig.TxPool.NoLocals = false

	nodeConf := node.DefaultConfig

	nodeConf.DataDir = datadir
	nodeConf.Name = ClientIdentifier
	nodeConf.Version = params.VersionWithMeta
	nodeConf.HTTPModules = append(nodeConf.HTTPModules, "eth")
	nodeConf.WSModules = append(nodeConf.WSModules, "eth")
	nodeConf.IPCPath = ""
	nodeConf.KeyStoreDir = filepath.Join(datadir, "keystore")
	//nodeConf.HTTPHost = node.DefaultHTTPHost
	//nodeConf.WSHost = node.DefaultWSHost
	nodeConf.HTTPPort, nodeConf.WSPort, nodeConf.AuthPort = getDefaultRPCPort()

	nodeConf.P2P.MaxPeers = 0
	nodeConf.P2P.DiscoveryV5 = false
	nodeConf.P2P.NoDiscovery = true
	nodeConf.P2P.NoDial = true
	nodeConf.P2P.ListenAddr = ""
	nodeConf.P2P.NAT = nil

	db, _ := enode.OpenDB("")
	key, _ := crypto.GenerateKey()
	ln := enode.NewLocalNode(db, key)
	ln.SetFallbackIP(net.IP{127, 0, 0, 1})
	ln.SetFallbackUDP(8538)
	nodeConf.P2P.BootstrapNodes = []*enode.Node{ln.Node()}
	//
	return &eth.Config{
		Eth:     econfig,
		Node:    nodeConf,
		Metrics: metrics.DefaultConfig,
	}, nil
}

func MakeParams(cfg *config.Config) (*eth.Config, []string, error) {
	ecfg, err := MakeConfig(cfg.DataDir)
	if err != nil {
		return ecfg, nil, err
	}
	args, err := mcommon.ProcessEnv(cfg.EVMEnv, ecfg.Node.Name, exclusionFlags)
	return ecfg, args, err
}

func getDefaultRPCPort() (int, int, int) {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		return 8535, 8536, 8537
	case protocol.TestNet:
		return 18535, 18536, 18537
	case protocol.MixNet:
		return 28535, 28536, 28537
	default:
		return 38535, 38536, 38537
	}
}

func createConsensusEngine(config *params.ChainConfig, db ethdb.Database) (consensus.Engine, error) {
	return mconsensus.New(), nil
}

func ChainConfig() *params.ChainConfig {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		return mparams.QngMainnetChainConfig
	case protocol.TestNet:
		return mparams.QngTestnetChainConfig
	case protocol.MixNet:
		return mparams.QngMixnetChainConfig
	case protocol.PrivNet:
		return mparams.QngPrivnetChainConfig
	}
	return nil
}

func Genesis() *core.Genesis {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		return QngGenesis()
	case protocol.TestNet:
		return QngTestnetGenesis()
	case protocol.MixNet:
		return QngMixnetGenesis()
	case protocol.PrivNet:
		return QngPrivnetGenesis()
	}
	return nil
}
