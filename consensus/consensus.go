package consensus

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/database/legacychaindb"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerevm/amana"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/index"
	"sync"
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

	amanaService *amana.AmanaService
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
	//
	if s.cfg.Amana && params.ActiveNetParams.Net != protocol.MainNet {
		ser, err := amana.New(s.cfg, s)
		if err != nil {
			return err
		}
		s.amanaService = ser
	}
	//
	s.subscribe()
	return blockchain.Init()
}

func (s *consensus) DatabaseContext() model.DataBase {
	return s.databaseContext
}

// TODO: Will be deleted in the future, will be completely replaced by DatabaseContext in the future
func (s *consensus) LegacyDB() legacydb.DB {
	return s.databaseContext.(*legacychaindb.LegacyChainDB).DB()
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

func (s *consensus) AmanaService() service.IService {
	if s.amanaService == nil {
		return nil
	}
	return s.amanaService
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
