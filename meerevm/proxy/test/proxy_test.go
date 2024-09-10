// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package test

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"testing"
)

// Source: https://github.com/Arachnid/deterministic-deployment-proxy/blob/master/scripts/test.sh
func TestDeterministicDeploymentProxy(t *testing.T) {
	//node, err := testutils.StartMockNode(nil)
	node, err := testutils.StartMockNode(func(cfg *config.Config) error {
		cfg.DebugLevel = "trace"
		cfg.DebugPrintOrigins = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer node.Stop()

	if err != nil {
		t.Fatalf("setup harness failed:%v", err)
	}
	acc := node.GetWalletManager().GetAccountByIdx(0)
	if err != nil {
		t.Fatalf("GetAcctInfo failed:%v", err)
	}
	coinbaseTxID := params.ActiveNetParams.GenesisBlock.Transactions()[1].Hash()
	txH := testutils.SendExportTxMockNode(t, node, coinbaseTxID.String(), 61, 100*types.AtomsPerCoin)
	if txH == nil {
		t.Fatalf("createExportRawTx failed:%v", err)
	}
	t.Logf("send export tx:%s", txH.String())
	testutils.GenerateBlocksWaitForTxs(t, node, []string{txH.String()})

	ba, err := node.GetEvmClient().BalanceAt(node.Node().Context(), acc.EvmAcct.Address, nil)
	if err != nil {
		t.Fatalf("GetBalance failed:%v", err)
	}
	if ba.Uint64() <= 0 {
		t.Fatalf("%s balance is empty", acc.EvmAcct.Address.String())
	}
	t.Logf("%s balance = %s", acc.EvmAcct.Address.String(), ba.String())

	MY_ADDRESS := acc.EvmAcct.Address

	// deploy our contract
	// contract: pragma solidity 0.5.8; contract Apple {function banana() external pure returns (uint8) {return 42;}}
	BYTECODE := common.FromHex("6080604052348015600f57600080fd5b5060848061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063c3cafc6f14602d575b600080fd5b6033604f565b604051808260ff1660ff16815260200191505060405180910390f35b6000602a90509056fea165627a7a72305820ab7651cb86b8c1487590004c2444f26ae30077a6b96c6bc62dda37f1328539250029")
	MY_CONTRACT_ADDRESS, err := node.DeterministicDeploymentProxy().GetContractAddress(MY_ADDRESS, BYTECODE, 0)
	if err != nil {
		t.Fatal(err)
	}
	txHash, err := node.DeterministicDeploymentProxy().DeployContract(MY_ADDRESS, BYTECODE, 0, nil, 0xf4240)
	if err != nil {
		t.Fatal(err)
	}
	testutils.GenerateBlocksWaitForTxs(t, node, []string{txHash.String()})
	t.Logf("Deply contract: txHash=%s contractAddress=%s", txHash.String(), MY_CONTRACT_ADDRESS.String())

	// call our contract (NOTE: MY_CONTRACT_ADDRESS is the same no matter what chain we deploy to!)
	MY_CONTRACT_METHOD_SIGNATURE := common.FromHex("c3cafc6f")
	addrBytes, err := node.GetEvmClient().CallContract(node.Node().Context(), ethereum.CallMsg{To: &MY_CONTRACT_ADDRESS, Data: MY_CONTRACT_METHOD_SIGNATURE}, nil)
	if err != nil {
		t.Fatal(err)
	}
	//# expected result is 0x000000000000000000000000000000000000000000000000000000000000002a (hex encoded 42)
	expect := "0x000000000000000000000000000000000000000000000000000000000000002a"
	if hexutil.Encode(addrBytes) != expect {
		t.Fatalf("Current:%s, but expect:%s", hexutil.Encode(addrBytes), expect)
	}
}
