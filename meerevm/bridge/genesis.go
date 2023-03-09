package bridge

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type GenesisContract interface {
	CommitState(event *EventRecordWithTime, state *state.StateDB, header *types.Header, chCtx ChainContext) (uint64, error)
	LastStateId(snapshotNumber uint64) (*big.Int, error)
}
