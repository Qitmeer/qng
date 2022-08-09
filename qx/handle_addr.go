package qx

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/encode/base58"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/params"
)

func EcPubKeyToAddress(version string, pubkey string) (string, error) {
	ver := []byte{}

	switch version {
	case "mainnet":
		ver = append(ver, params.MainNetParams.PubKeyHashAddrID[0:]...)
	case "privnet":
		ver = append(ver, params.PrivNetParams.PubKeyHashAddrID[0:]...)
	case "testnet":
		ver = append(ver, params.TestNetParams.PubKeyHashAddrID[0:]...)
	case "mixnet":
		ver = append(ver, params.MixNetParam.PubKeyHashAddrID[0:]...)
	default:
		v, err := hex.DecodeString(version)
		if err != nil {
			return "", err
		}
		ver = append(ver, v...)
	}

	data, err := hex.DecodeString(pubkey)
	if err != nil {
		return "", err
	}
	h := hash.Hash160(data)
	fmt.Println("hash160:", hex.EncodeToString(h))
	address, err := base58.QitmeerCheckEncode(h, ver[:])
	if err != nil {
		return "", err
	}
	return string(address), nil
}

func EcScriptKeyToAddress(version string, pubkey string) (string, error) {
	ver := []byte{}

	switch version {
	case "mainnet":
		ver = append(ver, params.MainNetParams.ScriptHashAddrID[0:]...)
	case "privnet":
		ver = append(ver, params.PrivNetParams.ScriptHashAddrID[0:]...)
	case "testnet":
		ver = append(ver, params.TestNetParams.ScriptHashAddrID[0:]...)
	case "mixnet":
		ver = append(ver, params.MixNetParam.ScriptHashAddrID[0:]...)
	default:
		v, err := hex.DecodeString(version)
		if err != nil {
			return "", err
		}
		ver = append(ver, v...)
	}

	data, err := hex.DecodeString(pubkey)
	if err != nil {
		return "", err
	}
	h := hash.Hash160(data)

	address, err := base58.QitmeerCheckEncode(h, ver[:])
	if err != nil {
		return "", err
	}
	return string(address), nil
}

func EcPubKeyToAddressSTDO(version []byte, pubkey string) {
	data, err := hex.DecodeString(pubkey)
	if err != nil {
		ErrExit(err)
	}
	h := hash.Hash160(data)

	address, _ := base58.QitmeerCheckEncode(h, version[:])
	fmt.Printf("%s\n", address)
}

func EcPubKeyToPKAddressSTDO(version string, pubkey string) {
	data, err := hex.DecodeString(pubkey)
	if err != nil {
		ErrExit(err)
	}

	pubKey, err := ecc.Secp256k1.ParsePubKey(data)
	if err != nil {
		ErrExit(err)
	}
	var param *params.Params
	switch version {
	case "mainnet":
		param = &params.MainNetParams
	case "privnet":
		param = &params.PrivNetParams
	case "testnet":
		param = &params.TestNetParams
	case "mixnet":
		param = &params.MixNetParams
	default:
		param = &params.MainNetParams
	}

	addr, err := address.NewSecpPubKeyAddress(pubKey.SerializeCompressed(), param)
	if err != nil {
		ErrExit(err)
	}

	fmt.Printf("%s\n", addr.String())
}

func EcPubKeyToETHAddressSTDO(pubkey string) {
	addr, err := common.NewMeerEVMAddress(pubkey)

	if err != nil {
		ErrExit(err)
	}

	fmt.Printf("%s\n", addr.String())
}

func PKAddressToPubKey(pkaddr string, compressed bool) (string, error) {
	ePKAddr, err := address.DecodeAddress(pkaddr)
	if err != nil {
		return "", err
	}
	pka, ok := ePKAddr.(*address.SecpPubKeyAddress)
	if !ok {
		return "", fmt.Errorf("%s is not public key address", pkaddr)
	}
	var key []byte
	if compressed {
		key = pka.PubKey().SerializeCompressed()
	} else {
		key = pka.PubKey().SerializeUncompressed()
	}
	return fmt.Sprintf("%x", key[:]), nil
}

func PKAddressToETHAddressSTDO(pkaddr string) {
	ePKAddr, err := address.DecodeAddress(pkaddr)
	if err != nil {
		ErrExit(err)
	}
	pka, ok := ePKAddr.(*address.SecpPubKeyAddress)
	if !ok {
		ErrExit(fmt.Errorf("%s is not public key address", pkaddr))
	}
	EcPubKeyToETHAddressSTDO(hex.EncodeToString(pka.PubKey().SerializeUncompressed()))
}
