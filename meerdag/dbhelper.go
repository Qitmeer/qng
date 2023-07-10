package meerdag

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/database/legacydb"
	"math"
)

const (
	DAGErrorEmpty = "empty"
)

type DAGError struct {
	err string
}

func (e *DAGError) Error() string {
	return e.err
}

func (e *DAGError) IsEmpty() bool {
	return e.Error() == DAGErrorEmpty
}

func NewDAGError(e error) error {
	if e == nil {
		return nil
	}
	return &DAGError{e.Error()}
}

// DBPutDAGBlock stores the information needed to reconstruct the provided
// block in the block index according to the format described above.
func DBPutDAGBlock(dbTx legacydb.Tx, block IBlock) error {
	bucket := dbTx.Metadata().Bucket(BlockIndexBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(block.GetID()))

	key := serializedID[:]

	var buff bytes.Buffer
	err := block.Encode(&buff)
	if err != nil {
		return err
	}
	return bucket.Put(key, buff.Bytes())
}

// DBGetDAGBlock get dag block data by resouce ID
func DBGetDAGBlock(dbTx legacydb.Tx, block IBlock) error {
	bucket := dbTx.Metadata().Bucket(BlockIndexBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(block.GetID()))

	data := bucket.Get(serializedID[:])
	if data == nil {
		return &DAGError{DAGErrorEmpty}
	}
	return NewDAGError(block.Decode(bytes.NewReader(data)))
}

func DBDelDAGBlock(dbTx legacydb.Tx, id uint) error {
	bucket := dbTx.Metadata().Bucket(BlockIndexBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))
	return bucket.Delete(serializedID[:])
}

func DBGetDAGBlockHashByID(dbTx legacydb.Tx, id uint64) (*hash.Hash, error) {
	bucket := dbTx.Metadata().Bucket(BlockIndexBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))

	data := bucket.Get(serializedID[:])
	if data == nil {
		return nil, nil
	}
	if len(data) < 4+hash.HashSize {
		return nil, fmt.Errorf("block(%d) data error", id)
	}
	return hash.NewHash(data[4 : hash.HashSize+4])
}

func GetOrderLogStr(order uint) string {
	if order == MaxBlockOrder {
		return "uncertainty"
	}
	return fmt.Sprintf("%d", order)
}

func DBPutDAGInfo(dbTx legacydb.Tx, bd *MeerDAG) error {
	var buff bytes.Buffer
	err := bd.Encode(&buff)
	if err != nil {
		return err
	}
	return dbTx.Metadata().Put(DagInfoBucketName, buff.Bytes())
}

func DBHasMainChainBlock(dbTx legacydb.Tx, id uint) bool {
	bucket := dbTx.Metadata().Bucket(DagMainChainBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))

	data := bucket.Get(serializedID[:])
	return data != nil
}

func DBPutMainChainBlock(dbTx legacydb.Tx, id uint) error {
	bucket := dbTx.Metadata().Bucket(DagMainChainBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))

	key := serializedID[:]
	return bucket.Put(key, []byte{0})
}

func DBRemoveMainChainBlock(dbTx legacydb.Tx, id uint) error {
	bucket := dbTx.Metadata().Bucket(DagMainChainBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))

	key := serializedID[:]
	return bucket.Delete(key)
}

// block order

func DBPutBlockIdByOrder(dbTx legacydb.Tx, order uint, id uint) error {
	// Serialize the order for use in the index entries.
	var serializedOrder [4]byte
	ByteOrder.PutUint32(serializedOrder[:], uint32(order))

	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))

	// Add the block order to id mapping to the index.
	bucket := dbTx.Metadata().Bucket(OrderIdBucketName)
	return bucket.Put(serializedOrder[:], serializedID[:])
}

func DBGetBlockIdByOrder(dbTx legacydb.Tx, order uint) (uint32, error) {
	if order > math.MaxUint32 {
		str := fmt.Sprintf("order %d is overflow", order)
		return uint32(MaxId), &DAGError{str}
	}
	var serializedOrder [4]byte
	ByteOrder.PutUint32(serializedOrder[:], uint32(order))

	bucket := dbTx.Metadata().Bucket(OrderIdBucketName)
	idBytes := bucket.Get(serializedOrder[:])
	if idBytes == nil {
		str := fmt.Sprintf("no block at order %d exists", order)
		return uint32(MaxId), &DAGError{str}
	}
	return ByteOrder.Uint32(idBytes), nil
}

func DBPutDAGBlockIdByHash(dbTx legacydb.Tx, block IBlock) error {
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(block.GetID()))

	bucket := dbTx.Metadata().Bucket(BlockIdBucketName)
	key := block.GetHash()[:]
	return bucket.Put(key, serializedID[:])
}

func DBGetBlockIdByHash(dbTx legacydb.Tx, h *hash.Hash) (uint32, error) {
	bucket := dbTx.Metadata().Bucket(BlockIdBucketName)
	data := bucket.Get(h[:])
	if data == nil {
		return uint32(MaxId), fmt.Errorf("get dag block error")
	}
	return ByteOrder.Uint32(data), nil
}

func DBDelBlockIdByHash(dbTx legacydb.Tx, h *hash.Hash) error {
	bucket := dbTx.Metadata().Bucket(BlockIdBucketName)
	return bucket.Delete(h[:])
}

// tips
func DBPutDAGTip(dbTx legacydb.Tx, id uint, isMain bool) error {
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))

	bucket := dbTx.Metadata().Bucket(DAGTipsBucketName)
	main := byte(0)
	if isMain {
		main = byte(1)
	}
	return bucket.Put(serializedID[:], []byte{main})
}

func DBGetDAGTips(dbTx legacydb.Tx) ([]uint, error) {
	bucket := dbTx.Metadata().Bucket(DAGTipsBucketName)
	cursor := bucket.Cursor()
	mainTip := MaxId
	tips := []uint{}
	for cok := cursor.First(); cok; cok = cursor.Next() {
		id := uint(ByteOrder.Uint32(cursor.Key()))
		main := cursor.Value()
		if len(main) > 0 {
			if main[0] > 0 {
				if mainTip != MaxId {
					return nil, fmt.Errorf("Too many main tip:cur(%d) => next(%d)", mainTip, id)
				}
				mainTip = id
				continue
			}
		}
		tips = append(tips, id)
	}
	if mainTip == MaxId {
		return nil, fmt.Errorf("Can't find main tip")
	}
	result := []uint{mainTip}
	if len(tips) > 0 {
		result = append(result, tips...)
	}
	return result, nil
}

func DBDelDAGTip(dbTx legacydb.Tx, id uint) error {
	bucket := dbTx.Metadata().Bucket(DAGTipsBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))
	return bucket.Delete(serializedID[:])
}

// diffAnticone
func DBPutDiffAnticone(dbTx legacydb.Tx, id uint) error {
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))
	bucket := dbTx.Metadata().Bucket(DiffAnticoneBucketName)
	return bucket.Put(serializedID[:], []byte{byte(0)})
}

func DBGetDiffAnticone(dbTx legacydb.Tx) ([]uint, error) {
	bucket := dbTx.Metadata().Bucket(DiffAnticoneBucketName)
	cursor := bucket.Cursor()
	diffs := []uint{}
	for cok := cursor.First(); cok; cok = cursor.Next() {
		id := uint(ByteOrder.Uint32(cursor.Key()))
		diffs = append(diffs, id)
	}
	return diffs, nil
}

func DBDelDiffAnticone(dbTx legacydb.Tx, id uint) error {
	bucket := dbTx.Metadata().Bucket(DiffAnticoneBucketName)
	var serializedID [4]byte
	ByteOrder.PutUint32(serializedID[:], uint32(id))
	return bucket.Delete(serializedID[:])
}
