/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"github.com/Qitmeer/qng-core/common/hash"
	"github.com/Qitmeer/qng-core/meerdag"
)

type BlockDAG interface {
	DAG
	GetBlock(h *hash.Hash) meerdag.IBlock
}