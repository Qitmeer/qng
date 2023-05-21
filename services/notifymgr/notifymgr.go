package notifymgr

import (
	"fmt"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/p2p"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/services/notifymgr/notify"
	"github.com/Qitmeer/qng/services/zmq"
	"github.com/libp2p/go-libp2p/core/peer"
	"sync"
	"time"
)

const NotifyTickerDur = time.Second

const MaxNotifyProcessTimeout = time.Second * 30

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

	// zmq notification
	zmqNotify zmq.IZMQNotification
}

// AnnounceNewTransactions generates and relays inventory vectors and notifies
// both websocket and getblocktemplate long poll clients of the passed
// transactions.  This function should be called whenever new transactions
// are added to the mempool.
func (ntmgr *NotifyMgr) AnnounceNewTransactions(newTxs []*types.TxDesc, filters []peer.ID) {
	if ntmgr.IsShutdown() {
		return
	}
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
	if ntmgr.IsShutdown() {
		return
	}
	_, ok := data.(types.BlockHeader)
	if !ok {
		log.Warn(fmt.Sprintf("No support relay data:%v", data))
		return
	}
	ntmgr.Server.PeerSync().RelayGraphState()
}

func (ntmgr *NotifyMgr) BroadcastMessage(data interface{}) {
	ntmgr.Server.BroadcastMessage(data)
}

func (ntmgr *NotifyMgr) AddRebroadcastInventory(newTxs []*types.TxDesc) {
	if ntmgr.IsShutdown() {
		return
	}
	for _, tx := range newTxs {
		ntmgr.Server.Rebroadcast().AddInventory(tx.Tx.Hash(), tx)
	}
}

// Transaction has one confirmation on the main chain. Now we can mark it as no
// longer needing rebroadcasting.
func (ntmgr *NotifyMgr) TransactionConfirmed(tx *types.Tx) {
	start := time.Now()
	ntmgr.Server.Rebroadcast().RemoveInventory(tx.Hash())
	if time.Now().UnixNano()/1e6-start.UnixNano()/1e6 > 100 {
		log.Info("startTransactionConfirmed", "txhash", tx.Hash().String(), "spent", time.Now().Sub(start))
	}
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

	close(ntmgr.quit)
	ntmgr.wg.Wait()

	if ntmgr.ticker != nil {
		ntmgr.ticker.Stop()
		ntmgr.ticker = nil
	}

	ntmgr.zmqNotify.Shutdown()
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

	if len(ntmgr.nds) <= 0 ||
		ntmgr.IsShutdown() {
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
		if ntmgr.ticker == nil {
			return
		}
		ntmgr.ticker.Reset(NotifyTickerDur)
	}
}

func (ntmgr *NotifyMgr) handleNotifyMsg(notification *blockchain.Notification) {
	if ntmgr.IsShutdown() {
		return
	}
	switch notification.Type {
	case blockchain.BlockAccepted:
		band, ok := notification.Data.(*blockchain.BlockAcceptedNotifyData)
		if !ok {
			log.Warn("Chain accepted notification is not BlockAcceptedNotifyData.")
			break
		}
		block := band.Block
		ntmgr.zmqNotify.BlockAccepted(block)
		// Don't relay if we are not current. Other peers that are current
		// should already know about it
		if !ntmgr.Server.IsCurrent() {
			log.Trace("we are not current")
			return
		}
		log.Trace("we are current, can do relay")
		ntmgr.RelayInventory(block.Block().Header, nil)

	// A block has been connected to the main block chain.
	case blockchain.BlockConnected:
		start := time.Now()
		blockSlice, ok := notification.Data.([]interface{})
		if !ok {
			log.Warn("Chain connected notification is not a block slice.")
			break
		}
		if len(blockSlice) != 2 {
			log.Warn("Chain connected notification is wrong size slice.")
			break
		}
		block := blockSlice[0].(*types.SerializedBlock)
		log.Info("starthandleNotifyMsgzmqNotify", "hash", block.Hash().String())
		ntmgr.zmqNotify.BlockConnected(block)
		log.Info("endhandleNotifyMsgzmqNotify", "hash", block.Hash().String(), "spent", time.Now().Sub(start))

	// A block has been disconnected from the main block chain.
	case blockchain.BlockDisconnected:
		log.Trace("Chain disconnected notification.")
		block, ok := notification.Data.(*types.SerializedBlock)
		if !ok {
			log.Warn("Chain disconnected notification is not a block slice.")
			break
		}
		ntmgr.zmqNotify.BlockDisconnected(block)
	}
}

func New(p2pServer *p2p.Service, consensus model.Consensus) *NotifyMgr {
	ntmgr := &NotifyMgr{
		quit:        make(chan struct{}),
		ticker:      time.NewTicker(time.Second),
		nds:         []*notify.NotifyData{},
		Server:      p2pServer,
		lastProTime: time.Now(),
		zmqNotify:   zmq.NewZMQNotification(consensus.Config()),
	}
	consensus.BlockChain().(*blockchain.BlockChain).Subscribe(ntmgr.handleNotifyMsg)
	return ntmgr
}
