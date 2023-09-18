/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"github.com/Qitmeer/qng/log"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/rpc"
	"sync"
)

type MeerEngine struct {
	log log.Logger

	threads int
	update  chan struct{}

	lock      sync.Mutex
	closeOnce sync.Once
}

func New() *MeerEngine {
	return &MeerEngine{
		log:    log.Root(),
		update: make(chan struct{}),
	}
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

func (me *MeerEngine) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{}
}
