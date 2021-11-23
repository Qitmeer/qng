/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package evm

import (
	"fmt"
	"github.com/Qitmeer/qng-core/common/hash"
	"github.com/Qitmeer/qng-core/consensus"
	"github.com/Qitmeer/qng-core/log"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"time"
)

type Block struct {
	id       *hash.Hash
	ethBlock *types.Block
	vm       *VM
	status   consensus.Status
}

func (b *Block) ID() *hash.Hash { return b.id }

func (b *Block) Accept() error {
	b.status = consensus.Accepted
	log.Debug(fmt.Sprintf("Accepting block %s at height %d", b.ID().String(), b.Height()))
	return nil
}

func (b *Block) Reject() error {
	b.status = consensus.Rejected
	log.Debug(fmt.Sprintf("Rejecting block %s at height %d", b.ID().String(), b.Height()))
	return nil
}

func (b *Block) SetStatus(status consensus.Status) { b.status = status }

func (b *Block) Status() consensus.Status {
	return b.status
}

func (b *Block) Parent() *hash.Hash {
	h := hash.MustBytesToHash(b.ethBlock.ParentHash().Bytes())
	return &h
}

func (b *Block) Height() uint64 {
	return b.ethBlock.Number().Uint64()
}

func (b *Block) Timestamp() time.Time {
	return time.Unix(int64(b.ethBlock.Time()), 0)
}

func (b *Block) Verify() error {
	return b.verify(true)
}

func (b *Block) verify(writes bool) error {
	return nil
}

func (b *Block) Bytes() []byte {
	res, err := rlp.EncodeToBytes(b.ethBlock)
	if err != nil {
		panic(err)
	}
	return res
}

func (b *Block) String() string { return fmt.Sprintf("EVM block, ID = %s", b.ID()) }
