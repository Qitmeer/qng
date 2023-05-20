/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package evm

import (
	"encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/address"
	qtypes "github.com/Qitmeer/qng/core/types"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/vm/consensus"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"runtime"
)

// meerevm ID of the platform
const (
	MeerEVMID = "meerevm"

	txSlotSize = 32 * 1024
	txMaxSize  = 4 * txSlotSize
)

type VM struct {
	ctx consensus.Context

	chain  *eth.ETHChain
	mchain *meer.MeerChain
}

func (vm *VM) GetID() string {
	return MeerEVMID
}

func (vm *VM) Initialize(ctx consensus.Context) error {
	log.Info("System info", "ETH VM Version", meer.Version, "Go version", runtime.Version())
	log.Debug(fmt.Sprintf("Initialize:%s", ctx.GetConfig().DataDir))

	vm.ctx = ctx

	//
	mchain, err := meer.NewMeerChain(ctx)
	if err != nil {
		return err
	}
	vm.mchain = mchain
	vm.chain = vm.mchain.ETHChain()
	return nil
}

func (vm *VM) Bootstrapping() error {
	log.Debug("Bootstrapping")
	err := vm.chain.Start()
	if err != nil {
		return err
	}
	//
	rpcClient, err := vm.chain.Node().Attach()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to attach to self: %v", err))
	}
	client := ethclient.NewClient(rpcClient)

	blockNum, err := client.BlockNumber(vm.ctx)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Debug(fmt.Sprintf("MeerETH block chain current block number:%d", blockNum))
	}

	cbh := vm.chain.Ether().BlockChain().CurrentBlock()
	if cbh != nil {
		log.Debug(fmt.Sprintf("MeerETH block chain current block:number=%d hash=%s", cbh.Number.Uint64(), cbh.Hash().String()))
	}

	//
	state, err := vm.chain.Ether().BlockChain().State()
	if err != nil {
		return nil
	}

	log.Debug(fmt.Sprintf("Etherbase:%v balance:%v", vm.chain.Config().Eth.Miner.Etherbase, state.GetBalance(vm.chain.Config().Eth.Miner.Etherbase)))

	//
	for addr := range vm.chain.Config().Eth.Genesis.Alloc {
		log.Debug(fmt.Sprintf("Alloc address:%v balance:%v", addr.String(), state.GetBalance(addr)))
	}

	vm.mchain.Start()
	//
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

	err := vm.chain.Stop()
	if err != nil {
		log.Error(err.Error())
	}
	vm.mchain.Stop()
	return nil
}

func (vm *VM) Version() string {
	result := map[string]string{}
	result["MeerVer"] = meer.Version
	result["EvmVer"] = vm.chain.Config().Node.Version
	result["ChainID"] = vm.chain.Ether().BlockChain().Config().ChainID.String()
	result["NetworkId"] = fmt.Sprintf("%d", vm.chain.Config().Eth.NetworkId)
	if len(vm.chain.Config().Node.HTTPHost) > 0 {
		result["http"] = fmt.Sprintf("http://%s:%d", vm.chain.Config().Node.HTTPHost, vm.chain.Config().Node.HTTPPort)
	}
	if len(vm.chain.Config().Node.WSHost) > 0 {
		result["ws"] = fmt.Sprintf("ws://%s:%d", vm.chain.Config().Node.WSHost, vm.chain.Config().Node.WSPort)
	}

	resultJson, err := json.Marshal(result)
	if err != nil {
		log.Error(err.Error())
		return ""
	}
	return string(resultJson)
}

func (vm *VM) GetBlock(bh *hash.Hash) (consensus.Block, error) {
	blockh := vm.chain.Ether().BlockChain().CurrentBlock()
	block := vm.chain.Ether().BlockChain().GetBlockByHash(blockh.Hash())
	h := hash.MustBytesToHash(block.Hash().Bytes())
	return &Block{id: &h, ethBlock: block, vm: vm, status: consensus.Accepted}, nil
}

func (vm *VM) GetBlockByNumber(num uint64) (consensus.Block, error) {
	block := vm.chain.Ether().BlockChain().GetBlockByNumber(num)
	if block == nil {
		return nil, fmt.Errorf("No block number:%d", num)
	}
	h := hash.MustBytesToHash(block.Hash().Bytes())
	return &Block{id: &h, ethBlock: block, vm: vm, status: consensus.Accepted}, nil
}

func (vm *VM) BuildBlock(txs []model.Tx) (consensus.Block, error) {
	return nil, nil
}

func (vm *VM) CheckConnectBlock(block consensus.Block) error {
	return vm.mchain.CheckConnectBlock(block)
}

func (vm *VM) ConnectBlock(block consensus.Block) (uint64, error) {
	return vm.mchain.ConnectBlock(block)
}

func (vm *VM) DisconnectBlock(block consensus.Block) (uint64, error) {
	return 0, nil
}

func (vm *VM) RewindTo(state model.BlockState) error {
	return vm.mchain.RewindTo(state)
}

func (vm *VM) ParseBlock([]byte) (consensus.Block, error) {
	return nil, nil
}

func (vm *VM) LastAccepted() (*hash.Hash, error) {
	block := vm.chain.Ether().BlockChain().CurrentBlock()
	h := hash.MustBytesToHash(block.Hash().Bytes())
	return &h, nil
}

func (vm *VM) GetBalance(addre string) (int64, error) {
	var eAddr common.Address
	if common.IsHexAddress(addre) {
		eAddr = common.HexToAddress(addre)
	} else {
		addr, err := address.DecodeAddress(addre)
		if err != nil {
			return 0, err
		}
		if !addr.IsForNetwork(params.ActiveNetParams.Net) {
			return 0, fmt.Errorf("network error:%s", addr.String())
		}
		secpPksAddr, ok := addr.(*address.SecpPubKeyAddress)
		if !ok {
			return 0, fmt.Errorf("Not SecpPubKeyAddress:%s", addr.String())
		}
		publicKey, err := crypto.UnmarshalPubkey(secpPksAddr.PubKey().SerializeUncompressed())
		if err != nil {
			return 0, err
		}
		eAddr = crypto.PubkeyToAddress(*publicKey)
	}
	state, err := vm.chain.Ether().BlockChain().State()
	if err != nil {
		return 0, err
	}
	ba := state.GetBalance(eAddr)
	if ba == nil {
		return 0, fmt.Errorf("No balance for address %s", eAddr)
	}
	ba = ba.Div(ba, qcommon.Precision)
	return ba.Int64(), nil
}

func (vm *VM) VerifyTx(tx model.Tx) (int64, error) {
	if tx.GetTxType() == qtypes.TxTypeCrossChainVM {
		txb := common.FromHex(string(tx.GetData()))
		var txe = &types.Transaction{}
		if err := txe.UnmarshalBinary(txb); err != nil {
			return 0, fmt.Errorf("rlp decoding failed: %v", err)
		}
		err := vm.validateTx(txe, true)
		if err != nil {
			return 0, err
		}
		cost := txe.Cost()
		cost = cost.Sub(cost, txe.Value())
		cost = cost.Div(cost, qcommon.Precision)
		return cost.Int64(), nil
	}
	return 0, fmt.Errorf("Not support")
}

func (vm *VM) VerifyTxSanity(tx model.Tx) error {
	if tx.GetTxType() == qtypes.TxTypeCrossChainVM {
		txb := common.FromHex(string(tx.GetData()))
		var txe = &types.Transaction{}
		if err := txe.UnmarshalBinary(txb); err != nil {
			return fmt.Errorf("rlp decoding failed: %v", err)
		}
		return vm.validateTx(txe, false)
	}
	return fmt.Errorf("Not support")
}

func (vm *VM) validateTx(tx *types.Transaction, checkState bool) error {
	// Reject transactions over defined size to prevent DOS attacks
	if tx.Size() > txMaxSize {
		return txpool.ErrOversizedData
	}
	if tx.Value().Sign() < 0 {
		return txpool.ErrNegativeValue
	}
	if tx.GasFeeCap().BitLen() > 256 {
		return core.ErrFeeCapVeryHigh
	}
	if tx.GasTipCap().BitLen() > 256 {
		return core.ErrTipVeryHigh
	}
	if tx.GasFeeCapIntCmp(tx.GasTipCap()) < 0 {
		return core.ErrTipAboveFeeCap
	}
	from, err := types.Sender(types.LatestSigner(vm.chain.Ether().BlockChain().Config()), tx)
	if err != nil {
		return txpool.ErrInvalidSender
	}
	if checkState {
		currentState, err := vm.chain.Ether().BlockChain().State()
		if err != nil {
			return err
		}
		if currentState.GetNonce(from) > tx.Nonce() {
			return core.ErrNonceTooLow
		}
		if currentState.GetBalance(from).Cmp(tx.Cost()) < 0 {
			return core.ErrInsufficientFunds
		}
	}
	intrGas, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), tx.To() == nil, true, true, false)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return core.ErrIntrinsicGas
	}
	return nil
}

func (vm *VM) AddTxToMempool(tx *qtypes.Transaction, local bool) (int64, error) {
	return vm.mchain.MeerPool().AddTx(tx, local)
}

func (vm *VM) GetTxsFromMempool() ([]*qtypes.Transaction, []*hash.Hash, error) {
	return vm.mchain.MeerPool().GetTxs()
}

func (vm *VM) GetMempoolSize() int64 {
	return vm.mchain.MeerPool().GetSize()
}

func (vm *VM) RemoveTxFromMempool(tx *qtypes.Transaction) error {
	return vm.mchain.MeerPool().RemoveTx(tx)
}

func (vm *VM) RegisterAPIs(apis []api.API) {
	vm.mchain.RegisterAPIs(apis)
}

func (vm *VM) SetLogLevel(level string) {
	eth.InitLog(level, vm.ctx.GetConfig().DebugPrintOrigins)
}

func (vm *VM) ResetTemplate() error {
	return vm.mchain.MeerPool().ResetTemplate()
}

func (vm *VM) Genesis() *hash.Hash {
	mbb := vm.chain.Ether().BlockChain().Genesis().Hash().Bytes()
	qcommon.ReverseBytes(&mbb)
	nmbb, err := hash.NewHash(mbb)
	if err != nil {
		return nil
	}
	return nmbb
}

func (vm *VM) GetBlockIDByTxHash(txhash *hash.Hash) uint64 {
	tx, _, blockNumber, _, _ := vm.chain.Backend().GetTransaction(nil, qcommon.ToEVMHash(txhash))
	if tx == nil {
		return 0
	}
	return blockNumber
}

func (vm *VM) GetCurStateRoot() common.Hash {
	return vm.chain.Ether().BlockChain().CurrentBlock().Root
}

func (vm *VM) GetCurHeader() *types.Header {
	return vm.chain.Ether().BlockChain().CurrentBlock()
}

func (vm *VM) BlockChain() *core.BlockChain {
	return vm.chain.Ether().BlockChain()
}

func (vm *VM) ChainDatabase() ethdb.Database {
	return vm.chain.Ether().ChainDb()
}

func (vm *VM) PrepareEnvironment(state model.BlockState) (*types.Header, error) {
	return vm.mchain.PrepareEnvironment(state)
}

func New() *VM {
	return &VM{}
}
