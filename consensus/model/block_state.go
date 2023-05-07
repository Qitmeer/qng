package model

import "github.com/Qitmeer/qng/common/hash"

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
}
