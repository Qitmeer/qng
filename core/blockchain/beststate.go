package blockchain

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerdag"
	"time"
)

// BestState houses information about the current best block and other info
// related to the state of the main chain as it exists from the point of view of
// the current best block.
//
// The BestSnapshot method can be used to obtain access to this information
// in a concurrent safe manner and the data will not be changed out from under
// the caller when chain state changes occur as the function name implies.
// However, the returned snapshot must be treated as immutable since it is
// shared by all callers.
type BestState struct {
	Hash         hash.Hash           // The hash of the main chain tip.
	Bits         uint32              // The difficulty bits of the main chain tip.
	BlockSize    uint64              // The size of the main chain tip.
	NumTxns      uint64              // The number of txns in the main chain tip.
	MedianTime   time.Time           // Median time as per CalcPastMedianTime.
	TotalTxns    uint64              // The total number of txns in the chain.
	TotalSubsidy uint64              // The total subsidy for the chain.
	TokenTipHash *hash.Hash          // The Hash of token state tip for the chain.
	GraphState   *meerdag.GraphState // The graph state of dag
}

// newBestState returns a new best stats instance for the given parameters.
func newBestState(tipHash *hash.Hash, bits uint32, blockSize, numTxns uint64, medianTime time.Time,
	totalTxns uint64, totalsubsidy uint64, gs *meerdag.GraphState, tokenTipHash *hash.Hash) *BestState {
	return &BestState{
		Hash:         *tipHash,
		Bits:         bits,
		BlockSize:    blockSize,
		NumTxns:      numTxns,
		MedianTime:   medianTime,
		TotalTxns:    totalTxns,
		TotalSubsidy: totalsubsidy,
		TokenTipHash: tokenTipHash,
		GraphState:   gs,
	}
}