// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package test

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/meerevm/meer/meerchange"
	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum/common"
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

	_, err = node.GetPublicMeerChainAPI().DeployMeerChange(acc.EvmAcct.Address)
	if err != nil {
		t.Fatal(err)
	}
	addr, err := node.GetPublicMeerChainAPI().GetMeerChangeAddr()
	if err != nil {
		t.Fatal(err)
	}
	expectAddr := "0x7a9a241E8AD3D9804d4545a86DeDb93b397C9899"
	if addr.(string) != expectAddr {
		t.Fatalf("Current:%v, but expect:%s", addr, expectAddr)
	}
}
