/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package common

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"io"
)

type Key struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

func NewKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *Key {
	key := &Key{
		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}
	return key
}

func NewKey(rand io.Reader) (*Key, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}
	return NewKeyFromECDSA(privateKeyECDSA), nil
}