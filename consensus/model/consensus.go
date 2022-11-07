package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/engine/txscript"
)

// Consensus maintains the current core state of the node
type Consensus interface {
	Init() error
	GenesisHash() *hash.Hash
	Config() *config.Config
	DatabaseContext() database.DB
	BlockChain() BlockChain
	IndexManager() IndexManager
	Events() *event.Feed
	MedianTimeSource() MedianTimeSource
	SigCache() *txscript.SigCache
	VMBlockIndexStore() VMBlockIndexStore
	Interrupt() <-chan struct{}
}
