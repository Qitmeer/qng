// Copyright (c) 2017-2018 The qitmeer developers

package address

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/evm"
	"github.com/Qitmeer/qng/common/encode/base58"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/address"
	qjson "github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"sync"
)

type AddressApi struct {
	sync.Mutex
	params *params.Params
	config *config.Config
	chain  *blockchain.BlockChain
}

type PublicAddressAPI struct {
	addressApi *AddressApi
}

func NewAddressApi(cfg *config.Config, par *params.Params, chain *blockchain.BlockChain) *AddressApi {
	return &AddressApi{
		config: cfg,
		params: par,
		chain:  chain,
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
		return false, rpc.RpcInvalidError("Invalid address :" + err.Error())
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
	default:
		return false, rpc.RpcInvalidError("Invalid network : privnet | testnet | mainnet | mixnet")
	}
	if p.PubKeyHashAddrID != ver {
		return false, rpc.RpcRuleError("address prefix error , need %s , actual: %s,network not match,please check it",
			p.NetworkAddressPrefix, address[0:1])
	}
	return true, nil
}

func (api *PublicAddressAPI) GetBalance(pkAddress string, coinID types.CoinID) (interface{}, error) {
	if coinID != types.ETHID {
		return nil, fmt.Errorf("Not support %v", coinID)
	}
	cv, err := api.addressApi.chain.VMService.GetVM(evm.MeerEVMID)
	if err != nil {
		return nil, err
	}
	return cv.GetBalance(pkAddress)
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
	privkeyByte, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil,err
	}
	if len(privkeyByte) != 32 {
		return nil,fmt.Errorf("error length:%d", len(privkeyByte))
	}
	privateKey, pubKey := ecc.Secp256k1.PrivKeyFromBytes(privkeyByte)

	serializedKey := pubKey.SerializeCompressed()
	addr, err := address.NewSecpPubKeyAddress(serializedKey, params.ActiveNetParams.Params)
	if err != nil {
		return nil,err
	}
	eaddr,err:=common.NewMeerEVMAddress(hex.EncodeToString(pubKey.SerializeUncompressed()))
	if err != nil {
		return nil,err
	}
	result := qjson.OrderedResult{
		qjson.KV{Key:"PrivateKey",Val:hex.EncodeToString(privateKey.Serialize())},
		qjson.KV{Key:"PKHAddress",Val:addr.PKHAddress().String()},
		qjson.KV{Key:"PKAddress",Val:addr.String()},
		qjson.KV{Key:"MeerEVM Address",Val:eaddr.String()},
	}

	return result, nil
}