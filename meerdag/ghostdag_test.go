package meerdag

import (
	"fmt"
	"testing"
)

func TestGhostDAGBlueSetFig2(t *testing.T) {
	ibd := InitBlockDAG(GHOSTDAG, "PH_fig2-blocks")
	if ibd == nil {
		t.FailNow()
	}
	ph := ibd.(*GhostDAG)
	//
	blueSet := ph.GetBlueSet()
	fmt.Println("Fig2 blue set：")
	printBlockSetTag(blueSet)
}
