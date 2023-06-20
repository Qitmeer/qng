package model

type Block interface {
	GetID() uint
	GetState() BlockState
	GetOrder() uint
}
