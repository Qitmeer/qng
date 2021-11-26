package opreturn

import (
	"fmt"
	"github.com/Qitmeer/qng-core/core/types"
	"github.com/Qitmeer/qng-core/engine/txscript"
)

// TODO:It should be mapped to the actual situation
const MeerEVMFee = types.AtomsPerCoin

type MeerEVM struct {
	hex string
}

func (m *MeerEVM) GetType() OPReturnType {
	return OPReturnType(txscript.OP_MEER_EVM)
}

func (m *MeerEVM) Verify(tx *types.Transaction) error {
	if len(tx.TxOut) <= 0 {
		return fmt.Errorf("Tx is error")
	}
	return nil
}

func (m *MeerEVM) Init(ops []txscript.ParsedOpcode) error {
	if len(ops) < 3 {
		return fmt.Errorf("Illegal %s", m.GetType().Name())
	}
	m.hex = string(ops[2].GetData())
	return nil
}

func (m *MeerEVM) PKScript() []byte {
	pks, err := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).AddOp(txscript.OP_MEER_EVM).AddData([]byte(m.hex)).Script()
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return pks
}

func (m *MeerEVM) GetHex() string {
	return m.hex
}

func NewEVMTx(hex string) *MeerEVM {
	return &MeerEVM{hex: hex}
}

func IsMeerEVM(pks []byte) bool {
	t := GetOPReturnType(pks)
	return t == OPReturnType(txscript.OP_MEER_EVM)
}
