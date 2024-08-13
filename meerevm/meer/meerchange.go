package meer

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	qtypes "github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerevm/meer/meerchange"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/core/types"
)

func (m *MeerPool) checkMeerChangeTxs(block *types.Block, receipts types.Receipts) error {
	txsNum := len(block.Transactions())
	if txsNum <= 0 {
		return nil
	}
	if txsNum != len(receipts) {
		return fmt.Errorf("The number of txs and receipts is inconsistent")
	}
	has := false
	for _, tx := range block.Transactions() {
		if meerchange.IsMeerChangeTx(tx) {
			has = true
			break
		}
	}
	if !has {
		return nil
	}
	for i, tx := range block.Transactions() {
		if meerchange.IsMeerChangeTx(tx) {
			for _, lg := range receipts[i].Logs {
				switch lg.Topics[0].Hex() {
				case meerchange.LogExportSigHash.Hex():
					ccExportEvent, err := meerchange.NewMeerchangeExportDataByLog(lg.Data)
					if err != nil {
						return err
					}
					err = m.checkMeerChangeExportTx(tx, ccExportEvent, nil)
					if err != nil {
						m.ethTxPool.RemoveTx(tx.Hash(), true)
						return err
					}
				default:
					log.Warn("Not Supported", "addr", lg.Address.String(), "tx", lg.TxHash.String(), "topic", lg.Topics[0].Hex())
				}
			}
		}
	}
	return nil
}

func (m *MeerPool) HasUtxo(txid *hash.Hash, idx uint32) bool {
	ue, err := m.consensus.BlockChain().GetUtxo(*qtypes.NewOutPoint(txid, idx))
	return err == nil && ue != nil
}

func (m *MeerPool) checkSignature(tx *types.Transaction, entry *utxo.UtxoEntry) error {
	if len(entry.PkScript()) <= 0 {
		return fmt.Errorf("PkScript is empty")
	}

	signer := types.NewPKSigner(m.eth.BlockChain().Config().ChainID)
	pkb, err := signer.GetPublicKey(tx)
	pubKey, err := ecc.Secp256k1.ParsePubKey(pkb)
	if err != nil {
		return err
	}
	addrUn, err := address.NewSecpPubKeyAddress(pubKey.SerializeUncompressed(), params.ActiveNetParams.Params)
	if err != nil {
		return err
	}

	addr, err := address.NewSecpPubKeyAddress(pubKey.SerializeCompressed(), params.ActiveNetParams.Params)
	if err != nil {
		return err
	}

	scriptClass, pksAddrs, _, err := txscript.ExtractPkScriptAddrs(entry.PkScript(), params.ActiveNetParams.Params)
	if err != nil {
		return err
	}
	if len(pksAddrs) != 1 {
		return fmt.Errorf("PKScript num no support:%d", len(pksAddrs))
	}

	switch scriptClass {
	case txscript.PubKeyHashTy, txscript.PubkeyHashAltTy, txscript.PubKeyTy, txscript.PubkeyAltTy:
		if pksAddrs[0].Encode() == addr.PKHAddress().String() ||
			pksAddrs[0].Encode() == addrUn.PKHAddress().String() {
			return nil
		}
	default:
		return fmt.Errorf("Signature error about no support %s", scriptClass.String())
	}
	return fmt.Errorf("Signature error")
}

func (m *MeerPool) checkMeerChangeExportTx(tx *types.Transaction, ced *meerchange.MeerchangeExportData, utxoView *utxo.UtxoViewpoint) error {
	op, err := ced.GetOutPoint()
	if err != nil {
		return err
	}
	var entry *utxo.UtxoEntry
	if utxoView != nil {
		entry = utxoView.LookupEntry(*op)
	}
	ok := false
	if entry == nil {
		ue, err := m.consensus.BlockChain().GetUtxo(*op)
		if err != nil {
			return err
		}
		if ue == nil {
			return fmt.Errorf("No utxo %s:%d", op.Hash.String(), op.OutIndex)
		}
		entry, ok = ue.(*utxo.UtxoEntry)
		if !ok || entry == nil {
			return fmt.Errorf("No utxo entry %s:%d", op.Hash.String(), op.OutIndex)
		}
	}

	err = m.checkSignature(tx, entry)
	if err != nil {
		return err
	}
	ced.Amount = entry.Amount()
	if utxoView != nil && ok {
		utxoView.AddEntry(*op, entry)
	}
	return nil
}
