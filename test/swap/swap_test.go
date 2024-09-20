// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package swap

import (
	"context"
	"log"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/test/swap/factory"
	"github.com/Qitmeer/qng/test/swap/pair"
	"github.com/Qitmeer/qng/test/swap/router"
	"github.com/Qitmeer/qng/test/testcommon"
	"github.com/Qitmeer/qng/test/token"
	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestSwap(t *testing.T) {
	h, err := testutils.StartMockNode(nil)
	if err != nil {
		t.Error(err)
	}
	defer h.Stop()

	if err != nil {
		t.Fatalf("setup harness failed:%v", err)
	}
	testutils.GenerateBlocks(t, h, 1)
	testutils.AssertBlockOrderHeightTotal(t, h, 2, 2, 1)

	lockTime := int64(20)
	spendAmt := types.Amount{Value: 14000 * types.AtomsPerCoin, Id: types.MEERA}
	txid := testutils.SendSelfMockNode(t, h, spendAmt, &lockTime)
	testutils.GenerateBlocks(t, h, 1)
	fee := int64(2200)
	txH := testutils.SendExportTxMockNode(t, h, txid.String(), 0, spendAmt.Value-fee)
	if err != nil {
		t.Fatalf("createExportRawTx failed:%v", err)
	}
	log.Println("send tx", txH.String())
	testutils.GenerateBlocks(t, h, 1)
	acc := h.GetWalletManager().GetAccountByIdx(0)
	ba, err := h.GetEvmClient().BalanceAt(context.Background(), acc.EvmAcct.Address, nil)
	if err != nil {
		t.Fatalf("GetBalance failed:%v", err)
	}
	assert.Equal(t, ba, new(big.Int).Mul(big.NewInt(1e10), big.NewInt(spendAmt.Value-fee)))
	txS, err := testutils.DeployContract(h, 0, common.FromHex(testcommon.ERC20Code))
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create token contract tx:", txS)
	txWETH, err := CreateWETH(h)
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create weth contract tx:", txWETH)
	txFACTORY, err := CreateFactory(h, acc.EvmAcct.Address)
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create FACTORY contract tx:", txFACTORY)
	testutils.GenerateBlocksWaitForTxs(t, h, []string{txS, txWETH, txFACTORY})
	// token addr
	txD, err := h.GetEvmClient().TransactionReceipt(context.Background(), common.HexToHash(txS))
	if err != nil {
		t.Fatal(err)
	}
	if txD == nil {
		t.Fatal("create contract failed")
	}
	txWETHD, err := h.GetEvmClient().TransactionReceipt(context.Background(), common.HexToHash(txWETH))
	if err != nil {
		t.Fatal(err)
	}
	if txWETHD == nil {
		t.Fatal("create weth failed")
	}
	txFACTORYD, err := h.GetEvmClient().TransactionReceipt(context.Background(), common.HexToHash(txFACTORY))
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
	txROUTER, err := CreateRouter(h, txFACTORYD.ContractAddress, txWETHD.ContractAddress)
	if err != nil {
		t.Fatal(err)
	}
	log.Println("create ROUTER contract tx:", txROUTER)
	testutils.GenerateBlocksWaitForTxs(t, h, []string{txROUTER})
	txROUTERD, err := h.GetEvmClient().TransactionReceipt(context.Background(), common.HexToHash(txROUTER))
	if err != nil {
		t.Fatal(err)
	}
	if txROUTERD == nil {
		t.Fatal("create router failed")
	}
	assert.Equal(t, txROUTERD.Status, uint64(0x1))
	log.Println("new router address:", txROUTERD.ContractAddress)
	tokenCall, err := token.NewToken(txD.ContractAddress, h.GetEvmClient())
	if err != nil {
		t.Fatal(err)
	}
	factoryCall, err := factory.NewToken(txFACTORYD.ContractAddress, h.GetEvmClient())
	if err != nil {
		t.Fatal(err)
	}
	routerCall, err := router.NewToken(txROUTERD.ContractAddress, h.GetEvmClient())
	if err != nil {
		t.Fatal(err)
	}
	authCaller, err := testutils.AuthTrans(h.GetBuilder().Get(0))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := tokenCall.Approve(authCaller, txROUTERD.ContractAddress, testutils.MAX_UINT256)
	if err != nil {
		t.Fatal("Approve error", err)
	}
	testutils.GenerateBlocksWaitForTxs(t, h, []string{tx.Hash().String()})
	amount := new(big.Int).SetUint64(1e18)
	amount = amount.Mul(amount, big.NewInt(1000))
	deadline := time.Now().Add(15 * time.Minute).Unix()
	authCaller.Value = amount
	// add liquidity
	tx, err = routerCall.AddLiquidityETH(authCaller, txD.ContractAddress, amount, big.NewInt(0), big.NewInt(0), acc.EvmAcct.Address, big.NewInt(deadline))
	if err != nil {
		t.Fatal("AddLiquidityETH error", err)
	}
	testutils.GenerateBlocksWaitForTxs(t, h, []string{tx.Hash().String()})
	// swap for a token
	h.NewAddress()
	to := h.GetWalletManager().GetAccountByIdx(1).EvmAcct.Address
	// send 10 meer
	txh, err := testutils.MeerTransfer(h, 0, to, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10)))
	if err != nil {
		t.Fatal(err)
	}
	testutils.GenerateBlocksWaitForTxs(t, h, []string{txh})
	// swap for a token  1 => 1 * 0.9975
	authCaller1, err := testutils.AuthTrans(h.GetBuilder().Get(1))
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
	testutils.GenerateBlocksWaitForTxs(t, h, []string{txSwap.Hash().String()})
	txSwapD, err := h.GetEvmClient().TransactionReceipt(context.Background(), txSwap.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if txSwapD == nil {
		t.Fatal("SwapExactETHForTokens Tx failed")
	}
	if txSwapD.Status != uint64(0x1) {
		txSwapD1, isPending, err := h.GetEvmClient().TransactionByHash(context.Background(), txSwap.Hash())
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
	pairCall, err := pair.NewToken(p, h.GetEvmClient())
	if err != nil {
		t.Fatal(err)
	}
	lpBalance, err := pairCall.BalanceOf(&bind.CallOpts{}, acc.EvmAcct.Address)
	if err != nil {
		t.Fatal(err)
	}
	authCaller, err = testutils.AuthTrans(h.GetBuilder().Get(0))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pairCall.Approve(authCaller, txROUTERD.ContractAddress, testutils.MAX_UINT256)
	if err != nil {
		t.Fatal(err)
	}
	testutils.GenerateBlocks(t, h, 1)
	deadline = time.Now().Add(15 * time.Minute).Unix()
	txRemove, err := routerCall.RemoveLiquidityETH(authCaller, txD.ContractAddress, lpBalance, big.NewInt(0), big.NewInt(0), acc.EvmAcct.Address, big.NewInt(deadline))
	if err != nil {
		t.Fatal("RemoveLiquidityETH error", err)
	}
	testutils.GenerateBlocksWaitForTxs(t, h, []string{txRemove.Hash().String()})
	txRemoveD, err := h.GetEvmClient().TransactionReceipt(context.Background(), txRemove.Hash())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, txRemoveD.Status, uint64(0x1))
}
func CreateWETH(node *testutils.MockNode) (string, error) {
	return testutils.DeployContract(node, 0, common.FromHex(testcommon.WETH))
}

func CreateFactory(node *testutils.MockNode, _feeToSetter common.Address) (string, error) {
	parsed, _ := abi.JSON(strings.NewReader(factory.TokenMetaData.ABI))
	// constructor params
	initP, _ := parsed.Pack("", _feeToSetter)
	return testutils.DeployContract(node, 0, append(common.FromHex(testcommon.FACTORY), initP...))
}

func CreateRouter(node *testutils.MockNode, factory, weth common.Address) (string, error) {
	parsed, _ := abi.JSON(strings.NewReader(router.TokenMetaData.ABI))
	initP, _ := parsed.Pack("", factory, weth)
	return testutils.DeployContract(node, 0, append(common.FromHex(testcommon.ROUTER), initP...))
}
