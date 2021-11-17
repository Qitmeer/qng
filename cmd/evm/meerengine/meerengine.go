/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meerengine

import (
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
	"sync"
)

const (
	MaximumExtraDataSize uint64 = 512 // Maximum size extra data may be after Genesis.
)

type Config struct {
	CacheDir         string
	CachesInMem      int
	CachesOnDisk     int
	CachesLockMmap   bool
	DatasetDir       string
	DatasetsInMem    int
	DatasetsOnDisk   int
	DatasetsLockMmap bool

	NotifyFull bool

	Log log.Logger `toml:"-"`
}

type MeerEngine struct {
	config Config

	threads  int
	update   chan struct{}
	hashrate metrics.Meter

	lock      sync.Mutex
	closeOnce sync.Once
}

func New(config Config, notify []string, noverify bool) *MeerEngine {
	if config.Log == nil {
		config.Log = log.Root()
	}
	if config.CachesInMem <= 0 {
		config.Log.Warn("One MeerEngine cache must always be in memory", "requested", config.CachesInMem)
		config.CachesInMem = 1
	}
	if config.CacheDir != "" && config.CachesOnDisk > 0 {
		config.Log.Info("Disk storage enabled for MeerEngine caches", "dir", config.CacheDir, "count", config.CachesOnDisk)
	}
	if config.DatasetDir != "" && config.DatasetsOnDisk > 0 {
		config.Log.Info("Disk storage enabled for MeerEngine DAGs", "dir", config.DatasetDir, "count", config.DatasetsOnDisk)
	}
	ethash := &MeerEngine{
		config:   config,
		update:   make(chan struct{}),
		hashrate: metrics.NewMeterForced(),
	}
	return ethash
}

func (me *MeerEngine) Close() error {
	return nil
}

func (me *MeerEngine) Threads() int {
	me.lock.Lock()
	defer me.lock.Unlock()

	return me.threads
}

func (me *MeerEngine) SetThreads(threads int) {
	me.lock.Lock()
	defer me.lock.Unlock()

	// Update the threads and ping any running seal to pull in any changes
	me.threads = threads
	select {
	case me.update <- struct{}{}:
	default:
	}
}

func (me *MeerEngine) Hashrate() float64 {
	return me.hashrate.Rate1()
}

func (me *MeerEngine) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{}
}
