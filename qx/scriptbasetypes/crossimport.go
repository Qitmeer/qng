package scriptbasetypes

import (
	"encoding/hex"
	"fmt"
	qconsensus "github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

type CrossImportScript struct {
}

// evm to meer lock script
func (this *CrossImportScript) Sign(privKey string, mtx *types.Transaction, inputIndex int, param *params.Params) error {
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
