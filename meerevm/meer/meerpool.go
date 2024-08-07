/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/consensus/model/meer"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/Qitmeer/qng/meerevm/meer/crosschain"
	"github.com/Qitmeer/qng/params"
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
	blockTag          = byte(1)
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

	syncing atomic.Bool // The indicator whether the node is still syncing.
	dirty   atomic.Bool

	p2pSer model.P2PService
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

	stallTicker := time.NewTicker(params.ActiveNetParams.TargetTimePerBlock)
	defer stallTicker.Stop()

	for {
		select {
		case ev := <-m.txsCh:
			if m.qTxPool == nil {
				continue
			}
			if !m.qTxPool.IsSupportVMTx() {
				for _, tx := range ev.Txs {
					m.ethTxPool.RemoveTx(tx.Hash(), true)
				}
				continue
			}
			for _, tx := range ev.Txs {
				if crosschain.IsCrossChainExportTx(tx) {
					vmtx, err := meer.NewVMTx(qcommon.ToQNGTx(tx, 0, true).Tx, nil)
					if err != nil {
						log.Error(err.Error())
						m.ethTxPool.RemoveTx(tx.Hash(), true)
						continue
					}
					err = m.consensus.BlockChain().VerifyMeerTx(vmtx)
					if err != nil {
						log.Error(err.Error())
						m.ethTxPool.RemoveTx(tx.Hash(), true)
						continue
					}
				}
			}

			m.qTxPool.TriggerDirty()
			m.p2pSer.Notify().AnnounceNewTransactions(nil, ev.Txs, nil)
			m.dirty.Store(true)
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
		case <-stallTicker.C:
			m.handleStallSample()
		}
	}
}

func (m *MeerPool) handleStallSample() {
	if m.syncing.Load() {
		return
	}
	if !m.eth.Synced() && m.p2pSer.IsCurrent() {
		m.eth.SetSynced()
	}
	if !m.dirty.Load() {
		return
	}
	m.snapshotMu.Lock()
	block := m.snapshotBlock
	m.snapshotMu.Unlock()
	if block != nil {
		if time.Since(time.Unix(int64(block.Time()), 0)) <= params.ActiveNetParams.TargetTimePerBlock {
			return
		}
	}

	go m.ResetTemplate()
}

func (m *MeerPool) updateTemplate(force bool) error {
	if m.syncing.Load() {
		return nil
	}
	parentHash := m.eth.BlockChain().CurrentBlock().Hash()
	m.snapshotMu.Lock()
	if m.snapshotBlock != nil && !force {
		if parentHash == m.snapshotBlock.ParentHash() {
			return nil
		}
	}
	m.snapshotMu.Unlock()

	block, receipts, _ := m.eth.Miner().Pending()
	if block == nil {
		return nil
	}
	err := m.checkCrossChainTxs(block, receipts)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	m.snapshotMu.Lock()
	defer m.snapshotMu.Unlock()

	m.snapshotBlock = block
	m.snapshotQTxsM = map[string]*snapshotTx{}
	m.snapshotTxsM = map[string]*snapshotTx{}
	txsNum := len(block.Transactions())
	if txsNum > 0 {
		for _, tx := range block.Transactions() {
			mtx := qcommon.ToQNGTx(tx, 0, true)
			stx := &snapshotTx{tx: mtx, eHash: tx.Hash()}
			m.snapshotQTxsM[mtx.Hash().String()] = stx
			m.snapshotTxsM[tx.Hash().String()] = stx
		}
	}
	//
	log.Debug("meerpool update block template", "txs", txsNum)
	m.dirty.Store(false)
	return nil
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
func (m *MeerPool) HasTx(h *hash.Hash) bool {
	m.snapshotMu.RLock()
	_, ok := m.snapshotQTxsM[h.String()]
	m.snapshotMu.RUnlock()
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
	h := qcommon.ToEVMHash(&tx.Tx.TxIn[0].PreviousOut.Hash)
	if m.eth.TxPool().Has(h) {
		m.ethTxPool.RemoveTx(h, false)
		log.Debug(fmt.Sprintf("Meer pool:remove tx %s(%s) from eth", tx.Hash(), h))
	}
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

func (m *MeerPool) SetP2P(ser model.P2PService) {
	m.p2pSer = ser
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
