package api

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math"
	"strings"
)

type BlockOrder int64

const (
	SafeBlockOrder      = BlockOrder(-4)
	FinalizedBlockOrder = BlockOrder(-3)
	PendingBlockOrder   = BlockOrder(-2)
	LatestBlockOrder    = BlockOrder(-1)
	EarliestBlockOrder  = BlockOrder(0)
)

// UnmarshalJSON parses the given JSON fragment into a BlockOrder. It supports:
// - "safe", "finalized", "latest", "earliest" or "pending" as string arguments
// - the block order
// Returned errors:
// - an invalid block order error when the given argument isn't a known strings
// - an out of range error when the given block order is either too little or too large
func (bo *BlockOrder) UnmarshalJSON(data []byte) error {
	input := strings.TrimSpace(string(data))
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		input = input[1 : len(input)-1]
	}

	switch input {
	case "earliest":
		*bo = EarliestBlockOrder
		return nil
	case "latest":
		*bo = LatestBlockOrder
		return nil
	case "pending":
		*bo = PendingBlockOrder
		return nil
	case "finalized":
		*bo = FinalizedBlockOrder
		return nil
	case "safe":
		*bo = SafeBlockOrder
		return nil
	}

	blckOrder, err := hexutil.DecodeUint64(input)
	if err != nil {
		return err
	}
	if blckOrder > math.MaxInt64 {
		return fmt.Errorf("block order larger than int64")
	}
	*bo = BlockOrder(blckOrder)
	return nil
}

// MarshalText implements encoding.TextMarshaler. It marshals:
// - "safe", "finalized", "latest", "earliest" or "pending" as strings
// - other orders as hex
func (bo BlockOrder) MarshalText() ([]byte, error) {
	switch bo {
	case EarliestBlockOrder:
		return []byte("earliest"), nil
	case LatestBlockOrder:
		return []byte("latest"), nil
	case PendingBlockOrder:
		return []byte("pending"), nil
	case FinalizedBlockOrder:
		return []byte("finalized"), nil
	case SafeBlockOrder:
		return []byte("safe"), nil
	default:
		return hexutil.Uint64(bo).MarshalText()
	}
}

func (bo BlockOrder) Int64() int64 {
	return (int64)(bo)
}
