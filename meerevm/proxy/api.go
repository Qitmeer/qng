// Copyright (c) 2017-2018 The qitmeer developers

package proxy

import (
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/ethereum/go-ethereum/common"
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

func (api *PublicDeterministicDeploymentProxyAPI) GetContractAddress(owner string, bytecodeHex string, version int64) (interface{}, error) {
	ownerAddr := common.HexToAddress(owner)
	bytecode := common.FromHex(bytecodeHex)
	addr, err := api.proxy.GetContractAddress(ownerAddr, bytecode, version)
	if err != nil {
		return nil, err
	}
	return addr.String(), nil
}
