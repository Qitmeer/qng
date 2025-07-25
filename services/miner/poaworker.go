package miner

import (
	"fmt"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/ethereum/go-ethereum/common"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Qitmeer/qng/v2/core/types"
)

type PoAWorker struct {
	started  int32
	shutdown int32
	wg       sync.WaitGroup
	quit     chan struct{}

	miner *Miner
	sync.Mutex
}

func (w *PoAWorker) GetType() string {
	return PoAWorkerType
}

func (w *PoAWorker) Start() error {
	err := w.miner.initCoinbase()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if w.miner.BlockChain().DagPoA().Signer() == (common.Address{}) {
		return fmt.Errorf("DagPoA signer address is empty")
	}
	// Already started?
	if atomic.AddInt32(&w.started, 1) != 1 {
		return nil
	}

	log.Info("Start PoA Worker", "address", w.miner.BlockChain().DagPoA().Signer().String())

	w.wg.Add(1)
	go w.generateBlocks()

	return nil
}

func (w *PoAWorker) Stop() {
	if atomic.AddInt32(&w.shutdown, 1) != 1 {
		log.Warn(fmt.Sprintf("PoA Worker is already in the process of shutting down"))
		return
	}
	log.Info("Stop PoA Worker...")

	close(w.quit)
	w.wg.Wait()
}

func (w *PoAWorker) IsRunning() bool {
	return atomic.LoadInt32(&w.started) != 0
}

func (w *PoAWorker) Update() {
}

func (w *PoAWorker) generateBlocks() {
	log.Info(fmt.Sprintf("Starting generate blocks worker:%s", w.GetType()))
	period := time.Duration(params.ActiveNetParams.ToPoAConfig().Period) * time.Second
out:
	for {
		// Quit when the miner is stopped.
		select {
		case <-w.quit:
			break out
		default:
			// Non-blocking select to fall through
		}
		start := time.Now()
		err := w.miner.CanMining()
		if err != nil {
			log.Warn(err.Error())
			time.Sleep(period)
			continue
		}

		err = w.miner.updateBlockTemplate(false)
		if err != nil {
			log.Warn(err.Error())
			time.Sleep(period)
			continue
		}

		sb := w.solveBlock()
		if sb != nil {
			block := types.NewBlock(sb)
			startSB := time.Now()
			info, err := w.miner.submitBlock(block)
			if err != nil {
				if !strings.Contains(err.Error(), "expired") {
					log.Error(fmt.Sprintf("Failed to submit new block:%s ,%v", block.Hash().String(), err))
				} else {
					log.Warn(fmt.Sprintf("Failed to submit new block:%s ,%v", block.Hash().String(), err))
				}
				continue
			} else {
				w.miner.StatsSubmit(startSB, block.Block().BlockHash().String(), len(block.Block().Transactions)-1)
			}
			log.Info(fmt.Sprintf("%v", info), "cost", time.Since(start).String(), "txs", len(block.Transactions()))

		}
		left := time.Since(start)
		if left <= period {
			time.Sleep(period - left)
		}
	}

	w.wg.Done()
	log.Info(fmt.Sprintf("Generate blocks worker done:%s", w.GetType()))
}

func (w *PoAWorker) solveBlock() *types.Block {
	if w.miner.template == nil {
		return nil
	}
	// Create a couple of convenience variables.
	block, err := w.miner.template.Block.Clone()
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	err = w.miner.BlockChain().Seal(block)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return block
}

func NewPoAWorker(miner *Miner) *PoAWorker {
	w := PoAWorker{
		quit:  make(chan struct{}),
		miner: miner,
	}

	return &w
}
