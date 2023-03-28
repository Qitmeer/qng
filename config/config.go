package config

import (
	"github.com/Qitmeer/qng/core/types"
)

type Config struct {
	HomeDir            string   `short:"A" long:"appdata" description:"Path to application home directory"`
	ShowVersion        bool     `short:"V" long:"version" description:"Display version information and exit"`
	ConfigFile         string   `short:"C" long:"configfile" description:"Path to configuration file"`
	DataDir            string   `short:"b" long:"datadir" description:"Directory to store data"`
	LogDir             string   `long:"logdir" description:"Directory to log output."`
	NoFileLogging      bool     `long:"nofilelogging" description:"Disable file logging."`
	Listener           string   `long:"listen" description:"Add an IP to listen for connections"`
	DefaultPort        string   `long:"port" description:"Default p2p port."`
	RPCListeners       []string `long:"rpclisten" description:"Add an interface/port to listen for RPC connections (default port: 8131 , testnet: 18131)"`
	MaxPeers           int      `long:"maxpeers" description:"Max number of inbound and outbound peers"`
	DisableListen      bool     `long:"nolisten" description:"Disable listening for incoming connections"`
	RPCUser            string   `short:"u" long:"rpcuser" description:"Username for RPC connections"`
	RPCPass            string   `short:"P" long:"rpcpass" default-mask:"-" description:"Password for RPC connections"`
	RPCCert            string   `long:"rpccert" description:"File containing the certificate file"`
	RPCKey             string   `long:"rpckey" description:"File containing the certificate key"`
	RPCMaxClients      int      `long:"rpcmaxclients" description:"Max number of RPC clients for standard connections"`
	DisableRPC         bool     `long:"norpc" description:"Disable built-in RPC server -- NOTE: The RPC server is disabled by default if no rpcuser/rpcpass or rpclimituser/rpclimitpass is specified"`
	DisableTLS         bool     `long:"notls" description:"Disable TLS for the RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	Modules            []string `long:"modules" description:"Modules is a list of API modules(See GetNodeInfo) to expose via the HTTP RPC interface. If the module list is empty, all RPC API endpoints designated public will be exposed."`
	DisableCheckpoints bool     `long:"nocheckpoints" description:"Disable built-in checkpoints.  Don't do this unless you know what you're doing."`
	LightNode          bool     `long:"light" description:"start as a qitmeer light node"`
	SigCacheMaxSize    uint     `long:"sigcachemaxsize" description:"The maximum number of entries in the signature verification cache"`
	TestNet            bool     `long:"testnet" description:"Use the test network"`
	MixNet             bool     `long:"mixnet" description:"Use the test mix pow network"`
	PrivNet            bool     `long:"privnet" description:"Use the private network"`
	DbType             string   `long:"dbtype" description:"Database backend to use for the Block Chain"`
	Profile            string   `long:"profile" description:"Enable HTTP profiling on given [addr:]port -- NOTE port must be between 1024 and 65536"`
	CPUProfile         string   `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	TrackHeap          bool     `long:"trackheap" description:"tracks the size of the heap and dumps a profile"`
	TrackHeapLimit     int      `long:"trackheaplimit" description:"track heap when limit in gigabytes (default:7G)"`
	DebugLevel         string   `short:"d" long:"debuglevel" description:"Logging level {trace, debug, info, warn, error, critical} "`
	DebugPrintOrigins  bool     `long:"printorigin" description:"Print log debug location (file:line) "`

	// MemPool Config
	NoRelayPriority  bool    `long:"norelaypriority" description:"Do not require free or low-fee transactions to have high priority for relaying"`
	FreeTxRelayLimit float64 `long:"limitfreerelay" description:"Limit relay of transactions with no transaction fee to the given amount in thousands of bytes per minute"`
	AcceptNonStd     bool    `long:"acceptnonstd" description:"Accept and relay non-standard transactions to the network regardless of the default settings for the active network."`
	MaxOrphanTxs     int     `long:"maxorphantx" description:"Max number of orphan transactions to keep in memory"`
	TxTimeScope      int64   `long:"txtimescope" description:"allow the mempool tx time scope(sec) with server time,default 0 will not check the time scope"`
	MinTxFee         int64   `long:"mintxfee" description:"The minimum transaction fee in AtomMEER/kB."`
	MempoolExpiry    int64   `long:"mempoolexpiry" description:"Do not keep transactions in the mempool more than mempoolexpiry"`
	Persistmempool   bool    `long:"persistmempool" description:"Whether to save the mempool on shutdown and load on restart"`
	NoMempoolBar     bool    `long:"nomempoolbar" description:"Whether to show progress bar when load mempool from file"`
	// Miner
	Miner             bool     `long:"miner" description:"Enable miner module"`
	Generate          bool     `long:"generate" description:"Generate (mine) coins using the CPU"`
	MiningAddrs       []string `long:"miningaddr" description:"Add the specified payment address to the list of addresses to use for generated blocks -- At least one address is required if the generate option is set"`
	MiningTimeOffset  int      `long:"miningtimeoffset" description:"Offset the mining timestamp of a block by this many seconds (positive values are in the past)"`
	BlockMinSize      uint32   `long:"blockminsize" description:"Mininum block size in bytes to be used when creating a block"`
	BlockMaxSize      uint32   `long:"blockmaxsize" description:"Maximum block size in bytes to be used when creating a block"`
	BlockPrioritySize uint32   `long:"blockprioritysize" description:"Size in bytes for high-priority/low-fee transactions when creating a block"`
	miningAddrs       []types.Address
	GBTNotify         []string `long:"gbtnotify" description:"HTTP URL list to be notified of new block template"`

	//WebSocket support
	RPCMaxWebsockets     int `long:"rpcmaxwebsockets" description:"Max number of RPC websocket connections"`
	RPCMaxConcurrentReqs int `long:"rpcmaxconcurrentreqs" description:"Max number of concurrent RPC requests that may be processed concurrently"`
	//P2P
	BlocksOnly      bool     `long:"blocksonly" description:"Do not accept transactions from remote peers."`
	MiningStateSync bool     `long:"miningstatesync" description:"Synchronizing the mining state with other nodes"`
	AddPeers        []string `short:"a" long:"addpeer" description:"Add a peer to connect with at startup"`
	Upnp            bool     `long:"upnp" description:"Use UPnP to map our listening port outside of NAT"`
	MaxInbound      int      `long:"maxinbound" description:"The max total of inbound peer for host"`
	//P2P - server ban
	Banning bool `long:"banning" description:"Enable banning of misbehaving peers"`

	DAGType     string `short:"G" long:"dagtype" description:"DAG type {phantom,conflux,spectre} "`
	Cleanup     bool   `short:"L" long:"cleanup" description:"Cleanup the block database "`
	BuildLedger bool   `long:"buildledger" description:"Generate the genesis ledger for the next qitmeer version."`

	Zmqpubhashblock string `long:"zmqpubhashblock" description:"Enable publish hash block  in <address>"`
	Zmqpubrawblock  string `long:"zmqpubrawblock" description:"Enable publish raw block in <address>"`

	Zmqpubhashtx string `long:"zmqpubhashtx" description:"Enable publish hash transaction in <address>"`
	Zmqpubrawtx  string `long:"zmqpubrawtx" description:"Enable publish raw transaction in <address>"`

	// index
	AddrIndex      bool `long:"addrindex" description:"Maintain a full address-based transaction index which makes the getrawtransactions RPC available"`
	VMBlockIndex   bool `long:"vmblockindex" description:"Maintain a full vm block index which makes the GetTxIDByMeerEVMTxHash RPC available"`
	InvalidTxIndex bool `long:"invalidtxindex" description:"Cache invalid transactions."`
	DropAddrIndex  bool `long:"dropaddrindex" description:"Deletes the address-based transaction index from the database on start up and then exits."`
	DropTxIndex    bool `long:"droptxindex" description:"Deletes the hash-based transaction index from the database on start up and then exits."`

	NTP bool `long:"ntp" description:"Auto sync time."`

	//net2.0
	BootstrapNodes []string `long:"bootstrapnode" description:"The address of bootstrap node."`
	NoDiscovery    bool     `long:"nodiscovery" description:"Enable only local network p2p and do not connect to cloud bootstrap nodes."`
	MetaDataDir    string   `long:"metadatadir" description:"meta data dir for p2p"`
	P2PUDPPort     int      `long:"p2pudpport" description:"The udp port used by P2P."`
	P2PTCPPort     int      `long:"p2ptcpport" description:"The tcp port used by P2P."`
	HostIP         string   `long:"externalip" description:"The IP address advertised by libp2p. This may be used to advertise an external IP."`
	HostDNS        string   `long:"externaldns" description:"The DNS address advertised by libp2p. This may be used to advertise an external DNS."`
	RelayNode      string   `long:"relaynode" description:"The address of relay node that routes traffic between two peers over a qitmeer “relay” peer."`
	Whitelist      []string `long:"whitelist" description:"Add an IP network or IP,PeerID that will not be banned or ignore dual channel mode detection. (eg. 192.168.1.0/24 or ::1 or [peer id])"`
	Blacklist      []string `long:"blacklist" description:"Add some IP network or IP that will be banned. (eg. 192.168.1.0/24 or ::1)"`
	MaxBadResp     int      `long:"maxbadresp" description:"maxbadresp is the maximum number of bad responses from a peer before we stop talking to it."`
	Circuit        bool     `long:"circuit" description:"All peers will ignore dual channel mode detection"`

	// meerevm environment
	EVMEnv string `long:"evmenv" description:"meer EVM environment"`

	Estimatefee bool `long:"estimatefee" description:"Enable estimate fee"`

	AcctMode   bool `long:"acctmode" description:"Enable support account system mode"`
	IsArchival bool `long:"archival" description:"Archival tells the consensus if it should not prune old blocks"`

	DAGCacheSize       uint64 `long:"dagcachesize" description:"DAG block cache size"`
	BlockDataCacheSize uint64 `long:"bdcachesize" description:"Block data cache size"`

	Amana    bool   `long:"amana" description:"Enable Amana"`
	AmanaEnv string `long:"amanaenv" description:"Amana environment"`

	// wallet
	WalletPass     string `long:"walletpass" description:"wallet password"`
	AutoCollectEvm bool   `long:"autocollectevm" description:"auto collect utxo to evm"`
}

func (c *Config) GetMinningAddrs() []types.Address {
	return c.miningAddrs
}

func (c *Config) SetMiningAddrs(addr types.Address) {
	c.miningAddrs = append(c.miningAddrs, addr)
}

var Cfg *Config
