package meerdag

import (
	"github.com/Qitmeer/qng/v2/common/hash"
	"testing"
)

func Test_Has(t *testing.T) {
	t.Parallel()
	hl := HashSlice{}
	hl = append(hl, &hash.ZeroHash)

	if !hl.Has(&hash.ZeroHash) {
		t.FailNow()
	}
}
