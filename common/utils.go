/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package common

import (
	"encoding/hex"
	"github.com/Qitmeer/qng-core/crypto/ecc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	qtypes "github.com/Qitmeer/qng-core/core/types"
)

func ReverseBytes(bs *[]byte) {
	length := len(*bs)
	for i := 0; i < length/2; i++ {
		index:=length-1-i
		temp := (*bs)[index]
		(*bs)[index] = (*bs)[i]
		(*bs)[i] = temp
	}
}

func NewMeerEVMAddress(pubkeyHex string) (common.Address,error) {
	pubkBytes,err:=hex.DecodeString(pubkeyHex)
	if err != nil {
		return common.Address{},err
	}

	publicKey, err := ecc.Secp256k1.ParsePubKey(pubkBytes)
	if err != nil {
		return common.Address{},err
	}
	return crypto.PubkeyToAddress(*publicKey.ToECDSA()),nil
}

var (
	Precision = big.NewInt(params.Ether).Mul(big.NewInt(params.Ether),big.NewInt(qtypes.AtomsPerCoin))
)