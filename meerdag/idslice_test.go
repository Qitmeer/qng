package meerdag

import (
	"testing"
)

func Test_HasId(t *testing.T) {
	t.Parallel()
	hl := IdSlice{}
	hl = append(hl, 0)

	if !hl.Has(0) {
		t.FailNow()
	}
}
