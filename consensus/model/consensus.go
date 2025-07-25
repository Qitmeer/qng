package model

import (
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/config"
	"github.com/Qitmeer/qng/v2/core/event"
	"github.com/Qitmeer/qng/v2/engine/txscript"
	"github.com/Qitmeer/qng/v2/params"
)

// Consensus maintains the current core state of the node
type Consensus interface {
	Init() error
	GenesisHash() *hash.Hash
	Config() *config.Config
	DatabaseContext() DataBase
	BlockChain() BlockChain
	IndexManager() IndexManager
	Events() *event.Feed
	MedianTimeSource() MedianTimeSource
	SigCache() *txscript.SigCache
	Interrupt() <-chan struct{}
	Params() *params.Params
	Rebuild() error
	Shutdown()
}
