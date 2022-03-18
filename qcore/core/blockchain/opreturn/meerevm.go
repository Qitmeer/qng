package opreturn

import (
	"fmt"
	"github.com/Qitmeer/qng-core/core/types"
	"github.com/Qitmeer/qng-core/engine/txscript"
)

type MeerEVM struct {
}

func (m *MeerEVM) GetType() OPReturnType {
	return OPReturnType(txscript.OP_MEER_EVM)
}

func (m *MeerEVM) Verify(tx *types.Transaction) error {
	if len(tx.TxOut) != 1 || len(tx.TxIn) != 1 {
		return fmt.Errorf("Tx is error")
	}
	if tx.TxOut[0].Amount.Id != types.ETHID {
		return fmt.Errorf("tx is not %s", types.ETHID.Name())
	}
	if tx.TxOut[0].Amount.Value != 0 {
		return fmt.Errorf("Tx output value must zero")
	}
	if len(tx.TxOut[0].PkScript) <= 0 {
		return fmt.Errorf("tx output is empty")
	}
	if len(tx.TxIn[0].SignScript) <= 0 {
		return fmt.Errorf("tx input is empty")
	}
	return nil
}

func (m *MeerEVM) Init(ops []txscript.ParsedOpcode) error {
	if len(ops) < 2 {
		return fmt.Errorf("Illegal %s", m.GetType().Name())
	}
	return nil
}

func (m *MeerEVM) PKScript() []byte {
	pks, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddOp(txscript.OP_MEER_EVM).Script()
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return pks
}

func NewEVMTx() *MeerEVM {
	return &MeerEVM{}
}

func IsMeerEVM(pks []byte) bool {
	t := GetOPReturnType(pks)
	return t == OPReturnType(txscript.OP_MEER_EVM)
}

func IsMeerEVMTx(tx *types.Transaction) bool {
	if !types.IsCrossChainVMTx(tx) {
		return false
	}
	return IsMeerEVM(tx.TxOut[0].PkScript)
}
