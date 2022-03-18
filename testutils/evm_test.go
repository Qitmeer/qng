// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testutils

import (
	"context"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils/token"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"log"
	"math/big"
	"testing"
)

func TestEVM(t *testing.T) {
	args := []string{"--modules=miner", "--modules=qitmeer",
		"--modules=test"}
	h, err := NewHarness(t, params.PrivNetParam.Params, args...)
	defer h.Teardown()
	if err != nil {
		t.Errorf("new harness failed: %v", err)
		h.Teardown()
	}
	err = h.Setup()
	if err != nil {
		t.Fatalf("setup harness failed:%v", err)
	}
	GenerateBlock(t, h, 20)
	AssertBlockOrderAndHeight(t, h, 21, 21, 20)

	lockTime := int64(20)
	spendAmt := types.Amount{Value: 14000 * types.AtomsPerCoin, Id: types.MEERID}
	txid := SendSelf(t, h, spendAmt, nil, &lockTime)
	GenerateBlock(t, h, 10)
	fee := int64(2200)
	txStr, err := h.Wallet.CreateExportRawTx(txid.String(), spendAmt.Value, fee)
	if err != nil {
		t.Fatalf("createExportRawTx failed:%v", err)
	}
	log.Println("send tx", txStr)
	GenerateBlock(t, h, 10)
	ba, err := h.Wallet.GetBalance(h.Wallet.ethAddrs[0].String())
	if err != nil {
		t.Fatalf("GetBalance failed:%v", err)
	}
	assert.Equal(t, ba, big.NewInt(spendAmt.Value-fee))
}

func TestCreateContract(t *testing.T) {
	args := []string{"--modules=miner", "--modules=qitmeer",
		"--modules=test"}
	h, err := NewHarness(t, params.PrivNetParam.Params, args...)
	defer h.Teardown()
	if err != nil {
		t.Errorf("new harness failed: %v", err)
		h.Teardown()
	}
	err = h.Setup()
	if err != nil {
		t.Fatalf("setup harness failed:%v", err)
	}
	GenerateBlock(t, h, 20)
	AssertBlockOrderAndHeight(t, h, 21, 21, 20)

	lockTime := int64(20)
	spendAmt := types.Amount{Value: 14000 * types.AtomsPerCoin, Id: types.MEERID}
	txid := SendSelf(t, h, spendAmt, nil, &lockTime)
	GenerateBlock(t, h, 10)
	fee := int64(2200)
	txStr, err := h.Wallet.CreateExportRawTx(txid.String(), spendAmt.Value, fee)
	if err != nil {
		t.Fatalf("createExportRawTx failed:%v", err)
	}
	log.Println("send tx", txStr)
	GenerateBlock(t, h, 10)
	ba, err := h.Wallet.GetBalance(h.Wallet.ethAddrs[0].String())
	if err != nil {
		t.Fatalf("GetBalance failed:%v", err)
	}
	assert.Equal(t, ba, big.NewInt(spendAmt.Value-fee))
	txS, err := h.Wallet.CreateErc20()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create contract tx:", txS)
	GenerateBlock(t, h, 2)
	// token addr
	txD, err := h.Wallet.evmClient.TransactionReceipt(context.Background(), common.HexToHash(txS))
	if err != nil {
		t.Fatal(err)
	}
	if txD == nil {
		t.Fatal("create contract failed")
	}
	assert.Equal(t, txD.Status, uint64(0x1))
	log.Println("new contract address:", txD.ContractAddress)
	tokenCall, err := token.NewToken(txD.ContractAddress, h.Wallet.evmClient)
	if err != nil {
		t.Fatal(err)
	}
	ba, err = tokenCall.BalanceOf(&bind.CallOpts{}, h.Wallet.ethAddrs[0])
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, ba, big.NewInt(30000000).Mul(big.NewInt(30000000), big.NewInt(1e18)))
}

func TestCallErc20Contract(t *testing.T) {
	args := []string{"--modules=miner", "--modules=qitmeer",
		"--modules=test"}
	h, err := NewHarness(t, params.PrivNetParam.Params, args...)
	defer h.Teardown()
	if err != nil {
		t.Errorf("new harness failed: %v", err)
		h.Teardown()
	}
	err = h.Setup()
	if err != nil {
		t.Fatalf("setup harness failed:%v", err)
	}
	GenerateBlock(t, h, 20)
	AssertBlockOrderAndHeight(t, h, 21, 21, 20)

	lockTime := int64(20)
	spendAmt := types.Amount{Value: 14000 * types.AtomsPerCoin, Id: types.MEERID}
	txid := SendSelf(t, h, spendAmt, nil, &lockTime)
	GenerateBlock(t, h, 10)
	fee := int64(2200)
	txStr, err := h.Wallet.CreateExportRawTx(txid.String(), spendAmt.Value, fee)
	if err != nil {
		t.Fatalf("createExportRawTx failed:%v", err)
	}
	log.Println("send tx", txStr)
	GenerateBlock(t, h, 10)
	ba, err := h.Wallet.GetBalance(h.Wallet.ethAddrs[0].String())
	if err != nil {
		t.Fatalf("GetBalance failed:%v", err)
	}
	assert.Equal(t, ba, big.NewInt(spendAmt.Value-fee))
	txS, err := h.Wallet.CreateErc20()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create contract tx:", txS)
	GenerateBlock(t, h, 2)
	// token addr
	txD, err := h.Wallet.evmClient.TransactionReceipt(context.Background(), common.HexToHash(txS))
	if err != nil {
		t.Fatal(err)
	}
	if txD == nil {
		t.Fatal("create contract failed")
	}
	assert.Equal(t, txD.Status, uint64(0x1))
	log.Println("new contract address:", txD.ContractAddress)
	tokenCall, err := token.NewToken(txD.ContractAddress, h.Wallet.evmClient)
	if err != nil {
		t.Fatal(err)
	}
	ba, err = tokenCall.BalanceOf(&bind.CallOpts{}, h.Wallet.ethAddrs[0])
	if err != nil {
		t.Fatal(err)
	}
	allAmount := int64(30000000)
	assert.Equal(t, ba, big.NewInt(allAmount).Mul(big.NewInt(allAmount), big.NewInt(1e18)))
	authCaller, err := h.Wallet.AuthTrans(h.Wallet.privkeys[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.Wallet.NewAddress()
	if err != nil {
		t.Fatal(err)
	}
	toAmount := int64(100)
	for i := 0; i < 10; i++ {
		tx, err := tokenCall.Transfer(authCaller, h.Wallet.ethAddrs[1], big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
		if err != nil {
			t.Fatal(err)
		}
		log.Println(i, "transfer tx:", tx.Hash().String())
	}

	GenerateBlock(t, h, 1)
	ba, err = tokenCall.BalanceOf(&bind.CallOpts{}, h.Wallet.ethAddrs[0])
	if err != nil {
		t.Fatal(err)
	}
	toAmount *= 10 // 转10次
	allAmount -= toAmount
	assert.Equal(t, ba, big.NewInt(allAmount).Mul(big.NewInt(allAmount), big.NewInt(1e18)))
	ba, err = tokenCall.BalanceOf(&bind.CallOpts{}, h.Wallet.ethAddrs[1])
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, ba, big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
}