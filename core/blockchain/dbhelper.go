// Copyright (c) 2017-2018 The qitmeer developers
package blockchain

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
)

// dbFetchBlockByHash uses an existing database transaction to retrieve the raw
// block for the provided hash, deserialize it, retrieve the appropriate height
// from the index, and return a dcrutil.Block with the height set.
func dbFetchBlockByHash(db model.DataBase, hash *hash.Hash) (*types.SerializedBlock, error) {
	return db.GetBlock(hash)
}

// dbFetchHeaderByHash uses an existing database transaction to retrieve the
// block header for the provided hash.
func dbFetchHeaderByHash(db model.DataBase, hash *hash.Hash) (*types.BlockHeader, error) {
	return db.GetHeader(hash)
}

// dbMaybeStoreBlock stores the provided block in the database if it's not
// already there.
func dbMaybeStoreBlock(db model.DataBase, block *types.SerializedBlock) error {
	err := db.PutBlock(block)
	if err != nil {
		if legacydb.IsError(err, legacydb.ErrBlockExists) {
			return nil
		}
		return err
	}
	return nil
}
