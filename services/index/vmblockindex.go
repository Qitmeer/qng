package index

import (
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/meerdag"
)

type VMBlockIndex struct {
}

func (idx *VMBlockIndex) Init(chain *blockchain.BlockChain) error {

	return nil
}

func (idx *VMBlockIndex) Key() []byte {
	return nil
}

func (idx *VMBlockIndex) Name() string {
	return "vm block index"
}

func (idx *VMBlockIndex) Create(dbTx database.Tx) error {
	return nil
}

func (idx *VMBlockIndex) ConnectBlock(dbTx database.Tx, block *types.SerializedBlock, stxos []blockchain.SpentTxOut, ib meerdag.IBlock) error {
	return nil
}

func (idx *VMBlockIndex) DisconnectBlock(dbTx database.Tx, block *types.SerializedBlock, stxos []blockchain.SpentTxOut) error {

	return nil
}

func NewVMBlockIndex(db database.DB) *VMBlockIndex {
	return &VMBlockIndex{}
}
