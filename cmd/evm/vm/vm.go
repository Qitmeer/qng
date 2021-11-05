/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package vm

import (
	"fmt"
	"github.com/Qitmeer/meerevm/cmd/evm/util"
	"github.com/Qitmeer/meerevm/eth"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"sync"
	"time"
)

type VM struct {
	ctx          *consensus.Context
	shutdownChan chan struct{}
	shutdownWg   sync.WaitGroup

	config *ethconfig.Config
	node   *node.Node
	chain  *meereth.Ether

	glog *log.GlogHandler
}

func (vm *VM) Initialize(ctx *consensus.Context) error {
	//log.Glogger().Verbosity(log.LvlTrace)
	lvl, err := log.LvlFromString(ctx.LogLevel)
	if err == nil {
		vm.glog.Verbosity(lvl)
	}

	log.Info(fmt.Sprintf("Initialize:%s", ctx.Datadir))

	vm.shutdownChan = make(chan struct{}, 1)
	vm.ctx = ctx

	//

	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(520),
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
	genAddress := common.HexToAddress("0x71bc4403Af41634Cda7C32600A8024d54e7F6499")

	genesis := &core.Genesis{
		Config:     chainConfig,
		Nonce:      0,
		Number:     0,
		ExtraData:  hexutil.MustDecode("0x00"),
		GasLimit:   100000000,
		Difficulty: big.NewInt(0),
		Alloc:      core.GenesisAlloc{genAddress: {Balance: genBalance}},
	}

	etherbase := genAddress

	vm.config = &ethconfig.Config{
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
		TrieCleanCache: 256,
	}
	vm.node, vm.chain = meereth.New(&meereth.Config{EthConfig: vm.config}, vm.ctx.Datadir)

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
	client := ethclient.NewClient(rpcClient)

	blockNum, err := client.BlockNumber(vm.ctx)
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

	vm.node.Close()

	close(vm.shutdownChan)
	vm.shutdownWg.Wait()
	return nil
}

func (vm *VM) Version() (string, error) {
	return util.Version + fmt.Sprintf("(eth:%s)", params.VersionWithMeta), nil
}

func (vm *VM) GetBlock(bh *hash.Hash) (consensus.Block, error) {
	block := vm.chain.Backend.BlockChain().CurrentBlock()
	h := hash.MustBytesToHash(block.Hash().Bytes())
	return &Block{id: &h, ethBlock: block, vm: vm, status: consensus.Accepted}, nil
}

func (vm *VM) BuildBlock(txs []string) (consensus.Block, error) {
	blocks, _ := core.GenerateChain(vm.config.Genesis.Config, vm.chain.Backend.BlockChain().CurrentBlock(), vm.chain.Backend.Engine(), vm.chain.Backend.ChainDb(), 1, func(i int, block *core.BlockGen) {
		block.SetCoinbase(vm.config.Miner.Etherbase)
		for _, tx := range txs {
			txb := common.FromHex(tx)
			var tx = &types.Transaction{}
			if err := tx.UnmarshalBinary(txb); err != nil {
				log.Error(fmt.Sprintf("rlp decoding failed: %v", err))
				continue
			}
			block.AddTx(tx)
		}
	})
	if len(blocks) != 1 {
		return nil, fmt.Errorf("BuildBlock error")
	}
	num, err := vm.chain.Backend.BlockChain().InsertChainWithoutSealVerification(blocks[0])
	if err != nil {
		return nil, err
	}
	if num != 1 {
		return nil, fmt.Errorf("BuildBlock error")
	}

	log.Info(fmt.Sprintf("BuildBlock:number=%d hash=%s txs=%d", blocks[0].Number().Uint64(), blocks[0].Hash().String(), len(blocks[0].Transactions())))

	h := hash.MustBytesToHash(blocks[0].Hash().Bytes())
	return &Block{id: &h, ethBlock: blocks[0], vm: vm, status: consensus.Accepted}, nil
}

func (vm *VM) ParseBlock([]byte) (consensus.Block, error) {
	return nil, nil
}

func (vm *VM) LastAccepted() (*hash.Hash, error) {
	block := vm.chain.Backend.BlockChain().CurrentBlock()
	h := hash.MustBytesToHash(block.Hash().Bytes())
	return &h, nil
}

func NewVM(glog *log.GlogHandler) *VM {
	return &VM{glog: glog}
}
