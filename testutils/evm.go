// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testutils

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils/swap/factory"
	"github.com/Qitmeer/qng/testutils/swap/router"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math/big"
	"strings"
)

var CHAIN_ID = params.PrivNetParams.MeerEVMCfg.ChainID

const GAS_LIMIT = 8000000

const RELEASE_ADDR = "0xB191d00579ba344565637468e0CCbD6f161C0333"

const RELEASE_AMOUNT = "1641215940636640000000000"

var MAX_UINT256 = new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 255), common.Big1)

func (w *testWallet) CreateExportRawTx(txid string, amount, fee int64) (string, error) {
	amount -= fee
	if txid == "" {
		return txid, errors.New("CreateExportRawTx Error,Amount Not Enough")
	}
	s, err := w.client.CreateExportRawTx(txid, w.pkAddrs[0].String(),
		0, amount)
	if err != nil {
		return "", err
	}
	s, err = w.client.TestSign(hex.EncodeToString(w.privkeys[0]), s, "")
	if err != nil {
		return "", err
	}
	tx, err := w.client.SendRawTx(s, true)
	if err != nil {
		return "", err
	}
	return tx.String(), nil
}

func (w *testWallet) GetBalance(addr string) (*big.Int, error) {
	hb, err := w.evmClient.BalanceAt(context.Background(), common.HexToAddress(addr), nil)
	if err != nil {
		return nil, err
	}
	return ConvertEthToMeer(hb), nil
}

func (w *testWallet) CreateErc20() (string, error) {
	return w.CreateLegacyTx(w.privkeys[0], nil, 0, 0, big.NewInt(0), common.FromHex(ERC20Code))
}

func (w *testWallet) CreateRelease() (string, error) {
	// constructor params
	return w.CreateLegacyTx(w.privkeys[0], nil, 0, 0, big.NewInt(0), common.FromHex(RELEASECode))
}

func (w *testWallet) CreateWETH() (string, error) {
	return w.CreateLegacyTx(w.privkeys[0], nil, 0, 0, big.NewInt(0), common.FromHex(WETH))
}

func (w *testWallet) CreateFactory(_feeToSetter common.Address) (string, error) {
	parsed, _ := abi.JSON(strings.NewReader(factory.TokenMetaData.ABI))
	// constructor params
	initP, _ := parsed.Pack("", _feeToSetter)
	return w.CreateLegacyTx(w.privkeys[0], nil, 0, 0, big.NewInt(0), append(common.FromHex(FACTORY), initP...))
}

func (w *testWallet) CreateRouter(factory, weth common.Address) (string, error) {
	parsed, _ := abi.JSON(strings.NewReader(router.TokenMetaData.ABI))
	initP, _ := parsed.Pack("", factory, weth)
	return w.CreateLegacyTx(w.privkeys[0], nil, 0, 0, big.NewInt(0), append(common.FromHex(ROUTER), initP...))
}

func (w *testWallet) AuthTrans(privatekeybyte []byte) (*bind.TransactOpts, error) {
	privateKey := crypto.ToECDSAUnsafe(privatekeybyte)
	return bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(CHAIN_ID))
}

func (w *testWallet) CreateLegacyTx(fromPkByte []byte, to *common.Address, nonce uint64, gas uint64, val *big.Int, d []byte) (string, error) {
	privateKey := crypto.ToECDSAUnsafe(fromPkByte)
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("private key error")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	log.Println("from address", fromAddress.String())
	var err error
	if nonce <= 0 {
		nonce, err = w.evmClient.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			return "", err
		}
	}
	gasLimit := uint64(GAS_LIMIT) // in units
	if gas > 0 {
		gasLimit = gas
	}
	gasPrice, err := w.evmClient.SuggestGasPrice(context.Background())
	if err != nil {
		return "", err
	}
	data := &types.LegacyTx{
		To:       to,
		Nonce:    nonce,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Value:    val,
		Data:     d,
	}
	tx := types.NewTx(data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(CHAIN_ID)), privateKey)
	if err != nil {
		return "", err
	}
	err = w.evmClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", err
	}
	return signedTx.Hash().Hex(), nil
}
