package rawdb

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/ethereum/go-ethereum/ethdb"
)

func ReadTxLookupEntry(db ethdb.Reader, hash *hash.Hash) ([]byte, error) {
	return db.Get(txLookupKey(hash))
}

func WriteTxLookupEntry(db ethdb.KeyValueWriter, hash *hash.Hash, bytes []byte) error {
	return db.Put(txLookupKey(hash), bytes)
}

func DeleteTxLookupEntry(db ethdb.KeyValueWriter, hash *hash.Hash) error {
	return db.Delete(txLookupKey(hash))
}
