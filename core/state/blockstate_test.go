package state

import (
	"github.com/Qitmeer/qng/v2/consensus/model"
	"testing"
)

func TestBlockStatus(t *testing.T) {
	bs := &BlockState{status: model.StatusNone}
	if bs.status.KnownInvalid() {
		t.Fatal("status", bs.status)
	}
	bs.Valid()
	if bs.status.KnownInvalid() {
		t.Fatal("status", bs.status)
	}
	bs.Invalid()
	if !bs.status.KnownInvalid() {
		t.Fatal("status", bs.status)
	}
}
