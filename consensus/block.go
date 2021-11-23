/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"github.com/Qitmeer/qng-core/common/hash"
	"time"
)

type Block interface {
	Decidable
	Parent() *hash.Hash
	Verify() error
	Bytes() []byte
	Height() uint64
	Timestamp() time.Time
}

type BlockHeader interface {
}
