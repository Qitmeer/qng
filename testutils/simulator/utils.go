package simulator

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types/pow"
	"strconv"
	"testing"
)

// GenerateBlock will generate a number of blocks by the input number
// It will return the hashes of the generated blocks or an error
func GenerateBlock(t *testing.T, node *MockNode, num uint64) []*hash.Hash {
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

// AssertBlockOrderAndHeight will verify the current block order, total block number
// and current main-chain height of the appointed test node and assert it ok or
// cause the test failed.
func AssertBlockOrderAndHeight(t *testing.T, node *MockNode, order, total, height uint) {
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
