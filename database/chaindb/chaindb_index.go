package chaindb

import (
	"encoding/binary"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/rawdb"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/index"
	"github.com/ethereum/go-ethereum/ethdb"
	"time"
)

func (cdb *ChainDB) PutTxIdxEntrys(sblock *types.SerializedBlock, block model.Block) error {
	if cdb.diff != nil {
		return cdb.diff.PutTxIdxEntrys(sblock, block)
	}
	return rawdb.WriteTxLookupEntriesByBlock(cdb.db, sblock, uint64(block.GetID()))
}

func (cdb *ChainDB) GetTxIdxEntry(id *hash.Hash, verbose bool) (*types.Tx, *hash.Hash, error) {
	if cdb.diff != nil {
		return cdb.diff.GetTxIdxEntry(id, verbose)
	}
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
	if cdb.diff != nil {
		return cdb.diff.DeleteTxIdxEntrys(block)
	}
	batch := cdb.db.NewBatch()
	for _, tx := range block.Transactions() {
		_, blockHash, _ := cdb.GetTxIdxEntry(tx.Hash(), false)
		if blockHash != nil && !blockHash.IsEqual(block.Hash()) {
			continue
		}
		err := rawdb.DeleteTxLookupEntry(batch, tx.Hash())
		if err != nil {
			return err
		}
	}
	return batch.Write()
}

func (cdb *ChainDB) PutTxHashs(block *types.SerializedBlock) error {
	batch := cdb.db.NewBatch()
	for _, tx := range block.Transactions() {
		if tx.IsDuplicate {
			continue
		}
		fhash := tx.Tx.TxHashFull()
		err := rawdb.WriteTxIdByFullHash(batch, &fhash, tx.Hash())
		if err != nil {
			return err
		}
	}
	return batch.Write()
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
	batch := cdb.db.NewBatch()
	err := rawdb.WriteInvalidTxLookupEntriesByBlock(cdb.db, sblock, uint64(block.GetID()))
	if err != nil {
		return err
	}
	return batch.Write()
}

func (cdb *ChainDB) DeleteInvalidTxs(sblock *types.SerializedBlock, block model.Block) error {
	batch := cdb.db.NewBatch()
	for _, tx := range sblock.Transactions() {
		err := rawdb.DeleteTxLookupEntry(batch, tx.Hash())
		if err != nil {
			return err
		}
	}
	return batch.Write()
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
	return rawdb.ReadAddrIdxTip(cdb.db)
}

func (cdb *ChainDB) PutAddrIdxTip(bh *hash.Hash, order uint) error {
	return rawdb.WriteAddrIdxTip(cdb.db, bh, order)
}

func (cdb *ChainDB) PutAddrIdx(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	// Build all of the address to transaction mappings in a local map.
	addrsToTxns := make(index.WriteAddrIdxData)
	index.AddrIndexBlock(addrsToTxns, sblock, stxos)
	b := &bucket{db: cdb.db}
	// Add all of the index entries for each address.
	for addrKey, txIdxs := range addrsToTxns {
		for _, txIdx := range txIdxs {
			// Switch to using the newest block ID for the stake transactions,
			// since these are not from the parent. Offset the index to be
			// correct for the location in this given block.
			err := index.DBPutAddrIndexEntry(b, addrKey,
				uint32(block.GetID()), types.TxLoc{TxStart: txIdx, TxLen: 0}, rawdb.AddridxPrefix)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cdb *ChainDB) GetTxForAddress(addr types.Address, numToSkip, numRequested uint32, reverse bool) ([]*common.RetrievedTx, uint32, error) {
	addrKey, err := index.AddrToKey(addr)
	if err != nil {
		return nil, 0, err
	}
	fetchBlockHash := func(id []byte) (*hash.Hash, error) {
		// Deserialize and populate the result.
		blockid := uint64(binary.LittleEndian.Uint32(id))
		return meerdag.DBGetDAGBlockHashByID(cdb, blockid)
	}
	fetchAddrLevelData := func(key []byte) []byte {
		levelData, _ := cdb.db.Get(key)
		return levelData
	}
	regions, dbSkipped, err := index.DBFetchAddrIndexEntries(fetchBlockHash, fetchAddrLevelData,
		addrKey, numToSkip, numRequested, reverse, rawdb.AddridxPrefix)
	if err != nil {
		return nil, 0, err
	}
	addressTxns := []*common.RetrievedTx{}
	for _, r := range regions {
		block, err := cdb.GetBlock(r.Hash)
		if err != nil {
			return nil, 0, err
		}
		addressTxns = append(addressTxns, &common.RetrievedTx{
			BlkHash: r.Hash,
			Tx:      block.Transactions()[r.Offset],
		})
	}
	return addressTxns, dbSkipped, nil
}

func (cdb *ChainDB) DeleteAddrIdx(sblock *types.SerializedBlock, stxos [][]byte) error {
	// Build all of the address to transaction mappings in a local map.
	addrsToTxns := make(index.WriteAddrIdxData)
	index.AddrIndexBlock(addrsToTxns, sblock, stxos)
	b := &bucket{db: cdb.db}

	for addrKey, txIdxs := range addrsToTxns {
		err := index.DBRemoveAddrIndexEntries(b, addrKey, len(txIdxs), rawdb.AddridxPrefix)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cdb *ChainDB) CleanAddrIdx(finish bool) error {
	if finish {
		return nil
	}
	return rawdb.CleanAddrIdx(cdb.DB())
}

func (cdb *ChainDB) RebuildAddrIndex(interrupt <-chan struct{}) (bool, error) {
	start := time.Now()
	processLog := func(task string, cur int, max int) {
		if time.Since(start) < time.Second*10 {
			return
		}
		log.Info(task, "cur", cur, "max", max)
		start = time.Now()
	}
	ops := []*types.TxOutPoint{}
	entrys := []*utxo.UtxoEntry{}
	err := rawdb.ForeachUtxo(cdb.DB(), func(key []byte, data []byte) error {
		op, err := common.ParseOutpoint(key)
		if err != nil {
			return err
		}
		serializedUtxo := data
		// Deserialize the utxo entry and return it.
		entry, err := utxo.DeserializeUtxoEntry(serializedUtxo)
		if err != nil {
			return err
		}
		if entry.IsSpent() {
			return nil
		}
		ops = append(ops, op)
		entrys = append(entrys, entry)
		processLog("Find utxo", len(ops)-1, len(ops))
		return nil
	})
	if err != nil {
		return true, err
	}
	if system.InterruptRequested(interrupt) {
		return true, fmt.Errorf("RebuildAddrIndex:interrupt")
	}
	b := &bucket{db: rawdb.NewMemoryDatabase()}
	if len(ops) > 0 {
		for i := 0; i < len(ops); i++ {
			_, addrs, _, err := txscript.ExtractPkScriptAddrs(entrys[i].PkScript(), params.ActiveNetParams.Params)
			if err != nil {
				return true, err
			}

			if len(addrs) == 0 {
				continue
			}
			tx, blockid, _, txIdx := rawdb.ReadTransaction(cdb.db, &ops[i].Hash)
			if tx == nil {
				continue
			}
			for _, addr := range addrs {
				addrKey, err := index.AddrToKey(addr)
				if err != nil {
					continue
				}
				err = index.DBPutAddrIndexEntry(b, addrKey, uint32(blockid), types.TxLoc{TxStart: txIdx, TxLen: 0}, rawdb.AddridxPrefix)
				if err != nil {
					return true, err
				}
			}
			processLog("Index addr for utxo", i, len(ops))
		}
	}
	if system.InterruptRequested(interrupt) {
		return true, fmt.Errorf("RebuildAddrIndex:interrupt")
	}
	stxos := []utxo.SpentTxOut{}
	err = rawdb.ForeachSpendJournal(cdb.DB(), func(key []byte, data []byte) error {
		sts, err := utxo.DeserializeSpendJournalEntry(data)
		if err != nil {
			return err
		}
		stxos = append(stxos, sts...)
		processLog("Find spend journal", len(sts), len(stxos))
		return nil
	})
	if err != nil {
		return true, err
	}
	if len(stxos) <= 0 {
		return true, nil
	}
	if system.InterruptRequested(interrupt) {
		return true, fmt.Errorf("RebuildAddrIndex:interrupt")
	}
	for i := 0; i < len(stxos); i++ {
		_, addrs, _, err := txscript.ExtractPkScriptAddrs(stxos[i].PkScript, params.ActiveNetParams.Params)
		if err != nil {
			return true, err
		}

		if len(addrs) == 0 {
			continue
		}

		blockID := rawdb.ReadBlockID(cdb.db, &stxos[i].BlockHash)
		if blockID == nil {
			continue
		}
		for _, addr := range addrs {
			addrKey, err := index.AddrToKey(addr)
			if err != nil {
				continue
			}
			err = index.DBPutAddrIndexEntry(b, addrKey, uint32(*blockID), types.TxLoc{TxStart: int(stxos[i].TxIndex), TxLen: 0}, rawdb.AddridxPrefix)
			if err != nil {
				return true, err
			}
		}
		processLog("Index addr for spend journal", i, len(stxos))
	}
	if system.InterruptRequested(interrupt) {
		return true, fmt.Errorf("RebuildAddrIndex:interrupt")
	}
	batch := cdb.db.NewBatch()
	it := b.db.NewIterator(nil, nil)
	total := 0
	for it.Next() {
		batch.Put(it.Key(), it.Value())
		total++
	}
	log.Info("Write batch", "objects", total, "size", batch.ValueSize())
	return true, batch.Write()
}

type bucket struct {
	db ethdb.Database
}

func (b *bucket) Get(key []byte) []byte {
	value, _ := b.db.Get(key)
	return value
}

func (b *bucket) Put(key []byte, value []byte) error {
	return b.db.Put(key, value)
}

func (b *bucket) Delete(key []byte) error {
	return b.db.Delete(key)
}
