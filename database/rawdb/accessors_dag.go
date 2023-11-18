package rawdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
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
	block := &meerdag.Block{}
	block.SetID(uint(id))
	ib := &meerdag.PhantomBlock{Block: block}
	err := ib.Decode(bytes.NewReader(data))
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return ib
}

func WriteDAGBlockRaw(db ethdb.KeyValueWriter, id uint, data []byte) error {
	err := db.Put(dagBlockKey(uint64(id)), data)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}

func WriteDAGBlock(db ethdb.KeyValueWriter, block meerdag.IBlock) error {
	return WriteDAGBlockRaw(db, block.GetID(), block.Bytes())
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
	var serializedID [8]byte
	binary.BigEndian.PutUint64(serializedID[:], id)
	if err := db.Put(blockIDKey(hash), serializedID[:]); err != nil {
		log.Error("Failed to store block id to hash mapping", "err", err)
	}
}

func DeleteBlockID(db ethdb.KeyValueWriter, hash *hash.Hash) {
	if err := db.Delete(blockIDKey(hash)); err != nil {
		log.Error("Failed to delete block id to hash mapping", "err", err)
	}
}

func ReadBlockHashByID(db ethdb.Reader, id uint64) (*hash.Hash, error) {
	data := ReadDAGBlockBaw(db, id)
	if len(data) == 0 {
		return nil, nil
	}
	if len(data) < 4+hash.HashSize {
		return nil, fmt.Errorf("block(%d) data error", id)
	}
	return hash.NewHash(data[4 : hash.HashSize+4])
}

// main chain

func ReadMainChainTip(db ethdb.Reader) *uint64 {
	tips := ReadDAGTips(db)
	if len(tips) <= 0 {
		return nil
	}
	if tips[0] == uint64(meerdag.MaxId) {
		return nil
	}
	return &tips[0]
}

func ReadMainChain(db ethdb.Reader, id uint64) bool {
	data, _ := db.Get(dagMainChainKey(id))
	if len(data) <= 0 {
		return false
	}
	return true
}

func WriteMainChain(db ethdb.KeyValueWriter, id uint64) error {
	return db.Put(dagMainChainKey(id), []byte{0})
}

func DeleteMainChain(db ethdb.KeyValueWriter, id uint64) {
	if err := db.Delete(dagMainChainKey(id)); err != nil {
		log.Crit("Failed to delete id mapping", "err", err)
	}
}

// dag info
func ReadDAGInfo(db ethdb.Reader) []byte {
	data, err := db.Get(dagInfoKey)
	if len(data) == 0 {
		if err != nil {
			log.Debug("dag info", "err", err.Error())
		}
		return nil
	}
	return data
}

func WriteDAGInfo(db ethdb.KeyValueWriter, data []byte) error {
	if len(data) <= 0 {
		return nil
	}
	return db.Put(dagInfoKey, data)
}

// dag tips
func ReadDAGTips(db ethdb.Reader) []uint64 {
	data, err := db.Get(dagTipsKey)
	if len(data) == 0 {
		log.Debug("dag tips", "err", err.Error())
		return nil
	}
	var tips []uint64
	err = rlp.Decode(bytes.NewReader(data), &tips)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return tips
}

func WriteDAGTips(db ethdb.KeyValueWriter, tips []uint64) error {
	if len(tips) <= 0 {
		return nil
	}
	bs, err := rlp.EncodeToBytes(tips)
	if err != nil {
		return err
	}
	return db.Put(dagTipsKey, bs)
}

// dag diff anticone
func ReadDiffAnticone(db ethdb.Reader) []uint64 {
	data, err := db.Get(diffAnticoneKey)
	if len(data) == 0 {
		return nil
	}
	var tips []uint64
	err = rlp.Decode(bytes.NewReader(data), &tips)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return tips
}

func WriteDiffAnticone(db ethdb.KeyValueWriter, da []uint64) error {
	if len(da) <= 0 {
		return db.Put(diffAnticoneKey, []byte{})
	}
	bs, err := rlp.EncodeToBytes(da)
	if err != nil {
		return err
	}
	return db.Put(diffAnticoneKey, bs)
}
