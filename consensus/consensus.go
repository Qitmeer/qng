package consensus

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/consensus/store/vm_block_index"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/index"
	"sync"
)

const (
	defaultPreallocateCaches = true
	defaultCacheSize         = 10_000
)

type consensus struct {
	lock            sync.Mutex
	databaseContext database.DB
	cfg             *config.Config
	quit            <-chan struct{}
	// signature cache
	sigCache *txscript.SigCache
	// event system
	events event.Feed
	// clock time service
	mediantimeSource model.MedianTimeSource

	vmblockindexStore model.VMBlockIndexStore

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

	if s.cfg.VMBlockIndex {
		vmblockindexStore, err := vm_block_index.New(s.databaseContext, 10_000, defaultPreallocateCaches)
		if err != nil {
			return err
		}
		s.vmblockindexStore = vmblockindexStore
	}
	//
	s.indexManager = index.NewManager(index.ToConfig(s.cfg), s)

	// Create a new block chain instance with the appropriate configuration.
	blockchain, err := blockchain.New(&blockchain.Config{
		DB:             s.databaseContext,
		Interrupt:      s.quit,
		ChainParams:    params.ActiveNetParams.Params,
		TimeSource:     s.mediantimeSource,
		Events:         &s.events,
		SigCache:       s.sigCache,
		IndexManager:   s.indexManager,
		DAGType:        s.cfg.DAGType,
		CacheInvalidTx: s.cfg.CacheInvalidTx,
		DataDir:        s.cfg.DataDir,
	})
	if err != nil {
		return err
	}
	s.blockchain = blockchain
	return blockchain.Init()
}

func (s *consensus) DatabaseContext() database.DB {
	return s.databaseContext
}

func (s *consensus) Config() *config.Config {
	return s.cfg
}

func (s *consensus) VMBlockIndexStore() model.VMBlockIndexStore {
	return s.vmblockindexStore
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

func (s *consensus) Quit() <-chan struct{} {
	return s.quit
}

func New(cfg *config.Config, databaseContext database.DB, quit <-chan struct{}) *consensus {
	return &consensus{
		cfg:              cfg,
		databaseContext:  databaseContext,
		mediantimeSource: blockchain.NewMedianTime(),
		quit:             quit,
		sigCache:         txscript.NewSigCache(cfg.SigCacheMaxSize),
	}
}
