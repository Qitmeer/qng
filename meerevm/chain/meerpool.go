/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package chain

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/ethereum/go-ethereum/miner"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	qtypes "github.com/Qitmeer/qng/core/types"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	qconsensus "github.com/Qitmeer/qng/vm/consensus"
	"github.com/deckarep/golang-set"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	txChanSize        = 4096
	chainHeadChanSize = 10
)

type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
}

type environment struct {
	signer types.Signer

	state     *state.StateDB // apply state changes here
	ancestors mapset.Set     // ancestor set (used for checking uncle parent validity)
	family    mapset.Set     // family set (used for checking uncle invalidity)
	tcount    int            // tx count in cycle
	gasPool   *core.GasPool  // available gas used to pack transactions

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
}

type MeerPool struct {
	wg      sync.WaitGroup
	quit    chan struct{}
	running int32

	ctx qconsensus.Context

	remoteTxsQM map[string]*qtypes.Transaction
	remoteTxsM  map[string]*qtypes.Transaction

	config      *miner.Config
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	eth         Backend
	chain       *core.BlockChain

	// Subscriptions
	mux          *event.TypeMux
	txsCh        chan core.NewTxsEvent
	txsSub       event.Subscription
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription

	current *environment // An environment for current running cycle.

	mu sync.RWMutex // The lock used to protect the coinbase and extra fields

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// Feeds
	pendingLogsFeed event.Feed
}

func (m *MeerPool) init(config *miner.Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, ctx qconsensus.Context) error {
	log.Info(fmt.Sprintf("Meer pool init..."))

	m.ctx = ctx
	m.config = config
	m.chainConfig = chainConfig
	m.engine = engine
	m.eth = eth
	m.mux = mux
	m.chain = eth.BlockChain()
	m.txsCh = make(chan core.NewTxsEvent, txChanSize)
	m.chainHeadCh = make(chan core.ChainHeadEvent, chainHeadChanSize)
	m.quit = make(chan struct{})
	m.remoteTxsQM = map[string]*qtypes.Transaction{}
	m.remoteTxsM = map[string]*qtypes.Transaction{}
	m.txsSub = eth.TxPool().SubscribeNewTxsEvent(m.txsCh)
	m.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(m.chainHeadCh)

	return nil
}

func (m *MeerPool) Start(coinbase common.Address) {
	if m.isRunning() {
		log.Info("Meer pool was started")
		return
	}

	atomic.StoreInt32(&m.running, 1)

	m.quit = make(chan struct{})
	m.wg.Add(1)
	go m.handler()

	m.updateTemplate(time.Now().Unix())
}

func (m *MeerPool) Close() {

}

func (m *MeerPool) Stop() {
	if !m.isRunning() {
		log.Info("Meer pool was stopped")
		return
	}
	atomic.StoreInt32(&m.running, 0)

	log.Info(fmt.Sprintf("Meer pool stopping"))
	if m.current != nil && m.current.state != nil {
		m.current.state.StopPrefetcher()
	}
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
			if m.current != nil {
				if gp := m.current.gasPool; gp != nil && gp.Gas() < params.TxGas {
					continue
				}

				txs := make(map[common.Address]types.Transactions)
				for _, tx := range ev.Txs {
					acc, _ := types.Sender(m.current.signer, tx)
					txs[acc] = append(txs[acc], tx)
				}
				txset := types.NewTransactionsByPriceAndNonce(m.current.signer, txs, m.current.header.BaseFee)
				tcount := m.current.tcount
				m.commitTransactions(txset, m.config.Etherbase)
				if tcount != m.current.tcount {
					m.updateSnapshot()

					m.AnnounceNewTransactions(ev.Txs)
				}
			}

		case <-m.chainHeadCh:
			m.updateTemplate(time.Now().Unix())

		// System stopped
		case <-m.quit:
			return
		case <-m.txsSub.Err():
			return
		case <-m.chainHeadSub.Err():
			return
		}
	}
}

func (m *MeerPool) makeCurrent(parent *types.Block, header *types.Header) error {
	state, err := m.chain.StateAt(parent.Root())
	if err != nil {
		return err
	}
	state.StartPrefetcher("meerpool")

	env := &environment{
		signer:    types.MakeSigner(m.chainConfig, header.Number),
		state:     state,
		ancestors: mapset.NewSet(),
		family:    mapset.NewSet(),
		header:    header,
	}
	for _, ancestor := range m.chain.GetBlocksFromHash(parent.Hash(), 7) {
		for _, uncle := range ancestor.Uncles() {
			env.family.Add(uncle.Hash())
		}
		env.family.Add(ancestor.Hash())
		env.ancestors.Add(ancestor.Hash())
	}
	env.tcount = 0

	if m.current != nil && m.current.state != nil {
		m.current.state.StopPrefetcher()
	}
	m.current = env
	return nil
}

func (m *MeerPool) updateSnapshot() {
	m.snapshotMu.Lock()
	defer m.snapshotMu.Unlock()

	var uncles []*types.Header
	m.snapshotBlock = types.NewBlock(
		m.current.header,
		m.current.txs,
		uncles,
		m.current.receipts,
		trie.NewStackTrie(nil),
	)

	m.snapshotReceipts = qcommon.CopyReceipts(m.current.receipts)
	m.snapshotState = m.current.state.Copy()
}

func (m *MeerPool) commitTransaction(tx *types.Transaction, coinbase common.Address) ([]*types.Log, error) {
	snap := m.current.state.Snapshot()

	receipt, err := core.ApplyTransaction(m.chainConfig, m.chain, &coinbase, m.current.gasPool, m.current.state, m.current.header, tx, &m.current.header.GasUsed, *m.chain.GetVMConfig())
	if err != nil {
		m.current.state.RevertToSnapshot(snap)
		return nil, err
	}
	m.current.txs = append(m.current.txs, tx)
	m.current.receipts = append(m.current.receipts, receipt)

	return receipt.Logs, nil
}

func (m *MeerPool) commitTransactions(txs *types.TransactionsByPriceAndNonce, coinbase common.Address) bool {
	if m.current == nil {
		return true
	}

	gasLimit := m.current.header.GasLimit
	if m.current.gasPool == nil {
		m.current.gasPool = new(core.GasPool).AddGas(gasLimit)
	}

	var coalescedLogs []*types.Log

	for {
		if m.current.gasPool.Gas() < params.TxGas {
			log.Trace("Not enough gas for further transactions", "have", m.current.gasPool, "want", params.TxGas)
			break
		}
		tx := txs.Peek()
		if tx == nil {
			break
		}
		from, _ := types.Sender(m.current.signer, tx)
		if tx.Protected() && !m.chainConfig.IsEIP155(m.current.header.Number) {
			log.Trace("Ignoring reply protected transaction", "hash", tx.Hash(), "eip155", m.chainConfig.EIP155Block)

			txs.Pop()
			continue
		}
		m.current.state.Prepare(tx.Hash(), m.current.tcount)

		logs, err := m.commitTransaction(tx, coinbase)
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txs.Pop()

		case errors.Is(err, core.ErrNonceTooLow):
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txs.Shift()

		case errors.Is(err, core.ErrNonceTooHigh):
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			txs.Pop()

		case errors.Is(err, nil):
			coalescedLogs = append(coalescedLogs, logs...)
			m.current.tcount++
			txs.Shift()

		case errors.Is(err, core.ErrTxTypeNotSupported):
			log.Trace("Skipping unsupported transaction type", "sender", from, "type", tx.Type())
			txs.Pop()

		default:
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
		}
	}
	if len(coalescedLogs) > 0 {
		cpy := make([]*types.Log, len(coalescedLogs))
		for i, l := range coalescedLogs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		m.pendingLogsFeed.Send(cpy)
	}
	return false
}

func (m *MeerPool) updateTemplate(timestamp int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tstart := time.Now()
	parent := m.chain.CurrentBlock()

	if parent.Time() >= uint64(timestamp) {
		timestamp = int64(parent.Time() + 1)
	}
	num := parent.Number()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		GasLimit:   0x7fffffffffffffff,
		Extra:      m.config.ExtraData,
		Time:       uint64(timestamp),
		Coinbase:   m.config.Etherbase,
		Difficulty: common.Big1,
	}
	// Set baseFee and GasLimit if we are on an EIP-1559 chain
	if m.chainConfig.IsLondon(header.Number) {
		header.BaseFee = misc.CalcBaseFee(m.chainConfig, parent.Header())
		if !m.chainConfig.IsLondon(parent.Number()) {
			parentGasLimit := parent.GasLimit() * params.ElasticityMultiplier
			header.GasLimit = core.CalcGasLimit(parentGasLimit, m.config.GasCeil)
		}
	}

	if err := m.engine.Prepare(m.chain, header); err != nil {
		log.Error("Failed to prepare header for meerpool", "err", err)
		return
	}
	if daoBlock := m.chainConfig.DAOForkBlock; daoBlock != nil {
		limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
		if header.Number.Cmp(daoBlock) >= 0 && header.Number.Cmp(limit) < 0 {
			if m.chainConfig.DAOForkSupport {
				header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
			} else if bytes.Equal(header.Extra, params.DAOForkBlockExtra) {
				header.Extra = []byte{} // If miner opposes, don't let it use the reserved extra-data
			}
		}
	}
	err := m.makeCurrent(parent, header)
	if err != nil {
		log.Error("Failed to create meerpool context", "err", err)
		return
	}
	env := m.current
	if m.chainConfig.DAOForkSupport && m.chainConfig.DAOForkBlock != nil && m.chainConfig.DAOForkBlock.Cmp(header.Number) == 0 {
		misc.ApplyDAOHardFork(env.state)
	}

	m.commit(false, tstart)

	pending, err := m.eth.TxPool().Pending(true)
	if err != nil {
		log.Error("Failed to fetch pending transactions", "err", err)
		return
	}
	if len(pending) == 0 {
		m.updateSnapshot()
		return
	}

	localTxs, remoteTxs := make(map[common.Address]types.Transactions), pending
	for _, account := range m.eth.TxPool().Locals() {
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = txs
		}
	}
	if len(localTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(m.current.signer, localTxs, header.BaseFee)
		if m.commitTransactions(txs, m.config.Etherbase) {
			return
		}
	}
	if len(remoteTxs) > 0 {
		txs := types.NewTransactionsByPriceAndNonce(m.current.signer, remoteTxs, header.BaseFee)
		if m.commitTransactions(txs, m.config.Etherbase) {
			return
		}
	}

	m.commit(true, tstart)
}

func (m *MeerPool) commit(update bool, start time.Time) error {
	receipts := qcommon.CopyReceipts(m.current.receipts)
	s := m.current.state.Copy()
	block, err := m.engine.FinalizeAndAssemble(m.chain, m.current.header, s, m.current.txs, []*types.Header{}, receipts)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	log.Debug("Update meerpool", "number", block.Number(), "txs", m.current.tcount,
		"gas", block.GasUsed(), "fees", qcommon.TotalFees(block, receipts), "elapsed", common.PrettyDuration(time.Since(start)))

	if update {
		m.updateSnapshot()
	}
	return nil
}

func (m *MeerPool) AddTx(tx *qtypes.Transaction, local bool) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if local {
		log.Warn("This function is not supported for the time being: local meer tx")
		return 0, nil
	}
	if !opreturn.IsMeerEVMTx(tx) {
		return 0, fmt.Errorf("%s is not %v", tx.TxHash().String(), qtypes.TxTypeCrossChainVM)
	}
	txb := common.FromHex(string(tx.TxIn[0].SignScript))
	var txmb = &types.Transaction{}
	err := txmb.UnmarshalBinary(txb)
	if err != nil {
		return 0, err
	}

	err = m.eth.TxPool().AddLocal(txmb)
	if err != nil {
		return 0, err
	}
	m.remoteTxsQM[tx.TxHash().String()] = tx
	m.remoteTxsM[txmb.Hash().String()] = tx
	log.Debug(fmt.Sprintf("Meer pool:add tx %s(%s)", tx.TxHash(), txmb.Hash()))

	//
	cost := txmb.Cost()
	cost = cost.Sub(cost, txmb.Value())
	cost = cost.Div(cost, qcommon.Precision)
	return cost.Int64(), nil
}

func (m *MeerPool) GetTxs() ([]*qtypes.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.snapshotMu.Lock()
	defer m.snapshotMu.Unlock()

	result := []*qtypes.Transaction{}

	if m.snapshotBlock != nil && len(m.snapshotBlock.Transactions()) > 0 {
		for _, tx := range m.snapshotBlock.Transactions() {

			var timestamp int64
			qtx, ok := m.remoteTxsM[tx.Hash().String()]
			if ok {
				timestamp = qtx.Timestamp.Unix()
			}
			//

			mtx := qcommon.ToQNGTx(tx, timestamp)
			if mtx == nil {
				continue
			}

			result = append(result, mtx)
		}
	}

	return result, nil
}

func (m *MeerPool) GetSize() int64 {
	m.snapshotMu.Lock()
	defer m.snapshotMu.Unlock()

	if m.snapshotBlock != nil {
		return int64(len(m.snapshotBlock.Transactions()))
	}
	return 0
}

func (m *MeerPool) RemoveTx(tx *qtypes.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !opreturn.IsMeerEVMTx(tx) {
		return fmt.Errorf("%s is not %v", tx.TxHash().String(), qtypes.TxTypeCrossChainVM)
	}
	h := qcommon.ToEVMHash(&tx.TxIn[0].PreviousOut.Hash)

	delete(m.remoteTxsQM, tx.TxHash().String())
	delete(m.remoteTxsM, h.String())

	m.eth.TxPool().RemoveTx(h, false)

	log.Debug(fmt.Sprintf("Meer pool:remove tx %s(%s)", tx.TxHash(), h))
	return nil
}

func (m *MeerPool) AnnounceNewTransactions(txs []*types.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	localTxs := []*qtypes.TxDesc{}
	blockTxs := map[string]struct{}{}

	m.snapshotMu.Lock()
	if m.snapshotBlock != nil && len(m.snapshotBlock.Transactions()) > 0 {
		for _, tx := range m.snapshotBlock.Transactions() {
			blockTxs[tx.Hash().String()] = struct{}{}
		}
	}
	m.snapshotMu.Unlock()

	for _, tx := range txs {
		_, ok := blockTxs[tx.Hash().String()]
		if !ok {
			continue
		}
		_, okR := m.remoteTxsM[tx.Hash().String()]
		if okR {
			continue
		}
		qtx := qcommon.ToQNGTx(tx, time.Now().Unix())
		if qtx == nil {
			continue
		}
		//
		cost := tx.Cost()
		cost = cost.Sub(cost, tx.Value())
		cost = cost.Div(cost, qcommon.Precision)
		fee := cost.Int64()

		td := &qtypes.TxDesc{
			Tx:       qtypes.NewTx(qtx),
			Added:    time.Now(),
			Height:   m.ctx.GetTxPool().GetMainHeight(),
			Fee:      fee,
			FeePerKB: fee * 1000 / int64(qtx.SerializeSize()),
		}

		localTxs = append(localTxs, td)

		m.ctx.GetTxPool().AddTransaction(td.Tx, uint64(td.Height), td.Fee)
	}

	//
	m.ctx.GetNotify().AnnounceNewTransactions(localTxs, nil)
	m.ctx.GetNotify().AddRebroadcastInventory(localTxs)

	return nil
}

func (m *MeerPool) Mining() bool {
	log.Debug("Temporarily not supported: Mining")
	return false
}

func (m *MeerPool) Hashrate() uint64 {
	log.Debug("Temporarily not supported: Hashrate")
	return 0
}

func (m *MeerPool) SetExtra(extra []byte) error {
	log.Debug("Temporarily not supported: SetExtra")
	return nil
}

func (m *MeerPool) SetRecommitInterval(interval time.Duration) {
	log.Debug("Temporarily not supported: SetRecommitInterval")
}

func (m *MeerPool) Pending() (*types.Block, *state.StateDB) {
	m.snapshotMu.RLock()
	defer m.snapshotMu.RUnlock()
	if m.snapshotState == nil {
		return nil, nil
	}
	return m.snapshotBlock, m.snapshotState.Copy()
}

func (m *MeerPool) PendingBlock() *types.Block {
	m.snapshotMu.RLock()
	defer m.snapshotMu.RUnlock()
	return m.snapshotBlock
}

func (m *MeerPool) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	m.snapshotMu.RLock()
	defer m.snapshotMu.RUnlock()
	return m.snapshotBlock, m.snapshotReceipts
}

func (m *MeerPool) SetEtherbase(addr common.Address) {
	log.Debug("Temporarily not supported: SetEtherbase")
}

func (m *MeerPool) SetGasCeil(ceil uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.GasCeil = ceil
}

func (m *MeerPool) EnablePreseal() {
	log.Debug("Temporarily not supported: EnablePreseal")
}

func (m *MeerPool) DisablePreseal() {
	log.Debug("Temporarily not supported: DisablePreseal")
}

func (m *MeerPool) SubscribePendingLogs(ch chan<- []*types.Log) event.Subscription {
	return m.pendingLogsFeed.Subscribe(ch)
}
