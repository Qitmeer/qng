package meerchange

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	qtypes "github.com/Qitmeer/qng/core/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"strconv"
	"strings"
)

var (
	LogExportSigHash = crypto.Keccak256Hash([]byte("Export(string,uint64,string)"))
)

func CalcExportHash(ops string, fee uint64) common.Hash {
	data := []byte(ops)

	var sfee [8]byte
	binary.BigEndian.PutUint64(sfee[:], fee)
	data = append(data, sfee[:]...)

	return common.BytesToHash(accounts.TextHash(data))
}

func CalcExportSig(hash common.Hash, privKeyHex string) ([]byte, error) {
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, err
	}
	return crypto.Sign(hash.Bytes(), privateKey)
}

type MeerchangeExportOptData struct {
	Ops string
	Fee uint64 // Atoms per meer
	Sig string
}

type MeerchangeExportData struct {
	Opt MeerchangeExportOptData
	//
	OutPoints []*qtypes.TxOutPoint
	Amount    qtypes.Amount
}

func (e *MeerchangeExportData) GetFuncName() string {
	return "export"
}

func (e *MeerchangeExportData) GetLogName() string {
	return "Export"
}

func (e *MeerchangeExportData) GetOutPoints() ([]*qtypes.TxOutPoint, error) {
	if e.OutPoints != nil {
		return e.OutPoints, nil
	}
	if len(e.Opt.Ops) <= 0 {
		return nil, fmt.Errorf("No outpoint in meerchang export")
	}
	opsArr := strings.Split(e.Opt.Ops, ",")
	if len(opsArr) <= 0 {
		return nil, fmt.Errorf("No outpoint in meerchang export")
	}
	opsM := map[string]bool{}
	e.OutPoints = []*qtypes.TxOutPoint{}
	for _, opiStr := range opsArr {
		_, ok := opsM[opiStr]
		if ok {
			return nil, fmt.Errorf("Duplicate utxo:%s", opiStr)
		}
		opiArr := strings.Split(opiStr, ":")
		if len(opiArr) != 2 {
			return nil, fmt.Errorf("MeerChange export parmas error:%v", e.Opt.Ops)
		}
		txid, err := hash.NewHashFromStr(opiArr[0])
		if err != nil {
			return nil, err
		}
		idx, err := strconv.ParseInt(opiArr[1], 10, 32)
		if err != nil {
			return nil, err
		}
		op := qtypes.NewOutPoint(txid, uint32(idx))
		e.OutPoints = append(e.OutPoints, op)
		opsM[opiStr] = true
	}
	return e.OutPoints, nil
}

func (e MeerchangeExportData) GetMaster() (common.Address, error) {
	pub, err := e.GetMasterPubkey()
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

func (e MeerchangeExportData) GetMasterPubkey() ([]byte, error) {
	eHash := CalcExportHash(e.Opt.Ops, e.Opt.Fee)
	sig, err := hex.DecodeString(e.Opt.Sig)
	if err != nil {
		return nil, err
	}
	return crypto.Ecrecover(eHash.Bytes(), sig)
}

func NewMeerchangeExportDataByLog(data []byte) (*MeerchangeExportData, error) {
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return nil, err
	}
	ced := &MeerchangeExportData{
		Opt:       MeerchangeExportOptData{},
		OutPoints: nil,
		Amount:    qtypes.Amount{Value: 0, Id: qtypes.MEERA},
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
		Opt:       MeerchangeExportOptData{},
		OutPoints: nil,
		Amount:    qtypes.Amount{Value: 0, Id: qtypes.MEERA},
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
	return isMeerChangeExportTxByData(tx.Data())
}

func isMeerChangeExportTxByData(data []byte) bool {
	if len(data) <= funcSigHashLen {
		return false
	}
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return false
	}
	method, err := contractAbi.MethodById(data[:funcSigHashLen])
	if err != nil {
		return false
	}
	if method.Name != (&MeerchangeExportData{}).GetFuncName() {
		return false
	}
	return true
}
