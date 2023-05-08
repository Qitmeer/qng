package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/meerdag"
	"math/big"
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
	StateRoot    hash.Hash
}

// newBestState returns a new best stats instance for the given parameters.
func newBestState(tipHash *hash.Hash, bits uint32, blockSize, numTxns uint64, medianTime time.Time,
	totalTxns uint64, totalsubsidy uint64, gs *meerdag.GraphState, tokenTipHash *hash.Hash, stateRoot hash.Hash) *BestState {
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
		StateRoot:    stateRoot,
	}
}

// -----------------------------------------------------------------------------
// The best chain state consists of the best block hash and order, the total
// number of transactions up to and including those in the best block, the
// total coin supply, the subsidy at the current block, the subsidy of the
// block prior (for rollbacks), and the accumulated work sum up to and
// including the best block.
//
// The serialized format is:
//
//   <block hash><block height><total txns><total subsidy><work sum length><work sum>
//
//   Field             Type             Size
//   block hash        chainhash.Hash   chainhash.HashSize
//   block order       uint32           4 bytes
//   total txns        uint64           8 bytes
//   total subsidy     int64            8 bytes
//   tokenTipHash      chainhash.Hash   chainhash.HashSize
//   work sum length   uint32           4 bytes
//   work sum          big.Int          work sum length
// -----------------------------------------------------------------------------

// bestChainState represents the data to be stored the database for the current
// best chain state.
type bestChainState struct {
	hash         hash.Hash
	total        uint64
	totalTxns    uint64
	tokenTipHash hash.Hash
	workSum      *big.Int
}

func (bcs *bestChainState) GetTotal() uint64 {
	return bcs.total
}

// dbPutBestState uses an existing database transaction to update the best chain
// state with the given parameters.
func dbPutBestState(dbTx database.Tx, snapshot *BestState, workSum *big.Int) error {
	// Serialize the current best chain state.
	tth := hash.ZeroHash
	if snapshot.TokenTipHash != nil {
		tth = *snapshot.TokenTipHash
	}
	serializedData := serializeBestChainState(bestChainState{
		hash:         snapshot.Hash,
		total:        uint64(snapshot.GraphState.GetTotal()),
		totalTxns:    snapshot.TotalTxns,
		workSum:      workSum,
		tokenTipHash: tth,
	})

	// Store the current best chain state into the database.
	return dbTx.Metadata().Put(dbnamespace.ChainStateKeyName, serializedData)
}

// serializeBestChainState returns the serialization of the passed block best
// chain state.  This is data to be stored in the chain state bucket.
func serializeBestChainState(state bestChainState) []byte {
	// Calculate the full size needed to serialize the chain state.
	workSumBytes := state.workSum.Bytes()
	workSumBytesLen := uint32(len(workSumBytes))
	serializedLen := hash.HashSize + 8 + 8 + hash.HashSize + 4 + workSumBytesLen

	// Serialize the chain state.
	serializedData := make([]byte, serializedLen)
	copy(serializedData[0:hash.HashSize], state.hash[:])
	offset := uint32(hash.HashSize)
	dbnamespace.ByteOrder.PutUint64(serializedData[offset:], state.total)
	offset += 8
	dbnamespace.ByteOrder.PutUint64(serializedData[offset:], state.totalTxns)
	offset += 8
	copy(serializedData[offset:offset+hash.HashSize], state.tokenTipHash[:])
	offset += hash.HashSize
	dbnamespace.ByteOrder.PutUint32(serializedData[offset:], workSumBytesLen)
	offset += 4
	copy(serializedData[offset:], workSumBytes)
	return serializedData[:]
}

// deserializeBestChainState deserializes the passed serialized best chain
// state.  This is data stored in the chain state bucket and is updated after
// every block is connected or disconnected form the main chain.
// block.
func DeserializeBestChainState(serializedData []byte) (bestChainState, error) {
	// Ensure the serialized data has enough bytes to properly deserialize
	// the hash, total, total transactions, total subsidy, current subsidy,
	// and work sum length.
	expectedMinLen := hash.HashSize + 8 + 8 + hash.HashSize + 4
	if len(serializedData) < expectedMinLen {
		return bestChainState{}, database.Error{
			ErrorCode: database.ErrCorruption,
			Description: fmt.Sprintf("corrupt best chain state size; min %v "+
				"got %v", expectedMinLen, len(serializedData)),
		}
	}

	state := bestChainState{}
	copy(state.hash[:], serializedData[0:hash.HashSize])
	offset := uint32(hash.HashSize)
	state.total = dbnamespace.ByteOrder.Uint64(serializedData[offset : offset+8])
	offset += 8
	state.totalTxns = dbnamespace.ByteOrder.Uint64(serializedData[offset : offset+8])
	offset += 8
	copy(state.tokenTipHash[:], serializedData[offset:offset+hash.HashSize])
	offset += hash.HashSize
	workSumBytesLen := dbnamespace.ByteOrder.Uint32(serializedData[offset : offset+4])
	offset += 4
	// Ensure the serialized data has enough bytes to deserialize the work
	// sum.
	if uint32(len(serializedData[offset:])) < workSumBytesLen {
		return bestChainState{}, database.Error{
			ErrorCode:   database.ErrCorruption,
			Description: "corrupt best chain state",
		}
	}
	workSumBytes := serializedData[offset : offset+workSumBytesLen]
	state.workSum = new(big.Int).SetBytes(workSumBytes)
	return state, nil
}
