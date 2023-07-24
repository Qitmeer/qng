package test

import (
	"fmt"
	"github.com/Qitmeer/qng/meerdag"
	"testing"
)

func TestGhostDAGBlueSetFig1(t *testing.T) {
	ibd := InitBlockDAG(meerdag.GHOSTDAG, "PH_fig1-blocks")
	if ibd == nil {
		t.FailNow()
	}
	ph := ibd.(*meerdag.GhostDAG)
	//
	blueSet := ph.GetBlueSet()
	fmt.Println("Fig1 blue set：")
	printBlockSetTag(blueSet)
	if !processResult(blueSet, changeToIDList(testData.GD_BlueSetFig1.Output)) {
		t.FailNow()
	}
}

func TestGhostDAGBlueSetFig2(t *testing.T) {
	ibd := InitBlockDAG(meerdag.GHOSTDAG, "PH_fig2-blocks")
	if ibd == nil {
		t.FailNow()
	}
	ph := ibd.(*meerdag.GhostDAG)
	//
	blueSet := ph.GetBlueSet()
	fmt.Println("Fig2 blue set：")
	printBlockSetTag(blueSet)
	if !processResult(blueSet, changeToIDList(testData.GD_BlueSetFig2.Output)) {
		t.FailNow()
	}
}

func TestGhostDAGBlueSetFig4(t *testing.T) {
	ibd := InitBlockDAG(meerdag.GHOSTDAG, "PH_fig4-blocks")
	if ibd == nil {
		t.FailNow()
	}
	ph := ibd.(*meerdag.GhostDAG)
	//
	blueSet := ph.GetBlueSet()
	fmt.Println("Fig4 blue set：")
	printBlockSetTag(blueSet)
	if !processResult(blueSet, changeToIDList(testData.GD_BlueSetFig4.Output)) {
		t.FailNow()
	}
}

func TestGhostDAGOrderFig1(t *testing.T) {
	ibd := InitBlockDAG(meerdag.GHOSTDAG, "PH_fig1-blocks")
	if ibd == nil {
		t.FailNow()
	}
	ph := ibd.(*meerdag.GhostDAG)
	order := []uint{}
	var i uint
	err := ph.UpdateOrders()
	if err != nil {
		t.Fatal(err)
	}
	for i = 0; i < bd.GetBlockTotal(); i++ {
		order = append(order, bd.GetBlockByOrder(uint(i)).GetID())
	}
	fmt.Printf("The Fig.1 Order: ")
	printBlockChainTag(order)

	if !processResult(order, changeToIDList(testData.GD_OrderFig1.Output)) {
		t.FailNow()
	}
}

func TestGhostDAGOrderFig2(t *testing.T) {
	ibd := InitBlockDAG(meerdag.GHOSTDAG, "PH_fig2-blocks")
	if ibd == nil {
		t.FailNow()
	}
	ph := ibd.(*meerdag.GhostDAG)
	order := []uint{}
	var i uint
	err := ph.UpdateOrders()
	if err != nil {
		t.Fatal(err)
	}
	for i = 0; i < bd.GetBlockTotal(); i++ {
		order = append(order, bd.GetBlockByOrder(uint(i)).GetID())
	}
	fmt.Printf("The Fig.2 Order: ")
	printBlockChainTag(order)

	if !processResult(order, changeToIDList(testData.GD_OrderFig2.Output)) {
		t.FailNow()
	}
}

func TestGhostDAGOrderFig4(t *testing.T) {
	ibd := InitBlockDAG(meerdag.GHOSTDAG, "PH_fig4-blocks")
	if ibd == nil {
		t.FailNow()
	}
	ph := ibd.(*meerdag.GhostDAG)
	order := []uint{}
	var i uint
	err := ph.UpdateOrders()
	if err != nil {
		t.Fatal(err)
	}
	for i = 0; i < bd.GetBlockTotal(); i++ {
		order = append(order, bd.GetBlockByOrder(uint(i)).GetID())
	}
	fmt.Printf("The Fig.4 Order: ")
	printBlockChainTag(order)

	if !processResult(order, changeToIDList(testData.GD_OrderFig4.Output)) {
		t.FailNow()
	}
}
