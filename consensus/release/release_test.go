// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package release

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestReleaseContract(t *testing.T) {
	h, err := testutils.StartMockNode(nil)
	if err != nil {
		t.Error(err)
	}
	defer h.Stop()

	if err != nil {
		t.Fatalf("setup harness failed:%v", err)
	}
	testutils.GenerateBlocks(t, h, 20)
	testutils.AssertBlockOrderHeightTotal(t, h, 21, 21, 20)

	lockTime := int64(20)
	spendAmt := types.Amount{Value: 14000 * types.AtomsPerCoin, Id: types.MEERA}
	txid := testutils.SendSelfMockNode(t, h, spendAmt, &lockTime)
	testutils.GenerateBlocks(t, h, 10)
	fee := int64(2200)
	txid = testutils.SendExportTxMockNode(t, h, txid.String(), 0, spendAmt.Value-fee)
	if err != nil {
		t.Fatalf("createExportRawTx failed:%v", err)
	}
	log.Println("send tx", txid.String())
	testutils.GenerateBlocks(t, h, 10)
	evmAddr := h.GetWalletManager().GetAccountByIdx(0).EvmAcct.Address
	ba, err := h.GetEvmClient().BalanceAt(context.Background(), evmAddr, nil)
	if err != nil {
		t.Fatalf("GetBalance failed:%v", err)
	}
	assert.Equal(t, ba, new(big.Int).Mul(big.NewInt(1e10), big.NewInt(spendAmt.Value-fee)))
	contract := common.HexToAddress("0x1000000000000000000000000000000000000000")

	tokenCall, err := NewToken(contract, h.GetEvmClient())
	if err != nil {
		t.Fatal(err)
	}
	// 0000000000000000000000000000000000000000000000000000000000000000
	testutils.GenerateBlocks(t, h, 1)
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

	b4, err := h.GetEvmClient().BalanceAt(context.Background(), common.HexToAddress(RELEASE_ADDR), nil)
	if err != nil {
		log.Fatalln(err)
	}
	assert.Equal(t, b4.String(), RELEASE_AMOUNT)
}
