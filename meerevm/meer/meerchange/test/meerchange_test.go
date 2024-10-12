// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package test

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/config"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/meerevm/meer/meerchange"
	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"testing"
)

func TestGetMeerChangeAddress(t *testing.T) {
	node, err := testutils.StartMockNode(func(cfg *config.Config) error {
		cfg.GenerateOnTx = true
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()

	testutils.ShowMeTheMoneyForMeer(t, node, 0)

	acc := node.GetWalletManager().GetAccountByIdx(0)
	err = node.DeterministicDeploymentProxy().Deploy(acc.EvmAcct.Address)
	if err != nil {
		t.Fatal(err)
	}

	addr, err := node.DeterministicDeploymentProxy().GetContractAddress(common.FromHex(meerchange.MeerchangeMetaData.Bin), meerchange.Version)
	if err != nil {
		t.Fatal(err)
	}
	expectAddr := "0x7a9a241E8AD3D9804d4545a86DeDb93b397C9899"
	if addr.String() != expectAddr {
		t.Fatalf("Current:%s, but expect:%s", addr.String(), expectAddr)
	}
}

func TestDeployMeerChange(t *testing.T) {
	node, err := testutils.StartMockNode(func(cfg *config.Config) error {
		cfg.GenerateOnTx = true
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()

	testutils.ShowMeTheMoneyForMeer(t, node, 0)

	acc := node.GetWalletManager().GetAccountByIdx(0)

	txh, err := node.GetPublicMeerChainAPI().DeployMeerChange(acc.EvmAcct.Address)
	if err != nil {
		t.Fatal(err)
	}
	testutils.GenerateBlocksWaitForTxs(t, node, []string{txh.(string)})
	addr, err := node.GetPublicMeerChainAPI().GetMeerChangeAddr()
	if err != nil {
		t.Fatal(err)
	}
	if addr.(string) != meerchange.ContractAddr.String() {
		t.Fatalf("Current:%v, but expect:%s", addr, meerchange.ContractAddr.String())
	}
}

func TestMeerChangeExport(t *testing.T) {
	node, err := testutils.StartMockNode(func(cfg *config.Config) error {
		cfg.GenerateOnTx = true
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()

	testutils.ShowMeTheMoneyForMeer(t, node, 0)

	acc := node.GetWalletManager().GetAccountByIdx(0)

	txh, err := node.GetPublicMeerChainAPI().DeployMeerChange(acc.EvmAcct.Address)
	if err != nil {
		t.Fatal(err)
	}
	testutils.GenerateBlocksWaitForTxs(t, node, []string{txh.(string)})
	// some outpoints
	ops := "288dede439a4e14df002a2a7af36cdb8ea451500d6a4f4f5d821002b28123a59:0,288dede439a4e14df002a2a7af36cdb8ea451500d6a4f4f5d821002b28123a59:135"
	fee := uint64(0)

	instance, err := meerchange.NewMeerchange(meerchange.ContractAddr, node.GetEvmClient())
	if err != nil {
		t.Fatal(err)
	}
	mci, err := node.GetPublicMeerChainAPI().GetMeerChainInfo()
	if err != nil {
		t.Fatal(err)
	}

	mcInfo := mci.(meer.MeerChainInfo)
	privateKey, err := crypto.HexToECDSA(node.GetBuilder().GetHex(0))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(int64(mcInfo.ChainID)))
	if err != nil {
		t.Fatal(err)
	}
	sig, err := meerchange.CalcExportSig(meerchange.CalcExportHash(ops, fee), node.GetBuilder().GetHex(0))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := instance.Export(auth, ops, fee, hex.EncodeToString(sig))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Waiting export tx:%s", tx.Hash().String())
	testutils.GenerateBlocksWaitForTxs(t, node, []string{tx.Hash().String()})

	ba, err := node.GetEvmClient().BalanceAt(node.Node().Context(), acc.EvmAcct.Address, nil)
	if err != nil {
		t.Fatal(err)
	}
	expect := big.NewInt(5000000000000)
	expect = expect.Mul(expect, qcommon.Precision)
	if ba.Uint64() < expect.Uint64() {
		t.Fatalf("current:%s expect:%s", ba.String(), expect.String())
	}
}
