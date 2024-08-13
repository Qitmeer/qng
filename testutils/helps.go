package testutils

import (
	"strconv"
	"testing"

	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
)

// GenerateBlocks will generate a number of blocks by the input number
// It will return the hashes of the generated blocks or an error
func GenerateBlocks(t *testing.T, node *MockNode, num uint64) []*hash.Hash {
	result := make([]*hash.Hash, 0)
	blocks, err := node.GetPrivateMinerAPI().Generate(uint32(num), pow.MEERXKECCAKV1)
	if err != nil {
		t.Errorf("generate block failed : %v", err)
		return nil
	}

	for _, b := range blocks {
		bh := hash.MustHexToDecodedHash(b)
		result = append(result, &bh)
		t.Logf("%v: generate block [%v] ok", node.ID(), b)
	}
	return result
}

// AssertBlockOrderHeightTotal will verify the current block order, total block number
// and current main-chain height of the appointed test node and assert it ok or
// cause the test failed.
func AssertBlockOrderHeightTotal(t *testing.T, node *MockNode, order, total, height uint) {
	// order
	c, err := node.GetPublicBlockAPI().GetBlockCount()
	if err != nil {
		t.Errorf("test failed : %v", err)
	} else {
		expect := order
		if c.(uint) != expect {
			t.Errorf("test failed, expect %v , but got %v", expect, c)
		}
	}
	// total block
	tal, err := node.GetPublicBlockAPI().GetBlockTotal()
	if err != nil {
		t.Errorf("test failed : %v", err)
	} else {
		expect := total
		if tal != expect {
			t.Errorf("test failed, expect %v , but got %v", expect, tal)
		}
	}
	// main height
	h, err := node.GetPublicBlockAPI().GetMainChainHeight()
	if err != nil {
		t.Errorf("test failed : %v", err)
	} else {
		expect := height
		hi, err := strconv.ParseUint(h.(string), 10, 64)
		if err != nil {
			t.Errorf("test failed : %v", err)
		}
		if hi != uint64(expect) {
			t.Errorf("test failed, expect %v , but got %v", expect, h)
		}
	}
}

// spend first HD account to new address create by HD
func SpendUtxo(t *testing.T, node *MockNode, preOutpoint *types.TxOutPoint, amt types.Amount, lockTime int64) (*types.Transaction, types.Address) {
	addr, err := node.NewAddress()
	if err != nil {
		t.Fatalf("failed to generate new address for test wallet: %v", err)
	}
	t.Logf("test wallet generated new address %v ok", addr.String())
	feeRate := int64(10)

	inputs := []json.TransactionInput{json.TransactionInput{Txid: preOutpoint.Hash.String(), Vout: preOutpoint.OutIndex}}
	aa := json.AdreesAmount{}
	aa[addr.PKHAddress().String()] = json.Amout{CoinId: uint16(amt.Id), Amount: amt.Value - feeRate}
	tx, err := node.GetWalletManager().SpendUtxo(inputs, aa, &lockTime)
	if err != nil {
		t.Fatal(err)
	}
	return tx, addr.PKHAddress()
}
