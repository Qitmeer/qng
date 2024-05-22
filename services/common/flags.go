// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2013-2016 The btcsuite developers

package common

import (
	"path/filepath"
	"time"

	"github.com/Qitmeer/qng/common/profiling"
	"github.com/Qitmeer/qng/common/util"
	"github.com/Qitmeer/qng/config"
	"github.com/urfave/cli/v2"
)

const (
	defaultConfigFilename    = "qng.conf"
	defaultDataDirname       = "data"
	defaultLogLevel          = "info"
	defaultDebugPrintOrigins = false
	defaultLogDirname        = "logs"
	defaultLogFilename       = "qng.log"
	defaultGenerate          = false
	defaultBlockMinSize      = 0
	// 122880 = 120 KB (120*1024)
	defaultBlockMaxSize           = 122880
	defaultMaxRPCClients          = 10
	defaultMaxRPCWebsockets       = 25
	defaultMaxRPCConcurrentReqs   = 20
	defaultMaxPeers               = 50
	defaultMiningStateSync        = false
	defaultMaxInboundPeersPerHost = 25 // The default max total of inbound peer for host
	defaultTrickleInterval        = 10 * time.Second
	defaultInvalidTxIndex         = false
	defaultTxhashIndex            = false
	defaultMempoolExpiry          = int64(time.Hour)
	defaultRPCUser                = "test"
	defaultRPCPass                = "test"
	defaultMinBlockPruneSize      = 2000
	defaultMinBlockDataCache      = 2000
	defaultMinRelayTxFee          = int64(1e4)
	defaultObsoleteHeight         = 5
	defaultGBTTimeout             = 1000 // default gbt timeout 1s = 1000ms
)
const (
	defaultSigCacheMaxSize = 100000
)
const (
	defaultMaxOrphanTxSize = 5000
)

var (
	defaultHomeDir        = util.AppDataDir("qng", false)
	defaultConfigFile     = filepath.Join(defaultHomeDir, defaultConfigFilename)
	defaultDataDir        = filepath.Join(defaultHomeDir, defaultDataDirname)
	defaultDbType         = "ffldb"
	defaultLogDir         = filepath.Join(defaultHomeDir, defaultLogDirname)
	defaultLogRotatorSize = int64(1024 * 10)
	defaultRPCKeyFile     = filepath.Join(defaultHomeDir, "rpc.key")
	defaultRPCCertFile    = filepath.Join(defaultHomeDir, "rpc.cert")
	defaultDAGType        = "phantom"
)

var (
	cfg = DefaultConfig("")

	RPCListeners      cli.StringSlice
	Modules           cli.StringSlice
	MiningAddrs       cli.StringSlice
	BlockMinSize      uint
	BlockMaxSize      uint
	BlockPrioritySize uint
	AddPeers          cli.StringSlice
	BootstrapNodes    cli.StringSlice
	Whitelist         cli.StringSlice
	Blacklist         cli.StringSlice
	GBTNotify         cli.StringSlice

	Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "appdata",
			Aliases:     []string{"A"},
			Usage:       "Path to application home directory",
			Value:       defaultHomeDir,
			Destination: &cfg.HomeDir,
		},
		&cli.BoolFlag{
			Name:        "ShowVersion",
			Aliases:     []string{"V"},
			Usage:       "Display version information and exit",
			Destination: &cfg.ShowVersion,
		},
		&cli.StringFlag{
			Name:        "configfile",
			Aliases:     []string{"C"},
			Usage:       "Path to configuration file",
			Value:       defaultConfigFile,
			Destination: &cfg.ConfigFile,
		},
		&cli.StringFlag{
			Name:        "datadir",
			Aliases:     []string{"b"},
			Usage:       "Directory to store data",
			Value:       defaultDataDir,
			Destination: &cfg.DataDir,
		},
		&cli.StringFlag{
			Name:        "logdir",
			Usage:       "Directory to log output.",
			Value:       defaultLogDir,
			Destination: &cfg.LogDir,
		},
		&cli.Int64Flag{
			Name:        "logrotatorsize",
			Usage:       "Directory to log output.",
			Value:       defaultLogRotatorSize,
			Destination: &cfg.LogRotatorSize,
		},
		&cli.BoolFlag{
			Name:        "nofilelogging",
			Usage:       "Disable file logging.",
			Destination: &cfg.NoFileLogging,
		},
		&cli.StringFlag{
			Name:        "listen",
			Usage:       "Add an IP to listen for connections",
			Destination: &cfg.Listener,
		},
		&cli.StringFlag{
			Name:        "port",
			Usage:       "Default p2p port.",
			Destination: &cfg.DefaultPort,
		},
		&cli.StringSliceFlag{
			Name:        "rpclisten",
			Usage:       "Add an interface/port to listen for RPC connections",
			Destination: &RPCListeners,
		},
		&cli.IntFlag{
			Name:        "maxpeers",
			Usage:       "Max number of inbound and outbound peers",
			Value:       defaultMaxPeers,
			Destination: &cfg.MaxPeers,
		},
		&cli.BoolFlag{
			Name:        "nolisten",
			Usage:       "Disable listening for incoming connections",
			Destination: &cfg.DisableListen,
		},
		&cli.StringFlag{
			Name:        "rpcuser",
			Aliases:     []string{"u"},
			Usage:       "Username for RPC connections",
			Value:       defaultRPCUser,
			Destination: &cfg.RPCUser,
		},
		&cli.StringFlag{
			Name:        "rpcpass",
			Aliases:     []string{"P"},
			Usage:       "Password for RPC connections",
			Value:       defaultRPCPass,
			Destination: &cfg.RPCPass,
		},
		&cli.StringFlag{
			Name:        "rpccert",
			Usage:       "File containing the certificate file",
			Value:       defaultRPCCertFile,
			Destination: &cfg.RPCCert,
		},
		&cli.StringFlag{
			Name:        "rpckey",
			Usage:       "File containing the certificate key",
			Value:       defaultRPCKeyFile,
			Destination: &cfg.RPCKey,
		},
		&cli.IntFlag{
			Name:        "rpcmaxclients",
			Usage:       "Max number of RPC clients for standard connections",
			Value:       defaultMaxRPCClients,
			Destination: &cfg.RPCMaxClients,
		},
		&cli.BoolFlag{
			Name:        "norpc",
			Usage:       "Disable built-in RPC server -- NOTE: The RPC server is disabled by default if no rpcuser/rpcpass or rpclimituser/rpclimitpass is specified",
			Destination: &cfg.DisableRPC,
		},
		&cli.BoolFlag{
			Name:        "notls",
			Usage:       "Disable TLS for the RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost",
			Destination: &cfg.DisableTLS,
		},
		&cli.StringSliceFlag{
			Name:        "modules",
			Usage:       "Modules is a list of API modules(See GetNodeInfo) to expose via the HTTP RPC interface. If the module list is empty, all RPC API endpoints designated public will be exposed.",
			Destination: &Modules,
		},
		&cli.BoolFlag{
			Name:        "nocheckpoints",
			Usage:       "Disable built-in checkpoints.  Don't do this unless you know what you're doing.",
			Destination: &cfg.DisableCheckpoints,
		},
		&cli.BoolFlag{
			Name:        "addrindex",
			Usage:       "Maintain a full address-based transaction index which makes the getrawtransactions RPC available",
			Destination: &cfg.AddrIndex,
		},
		&cli.BoolFlag{
			Name:        "dropaddrindex",
			Usage:       "Deletes the address-based transaction index from the database on start up and then exits.",
			Destination: &cfg.DropAddrIndex,
		},
		&cli.BoolFlag{
			Name:        "light",
			Usage:       "start as a qitmeer light node",
			Destination: &cfg.LightNode,
		},
		&cli.UintFlag{
			Name:        "sigcachemaxsize",
			Usage:       "The maximum number of entries in the signature verification cache",
			Value:       defaultSigCacheMaxSize,
			Destination: &cfg.SigCacheMaxSize,
		},
		&cli.BoolFlag{
			Name:        "testnet",
			Usage:       "Use the test network",
			Destination: &cfg.TestNet,
		},
		&cli.BoolFlag{
			Name:        "mixnet",
			Usage:       "Use the test mix pow network",
			Destination: &cfg.MixNet,
		},
		&cli.BoolFlag{
			Name:        "privnet",
			Usage:       "Use the private network",
			Destination: &cfg.PrivNet,
		},
		&cli.StringFlag{
			Name:        "dbtype",
			Usage:       "Database backend to use for the Block Chain",
			Value:       defaultDbType,
			Destination: &cfg.DbType,
		},
		&cli.StringFlag{
			Name:        "profile",
			Usage:       "Enable HTTP profiling on given [addr:]port -- NOTE port must be between 1024 and 65536",
			Destination: &cfg.Profile,
		},
		&cli.StringFlag{
			Name:        "cpuprofile",
			Usage:       "Write CPU profile to the specified file",
			Destination: &cfg.CPUProfile,
		},
		&cli.BoolFlag{
			Name:        "trackheap",
			Usage:       "tracks the size of the heap and dumps a profile",
			Destination: &cfg.TrackHeap,
		},
		&cli.IntFlag{
			Name:        "trackheaplimit",
			Usage:       "track heap when limit in gigabytes (default:7G)",
			Destination: &cfg.TrackHeapLimit,
			Value:       profiling.DefaultTrackHeapLimit,
		},
		&cli.StringFlag{
			Name:        "debuglevel",
			Aliases:     []string{"d"},
			Usage:       "Logging level {trace, debug, info, warn, error, critical}",
			Value:       defaultLogLevel,
			Destination: &cfg.DebugLevel,
		},
		&cli.BoolFlag{
			Name:        "printorigin",
			Usage:       "Print log debug location (file:line)",
			Destination: &cfg.DebugPrintOrigins,
		},
		&cli.BoolFlag{
			Name:        "norelaypriority",
			Usage:       "Do not require free or low-fee transactions to have high priority for relaying",
			Destination: &cfg.NoRelayPriority,
		},
		&cli.Float64Flag{
			Name:        "limitfreerelay",
			Usage:       "Limit relay of transactions with no transaction fee to the given amount in thousands of bytes per minute",
			Destination: &cfg.FreeTxRelayLimit,
		},
		&cli.BoolFlag{
			Name:        "acceptnonstd",
			Usage:       "Accept and relay non-standard transactions to the network regardless of the default settings for the active network.",
			Destination: &cfg.AcceptNonStd,
			Value:       true,
		},
		&cli.IntFlag{
			Name:        "maxorphantx",
			Usage:       "Max number of orphan transactions to keep in memory",
			Destination: &cfg.MaxOrphanTxs,
		},
		&cli.Int64Flag{
			Name:        "mintxfee",
			Usage:       "The minimum transaction fee in AtomMEER/kB.",
			Value:       defaultMinRelayTxFee,
			Destination: &cfg.MinTxFee,
		},
		&cli.Int64Flag{
			Name:        "mempoolexpiry",
			Usage:       "Do not keep transactions in the mempool more than mempoolexpiry",
			Value:       defaultMempoolExpiry,
			Destination: &cfg.MempoolExpiry,
		},
		&cli.BoolFlag{
			Name:        "persistmempool",
			Usage:       "Whether to save the mempool on shutdown and load on restart",
			Destination: &cfg.Persistmempool,
		},
		&cli.BoolFlag{
			Name:        "nomempoolbar",
			Usage:       "Whether to show progress bar when load mempool from file",
			Destination: &cfg.NoMempoolBar,
		},
		&cli.BoolFlag{
			Name:        "miner",
			Usage:       "Enable miner module",
			Destination: &cfg.Miner,
		},
		&cli.BoolFlag{
			Name:        "generate",
			Usage:       "Generate (mine) coins using the CPU",
			Destination: &cfg.Generate,
		},
		&cli.StringSliceFlag{
			Name:        "miningaddr",
			Usage:       "Add the specified payment address to the list of addresses to use for generated blocks -- At least one address is required if the generate option is set",
			Destination: &MiningAddrs,
		},
		&cli.IntFlag{
			Name:        "miningtimeoffset",
			Usage:       "Offset the mining timestamp of a block by this many seconds (positive values are in the past)",
			Destination: &cfg.MiningTimeOffset,
		},
		&cli.UintFlag{
			Name:        "blockminsize",
			Usage:       "Mininum block size in bytes to be used when creating a block",
			Value:       defaultBlockMinSize,
			Destination: &BlockMinSize,
		},
		&cli.UintFlag{
			Name:        "blockmaxsize",
			Usage:       "Maximum block size in bytes to be used when creating a block",
			Value:       defaultBlockMaxSize,
			Destination: &BlockMaxSize,
		},
		&cli.UintFlag{
			Name:        "blockprioritysize",
			Usage:       "Size in bytes for high-priority/low-fee transactions when creating a block",
			Destination: &BlockPrioritySize,
		},
		&cli.IntFlag{
			Name:        "rpcmaxwebsockets",
			Usage:       "Max number of RPC websocket connections",
			Value:       defaultMaxRPCWebsockets,
			Destination: &cfg.RPCMaxWebsockets,
		},
		&cli.IntFlag{
			Name:        "rpcmaxconcurrentreqs",
			Usage:       "Max number of concurrent RPC requests that may be processed concurrently",
			Value:       defaultMaxRPCConcurrentReqs,
			Destination: &cfg.RPCMaxConcurrentReqs,
		},
		&cli.BoolFlag{
			Name:        "blocksonly",
			Usage:       "Do not accept transactions from remote peers",
			Destination: &cfg.BlocksOnly,
		},
		&cli.BoolFlag{
			Name:        "miningstatesync",
			Usage:       "Synchronizing the mining state with other nodes",
			Destination: &cfg.MiningStateSync,
		},
		&cli.StringSliceFlag{
			Name:        "addpeer",
			Aliases:     []string{"a"},
			Usage:       "Add a peer to connect with at startup",
			Destination: &AddPeers,
		},
		&cli.BoolFlag{
			Name:        "upnp",
			Usage:       "Use UPnP to map our listening port outside of NAT",
			Destination: &cfg.Upnp,
		},
		&cli.IntFlag{
			Name:        "maxinbound",
			Usage:       "The max total of inbound peer for host",
			Value:       defaultMaxInboundPeersPerHost,
			Destination: &cfg.MaxInbound,
		},
		&cli.BoolFlag{
			Name:        "banning",
			Usage:       "Enable banning of misbehaving peers",
			Destination: &cfg.Banning,
			Value:       true,
		},
		&cli.StringFlag{
			Name:        "dagtype",
			Aliases:     []string{"G"},
			Usage:       "DAG type {phantom,spectre}",
			Value:       defaultDAGType,
			Destination: &cfg.DAGType,
		},
		&cli.BoolFlag{
			Name:        "cleanup",
			Aliases:     []string{"L"},
			Usage:       "Cleanup the block database",
			Destination: &cfg.Cleanup,
		},
		&cli.BoolFlag{
			Name:        "buildledger",
			Usage:       "Generate the genesis ledger for the next qitmeer version",
			Destination: &cfg.BuildLedger,
		},
		&cli.StringFlag{
			Name:        "zmqpubhashblock",
			Usage:       "Enable publish hash block  in <address>",
			Destination: &cfg.Zmqpubhashblock,
		},
		&cli.StringFlag{
			Name:        "zmqpubrawblock",
			Usage:       "Enable publish raw block in <address>",
			Destination: &cfg.Zmqpubrawblock,
		},
		&cli.StringFlag{
			Name:        "zmqpubhashtx",
			Usage:       "Enable publish hash transaction in <address>",
			Destination: &cfg.Zmqpubhashtx,
		},
		&cli.StringFlag{
			Name:        "zmqpubrawtx",
			Usage:       "Enable publish raw transaction in <address>",
			Destination: &cfg.Zmqpubrawtx,
		},
		&cli.BoolFlag{
			Name:        "invalidtxindex",
			Usage:       "invalid transaction index.",
			Destination: &cfg.InvalidTxIndex,
		},
		&cli.BoolFlag{
			Name:        "txhashindex",
			Usage:       "Cache transaction full hash.",
			Destination: &cfg.TxHashIndex,
		},
		&cli.BoolFlag{
			Name:        "ntp",
			Usage:       "Auto sync time.",
			Destination: &cfg.NTP,
		},
		&cli.StringSliceFlag{
			Name:        "bootstrapnode",
			Usage:       "The address of bootstrap node.",
			Destination: &BootstrapNodes,
		},
		&cli.BoolFlag{
			Name:        "nodiscovery",
			Usage:       "Enable only local network p2p and do not connect to cloud bootstrap nodes.",
			Destination: &cfg.NoDiscovery,
		},
		&cli.StringFlag{
			Name:        "metadatadir",
			Usage:       "meta data dir for p2p",
			Destination: &cfg.MetaDataDir,
		},
		&cli.IntFlag{
			Name:        "p2pudpport",
			Usage:       "The udp port used by P2P",
			Destination: &cfg.P2PUDPPort,
		},
		&cli.IntFlag{
			Name:        "p2ptcpport",
			Usage:       "The tcp port used by P2P.",
			Destination: &cfg.P2PTCPPort,
		},
		&cli.StringFlag{
			Name:        "externalip",
			Usage:       "The IP address advertised by libp2p. This may be used to advertise an external IP.",
			Destination: &cfg.HostIP,
		},
		&cli.StringFlag{
			Name:        "externaldns",
			Usage:       "The DNS address advertised by libp2p. This may be used to advertise an external DNS.",
			Destination: &cfg.HostDNS,
		},
		&cli.StringFlag{
			Name:        "relaynode",
			Usage:       "The address of relay node that routes traffic between two peers over a qitmeer “relay” peer.",
			Destination: &cfg.RelayNode,
		},
		&cli.StringSliceFlag{
			Name:        "whitelist",
			Usage:       "Add an IP network or IP,PeerID that will not be banned or ignore dual channel mode detection. (eg. 192.168.1.0/24 or ::1 or [peer id])",
			Destination: &Whitelist,
		},
		&cli.StringSliceFlag{
			Name:        "blacklist",
			Usage:       "Add some IP network or IP that will be banned. (eg. 192.168.1.0/24 or ::1)",
			Destination: &Blacklist,
		},
		&cli.IntFlag{
			Name:        "maxbadresp",
			Usage:       "maxbadresp is the maximum number of bad responses from a peer before we stop talking to it.",
			Destination: &cfg.MaxBadResp,
		},
		&cli.BoolFlag{
			Name:        "circuit",
			Usage:       "All peers will ignore dual channel mode detection",
			Destination: &cfg.Circuit,
			Value:       true,
		},
		&cli.StringFlag{
			Name:        "evmenv",
			Usage:       "meer EVM environment",
			Destination: &cfg.EVMEnv,
		},
		&cli.BoolFlag{
			Name:        "estimatefee",
			Usage:       "Enable estimate fee",
			Destination: &cfg.Estimatefee,
		},
		&cli.StringSliceFlag{
			Name:        "gbtnotify",
			Usage:       "HTTP URL list to be notified of new block template",
			Destination: &GBTNotify,
		},
		&cli.BoolFlag{
			Name:        "acctmode",
			Usage:       "Enable support account system mode",
			Destination: &cfg.AcctMode,
		},
		&cli.Uint64Flag{
			Name:        "dagcachesize",
			Usage:       "DAG block cache size",
			Value:       defaultMinBlockPruneSize,
			Destination: &cfg.DAGCacheSize,
		},
		&cli.Uint64Flag{
			Name:        "bdcachesize",
			Usage:       "Block data cache size",
			Value:       defaultMinBlockDataCache,
			Destination: &cfg.BlockDataCacheSize,
		},
		&cli.StringFlag{
			Name:        "amanaenv",
			Usage:       "Amana environment",
			Destination: &cfg.AmanaEnv,
		},
		&cli.BoolFlag{
			Name:        "amana",
			Usage:       "Enable Amana",
			Destination: &cfg.Amana,
		},
		&cli.BoolFlag{
			Name:        "consistency",
			Usage:       "Detect data consistency through P2P",
			Destination: &cfg.Consistency,
			Value:       true,
		},
		&cli.BoolFlag{
			Name:        "metrics",
			Usage:       "Enable metrics collection and reporting",
			Destination: &cfg.Metrics,
		},
		&cli.BoolFlag{
			Name:        "metrics.expensive",
			Usage:       "Enable expensive metrics collection and reporting",
			Destination: &cfg.MetricsExpensive,
		},
		&cli.Uint64Flag{
			Name:        "minfreedisk",
			Usage:       "Minimum free disk space in MB, once reached triggers auto shut down (default = 512M, 0 = disabled)",
			Value:       512,
			Destination: &cfg.Minfreedisk,
		},
		&cli.IntFlag{
			Name:        "cache",
			Usage:       "Megabytes of memory allocated to internal caching (default = 1024 mainnet full node)",
			Value:       1024,
			Destination: &cfg.Cache,
		},
		&cli.IntFlag{
			Name:        "cache.database",
			Usage:       "Percentage of cache memory allowance to use for database io",
			Value:       50,
			Destination: &cfg.CacheDatabase,
		},
		&cli.IntFlag{
			Name:        "cache.snapshot",
			Usage:       "Percentage of cache memory allowance to use for snapshot caching (default = 5% full mode)",
			Value:       5,
			Destination: &cfg.CacheSnapshot,
		},
		&cli.BoolFlag{
			Name:        "devnextgdb",
			Usage:       "Enable next generation databases that only exist in development mode",
			Value:       true,
			Destination: &cfg.DevNextGDB,
		},
		&cli.BoolFlag{
			Name:        "autocollectevm",
			Usage:       "auto collect miner coinbase utxo to evm",
			Destination: &cfg.AutoCollectEvm,
		},
		&cli.StringFlag{
			Name:        "walletpass",
			Usage:       "wallet password",
			Destination: &cfg.WalletPass,
		},
		&cli.IntFlag{
			Name:        "evmtrietimeout",
			Usage:       "Set the interval time(seconds) for flush evm trie to disk",
			Destination: &cfg.EVMTrieTimeout,
		},
		&cli.StringFlag{
			Name:        "state.scheme",
			Usage:       "Scheme to use for storing ethereum state ('hash' or 'path')",
			Destination: &cfg.StateScheme,
		},
		&cli.IntFlag{
			Name:        "obsoleteheight",
			Usage:       "What is the maximum allowable height of block obsolescence for submission",
			Value:       defaultObsoleteHeight,
			Destination: &cfg.ObsoleteHeight,
		},
		&cli.BoolFlag{
			Name:        "allowsubmitwhennotsynced",
			Usage:       "Allow the node to accept blocks from RPC while not synced (this flag is mainly used for testing)",
			Value:       false,
			Destination: &cfg.SubmitNoSynced,
		},
		&cli.IntFlag{
			Name:        "gbttimeout",
			Usage:       "Build block template timeout by Millisecond.(Can limit the number of transactions included in the block)",
			Destination: &cfg.GBTTimeOut,
		},
	}
)

func DefaultConfig(homeDir string) *config.Config {
	cfg := &config.Config{
		HomeDir:              defaultHomeDir,
		ConfigFile:           defaultConfigFile,
		DebugLevel:           defaultLogLevel,
		DebugPrintOrigins:    defaultDebugPrintOrigins,
		DataDir:              defaultDataDir,
		LogDir:               defaultLogDir,
		DbType:               defaultDbType,
		RPCKey:               defaultRPCKeyFile,
		RPCCert:              defaultRPCCertFile,
		RPCMaxClients:        defaultMaxRPCClients,
		RPCMaxWebsockets:     defaultMaxRPCWebsockets,
		RPCMaxConcurrentReqs: defaultMaxRPCConcurrentReqs,
		Generate:             defaultGenerate,
		MaxPeers:             defaultMaxPeers,
		MinTxFee:             defaultMinRelayTxFee,
		BlockMinSize:         defaultBlockMinSize,
		BlockMaxSize:         defaultBlockMaxSize,
		SigCacheMaxSize:      defaultSigCacheMaxSize,
		MiningStateSync:      defaultMiningStateSync,
		DAGType:              defaultDAGType,
		Banning:              true,
		MaxInbound:           defaultMaxInboundPeersPerHost,
		InvalidTxIndex:       defaultInvalidTxIndex,
		TxHashIndex:          defaultTxhashIndex,
		NTP:                  false,
		MempoolExpiry:        defaultMempoolExpiry,
		AcceptNonStd:         true,
		RPCUser:              defaultRPCUser,
		RPCPass:              defaultRPCPass,
		ObsoleteHeight:       defaultObsoleteHeight,
		SubmitNoSynced:       false,
		DevNextGDB:           true,
		GBTTimeOut:           defaultGBTTimeout,
	}
	if len(homeDir) > 0 {
		hd, err := filepath.Abs(homeDir)
		if err != nil {
			panic(err)
		}
		cfg.HomeDir = hd
		cfg.ConfigFile = filepath.Join(cfg.HomeDir, defaultConfigFilename)
		cfg.DataDir = filepath.Join(cfg.HomeDir, defaultDataDirname)
		cfg.LogDir = filepath.Join(cfg.HomeDir, defaultLogDirname)
		cfg.RPCKey = filepath.Join(cfg.HomeDir, "rpc.key")
		cfg.RPCCert = filepath.Join(cfg.HomeDir, "rpc.cert")
	}
	return cfg
}
