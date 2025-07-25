package common

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/v2/core/address"
	"github.com/Qitmeer/qng/v2/crypto/ecc"
	"github.com/Qitmeer/qng/v2/meerevm/common"
	"github.com/Qitmeer/qng/v2/params"
	ecommon "github.com/ethereum/go-ethereum/common"
)

func NewAddresses(privateKeyHex string) (ecc.PrivateKey, *address.SecpPubKeyAddress, ecommon.Address, error) {
	privkeyByte, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, nil, ecommon.Address{}, err
	}
	if len(privkeyByte) != 32 {
		return nil, nil, ecommon.Address{}, fmt.Errorf("error length:%d", len(privkeyByte))
	}
	privateKey, pubKey := ecc.Secp256k1.PrivKeyFromBytes(privkeyByte)
	serializedKey := pubKey.SerializeCompressed()
	addr, err := address.NewSecpPubKeyAddress(serializedKey, params.ActiveNetParams.Params)
	if err != nil {
		return nil, nil, ecommon.Address{}, err
	}
	eaddr, err := common.NewMeerEVMAddress(pubKey.SerializeUncompressed())
	if err != nil {
		return nil, nil, ecommon.Address{}, err
	}
	return privateKey, addr, eaddr, nil
}
