// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testutils

import (
	"context"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils/release"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"log"
	"math/big"
	"testing"
	"time"
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

	ba, err = h.Wallet.evmClient.BalanceAt(context.Background(), contract, nil)
	if err != nil {
		t.Fatal(err)
	}
	authCaller, err := h.Wallet.AuthTrans(h.Wallet.privkeys[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = tokenCall.SetStartTime(authCaller, big.NewInt(time.Now().Unix()-86400))
	if err != nil {
		t.Fatal(err)
	}
	GenerateBlock(t, h, 1)
	_, err = tokenCall.SetEndTime(authCaller, big.NewInt(time.Now().Unix()+86400))
	if err != nil {
		t.Fatal(err)
	}
	GenerateBlock(t, h, 1)
	_, err = tokenCall.Lock(authCaller, h.Wallet.ethAddrs[0], big.NewInt(1e18))
	if err != nil {
		t.Fatal(err)
	}
	GenerateBlock(t, h, 1)
	cr, err := tokenCall.CanRelease(&bind.CallOpts{}, h.Wallet.ethAddrs[0])
	if err != nil {
		t.Fatal(err)
	}
	log.Println(cr.String())
	_, err = tokenCall.Claim(authCaller, h.Wallet.ethAddrs[0])
	if err != nil {
		t.Fatal(err)
	}
	GenerateBlock(t, h, 1)
	cr, err = tokenCall.CanRelease(&bind.CallOpts{}, h.Wallet.ethAddrs[0])
	if err != nil {
		t.Fatal(err)
	}
	log.Println(cr.String())
}
