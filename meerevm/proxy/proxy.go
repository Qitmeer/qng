package proxy

import (
	"context"
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
	var addrBytes hexutil.Bytes
	err := ddp.rpc.Client().CallContext(ddp.ctx, &addrBytes, "eth_sendTransaction", arg)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(addrBytes), nil
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
