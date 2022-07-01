package txencodetypes

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
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
	if this.Sequence > 0 {
		txIn.Sequence = this.Sequence
	}
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

	pkScript, err := txscript.PayToCLTVPubKeyHashScript(addr.Script(), this.LockTime)
	if err != nil {
		return err
	}
	txOut := types.NewTxOutput(this.Amount, pkScript)
	mtx.AddTxOut(txOut)
	return nil
}
