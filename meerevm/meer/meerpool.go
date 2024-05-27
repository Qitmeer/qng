/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/miner"
	"sync"
	"sync/atomic"
	"time"

	qevent "github.com/Qitmeer/qng/core/event"
	qtypes "github.com/Qitmeer/qng/core/types"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

const (
	txChanSize        = 4096
	chainHeadChanSize = 10
)

type resetTemplateMsg struct {
	reply chan struct{}
}

type snapshotTx struct {
	tx    *qtypes.Tx
	eHash common.Hash
}

type MeerPool struct {
	wg      sync.WaitGroup
	quit    chan struct{}
	running int32

	consensus model.Consensus

	remoteTxsM  map[string]*qtypes.Tx
	remoteQTxsM map[string]*snapshotTx
	remoteMu    sync.RWMutex

	config    *miner.Config
	eth       *eth.Ethereum
	ethTxPool *legacypool.LegacyPool

	// Subscriptions
	txsCh        chan core.NewTxsEvent
	txsSub       event.Subscription
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription

	snapshotMu    sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock *types.Block
	snapshotQTxsM map[string]*snapshotTx
	snapshotTxsM  map[string]*snapshotTx

	resetTemplate chan *resetTemplateMsg

	qTxPool model.TxPool
	notify  model.Notify

	syncing atomic.Bool // The indicator whether the node is still syncing.

	payload *miner.Payload
}

func (m *MeerPool) Start() {
	if m.isRunning() {
		log.Info("Meer pool was started")
		return
	}

	atomic.StoreInt32(&m.running, 1)

	m.quit = make(chan struct{})
	m.wg.Add(1)
	go m.handler()
	m.subscribe()

	m.updateTemplate(true)
}

func (m *MeerPool) Stop() {
	if !m.isRunning() {
		log.Info("Meer pool was stopped")
		return
	}
	atomic.StoreInt32(&m.running, 0)

	log.Info(fmt.Sprintf("Meer pool stopping"))
	close(m.quit)
	m.wg.Wait()

	log.Info(fmt.Sprintf("Meer pool stopped"))
}

func (m *MeerPool) isRunning() bool {
	return atomic.LoadInt32(&m.running) == 1
}

func (m *MeerPool) handler() {
	defer m.txsSub.Unsubscribe()
	defer m.chainHeadSub.Unsubscribe()
	defer m.wg.Done()

	for {
		select {
		case ev := <-m.txsCh:
			if m.qTxPool == nil {
				continue
			}
			if !m.qTxPool.IsSupportVMTx() {
				for _, tx := range ev.Txs {
					m.ethTxPool.RemoveTx(tx.Hash(), false)
				}
				continue
			}
			m.AnnounceNewTransactions(ev.Txs)

		// System stopped
		case <-m.quit:
			return
		case <-m.txsSub.Err():
			return
		case <-m.chainHeadCh:
			m.updateTemplate(false)
		case <-m.chainHeadSub.Err():
			return
		case msg := <-m.resetTemplate:
			m.updateTemplate(true)
			msg.reply <- struct{}{}
		}
	}
}

func (m *MeerPool) updateSnapshot() error {
	if m.payload == nil {
		return nil
	}
	m.snapshotMu.Lock()
	defer m.snapshotMu.Unlock()

	block := m.payload.ResolveFullBlock()

	m.snapshotBlock = block
	m.snapshotQTxsM = map[string]*snapshotTx{}
	m.snapshotTxsM = map[string]*snapshotTx{}
	txsNum := len(block.Transactions())
	if txsNum > 0 {
		for _, tx := range block.Transactions() {
			var mtx *qtypes.Tx
			m.remoteMu.RLock()
			qtx, ok := m.remoteTxsM[tx.Hash().String()]
			m.remoteMu.RUnlock()
			if ok {
				mtx = qtx
			} else {
				mtx = qcommon.ToQNGTx(tx, 0, true)
			}
			if mtx == nil {
				continue
			}
			stx := &snapshotTx{tx: mtx, eHash: tx.Hash()}
			m.snapshotQTxsM[mtx.Hash().String()] = stx
			m.snapshotTxsM[tx.Hash().String()] = stx
		}
	}
	//
	m.remoteMu.RLock()
	remoteSize := len(m.remoteTxsM)
	m.remoteMu.RUnlock()
	log.Debug("update meerpool snapshot", "size", txsNum, "remoteSize", remoteSize)
	return nil
}

func (m *MeerPool) updateTemplate(force bool) error {
	if m.syncing.Load() {
		return nil
	}
	parentHash := m.eth.BlockChain().CurrentBlock().Hash()
	if m.payload != nil && !force {
		if parentHash == m.payload.ResolveEmpty().ExecutionPayload.ParentHash {
			return nil
		}
	}
	log.Debug("meerpool update block template")

	args := &miner.BuildPayloadArgs{
		Parent:       parentHash,
		Timestamp:    uint64(time.Now().Unix() + 1),
		FeeRecipient: common.Address{},
		Random:       common.Hash{},
		Withdrawals:  nil,
		BeaconRoot:   nil,
	}
	payload, err := m.eth.Miner().BuildPayload(args)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	m.payload = payload
	return m.updateSnapshot()
}

func (m *MeerPool) AddTx(tx *qtypes.Tx, local bool) (int64, error) {
	if local {
		log.Warn("This function is not supported for the time being: local meer tx")
		return 0, nil
	}
	if !opreturn.IsMeerEVMTx(tx.Tx) {
		return 0, fmt.Errorf("%s is not %v", tx.Hash().String(), qtypes.TxTypeCrossChainVM)
	}
	h := qcommon.ToEVMHash(&tx.Tx.TxIn[0].PreviousOut.Hash)
	if m.eth.TxPool().Has(h) {
		return 0, fmt.Errorf("already exists:%s (evm:%s)", tx.Hash().String(), h.String())
	}
	txb := qcommon.ToTxHex(tx.Tx.TxIn[0].SignScript)
	var txmb = &types.Transaction{}
	err := txmb.UnmarshalBinary(txb)
	if err != nil {
		return 0, err
	}

	errs := m.ethTxPool.AddRemotesSync(types.Transactions{txmb})
	if len(errs) > 0 && errs[0] != nil {
		return 0, errs[0]
	}
	m.remoteMu.Lock()
	m.remoteTxsM[txmb.Hash().String()] = tx
	m.remoteQTxsM[tx.Hash().String()] = &snapshotTx{tx: tx, eHash: txmb.Hash()}
	remoteSize := len(m.remoteTxsM)
	m.remoteMu.Unlock()
	log.Debug("Meer pool:add", "hash", tx.Hash(), "eHash", txmb.Hash(), "size", remoteSize)

	//
	cost := txmb.Cost()
	cost = cost.Sub(cost, txmb.Value())
	cost = cost.Div(cost, qcommon.Precision)
	return cost.Int64(), nil
}

func (m *MeerPool) GetTxs() ([]*qtypes.Tx, []*hash.Hash, error) {
	m.snapshotMu.RLock()
	defer m.snapshotMu.RUnlock()

	result := []*qtypes.Tx{}
	mtxhs := []*hash.Hash{}

	if m.snapshotBlock != nil && len(m.snapshotBlock.Transactions()) > 0 {
		for _, tx := range m.snapshotBlock.Transactions() {
			qtx, ok := m.snapshotTxsM[tx.Hash().String()]
			if !ok {
				continue
			}
			result = append(result, qtx.tx)
			mtxhs = append(mtxhs, qcommon.FromEVMHash(tx.Hash()))
		}
	}

	return result, mtxhs, nil
}

// all: contain txs in pending and queue
func (m *MeerPool) HasTx(h *hash.Hash, all bool) bool {
	m.snapshotMu.RLock()
	_, ok := m.snapshotQTxsM[h.String()]
	m.snapshotMu.RUnlock()

	if all && !ok {
		m.remoteMu.RLock()
		stx, okR := m.remoteQTxsM[h.String()]
		m.remoteMu.RUnlock()
		if okR && stx != nil {
			ok = m.eth.TxPool().Has(stx.eHash)
		}
	}
	return ok
}

func (m *MeerPool) GetSize() int64 {
	m.snapshotMu.RLock()
	defer m.snapshotMu.RUnlock()

	if m.snapshotBlock != nil {
		return int64(len(m.snapshotBlock.Transactions()))
	}
	return 0
}

func (m *MeerPool) RemoveTx(tx *qtypes.Tx) error {
	if !m.isRunning() {
		return fmt.Errorf("meer pool is not running")
	}
	if !opreturn.IsMeerEVMTx(tx.Tx) {
		return fmt.Errorf("%s is not %v", tx.Hash().String(), qtypes.TxTypeCrossChainVM)
	}

	m.remoteMu.Lock()
	h := qcommon.ToEVMHash(&tx.Tx.TxIn[0].PreviousOut.Hash)
	_, ok := m.remoteTxsM[h.String()]
	if ok {
		delete(m.remoteTxsM, h.String())
		delete(m.remoteQTxsM, tx.Hash().String())
		log.Debug(fmt.Sprintf("Meer pool:remove tx %s(%s) from remote, size:%d", tx.Hash().String(), h, len(m.remoteTxsM)))
	}
	m.remoteMu.Unlock()

	if m.eth.TxPool().Has(h) {
		m.ethTxPool.RemoveTx(h, false)
		log.Debug(fmt.Sprintf("Meer pool:remove tx %s(%s) from eth", tx.Hash(), h))
	}

	return nil
}

func (m *MeerPool) AnnounceNewTransactions(txs []*types.Transaction) error {
	if m.ethTxPool.All().LocalCount() <= 0 {
		return nil
	}
	localTxs := []*qtypes.TxDesc{}

	for _, tx := range txs {
		if m.ethTxPool.All().GetLocal(tx.Hash()) == nil {
			continue
		}
		var mtx *qtypes.Tx
		m.remoteMu.RLock()
		qtx, ok := m.remoteTxsM[tx.Hash().String()]
		m.remoteMu.RUnlock()
		if ok {
			mtx = qtx
		} else {
			mtx = qcommon.ToQNGTx(tx, 0, true)
		}
		//
		cost := tx.Cost()
		cost = cost.Sub(cost, tx.Value())
		cost = cost.Div(cost, qcommon.Precision)
		fee := cost.Int64()

		td := &qtypes.TxDesc{
			Tx:       mtx,
			Added:    time.Now(),
			Height:   m.qTxPool.GetMainHeight(),
			Fee:      fee,
			FeePerKB: fee * 1000 / int64(mtx.Tx.SerializeSize()),
		}

		localTxs = append(localTxs, td)
		m.qTxPool.AddTransaction(td.Tx, uint64(td.Height), td.Fee)
	}
	if len(localTxs) <= 0 {
		return nil
	}
	//
	m.notify.AnnounceNewTransactions(localTxs, nil)
	go m.notify.AddRebroadcastInventory(localTxs)

	return nil
}

func (m *MeerPool) ResetTemplate() error {
	log.Debug("Try to reset meer pool")
	msg := &resetTemplateMsg{reply: make(chan struct{})}
	m.resetTemplate <- msg
	<-msg.reply
	return nil
}

func (m *MeerPool) SetTxPool(tp model.TxPool) {
	m.qTxPool = tp
}

func (m *MeerPool) SetNotify(notify model.Notify) {
	m.notify = notify
}

func (m *MeerPool) subscribe() {
	ch := make(chan *qevent.Event)
	sub := m.consensus.Events().Subscribe(ch)
	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case ev := <-ch:
				if ev.Data != nil {
					switch value := ev.Data.(type) {
					case int:
						if value == qevent.DownloaderStart {
							m.syncing.Store(true)
						} else if value == qevent.DownloaderEnd {
							m.syncing.Store(false)
							m.ResetTemplate()
						}
					}
				}
				if ev.Ack != nil {
					ev.Ack <- struct{}{}
				}
			case <-m.quit:
				log.Info("Close MeerPool Event Subscribe")
				return
			}
		}
	}()
}

func newMeerPool(consensus model.Consensus, eth *eth.Ethereum) *MeerPool {
	log.Info(fmt.Sprintf("New Meer pool"))
	m := &MeerPool{}
	m.consensus = consensus
	m.eth = eth
	m.txsCh = make(chan core.NewTxsEvent, txChanSize)
	m.chainHeadCh = make(chan core.ChainHeadEvent, chainHeadChanSize)
	m.quit = make(chan struct{})
	m.resetTemplate = make(chan *resetTemplateMsg)
	m.remoteTxsM = map[string]*qtypes.Tx{}
	m.remoteQTxsM = map[string]*snapshotTx{}
	m.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(m.chainHeadCh)
	for _, sp := range eth.TxPool().Subpools() {
		ltp, ok := sp.(*legacypool.LegacyPool)
		if ok {
			m.ethTxPool = ltp
			break
		}
	}
	m.txsSub = m.eth.TxPool().SubscribeTransactions(m.txsCh, true)
	m.snapshotQTxsM = map[string]*snapshotTx{}
	m.snapshotTxsM = map[string]*snapshotTx{}
	return m
}
