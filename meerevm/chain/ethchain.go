/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package chain

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/Qitmeer/qng/core/protocol"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/evm/engine"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"math/big"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	// Force-load the tracer engines to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

var (
	// ClientIdentifier is a hard coded identifier to report into the network.
	ClientIdentifier = "meereth"

	MeerethChainID int64    = 223
	Args           []string = []string{ClientIdentifier}

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

	ChainConfig = &params.ChainConfig{
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
		LondonBlock:         nil,
		Ethash:              new(params.EthashConfig),
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

	if ctx.Bool(utils.ExitWhenSyncedFlag.Name) {
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

	if ctx.Bool(utils.MiningEnabledFlag.Name) || ctx.Bool(utils.DeveloperFlag.Name) {
		if ctx.String(utils.SyncModeFlag.Name) == "light" {
			utils.Fatalf("Light clients do not support mining")
		}

		gasprice := qcommon.GlobalBig(ctx, utils.MinerGasPriceFlag.Name)
		ec.backend.TxPool().SetGasPrice(gasprice)
		threads := ctx.Int(utils.MinerThreadsFlag.Name)
		if err := ec.backend.StartMining(threads); err != nil {
			utils.Fatalf("Failed to start mining: %v", err)
		}
	}

	return nil
}

func (ec *ETHChain) unlockAccounts() {
	stack := ec.node
	var unlocks []string
	inputs := strings.Split(ec.ctx.String(utils.UnlockedAccountFlag.Name), ",")
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
	app.Authors = []*cli.Author{
		{Name: ClientIdentifier, Email: ClientIdentifier},
	}
	app.Version = params.VersionWithMeta
	app.Usage = ClientIdentifier

	//

	utils.CacheFlag.Value = 4096

	app.Action = func(ctx *cli.Context) error {
		ec.ctx = ctx
		ec.node, ec.backend, ec.ether = makeFullNode(ec.ctx, ec.config)
		return nil
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
	ChainConfig.ChainID = big.NewInt(qparams.ActiveNetParams.MeerEVMCfg.ChainID)
	genesis := DefaultGenesisBlock(ChainConfig)

	etherbase := common.Address{}
	econfig := ethconfig.Defaults

	econfig.NetworkId = uint64(qparams.ActiveNetParams.MeerEVMCfg.ChainID)
	econfig.Genesis = genesis
	econfig.SyncMode = downloader.FullSync
	econfig.NoPruning = true
	econfig.SkipBcVersionCheck = false
	econfig.TrieDirtyCache = 0
	econfig.ConsensusEngine = CreateConsensusEngine

	econfig.Ethash.DatasetDir = "ethash/dataset"

	econfig.Miner.Etherbase = etherbase
	econfig.Miner.ExtraData = []byte{byte(0)}
	econfig.Miner.External = &MeerPool{}

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
	nodeConf.HTTPPort, nodeConf.WSPort, nodeConf.AuthPort = getDefaultRPCPort()

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
	if ctx.IsSet(utils.OverrideTerminalTotalDifficulty.Name) {
		cfg.Eth.OverrideTerminalTotalDifficulty = qcommon.GlobalBig(ctx, utils.OverrideTerminalTotalDifficulty.Name)
	}
	if ctx.IsSet(utils.OverrideTerminalTotalDifficultyPassed.Name) {
		override := ctx.Bool(utils.OverrideTerminalTotalDifficultyPassed.Name)
		cfg.Eth.OverrideTerminalTotalDifficultyPassed = &override
	}
	backend, ethe := utils.RegisterEthService(stack, &cfg.Eth)

	if ethe != nil {
		firstIdx := uint64(0)
		// Hack to speed up check for mainnet because we know
		// the first non-empty block.
		ghash := rawdb.ReadCanonicalHash(ethe.ChainDb(), 0)
		if cfg.Eth.NetworkId == 1 && ghash == params.MainnetGenesisHash {
			firstIdx = 46147
		}
		isLegacy, _, err := DBHasLegacyReceipts(ethe.ChainDb(), firstIdx)
		if err != nil {
			log.Error("Failed to check db for legacy receipts", "err", err)
		} else if isLegacy {
			log.Warn("Database has receipts with a legacy format. Please run `qng db freezer-migrate`.")
		}
	}
	// Configure log filter RPC API.
	filterSystem := utils.RegisterFilterAPI(stack, backend, &cfg.Eth)

	if ctx.IsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend,filterSystem, &cfg.Node)
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
	if ctx.IsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.String(utils.EthStatsURLFlag.Name)
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
	if ctx.IsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.Bool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		cfg.Metrics.EnabledExpensive = ctx.Bool(utils.MetricsEnabledExpensiveFlag.Name)
	}
	if ctx.IsSet(utils.MetricsHTTPFlag.Name) {
		cfg.Metrics.HTTP = ctx.String(utils.MetricsHTTPFlag.Name)
	}
	if ctx.IsSet(utils.MetricsPortFlag.Name) {
		cfg.Metrics.Port = ctx.Int(utils.MetricsPortFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		cfg.Metrics.EnableInfluxDB = ctx.Bool(utils.MetricsEnableInfluxDBFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBEndpointFlag.Name) {
		cfg.Metrics.InfluxDBEndpoint = ctx.String(utils.MetricsInfluxDBEndpointFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
		cfg.Metrics.InfluxDBDatabase = ctx.String(utils.MetricsInfluxDBDatabaseFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBUsernameFlag.Name) {
		cfg.Metrics.InfluxDBUsername = ctx.String(utils.MetricsInfluxDBUsernameFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBPasswordFlag.Name) {
		cfg.Metrics.InfluxDBPassword = ctx.String(utils.MetricsInfluxDBPasswordFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTagsFlag.Name) {
		cfg.Metrics.InfluxDBTags = ctx.String(utils.MetricsInfluxDBTagsFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBV2Flag.Name) {
		cfg.Metrics.EnableInfluxDBV2 = ctx.Bool(utils.MetricsEnableInfluxDBV2Flag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTokenFlag.Name) {
		cfg.Metrics.InfluxDBToken = ctx.String(utils.MetricsInfluxDBTokenFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBBucketFlag.Name) {
		cfg.Metrics.InfluxDBBucket = ctx.String(utils.MetricsInfluxDBBucketFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBOrganizationFlag.Name) {
		cfg.Metrics.InfluxDBOrganization = ctx.String(utils.MetricsInfluxDBOrganizationFlag.Name)
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
	if e, err := strconv.Unquote(env); err == nil {
		env = e
	}
	args := strings.Split(env, " ")
	if len(args) <= 0 {
		return
	}
	log.Debug(fmt.Sprintf("Initialize meerevm environment: %v %v ", len(args), args))
	Args = append(Args, args...)
}

func filterConfig(ctx *cli.Context) {
	hms := ctx.String(utils.HTTPApiFlag.Name)
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
		ctx.Set(utils.HTTPApiFlag.Name, nmodules)
	}

	wms := ctx.String(utils.WSApiFlag.Name)
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
		ctx.Set(utils.WSApiFlag.Name, nmodules)
	}
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

func DBHasLegacyReceipts(db ethdb.Database, firstIdx uint64) (bool, uint64, error) {
	// Check first block for legacy receipt format
	numAncients, err := db.Ancients()
	if err != nil {
		return false, 0, err
	}
	if numAncients < 1 {
		return false, 0, nil
	}
	if firstIdx >= numAncients {
		return false, firstIdx, nil
	}
	var (
		legacy       bool
		blob         []byte
		emptyRLPList = []byte{192}
	)
	// Find first block with non-empty receipt, only if
	// the index is not already provided.
	if firstIdx == 0 {
		for i := uint64(0); i < numAncients; i++ {
			blob, err = db.Ancient("receipts", i)
			if err != nil {
				return false, 0, err
			}
			if len(blob) == 0 {
				continue
			}
			if !bytes.Equal(blob, emptyRLPList) {
				firstIdx = i
				break
			}
		}
	}
	// Is first non-empty receipt legacy?
	first, err := db.Ancient("receipts", firstIdx)
	if err != nil {
		return false, 0, err
	}
	legacy, err = types.IsLegacyStoredReceipts(first)
	return legacy, firstIdx, err
}
