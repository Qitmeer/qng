// (c) 2021, the Qitmeer developers. All rights reserved.
// license that can be found in the LICENSE file.

package main

import (
	"crypto/rand"
	"github.com/Qitmeer/meerevm/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"os"
	"os/signal"
	"time"
)

func checkError(err error) {
	if err != nil { panic(err) }
}

func main() {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	chainConfig := &params.ChainConfig {
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
		LondonBlock:         nil,
		Ethash:              nil,
	}

	genBalance := big.NewInt(1000000000000000000)
	genKey, _ := meereth.NewKey(rand.Reader)

	genesis := &core.Genesis{
		Config: chainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc: core.GenesisAlloc{ genKey.Address: { Balance: genBalance }},
	}

	etherbase := common.Address{1}

	config := ethconfig.Config{
		Genesis:         genesis,
		SyncMode:        downloader.FullSync,
		DatabaseCache:   256,
		DatabaseHandles: 256,
		TxPool:          core.DefaultTxPoolConfig,
		GPO:             ethconfig.Defaults.GPO,
		Ethash:          ethconfig.Defaults.Ethash,
		Miner: miner.Config{
			Etherbase: etherbase,
			GasCeil:  genesis.GasLimit * 11 / 10,
			GasPrice: big.NewInt(1),
			Recommit: time.Second,
		},
	}
	stack, eth := meereth.New(&meereth.Config{EthConfig:&config},"./data")
	stack.Start()
	eth.Backend.StartMining(1)

	// Handle interrupts.
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)

	chainID := chainConfig.ChainID
	nonce := uint64(0)
	value := big.NewInt(1000000000000)
	gasLimit := 21000

	// LondonBlock is not nil
	gasPrice := big.NewInt(500000000)  // how the price work ?  not work
	gasPrice = big.NewInt(510000000)   // how the price work ?  work, after 5 block (height=6)
	gasPrice = big.NewInt(1000000000)  // how the price work ?  work immediately
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

	payee, err := meereth.NewKey(rand.Reader); checkError(err)

	showBalance := func() {
		state, err := eth.Backend.BlockChain().State()
		checkError(err)
		log.Info("miner account", "addr", etherbase, "balance", state.GetBalance(etherbase))
		log.Info("genesis account", "addr", genKey.Address, "balance",state.GetBalance(genKey.Address))
		log.Info("payee account", "addr", payee.Address, "balance", state.GetBalance(payee.Address))
	}

	showTxPoolStatus := func() int {
		pending, queued := eth.Backend.TxPool().Stats();
		log.Info("TxPool status", "pending", pending, "queued", queued)
		return pending
	}

	sendTxs := func() {
		// send the tx
		for i := 0; i < 9; i++ {
			tx := types.NewTransaction(nonce, payee.Address, value, uint64(gasLimit), gasPrice, nil)
			signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), genKey.PrivateKey); checkError(err)
			log.Info("Add signed tx to the local TxPool", "tx",signedTx.Hash())
			if err := eth.Backend.TxPool().AddLocal(signedTx); err != nil {
				log.Error("error when send tx", "error", err)
				continue
			}
			eth.Backend.Miner().Mining()
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
			eth.Backend.StopMining()
			stack.Close()
			return
		default:
		}
		time.Sleep(1000 * time.Millisecond)
		showTxPoolStatus()
		showBalance()
	}
}
