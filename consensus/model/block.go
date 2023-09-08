package model

import "github.com/Qitmeer/qng/common/hash"

type Block interface {
	GetID() uint
	GetHash() *hash.Hash
	GetState() BlockState
	GetOrder() uint
}
