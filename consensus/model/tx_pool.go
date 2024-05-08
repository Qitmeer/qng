/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

type TxPool interface {
	RemoveTransaction(tx *types.Tx, removeRedeemers bool)

	RemoveDoubleSpends(tx *types.Tx)

	RemoveOrphan(tx *types.Tx)

	ProcessOrphans(tx *types.Tx) []*types.TxDesc

	MaybeAcceptTransaction(tx *types.Tx, isNew, rateLimit bool) ([]*hash.Hash, error)

	HaveTransaction(hash *hash.Hash) bool

	PruneExpiredTx()

	ProcessTransaction(tx *types.Tx, allowOrphan, rateLimit, allowHighFees bool) ([]*types.TxDesc, error)

	GetMainHeight() int64

	AddTransaction(tx *types.Tx, height uint64, fee int64)

	IsSupportVMTx() bool
}

type FeeEstimator interface {
	RegisterBlock(block *types.SerializedBlock, mainheight uint) error
	Rollback(hash *hash.Hash) error
}
