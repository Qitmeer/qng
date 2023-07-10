package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
)

// Consensus maintains the current core state of the node
type Consensus interface {
	Init() error
	GenesisHash() *hash.Hash
	Config() *config.Config
	DatabaseContext() legacydb.DB
	BlockChain() BlockChain
	IndexManager() IndexManager
	Events() *event.Feed
	MedianTimeSource() MedianTimeSource
	SigCache() *txscript.SigCache
	InvalidTxIndexStore() InvalidTxIndexStore
	Interrupt() <-chan struct{}
	Params() *params.Params
	Rebuild() error
	AmanaService() service.IService
	Shutdown()
}
