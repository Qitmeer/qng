// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package testutils

import (
	"context"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils/swap/factory"
	"github.com/Qitmeer/qng/testutils/swap/pair"
	"github.com/Qitmeer/qng/testutils/swap/router"
	"github.com/Qitmeer/qng/testutils/token"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

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
	spendAmt := types.Amount{Value: 14000 * types.AtomsPerCoin, Id: types.MEERA}
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
	toAmount := int64(2)
	toMeerAmount := big.NewInt(1e18).Mul(big.NewInt(1e18), big.NewInt(2))
	for i := 0; i < 2; i++ {
		_, _ = h.Wallet.NewAddress()
		to := h.Wallet.ethAddrs[uint32(i+1)]
		// send 2 meer
		txid, err := h.Wallet.CreateLegacyTx(h.Wallet.privkeys[0], &to, 0, 21000, toMeerAmount, nil)
		if err != nil {
			t.Fatal(err)
		}
		log.Println(i, "transfer meer:", txid)
		// send 2 token
		tx, err := tokenCall.Transfer(authCaller, h.Wallet.ethAddrs[uint32(i+1)], big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
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
	allAmount -= toAmount * 2
	assert.Equal(t, ba, big.NewInt(allAmount).Mul(big.NewInt(allAmount), big.NewInt(1e18)))
	for i := 1; i < 3; i++ {
		meerBa, err := h.EVMClient.BalanceAt(context.Background(), h.Wallet.ethAddrs[uint32(i)], nil)
		if err != nil {
			log.Fatal(err)
		}
		assert.Equal(t, meerBa, toMeerAmount)
		ba, err = tokenCall.BalanceOf(&bind.CallOpts{}, h.Wallet.ethAddrs[uint32(i)])
		if err != nil {
			t.Fatal(err)
		}
		log.Println(i, "address", h.Wallet.ethAddrs[uint32(i)].String(), "balance", ba)
		assert.Equal(t, ba, big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
		_, _ = h.Wallet.NewAddress()
		authCaller, err := h.Wallet.AuthTrans(h.Wallet.privkeys[uint32(i)])
		if err != nil {
			t.Fatal(err)
		}
		tx, err := tokenCall.Transfer(authCaller, h.Wallet.ethAddrs[uint32(i+2)], big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
		if err != nil {
			t.Fatal(err)
		}
		log.Println(i, "transfer tx:", tx.Hash().String())
	}
	GenerateBlock(t, h, 1)
	for i := 3; i < 5; i++ {
		ba, err = tokenCall.BalanceOf(&bind.CallOpts{}, h.Wallet.ethAddrs[uint32(i)])
		if err != nil {
			t.Fatal(err)
		}
		log.Println(i, "address", h.Wallet.ethAddrs[uint32(i)].String(), "balance", ba)
		assert.Equal(t, ba, big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
	}
	GenerateBlock(t, h, 1)
	// check transferFrom

	// not approve
	authCaller1, err := h.Wallet.AuthTrans(h.Wallet.privkeys[1])
	if err != nil {
		t.Fatal(err)
	}
	_, err = tokenCall.TransferFrom(authCaller1, h.Wallet.ethAddrs[0], h.Wallet.ethAddrs[1], big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
	if err == nil {
		t.Fatal("Token Bug,TransferFrom without approve")
	}
	log.Println(err)
	//  approve
	_, err = tokenCall.Approve(authCaller, h.Wallet.ethAddrs[1], big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
	if err != nil {
		t.Fatal("approve error", err)
	}
	GenerateBlock(t, h, 1)
	_, err = tokenCall.TransferFrom(authCaller1, h.Wallet.ethAddrs[0], h.Wallet.ethAddrs[1], big.NewInt(toAmount).Mul(big.NewInt(toAmount), big.NewInt(1e18)))
	if err != nil {
		t.Fatal("TransferFrom error", err)
	}
	GenerateBlock(t, h, 1)
}

func TestSwap(t *testing.T) {
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
	spendAmt := types.Amount{Value: 14000 * types.AtomsPerCoin, Id: types.MEERA}
	txid := SendSelf(t, h, spendAmt, nil, &lockTime)
	GenerateBlock(t, h, 10)
	fee := int64(2200)
	txStr, err := h.Wallet.CreateExportRawTx(txid.String(), spendAmt.Value, fee)
	if err != nil {
		t.Fatalf("createExportRawTx failed:%v", err)
	}
	log.Println("send tx", txStr)
	GenerateBlock(t, h, 2)
	ba, err := h.Wallet.GetBalance(h.Wallet.ethAddrs[0].String())
	if err != nil {
		t.Fatalf("GetBalance failed:%v", err)
	}
	assert.Equal(t, ba, big.NewInt(spendAmt.Value-fee))
	txS, err := h.Wallet.CreateErc20()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create token contract tx:", txS)
	txWETH, err := h.Wallet.CreateWETH()
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create weth contract tx:", txWETH)
	txFACTORY, err := h.Wallet.CreateFactory(h.Wallet.ethAddrs[0])
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create FACTORY contract tx:", txFACTORY)
	GenerateBlock(t, h, 2)
	// token addr
	txD, err := h.Wallet.evmClient.TransactionReceipt(context.Background(), common.HexToHash(txS))
	if err != nil {
		t.Fatal(err)
	}
	if txD == nil {
		t.Fatal("create contract failed")
	}
	txWETHD, err := h.Wallet.evmClient.TransactionReceipt(context.Background(), common.HexToHash(txWETH))
	if err != nil {
		t.Fatal(err)
	}
	if txWETHD == nil {
		t.Fatal("create weth failed")
	}
	txFACTORYD, err := h.Wallet.evmClient.TransactionReceipt(context.Background(), common.HexToHash(txFACTORY))
	if err != nil {
		t.Fatal(err)
	}
	if txFACTORYD == nil {
		t.Fatal("create factory failed")
	}
	assert.Equal(t, txD.Status, uint64(0x1))
	assert.Equal(t, txWETHD.Status, uint64(0x1))
	assert.Equal(t, txFACTORYD.Status, uint64(0x1))
	log.Println("new token address:", txD.ContractAddress)
	log.Println("new weth address:", txWETHD.ContractAddress)
	log.Println("new factory address:", txFACTORYD.ContractAddress)
	txROUTER, err := h.Wallet.CreateRouter(txFACTORYD.ContractAddress, txWETHD.ContractAddress)
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create ROUTER contract tx:", txROUTER)
	GenerateBlock(t, h, 1)
	txROUTERD, err := h.Wallet.evmClient.TransactionReceipt(context.Background(), common.HexToHash(txROUTER))
	if err != nil {
		t.Fatal(err)
	}
	if txROUTERD == nil {
		t.Fatal("create router failed")
	}
	assert.Equal(t, txROUTERD.Status, uint64(0x1))
	log.Println("new router address:", txROUTERD.ContractAddress)
	tokenCall, err := token.NewToken(txD.ContractAddress, h.Wallet.evmClient)
	if err != nil {
		t.Fatal(err)
	}
	factoryCall, err := factory.NewToken(txFACTORYD.ContractAddress, h.Wallet.evmClient)
	if err != nil {
		t.Fatal(err)
	}
	routerCall, err := router.NewToken(txROUTERD.ContractAddress, h.Wallet.evmClient)
	if err != nil {
		t.Fatal(err)
	}
	authCaller, err := h.Wallet.AuthTrans(h.Wallet.privkeys[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = tokenCall.Approve(authCaller, txROUTERD.ContractAddress, MAX_UINT256)
	if err != nil {
		t.Fatal("Approve error", err)
	}
	GenerateBlock(t, h, 1)
	amount := new(big.Int).SetUint64(1e18)
	amount = amount.Mul(amount, big.NewInt(1000))
	deadline := time.Now().Add(15 * time.Minute).Unix()
	authCaller.Value = amount
	// add liquidity
	_, err = routerCall.AddLiquidityETH(authCaller, txD.ContractAddress, amount, big.NewInt(0), big.NewInt(0), h.Wallet.ethAddrs[0], big.NewInt(deadline))
	if err != nil {
		t.Fatal("AddLiquidityETH error", err)
	}
	GenerateBlock(t, h, 1)
	// swap for a token
	_, _ = h.Wallet.NewAddress()
	to := h.Wallet.ethAddrs[1]
	// send 10 meer
	_, err = h.Wallet.CreateLegacyTx(h.Wallet.privkeys[0], &to, 0, 21000, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10)), nil)
	if err != nil {
		t.Fatal(err)
	}
	GenerateBlock(t, h, 1)
	// swap for a token  1 => 1 * 0.9975
	authCaller1, err := h.Wallet.AuthTrans(h.Wallet.privkeys[1])
	if err != nil {
		t.Fatal(err)
	}
	// 1 meer
	authCaller1.Value = big.NewInt(1e18)
	path := make([]common.Address, 0)
	path = append(path, txWETHD.ContractAddress, txD.ContractAddress)

	bas, err := routerCall.GetAmountsOut(&bind.CallOpts{}, big.NewInt(1e18), path)
	if err != nil {
		t.Fatal("GetAmountsOut error", err)
	}
	log.Println("expected balance", bas[0].String(), bas[1].String())
	deadline = time.Now().Add(15 * time.Minute).Unix()
	txSwap, err := routerCall.SwapExactETHForTokens(authCaller1, big.NewInt(0), path, to, big.NewInt(deadline))
	if err != nil {
		t.Fatal("SwapExactETHForTokens error", err)
	}
	GenerateBlock(t, h, 1)
	txSwapD, err := h.Wallet.evmClient.TransactionReceipt(context.Background(), txSwap.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if txSwapD == nil {
		t.Fatal("SwapExactETHForTokens Tx failed")
	}
	if txSwapD.Status != uint64(0x1) {
		txSwapD1, isPending, err := h.Wallet.evmClient.TransactionByHash(context.Background(), txSwap.Hash())
		if err != nil {
			t.Fatal(err)
		}
		t.Fatal("SwapExactETHForTokens Tx Error", "GasLimit", txSwapD1.Gas(), "Used Gas", txSwapD.GasUsed, "isPending", isPending)
	}

	ba1, err := tokenCall.BalanceOf(&bind.CallOpts{}, to)
	if err != nil {
		t.Fatal("BalanceOf Call error", err)
	}
	assert.Equal(t, ba1, bas[1])
	// remove liquidity
	// get pair address
	p, err := factoryCall.GetPair(&bind.CallOpts{}, txWETHD.ContractAddress, txD.ContractAddress)
	if err != nil {
		t.Fatal("GetPair Call error", err)
	}
	pairCall, err := pair.NewToken(p, h.Wallet.evmClient)
	if err != nil {
		t.Fatal(err)
	}
	lpBalance, err := pairCall.BalanceOf(&bind.CallOpts{}, h.Wallet.ethAddrs[0])
	if err != nil {
		t.Fatal(err)
	}
	authCaller, err = h.Wallet.AuthTrans(h.Wallet.privkeys[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = pairCall.Approve(authCaller, txROUTERD.ContractAddress, MAX_UINT256)
	if err != nil {
		t.Fatal(err)
	}
	GenerateBlock(t, h, 1)
	deadline = time.Now().Add(15 * time.Minute).Unix()
	txRemove, err := routerCall.RemoveLiquidityETH(authCaller, txD.ContractAddress, lpBalance, big.NewInt(0), big.NewInt(0), h.Wallet.ethAddrs[0], big.NewInt(deadline))
	if err != nil {
		t.Fatal("RemoveLiquidityETH error", err)
	}
	GenerateBlock(t, h, 1)
	txRemoveD, err := h.Wallet.evmClient.TransactionReceipt(context.Background(), txRemove.Hash())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, txRemoveD.Status, uint64(0x1))
}
