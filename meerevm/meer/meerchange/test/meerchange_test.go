// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package test

import (
	"github.com/Qitmeer/qng/meerevm/meer/meerchange"
	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestGetMeerChangeAddress(t *testing.T) {
	node, err := testutils.StartMockNode(nil)
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()

	if err != nil {
		t.Fatalf("setup harness failed:%v", err)
	}
	acc := node.GetWalletManager().GetAccountByIdx(0)
	if err != nil {
		t.Fatalf("GetAcctInfo failed:%v", err)
	}
	addr, err := node.DeterministicDeploymentProxy().GetContractAddress(acc.EvmAcct.Address, common.FromHex(meerchange.MeerchangeMetaData.Bin), 0)
	if err != nil {
		t.Fatal(err)
	}
	expectAddr := "0xC22dAf4BDa54B15c71EdcaCE5F89b9824ef23699"
	if addr.String() != expectAddr {
		t.Fatalf("Current:%s, but expect:%s", addr.String(), expectAddr)
	}
}
