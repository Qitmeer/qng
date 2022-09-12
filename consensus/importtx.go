/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

type ImportTx struct {
	*Tx
	*types.Transaction
}

func (itx *ImportTx) GetPKAddress() (*address.SecpPubKeyAddress, error) {
	pk, err := hex.DecodeString(itx.From)
	if err != nil {
		return nil, err
	}
	return address.NewSecpPubKeyAddress(pk, params.ActiveNetParams.Params)
}

func (itx *ImportTx) GetPKScript() ([]byte, error) {
	spka, err := itx.GetPKAddress()
	if err != nil {
		return nil, err
	}
	return txscript.PayToAddrScript(spka)
}

func (itx *ImportTx) Sign(privateKey ecc.PrivateKey) error {
	pks, err := itx.GetPKScript()
	if err != nil {
		return err
	}
	var kdb txscript.KeyClosure = func(types.Address) (ecc.PrivateKey, bool, error) {
		return privateKey, true, nil // compressed is true
	}

	sigScript, err := txscript.SignTxOutput(params.ActiveNetParams.Params, itx.Transaction, 0, pks, txscript.SigHashAll, kdb, nil, nil, ecc.ECDSA_Secp256k1)
	if err != nil {
		return err
	}

	spka, err := itx.GetPKAddress()
	if err != nil {
		return err
	}
	pkaScript, err := txscript.NewScriptBuilder().AddData([]byte(spka.String())).AddData(sigScript).Script()
	if err != nil {
		return err
	}

	itx.Transaction.TxIn[0].SignScript = pkaScript

	return nil
}

func (itx *ImportTx) CheckSanity() error {
	if !types.IsCrossChainImportTx(itx.Transaction) {
		return fmt.Errorf("Not import tx data")
	}
	if itx.Transaction.TxOut[0].Amount.Id != types.MEERA {
		return fmt.Errorf("Import output must MEER coin")
	}
	if len(itx.Transaction.TxOut[0].PkScript) <= 0 {
		return fmt.Errorf("PKScript is error")
	}
	_, pksAddrs, _, err := txscript.ExtractPkScriptAddrs(itx.Transaction.TxOut[0].PkScript, params.ActiveNetParams.Params)
	if err != nil {
		return err
	}
	if len(pksAddrs) <= 0 {
		return fmt.Errorf("PKScript is error")
	}
	pk, err := hex.DecodeString(itx.From)
	if err != nil {
		return err
	}
	spka, err := address.NewSecpPubKeyAddress(pk, params.ActiveNetParams.Params)
	if err != nil {
		return err
	}
	if spka.PKHAddress().Encode() == pksAddrs[0].Encode() {
		return fmt.Errorf("Address (%s) is not equal with (%s)\n", spka.PKHAddress().Encode(), pksAddrs[0].Encode())
	}
	return nil
}

func (itx *ImportTx) SetCoinbaseTx(tx *types.Transaction) error {
	_, pksAddrs, _, err := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, params.ActiveNetParams.Params)
	if err != nil {
		return err
	}
	if len(pksAddrs) > 0 {
		secpPksAddr, ok := pksAddrs[0].(*address.SecpPubKeyAddress)
		if !ok {
			return fmt.Errorf(fmt.Sprintf("Not SecpPubKeyAddress:%s", pksAddrs[0].String()))
		}
		itx.To = hex.EncodeToString(secpPksAddr.PubKey().SerializeUncompressed())
		return nil
	}
	return fmt.Errorf("tx format error :TxTypeCrossChainVM")
}

func (itx *ImportTx) GetTransactionForEngine() (*types.Transaction, error) {
	mtx := types.NewTransaction()
	mtx.AddTxIn(&types.TxInput{
		PreviousOut: itx.Transaction.TxIn[0].PreviousOut,
		Sequence:    itx.Transaction.TxIn[0].Sequence,
	})
	mtx.AddTxOut(itx.TxOut[0])

	ops, err := txscript.ParseScript(itx.Transaction.TxIn[0].SignScript)
	if err != nil {
		return nil, err
	}
	if len(ops) <= 1 {
		return nil, fmt.Errorf("No signScript")
	}
	if len(ops[1].GetData()) <= 0 {
		return nil, fmt.Errorf("No signScript")
	}
	mtx.TxIn[0].SignScript = ops[1].GetData()

	return mtx, nil
}

func NewImportTx(tx *types.Transaction) (*ImportTx, error) {

	itx := &ImportTx{Transaction: tx, Tx: &Tx{}}
	itx.Type = types.TxTypeCrossChainImport

	ops, err := txscript.ParseScript(tx.TxIn[0].SignScript)
	if err != nil {
		return nil, err
	}
	if len(ops) <= 0 {
		return nil, fmt.Errorf("No pk address")
	}
	if len(ops[0].GetData()) <= 0 {
		return nil, fmt.Errorf("Import tx script error")
	}
	addrStr := string(ops[0].GetData())
	addr, err := address.DecodeAddress(addrStr)
	if err != nil {
		return nil, err
	}
	secpPksAddr, ok := addr.(*address.SecpPubKeyAddress)
	if !ok {
		return nil, fmt.Errorf("Not SecpPubKeyAddress:%s", addr.String())
	}
	itx.From = hex.EncodeToString(secpPksAddr.PubKey().SerializeUncompressed())
	itx.Value = uint64(tx.TxOut[0].Amount.Value)
	return itx, nil
}
