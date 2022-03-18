/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"math/big"
)

// Any ConsensusState
type ConsensusState interface {
}

type ConsensusAlgorithm interface {
	SetState(state ConsensusState) (ConsensusState, error)
}

// agree on a consensus state
type Consensus interface {
	GetCurrentState() (ConsensusState, error)
	Commit(state ConsensusState) (ConsensusState, error)
	SetAlgorithm(algorithm ConsensusAlgorithm)
}

type Operation interface {
	ApplyTo(state ConsensusState) (ConsensusState, error)
}

// agree on a operation that update the consensus state
type Agreement interface {
	Commit(operation Operation) (ConsensusState, error)
	GetHeadState() (ConsensusState, error)
	Rollback(state ConsensusState) error
}

// the algorithm agnostic consensus engine.
type BlockChainConsensue interface {

	// VerifySeal checks whether the crypto seal on a header is valid according to
	// the consensus rules of the given engine.
	Verify(chain BlockChain, header BlockHeader) error

	// Prepare initializes the consensus fields of a block header according to the
	// rules of a particular engine. The changes are executed inline.
	Prepare(chain BlockChain, header BlockHeader) error

	// Finalize runs any post-transaction state modifications (e.g. block rewards)
	// and assembles the final block.
	// Note: The block header and state database might be updated to reflect any
	// consensus rules that happen at finalization (e.g. block rewards).
	Finalize() (Block, error)

	// Generates a new block for the given input block with the local miner's
	// seal place on top.
	Generate(chain BlockChain, block Block, stop <-chan struct{}) (Block, error)
}

// PoW is a consensus engine based on proof-of-work.
type PoW interface {
	BlockChainConsensue
	// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
	// that a new block should have.
	CalcDifficulty(chain BlockChain, time uint64, parent BlockHeader) *big.Int

	// Hashrate returns the current mining hashrate of a PoW consensus engine.
	Hashrate() float64
}

type PoA interface {
	BlockChainConsensue
}
