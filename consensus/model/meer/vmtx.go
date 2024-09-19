package meer

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/meer/meerchange"
	"github.com/Qitmeer/qng/params"
	etypes "github.com/ethereum/go-ethereum/core/types"
)

type VMTx struct {
	*Tx
	Coinbase hash.Hash
	ETx      *etypes.Transaction

	ExportData *meerchange.MeerchangeExportData
	ImportData *meerchange.MeerchangeImportData
}

func (vt *VMTx) setCoinbaseTx(tx *types.Transaction) error {
	_, pksAddrs, _, err := txscript.ExtractPkScriptAddrs(tx.TxOut[0].PkScript, params.ActiveNetParams.Params)
	if err != nil {
		return err
	}
	if len(pksAddrs) > 0 {
		secpPksAddr, ok := pksAddrs[0].(*address.SecpPubKeyAddress)
		if !ok {
			return fmt.Errorf("Not SecpPubKeyAddress:%s", pksAddrs[0].String())
		}
		vt.To = secpPksAddr.PubKey().SerializeUncompressed()
		vt.Coinbase = tx.TxHash()
		return nil
	}
	return fmt.Errorf("tx format error :TxTypeCrossChainVM")
}

func NewVMTx(tx *types.Transaction, coinbase *types.Transaction) (*VMTx, error) {
	if !opreturn.IsMeerEVM(tx.TxOut[0].PkScript) {
		return nil, fmt.Errorf("Not MeerVM tx")
	}

	vt := &VMTx{
		Tx: &Tx{Type: types.TxTypeCrossChainVM, Data: common.ToTxHex(tx.TxIn[0].SignScript)},
	}
	if coinbase != nil {
		err := vt.setCoinbaseTx(coinbase)
		if err != nil {
			return nil, err
		}
	}
	me, err := opreturn.NewOPReturnFrom(tx.TxOut[0].PkScript)
	if err != nil {
		return nil, err
	}
	err = me.Verify(tx)
	if err != nil {
		return nil, err
	}
	txb := vt.GetData()
	var txe = &etypes.Transaction{}
	if err := txe.UnmarshalBinary(txb); err != nil {
		return nil, fmt.Errorf("rlp decoding failed: %v", err)
	}
	vt.ETx = txe
	if meerchange.IsMeerChangeExportTx(txe) {
		ed, err := meerchange.NewMeerchangeExportDataByInput(txe.Data())
		if err != nil {
			return nil, err
		}
		vt.ExportData = ed
	} else if meerchange.IsEntrypointExportTx(txe) {
		ed, err := meerchange.NewEntrypointExportDataByInput(txe.Data())
		if err != nil {
			return nil, err
		}
		vt.ExportData = ed
	} else if meerchange.IsMeerChangeImportTx(txe) {
		ed, err := meerchange.NewMeerchangeImportData(tx, txe)
		if err != nil {
			return nil, err
		}
		vt.ImportData = ed
	}
	return vt, nil
}
