// (c) 2021, the Qitmeer developers. All rights reserved.
// license that can be found in the LICENSE file.

package main

import (
	"crypto/rand"
	"fmt"
	mcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      true,
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
		//		LondonBlock:         big.NewInt(0),
		LondonBlock: nil,
		Ethash:      nil,
	}

	genBalance := big.NewInt(1000000000000000000)
	genKey, _ := mcommon.NewKey(rand.Reader)

	genesis := &core.Genesis{
		Config:     chainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      core.GenesisAlloc{genKey.Address: {Balance: genBalance}},
	}

	etherbase := common.Address{1}

	config := ethconfig.Config{
		Genesis:         genesis,
		SyncMode:        downloader.FullSync,
		DatabaseCache:   256,
		DatabaseHandles: 256,
		TxPool:          txpool.DefaultConfig,
		GPO:             ethconfig.Defaults.GPO,
		Ethash:          ethconfig.Defaults.Ethash,
		Miner: miner.Config{
			Etherbase: etherbase,
			GasCeil:   genesis.GasLimit * 11 / 10,
			GasPrice:  big.NewInt(1),
			Recommit:  time.Second,
		},
		ConsensusEngine: ethconfig.CreateDefaultConsensusEngine,
	}
	datadir, err := filepath.Abs("./data")
	if err != nil {
		fmt.Println(err)
		return
	}
	edatadir := filepath.Join(datadir, meer.ClientIdentifier)

	ecethash := ethconfig.Defaults.Ethash
	ecethash.DatasetDir = filepath.Join(edatadir, "dataset")
	config.Ethash = ecethash

	nodeConf := node.Config{
		Name:                meer.ClientIdentifier,
		Version:             params.VersionWithMeta,
		DataDir:             edatadir,
		KeyStoreDir:         filepath.Join(edatadir, "keystore"),
		HTTPHost:            node.DefaultHTTPHost,
		HTTPPort:            node.DefaultHTTPPort,
		HTTPModules:         []string{"net", "web3", "eth"},
		HTTPVirtualHosts:    []string{"localhost"},
		HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
		WSHost:              node.DefaultWSHost,
		WSPort:              node.DefaultWSPort,
		WSModules:           []string{"net", "web3"},
		GraphQLVirtualHosts: []string{"localhost"},
		P2P: p2p.Config{
			MaxPeers:    0,
			DiscoveryV5: false,
			NoDiscovery: true,
			NoDial:      true,
		},
		Logger: nil,
	}

	ethchain, err := eth.NewETHChain(&eth.Config{
		Eth:     config,
		Node:    nodeConf,
		Metrics: metrics.DefaultConfig,
	}, mcommon.ProcessEnv("", meer.ClientIdentifier))

	err = ethchain.Start()
	if err != nil {
		fmt.Println(err)
		return
	}

	ethchain.Backend().StartMining(1)

	// Handle interrupts.
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)

	chainID := chainConfig.ChainID
	nonce := uint64(0)
	value := big.NewInt(1000000000000)
	gasLimit := 21000

	// LondonBlock is not nil
	gasPrice := big.NewInt(500000000) // how the price work ?  not work
	gasPrice = big.NewInt(510000000)  // how the price work ?  work, after 5 block (height=6)
	gasPrice = big.NewInt(1000000000) // how the price work ?  work immediately
	// The working price from the example: miner/stress/1559/main.go
	// gasPrice = big.NewInt(100000000000+mrand.Int63n(65536))

	// LondonBlock is nil
	gasPrice = big.NewInt(10)
	//genesis balance=999,990,999,998,110,000
	//payee balance=9,000,000,000,000
	gasPrice = big.NewInt(100)
	//genesis balance=999,990,999,981,100,000
	//payee balance=9,000,000,000,000
	gasPrice = big.NewInt(1)
	//genesis balance=999,990,999,999,811,000
	//payee   balance=9,000,000,000,000
	//21000*9 = 189000 => gas=189,000, price = 1 => fees = 189,000

	payee, err := mcommon.NewKey(rand.Reader)
	checkError(err)

	showBalance := func() {
		state, err := ethchain.Ether().BlockChain().State()
		checkError(err)
		log.Info("miner account", "addr", etherbase, "balance", state.GetBalance(etherbase))
		log.Info("genesis account", "addr", genKey.Address, "balance", state.GetBalance(genKey.Address))
		log.Info("payee account", "addr", payee.Address, "balance", state.GetBalance(payee.Address))
	}

	showTxPoolStatus := func() int {
		pending, queued := ethchain.Backend().TxPool().Stats()
		log.Info("TxPool status", "pending", pending, "queued", queued)
		return pending
	}

	sendTxs := func() {
		// send the tx
		for i := 0; i < 9; i++ {
			tx := types.NewTransaction(nonce, payee.Address, value, uint64(gasLimit), gasPrice, nil)
			signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), genKey.PrivateKey)
			checkError(err)
			log.Info("Add signed tx to the local TxPool", "tx", signedTx.Hash())
			if err := ethchain.Backend().TxPool().AddLocal(signedTx); err != nil {
				log.Error("error when send tx", "error", err)
				continue
			}
			ethchain.Backend().Miner().Mining()
			nonce++
			// Wait if we're too saturated
			if pend := showTxPoolStatus(); pend > 4192 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	showBalance()
	sendTxs()
	showBalance()

	for {
		// Stop when interrupted.
		select {
		case <-interruptCh:
			log.Info("Got interrupt, shutting down...")
			ethchain.Ether().StopMining()
			ethchain.Stop()
			return
		default:
		}
		time.Sleep(1000 * time.Millisecond)
		showTxPoolStatus()
		showBalance()
	}
}
