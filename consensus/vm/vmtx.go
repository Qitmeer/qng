package vm

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

type VMTx struct {
	*Tx
	*types.Transaction
	vmi VMI
}

func (vt *VMTx) SetVMI(vmi VMI) {
	vt.vmi = vmi
}

func (vt *VMTx) SetCoinbaseTx(tx *types.Transaction) error {
	_, pksAddrs, _, err := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, params.ActiveNetParams.Params)
	if err != nil {
		return err
	}
	if len(pksAddrs) > 0 {
		secpPksAddr, ok := pksAddrs[0].(*address.SecpPubKeyAddress)
		if !ok {
			return fmt.Errorf(fmt.Sprintf("Not SecpPubKeyAddress:%s", pksAddrs[0].String()))
		}
		vt.To = hex.EncodeToString(secpPksAddr.PubKey().SerializeUncompressed())
		return nil
	}
	return fmt.Errorf("tx format error :TxTypeCrossChainVM")
}

func (vt *VMTx) CheckSanity() error {
	me, err := opreturn.NewOPReturnFrom(vt.TxOut[0].PkScript)
	if err != nil {
		return err
	}
	err = me.Verify(vt.Transaction)
	if err != nil {
		return err
	}
	if vt.vmi != nil {
		return vt.vmi.VerifyTxSanity(vt)
	}
	return nil
}

func NewVMTx(tx *types.Transaction) (*VMTx, error) {
	if !opreturn.IsMeerEVM(tx.TxOut[0].PkScript) {
		return nil, fmt.Errorf("Not MeerVM tx")
	}
	return &VMTx{
		Tx:          &Tx{Type: types.TxTypeCrossChainVM, Data: []byte(tx.TxIn[0].SignScript)},
		Transaction: tx,
	}, nil
}
