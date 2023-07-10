// Copyright (c) 2017-2018 The qitmeer developers
package node

import (
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
	"github.com/gofrs/flock"
	"os"
	"path/filepath"
	"sync"
)

// Node works as a server container for all service can be registered.
// such as p2p, rpc, ws etc.
type Node struct {
	service.Service
	lock sync.RWMutex

	startupTime int64

	// config
	Config *config.Config
	Params *params.Params

	// database layer
	// TODO:Will gradually be deprecated in the future
	DB  legacydb.DB
	CDB model.DataBase

	dirLock *flock.Flock // prevents concurrent use of instance directory

	interrupt <-chan struct{}

	consensus model.Consensus
}

func NewNode(cfg *config.Config, database legacydb.DB, chainParams *params.Params, interrupt <-chan struct{}) (*Node, error) {
	n := Node{
		Config:    cfg,
		DB:        database,
		Params:    chainParams,
		interrupt: interrupt,
		consensus: consensus.New(cfg, database, interrupt, system.ShutdownRequestChannel),
	}
	n.InitServices()

	// Acquire the instance directory lock.
	if err := n.openDataDir(); err != nil {
		return nil, err
	}
	return &n, nil
}

func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	log.Info("Stopping Server")
	if err := n.Service.Stop(); err != nil {
		return err
	}
	n.CDB.Close()
	// Release instance directory lock.
	n.closeDataDir()
	return nil
}

func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	log.Info("Starting Node")
	// Already started?
	if err := n.Service.Start(); err != nil {
		return err
	}

	// Finished node start
	// Server startup time. Used for the uptime command for uptime calculation.
	n.startupTime = roughtime.Now().Unix()
	n.consensus.Events().Send(event.New(event.Initialized))
	return nil
}

func (n *Node) RegisterService() error {
	if n.Config.LightNode {
		return n.registerQitmeerLight()
	}
	return n.registerQitmeerFull()
}

// register services as qitmeer Full node
func (n *Node) registerQitmeerFull() error {
	fullNode, err := newQitmeerFullNode(n)
	if err != nil {
		return err
	}
	n.Services().RegisterService(fullNode)
	return nil
}

// register services as the qitmeer Light node
func (n *Node) registerQitmeerLight() error {
	lightNode, err := newQitmeerLight(n)
	if err != nil {
		return err
	}
	n.Services().RegisterService(lightNode)
	return nil
}

// return qitmeer full
func (n *Node) GetQitmeerFull() *QitmeerFull {
	var qm *QitmeerFull
	if err := n.Services().FetchService(&qm); err != nil {
		log.Error(err.Error())
		return nil
	}
	return qm
}

func (n *Node) openDataDir() error {
	if n.Config.DataDir == "" {
		return nil // ephemeral
	}

	instdir := n.Config.DataDir
	if err := os.MkdirAll(instdir, 0700); err != nil {
		return err
	}
	// Lock the instance directory to prevent concurrent use by another instance as well as
	// accidental use of the instance directory as a database.
	n.dirLock = flock.New(filepath.Join(instdir, "LOCK"))

	if locked, err := n.dirLock.TryLock(); err != nil {
		return err
	} else if !locked {
		return ErrDatadirUsed
	}
	return nil
}

func (n *Node) closeDataDir() {
	// Release instance directory lock.
	if n.dirLock != nil && n.dirLock.Locked() {
		n.dirLock.Unlock()
		n.dirLock = nil
	}
}
