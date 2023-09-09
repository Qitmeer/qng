package index

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
)

const (
	txhashIndexName = "transaction full hash index"
)

type TxHashIndex struct {
	consensus model.Consensus
}

func (idx *TxHashIndex) Init() error {
	log.Info("Init", "index", idx.Name())
	return nil
}

func (idx *TxHashIndex) Name() string {
	return txhashIndexName
}

func (idx *TxHashIndex) ConnectBlock(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	// TODO: For compatibility, it will be removed in the future
	if idx.consensus.DatabaseContext().IsLegacy() &&
		block.GetState().GetStatus().KnownInvalid() {
		return nil
	}
	return idx.consensus.DatabaseContext().PutTxHashs(sblock)
}

func (idx *TxHashIndex) DisconnectBlock(sblock *types.SerializedBlock, block model.Block, stxos [][]byte) error {
	return idx.consensus.DatabaseContext().DeleteTxHashs(sblock)
}

func (idx *TxHashIndex) GetTxIdByHash(fullHash hash.Hash) (*hash.Hash, error) {
	return idx.consensus.DatabaseContext().GetTxIdByHash(&fullHash)
}

func NewTxHashIndex(consensus model.Consensus) *TxHashIndex {
	return &TxHashIndex{consensus: consensus}
}
