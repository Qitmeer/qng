package chaindb

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/rawdb"
	"github.com/Qitmeer/qng/meerdag"
	"math"
)

func (cdb *ChainDB) PutTxIdxEntrys(sblock *types.SerializedBlock, block model.Block) error {
	return rawdb.WriteTxLookupEntriesByBlock(cdb.db, sblock, uint64(block.GetID()))
}

func (cdb *ChainDB) GetTxIdxEntry(id *hash.Hash, verbose bool) (*types.Tx, *hash.Hash, error) {
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

func (cdb *ChainDB) DeleteTxIdxEntrys(block *types.SerializedBlock) error {
	for _, tx := range block.Transactions() {
		_, blockHash, _ := cdb.GetTxIdxEntry(tx.Hash(), false)
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

func (cdb *ChainDB) IsInvalidTxIdxEmpty() bool {
	return rawdb.IsInvalidTxEmpty(cdb.db)
}

func (cdb *ChainDB) GetInvalidTxIdxTip() (uint64, *hash.Hash, error) {
	return 0, nil, nil
}

func (cdb *ChainDB) PutInvalidTxIdxTip(order uint64, bh *hash.Hash) error {
	return nil
}

func (cdb *ChainDB) PutInvalidTxs(sblock *types.SerializedBlock, block model.Block) error {
	return rawdb.WriteInvalidTxLookupEntriesByBlock(cdb.db, sblock, uint64(block.GetID()))
}

func (cdb *ChainDB) DeleteInvalidTxs(sblock *types.SerializedBlock, block model.Block) error {
	for _, tx := range sblock.Transactions() {
		err := rawdb.DeleteTxLookupEntry(cdb.db, tx.Hash())
		if err != nil {
			return err
		}
	}
	return nil
}

func (cdb *ChainDB) GetInvalidTx(id *hash.Hash) (*types.Transaction, error) {
	tx, _, _, _ := rawdb.ReadInvalidTransaction(cdb.db, id)
	return tx.Tx, nil
}

func (cdb *ChainDB) GetInvalidTxIdByHash(fullHash *hash.Hash) (*hash.Hash, error) {
	txid := rawdb.ReadInvalidTxIdByFullHash(cdb.db, fullHash)
	return txid, nil
}

func (cdb *ChainDB) CleanInvalidTxIdx() error {
	err := rawdb.CleanInvalidTxs(cdb.db)
	if err != nil {
		return err
	}
	return rawdb.CleanInvalidTxHashs(cdb.db)
}

func (cdb *ChainDB) GetAddrIdxTip() (*hash.Hash, uint, error) {
	return nil, math.MaxUint32, nil
}

func (cdb *ChainDB) PutAddrIdxTip(bh *hash.Hash, order uint) error {
	return nil
}

func (cdb *ChainDB) PutAddrIdx(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	return nil
}

func (cdb *ChainDB) GetTxForAddress(addr types.Address, numToSkip, numRequested uint32, reverse bool) ([]*common.RetrievedTx, uint32, error) {
	return nil, 0, nil
}

func (cdb *ChainDB) DeleteAddrIdx(sblock *types.SerializedBlock, stxos [][]byte) error {
	return nil
}

func (cdb *ChainDB) CleanAddrIdx(finish bool) error {
	return nil
}
