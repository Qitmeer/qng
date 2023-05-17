/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"time"
)

type Block interface {
	Decidable
	Parent() *hash.Hash
	Verify() error
	Bytes() []byte
	Height() uint64
	Timestamp() time.Time
	Transactions() []model.Tx
	ParentState() model.BlockState
}

type BlockHeader interface {
}
