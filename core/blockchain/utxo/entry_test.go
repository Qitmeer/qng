package utxo

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"testing"
)

func TestUtxoEntry(t *testing.T) {
	en0 := &UtxoEntry{}
	en0.Modified()
	en1 := NewUtxoEntry(types.Amount{}, nil, &hash.ZeroHash, false)
	en1.Modified()
	if en0.packedFlags != en1.packedFlags {
		t.FailNow()
	}
}
