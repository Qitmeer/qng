package model

type Block interface {
	GetID() uint
	// GetStatus
	GetStatus() BlockStatus
}
