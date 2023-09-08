package meerdag

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
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

func NewDAGErrorByStr(e string) error {
	return &DAGError{e}
}

// DBPutDAGBlock stores the information needed to reconstruct the provided
// block in the block index according to the format described above.
func DBPutDAGBlock(db model.DataBase, block IBlock) error {
	var buff bytes.Buffer
	err := block.Encode(&buff)
	if err != nil {
		return err
	}
	return db.PutDAGBlock(block.GetID(), buff.Bytes())
}

// DBGetDAGBlock get dag block data by resouce ID
func DBGetDAGBlock(db model.DataBase, block IBlock) error {
	data, _ := db.GetDAGBlock(block.GetID())
	if data == nil {
		return &DAGError{DAGErrorEmpty}
	}
	return NewDAGError(block.Decode(bytes.NewReader(data)))
}

func DBDelDAGBlock(db model.DataBase, id uint) error {
	return db.DeleteDAGBlock(id)
}

func DBGetDAGBlockHashByID(db model.DataBase, id uint64) (*hash.Hash, error) {
	data, _ := db.GetDAGBlock(uint(id))
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

func DBPutDAGInfo(bd *MeerDAG) error {
	var buff bytes.Buffer
	err := bd.Encode(&buff)
	if err != nil {
		return err
	}
	return bd.db.PutDagInfo(buff.Bytes())
}

func DBHasMainChainBlock(db model.DataBase, id uint) bool {
	return db.HasMainChainBlock(id)
}

func DBPutMainChainBlock(db model.DataBase, id uint) error {
	return db.PutMainChainBlock(id)
}

func DBRemoveMainChainBlock(db model.DataBase, id uint) error {
	return db.DeleteMainChainBlock(id)
}

// block order

func DBPutBlockIdByOrder(db model.DataBase, order uint, id uint) error {
	return db.PutBlockIdByOrder(order, id)
}

func DBGetBlockIdByOrder(db model.DataBase, order uint) (uint, error) {
	return db.GetBlockIdByOrder(order)
}

func DBPutDAGBlockIdByHash(db model.DataBase, block IBlock) error {
	return db.PutDAGBlockIdByHash(block.GetHash(), block.GetID())
}

func DBGetBlockIdByHash(db model.DataBase, h *hash.Hash) (uint, error) {
	return db.GetDAGBlockIdByHash(h)
}

func DBDelBlockIdByHash(db model.DataBase, h *hash.Hash) error {
	return db.DeleteDAGBlockIdByHash(h)
}

// tips
func DBPutDAGTip(db model.DataBase, id uint, isMain bool) error {
	return db.PutDAGTip(id, isMain)
}

func DBGetDAGTips(db model.DataBase) ([]uint, error) {
	return db.GetDAGTips()
}

func DBDelDAGTip(db model.DataBase, id uint) error {
	return db.DeleteDAGTip(id)
}

// diffAnticone
func DBPutDiffAnticone(db model.DataBase, id uint) error {
	return db.PutDiffAnticone(id)
}

func DBGetDiffAnticone(db model.DataBase) ([]uint, error) {
	return db.GetDiffAnticones()
}

func DBDelDiffAnticone(db model.DataBase, id uint) error {
	return db.DeleteDiffAnticone(id)
}
