/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package eth

import (
	"fmt"
	qcommon "github.com/Qitmeer/qng/v2/services/common"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli/v2"
	"os"
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
	app.Version = config.Node.Version
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

	if cfg.Node.P2P.NoDiscovery {
		ctx.Set(utils.NoDiscoverFlag.Name, "true")
	}
	if !ctx.IsSet(utils.NoDiscoverFlag.Name) {
		SetDNSDiscoveryDefaults(cfg)
	}
}

func makeFullNode(ctx *cli.Context, cfg *Config) (*node.Node, *eth.EthAPIBackend, *eth.Ethereum) {
	stack := makeConfigNode(ctx, cfg)
	if ctx.IsSet(utils.OverrideOsaka.Name) {
		v := ctx.Uint64(utils.OverrideOsaka.Name)
		cfg.Eth.OverrideOsaka = &v
	}
	if ctx.IsSet(utils.OverrideVerkle.Name) {
		v := ctx.Uint64(utils.OverrideVerkle.Name)
		cfg.Eth.OverrideVerkle = &v
	}

	// Start metrics export if enabled
	utils.SetupMetrics(&cfg.Metrics)

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
	return stack, backend, ethe
}

func makeConfigNode(ctx *cli.Context, cfg *Config) *node.Node {
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
	if err := SetAccountManagerBackends(stack.Config(), stack.AccountManager(), stack.KeyStoreDir()); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	SetAccountConfig(ctx, stack, &cfg.Eth)
	if ctx.IsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.String(utils.EthStatsURLFlag.Name)
	}
	applyMetricConfig(ctx, cfg)

	return stack
}

func SetAccountManagerBackends(conf *node.Config, am *accounts.Manager, keydir string) error {
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
	// For now, we're using EITHER external signer OR local signers.
	// If/when we implement some form of lockfile for USB and keystore wallets,
	// we can have both, but it's very confusing for the user to see the same
	// accounts in both externally and locally, plus very racey.
	am.AddBackend(qcommon.NewQngKeyStore(keystore.NewKeyStore(keydir, scryptN, scryptP)))
	if conf.USB {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			am.AddBackend(ledgerhub)
		}
		// Start a USB hub for Trezor hardware wallets (HID version)
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start HID Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
		// Start a USB hub for Trezor hardware wallets (WebUSB version)
		if trezorhub, err := usbwallet.NewTrezorHubWithWebUSB(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start WebUSB Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
	}
	if len(conf.SmartCardDaemonPath) > 0 {
		// Start a smart card hub
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
	// Sanity-check the commandline flags. It is fine if some unused fields is part
	// of the toml-config, but we expect the commandline to only contain relevant
	// arguments, otherwise it indicates an error.
	var (
		enableExport   = ctx.Bool(utils.MetricsEnableInfluxDBFlag.Name)
		enableExportV2 = ctx.Bool(utils.MetricsEnableInfluxDBV2Flag.Name)
	)
	if enableExport || enableExportV2 {
		v1FlagIsSet := ctx.IsSet(utils.MetricsInfluxDBUsernameFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBPasswordFlag.Name)

		v2FlagIsSet := ctx.IsSet(utils.MetricsInfluxDBTokenFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBOrganizationFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBBucketFlag.Name)

		if enableExport && v2FlagIsSet {
			utils.Fatalf("Flags --influxdb.metrics.organization, --influxdb.metrics.token, --influxdb.metrics.bucket are only available for influxdb-v2")
		} else if enableExportV2 && v1FlagIsSet {
			utils.Fatalf("Flags --influxdb.metrics.username, --influxdb.metrics.password are only available for influxdb-v1")
		}
	}
}

func startNode(ctx *cli.Context, stack *node.Node, backend *eth.EthAPIBackend) error {
	err := stack.Start()
	if err != nil {
		return err
	}
	if ctx.IsSet(utils.UnlockedAccountFlag.Name) {
		log.Warn(`The "unlock" flag has been deprecated and has no effect`)
	}

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

func MakeNakedNode(config *Config, args []string) (*node.Node, *cli.Context, error) {
	app := cli.NewApp()
	app.Name = config.Node.Name
	app.Authors = []*cli.Author{
		{Name: config.Node.Name, Email: config.Node.Name},
	}
	app.Version = config.Node.Version
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

func SetAccountConfig(ctx *cli.Context, stack *node.Node, cfg *ethconfig.Config) {
	var (
		acct       accounts.Account
		passphrase string
	)
	if path := ctx.Path(utils.PasswordFileFlag.Name); path != "" {
		if text, err := os.ReadFile(path); err != nil {
			utils.Fatalf("Failed to read password file: %v", err)
		} else {
			if lines := strings.Split(string(text), "\n"); len(lines) > 0 {
				passphrase = strings.TrimRight(lines[0], "\r") // Sanitise DOS line endings.
			}
		}
	}
	// Unlock the account by local keystore.
	var ks *qcommon.QngKeyStore
	if keystores := stack.AccountManager().Backends(keystore.KeyStoreType); len(keystores) > 0 {
		ks = keystores[0].(*qcommon.QngKeyStore)
	}
	if ks == nil {
		return
	}

	// Figure out the account address.
	if cfg.Miner.PendingFeeRecipient != (common.Address{}) {
		acct = accounts.Account{Address: cfg.Miner.PendingFeeRecipient}
	} else if accs := ks.Accounts(); len(accs) > 0 {
		acct = ks.Accounts()[0]
		// Make sure the address is configured as fee recipient, otherwise
		// the miner will fail to start.
		cfg.Miner.PendingFeeRecipient = acct.Address
	} else {
		return
	}
	if !ks.HasAddress(acct.Address) {
		log.Info("Can't find keystore", "address", acct.Address)
		return
	}
	if err := ks.Unlock(acct, passphrase); err != nil {
		utils.Fatalf("Failed to unlock account: %v", err)
	}
	log.Info("Using eth miner account", "address", acct.Address)
}
