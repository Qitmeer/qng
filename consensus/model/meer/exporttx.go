package meer

import (
	"fmt"
	"github.com/Qitmeer/qng/v2/core/address"
	"github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/engine/txscript"
	"github.com/Qitmeer/qng/v2/params"
)

type ExportTx struct {
	*Transaction
}

func NewExportTx(tx *types.Transaction) (*ExportTx, error) {
	if !types.IsCrossChainExportTx(tx) {
		return nil, fmt.Errorf("Not import tx data:%s", tx.TxHash())
	}

	etx := &ExportTx{Transaction: &Transaction{}}
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
			return nil, fmt.Errorf("Not SecpPubKeyAddress:%s in tx(%s)", pksAddrs[0].String(), tx.TxHash())
		}
		etx.To = secpPksAddr.PubKey().SerializeUncompressed()
		etx.Value = uint64(tx.TxOut[0].Amount.Value)
	} else {
		return nil, fmt.Errorf("tx format error :TxTypeCrossChainExport in tx(%s)", tx.TxHash())
	}

	return etx, nil
}
