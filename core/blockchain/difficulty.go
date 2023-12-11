// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"math/big"
	"time"

	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types/pow"
)

// CalcEasiestDifficulty calculates the easiest possible difficulty that a block
// can have given starting difficulty bits and a duration.  It is mainly used to
// verify that claimed proof of work by a block is sane as compared to a
// known good checkpoint.
func (m *BlockChain) calcEasiestDifficulty(bits uint32, duration time.Duration, powInstance pow.IPow) uint32 {
	return m.difficultyManager.CalcEasiestDifficulty(bits, duration, powInstance)
}

func (m *BlockChain) calcNextRequiredDifficulty(block model.Block, newBlockTime time.Time, powInstance pow.IPow) (uint32, error) {
	return m.difficultyManager.RequiredDifficulty(block, newBlockTime, powInstance)
}

// CalcNextRequiredDifficulty calculates the required difficulty for the block
// after the end of the current best chain based on the difficulty retarget
// rules.
//
// This function is safe for concurrent access.
func (b *BlockChain) CalcNextRequiredDifficulty(timestamp time.Time, powType pow.PowType) (uint32, error) {
	b.ChainRLock()
	block := b.bd.GetMainChainTip()
	instance := pow.GetInstance(powType, 0, []byte{})
	instance.SetParams(b.params.PowConfig)
	instance.SetMainHeight(pow.MainHeight(block.GetHeight() + 1))
	difficulty, err := b.difficultyManager.RequiredDifficulty(block, timestamp, instance)
	b.ChainRUnlock()
	return difficulty, err
}

// find block node by pow type
func (b *BlockChain) GetCurrentPowDiff(ib model.Block, powType pow.PowType) *big.Int {
	return b.difficultyManager.GetCurrentPowDiff(ib, powType)
}
