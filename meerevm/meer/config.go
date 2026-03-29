package meer

import (
	"errors"
	"math/big"
	"path/filepath"
	"time"

	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/address"
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
	"github.com/ethereum/go-ethereum/params"
)

var (
	// ClientIdentifier is a hard coded identifier to report into the network.
	ClientIdentifier = "meereth"

	exclusionFlags = utils.NetworkFlags
)

func MakeConfig(cfg *config.Config) (*eth.Config, error) {
	datadir := cfg.DataDir
	genesis := CurrentGenesis()
	if genesis == nil {
		return nil, errors.New("no genesis config")
	}

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
	if cfg.GetMinningAddr() != nil {
		pka, ok := cfg.GetMinningAddr().(*address.SecpPubKeyAddress)
		if ok {
			mea, err := mcommon.NewMeerEVMAddress(pka.PubKey().SerializeUncompressed())
			if err != nil {
				return nil, err
			}
			econfig.Miner.PendingFeeRecipient = mea
		}
	}

	nodeConf := node.DefaultConfig

	nodeConf.DataDir = datadir
	nodeConf.Name = ClientIdentifier
	nodeConf.Version = WithMeta
	nodeConf.HTTPModules = append(nodeConf.HTTPModules, "eth")
	nodeConf.WSModules = append(nodeConf.WSModules, "eth")
	if !cfg.DisableRPC {
		nodeConf.IPCPath = ClientIdentifier + ".ipc"
	}

	if len(datadir) > 0 {
		nodeConf.KeyStoreDir = filepath.Join(datadir, "keystore")
	}

	nodeConf.HTTPPort, nodeConf.WSPort, nodeConf.AuthPort = getDefaultPort()
	nodeConf.P2P.ListenAddr = ""
	nodeConf.P2P.NoDial = true
	nodeConf.P2P.NoDiscovery = true
	nodeConf.P2P.DiscoveryV4 = false
	nodeConf.P2P.DiscoveryV5 = false
	nodeConf.P2P.NAT = nil

	pk, err := common.PrivateKey(cfg.DataDir, "", 0600)
	if err != nil {
		return nil, err
	}
	nodeConf.P2P.PrivateKey, err = common.ToECDSAPrivKey(pk)
	if err != nil {
		return nil, err
	}

	metrics.DefaultConfig.InfluxDBDatabase = "qng"
	metrics.DefaultConfig.InfluxDBBucket = "qng"
	metrics.DefaultConfig.InfluxDBOrganization = "qng"

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

func getDefaultPort() (int, int, int) {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		return 8535, 8536, 8537
	case protocol.TestNet:
		return 18535, 18536, 18537
	case protocol.MixNet:
		return 28535, 28536, 28537
	case protocol.PrivNet:
		return 38535, 38536, 38537
	default:
		return 48535, 48536, 48537
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
	if qparams.ActiveNetParams.MeerGenesis != nil {
		gen := Genesis(qparams.ActiveNetParams.Params, types.GenesisAlloc{})
		if len(qparams.ActiveNetParams.MeerGenesis.ExtraData) > 0 {
			gen.ExtraData = qparams.ActiveNetParams.MeerGenesis.ExtraData
		}
		if len(qparams.ActiveNetParams.MeerGenesis.Alloc) > 0 {
			gen.Alloc = qparams.ActiveNetParams.MeerGenesis.Alloc
		}
		if qparams.ActiveNetParams.MeerGenesis.Config != nil {
			gen.Config = qparams.ActiveNetParams.MeerGenesis.Config
		}
		return gen
	}
	return Genesis(qparams.ActiveNetParams.Params, nil)
}
