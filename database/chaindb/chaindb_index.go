package chaindb

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/rawdb"
	"github.com/Qitmeer/qng/meerdag"
)

func (cdb *ChainDB) PutTxIndexEntrys(sblock *types.SerializedBlock, block model.Block) error {
	return rawdb.WriteTxLookupEntriesByBlock(cdb.db, sblock, uint64(block.GetID()))
}

func (cdb *ChainDB) GetTxIndexEntry(id *hash.Hash, verbose bool) (*types.Tx, *hash.Hash, error) {
	if !verbose {
		blockID := rawdb.ReadTxLookupEntry(cdb.db, id)
		if blockID == nil {
			return nil, nil, nil
		}
		blockhash, err := meerdag.DBGetDAGBlockHashByID(cdb, *blockID)
		if err != nil {
			return nil, nil, err
		}
		return nil, blockhash, nil
	}
	tx, _, blockhash, _ := rawdb.ReadTransaction(cdb.db, id)
	return tx, blockhash, nil
}

func (cdb *ChainDB) DeleteTxIndexEntrys(block *types.SerializedBlock) error {
	for _, tx := range block.Transactions() {
		_, blockHash, _ := cdb.GetTxIndexEntry(tx.Hash(), false)
		if blockHash != nil && !blockHash.IsEqual(block.Hash()) {
			continue
		}
		err := rawdb.DeleteTxLookupEntry(cdb.db, tx.Hash())
		if err != nil {
			return err
		}
	}
	return nil
}

func (cdb *ChainDB) PutTxHashs(block *types.SerializedBlock) error {
	for _, tx := range block.Transactions() {
		if tx.IsDuplicate {
			continue
		}
		fhash := tx.Tx.TxHashFull()
		err := rawdb.WriteTxIdByFullHash(cdb.db, &fhash, tx.Hash())
		if err != nil {
			return err
		}
	}
	return nil
}

func (cdb *ChainDB) GetTxIdByHash(fullHash *hash.Hash) (*hash.Hash, error) {
	txid := rawdb.ReadTxIdByFullHash(cdb.db, fullHash)
	return txid, nil
}

func (cdb *ChainDB) DeleteTxHashs(block *types.SerializedBlock) error {
	for _, tx := range block.Transactions() {
		fhash := tx.Tx.TxHashFull()
		err := rawdb.DeleteTxIdByFullHash(cdb.db, &fhash)
		if err != nil {
			return err
		}
	}
	return nil
}
