package test

import (
	"github.com/Qitmeer/qng/meerdag"
	"testing"
)

func Test_CheckBlueAndMature(t *testing.T) {
	ibd := InitBlockDAG(meerdag.phantom, "PH_fig2-blocks")
	if ibd == nil {
		t.FailNow()
	}
	err := bd.CheckBlueAndMature([]uint{meerdag.tbMap["D"].GetID()}, []uint{meerdag.tbMap["I"].GetID()}, 2)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_CheckBlueAndMatureMT(t *testing.T) {
	ibd := InitBlockDAG(meerdag.phantom, "PH_fig2-blocks")
	if ibd == nil {
		t.FailNow()
	}
	err := bd.CheckBlueAndMatureMT([]uint{meerdag.tbMap["D"].GetID()}, []uint{meerdag.tbMap["I"].GetID()}, 2)
	if err != nil {
		t.Fatal(err)
	}
}
