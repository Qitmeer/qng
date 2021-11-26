/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package opreturn

import (
	"fmt"
	"github.com/Qitmeer/qng-core/core/types"
	"github.com/Qitmeer/qng-core/engine/txscript"
)

type OPReturnType byte

var OPRNameMap = map[OPReturnType]string{
	OPReturnType(txscript.OP_MEER_LOCK): "LockAmount",
	OPReturnType(txscript.OP_MEER_EVM):  "MeerEVM",
}

func (t OPReturnType) Name() string {
	if name, ok := OPRNameMap[t]; ok {
		return name
	} else {
		return "Unknown-OPReturn type:" + fmt.Sprintf("%d", t)
	}
}

// Exclusive to Coinbase OP Return function
type IOPReturn interface {
	GetType() OPReturnType
	Verify(tx *types.Transaction) error
	Init(ops []txscript.ParsedOpcode) error
	PKScript() []byte
}

func IsOPReturn(pks []byte) bool {
	if len(pks) <= 0 {
		return false
	}
	ops, err := txscript.ParseScript(pks)
	if err != nil {
		return false
	}
	if len(ops) <= 0 {
		return false
	}
	if ops[0].GetOpcode() == nil {
		return false
	}
	return ops[0].GetOpcode().GetValue() == txscript.OP_RETURN
}

func GetOPReturnType(pks []byte) OPReturnType {
	if len(pks) <= 0 {
		return OPReturnType(txscript.OP_0)
	}
	ops, err := txscript.ParseScript(pks)
	if err != nil {
		return OPReturnType(txscript.OP_0)
	}
	if len(ops) <= 1 {
		return OPReturnType(txscript.OP_0)
	}
	if ops[1].GetOpcode() == nil {
		return OPReturnType(txscript.OP_0)
	}
	return OPReturnType(ops[1].GetOpcode().GetValue())
}

func NewOPReturnFrom(pks []byte) (IOPReturn, error) {
	ops, err := txscript.ParseScript(pks)
	if err != nil {
		return nil, err
	}
	if len(ops) <= 0 {
		return nil, fmt.Errorf("Is is not coinbase opreturn")
	}
	opType := ops[1].GetOpcode().GetValue()
	switch opType {
	case txscript.OP_MEER_LOCK:
		sa := LockAmount{}
		err := sa.Init(ops)
		if err != nil {
			return nil, err
		}
		return &sa, nil
	case txscript.OP_MEER_EVM:
		sa := MeerEVM{}
		err := sa.Init(ops)
		if err != nil {
			return nil, err
		}
		return &sa, nil
	}
	return nil, fmt.Errorf("No support %s", OPReturnType(opType).Name())
}

func GetOPReturnTxOutput(opr IOPReturn) *types.TxOutput {
	return &types.TxOutput{
		Amount:   types.Amount{Value: 0, Id: types.MEERID},
		PkScript: opr.PKScript(),
	}
}
