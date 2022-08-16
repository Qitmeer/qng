// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testutils

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils/release"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"log"
	"math/big"
	"testing"
)

func TestReleaseContract(t *testing.T) {
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
	contract := common.HexToAddress("0x1000000000000000000000000000000000000000")
	tokenCall, err := release.NewToken(contract, h.Wallet.evmClient)
	if err != nil {
		t.Fatal(err)
	}
	// 0000000000000000000000000000000000000000000000000000000000000000
	GenerateBlock(t, h, 1)
	maddr := "Mmf93CE9Cvvf3chYYn1okcBFB22u5wH2dyg"
	addr, _ := address.DecodeAddress(maddr)
	hash160 := hex.EncodeToString(addr.Hash160()[:])
	fmt.Println("hash160", hash160)

	b0, _ := hex.DecodeString(hash160)
	b, err := tokenCall.QueryAmount(&bind.CallOpts{}, b0)
	if err != nil {
		log.Fatalln(err)
	}
	b3, err := tokenCall.MeerMappingCount(&bind.CallOpts{}, b0)
	if err != nil {
		log.Fatalln(err)
	}
	assert.Equal(t, b3.String(), "1")
	b1, err := tokenCall.QueryBurnDetails(&bind.CallOpts{}, b0)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(b1)
	assert.Equal(t, b.String(), "100000000000")

	b4, err := h.EVMClient.BalanceAt(context.Background(), common.HexToAddress(RELEASE_ADDR), nil)
	if err != nil {
		log.Fatalln(err)
	}
	assert.Equal(t, b4.String(), RELEASE_AMOUNT)
}
