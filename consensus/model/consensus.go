package model

// Consensus maintains the current core state of the node
type Consensus interface {
	Init() error
}
