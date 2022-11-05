package txbasetypes

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	qconsensus "github.com/Qitmeer/qng/consensus/vm"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

type TxTypeCrossChainImportUTXO struct {
	BaseUTXO
}

func (this *TxTypeCrossChainImportUTXO) AssembleVin(mtx *types.Transaction) error {
	h, err := hash.NewHashFromStr(this.TxID)
	if err != nil {
		return err
	}
	mtx.AddTxIn(&types.TxInput{
		PreviousOut: *types.NewOutPoint(h, types.SupperPrevOutIndex),
		Sequence:    uint32(types.TxTypeCrossChainImport),
	})
	return nil
}

func (this *TxTypeCrossChainImportUTXO) AssembleVout(mtx *types.Transaction) error {
	addr, err := address.DecodeAddress(this.Addr)
	if err != nil {
		return fmt.Errorf("could not decode "+
			"address: %v", err)
	}

	pkAddr, ok := addr.(*address.SecpPubKeyAddress)
	if !ok {
		return fmt.Errorf("invalid type: %T", addr)
	}
	pkScript, err := txscript.PayToAddrScript(pkAddr.PKHAddress())
	if err != nil {
		return fmt.Errorf("invalid Pay to address script: %T", addr)
	}
	txOut := types.NewTxOutput(this.Amount, pkScript)
	mtx.AddTxOut(txOut)
	return nil
}

type TxTypeSignImport struct {
}

func (this *TxTypeSignImport) Sign(privKey string, mtx *types.Transaction, inputIndex int, param *params.Params) error {
	privkeyByte, err := hex.DecodeString(privKey)
	if err != nil {
		return err
	}
	if len(privkeyByte) != 32 {
		return fmt.Errorf("invaid ec private key bytes: %d", len(privkeyByte))
	}
	privateKey, pubkey := ecc.Secp256k1.PrivKeyFromBytes(privkeyByte)
	addr, err := address.NewSecpPubKeyAddress(pubkey.SerializeCompressed(), param)
	if err != nil {
		return err
	}
	pkaScript, err := txscript.NewScriptBuilder().AddData([]byte(addr.String())).Script()
	if err != nil {
		return err
	}
	mtx.TxIn[inputIndex].SignScript = pkaScript
	itx, err := qconsensus.NewImportTx(mtx)
	if err != nil {
		return err
	}
	err = itx.Sign(privateKey)
	if err != nil {
		return err
	}
	return nil
}
