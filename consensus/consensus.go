package consensus

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/log"
	"sync"
)

type consensus struct {
	lock            sync.Mutex
	databaseContext database.DB
	cfg             *config.Config
}

// Init initializes consensus
func (s *consensus) Init() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if onEnd := log.LogAndMeasureExecutionTime(log.Root(), "consensus.Init"); onEnd != nil {
		defer onEnd()
	}

	return nil
}

func New(cfg *config.Config, databaseContext database.DB) *consensus {
	return &consensus{
		cfg:             cfg,
		databaseContext: databaseContext,
	}
}
