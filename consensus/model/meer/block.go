/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import (
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/ethereum/go-ethereum/core/types"
	"time"
)

type Block struct {
	Id       *hash.Hash
	Txs      []Tx
	Time     time.Time
	EvmBlock *types.Block
}

func (b *Block) ID() *hash.Hash {
	return b.Id
}

func (b *Block) Timestamp() time.Time {
	return b.Time
}

func (b *Block) Transactions() []Tx {
	return b.Txs
}

func (b *Block) Clean() {
	b.Id = nil
	b.Txs = nil
	b.EvmBlock = nil
}
