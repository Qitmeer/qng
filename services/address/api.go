// Copyright (c) 2017-2018 The qitmeer developers

package address

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/v2/common/encode/base58"
	"github.com/Qitmeer/qng/v2/config"
	qjson "github.com/Qitmeer/qng/v2/core/json"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/Qitmeer/qng/v2/rpc"
	"github.com/Qitmeer/qng/v2/rpc/api"
	"github.com/Qitmeer/qng/v2/rpc/client/cmds"
	"github.com/Qitmeer/qng/v2/services/common"
	"sync"
)

type AddressApi struct {
	sync.Mutex
	params *params.Params
	config *config.Config
}

type PublicAddressAPI struct {
	addressApi *AddressApi
}

func NewAddressApi(cfg *config.Config, par *params.Params) *AddressApi {
	return &AddressApi{
		config: cfg,
		params: par,
	}
}

func NewPublicAddressAPI(ai *AddressApi) *PublicAddressAPI {
	pmAPI := &PublicAddressAPI{addressApi: ai}
	return pmAPI
}

func (c *AddressApi) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicAddressAPI(c),
			Public:    true,
		},
		{
			NameSpace: cmds.TestNameSpace,
			Service:   NewPrivateAddressAPI(c),
			Public:    false,
		},
	}
}

func (api *PublicAddressAPI) CheckAddress(address string, network string) (interface{}, error) {
	_, ver, err := base58.QitmeerCheckDecode(address)
	if err != nil {
		return false, rpc.RpcInvalidError("Invalid address :%s", err.Error())
	}
	var p *params.Params
	switch network {
	case "privnet":
		p = &params.PrivNetParams
	case "testnet":
		p = &params.TestNetParams
	case "mainnet":
		p = &params.MainNetParams
	case "mixnet":
		p = &params.MixNetParams
	case "amananet":
		p = &params.AmanaNetParams
	default:
		return false, rpc.RpcInvalidError("Invalid network : privnet | testnet | mainnet | mixnet | amananet")
	}
	if p.PubKeyHashAddrID != ver {
		return false, rpc.RpcRuleError("address prefix error , need %s , actual: %s,network not match,please check it",
			p.NetworkAddressPrefix, address[0:1])
	}
	return true, nil
}

// private
type PrivateAddressAPI struct {
	addressApi *AddressApi
}

func NewPrivateAddressAPI(ai *AddressApi) *PrivateAddressAPI {
	pmAPI := &PrivateAddressAPI{addressApi: ai}
	return pmAPI
}

func (api *PrivateAddressAPI) GetAddresses(privateKeyHex string) (interface{}, error) {
	privateKey, addr, eaddr, err := common.NewAddresses(privateKeyHex)
	if err != nil {
		return nil, err
	}
	result := qjson.OrderedResult{
		qjson.KV{Key: "PrivateKey", Val: hex.EncodeToString(privateKey.Serialize())},
		qjson.KV{Key: "PKHAddress", Val: addr.PKHAddress().String()},
		qjson.KV{Key: "PKAddress", Val: addr.String()},
		qjson.KV{Key: "MeerEVM Address", Val: eaddr.String()},
	}
	return result, nil
}
