package scriptbasetypes

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/v2/core/address"
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/crypto/ecc"
	"github.com/Qitmeer/qng/v2/engine/txscript"
	"github.com/Qitmeer/qng/v2/params"
	"log"
)

type PubKeyScript struct {
}

func (this *PubKeyScript) Sign(privKey string, mtx *types.Transaction, inputIndex int, param *params.Params) error {
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
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		log.Fatalln("PayToAddrScript Error", err)
		return err
	}
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
