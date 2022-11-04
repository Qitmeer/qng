package index

import (
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
)

const (
	vmblockIndexName = "vm block index"
)

type VMBlockIndex struct {
}

func (idx *VMBlockIndex) Init(chain model.BlockChain) error {

	return nil
}

func (idx *VMBlockIndex) Key() []byte {
	return nil
}

func (idx *VMBlockIndex) Name() string {
	return vmblockIndexName
}

func (idx *VMBlockIndex) Create(dbTx database.Tx) error {
	return nil
}

func (idx *VMBlockIndex) ConnectBlock(dbTx database.Tx, block *types.SerializedBlock, stxos [][]byte, blk model.Block) error {
	return nil
}

func (idx *VMBlockIndex) DisconnectBlock(dbTx database.Tx, block *types.SerializedBlock, stxos [][]byte) error {

	return nil
}

func NewVMBlockIndex(db database.DB) *VMBlockIndex {
	return &VMBlockIndex{}
}
