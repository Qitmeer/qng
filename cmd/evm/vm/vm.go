/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package vm

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/Qitmeer/meerevm/cmd/evm/util"
	"github.com/Qitmeer/meerevm/eth"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"sync"
	"time"
)

// ID of the platform VM
var (
	ID = "meerevm"
)

type VM struct {
	ctx          context.Context
	shutdownChan chan struct{}
	shutdownWg   sync.WaitGroup

	config *ethconfig.Config
	node   *node.Node
	chain  *meereth.Ether
	client *ethclient.Client
}

func (vm *VM) Initialize(ctx context.Context) error {
	log.Debug("Initialize")

	vm.shutdownChan = make(chan struct{}, 1)
	vm.ctx = ctx

	//
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
	genKey, _ := meereth.NewKey(rand.Reader)

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
		TxPool:          core.DefaultTxPoolConfig,
		GPO:             ethconfig.Defaults.GPO,
		Ethash:          ethconfig.Defaults.Ethash,
		Miner: miner.Config{
			Etherbase: etherbase,
			GasCeil:   genesis.GasLimit * 11 / 10,
			GasPrice:  big.NewInt(1),
			Recommit:  time.Second,
		},
	}
	vm.node, vm.chain = meereth.New(&meereth.Config{EthConfig: &config})

	return nil
}

func (vm *VM) Bootstrapping() error {
	log.Debug("Bootstrapping")
	vm.node.Start()
	//
	rpcClient, err := vm.node.Attach()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to attach to self: %v", err))
	}
	vm.client = ethclient.NewClient(rpcClient)

	blockNum, err := vm.client.BlockNumber(vm.ctx)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Info(fmt.Sprintf("Current block number:%d", blockNum))
	}

	//
	state, err := vm.chain.Backend.BlockChain().State()
	if err != nil {
		return nil
	}

	log.Info(fmt.Sprintf("miner account,addr:%v balance:%v", vm.config.Miner.Etherbase, state.GetBalance(vm.config.Miner.Etherbase)))

	return nil
}

func (vm *VM) Bootstrapped() error {
	log.Debug("Bootstrapped")
	return nil
}

func (vm *VM) Shutdown() error {
	log.Debug("Shutdown")
	if vm.ctx == nil {
		return nil
	}

	vm.client.Close()
	vm.node.Close()

	close(vm.shutdownChan)
	vm.shutdownWg.Wait()
	return nil
}

func (vm *VM) Version() (string, error) {
	return util.Version, nil
}

func (vm *VM) GetBlock(*hash.Hash) (consensus.Block, error) {
	return nil, nil
}

func (vm *VM) BuildBlock() (consensus.Block, error) {

	blocks, _ := core.GenerateChain(vm.config.Genesis.Config, vm.chain.Backend.BlockChain().CurrentBlock(), vm.chain.Backend.Engine(), vm.chain.Backend.ChainDb(), 1, func(i int, block *core.BlockGen) {
		//block.SetCoinbase(common.Address{0x00})
	})
	if len(blocks) != 1 {
		return nil, fmt.Errorf("BuildBlock error")
	}
	num, err := vm.chain.Backend.BlockChain().InsertChain(blocks)
	if err != nil {
		return nil, err
	}
	if num != 1 {
		return nil, fmt.Errorf("BuildBlock error")
	}

	log.Info(fmt.Sprintf("BuildBlock:number=%d hash=%s", blocks[0].Number().Uint64(), blocks[0].Hash().String()))

	h := hash.MustHexToHash(blocks[0].Hash().String())
	return &Block{id: &h, ethBlock: *blocks[0], vm: vm, status: consensus.Accepted}, nil
}

func (vm *VM) ParseBlock([]byte) (consensus.Block, error) {
	return nil, nil
}

func (vm *VM) LastAccepted() (*hash.Hash, error) {
	return nil, nil
}
