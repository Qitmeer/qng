// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testutils

import (
	"encoding/hex"
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
	// hash160
	hash160 := "bef272d0be043949ce88ec01602cb1d3bce9aade"
	GenerateBlock(t, h, 1)
	b0, _ := hex.DecodeString(hash160)
	GenerateBlock(t, h, 1)
	a, _ := tokenCall.QueryAmount(&bind.CallOpts{}, b0)
	assert.Equal(t, a.String(), 1659715200)
}
