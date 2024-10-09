// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package test

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils"
	"testing"
)

func TestCheckUTXO(t *testing.T) {
	node, err := testutils.StartMockNode(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Stop()

	if err != nil {
		t.Fatalf("setup harness failed:%v", err)
	}
	txid := params.ActiveNetParams.GenesisBlock.Transactions()[1].Hash()
	idx := uint32(0)
	ret, err := node.GetPrivateTxAPI().CalcUTXOSig(*txid, idx, node.GetPriKeyBuilder().GetHex(0))
	if err != nil {
		t.Fatal(err)
	}
	pubkey, err := node.GetPublicTxAPI().CheckUTXO(*txid, idx, ret.(string))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("PublicKey:%s", hex.EncodeToString(pubkey.([]byte)))
}
