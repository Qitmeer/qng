package blockchain

import (
	"github.com/Qitmeer/qng/params"
	"testing"
)

func TestBlockClone(t *testing.T) {
	block := params.PrivNetParam.GenesisBlock
	cloneBlock, err := block.Block().Clone()
	if err != nil {
		t.Fatal(err)
	}
	cbHash := cloneBlock.BlockHash()
	if !block.Hash().IsEqual(&cbHash) {
		t.Fatalf("block hash:%s != %s (expect)", cbHash.String(), block.Hash().String())
	}
}
