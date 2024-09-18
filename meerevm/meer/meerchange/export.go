package meerchange

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	qtypes "github.com/Qitmeer/qng/core/types"
	mcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"strings"
)

var (
	LogExportSigHash = crypto.Keccak256Hash([]byte("Export(bytes32,uint32)"))
)

type MeerchangeExportOptData struct {
	Txid [32]byte
	Idx  uint32
}

type MeerchangeExportData struct {
	Opt MeerchangeExportOptData
	//
	OutPoint *qtypes.TxOutPoint
	Amount   qtypes.Amount
}

func (e *MeerchangeExportData) GetFuncName() string {
	return "export"
}

func (e *MeerchangeExportData) GetLogName() string {
	return "Export"
}

func (e *MeerchangeExportData) GetOutPoint() (*qtypes.TxOutPoint, error) {
	if e.OutPoint != nil {
		return e.OutPoint, nil
	}
	txidBytes := e.Opt.Txid[:]
	txid, err := hash.NewHash(*mcommon.ReverseBytes(&txidBytes))
	if err != nil {
		return nil, err
	}
	e.OutPoint = qtypes.NewOutPoint(txid, e.Opt.Idx)
	return e.OutPoint, nil
}

func NewMeerchangeExportDataByLog(data []byte) (*MeerchangeExportData, error) {
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return nil, err
	}
	ced := &MeerchangeExportData{
		Opt:      MeerchangeExportOptData{},
		OutPoint: nil,
		Amount:   qtypes.Amount{Value: 0, Id: qtypes.MEERA},
	}
	err = contractAbi.UnpackIntoInterface(&ced.Opt, ced.GetLogName(), data)
	if err != nil {
		return nil, err
	}
	return ced, nil
}

func NewMeerchangeExportDataByInput(data []byte) (*MeerchangeExportData, error) {
	if len(data) <= funcSigHashLen {
		return nil, fmt.Errorf("input data format error")
	}
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return nil, err
	}
	ced := &MeerchangeExportData{
		Opt:      MeerchangeExportOptData{},
		OutPoint: nil,
		Amount:   qtypes.Amount{Value: 0, Id: qtypes.MEERA},
	}
	method, err := contractAbi.MethodById(data[:funcSigHashLen])
	if err != nil {
		return nil, err
	}
	if method.Name != ced.GetFuncName() {
		return nil, fmt.Errorf("Inconsistent methods and parameters:%s, expect:%s", method.Name, ced.GetFuncName())
	}
	unpacked, err := method.Inputs.Unpack(data[funcSigHashLen:])
	if err != nil {
		return nil, err
	}
	err = method.Inputs.Copy(&ced.Opt, unpacked)
	if err != nil {
		return nil, err
	}
	return ced, nil
}

func IsMeerChangeExportTx(tx *types.Transaction) bool {
	if !IsDirectMeerChangeTx(tx) {
		return false
	}
	if len(tx.Data()) <= funcSigHashLen {
		return false
	}
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return false
	}
	method, err := contractAbi.MethodById(tx.Data()[:funcSigHashLen])
	if err != nil {
		return false
	}
	if method.Name != (&MeerchangeExportData{}).GetFuncName() {
		return false
	}
	return true
}
