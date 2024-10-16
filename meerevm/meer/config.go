package meer

import (
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/protocol"
	mcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/eth"
	mconsensus "github.com/Qitmeer/qng/meerevm/meer/consensus"
	"github.com/Qitmeer/qng/p2p/common"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"path/filepath"
	"time"
)

var (
	// ClientIdentifier is a hard coded identifier to report into the network.
	ClientIdentifier = "meereth"

	exclusionFlags = utils.NetworkFlags
)

func MakeConfig(cfg *config.Config) (*eth.Config, error) {
	datadir := cfg.DataDir
	genesis := CurrentGenesis()

	econfig := ethconfig.Defaults

	econfig.NetworkId = genesis.Config.ChainID.Uint64()
	econfig.Genesis = genesis
	econfig.NoPruning = false
	econfig.SkipBcVersionCheck = false
	econfig.ConsensusEngine = createConsensusEngine

	if cfg.EVMTrieTimeout > 0 {
		econfig.TrieTimeout = time.Second * time.Duration(cfg.EVMTrieTimeout)
	}
	if len(cfg.StateScheme) > 0 {
		econfig.StateScheme = cfg.StateScheme
	}

	nodeConf := node.DefaultConfig

	nodeConf.DataDir = datadir
	nodeConf.Name = ClientIdentifier
	nodeConf.Version = params.VersionWithMeta
	nodeConf.HTTPModules = append(nodeConf.HTTPModules, "eth")
	nodeConf.WSModules = append(nodeConf.WSModules, "eth")
	if !cfg.DisableRPC {
		nodeConf.IPCPath = ClientIdentifier + ".ipc"
	}

	if len(datadir) > 0 {
		nodeConf.KeyStoreDir = filepath.Join(datadir, "keystore")
	}

	var p2pPort int
	nodeConf.HTTPPort, nodeConf.WSPort, nodeConf.AuthPort, p2pPort = getDefaultPort()
	if !cfg.DisableListen {
		nodeConf.P2P.ListenAddr = fmt.Sprintf(":%d", p2pPort)
		nodeConf.P2P.BootstrapNodes = getBootstrapNodes(p2pPort)

		pk, err := common.PrivateKey(cfg.DataDir, "", 0600)
		if err != nil {
			return nil, err
		}
		nodeConf.P2P.PrivateKey, err = common.ToECDSAPrivKey(pk)
		if err != nil {
			return nil, err
		}
	} else {
		nodeConf.P2P.ListenAddr = ""
		nodeConf.P2P.MaxPeers = 0
		nodeConf.P2P.NAT = nil
	}

	return &eth.Config{
		Eth:     econfig,
		Node:    nodeConf,
		Metrics: metrics.DefaultConfig,
	}, nil
}

func MakeParams(cfg *config.Config) (*eth.Config, []string, error) {
	ecfg, err := MakeConfig(cfg)
	if err != nil {
		return ecfg, nil, err
	}
	args, err := mcommon.ProcessEnv(cfg.EVMEnv, ecfg.Node.Name, exclusionFlags)
	return ecfg, args, err
}

func getDefaultPort() (int, int, int, int) {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		return 8535, 8536, 8537, 8538
	case protocol.TestNet:
		return 18535, 18536, 18537, 18538
	case protocol.MixNet:
		return 28535, 28536, 28537, 28538
	default:
		return 38535, 38536, 38537, 38538
	}
}

func createConsensusEngine(config *params.ChainConfig, db ethdb.Database) (consensus.Engine, error) {
	return mconsensus.New(), nil
}

func Genesis(net *qparams.Params, alloc types.GenesisAlloc) *core.Genesis {
	if alloc == nil {
		alloc = DecodeAlloc(net)
	}
	gen := &core.Genesis{
		Config:     net.MeerConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      alloc,
		Timestamp:  uint64(net.GenesisBlock.Block().Header.Timestamp.Unix()),
	}
	if net.Net == protocol.TestNet {
		gen.GasLimit = 8000000
	}
	return gen
}

func CurrentGenesis() *core.Genesis {
	return Genesis(qparams.ActiveNetParams.Params, nil)
}

func getBootstrapNodes(port int) []*enode.Node {
	urls := []string{}
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		urls = MainnetBootnodes
	case protocol.TestNet:
		urls = TestnetBootnodes
	case protocol.MixNet:
		urls = MixnetBootnodes
	case protocol.PrivNet:
		urls = PrivnetBootnodes
	}
	return eth.GetBootstrapNodes(port, urls)
}
