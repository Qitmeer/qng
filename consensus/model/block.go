package model

import "github.com/Qitmeer/qng/v2/common/hash"

type Block interface {
	GetID() uint
	GetHash() *hash.Hash
	GetState() BlockState
	GetOrder() uint
	HasParents() bool
	GetMainParent() uint
	GetHeight() uint
}
