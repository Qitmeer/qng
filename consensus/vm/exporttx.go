package vm

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

type ExportTx struct {
	*Tx
	*types.Transaction
}

func (etx *ExportTx) CheckSanity() error {
	if !types.IsCrossChainExportTx(etx.Transaction) {
		return fmt.Errorf("Not import tx data:%s", etx.Transaction.TxHash())
	}

	return nil
}

func NewExportTx(tx *types.Transaction) (*ExportTx, error) {

	etx := &ExportTx{Transaction: tx, Tx: &Tx{}}
	etx.Type = types.TxTypeCrossChainExport

	if len(tx.TxIn) < 1 || len(tx.TxOut) < 1 {
		return nil, fmt.Errorf("Tx fmt is error")
	}
	if len(tx.TxOut[0].PkScript) <= 0 {
		return nil, fmt.Errorf("Tx output is error:%s in tx(%s)", types.DetermineTxType(tx), tx.TxHash())
	}

	_, pksAddrs, _, err := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, params.ActiveNetParams.Params)
	if err != nil {
		return nil, err
	}

	if len(pksAddrs) > 0 {
		secpPksAddr, ok := pksAddrs[0].(*address.SecpPubKeyAddress)
		if !ok {
			return nil, fmt.Errorf(fmt.Sprintf("Not SecpPubKeyAddress:%s in tx(%s)", pksAddrs[0].String(), tx.TxHash()))
		}
		etx.To = hex.EncodeToString(secpPksAddr.PubKey().SerializeUncompressed())
		etx.Value = uint64(tx.TxOut[0].Amount.Value)
	} else {
		return nil, fmt.Errorf("tx format error :TxTypeCrossChainExport in tx(%s)", tx.TxHash())
	}

	return etx, nil
}
