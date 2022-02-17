/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package chain

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Qitmeer/meerevm/evm/engine"
	"github.com/Qitmeer/qng-core/core/protocol"
	qparams "github.com/Qitmeer/qng-core/params"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

var (
	// ClientIdentifier is a hard coded identifier to report into the network.
	ClientIdentifier = "meereth"

	MeerethChainID int64    = 223
	Args           []string = []string{ClientIdentifier}

	//

	NodeFlags = []cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.AncientFlag,
		utils.MinFreeDiskSpaceFlag,
		utils.KeyStoreDirFlag,
		utils.ExternalSignerFlag,
		utils.NoUSBFlag,
		utils.USBFlag,
		utils.SmartCardDaemonPathFlag,
		utils.OverrideLondonFlag,
		utils.EthashCacheDirFlag,
		utils.EthashCachesInMemoryFlag,
		utils.EthashCachesOnDiskFlag,
		utils.EthashCachesLockMmapFlag,
		utils.EthashDatasetDirFlag,
		utils.EthashDatasetsInMemoryFlag,
		utils.EthashDatasetsOnDiskFlag,
		utils.EthashDatasetsLockMmapFlag,
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
		utils.WhitelistFlag,
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
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.LegacyMinerGasTargetFlag,
		utils.MinerGasLimitFlag,
		utils.MinerGasPriceFlag,
		utils.MinerExtraDataFlag,
		utils.MinerRecommitIntervalFlag,
		utils.MinerNoVerifyFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.MainnetFlag,
		utils.RopstenFlag,
		utils.RinkebyFlag,
		utils.GoerliFlag,
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
	}

	RpcFlags = []cli.Flag{
		utils.HTTPEnabledFlag,
		utils.HTTPListenAddrFlag,
		utils.HTTPPortFlag,
		utils.HTTPCORSDomainFlag,
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
		utils.RPCGlobalTxFeeCapFlag,
		utils.AllowUnprotectedTxs,
	}

	MetricsFlags = []cli.Flag{
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
)

type ETHChain struct {
	ctx *cli.Context

	started  int32
	shutdown int32

	config  *MeerethConfig
	node    *node.Node
	ether   *eth.Ethereum
	backend *eth.EthAPIBackend

	genPrivateKey *ecdsa.PrivateKey
	genAddress    common.Address
}

func (ec *ETHChain) Start() error {
	if atomic.AddInt32(&ec.started, 1) != 1 {
		return fmt.Errorf("Service is already in the process of started")
	}
	return ec.startNode()
}

func (ec *ETHChain) Wait() {
	ec.node.Wait()
}

func (ec *ETHChain) Stop() error {
	if atomic.AddInt32(&ec.shutdown, 1) != 1 {
		return fmt.Errorf("Service is already in the process of shutting down")
	}

	ec.node.Close()

	ec.Wait()
	return nil
}

func (ec *ETHChain) IsStarted() bool {
	return atomic.LoadInt32(&ec.started) != 0
}

func (ec *ETHChain) IsShutdown() bool {
	return atomic.LoadInt32(&ec.shutdown) != 0
}

func (ec *ETHChain) Node() *node.Node {
	return ec.node
}

func (ec *ETHChain) Backend() *eth.EthAPIBackend {
	return ec.backend
}

func (ec *ETHChain) Ether() *eth.Ethereum {
	return ec.ether
}

func (ec *ETHChain) Config() *MeerethConfig {
	return ec.config
}

func (ec *ETHChain) startNode() error {
	stack := ec.node
	ctx := ec.ctx
	err := stack.Start()
	if err != nil {
		return err
	}

	ec.unlockAccounts()

	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	rpcClient, err := stack.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to self: %v", err)
	}
	ethClient := ethclient.NewClient(rpcClient)

	go func() {
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				var derivationPaths []accounts.DerivationPath
				if event.Wallet.URL().Scheme == "ledger" {
					derivationPaths = append(derivationPaths, accounts.LegacyLedgerBaseDerivationPath)
				}
				derivationPaths = append(derivationPaths, accounts.DefaultBaseDerivationPath)

				event.Wallet.SelfDerive(derivationPaths, ethClient)

			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()

	if ctx.GlobalBool(utils.ExitWhenSyncedFlag.Name) {
		go func() {
			sub := stack.EventMux().Subscribe(downloader.DoneEvent{})
			defer sub.Unsubscribe()
			for {
				event := <-sub.Chan()
				if event == nil {
					continue
				}
				done, ok := event.Data.(downloader.DoneEvent)
				if !ok {
					continue
				}
				if timestamp := time.Unix(int64(done.Latest.Time), 0); time.Since(timestamp) < 10*time.Minute {
					log.Info("Synchronisation completed", "latestnum", done.Latest.Number, "latesthash", done.Latest.Hash(),
						"age", common.PrettyAge(timestamp))
					stack.Close()
				}
			}
		}()
	}

	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) || ctx.GlobalBool(utils.DeveloperFlag.Name) {
		if ctx.GlobalString(utils.SyncModeFlag.Name) == "light" {
			utils.Fatalf("Light clients do not support mining")
		}

		gasprice := utils.GlobalBig(ctx, utils.MinerGasPriceFlag.Name)
		ec.backend.TxPool().SetGasPrice(gasprice)
		threads := ctx.GlobalInt(utils.MinerThreadsFlag.Name)
		if err := ec.backend.StartMining(threads); err != nil {
			utils.Fatalf("Failed to start mining: %v", err)
		}
	}

	return nil
}

func (ec *ETHChain) unlockAccounts() {
	stack := ec.node
	var unlocks []string
	inputs := strings.Split(ec.ctx.GlobalString(utils.UnlockedAccountFlag.Name), ",")
	for _, input := range inputs {
		if trimmed := strings.TrimSpace(input); trimmed != "" {
			unlocks = append(unlocks, trimmed)
		}
	}
	// Short circuit if there is no account to unlock.
	if len(unlocks) == 0 {
		return
	}
	// If insecure account unlocking is not allowed if node's APIs are exposed to external.
	// Print warning log to user and skip unlocking.
	if !stack.Config().InsecureUnlockAllowed && stack.Config().ExtRPCEnabled() {
		utils.Fatalf("Account unlock with HTTP access is forbidden!")
	}
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	passwords := utils.MakePasswordList(ec.ctx)
	for i, account := range unlocks {
		UnlockAccount(ks, account, i, passwords)
	}
}

func NewETHChainByCfg(config *MeerethConfig) (*ETHChain, error) {
	ec := &ETHChain{config: config}

	//
	app := cli.NewApp()
	app.Name = ClientIdentifier
	app.Author = ClientIdentifier
	app.Email = ClientIdentifier
	app.Version = params.VersionWithMeta
	app.Usage = ClientIdentifier

	//

	utils.CacheFlag.Value = 4096

	app.Action = func(ctx *cli.Context) {
		ec.ctx = ctx
		ec.node, ec.backend, ec.ether = makeFullNode(ec.ctx, ec.config)
	}
	app.HideVersion = true
	app.Copyright = ClientIdentifier

	app.Flags = append(app.Flags, NodeFlags...)
	app.Flags = append(app.Flags, RpcFlags...)
	app.Flags = append(app.Flags, MetricsFlags...)

	err := app.Run(Args)
	if err != nil {
		return nil, err
	}

	return ec, nil
}

func NewETHChain(datadir string) (*ETHChain, error) {
	config, err := MakeMeerethConfig(datadir)
	if err != nil {
		return nil, err
	}
	return NewETHChainByCfg(config)
}

func MakeMeerethConfig(datadir string) (*MeerethConfig, error) {
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(MeerethChainID),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		CatalystBlock:       big.NewInt(0),
		LondonBlock:         nil,
		Ethash:              new(params.EthashConfig),
	}

	genBalance := big.NewInt(1000000000000000000)
	genAddress := common.HexToAddress("0x71bc4403Af41634Cda7C32600A8024d54e7F6499")

	genesis := &core.Genesis{
		Config:     chainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      core.GenesisAlloc{genAddress: {Balance: genBalance}},
		Timestamp:  uint64(qparams.ActiveNetParams.GenesisBlock.Header.Timestamp.Unix()),
	}

	etherbase := common.Address{}
	econfig := ethconfig.Defaults

	econfig.NetworkId = uint64(MeerethChainID)
	econfig.Genesis = genesis
	econfig.SyncMode = downloader.FullSync
	econfig.NoPruning = true
	econfig.SkipBcVersionCheck = false
	econfig.TrieDirtyCache = 0
	econfig.ConsensusEngine = CreateConsensusEngine

	econfig.Ethash.DatasetDir = "ethash/dataset"

	econfig.Miner.Etherbase = etherbase
	econfig.Miner.ExtraData = []byte{byte(0)}

	econfig.TxPool.NoLocals = true

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
	nodeConf.HTTPPort, nodeConf.WSPort = getDefaultRPCPort()

	nodeConf.P2P.MaxPeers = 0
	nodeConf.P2P.DiscoveryV5 = false
	nodeConf.P2P.NoDiscovery = true
	nodeConf.P2P.NoDial = true
	nodeConf.P2P.ListenAddr = ""
	nodeConf.P2P.NAT = nil

	//
	return &MeerethConfig{
		Eth:     econfig,
		Node:    nodeConf,
		Metrics: metrics.DefaultConfig,
	}, nil
}

func makeFullNode(ctx *cli.Context, cfg *MeerethConfig) (*node.Node, *eth.EthAPIBackend, *eth.Ethereum) {
	stack := makeConfigNode(ctx, cfg)
	if ctx.GlobalIsSet(utils.OverrideLondonFlag.Name) {
		cfg.Eth.OverrideLondon = new(big.Int).SetUint64(ctx.GlobalUint64(utils.OverrideLondonFlag.Name))
	}
	backend, ethe := utils.RegisterEthService(stack, &cfg.Eth)

	if ctx.GlobalBool(utils.CatalystFlag.Name) {
		if ethe == nil {
			utils.Fatalf("Catalyst does not work in light client mode.")
		}
		if err := catalyst.Register(stack, ethe); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	if ctx.GlobalIsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend, cfg.Node)
	}
	if cfg.Ethstats.URL != "" {
		utils.RegisterEthStatsService(stack, backend, cfg.Ethstats.URL)
	}
	return stack, backend.(*eth.EthAPIBackend), ethe
}

func makeConfigNode(ctx *cli.Context, cfg *MeerethConfig) *node.Node {
	filterConfig(ctx)
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	if err := setAccountManagerBackends(stack); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	if ctx.GlobalIsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.GlobalString(utils.EthStatsURLFlag.Name)
	}
	applyMetricConfig(ctx, cfg)

	return stack
}

func MakeMeerethConfigNode(ctx *cli.Context, datadir string) (*node.Node, *MeerethConfig) {
	config, err := MakeMeerethConfig(datadir)
	if err != nil {
		log.Error(err.Error())
		return nil, nil
	}
	return makeConfigNode(ctx, config), config
}

func setAccountManagerBackends(stack *node.Node) error {
	conf := stack.Config()
	am := stack.AccountManager()
	keydir := stack.KeyStoreDir()
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if conf.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	if len(conf.ExternalSigner) > 0 {
		log.Info("Using external signer", "url", conf.ExternalSigner)
		if extapi, err := external.NewExternalBackend(conf.ExternalSigner); err == nil {
			am.AddBackend(extapi)
			return nil
		} else {
			return fmt.Errorf("error connecting to external signer: %v", err)
		}
	}

	am.AddBackend(keystore.NewKeyStore(keydir, scryptN, scryptP))
	if conf.USB {
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			am.AddBackend(ledgerhub)
		}
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start HID Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
		if trezorhub, err := usbwallet.NewTrezorHubWithWebUSB(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start WebUSB Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
	}
	if len(conf.SmartCardDaemonPath) > 0 {
		if schub, err := scwallet.NewHub(conf.SmartCardDaemonPath, scwallet.Scheme, keydir); err != nil {
			log.Warn(fmt.Sprintf("Failed to start smart card hub, disabling: %v", err))
		} else {
			am.AddBackend(schub)
		}
	}

	return nil
}

func applyMetricConfig(ctx *cli.Context, cfg *MeerethConfig) {
	if ctx.GlobalIsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.GlobalBool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		cfg.Metrics.EnabledExpensive = ctx.GlobalBool(utils.MetricsEnabledExpensiveFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsHTTPFlag.Name) {
		cfg.Metrics.HTTP = ctx.GlobalString(utils.MetricsHTTPFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsPortFlag.Name) {
		cfg.Metrics.Port = ctx.GlobalInt(utils.MetricsPortFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		cfg.Metrics.EnableInfluxDB = ctx.GlobalBool(utils.MetricsEnableInfluxDBFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBEndpointFlag.Name) {
		cfg.Metrics.InfluxDBEndpoint = ctx.GlobalString(utils.MetricsInfluxDBEndpointFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
		cfg.Metrics.InfluxDBDatabase = ctx.GlobalString(utils.MetricsInfluxDBDatabaseFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBUsernameFlag.Name) {
		cfg.Metrics.InfluxDBUsername = ctx.GlobalString(utils.MetricsInfluxDBUsernameFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBPasswordFlag.Name) {
		cfg.Metrics.InfluxDBPassword = ctx.GlobalString(utils.MetricsInfluxDBPasswordFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBTagsFlag.Name) {
		cfg.Metrics.InfluxDBTags = ctx.GlobalString(utils.MetricsInfluxDBTagsFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsEnableInfluxDBV2Flag.Name) {
		cfg.Metrics.EnableInfluxDBV2 = ctx.GlobalBool(utils.MetricsEnableInfluxDBV2Flag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBTokenFlag.Name) {
		cfg.Metrics.InfluxDBToken = ctx.GlobalString(utils.MetricsInfluxDBTokenFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBBucketFlag.Name) {
		cfg.Metrics.InfluxDBBucket = ctx.GlobalString(utils.MetricsInfluxDBBucketFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBOrganizationFlag.Name) {
		cfg.Metrics.InfluxDBOrganization = ctx.GlobalString(utils.MetricsInfluxDBOrganizationFlag.Name)
	}
}

func UnlockAccount(ks *keystore.KeyStore, address string, i int, passwords []string) (accounts.Account, string) {
	account, err := utils.MakeAddress(ks, address)
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := utils.GetPassPhraseWithList(prompt, false, i, passwords)
		err = ks.Unlock(account, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	utils.Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError, auth string) accounts.Account {
	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.URL)
	}
	fmt.Println("Testing your password against all of them...")
	var match *accounts.Account
	for _, a := range err.Matches {
		if err := ks.Unlock(a, auth); err == nil {
			match = &a
			break
		}
	}
	if match == nil {
		utils.Fatalf("None of the listed files could be unlocked.")
	}
	fmt.Printf("Your password unlocked %s\n", match.URL)
	fmt.Println("In order to avoid this warning, you need to remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.URL)
		}
	}
	return *match
}

func CreateConsensusEngine(stack *node.Node, chainConfig *params.ChainConfig, config *ethash.Config, notify []string, noverify bool, db ethdb.Database) consensus.Engine {
	engine := engine.New(engine.Config{
		CacheDir:         stack.ResolvePath(config.CacheDir),
		CachesInMem:      config.CachesInMem,
		CachesOnDisk:     config.CachesOnDisk,
		CachesLockMmap:   config.CachesLockMmap,
		DatasetDir:       stack.ResolvePath(config.DatasetDir),
		DatasetsInMem:    config.DatasetsInMem,
		DatasetsOnDisk:   config.DatasetsOnDisk,
		DatasetsLockMmap: config.DatasetsLockMmap,
		NotifyFull:       config.NotifyFull,
	}, notify, noverify)
	engine.SetThreads(-1) // Disable CPU mining
	return engine
}

func InitEnv(env string) {
	if len(env) <= 0 {
		return
	}
	args := strings.Split(env, " ")
	if len(args) <= 0 {
		return
	}
	log.Debug(fmt.Sprintf("Initialize meerevm environment:%v", args))
	Args = append(Args, args...)
}

func filterConfig(ctx *cli.Context) {
	hms := ctx.GlobalString(utils.HTTPApiFlag.Name)
	if len(hms) > 0 {
		modules := utils.SplitAndTrim(hms)
		nmodules := ""
		for _, mod := range modules {
			if mod == "ethash" || mod == "miner" {
				continue
			}
			if len(nmodules) > 0 {
				nmodules = nmodules + "," + mod
			} else {
				nmodules = mod
			}
		}
		ctx.GlobalSet(utils.HTTPApiFlag.Name, nmodules)
	}

	wms := ctx.GlobalString(utils.WSApiFlag.Name)
	if len(hms) > 0 {
		modules := utils.SplitAndTrim(wms)
		nmodules := ""
		for _, mod := range modules {
			if mod == "ethash" || mod == "miner" {
				continue
			}
			if len(nmodules) > 0 {
				nmodules = nmodules + "," + mod
			} else {
				nmodules = mod
			}
		}
		ctx.GlobalSet(utils.WSApiFlag.Name, nmodules)
	}
}

func getDefaultRPCPort() (int, int) {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		return 8535, 8536
	case protocol.TestNet:
		return 18535, 18536
	case protocol.MixNet:
		return 28535, 28536
	default:
		return 38535, 38536
	}
}
