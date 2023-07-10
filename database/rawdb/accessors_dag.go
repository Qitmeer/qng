package rawdb

import (
	"bytes"
	"encoding/binary"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/ethereum/go-ethereum/ethdb"
)

func ReadDAGBlockBaw(db ethdb.Reader, id uint64) []byte {
	var data []byte
	data, _ = db.Get(dagBlockKey(id))
	if len(data) > 0 {
		return data
	}

	db.ReadAncients(func(reader ethdb.AncientReaderOp) error {
		data, _ = reader.Ancient(ChainFreezerDAGBlockTable, id)
		return nil
	})
	return data
}

func ReadDAGBlock(db ethdb.Reader, id uint64) meerdag.IBlock {
	data := ReadDAGBlockBaw(db, id)
	if len(data) == 0 {
		return nil
	}
	var block meerdag.IBlock
	err := block.Decode(bytes.NewReader(data))
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return block
}

func WriteDAGBlock(db ethdb.KeyValueWriter, block meerdag.IBlock) error {
	err := db.Put(dagBlockKey(uint64(block.GetID())), block.Bytes())
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func DeleteDAGBlock(db ethdb.KeyValueWriter, id uint64) {
	err := db.Delete(dagBlockKey(id))
	if err != nil {
		log.Error(err.Error())
	}
}

func ReadBlockID(db ethdb.Reader, hash *hash.Hash) *uint64 {
	data, err := db.Get(blockIDKey(hash))
	if err != nil {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

func WriteBlockID(db ethdb.KeyValueWriter, hash *hash.Hash, id uint64) {
	if err := db.Put(blockIDKey(hash), hash.Bytes()); err != nil {
		log.Error("Failed to store block id to hash mapping", "err", err)
	}
}

func DeleteBlockID(db ethdb.KeyValueWriter, hash *hash.Hash) {
	if err := db.Delete(blockIDKey(hash)); err != nil {
		log.Error("Failed to delete block id to hash mapping", "err", err)
	}
}

// ----

func ReadMainChainTip(db ethdb.Reader) *uint64 {
	data, err := db.Get(mainchainTipKey)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

func WriteMainChainTip(db ethdb.KeyValueWriter, mainchaintip uint64) error {
	err := db.Put(mainchainTipKey, encodeBlockID(mainchaintip))
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func DeleteMainChainTip(db ethdb.KeyValueWriter) error {
	err := db.Delete(mainchainTipKey)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}
