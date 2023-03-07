package bridge

import "github.com/ethereum/go-ethereum/common"

// StateSyncData represents state received from Root Blockchain
type StateSyncData struct {
	ID       uint64
	Contract common.Address
	Data     string
	TxHash   common.Hash
}
