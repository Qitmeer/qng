package consensus

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/consensus/store/invalid_tx_index"
	"github.com/Qitmeer/qng/consensus/store/vm_block_index"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/index"
	"github.com/Qitmeer/qng/vm"
	"math"
	"sync"
)

const (
	defaultPreallocateCaches = false
	defaultCacheSize         = 10
)

type consensus struct {
	lock                   sync.Mutex
	databaseContext        database.DB
	cfg                    *config.Config
	interrupt              <-chan struct{}
	shutdownRequestChannel chan struct{}
	// signature cache
	sigCache *txscript.SigCache
	// event system
	events event.Feed
	// clock time service
	mediantimeSource model.MedianTimeSource

	vmblockindexStore   model.VMBlockIndexStore
	invalidtxindexStore model.InvalidTxIndexStore

	blockchain   model.BlockChain
	indexManager model.IndexManager

	vmService *vm.Service
}

// Init initializes consensus
func (s *consensus) Init() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if onEnd := log.LogAndMeasureExecutionTime(log.Root(), "consensus.Init"); onEnd != nil {
		defer onEnd()
	}

	if s.cfg.VMBlockIndex {
		vmblockindexStore, err := vm_block_index.New(s.databaseContext, defaultCacheSize, defaultPreallocateCaches)
		if err != nil {
			return err
		}
		s.vmblockindexStore = vmblockindexStore
	}
	if s.cfg.InvalidTxIndex {
		invalidtxindexStore, err := invalid_tx_index.New(s.databaseContext, defaultCacheSize, defaultPreallocateCaches)
		if err != nil {
			return err
		}
		s.invalidtxindexStore = invalidtxindexStore
	}
	//
	s.indexManager = index.NewManager(index.ToConfig(s.cfg), s)

	// Create a new block chain instance with the appropriate configuration.
	blockchain, err := blockchain.New(s)
	if err != nil {
		return err
	}
	s.blockchain = blockchain
	//
	vmService, err := vm.NewService(s.Config(), s.Events())
	if err != nil {
		return err
	}
	s.vmService = vmService
	s.subscribe()
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

func (s *consensus) InvalidTxIndexStore() model.InvalidTxIndexStore {
	return s.invalidtxindexStore
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

func (s *consensus) VMService() model.VMI {
	return s.vmService
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
	err := s.vmService.Start()
	if err != nil {
		return err
	}
	//
	err = index.DropVMBlockIndex(s.databaseContext, s.interrupt)
	if err != nil {
		log.Info(err.Error())
	}
	err = index.DropInvalidTxIndex(s.databaseContext, s.interrupt)
	if err != nil {
		log.Info(err.Error())
	}
	err = index.DropTxIndex(s.databaseContext, s.interrupt)
	if err != nil {
		log.Info(err.Error())
	}
	//
	err = s.databaseContext.Update(func(tx database.Tx) error {
		meta := tx.Metadata()
		err = meta.DeleteBucket(dbnamespace.SpendJournalBucketName)
		if err != nil {
			return err
		}
		err = meta.DeleteBucket(dbnamespace.UtxoSetBucketName)
		if err != nil {
			return err
		}
		err = meta.DeleteBucket(dbnamespace.TokenBucketName)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	//
	err = s.databaseContext.Update(func(tx database.Tx) error {
		meta := tx.Metadata()
		_, err = meta.CreateBucket(dbnamespace.SpendJournalBucketName)
		if err != nil {
			return err
		}
		_, err = meta.CreateBucket(dbnamespace.UtxoSetBucketName)
		if err != nil {
			return err
		}
		_, err = meta.CreateBucket(dbnamespace.TokenBucketName)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	txIndex := s.indexManager.(*index.Manager).TxIndex()
	txIndex.SetCurOrder(-1)
	err = s.databaseContext.Update(func(tx database.Tx) error {
		err = txIndex.Create(tx)
		if err != nil {
			return err
		}
		err = index.DBPutIndexerTip(tx, txIndex.Key(), &hash.ZeroHash, math.MaxUint32)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.blockchain.Rebuild()
}

func New(cfg *config.Config, databaseContext database.DB, interrupt <-chan struct{}, shutdownRequestChannel chan struct{}) *consensus {
	return &consensus{
		cfg:                    cfg,
		databaseContext:        databaseContext,
		mediantimeSource:       blockchain.NewMedianTime(),
		interrupt:              interrupt,
		sigCache:               txscript.NewSigCache(cfg.SigCacheMaxSize),
		shutdownRequestChannel: shutdownRequestChannel,
	}
}

func NewPure(cfg *config.Config, databaseContext database.DB) *consensus {
	return New(cfg, databaseContext, make(chan struct{}), make(chan struct{}))
}
