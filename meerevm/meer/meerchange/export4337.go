package meerchange

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	qtypes "github.com/Qitmeer/qng/core/types"
	mcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"strings"
)

var (
	LogExport4337SigHash = crypto.Keccak256Hash([]byte("Export4337(bytes32,uint32,uint64,string)"))
)

func CalcExport4337Hash(txid *hash.Hash, idx uint32, fee uint64) common.Hash {
	data := txid.CloneBytes()

	var sidx [4]byte
	binary.BigEndian.PutUint32(sidx[:], idx)
	data = append(data, sidx[:]...)

	var sfee [8]byte
	binary.BigEndian.PutUint64(sfee[:], fee)
	data = append(data, sfee[:]...)

	return common.BytesToHash(accounts.TextHash(data))
}

func CalcExport4337Sig(hash common.Hash, privKeyHex string) ([]byte, error) {
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, err
	}
	return crypto.Sign(hash.Bytes(), privateKey)
}

type MeerchangeExport4337OptData struct {
	Txid [32]byte
	Idx  uint32
	Fee  uint64 // Atoms per meer
	Sig  string
}

type MeerchangeExport4337Data struct {
	Opt MeerchangeExport4337OptData
	//
	OutPoint *qtypes.TxOutPoint
	Amount   qtypes.Amount
}

func (e *MeerchangeExport4337Data) GetFuncName() string {
	return "export4337"
}

func (e *MeerchangeExport4337Data) GetLogName() string {
	return "Export4337"
}

func (e *MeerchangeExport4337Data) GetOutPoint() (*qtypes.TxOutPoint, error) {
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

func (e MeerchangeExport4337Data) GetMaster() (common.Address, error) {
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

func (e MeerchangeExport4337Data) GetMasterPubkey() ([]byte, error) {
	op, err := e.GetOutPoint()
	if err != nil {
		return nil, err
	}
	eHash := CalcExport4337Hash(&op.Hash, op.OutIndex, e.Opt.Fee)
	sig, err := hex.DecodeString(e.Opt.Sig)
	if err != nil {
		return nil, err
	}
	return crypto.Ecrecover(eHash.Bytes(), sig)
}

func NewMeerchangeExport4337DataByLog(data []byte) (*MeerchangeExport4337Data, error) {
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return nil, err
	}
	ced := &MeerchangeExport4337Data{
		Opt:      MeerchangeExport4337OptData{},
		OutPoint: nil,
		Amount:   qtypes.Amount{Value: 0, Id: qtypes.MEERA},
	}
	err = contractAbi.UnpackIntoInterface(&ced.Opt, ced.GetLogName(), data)
	if err != nil {
		return nil, err
	}
	return ced, nil
}

func NewMeerchangeExport4337DataByInput(data []byte) (*MeerchangeExport4337Data, error) {
	if len(data) <= funcSigHashLen {
		return nil, fmt.Errorf("input data format error")
	}
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return nil, err
	}
	ced := &MeerchangeExport4337Data{
		Opt:      MeerchangeExport4337OptData{},
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

func IsMeerChangeExport4337Tx(tx *types.Transaction) bool {
	if !IsDirectMeerChangeTx(tx) {
		return false
	}
	return isMeerChangeExport4337TxByData(tx.Data())
}

func isMeerChangeExport4337TxByData(data []byte) bool {
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
	if method.Name != (&MeerchangeExport4337Data{}).GetFuncName() {
		return false
	}
	return true
}
