// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2013-2016 The btcsuite developers

package common

import (
	"fmt"
	"github.com/Qitmeer/qng/common/profiling"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/common/util"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/version"
	"github.com/jessevdk/go-flags"
	"github.com/urfave/cli/v2"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConfigFilename         = "qng.conf"
	defaultDataDirname            = "data"
	defaultLogLevel               = "info"
	defaultDebugPrintOrigins      = false
	defaultLogDirname             = "logs"
	defaultLogFilename            = "qng.log"
	defaultGenerate               = false
	defaultBlockMinSize           = 0
	defaultBlockMaxSize           = types.MaxBlockPayload / 2
	defaultMaxRPCClients          = 10
	defaultMaxRPCWebsockets       = 25
	defaultMaxRPCConcurrentReqs   = 20
	defaultMaxPeers               = 50
	defaultMiningStateSync        = false
	defaultMaxInboundPeersPerHost = 25 // The default max total of inbound peer for host
	defaultTrickleInterval        = 10 * time.Second
	defaultInvalidTxIndex         = false
	defaultMempoolExpiry          = int64(time.Hour)
	defaultRPCUser                = "test"
	defaultRPCPass                = "test"
	defaultMinBlockPruneSize      = 2000
	defaultMinBlockDataCache      = 2000
	defaultMinRelayTxFee          = int64(1e4)
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
			Name:        "droptxindex",
			Usage:       "Deletes the hash-based transaction index from the database on start up and then exits",
			Destination: &cfg.DropTxIndex,
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
			Usage:       "Percentage of cache memory allowance to use for snapshot caching (default = 10% full mode, 20% archive mode)",
			Value:       10,
			Destination: &cfg.CacheSnapshot,
		},
	}
)

// loadConfig initializes and parses the config using a config file and command
// line options.
func LoadConfig(ctx *cli.Context, parsefile bool) (*config.Config, error) {
	cfg.RPCListeners = RPCListeners.Value()
	cfg.Modules = Modules.Value()
	cfg.MiningAddrs = MiningAddrs.Value()
	cfg.BlockMinSize = uint32(BlockMinSize)
	cfg.BlockMaxSize = uint32(BlockMaxSize)
	cfg.BlockPrioritySize = uint32(BlockPrioritySize)
	cfg.AddPeers = AddPeers.Value()
	cfg.BootstrapNodes = BootstrapNodes.Value()
	cfg.Whitelist = Whitelist.Value()
	cfg.Blacklist = Blacklist.Value()
	cfg.GBTNotify = GBTNotify.Value()

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	if cfg.ShowVersion {
		fmt.Printf("%s version %s (Go version %s)\n", appName, version.String(), runtime.Version())
		os.Exit(0)
	}

	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)

	// TODO
	// Perform service command and exit if specified.  Invalid service
	// commands show an appropriate error.  Only runs on Windows since
	// the runServiceCommand function will be nil when not on Windows.
	// TODO

	// Update the home directory for qitmeerd if specified. Since the home
	// directory is updated, other variables need to be updated to
	// reflect the new changes.
	if cfg.HomeDir != defaultHomeDir {
		cfg.HomeDir, _ = filepath.Abs(cfg.HomeDir)

		if cfg.ConfigFile == defaultConfigFile {
			defaultConfigFile = filepath.Join(cfg.HomeDir,
				defaultConfigFilename)
			cfg.ConfigFile = defaultConfigFile
		}
		if cfg.DataDir == defaultDataDir {
			cfg.DataDir = filepath.Join(cfg.HomeDir, defaultDataDirname)
		}
		if cfg.RPCKey == defaultRPCKeyFile {
			cfg.RPCKey = filepath.Join(cfg.HomeDir, "rpc.key")
		}
		if cfg.RPCCert == defaultRPCCertFile {
			cfg.RPCCert = filepath.Join(cfg.HomeDir, "rpc.cert")
		}
		if cfg.LogDir == defaultLogDir {
			cfg.LogDir = filepath.Join(cfg.HomeDir, defaultLogDirname)
		}
	}

	// TODO
	// Create a default config file when one does not exist and the user did
	// not specify an override.
	// TODO

	if ctx.IsSet("configfile") && parsefile {
		// Load additional config from file.
		parser := newConfigParser(cfg, flags.Default)
		err := flags.NewIniParser(parser).ParseFile(cfg.ConfigFile)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				fmt.Fprintf(os.Stderr, "Error parsing config "+
					"file: %v\n", err)
				fmt.Fprintln(os.Stderr, usageMessage)
				return nil, err
			}
			log.Warn(fmt.Sprintf("missing config file error:%s", err))
		}

		// Parse command line options again to ensure they take precedence.
		_, err = parser.Parse()
		if err != nil {
			if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
				fmt.Fprintln(os.Stderr, usageMessage)
			}
			return nil, err
		}
	}

	// Create the home directory if it doesn't already exist.
	funcName := "loadConfig"
	err := os.MkdirAll(cfg.HomeDir, 0700)
	if err != nil {
		// Show a nicer error message if it's because a symlink is
		// linked to a directory that does not exist (probably because
		// it's not mounted).
		if e, ok := err.(*os.PathError); ok && os.IsExist(err) {
			if link, lerr := os.Readlink(e.Path); lerr == nil {
				str := "is symlink %s -> %s mounted?"
				err = fmt.Errorf(str, e.Path, link)
			}
		}
		str := "%s: failed to create home directory: %v"
		err := fmt.Errorf(str, funcName, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	// assign active network params while we're at it
	numNets := 0
	if cfg.TestNet {
		numNets++
		params.ActiveNetParams = &params.TestNetParam
	}
	if cfg.PrivNet {
		numNets++
		// Also disable dns seeding on the private test network.
		params.ActiveNetParams = &params.PrivNetParam
	}
	if cfg.MixNet {
		numNets++
		params.ActiveNetParams = &params.MixNetParam
	}
	// Multiple networks can't be selected simultaneously.
	if numNets > 1 {
		str := "%s: the testnet and simnet params can't be " +
			"used together -- choose one of the three"
		err := fmt.Errorf(str, funcName)
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, err
	}

	// default p2p port
	if len(cfg.DefaultPort) > 0 {
		params.ActiveNetParams.Params.DefaultPort = cfg.DefaultPort
	}

	if cfg.P2PTCPPort <= 0 {
		P2PTCPPort, err := strconv.Atoi(params.ActiveNetParams.DefaultPort)
		if err != nil {
			return nil, err
		}
		cfg.P2PTCPPort = P2PTCPPort
	}

	if cfg.P2PUDPPort <= 0 {
		cfg.P2PUDPPort = params.ActiveNetParams.DefaultUDPPort
	}
	//
	if err := params.ActiveNetParams.PowConfig.Check(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	// Add default port to all rpc listener addresses if needed and remove
	// duplicate addresses.
	cfg.RPCListeners = normalizeAddresses(cfg.RPCListeners,
		params.ActiveNetParams.RpcPort)

	// Only allow TLS to be disabled if the RPC is bound to localhost
	// addresses.
	if !cfg.DisableRPC && cfg.DisableTLS {
		allowedTLSListeners := map[string]struct{}{
			"localhost": {},
			"127.0.0.1": {},
			"0.0.0.0":   {},
			"::1":       {},
		}
		for _, addr := range cfg.RPCListeners {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				str := "%s: RPC listen interface '%s' is " +
					"invalid: %v"
				err := fmt.Errorf(str, funcName, addr, err)
				fmt.Fprintln(os.Stderr, err)
				fmt.Fprintln(os.Stderr, usageMessage)
				return nil, err
			}
			if _, ok := allowedTLSListeners[host]; !ok {
				str := "%s: the --notls option may not be used " +
					"when binding RPC to non localhost " +
					"addresses: %s"
				err := fmt.Errorf(str, funcName, addr)
				fmt.Fprintln(os.Stderr, err)
				fmt.Fprintln(os.Stderr, usageMessage)
				return nil, err
			}
		}
	}

	// Default RPC to listen on localhost only.
	if !cfg.DisableRPC && len(cfg.RPCListeners) == 0 {
		addrs, err := net.LookupHost("localhost")
		if err != nil {
			return nil, err
		}
		cfg.RPCListeners = make([]string, 0, len(addrs))
		for _, addr := range addrs {
			addr = net.JoinHostPort(addr, params.ActiveNetParams.RpcPort)
			cfg.RPCListeners = append(cfg.RPCListeners, addr)
		}
	}

	if cfg.RPCMaxConcurrentReqs < 0 {
		str := "%s: The rpcmaxwebsocketconcurrentrequests option may " +
			"not be less than 0 -- parsed [%d]"
		err := fmt.Errorf(str, funcName, cfg.RPCMaxConcurrentReqs)
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, err
	}

	// Append the network type to the data directory so it is "namespaced"
	// per network.  In addition to the block database, there are other
	// pieces of data that are saved to disk such as address manager state.
	// All data is specific to a network, so namespacing the data directory
	// means each individual piece of serialized data does not have to
	// worry about changing names per network and such.
	cfg.DataDir = util.CleanAndExpandPath(cfg.DataDir)
	cfg.DataDir = filepath.Join(cfg.DataDir, params.ActiveNetParams.Name)

	// Set logging file if presented
	if !cfg.NoFileLogging {
		// Append the network type to the log directory so it is "namespaced"
		// per network in the same fashion as the data directory.
		cfg.LogDir = util.CleanAndExpandPath(cfg.LogDir)
		cfg.LogDir = filepath.Join(cfg.LogDir, params.ActiveNetParams.Name)

		// Initialize log rotation.  After log rotation has been initialized, the
		// logger variables may be used.
		log.InitLogRotator(filepath.Join(cfg.LogDir, defaultLogFilename), cfg.LogRotatorSize)
	}

	// Parse, validate, and set debug log level(s).
	if err := ParseAndSetDebugLevels(cfg.DebugLevel); err != nil {
		err := fmt.Errorf("%s: %v", funcName, err.Error())
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, err
	}

	// DebugPrintOrigins
	if cfg.DebugPrintOrigins {
		log.PrintOrigins(true)
	}

	// --addrindex and --dropaddrindex do not mix.
	if cfg.AddrIndex && cfg.DropAddrIndex {
		err := fmt.Errorf("%s: the --addrindex and --dropaddrindex "+
			"options may not be activated at the same time",
			funcName)
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, err
	}

	// --addrindex and --droptxindex do not mix.
	if cfg.AddrIndex && cfg.DropTxIndex {
		err := fmt.Errorf("%s: the --addrindex and --droptxindex "+
			"options may not be activated at the same time "+
			"because the address index relies on the transaction "+
			"index",
			funcName)
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usageMessage)
		return nil, err
	}

	// Check mining addresses are valid and saved parsed versions.
	for _, strAddr := range cfg.MiningAddrs {
		addr, err := address.DecodeAddress(strAddr)
		if err != nil {
			str := "%s: mining address '%s' failed to decode: %v"
			err := fmt.Errorf(str, funcName, strAddr, err)
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, err
		}
		// TODO, check network by using IsForNetwork()

		if !address.IsForNetwork(addr, params.ActiveNetParams.Params) {
			str := "%s: mining address '%s' is on the wrong network"
			err := fmt.Errorf(str, funcName, strAddr)
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, err
		}
		cfg.SetMiningAddrs(addr)
	}

	if cfg.Generate {
		cfg.Miner = true
	}
	// Ensure there is at least one mining address when the generate or miner flag is
	// set.
	if len(cfg.MiningAddrs) == 0 {
		var str string
		if cfg.Generate {
			str = "%s: the generate flag is set, but there are no mining " +
				"addresses specified "
		}
		if len(str) > 0 {
			err := fmt.Errorf(str, funcName)
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, usageMessage)
			return nil, err
		}
	}

	if cfg.NTP {
		roughtime.Init()
	}

	return cfg, nil
}

// newConfigParser returns a new command line flags parser.
func newConfigParser(cfg *config.Config, options flags.Options) *flags.Parser {
	parser := flags.NewParser(cfg, options)
	return parser
}

// parseAndSetDebugLevels attempts to parse the specified debug level and set
// the levels accordingly.  An appropriate error is returned if anything is
// invalid.
func ParseAndSetDebugLevels(debugLevel string) error {
	// When the specified string doesn't have any delimters, treat it as
	// the log level for all subsystems.
	if !strings.Contains(debugLevel, ",") && !strings.Contains(debugLevel, "=") {
		// Validate debug log level.
		lvl, err := log.LvlFromString(debugLevel)
		if err != nil {
			str := "the specified debug level [%v] is invalid"
			return fmt.Errorf(str, debugLevel)
		}
		// Change the logging level for all subsystems.
		log.Glogger().Verbosity(lvl)
		return nil
	}
	// TODO support log for subsystem
	return nil
}

// normalizeAddress returns addr with the passed default port appended if
// there is not already a port specified.
func normalizeAddress(addr, defaultPort string) string {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		return net.JoinHostPort(addr, defaultPort)
	}
	return addr
}

// normalizeAddresses returns a new slice with all the passed peer addresses
// normalized with the given default port, and all duplicates removed.
func normalizeAddresses(addrs []string, defaultPort string) []string {
	for i, addr := range addrs {
		addrs[i] = normalizeAddress(addr, defaultPort)
	}

	return removeDuplicateAddresses(addrs)
}

// removeDuplicateAddresses returns a new slice with all duplicate entries in
// addrs removed.
func removeDuplicateAddresses(addrs []string) []string {
	result := make([]string, 0, len(addrs))
	seen := map[string]struct{}{}
	for _, val := range addrs {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = struct{}{}
		}
	}
	return result
}

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
		NTP:                  false,
		MempoolExpiry:        defaultMempoolExpiry,
		AcceptNonStd:         true,
		RPCUser:              defaultRPCUser,
		RPCPass:              defaultRPCPass,
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
