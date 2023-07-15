package rawdb

import (
	"encoding/binary"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

func ReadTxLookupEntry(db ethdb.Reader, hash *hash.Hash) *uint64 {
	data, err := db.Get(txLookupKey(hash))
	if len(data) == 0 {
		log.Error(err.Error())
		return nil
	}
	id := binary.BigEndian.Uint64(data)
	return &id
}

func writeTxLookupEntry(db ethdb.KeyValueWriter, hash *hash.Hash, id uint64) error {
	var serializedID [4]byte
	binary.BigEndian.PutUint64(serializedID[:], id)
	return db.Put(txLookupKey(hash), serializedID[:])
}

func WriteTxLookupEntriesByBlock(db ethdb.KeyValueWriter, block *types.SerializedBlock, id uint64) {
	var serializedID [4]byte
	binary.BigEndian.PutUint64(serializedID[:], id)
	for _, tx := range block.Transactions() {
		db.Put(txLookupKey(tx.Hash()), serializedID[:])
	}
}

func DeleteTxLookupEntry(db ethdb.KeyValueWriter, hash *hash.Hash) error {
	return db.Delete(txLookupKey(hash))
}

func ReadTransaction(db ethdb.Reader, hash *hash.Hash) (*types.Tx, uint64, *hash.Hash, int) {
	blockID := ReadTxLookupEntry(db, hash)
	if blockID == nil {
		return nil, 0, nil, 0
	}
	body := ReadBodyByID(db, *blockID)
	if body == nil {
		log.Error("Transaction referenced missing", "blockID", *blockID, "txHash", hash.String())
		return nil, 0, nil, 0
	}
	for txIndex, tx := range body.Transactions() {
		if tx.Hash().IsEqual(hash) {
			return tx, *blockID, body.Hash(), txIndex
		}
	}
	log.Error("Transaction not found", "blockID", *blockID, "txHash", hash.String())
	return nil, 0, nil, 0
}
