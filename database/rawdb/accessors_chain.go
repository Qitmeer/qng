package rawdb

import (
	"bytes"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/Qitmeer/qng/core/types"
)

func ReadBodyRaw(db ethdb.Reader, hash *hash.Hash) []byte {
	var data []byte
	data, _ = db.Get(blockKey(hash))
	if len(data) > 0 {
		return data
	}
	blockID := ReadBlockID(db, hash)
	if blockID == nil {
		return nil
	}

	db.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		data, _ = reader.Ancient(ChainFreezerBlockTable, *blockID)
		return nil
	})
	return data
}

func ReadBody(db ethdb.Reader, hash *hash.Hash) *types.SerializedBlock {
	data := ReadBodyRaw(db, hash)
	if len(data) == 0 {
		return nil
	}
	block, err := types.NewBlockFromBytes(data)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return block
}

func ReadBodyRawByID(db ethdb.Reader, id uint64) []byte {
	var data []byte

	db.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		data, _ = reader.Ancient(ChainFreezerBlockTable, id)
		return nil
	})
	if len(data) > 0 {
		return data
	}
	bh, err := ReadBlockHashByID(db, id)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	if bh == nil {
		return nil
	}

	data, _ = db.Get(blockKey(bh))
	return data
}

func ReadBodyByID(db ethdb.Reader, id uint64) *types.SerializedBlock {
	data := ReadBodyRawByID(db, id)
	if len(data) == 0 {
		return nil
	}
	block, err := types.NewBlockFromBytes(data)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return block
}

func WriteBody(db ethdb.KeyValueWriter, block *types.SerializedBlock) error {
	data, err := block.Bytes()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	key := blockKey(block.Hash())
	err = db.Put(key, data)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func DeleteBody(db ethdb.KeyValueWriter, hash *hash.Hash) {
	if err := db.Delete(blockKey(hash)); err != nil {
		log.Crit("Failed to delete hash to block mapping", "err", err)
	}
}

func HasBody(db ethdb.Reader, hash *hash.Hash) bool {
	if has, err := db.Has(blockKey(hash)); !has || err != nil {
		return false
	}
	blockID := ReadBlockID(db, hash)
	return blockID != nil
}

func WriteBlock(db ethdb.KeyValueWriter, block *types.SerializedBlock) error {
	err := WriteHeader(db, &block.Block().Header)
	if err != nil {
		return err
	}
	return WriteBody(db, block)
}

func WriteAncientBlocks(db ethdb.AncientWriter, blocks []*types.SerializedBlock, dagblocks []meerdag.IBlock) (int64, error) {
	return db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i, block := range blocks {
			if err := writeAncientBlock(op, block, dagblocks[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func writeAncientBlock(op ethdb.AncientWriteOp, block *types.SerializedBlock, dagblock meerdag.IBlock) error {
	data, err := block.Bytes()
	if err != nil {
		return err
	}
	err = op.AppendRaw(ChainFreezerBlockTable, uint64(dagblock.GetID()), data)
	if err != nil {
		return err
	}
	err = op.AppendRaw(ChainFreezerDAGBlockTable, uint64(dagblock.GetID()), dagblock.Bytes())
	if err != nil {
		return err
	}
	var headerBuf bytes.Buffer
	err = block.Block().Header.Serialize(&headerBuf)
	if err != nil {
		return err
	}
	data = headerBuf.Bytes()
	return op.AppendRaw(ChainFreezerHeaderTable, uint64(dagblock.GetID()), data)
}

// header
func ReadHeaderRaw(db ethdb.Reader, hash *hash.Hash) []byte {
	var data []byte
	data, _ = db.Get(headerKey(hash))
	if len(data) > 0 {
		return data
	}
	blockID := ReadBlockID(db, hash)
	if blockID == nil {
		return nil
	}
	db.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		data, _ = reader.Ancient(ChainFreezerHeaderTable, *blockID)
		return nil
	})
	return data
}

func HasHeader(db ethdb.Reader, hash *hash.Hash) bool {
	if has, err := db.Has(headerKey(hash)); !has || err != nil {
		return false
	}
	blockID := ReadBlockID(db, hash)
	return blockID != nil
}

func ReadHeader(db ethdb.Reader, hash *hash.Hash) *types.BlockHeader {
	data := ReadHeaderRaw(db, hash)
	if len(data) == 0 {
		return nil
	}
	var header types.BlockHeader
	err := header.Deserialize(bytes.NewReader(data))
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return &header
}

func WriteHeader(db ethdb.KeyValueWriter, header *types.BlockHeader) error {
	var headerBuf bytes.Buffer
	err := header.Serialize(&headerBuf)
	if err != nil {
		return err
	}
	data := headerBuf.Bytes()
	h := header.BlockHash()
	key := headerKey(&h)
	err = db.Put(key, data)
	if err != nil {
		return err
	}
	return nil
}

func DeleteHeader(db ethdb.KeyValueWriter, hash *hash.Hash) {
	if err := db.Delete(headerKey(hash)); err != nil {
		log.Crit("Failed to delete hash to header mapping", "err", err)
	}
}

func DeleteBlock(db ethdb.KeyValueWriter, hash *hash.Hash) {
	DeleteHeader(db, hash)
	DeleteBody(db, hash)
}

// stxo

func ReadSpendJournal(db ethdb.Reader, hash *hash.Hash) []byte {
	data, err := db.Get(spendJournalKey(hash))
	if len(data) == 0 {
		if err != nil {
			log.Debug("spend journal", "err", err.Error())
		}
		return nil
	}
	return data
}

func WriteSpendJournal(db ethdb.KeyValueWriter, hash *hash.Hash, data []byte) error {
	if len(data) <= 0 {
		return nil
	}
	return db.Put(spendJournalKey(hash), data)
}

func DeleteSpendJournal(db ethdb.KeyValueWriter, hash *hash.Hash) {
	if err := db.Delete(spendJournalKey(hash)); err != nil {
		log.Crit("Failed to delete hash to Spend Journal mapping", "err", err)
	}
}

func CleanSpendJournal(db ethdb.Database) error {
	it := db.NewIterator(spendJournalPrefix, nil)
	total := 0
	defer func() {
		log.Debug("Clean spend journal", "total", total)
	}()
	for it.Next() {
		err := db.Delete(it.Key())
		if err != nil {
			return err
		}
		total++
	}
	return nil
}

// utxo

func ReadUtxo(db ethdb.Reader, opd []byte) []byte {
	data, err := db.Get(utxoKey(opd))
	if len(data) == 0 {
		log.Debug("read utxo", "err", err.Error())
		return nil
	}
	return data
}

func WriteUtxo(db ethdb.KeyValueWriter, opd []byte, data []byte) error {
	if len(data) <= 0 {
		return nil
	}
	return db.Put(utxoKey(opd), data)
}

func DeleteUtxo(db ethdb.KeyValueWriter, opd []byte) {
	if err := db.Delete(utxoKey(opd)); err != nil {
		log.Crit("Failed to delete hash to utxo mapping", "err", err)
	}
}

func ForeachUtxo(db ethdb.KeyValueStore, fn func(opd []byte, data []byte) error) error {
	it := db.NewIterator(utxoPrefix, nil)
	defer it.Release()

	for it.Next() {
		err := fn(it.Key(), it.Value())
		if err != nil {
			return err
		}
	}
	return nil
}

func CleanUtxo(db ethdb.Database) error {
	it := db.NewIterator(utxoPrefix, nil)
	total := 0
	defer func() {
		log.Debug("Clean utxo", "total", total)
	}()
	for it.Next() {
		err := db.Delete(it.Key())
		if err != nil {
			return err
		}
		total++
	}
	return nil
}

// tokenState

func ReadTokenState(db ethdb.Reader, id uint64) []byte {
	data, err := db.Get(tokenStateKey(id))
	if len(data) == 0 {
		log.Debug("read token state", "err", err.Error())
		return nil
	}
	return data
}

func WriteTokenState(db ethdb.KeyValueWriter, id uint64, data []byte) error {
	if len(data) <= 0 {
		return nil
	}
	return db.Put(tokenStateKey(id), data)
}

func DeleteTokenState(db ethdb.KeyValueWriter, id uint64) {
	if err := db.Delete(tokenStateKey(id)); err != nil {
		log.Crit("Failed to delete hash to token state mapping", "err", err)
	}
}

func CleanTokenState(db ethdb.Database) error {
	it := db.NewIterator(tokenStatePrefix, nil)
	total := 0
	defer func() {
		log.Debug("Clean token state", "total", total)
	}()
	for it.Next() {
		err := db.Delete(it.Key())
		if err != nil {
			return err
		}
		total++
	}
	return nil
}
