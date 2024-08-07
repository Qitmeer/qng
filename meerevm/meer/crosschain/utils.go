package crosschain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	qtypes "github.com/Qitmeer/qng/core/types"
	mcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"strings"
)

const (
	// Make a transfer between utxo and evm
	CROSSCHAIN_CONTRACT_ADDR = "0x2000000000000000000000000000000000000000"
)

var (
	LogExportSigHash = crypto.Keccak256Hash([]byte("Export(bytes32,uint32)"))
)

type CrosschainExportOptData struct {
	Txid [32]byte
	Idx  uint32
}

type CrosschainExportData struct {
	Opt CrosschainExportOptData
	//
	OutPoint *qtypes.TxOutPoint
	Amount   qtypes.Amount
}

func (e *CrosschainExportData) GetFuncName() string {
	return "export"
}

func (e *CrosschainExportData) GetLogName() string {
	return "Export"
}

func (e *CrosschainExportData) GetOutPoint() (*qtypes.TxOutPoint, error) {
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

func NewCrosschainExportDataByLog(data []byte) (*CrosschainExportData, error) {
	contractAbi, err := abi.JSON(strings.NewReader(CrosschainMetaData.ABI))
	if err != nil {
		return nil, err
	}
	ced := &CrosschainExportData{
		Opt:      CrosschainExportOptData{},
		OutPoint: nil,
		Amount:   qtypes.Amount{Value: 0, Id: qtypes.MEERA},
	}
	err = contractAbi.UnpackIntoInterface(&ced.Opt, ced.GetLogName(), data)
	if err != nil {
		return nil, err
	}
	return ced, nil
}

func NewCrosschainExportDataByInput(data []byte) (*CrosschainExportData, error) {
	if len(data) <= 4 {
		return nil, fmt.Errorf("input data format error")
	}
	contractAbi, err := abi.JSON(strings.NewReader(CrosschainMetaData.ABI))
	if err != nil {
		return nil, err
	}
	ced := &CrosschainExportData{
		Opt:      CrosschainExportOptData{},
		OutPoint: nil,
		Amount:   qtypes.Amount{Value: 0, Id: qtypes.MEERA},
	}
	method, err := contractAbi.MethodById(data[:4])
	if err != nil {
		return nil, err
	}
	if method.Name != ced.GetFuncName() {
		return nil, fmt.Errorf("Inconsistent methods and parameters:%s, expect:%s", method.Name, ced.GetFuncName())
	}
	unpacked, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	err = method.Inputs.Copy(&ced.Opt, unpacked)
	if err != nil {
		return nil, err
	}
	return ced, nil
}

func IsCrossChainTx(tx *types.Transaction) bool {
	if tx == nil {
		return false
	}
	if tx.To() == nil {
		return false
	}
	return *tx.To() == common.HexToAddress(CROSSCHAIN_CONTRACT_ADDR)
}

func IsCrossChainExportTx(tx *types.Transaction) bool {
	if !IsCrossChainTx(tx) {
		return false
	}
	if len(tx.Data()) <= 4 {
		return false
	}
	contractAbi, err := abi.JSON(strings.NewReader(CrosschainMetaData.ABI))
	if err != nil {
		return false
	}
	method, err := contractAbi.MethodById(tx.Data()[:4])
	if err != nil {
		return false
	}
	if method.Name != (&CrosschainExportData{}).GetFuncName() {
		return false
	}
	return true
}
