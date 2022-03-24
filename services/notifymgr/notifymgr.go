package notifymgr

import (
	"fmt"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/p2p"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/services/notifymgr/notify"
	"github.com/libp2p/go-libp2p-core/peer"
	"sync"
	"time"
)

const NotifyTickerDur = time.Second

const MaxNotifyProcessTimeout = time.Second*30

// NotifyMgr manage message announce & relay & notification between mempool, websocket, gbt long pull
// and rpc server.
type NotifyMgr struct {
	service.Service

	Server    *p2p.Service
	RpcServer *rpc.RpcServer

	nds    []*notify.NotifyData
	wg     sync.WaitGroup
	quit   chan struct{}
	ticker *time.Ticker

	sync.Mutex

	lastProTime time.Time
}

// AnnounceNewTransactions generates and relays inventory vectors and notifies
// both websocket and getblocktemplate long poll clients of the passed
// transactions.  This function should be called whenever new transactions
// are added to the mempool.
func (ntmgr *NotifyMgr) AnnounceNewTransactions(newTxs []*types.TxDesc, filters []peer.ID) {
	if len(newTxs) <= 0 {
		return
	}

	ntmgr.Lock()
	defer ntmgr.Unlock()

	for _, tx := range newTxs {
		ntmgr.nds = append(ntmgr.nds, &notify.NotifyData{Data: tx, Filters: filters})
	}

	ntmgr.Reset()
}

// RelayInventory relays the passed inventory vector to all connected peers
// that are not already known to have it.
func (ntmgr *NotifyMgr) RelayInventory(data interface{}, filters []peer.ID) {
	ntmgr.Lock()
	defer ntmgr.Unlock()

	ntmgr.nds = append(ntmgr.nds, &notify.NotifyData{Data: data, Filters: filters})
	ntmgr.Reset()
}

func (ntmgr *NotifyMgr) BroadcastMessage(data interface{}) {
	ntmgr.Server.BroadcastMessage(data)
}

func (ntmgr *NotifyMgr) AddRebroadcastInventory(newTxs []*types.TxDesc) {
	for _, tx := range newTxs {
		ntmgr.Server.Rebroadcast().AddInventory(tx.Tx.Hash(), tx)
	}
}

// Transaction has one confirmation on the main chain. Now we can mark it as no
// longer needing rebroadcasting.
func (ntmgr *NotifyMgr) TransactionConfirmed(tx *types.Tx) {
	ntmgr.Server.Rebroadcast().RemoveInventory(tx.Hash())
}

func (ntmgr *NotifyMgr) Start() error {
	if err := ntmgr.Service.Start(); err != nil {
		return err
	}
	//
	log.Info("Start NotifyMgr...")

	ntmgr.wg.Add(1)
	go ntmgr.handler()

	return nil
}

func (ntmgr *NotifyMgr) Stop() error {
	log.Info("try stop NotifyMgr")
	if err := ntmgr.Service.Stop(); err != nil {
		return err
	}
	log.Info("Stop NotifyMgr...")

	if ntmgr.ticker != nil {
		ntmgr.ticker.Stop()
		ntmgr.ticker = nil
	}

	close(ntmgr.quit)
	ntmgr.wg.Wait()

	return nil
}

func (ntmgr *NotifyMgr) handler() {
out:
	for {
		select {
		case <-ntmgr.ticker.C:
			ntmgr.handleStallSample()

		case <-ntmgr.quit:
			break out
		}
	}

	ntmgr.wg.Done()
	log.Trace("NotifyMgr handler done")
}

func (ntmgr *NotifyMgr) handleStallSample() {
	ntmgr.Lock()
	defer ntmgr.Unlock()

	if len(ntmgr.nds) <= 0 {
		return
	}

	txds := []*types.TxDesc{}
	for _, nd := range ntmgr.nds {
		txd, ok := nd.Data.(*types.TxDesc)
		if ok {
			log.Trace(fmt.Sprintf("Announce new transaction :hash=%s height=%d add=%s", txd.Tx.Hash().String(), txd.Height, txd.Added.String()))

			txds = append(txds, txd)
		}
	}
	ntmgr.Server.RelayInventory(ntmgr.nds)

	if len(txds) > 0 {
		if ntmgr.RpcServer != nil && ntmgr.RpcServer.IsStarted() {
			ntmgr.RpcServer.NotifyNewTransactions(txds)
		}
	}

	ntmgr.nds = []*notify.NotifyData{}
	ntmgr.lastProTime = time.Now()
}

func (ntmgr *NotifyMgr) IsTimeout() bool {
	return time.Since(ntmgr.lastProTime) >= MaxNotifyProcessTimeout
}

func (ntmgr *NotifyMgr) Reset() {
	if !ntmgr.IsTimeout() {
		ntmgr.ticker.Reset(NotifyTickerDur)
	}
}

func New(p2pServer *p2p.Service) *NotifyMgr {
	ntmgr := &NotifyMgr{
		quit:        make(chan struct{}),
		ticker:      time.NewTicker(time.Second),
		nds:         []*notify.NotifyData{},
		Server:      p2pServer,
		lastProTime: time.Now(),
	}
	return ntmgr
}
