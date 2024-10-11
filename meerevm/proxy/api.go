// Copyright (c) 2017-2018 The qitmeer developers

package proxy

import (
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type PublicDeterministicDeploymentProxyAPI struct {
	proxy *DeterministicDeploymentProxy
}

func NewPublicDeterministicDeploymentProxyAPI(proxy *DeterministicDeploymentProxy) *PublicDeterministicDeploymentProxyAPI {
	pmAPI := &PublicDeterministicDeploymentProxyAPI{proxy: proxy}
	return pmAPI
}

func (ddp *DeterministicDeploymentProxy) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicDeterministicDeploymentProxyAPI(ddp),
			Public:    true,
		},
	}
}

func (api *PublicDeterministicDeploymentProxyAPI) GetContractAddress(bytecodeHex string, version int64) (interface{}, error) {
	bytecode := common.FromHex(bytecodeHex)
	addr, err := api.proxy.GetContractAddress(bytecode, version)
	if err != nil {
		return nil, err
	}
	return addr.String(), nil
}

func (api *PublicDeterministicDeploymentProxyAPI) DeployContract(owner string, bytecodeHex string, version int64, value uint64, gas uint64) (interface{}, error) {
	ownerAddr := common.HexToAddress(owner)
	bytecode := common.FromHex(bytecodeHex)
	var val *big.Int
	if value > 0 {
		val = big.NewInt(0)
		val.SetUint64(value)
	}
	txHash, err := api.proxy.DeployContract(ownerAddr, bytecode, version, val, gas)
	if err != nil {
		return nil, err
	}
	return txHash.String(), nil
}

func (api *PublicDeterministicDeploymentProxyAPI) GetContractDeployData(bytecodeHex string, version int64) (interface{}, error) {
	bytecode := common.FromHex(bytecodeHex)
	ret := api.proxy.GetContractDeployData(bytecode, version)
	return common.Bytes2Hex(ret), nil
}

type proxyInfo struct {
	Name string `json:"name"`
	Addr string `json:"address"`
	Code string `json:"code"`
}

func (api *PublicDeterministicDeploymentProxyAPI) ProxyInfo() (interface{}, error) {
	return *api.proxy.Info(), nil
}

func (api *PublicDeterministicDeploymentProxyAPI) DeployProxy(owner string) (interface{}, error) {
	err := api.proxy.Deploy(common.HexToAddress(owner))
	if err != nil {
		return nil, err
	}
	return api.proxy.GetAddress().String(), nil
}
