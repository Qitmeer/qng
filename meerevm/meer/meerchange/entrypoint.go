package meerchange

import (
	"fmt"
	"github.com/Qitmeer/qng/meerevm/meer/entrypoint"
	"github.com/Qitmeer/qng/meerevm/meer/qngaccount"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
)

const (
	handleOps = "handleOps"
	execute   = "execute"
)

type HandleOpsData struct {
	Ops         []entrypoint.UserOperation
	Beneficiary common.Address
}

type ExecuteData struct {
	Dest  common.Address
	Value *big.Int
	Func  []byte
}

func IsEntrypointMeerChangeTx(tx *types.Transaction) bool {
	// TODO: In the future, we should be able to obtain deterministic 4337 address
	data, err := parseHandleOpsData(tx.Data())
	if err != nil {
		return false
	}
	for _, op := range data.Ops {
		ex, err := parseExecuteData(op.CallData)
		if err != nil || ex == nil {
			continue
		}
		if ex.Dest == common.HexToAddress(params.ActiveNetParams.MeerChangeContractAddr) {
			return true
		}
	}
	return false
}

func IsEntrypointExport4337Tx(tx *types.Transaction) bool {
	// TODO: In the future, we should be able to obtain deterministic 4337 address
	data, err := parseHandleOpsData(tx.Data())
	if err != nil {
		return false
	}
	for _, op := range data.Ops {
		ex, err := parseExecuteData(op.CallData)
		if err != nil || ex == nil {
			continue
		}
		if ex.Dest != common.HexToAddress(params.ActiveNetParams.MeerChangeContractAddr) {
			continue
		}
		if isMeerChangeExport4337TxByData(ex.Func) {
			return true
		}
	}
	return false
}

func parseHandleOpsData(data []byte) (*HandleOpsData, error) {
	if len(data) <= funcSigHashLen {
		return nil, fmt.Errorf("data is too short")
	}
	contractAbi, err := abi.JSON(strings.NewReader(entrypoint.EntrypointMetaData.ABI))
	if err != nil {
		return nil, err
	}

	method, err := contractAbi.MethodById(data[:funcSigHashLen])
	if err != nil {
		return nil, err
	}
	if method.Name != handleOps {
		return nil, fmt.Errorf("Inconsistent methods and parameters:%s, expect:%s", method.Name, handleOps)
	}
	unpacked, err := method.Inputs.Unpack(data[funcSigHashLen:])
	if err != nil {
		return nil, err
	}
	hoData := HandleOpsData{}
	err = method.Inputs.Copy(&hoData, unpacked)
	if err != nil {
		return nil, err
	}
	return &hoData, nil
}

func parseExecuteData(data []byte) (*ExecuteData, error) {
	if len(data) <= funcSigHashLen {
		return nil, fmt.Errorf("export 4337 data error")
	}
	contractAbi, err := abi.JSON(strings.NewReader(qngaccount.QngaccountMetaData.ABI))
	if err != nil {
		return nil, err
	}
	method, err := contractAbi.MethodById(data[:funcSigHashLen])
	if err != nil {
		return nil, err
	}
	if method.Name != execute {
		return nil, fmt.Errorf("Inconsistent methods and parameters:%s, expect:%s", method.Name, execute)
	}
	unpacked, err := method.Inputs.Unpack(data[funcSigHashLen:])
	if err != nil {
		return nil, err
	}

	eData := ExecuteData{}
	err = method.Inputs.Copy(&eData, unpacked)
	if err != nil {
		return nil, err
	}
	return &eData, nil
}

func NewEntrypointExport4337DataByInput(txdata []byte) (*MeerchangeExport4337Data, error) {
	data, err := parseHandleOpsData(txdata)
	if err != nil {
		return nil, err
	}
	has := false
	var edata []byte
	for _, op := range data.Ops {
		ex, err := parseExecuteData(op.CallData)
		if err != nil || ex == nil {
			continue
		}
		edata = ex.Func
		if isMeerChangeExport4337TxByData(edata) {
			if has {
				return nil, fmt.Errorf("Cannot support multiple MeerChange call in one transaction")
			}
			has = true
		}
	}
	if len(edata) <= 0 {
		return nil, fmt.Errorf("No Export4337Data")
	}
	return NewMeerchangeExport4337DataByInput(edata)
}
