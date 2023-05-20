package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
)

type BlockState interface {
	GetID() uint64
	SetOrder(order uint64)
	GetOrder() uint64
	IsOrdered() bool
	SetWeight(weight uint64)
	GetWeight() uint64
	GetStatus() BlockStatus
	Valid()
	Invalid()
	Root() *hash.Hash
	Bytes() ([]byte, error)
	GetEVMRoot() common.Hash
	GetEVMHash() common.Hash
	GetEVMNumber() uint64
	SetEVM(header *etypes.Header)
	GetDuplicateTxs() []int
	Update(block *types.SerializedBlock, prev BlockState, header *etypes.Header)
}
