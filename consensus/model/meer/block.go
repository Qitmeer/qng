/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/ethereum/go-ethereum/core/types"
	"time"
)

type Block struct {
	Id       *hash.Hash
	Txs      []model.Tx
	Time     time.Time
	EvmBlock *types.Block
}

func (b *Block) ID() *hash.Hash {
	return b.Id
}

func (b *Block) Timestamp() time.Time {
	return b.Time
}

func (b *Block) Transactions() []model.Tx {
	return b.Txs
}
