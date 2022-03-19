/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/vm/consensus"
	"time"
)

type Block struct {
	Id   *hash.Hash
	Txs  []consensus.Tx
	Time time.Time
}

func (b *Block) ID() *hash.Hash {
	return b.Id
}

func (b *Block) Accept() error {
	return nil
}

func (b *Block) Reject() error {
	return nil
}

func (b *Block) SetStatus(status consensus.Status) {

}

func (b *Block) Status() consensus.Status {
	return consensus.Unknown
}

func (b *Block) Parent() *hash.Hash {
	return nil
}

func (b *Block) Height() uint64 {
	return 0
}

func (b *Block) Timestamp() time.Time {
	return b.Time
}

func (b *Block) Verify() error {
	return nil
}

func (b *Block) verify(writes bool) error {
	return nil
}

func (b *Block) Bytes() []byte {
	return nil
}

func (b *Block) String() string {
	return fmt.Sprintf("%s", b.ID().String())
}

func (b *Block) Transactions() []consensus.Tx {
	return b.Txs
}
