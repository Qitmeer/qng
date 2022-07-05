package txbasetypes

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
	"log"
)

type TxTypeGenesisLockUTXO struct {
	BaseUTXO
}

func (this *TxTypeGenesisLockUTXO) AssembleVin(mtx *types.Transaction) error {
	txHash, err := hash.NewHashFromStr(this.TxID)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	prevOut := types.NewOutPoint(txHash, this.OutIndex)
	txIn := types.NewTxInput(prevOut, []byte{})
	if this.Sequence <= 0 {
		return errors.New("genesis lock tx need unlock time")
	}
	txIn.Sequence = this.Sequence
	mtx.AddTxIn(txIn)
	return nil
}

func (this *TxTypeGenesisLockUTXO) AssembleVout(mtx *types.Transaction) error {
	addr, err := address.DecodeAddress(this.Addr)
	if err != nil {
		return fmt.Errorf("could not decode "+
			"address: %v", err)
	}

	switch addr.(type) {
	case *address.PubKeyHashAddress:
	case *address.SecpPubKeyAddress:
	case *address.ScriptHashAddress:
	default:
		return fmt.Errorf("invalid type: %T", addr)
	}

	pkScript, err := txscript.PayToAddrScript(addr)
	if this.LockTime > 0 { // lockTx
		pkScript, err = txscript.PayToCLTVPubKeyHashScript(addr.Script(), this.LockTime)
	}
	if err != nil {
		return err
	}
	txOut := types.NewTxOutput(this.Amount, pkScript)
	mtx.AddTxOut(txOut)
	return nil
}

type TxTypeSignGenesisBlock struct {
	BaseSign
}

func (this *TxTypeSignGenesisBlock) Sign(privKey string, mtx *types.Transaction, inputIndex int, param *params.Params) error {
	privkeyByte, err := hex.DecodeString(privKey)
	if err != nil {
		return err
	}
	if len(privkeyByte) != 32 {
		return fmt.Errorf("invaid ec private key bytes: %d", len(privkeyByte))
	}
	privateKey, pubkey := ecc.Secp256k1.PrivKeyFromBytes(privkeyByte)
	h160 := hash.Hash160(pubkey.SerializeCompressed())
	addr, err := address.NewPubKeyHashAddress(h160, param, ecc.ECDSA_Secp256k1)
	if err != nil {
		return err
	}
	pkScript, err := txscript.PayToCLTVPubKeyHashScript(addr.Script(), int64(mtx.TxIn[inputIndex].Sequence))
	if err != nil {
		log.Fatalln("PayToAddrScript Error", err)
		return err
	}
	mtx.TxIn[inputIndex].Sequence = types.MaxTxInSequenceNum - 1
	var kdb txscript.KeyClosure = func(types.Address) (ecc.PrivateKey, bool, error) {
		return privateKey, true, nil // compressed is true
	}
	sigScript, err := txscript.SignTxOutput(param, mtx, inputIndex, pkScript, txscript.SigHashAll, kdb, nil, nil, ecc.ECDSA_Secp256k1)
	if err != nil {
		return err
	}
	mtx.TxIn[inputIndex].SignScript = sigScript
	return nil
}
