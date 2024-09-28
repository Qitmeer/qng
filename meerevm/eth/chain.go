/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package eth

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"math/big"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	// Force-load the tracer engines to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

type ETHChain struct {
	ctx *cli.Context

	started  int32
	shutdown int32

	config  *Config
	node    *node.Node
	ether   *eth.Ethereum
	backend *eth.EthAPIBackend
}

func (ec *ETHChain) Start() error {
	if atomic.AddInt32(&ec.started, 1) != 1 {
		return fmt.Errorf("Service is already in the process of started")
	}
	return startNode(ec.ctx, ec.node, ec.backend)
}

func (ec *ETHChain) wait() {
	ec.node.Wait()
}

func (ec *ETHChain) Stop() error {
	if atomic.AddInt32(&ec.shutdown, 1) != 1 {
		return fmt.Errorf("Service is already in the process of shutting down")
	}

	ec.node.Close()

	ec.wait()
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

func (ec *ETHChain) Config() *Config {
	return ec.config
}

func (ec *ETHChain) Context() *cli.Context {
	return ec.ctx
}

func NewETHChain(config *Config, args []string) (*ETHChain, error) {
	ec := &ETHChain{config: config}

	//
	app := cli.NewApp()
	app.Name = config.Node.Name
	app.Authors = []*cli.Author{
		{Name: config.Node.Name, Email: config.Node.Name},
	}
	app.Version = params.VersionWithMeta
	app.Usage = config.Node.Name

	//

	utils.CacheFlag.Value = 4096

	app.Action = func(ctx *cli.Context) error {
		ec.ctx = ctx
		prepare(ec.ctx, ec.config)
		ec.node, ec.backend, ec.ether = makeFullNode(ec.ctx, ec.config)
		return nil
	}
	app.HideVersion = true
	app.Copyright = config.Node.Name

	app.Flags = GetFlags()

	err := app.Run(args)
	if err != nil {
		return nil, err
	}

	return ec, nil
}

func prepare(ctx *cli.Context, cfg *Config) {
	log.Info(fmt.Sprintf("Prepare %s on NetWork(%d)...", cfg.Node.Name, cfg.Eth.NetworkId))

	if !ctx.IsSet(utils.NoDiscoverFlag.Name) {
		SetDNSDiscoveryDefaults(cfg)
	}

	if ctx.IsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		if !ctx.IsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
			ctx.Set(utils.MetricsInfluxDBDatabaseFlag.Name, "qng")
		}
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBV2Flag.Name) {
		if !ctx.IsSet(utils.MetricsInfluxDBBucketFlag.Name) {
			ctx.Set(utils.MetricsInfluxDBBucketFlag.Name, "qng")
		}
		if !ctx.IsSet(utils.MetricsInfluxDBOrganizationFlag.Name) {
			ctx.Set(utils.MetricsInfluxDBOrganizationFlag.Name, "qng")
		}
	}

	// Start metrics export if enabled
	utils.SetupMetrics(ctx)

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)
}

func makeFullNode(ctx *cli.Context, cfg *Config) (*node.Node, *eth.EthAPIBackend, *eth.Ethereum) {
	stack := makeConfigNode(ctx, cfg)
	if ctx.IsSet(utils.OverrideCancun.Name) {
		v := ctx.Uint64(utils.OverrideCancun.Name)
		cfg.Eth.OverrideCancun = &v
	}
	if ctx.IsSet(utils.OverrideVerkle.Name) {
		v := ctx.Uint64(utils.OverrideVerkle.Name)
		cfg.Eth.OverrideVerkle = &v
	}
	backend, ethe := utils.RegisterEthService(stack, &cfg.Eth)
	// Create gauge with geth system and build information
	if ethe != nil { // The 'eth' backend may be nil in light mode
		var protos []string
		for _, p := range ethe.Protocols() {
			protos = append(protos, fmt.Sprintf("%v/%d", p.Name, p.Version))
		}
		metrics.NewRegisteredGaugeInfo("geth/info", nil).Update(metrics.GaugeInfoValue{
			"arch":      runtime.GOARCH,
			"os":        runtime.GOOS,
			"version":   cfg.Node.Version,
			"protocols": strings.Join(protos, ","),
		})
	}
	// Configure log filter RPC API.
	filterSystem := utils.RegisterFilterAPI(stack, backend, &cfg.Eth)

	if ctx.IsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend, filterSystem, &cfg.Node)
	}
	if cfg.Ethstats.URL != "" {
		utils.RegisterEthStatsService(stack, backend, cfg.Ethstats.URL)
	}
	// Configure full-sync tester service if requested
	if ctx.IsSet(utils.SyncTargetFlag.Name) {
		hex := hexutil.MustDecode(ctx.String(utils.SyncTargetFlag.Name))
		if len(hex) != common.HashLength {
			utils.Fatalf("invalid sync target length: have %d, want %d", len(hex), common.HashLength)
		}
		utils.RegisterFullSyncTester(stack, ethe, common.BytesToHash(hex))
	}
	return stack, backend.(*eth.EthAPIBackend), ethe
}

func makeConfigNode(ctx *cli.Context, cfg *Config) *node.Node {
	filterConfig(ctx, cfg)
	// Load config file.
	if file := ctx.String(ConfigFileFlag.Name); file != "" {
		if err := LoadConfig(file, cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

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

	// Assemble the supported backends
	if len(conf.ExternalSigner) > 0 {
		log.Info("Using external signer", "url", conf.ExternalSigner)
		if extBackend, err := external.NewExternalBackend(conf.ExternalSigner); err == nil {
			am.AddBackend(extBackend)
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

func applyMetricConfig(ctx *cli.Context, cfg *Config) {
	if ctx.IsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.Bool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		log.Warn("Expensive metrics are collected by default, please remove this flag", "flag", utils.MetricsEnabledExpensiveFlag.Name)
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

func startNode(ctx *cli.Context, stack *node.Node, backend *eth.EthAPIBackend) error {
	err := stack.Start()
	if err != nil {
		return err
	}

	unlockAccounts(ctx, stack)

	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	rpcClient := stack.Attach()
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
	return nil
}

func unlockAccounts(ctx *cli.Context, stack *node.Node) {
	var unlocks []string
	inputs := strings.Split(ctx.String(utils.UnlockedAccountFlag.Name), ",")
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
	backends := stack.AccountManager().Backends(keystore.KeyStoreType)
	if len(backends) == 0 {
		log.Warn("Failed to unlock accounts, keystore is not available")
		return
	}
	ks := backends[0].(*keystore.KeyStore)
	passwords := utils.MakePasswordList(ctx)
	for i, account := range unlocks {
		UnlockAccount(ks, account, i, passwords)
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
	for i, a := range err.Matches {
		if e := ks.Unlock(a, auth); e == nil {
			match = &err.Matches[i]
			break
		}
	}
	if match == nil {
		utils.Fatalf("None of the listed files could be unlocked.")
		return accounts.Account{}
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

func filterConfig(ctx *cli.Context, cfg *Config) {
	hms := ctx.String(utils.HTTPApiFlag.Name)
	if len(hms) > 0 {
		modules := utils.SplitAndTrim(hms)
		nmodules := ""
		for _, mod := range modules {
			if mod == "miner" {
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
	if len(wms) > 0 {
		modules := utils.SplitAndTrim(wms)
		nmodules := ""
		for _, mod := range modules {
			if mod == "miner" {
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

func MakeNakedNode(config *Config, args []string) (*node.Node, *cli.Context, error) {
	app := cli.NewApp()
	app.Name = config.Node.Name
	app.Authors = []*cli.Author{
		{Name: config.Node.Name, Email: config.Node.Name},
	}
	app.Version = params.VersionWithMeta
	app.Usage = config.Node.Name

	//

	utils.CacheFlag.Value = 4096

	var n *node.Node
	var context *cli.Context
	app.Action = func(ctx *cli.Context) error {
		n = makeConfigNode(ctx, config)
		context = ctx
		return nil
	}
	app.HideVersion = true
	app.Copyright = config.Node.Name

	app.Flags = GetFlags()

	err := app.Run(args)
	if err != nil {
		return nil, nil, err
	}

	return n, context, nil
}

// MakeChain creates a chain manager from set command line flags.
func MakeChain(ctx *cli.Context, stack *node.Node, readonly bool, cfg *Config) (*core.BlockChain, ethdb.Database, error) {
	if !cfg.Eth.SyncMode.IsValid() {
		return nil, nil, fmt.Errorf("invalid sync mode %d", cfg.Eth.SyncMode)
	}
	if cfg.Eth.Miner.GasPrice == nil || cfg.Eth.Miner.GasPrice.Sign() <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", cfg.Eth.Miner.GasPrice, "updated", ethconfig.Defaults.Miner.GasPrice)
		cfg.Eth.Miner.GasPrice = new(big.Int).Set(ethconfig.Defaults.Miner.GasPrice)
	}
	if cfg.Eth.NoPruning && cfg.Eth.TrieDirtyCache > 0 {
		if cfg.Eth.SnapshotCache > 0 {
			cfg.Eth.TrieCleanCache += cfg.Eth.TrieDirtyCache * 3 / 5
			cfg.Eth.SnapshotCache += cfg.Eth.TrieDirtyCache * 2 / 5
		} else {
			cfg.Eth.TrieCleanCache += cfg.Eth.TrieDirtyCache
		}
		cfg.Eth.TrieDirtyCache = 0
	}
	log.Info("Allocated trie memory caches", "clean", common.StorageSize(cfg.Eth.TrieCleanCache)*1024*1024, "dirty", common.StorageSize(cfg.Eth.TrieDirtyCache)*1024*1024)
	// Assemble the Ethereum object
	chainDb, err := stack.OpenDatabaseWithFreezer("chaindata", cfg.Eth.DatabaseCache, cfg.Eth.DatabaseHandles, cfg.Eth.DatabaseFreezer, "eth/db/chaindata/", false)
	if err != nil {
		return nil, nil, err
	}
	scheme, err := rawdb.ParseStateScheme(cfg.Eth.StateScheme, chainDb)
	if err != nil {
		return nil, nil, err
	}
	gspec := cfg.Eth.Genesis
	chainConfig, err := core.LoadChainConfig(chainDb, gspec)
	if err != nil {
		return nil, nil, err
	}
	if cfg.Eth.ConsensusEngine == nil {
		cfg.Eth.ConsensusEngine = ethconfig.CreateDefaultConsensusEngine
	}
	engine, err := cfg.Eth.ConsensusEngine(chainConfig, chainDb)
	if err != nil {
		return nil, nil, err
	}

	var (
		vmcfg = vm.Config{
			EnablePreimageRecording: cfg.Eth.EnablePreimageRecording,
			EnableWitnessCollection: cfg.Eth.EnableWitnessCollection,
		}
		cacheConfig = &core.CacheConfig{
			TrieCleanLimit:      cfg.Eth.TrieCleanCache,
			TrieCleanNoPrefetch: cfg.Eth.NoPrefetch,
			TrieDirtyLimit:      cfg.Eth.TrieDirtyCache,
			TrieDirtyDisabled:   cfg.Eth.NoPruning,
			TrieTimeLimit:       cfg.Eth.TrieTimeout,
			SnapshotLimit:       cfg.Eth.SnapshotCache,
			Preimages:           cfg.Eth.Preimages,
			StateHistory:        cfg.Eth.StateHistory,
			StateScheme:         scheme,
		}
	)

	if cacheConfig.TrieDirtyDisabled && !cacheConfig.Preimages {
		cacheConfig.Preimages = true
		log.Info("Enabling recording of key preimages since archive mode is used")
	}
	if !ctx.Bool(utils.SnapshotFlag.Name) {
		cacheConfig.SnapshotLimit = 0 // Disabled
	}
	// If we're in readonly, do not bother generating snapshot data.
	if readonly {
		cacheConfig.SnapshotNoBuild = true
	}

	if ctx.IsSet(utils.CacheFlag.Name) || ctx.IsSet(utils.CacheTrieFlag.Name) {
		cacheConfig.TrieCleanLimit = ctx.Int(utils.CacheFlag.Name) * ctx.Int(utils.CacheTrieFlag.Name) / 100
	}
	if ctx.IsSet(utils.CacheFlag.Name) || ctx.IsSet(utils.CacheGCFlag.Name) {
		cacheConfig.TrieDirtyLimit = ctx.Int(utils.CacheFlag.Name) * ctx.Int(utils.CacheGCFlag.Name) / 100
	}

	if ctx.IsSet(utils.VMTraceFlag.Name) {
		if name := ctx.String(utils.VMTraceFlag.Name); name != "" {
			var config json.RawMessage
			if ctx.IsSet(utils.VMTraceJsonConfigFlag.Name) {
				config = json.RawMessage(ctx.String(utils.VMTraceJsonConfigFlag.Name))
			}
			t, err := tracers.LiveDirectory.New(name, config)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to create tracer %q: %v", name, err)
			}
			vmcfg.Tracer = t
		}
	}
	// Disable transaction indexing/unindexing by default.
	chain, err := core.NewBlockChain(chainDb, cacheConfig, gspec, nil, engine, vmcfg, nil)
	if err != nil {
		return nil, nil, err
	}

	return chain, chainDb, nil
}
