/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package eth

import (
	"bytes"
	"fmt"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/external"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/scwallet"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
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

func NewETHChain(config *Config, args []string, flags []cli.Flag) (*ETHChain, error) {
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

	app.Flags = flags

	err := app.Run(args)
	if err != nil {
		return nil, err
	}

	return ec, nil
}

func prepare(ctx *cli.Context, cfg *Config) {
	if cfg.Eth.Genesis.Config.ChainID.Int64() != qparams.ActiveNetParams.MeerEVMCfg.ChainID {
		return
	}

	log.Info(fmt.Sprintf("Prepare %s on NetWork(%d)...", cfg.Node.Name, cfg.Eth.NetworkId))
	// Start metrics export if enabled
	utils.SetupMetrics(ctx)

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)
}

func makeFullNode(ctx *cli.Context, cfg *Config) (*node.Node, *eth.EthAPIBackend, *eth.Ethereum) {
	stack := makeConfigNode(ctx, cfg)
	if ctx.IsSet(utils.OverrideTerminalTotalDifficulty.Name) {
		cfg.Eth.OverrideTerminalTotalDifficulty = qcommon.GlobalBig(ctx, utils.OverrideTerminalTotalDifficulty.Name)
	}
	if ctx.IsSet(utils.OverrideTerminalTotalDifficultyPassed.Name) {
		override := ctx.Bool(utils.OverrideTerminalTotalDifficultyPassed.Name)
		cfg.Eth.OverrideTerminalTotalDifficultyPassed = &override
	}
	backend, ethe := utils.RegisterEthService(stack, &cfg.Eth)

	if ethe != nil && !ctx.IsSet(utils.IgnoreLegacyReceiptsFlag.Name) {
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
			stack.Close()
			utils.Fatalf("Database has receipts with a legacy format. Please run `qng db freezer-migrate`.")
		}
	}
	// Configure log filter RPC API.
	filterSystem := utils.RegisterFilterAPI(stack, backend, &cfg.Eth)

	if ctx.IsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend, filterSystem, &cfg.Node)
	}
	if cfg.Ethstats.URL != "" {
		utils.RegisterEthStatsService(stack, backend, cfg.Ethstats.URL)
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

func applyMetricConfig(ctx *cli.Context, cfg *Config) {
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

func startNode(ctx *cli.Context, stack *node.Node, backend *eth.EthAPIBackend) error {
	err := stack.Start()
	if err != nil {
		return err
	}

	unlockAccounts(ctx, stack)

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
		backend.TxPool().SetGasPrice(gasprice)
		threads := ctx.Int(utils.MinerThreadsFlag.Name)
		if err := backend.StartMining(threads); err != nil {
			utils.Fatalf("Failed to start mining: %v", err)
		}
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
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
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

func filterConfig(ctx *cli.Context, cfg *Config) {
	if cfg.Eth.Genesis.Config.ChainID.Int64() != qparams.ActiveNetParams.MeerEVMCfg.ChainID {
		return
	}

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

func MakeNakedNode(config *Config, args []string, flags []cli.Flag) (*node.Node, error) {
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
	app.Action = func(ctx *cli.Context) error {
		n = makeConfigNode(ctx, config)
		return nil
	}
	app.HideVersion = true
	app.Copyright = config.Node.Name

	app.Flags = flags

	err := app.Run(args)
	if err != nil {
		return nil, err
	}

	return n, nil
}
