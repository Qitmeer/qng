package meerchange

import (
	"bytes"
	"encoding/hex"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils/testprivatekey"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"strings"
	"testing"
)

func TestMeerChangeExportLog(t *testing.T) {
	lgData, err := hex.DecodeString("81edb1f045f05af68f77a6b8e0e06283c3a761a14084919644b141b3cd77666d0000000000000000000000000000000000000000000000000000000000000011")
	if err != nil {
		t.Fatal(err)
	}
	ccExportEvent, err := NewMeerchangeExportDataByLog(lgData)
	if err != nil {
		t.Fatal(err)
	}
	op, err := ccExportEvent.GetOutPoint()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("outpoint", "txid", op.Hash.String(), "idx", op.OutIndex)
}

func TestMeerChangeExportInput(t *testing.T) {
	inputData, err := hex.DecodeString("4cccceada1709fc7f8a02fca1c8520861440dc21d086aab5df497902bd867cb3b813806a0000000000000000000000000000000000000000000000000000000000003044")
	if err != nil {
		t.Fatal(err)
	}
	ccExportEvent, err := NewMeerchangeExportDataByInput(inputData)
	if err != nil {
		t.Fatal(err)
	}
	op, err := ccExportEvent.GetOutPoint()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("outpoint", "txid", op.Hash.String(), "idx", op.OutIndex)
}

func TestMeerChangeExport4337(t *testing.T) {
	params.ActiveNetParams = &params.PrivNetParam
	pb, err := testprivatekey.NewBuilder(0)
	if err != nil {
		t.Fatal(err)
	}
	privateKeyHex := hex.EncodeToString(pb.Get(0))
	txid := hash.MustHexToDecodedHash("0")
	dataHash := CalcExport4337Hash(&txid, 0, 123)
	sig, err := CalcExport4337Sig(dataHash, privateKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	sigPublicKey, err := crypto.Ecrecover(dataHash.Bytes(), sig)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sigPublicKey, crypto.FromECDSAPub(&privateKey.PublicKey)) {
		t.Fatalf("export4377 sig error")
	}
}

func TestMeerChangeExport4337Log(t *testing.T) {
	lgData, err := hex.DecodeString("86f227a1220d60b7287ea5c4e844b57c9e19af9c22557f581f88ac312d5f98040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000008430786232363438343635316136366565613337316461633137393837663137366235393631626165333931363961623434313931613934626339633366396631656131356261323938636232663331386265343064336533366538613965393962613537383862363633386538316166653364646632316234313663366236663931303000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	ccExportEvent, err := NewMeerchangeExport4337DataByLog(lgData)
	if err != nil {
		t.Fatal(err)
	}
	op, err := ccExportEvent.GetOutPoint()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("outpoint", "txid", op.Hash.String(), "idx", op.OutIndex, "fee", ccExportEvent.Opt.Fee, "sig", ccExportEvent.Opt.Sig)
}

func TestMeerChangeExport4337Input(t *testing.T) {
	inputData, err := hex.DecodeString("9801767c86f227a1220d60b7287ea5c4e844b57c9e19af9c22557f581f88ac312d5f98040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000008430786232363438343635316136366565613337316461633137393837663137366235393631626165333931363961623434313931613934626339633366396631656131356261323938636232663331386265343064336533366538613965393962613537383862363633386538316166653364646632316234313663366236663931303000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	ccExportEvent, err := NewMeerchangeExport4337DataByInput(inputData)
	if err != nil {
		t.Fatal(err)
	}
	op, err := ccExportEvent.GetOutPoint()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("outpoint", "txid", op.Hash.String(), "idx", op.OutIndex, "fee", ccExportEvent.Opt.Fee, "sig", ccExportEvent.Opt.Sig)
}

func TestMeerChangeImportLog(t *testing.T) {
	topicHex := "0xb9ba2e23b17fbc3f0029c3a6600ef2dd4484bea87a99c7aab54caf84dedcf96b"

	if topicHex != LogImportSigHash.String() {
		t.Fatalf("import log error:%s expect:%s", topicHex, LogImportSigHash.String())
	}
}

func TestMeerChangeImportInput(t *testing.T) {
	data, err := hex.DecodeString("a8770e69")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 4 {
		t.Fatalf("input data format error")
	}
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		t.Fatal(err)
	}

	method, err := contractAbi.MethodById(data[:4])
	if err != nil {
		t.Fatal(err)
	}
	funcName := (&MeerchangeImportData{}).GetFuncName()
	if method.Name != funcName {
		t.Fatalf("Inconsistent methods and parameters:%s, expect:%s", method.Name, funcName)
	}
}
