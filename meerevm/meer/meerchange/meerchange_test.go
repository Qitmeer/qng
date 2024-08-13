package meerchange

import (
	"encoding/hex"
	"testing"
)

func TestCrossChainLog(t *testing.T) {
	lgData, err := hex.DecodeString("81edb1f045f05af68f77a6b8e0e06283c3a761a14084919644b141b3cd77666d0000000000000000000000000000000000000000000000000000000000000011")
	if err != nil {
		t.Fatal(err)
	}
	ccExportEvent, err := NewCrosschainExportDataByLog(lgData)
	if err != nil {
		t.Fatal(err)
	}
	op, err := ccExportEvent.GetOutPoint()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("outpoint", "txid", op.Hash.String(), "idx", op.OutIndex)
}

func TestCrossChainInput(t *testing.T) {
	inputData, err := hex.DecodeString("4cccceada1709fc7f8a02fca1c8520861440dc21d086aab5df497902bd867cb3b813806a0000000000000000000000000000000000000000000000000000000000003044")
	if err != nil {
		t.Fatal(err)
	}
	ccExportEvent, err := NewCrosschainExportDataByInput(inputData)
	if err != nil {
		t.Fatal(err)
	}
	op, err := ccExportEvent.GetOutPoint()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("outpoint", "txid", op.Hash.String(), "idx", op.OutIndex)
}
