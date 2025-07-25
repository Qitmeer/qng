package consensus

import (
	"sync"

	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/config"
	"github.com/Qitmeer/qng/v2/consensus/model"
	"github.com/Qitmeer/qng/v2/core/blockchain"
	"github.com/Qitmeer/qng/v2/core/event"
	"github.com/Qitmeer/qng/v2/engine/txscript"
	"github.com/Qitmeer/qng/v2/log"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/Qitmeer/qng/v2/services/index"
)

type consensus struct {
	lock                   sync.Mutex
	databaseContext        model.DataBase
	cfg                    *config.Config
	interrupt              <-chan struct{}
	shutdownRequestChannel chan struct{}
	// signature cache
	sigCache *txscript.SigCache
	// event system
	events event.Feed
	// clock time service
	mediantimeSource model.MedianTimeSource

	blockchain   model.BlockChain
	indexManager model.IndexManager
}

// Init initializes consensus
func (s *consensus) Init() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if onEnd := log.LogAndMeasureExecutionTime(log.Root(), "consensus.Init"); onEnd != nil {
		defer onEnd()
	}

	s.indexManager = index.NewManager(index.ToConfig(s.cfg), s)

	// Create a new block chain instance with the appropriate configuration.
	blockchain, err := blockchain.New(s)
	if err != nil {
		return err
	}
	s.blockchain = blockchain
	s.subscribe()
	return blockchain.Init()
}

func (s *consensus) DatabaseContext() model.DataBase {
	return s.databaseContext
}

func (s *consensus) Config() *config.Config {
	return s.cfg
}

func (s *consensus) BlockChain() model.BlockChain {
	return s.blockchain
}

func (s *consensus) IndexManager() model.IndexManager {
	return s.indexManager
}

func (s *consensus) Events() *event.Feed {
	return &s.events
}

func (s *consensus) MedianTimeSource() model.MedianTimeSource {
	return s.mediantimeSource
}

func (s *consensus) SigCache() *txscript.SigCache {
	return s.sigCache
}

func (s *consensus) Interrupt() <-chan struct{} {
	return s.interrupt
}

func (s *consensus) Shutdown() {
	s.shutdownRequestChannel <- struct{}{}
}

func (s *consensus) GenesisHash() *hash.Hash {
	return params.ActiveNetParams.Params.GenesisHash
}

func (s *consensus) Params() *params.Params {
	return params.ActiveNetParams.Params
}

func (s *consensus) subscribe() {
	//ch := make(chan *event.Event)
	//sub := s.events.Subscribe(ch)
	//go func() {
	//	defer sub.Unsubscribe()
	//	for {
	//		select {
	//		case ev := <-ch:
	//			if ev.Data != nil {
	//			}
	//			if ev.Ack != nil {
	//				ev.Ack <- struct{}{}
	//			}
	//		case <-s.interrupt:
	//			log.Info("Close consensus Event Subscribe")
	//			return
	//		}
	//	}
	//}()
}

func (s *consensus) Rebuild() error {
	err := s.databaseContext.Rebuild(s.indexManager)
	if err != nil {
		return err
	}
	return s.blockchain.Rebuild()
}

func New(cfg *config.Config, databaseContext model.DataBase, interrupt <-chan struct{}, shutdownRequestChannel chan struct{}) *consensus {
	return &consensus{
		cfg:                    cfg,
		databaseContext:        databaseContext,
		mediantimeSource:       blockchain.NewMedianTime(),
		interrupt:              interrupt,
		sigCache:               txscript.NewSigCache(cfg.SigCacheMaxSize),
		shutdownRequestChannel: shutdownRequestChannel,
	}
}

func NewPure(cfg *config.Config, databaseContext model.DataBase) *consensus {
	return New(cfg, databaseContext, make(chan struct{}), make(chan struct{}))
}
