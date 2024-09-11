package amana

import (
	"github.com/ethereum/go-ethereum/core"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/triedb"
)

func TestInvalidCliqueConfig(t *testing.T) {
	block := AmanaGenesis()
	block.ExtraData = []byte{}
	db := rawdb.NewMemoryDatabase()
	if _, err := block.Commit(db, triedb.NewDatabase(db, nil)); err == nil {
		t.Fatal("Expected error on invalid clique config")
	}
}

func TestGenesisHashes(t *testing.T) {
	for i, c := range []struct {
		genesis *core.Genesis
		want    common.Hash
	}{
		{AmanaGenesis(), GenesisHash},
	} {
		// Test via MustCommit
		db := rawdb.NewMemoryDatabase()
		if have := c.genesis.MustCommit(db, triedb.NewDatabase(db, triedb.HashDefaults)).Hash(); have != c.want {
			t.Errorf("case: %d a), want: %s, got: %s", i, c.want.Hex(), have.Hex())
		}
		// Test via ToBlock
		if have := c.genesis.ToBlock().Hash(); have != c.want {
			t.Errorf("case: %d a), want: %s, got: %s", i, c.want.Hex(), have.Hex())
		}
	}
}
