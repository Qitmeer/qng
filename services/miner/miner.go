package miner

import (
	"bytes"
	"context"
	ejson "encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/services/mempool"
	"github.com/Qitmeer/qng/services/mining"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	// gbtRegenerateSeconds is the number of seconds that must pass before
	// a new template is generated when the previous block hash has not
	// changed and there have been changes to the available transactions
	// in the memory pool.
	gbtRegenerateSeconds = 60

	// This is the timeout for HTTP requests to notify external miners.
	NotifyURLTimeout = 1 * time.Second
)

// mining stats
type MiningStats struct {
	LastestGbt                        time.Time `json:"lastest_gbt"`
	LastestGbtRequest                 time.Time `json:"lastest_gbt_request"`
	LastestSubmit                     time.Time `json:"lastest_submit"`
	Lastest100GbtRequests             []int64   `json:"-"`
	Lastest100Gbts                    []int64   `json:"-"`
	Lastest100GbtAvgDuration          float64   `json:"lastest_100_gbt_avg_duration"`
	Lastest100GbtRequestAvgDuration   float64   `json:"lastest_100_gbt_request_avg_duration"`
	Last100Submits                    []int64   `json:"-"`
	Last100SubmitAvgDuration          float64   `json:"last_100_submit_avg_duration"`
	SubmitAvgDuration                 float64   `json:"submit_avg_duration"`
	GbtAvgDuration                    float64   `json:"gbt_avg_duration"`
	GbtRequestAvgDuration             float64   `json:"gbt_request_avg_duration"`
	MaxGbtDuration                    float64   `json:"max_gbt_duration"`
	MaxGbtRequestDuration             float64   `json:"max_gbt_request_duration"`
	MaxGbtRequestTimeLongpollid       string    `json:"max_gbt_time_longpollid"`
	MaxSubmitDuration                 float64   `json:"max_submit_duration"`
	MaxSubmitDurationBlockHash        string    `json:"max_submit_duration_block_hash"`
	TotalGbts                         int64     `json:"total_gbts"`
	TotalGbtRequests                  int64     `json:"total_gbt_requests"`
	TotalEmptyGbts                    int64     `json:"total_empty_gbts"`
	TotalEmptyGbtResponse             int64     `json:"total_empty_gbt_response"`
	TotalSubmits                      int64     `json:"total_submits"`
	TotalTxEmptySubmits               int64     `json:"total_tx_empty_submits"`
	LastestMempoolEmptyTimestamp      int64     `json:"-"`
	MempoolEmptyAvgDuration           float64   `json:"mempool_empty_avg_duration"`
	MempoolEmptyMaxDuration           float64   `json:"mempool_empty_max_duration"`
	MempoolEmptyWarns                 int64     `json:"mempool_empty_warns"`
	Lastest100MempoolEmptyDuration    []float64 `json:"lastest_100_mempool_empty_duration"`
	Lastest100MempoolEmptyAvgDuration float64   `json:"lastest_100_mempool_empty_avg_duration"`
}

func (ms *MiningStats) MarshalJSON() ([]byte, error) {
	tFormat := "2006-01-02 15:04:05"
	type MiningStatsOutput MiningStats
	tmpMiningStats := struct {
		MiningStatsOutput
		LastestGbt                        string `json:"lastest_gbt"`
		LastestGbtRequest                 string `json:"lastest_gbt_request"`
		LastestSubmit                     string `json:"lastest_submit"`
		Lastest1MempoolEmptyAvgDuration   string `json:"lastest_1_mempool_empty_avg_duration"`
		Lastest100GbtAvgDuration          string `json:"lastest_100_gbt_avg_duration"`
		Lastest100GbtRequestAvgDuration   string `json:"lastest_100_gbt_request_avg_duration"`
		Last100SubmitAvgDuration          string `json:"last_100_submit_avg_duration"`
		SubmitAvgDuration                 string `json:"submit_avg_duration"`
		GbtAvgDuration                    string `json:"gbt_avg_duration"`
		GbtRequestAvgDuration             string `json:"gbt_request_avg_duration"`
		MaxGbtDuration                    string `json:"max_gbt_duration"`
		MaxGbtRequestDuration             string `json:"max_gbt_request_duration"`
		MaxSubmitDuration                 string `json:"max_submit_duration"`
		MempoolEmptyAvgDuration           string `json:"mempool_empty_avg_duration"`
		Lastest100MempoolEmptyAvgDuration string `json:"lastest_100_mempool_empty_avg_duration"`
		MempoolEmptyMaxDuration           string `json:"mempool_empty_max_duration"`
	}{
		MiningStatsOutput:                 (MiningStatsOutput)(*ms),
		LastestGbt:                        ms.LastestGbt.Format(tFormat),
		LastestGbtRequest:                 ms.LastestGbtRequest.Format(tFormat),
		LastestSubmit:                     ms.LastestSubmit.Format(tFormat),
		Lastest100GbtAvgDuration:          fmt.Sprintf("%.3f s", ms.Lastest100GbtAvgDuration),
		Lastest100GbtRequestAvgDuration:   fmt.Sprintf("%.3f s", ms.Lastest100GbtRequestAvgDuration),
		Last100SubmitAvgDuration:          fmt.Sprintf("%.3f s", ms.Last100SubmitAvgDuration),
		SubmitAvgDuration:                 fmt.Sprintf("%.3f s", ms.SubmitAvgDuration),
		GbtAvgDuration:                    fmt.Sprintf("%.3f s", ms.GbtAvgDuration),
		GbtRequestAvgDuration:             fmt.Sprintf("%.3f s", ms.GbtRequestAvgDuration),
		MaxGbtDuration:                    fmt.Sprintf("%.3f s", ms.MaxGbtDuration),
		MaxGbtRequestDuration:             fmt.Sprintf("%.3f s", ms.MaxGbtRequestDuration),
		MaxSubmitDuration:                 fmt.Sprintf("%.3f s", ms.MaxSubmitDuration),
		MempoolEmptyAvgDuration:           fmt.Sprintf("%.3f s", ms.MempoolEmptyAvgDuration),
		Lastest100MempoolEmptyAvgDuration: fmt.Sprintf("%.3f s", ms.Lastest100MempoolEmptyAvgDuration),
		MempoolEmptyMaxDuration:           fmt.Sprintf("%.3f s", ms.MempoolEmptyMaxDuration),
	}
	if len(ms.Lastest100MempoolEmptyDuration) > 0 {
		tmpMiningStats.Lastest1MempoolEmptyAvgDuration = fmt.Sprintf("%.3f s",
			ms.Lastest100MempoolEmptyDuration[len(ms.Lastest100MempoolEmptyDuration)-1 : len(ms.Lastest100MempoolEmptyDuration)][0])
	}
	return ejson.Marshal(tmpMiningStats)
}

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
	service.Service
	msgChan chan interface{}
	wg      sync.WaitGroup
	quit    chan struct{}

	cfg        *config.Config
	events     *event.Feed
	txpool     *mempool.TxPool
	timeSource model.MedianTimeSource
	consensus  model.Consensus
	policy     *mining.Policy
	sigCache   *txscript.SigCache
	worker     IWorker

	template        *types.BlockTemplate
	lastTxUpdate    time.Time
	lastTemplate    time.Time
	minTimestamp    time.Time
	coinbaseAddress types.Address
	powType         pow.PowType

	sync.Mutex
	submitLocker sync.Mutex

	totalSubmit   int
	successSubmit int

	coinbaseFlags mining.CoinbaseFlags

	reqWG sync.WaitGroup

	RpcSer *rpc.RpcServer
	p2pSer model.P2PService
	stats  MiningStats
}

func (m *Miner) StatsEmptyGbt() {
	if m.stats.LastestMempoolEmptyTimestamp <= 0 {
		m.stats.LastestMempoolEmptyTimestamp = time.Now().Unix()
	}
}

func (m *Miner) StatsGbtTxEmptyAvgTimes() {
	if m.stats.LastestMempoolEmptyTimestamp <= 0 || time.Now().Unix() <= m.stats.LastestMempoolEmptyTimestamp {
		return
	}
	duration := float64(time.Now().Unix() - m.stats.LastestMempoolEmptyTimestamp)
	if m.stats.MempoolEmptyAvgDuration <= 0 {
		m.stats.MempoolEmptyAvgDuration = duration
	} else {
		m.stats.MempoolEmptyAvgDuration = (m.stats.MempoolEmptyAvgDuration + duration) / 2
	}
	if len(m.stats.Lastest100MempoolEmptyDuration) >= 100 {
		m.stats.Lastest100MempoolEmptyDuration = m.stats.Lastest100MempoolEmptyDuration[len(m.stats.Lastest100MempoolEmptyDuration)-99:]
	}
	m.stats.Lastest100MempoolEmptyDuration = append(m.stats.Lastest100MempoolEmptyDuration, duration)
	sum := float64(0)
	for _, v := range m.stats.Lastest100MempoolEmptyDuration {
		sum += v
	}
	m.stats.Lastest100MempoolEmptyAvgDuration = float64(sum) / float64(len(m.stats.Lastest100MempoolEmptyDuration))

	if duration > m.stats.MempoolEmptyAvgDuration {
		m.stats.MempoolEmptyMaxDuration = duration
	}
}

func (m *Miner) StatsSubmit(currentReqMillSec int64, bh string, txcount int) {
	if len(m.stats.Last100Submits) >= 100 {
		m.stats.Last100Submits = m.stats.Last100Submits[len(m.stats.Last100Submits)-99:]
	}
	m.stats.Last100Submits = append(m.stats.Last100Submits, currentReqMillSec)
	sum := int64(0)
	for _, v := range m.stats.Last100Submits {
		sum += v
	}
	m.stats.Last100SubmitAvgDuration = float64(sum) / float64(len(m.stats.Last100Submits)) / 1000
	if m.stats.SubmitAvgDuration > 0 {
		m.stats.SubmitAvgDuration = (m.stats.SubmitAvgDuration + float64(currentReqMillSec)) / 2 / 1000
	} else {
		m.stats.SubmitAvgDuration = float64(currentReqMillSec) / 1000
	}

	if float64(currentReqMillSec)/1000 > m.stats.MaxSubmitDuration {
		m.stats.MaxSubmitDuration = float64(currentReqMillSec) / 1000
		m.stats.MaxSubmitDurationBlockHash = bh
	}
	m.stats.TotalSubmits++
	if txcount < 1 {
		m.stats.TotalTxEmptySubmits++
	}
}

func (m *Miner) StatsGbtRequest(currentReqMillSec int64, txcount int, longpollid string) {
	if len(m.stats.Lastest100GbtRequests) >= 100 {
		m.stats.Lastest100GbtRequests = m.stats.Lastest100GbtRequests[len(m.stats.Lastest100GbtRequests)-99:]
	}
	m.stats.Lastest100GbtRequests = append(m.stats.Lastest100GbtRequests, currentReqMillSec)
	sum := int64(0)
	for _, v := range m.stats.Lastest100GbtRequests {
		sum += v
	}
	m.stats.LastestGbtRequest = time.Now()
	m.stats.Lastest100GbtRequestAvgDuration = float64(sum) / float64(len(m.stats.Lastest100GbtRequests)) / 1000
	if m.stats.GbtRequestAvgDuration > 0 {
		m.stats.GbtRequestAvgDuration = (m.stats.GbtRequestAvgDuration + float64(currentReqMillSec)) / 2 / 1000
	} else {
		m.stats.GbtRequestAvgDuration = float64(currentReqMillSec) / 1000
	}
	if float64(currentReqMillSec)/1000 > m.stats.MaxGbtRequestDuration {
		m.stats.MaxGbtRequestDuration = float64(currentReqMillSec) / 1000
		m.stats.MaxGbtRequestTimeLongpollid = longpollid
	}
	if txcount < 1 {
		m.stats.TotalEmptyGbtResponse++
	}
}

func (m *Miner) StatsGbt(currentReqMillSec int64, txcount int) {
	if len(m.stats.Lastest100Gbts) >= 100 {
		m.stats.Lastest100Gbts = m.stats.Lastest100Gbts[len(m.stats.Lastest100Gbts)-99:]
	}
	m.stats.Lastest100Gbts = append(m.stats.Lastest100Gbts, currentReqMillSec)
	sum := int64(0)
	for _, v := range m.stats.Lastest100Gbts {
		sum += v
	}
	m.stats.LastestGbt = time.Now()
	m.stats.Lastest100GbtAvgDuration = float64(sum) / float64(len(m.stats.Lastest100Gbts)) / 1000
	if m.stats.GbtAvgDuration > 0 {
		m.stats.GbtAvgDuration = (m.stats.GbtAvgDuration + float64(currentReqMillSec)) / 2 / 1000
	} else {
		m.stats.GbtAvgDuration = float64(currentReqMillSec) / 1000
	}
	if float64(currentReqMillSec)/1000 > m.stats.MaxGbtDuration {
		m.stats.MaxGbtDuration = float64(currentReqMillSec) / 1000
	}
	if txcount < 1 {
		m.StatsEmptyGbt()
		m.stats.TotalEmptyGbts++
	} else {
		m.StatsGbtTxEmptyAvgTimes()
		m.stats.LastestMempoolEmptyTimestamp = 0
	}
}

func (m *Miner) Start() error {
	if !m.cfg.Miner {
		return nil
	}
	if err := m.Service.Start(); err != nil {
		return err
	}

	//
	log.Info("Start Miner...")

	m.subscribe()

	m.wg.Add(1)
	go m.handler()
	return nil
}

func (m *Miner) Stop() error {
	if !m.cfg.Miner {
		return nil
	}
	log.Info("try stop miner")
	if err := m.Service.Stop(); err != nil {
		return err
	}
	log.Info("Stop Miner...")

	close(m.quit)
	m.wg.Wait()
	m.reqWG.Wait()

	return nil
}

func (m *Miner) handler() {
	stallTicker := time.NewTicker(params.ActiveNetParams.TargetTimePerBlock)
	defer stallTicker.Stop()

out:
	for {
		select {
		case mc := <-m.msgChan:
			switch msg := mc.(type) {
			case *StartCPUMiningMsg:
				if m.worker != nil {
					if m.worker.GetType() == CPUWorkerType {
						continue
					}
					m.worker.Stop()
					m.worker = nil
				}
				m.worker = NewCPUWorker(m)
				if m.worker.Start() != nil {
					m.worker = nil
					continue
				}
				m.worker.Update()

			case *CPUMiningGenerateMsg:
				if msg.discreteNum <= 0 {
					if msg.block != nil {
						close(msg.block)
						msg.block = nil
					}
					continue
				}
				if m.worker != nil {
					if m.worker.GetType() == CPUWorkerType {
						if !m.worker.(*CPUWorker).generateDiscrete(msg.discreteNum, msg.block) {
							if msg.block != nil {
								close(msg.block)
								msg.block = nil
							}
						}
						if m.powType != msg.powType {
							m.powType = msg.powType
						}
						if m.updateBlockTemplate(true) == nil {
							m.worker.Update()
						} else {
							if msg.block != nil {
								close(msg.block)
								msg.block = nil
							}
						}
						continue
					}
					m.worker.Stop()
					m.worker = nil
				}
				worker := NewCPUWorker(m)
				m.worker = worker
				if m.worker.Start() != nil {
					m.worker = nil
					if msg.block != nil {
						close(msg.block)
						msg.block = nil
					}
					continue
				}
				if !worker.generateDiscrete(msg.discreteNum, msg.block) {
					if msg.block != nil {
						close(msg.block)
						msg.block = nil
					}
				}
				worker.Update()

			case *BlockChainChangeMsg:
				if m.updateBlockTemplate(false) == nil {
					if m.worker != nil {
						m.worker.Update()
					}
				}
			case *MempoolChangeMsg:
				// when mempool has changed
				// Speed up packing efficiency
				// recreate BlockTemplate when transactions is empty except coinbase tx
				if m.updateBlockTemplate(m.template != nil && len(m.template.Block.Transactions) <= 1) == nil {
					if m.worker != nil {
						m.worker.Update()
					}
				}

			case *GBTMiningMsg:
				if m.worker != nil {
					if m.worker.GetType() == GBTWorkerType {
						m.worker.(*GBTWorker).GetRequest(msg.request, msg.reply)
						continue
					}
					m.worker.Stop()
					m.worker = nil
				}
				worker := NewGBTWorker(m)
				m.worker = worker
				err := m.worker.Start()
				if err != nil {
					log.Error(err.Error())
					m.worker = nil
					if msg.reply != nil {
						msg.reply <- &gbtResponse{nil, err}
					}
					continue
				}
				worker.Update()
				worker.GetRequest(msg.request, msg.reply)

			case *RemoteMiningMsg:
				if m.worker != nil {
					if m.worker.GetType() == RemoteWorkerType {
						m.worker.(*RemoteWorker).GetRequest(msg.powType, msg.coinbaseFlags, msg.reply)
						continue
					}
					m.worker.Stop()
					m.worker = nil
				}
				worker := NewRemoteWorker(m)
				m.worker = worker
				err := m.worker.Start()
				if err != nil {
					log.Error(err.Error())
					m.worker = nil
					if msg.reply != nil {
						msg.reply <- &gbtResponse{nil, err}
					}
					continue
				}
				worker.Update()
				worker.GetRequest(msg.powType, msg.coinbaseFlags, msg.reply)

			default:
				log.Warn("Invalid message type in task handler: %T", msg)
			}

		case <-stallTicker.C:
			m.handleStallSample()

		case <-m.quit:
			break out
		}
	}

cleanup:
	for {
		select {
		case <-m.msgChan:
		default:
			break cleanup
		}
	}

	if m.worker != nil {
		m.worker.Stop()
	}

	m.wg.Done()
	log.Trace("Miner handler done")
}

func (m *Miner) updateBlockTemplate(force bool) error {

	reCreate := false
	//
	if force {
		reCreate = true
	} else if m.template == nil {
		reCreate = true
	}
	if !reCreate {
		hasCoinbaseAddr := m.coinbaseAddress != nil
		if hasCoinbaseAddr != m.template.ValidPayAddress {
			reCreate = true
		}
	}
	if !reCreate {
		parentsSet := meerdag.NewHashSet()
		parentsSet.AddList(m.consensus.BlockChain().GetMiningTips(meerdag.MaxPriority))

		tparentSet := meerdag.NewHashSet()
		tparentSet.AddList(m.template.Block.Parents)
		if !parentsSet.IsEqual(tparentSet) {
			reCreate = true
		} else {
			lastTxUpdate := m.txpool.LastUpdated()
			if lastTxUpdate.IsZero() {
				lastTxUpdate = roughtime.Now()
			}
			if lastTxUpdate != m.lastTxUpdate && roughtime.Now().After(m.lastTemplate.Add(time.Second*gbtRegenerateSeconds)) {
				reCreate = true
			}
		}
	}

	if reCreate {
		m.stats.TotalGbts++ //gbt generates
		start := time.Now().UnixMilli()
		template, err := mining.NewBlockTemplate(m.policy, params.ActiveNetParams.Params, m.sigCache, m.txpool, m.timeSource, m.consensus, m.coinbaseAddress, nil, m.powType, m.coinbaseFlags)
		if err != nil {
			e := fmt.Errorf("Failed to create new block template: %s", err.Error())
			log.Warn(e.Error())
			m.consensus.VMService().ResetTemplate()
			return e
		}
		m.template = template
		m.lastTxUpdate = m.txpool.LastUpdated()
		m.lastTemplate = time.Now()

		// Get the minimum allowed timestamp for the block based on the
		// median timestamp of the last several blocks per the chain
		// consensus rules.
		m.minTimestamp = mining.MinimumMedianTime(m.consensus.BlockChain().(*blockchain.BlockChain))
		m.StatsGbt(time.Now().UnixMilli()-start, len(template.Block.Transactions)-1) //exclude coinbase
		m.notifyBlockTemplate()
		return nil
	} else {
		err := mining.UpdateBlockTime(m.template.Block, m.BlockChain(), m.timeSource, params.ActiveNetParams.Params)
		if err != nil {
			log.Warn(fmt.Sprintf("%s unable to update block template time: %v", m.worker.GetType(), err))
			return err
		}
	}
	return nil
}

func (m *Miner) subscribe() {
	ch := make(chan *event.Event)
	sub := m.events.Subscribe(ch)
	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case ev := <-ch:
				if ev.Data != nil {
					switch value := ev.Data.(type) {
					case *blockchain.Notification:
						m.handleNotifyMsg(value)
					case int:
						if value == event.MempoolTxAdd {
							go m.MempoolChange()
						} else if value == event.Initialized {
							if m.cfg.Generate {
								m.StartCPUMining()
							}
						}
					}
				}
				if ev.Ack != nil {
					ev.Ack <- struct{}{}
				}
			case <-m.quit:
				log.Info("Close Miner Event Subscribe")
				return
			}
		}
	}()
}
func (m *Miner) handleNotifyMsg(notification *blockchain.Notification) {
	if m.worker == nil {
		return
	}
	switch notification.Type {
	case blockchain.BlockAccepted:
		band, ok := notification.Data.(*blockchain.BlockAcceptedNotifyData)
		if !ok {
			return
		}
		if band.IsMainChainTipChange {
			go m.BlockChainChange()
		}
	}
}

// submitBlock submits the passed block to network after ensuring it passes all
// of the consensus validation rules.
func (m *Miner) submitBlock(block *types.SerializedBlock) (interface{}, error) {
	if m.worker == nil {
		return nil, fmt.Errorf("You must enable miner by --miner.")
	}
	m.submitLocker.Lock()
	defer m.submitLocker.Unlock()
	m.totalSubmit++

	// Process this block using the same rules as blocks coming from other
	// nodes. This will in turn relay it to the network like normal.
	IsOrphan, IsTipsExpired, err := m.consensus.BlockChain().(*blockchain.BlockChain).ProcessBlock(block, blockchain.BFRPCAdd)
	if err != nil {
		if IsTipsExpired {
			go m.BlockChainChange()
		}
		// Anything other than a rule violation is an unexpected error,
		// so log that error as an internal error.
		rErr, ok := err.(blockchain.RuleError)
		if !ok {
			return nil, fmt.Errorf(fmt.Sprintf("Unexpected error while processing block submitted miner: %v (%s)", err, m.worker.GetType()))
		}
		// Occasionally errors are given out for timing errors with
		// ReduceMinDifficulty and high block works that is above
		// the target. Feed these to debug.
		if params.ActiveNetParams.Params.ReduceMinDifficulty &&
			rErr.ErrorCode == blockchain.ErrHighHash {
			return nil, fmt.Errorf(fmt.Sprintf("Block submitted via miner rejected "+
				"because of ReduceMinDifficulty time sync failure: %v (%s)",
				err, m.worker.GetType()))
		}
		// Other rule errors should be reported.
		return nil, fmt.Errorf(fmt.Sprintf("Block submitted via %s rejected: %v ", m.worker.GetType(), err))
	}
	if IsOrphan {
		return nil, fmt.Errorf(fmt.Sprintf("Block submitted via %s is an orphan building "+
			"on parent %v", m.worker.GetType(), block.Block().Header.ParentRoot))
	} else {
		m.txpool.PruneExpiredTx()
	}

	m.successSubmit++

	// The block was accepted.
	coinbaseTxOuts := block.Block().Transactions[0].TxOut
	coinbaseTxGenerated := uint64(0)
	for _, out := range coinbaseTxOuts {
		coinbaseTxGenerated += uint64(out.Amount.Value)
	}
	return json.SubmitBlockResult{
		BlockHash:      block.Hash().String(),
		CoinbaseTxID:   block.Transactions()[0].Hash().String(),
		Order:          meerdag.GetOrderLogStr(uint(block.Order())),
		Height:         int64(block.Height()),
		CoinbaseAmount: coinbaseTxGenerated,
		MinerType:      m.worker.GetType(),
	}, nil
}

func (m *Miner) submitBlockHeader(header *types.BlockHeader, extraNonce uint64) (interface{}, error) {
	if !m.IsEnable() || m.template == nil {
		return nil, fmt.Errorf("You must enable miner by --miner.")
	}
	tHeader := &m.template.Block.Header
	if !IsEqualForMiner(tHeader, header) {
		return nil, fmt.Errorf("You're overdue")
	}
	if extraNonce <= 0 {
		if !tHeader.TxRoot.IsEqual(&header.TxRoot) {
			return nil, fmt.Errorf("You're overdue about tx root.")
		}
	} else {
		ctx := types.NewTx(m.template.Block.Transactions[0]).Tx
		txRoot, err := mining.DoCalculateTransactionsRoot(ctx, m.template.TxMerklePath, m.template.TxWitnessRoot, extraNonce)
		if err != nil {
			return nil, err
		}
		if !txRoot.IsEqual(&header.TxRoot) {
			return nil, fmt.Errorf("You're overdue about tx root.")
		}
		tHeader.TxRoot = header.TxRoot
		m.template.Block.Transactions[0] = ctx
	}

	tHeader.Difficulty = header.Difficulty
	tHeader.Timestamp = header.Timestamp
	tHeader.Pow = header.Pow
	block := types.NewBlock(m.template.Block)
	block.SetHeight(uint(m.template.Height))
	return m.submitBlock(block)
}

func (m *Miner) CanMining() error {
	currentOrder := m.BlockChain().BestSnapshot().GraphState.GetTotal() - 1
	if currentOrder != 0 && !m.p2pSer.IsCurrent() {
		log.Trace("Client in initial download, qitmeer is downloading blocks...")
		return rpc.RPCClientInInitialDownloadError("Client in initial download ",
			"qitmeer is downloading blocks...")
	}
	return nil
}

func (m *Miner) IsEnable() bool {
	if !m.cfg.Miner {
		return false
	}
	if m.IsShutdown() {
		return false
	}
	if !m.IsStarted() {
		return false
	}
	return true
}

func (m *Miner) initCoinbase() error {
	if m.coinbaseAddress != nil {
		return nil
	}
	mAddrs := m.cfg.GetMinningAddrs()
	if len(mAddrs) <= 0 {
		// Respond with an error if there are no addresses to pay the
		// created blocks to.
		return fmt.Errorf("No payment addresses specified via --miningaddr.")
	}
	// Choose a payment address at random.
	if len(mAddrs) == 1 {
		m.coinbaseAddress = mAddrs[0]
	} else {
		m.coinbaseAddress = mAddrs[rand.Intn(len(mAddrs))]
	}
	if m.GetCoinbasePKAddress() != nil {
		log.Info(fmt.Sprintf("Init Coinbase PK Address:%s    PKH Address:%s", m.GetCoinbasePKAddress().String(), m.GetCoinbasePKAddress().PKHAddress().String()))
	} else {
		log.Info(fmt.Sprintf("Init Coinbase Address:%s", m.coinbaseAddress.String()))
	}

	return nil
}

func (m *Miner) GetCoinbasePKAddress() *address.SecpPubKeyAddress {
	pka, ok := m.coinbaseAddress.(*address.SecpPubKeyAddress)
	if ok {
		return pka
	}
	return nil
}

func (m *Miner) handleStallSample() {
	//if atomic.LoadInt32(&m.shutdown) != 0 {
	//	return
	//}
	//log.Debug("Miner stall sample")
}

func (m *Miner) StartCPUMining() {
	// Ignore if we are shutting down.
	if m.IsShutdown() {
		return
	}

	m.msgChan <- &StartCPUMiningMsg{}
}

func (m *Miner) CPUMiningGenerate(discreteNum int, block chan *hash.Hash, powType pow.PowType) error {
	if err := m.CanMining(); err != nil {
		return err
	}
	if !m.IsStarted() {
		if !m.cfg.Miner {
			m.cfg.Miner = true
		}
		if err := m.Start(); err != nil {
			log.Error(err.Error())
			return err
		}
	}
	// Ignore if we are shutting down.
	if m.IsShutdown() {
		return fmt.Errorf("Miner is quit")
	}
	m.msgChan <- &CPUMiningGenerateMsg{discreteNum: discreteNum, block: block, powType: powType}
	return nil
}

func (m *Miner) BlockChainChange() {
	// Ignore if we are shutting down.
	if m.IsShutdown() {
		return
	}
	if err := m.CanMining(); err != nil {
		return
	}

	m.msgChan <- &BlockChainChangeMsg{}
}

func (m *Miner) MempoolChange() {
	// Ignore if we are shutting down.
	if m.IsShutdown() {
		return
	}
	if m.worker == nil {
		return
	}
	if err := m.CanMining(); err != nil {
		return
	}
	m.msgChan <- &MempoolChangeMsg{}
}

func (m *Miner) GBTMining(request *json.TemplateRequest, reply chan *gbtResponse) error {
	if !m.IsStarted() {
		if !m.cfg.Miner {
			m.cfg.Miner = true
		}
		if err := m.Start(); err != nil {
			log.Error(err.Error())
			return err
		}
	}
	// Ignore if we are shutting down.
	if m.IsShutdown() {
		return fmt.Errorf("Miner is shutdown")
	}
	if err := m.CanMining(); err != nil {
		return err
	}

	m.msgChan <- &GBTMiningMsg{request: request, reply: reply}
	return nil
}

func (m *Miner) RemoteMining(powType pow.PowType, coinbaseFlags mining.CoinbaseFlags, reply chan *gbtResponse) error {
	if !m.cfg.Miner {
		return fmt.Errorf("Miner is disable. You can enable by --miner.")
	}
	// Ignore if we are shutting down.
	if m.IsShutdown() {
		return fmt.Errorf("Miner is shutdown")
	}
	if err := m.CanMining(); err != nil {
		return err
	}

	m.msgChan <- &RemoteMiningMsg{powType: powType, coinbaseFlags: coinbaseFlags, reply: reply}
	return nil
}

func (m *Miner) notifyBlockTemplate() {
	var err error
	var bt *json.RemoteGBTResult
	if m.RpcSer != nil {
		if m.worker.GetType() == RemoteWorkerType {
			bt = m.worker.(*RemoteWorker).GetRemoteGBTResult()
			if bt == nil {
				return
			}
			m.RpcSer.NotifyBlockTemplate(bt)
		}
	}
	if len(m.cfg.GBTNotify) <= 0 ||
		m.worker == nil {
		return
	}

	var jsonData []byte
	if m.worker.GetType() == RemoteWorkerType {
		if bt == nil {
			bt = m.worker.(*RemoteWorker).GetRemoteGBTResult()
		}
		jsonData, err = ejson.Marshal(bt)
		if err != nil {
			log.Error(err.Error())
		}
	}

	m.reqWG.Add(len(m.cfg.GBTNotify))
	for _, url := range m.cfg.GBTNotify {
		go m.sendNotification(url, jsonData)
	}
}

func (m *Miner) sendNotification(url string, jsonData []byte) {
	defer m.reqWG.Done()
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		log.Error(err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(m.Context(), NotifyURLTimeout)
	defer cancel()
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err.Error())
	} else {
		defer resp.Body.Close()
		log.Trace(fmt.Sprintf("Notified remote miner:%s %s", url, resp.Status))
	}
}

func (m *Miner) BlockChain() *blockchain.BlockChain {
	return m.consensus.BlockChain().(*blockchain.BlockChain)
}

func NewMiner(consensus model.Consensus, policy *mining.Policy, txpool *mempool.TxPool, p2pSer model.P2PService) *Miner {
	m := Miner{
		msgChan:       make(chan interface{}),
		quit:          make(chan struct{}),
		cfg:           consensus.Config(),
		policy:        policy,
		sigCache:      consensus.SigCache(),
		txpool:        txpool,
		timeSource:    consensus.MedianTimeSource(),
		powType:       pow.MEERXKECCAKV1,
		events:        consensus.Events(),
		coinbaseFlags: mining.CoinbaseFlagsStatic,
		consensus:     consensus,
		p2pSer:        p2pSer,
	}

	return &m
}

func IsEqualForMiner(header *types.BlockHeader, other *types.BlockHeader) bool {
	if header.Version != other.Version ||
		!header.ParentRoot.IsEqual(&other.ParentRoot) ||
		!header.StateRoot.IsEqual(&other.StateRoot) {
		return false
	}
	return true
}
