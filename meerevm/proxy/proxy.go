package proxy

import (
	"context"
	"errors"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
)

type DeterministicDeploymentProxy struct {
	ctx context.Context
	rpc *ethclient.Client
}

func (ddp DeterministicDeploymentProxy) GetAddress() *common.Address {
	return &params.ActiveNetParams.DeterministicDeploymentProxyAddr
}

func (ddp DeterministicDeploymentProxy) GetContractAddress(owner common.Address, bytecode []byte, version int64) (common.Address, error) {
	if ddp.GetAddress().Cmp(common.Address{}) == 0 {
		return common.Address{}, errors.New("No support DeterministicDeploymentProxy")
	}
	msg := ethereum.CallMsg{
		From: owner,
		To:   ddp.GetAddress(),
		Data: ddp.GetContractDeployData(bytecode, version),
	}
	addrBytes, err := ddp.rpc.CallContract(ddp.ctx, msg, nil)
	if err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(addrBytes), nil
}

func (ddp DeterministicDeploymentProxy) DeployContract(owner common.Address, bytecode []byte, version int64, value *big.Int, gas uint64) (common.Hash, error) {
	if ddp.GetAddress().Cmp(common.Address{}) == 0 {
		return common.Hash{}, errors.New("No support DeterministicDeploymentProxy")
	}
	arg := map[string]interface{}{
		"from":  owner,
		"to":    ddp.GetAddress(),
		"input": hexutil.Bytes(ddp.GetContractDeployData(bytecode, version)),
	}
	if gas != 0 {
		arg["gas"] = hexutil.Uint64(gas)
	}
	if value != nil {
		arg["value"] = (*hexutil.Big)(value)
	}
	var hashBytes hexutil.Bytes
	err := ddp.rpc.Client().CallContext(ddp.ctx, &hashBytes, "eth_sendTransaction", arg)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(hashBytes), nil
}

func (ddp DeterministicDeploymentProxy) GetContractDeployData(bytecode []byte, version int64) []byte {
	salt := common.BigToHash(big.NewInt(version))
	data := []byte{}
	data = append(salt.Bytes(), bytecode...)
	return data
}

func NewDeterministicDeploymentProxy(ctx context.Context, rpc *ethclient.Client) *DeterministicDeploymentProxy {
	return &DeterministicDeploymentProxy{
		ctx: ctx,
		rpc: rpc,
	}
}
