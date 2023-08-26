package chaindb

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
)

func (cdb *ChainDB) PutTxIndexEntrys(sblock *types.SerializedBlock, block model.Block) error {

	return nil
}

func (cdb *ChainDB) GetTxIndexEntry(id *hash.Hash, verbose bool) (*types.Tx, *hash.Hash, error) {
	return nil, nil, nil
}

func (cdb *ChainDB) DeleteTxIndexEntrys(block *types.SerializedBlock) error {
	return nil
}

func (cdb *ChainDB) PutTxHashs(block *types.SerializedBlock) error {
	return nil
}

func (cdb *ChainDB) GetTxIdByHash(fullHash *hash.Hash) (*hash.Hash, error) {
	return nil, nil
}

func (cdb *ChainDB) DeleteTxHashs(block *types.SerializedBlock) error {
	return nil
}
