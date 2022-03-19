// Copyright (c) 2020 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blkmgr

import (
	"github.com/Qitmeer/qng/vm/consensus"
)

type TxManager interface {
	MemPool() consensus.TxPool
	FeeEstimator() consensus.FeeEstimator
	InitDefaultFeeEstimator()
}
