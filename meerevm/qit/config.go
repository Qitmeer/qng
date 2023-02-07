package qit

import (
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/protocol"
	mcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/eth"
	mconsensus "github.com/Qitmeer/qng/meerevm/qit/consensus"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"math/big"
	"net"
	"path/filepath"
)

var (
	ClientIdentifier       = "qit"
	chainID          int64 = 223
	nodeFlags              = mcommon.Merge([]cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.MinFreeDiskSpaceFlag,
		utils.KeyStoreDirFlag,
		utils.ExternalSignerFlag,
		utils.NoUSBFlag,
		utils.USBFlag,
		utils.SmartCardDaemonPathFlag,
		utils.OverrideTerminalTotalDifficulty,
		utils.OverrideTerminalTotalDifficultyPassed,
		utils.EthashCacheDirFlag,
		utils.EthashCachesInMemoryFlag,
		utils.EthashCachesOnDiskFlag,
		utils.EthashCachesLockMmapFlag,
		utils.EthashDatasetDirFlag,
		utils.EthashDatasetsInMemoryFlag,
		utils.EthashDatasetsOnDiskFlag,
		utils.EthashDatasetsLockMmapFlag,
		utils.TxPoolLocalsFlag,
		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,
		utils.SyncModeFlag,
		utils.ExitWhenSyncedFlag,
		utils.GCModeFlag,
		utils.SnapshotFlag,
		utils.TxLookupLimitFlag,
		utils.LightServeFlag,
		utils.LightIngressFlag,
		utils.LightEgressFlag,
		utils.LightMaxPeersFlag,
		utils.LightNoPruneFlag,
		utils.LightKDFFlag,
		utils.UltraLightServersFlag,
		utils.UltraLightFractionFlag,
		utils.UltraLightOnlyAnnounceFlag,
		utils.LightNoSyncServeFlag,
		utils.EthRequiredBlocksFlag,
		utils.LegacyWhitelistFlag,
		utils.BloomFilterSizeFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheTrieFlag,
		utils.CacheTrieJournalFlag,
		utils.CacheTrieRejournalFlag,
		utils.CacheGCFlag,
		utils.CacheSnapshotFlag,
		utils.CacheNoPrefetchFlag,
		utils.CachePreimagesFlag,
		utils.CacheLogSizeFlag,
		utils.FDLimitFlag,
		utils.ListenPortFlag,
		utils.DiscoveryPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.MiningEnabledFlag,
		utils.MinerThreadsFlag,
		utils.MinerNotifyFlag,
		utils.LegacyMinerGasTargetFlag,
		utils.MinerGasLimitFlag,
		utils.MinerGasPriceFlag,
		utils.MinerEtherbaseFlag,
		utils.MinerExtraDataFlag,
		utils.MinerRecommitIntervalFlag,
		utils.MinerNoVerifyFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.DNSDiscoveryFlag,
		utils.DeveloperFlag,
		utils.DeveloperPeriodFlag,
		utils.DeveloperGasLimitFlag,
		utils.VMEnableDebugFlag,
		utils.NetworkIdFlag,
		utils.EthStatsURLFlag,
		utils.FakePoWFlag,
		utils.NoCompactionFlag,
		utils.GpoBlocksFlag,
		utils.GpoPercentileFlag,
		utils.GpoMaxGasPriceFlag,
		utils.GpoIgnoreGasPriceFlag,
		utils.MinerNotifyFullFlag,
		utils.IgnoreLegacyReceiptsFlag,
		eth.ConfigFileFlag,
	}, utils.NetworkFlags, utils.DatabasePathFlags)

	rpcFlags = []cli.Flag{
		utils.HTTPEnabledFlag,
		utils.HTTPListenAddrFlag,
		utils.HTTPPortFlag,
		utils.HTTPCORSDomainFlag,
		utils.AuthListenFlag,
		utils.AuthPortFlag,
		utils.AuthVirtualHostsFlag,
		utils.JWTSecretFlag,
		utils.HTTPVirtualHostsFlag,
		utils.GraphQLEnabledFlag,
		utils.GraphQLCORSDomainFlag,
		utils.GraphQLVirtualHostsFlag,
		utils.HTTPApiFlag,
		utils.HTTPPathPrefixFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.WSPathPrefixFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.InsecureUnlockAllowedFlag,
		utils.RPCGlobalGasCapFlag,
		utils.RPCGlobalEVMTimeoutFlag,
		utils.RPCGlobalTxFeeCapFlag,
		utils.AllowUnprotectedTxs,
	}

	metricsFlags = []cli.Flag{
		utils.MetricsEnabledFlag,
		utils.MetricsEnabledExpensiveFlag,
		utils.MetricsHTTPFlag,
		utils.MetricsPortFlag,
		utils.MetricsEnableInfluxDBFlag,
		utils.MetricsInfluxDBEndpointFlag,
		utils.MetricsInfluxDBDatabaseFlag,
		utils.MetricsInfluxDBUsernameFlag,
		utils.MetricsInfluxDBPasswordFlag,
		utils.MetricsInfluxDBTagsFlag,
		utils.MetricsEnableInfluxDBV2Flag,
		utils.MetricsInfluxDBTokenFlag,
		utils.MetricsInfluxDBBucketFlag,
		utils.MetricsInfluxDBOrganizationFlag,
	}

	chainConfig = &params.ChainConfig{
		ChainID:             big.NewInt(chainID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   big.NewInt(0),
		GrayGlacierBlock:    big.NewInt(0),
		Clique: &params.CliqueConfig{
			Period: 3,
			Epoch:  100,
		},
	}
)

func MakeConfig(datadir string) (*eth.Config, error) {
	chainConfig.ChainID = big.NewInt(qparams.ActiveNetParams.MeerEVMCfg.QitChainID)
	genesis := DefaultGenesisBlock(chainConfig)

	econfig := ethconfig.Defaults

	econfig.NetworkId = uint64(qparams.ActiveNetParams.MeerEVMCfg.QitChainID)
	econfig.Genesis = genesis
	econfig.ConsensusEngine = createConsensusEngine

	nodeConf := node.DefaultConfig
	nodeConf.DataDir = datadir
	nodeConf.Name = ClientIdentifier
	nodeConf.Version = params.VersionWithMeta
	nodeConf.HTTPModules = append(nodeConf.HTTPModules, "eth")
	nodeConf.WSModules = append(nodeConf.WSModules, "eth")
	nodeConf.IPCPath = "qit.ipc"
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

func MakeParams(cfg *config.Config) (*eth.Config, []string, []cli.Flag, error) {
	ecfg, err := MakeConfig(cfg.DataDir)
	if err != nil {
		return ecfg, nil, nil, err
	}
	return ecfg, mcommon.ProcessEnv(cfg.QitEnv, ecfg.Node.Name), GetFlags(), nil
}

func GetFlags() []cli.Flag {
	flags := []cli.Flag{}
	flags = append(flags, nodeFlags...)
	flags = append(flags, rpcFlags...)
	flags = append(flags, metricsFlags...)
	return flags
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

func createConsensusEngine(stack *node.Node, chainConfig *params.ChainConfig, config *ethash.Config, notify []string, noverify bool, db ethdb.Database) consensus.Engine {
	engine := mconsensus.New(chainConfig.Clique, db)
	return engine
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
		urls = append(urls, "enr:-KO4QHVyY3Ph0LcReHgyNyqFrGsoPp5tgr6_vSDmcXueeLqARDPBI5VY2QSdDdD19YG5bxRPJZzFE9bY9mVn00viYGqGAYWak76qg2V0aMfGhE5aFCGAgmlkgnY0gmlwhC0gC3eJc2VjcDI1NmsxoQK6ou0UT10hfVO8-b-2zWawj1_l8xflj764RASTmM8f64RzbmFwwIN0Y3CCSGCDdWRwgkhg")
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
