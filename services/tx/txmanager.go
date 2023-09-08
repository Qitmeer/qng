package tx

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/services/index"
	"github.com/Qitmeer/qng/services/mempool"
	"time"
)

type TxManager struct {
	service.Service
	indexManager *index.Manager

	// mempool hold tx that need to be mined into blocks and relayed to other peers.
	txMemPool *mempool.TxPool

	// notify
	ntmgr model.Notify

	// db
	db legacydb.DB

	//invalidTx hash->block hash
	invalidTx map[hash.Hash]*meerdag.HashSet

	// The fee estimator keeps track of how long transactions are left in
	// the mempool before they are mined into blocks.
	feeEstimator *mempool.FeeEstimator

	enableFeeEst bool

	consensus model.Consensus
}

func (tm *TxManager) Start() error {
	log.Info("Starting tx manager")
	if err := tm.Service.Start(); err != nil {
		return err
	}
	tm.LoadMempool()
	return tm.initFeeEstimator()
}

func (tm *TxManager) LoadMempool() {
	err := tm.txMemPool.Load()
	if err != nil {
		log.Error(err.Error())
	}
}

func (tm *TxManager) initFeeEstimator() error {
	if !tm.enableFeeEst {
		return nil
	}
	// Search for a FeeEstimator state in the database. If none can be found
	// or if it cannot be loaded, create a new one.
	tm.db.Update(func(tx legacydb.Tx) error {
		metadata := tx.Metadata()
		feeEstimationData := metadata.Get(mempool.EstimateFeeDatabaseKey)
		if feeEstimationData != nil {
			// delete it from the database so that we don't try to restore the
			// same thing again somehow.
			metadata.Delete(mempool.EstimateFeeDatabaseKey)

			// If there is an error, log it and make a new fee estimator.
			var err error
			tm.feeEstimator, err = mempool.RestoreFeeEstimator(feeEstimationData)

			if err != nil {
				log.Error(fmt.Sprintf("Failed to restore fee estimator %v", err))
			}
		}

		return nil
	})

	// If no feeEstimator has been found, or if the one that has been found
	// is behind somehow, create a new one and start over.
	if tm.feeEstimator == nil || tm.feeEstimator.LastKnownHeight() != int32(tm.GetChain().BestSnapshot().GraphState.GetMainHeight()) {
		tm.feeEstimator = mempool.NewFeeEstimator(
			mempool.DefaultEstimateFeeMaxRollback,
			mempool.DefaultEstimateFeeMinRegisteredBlocks)
	}

	tm.txMemPool.GetConfig().FeeEstimator = tm.feeEstimator
	return nil
}

func (tm *TxManager) Stop() error {
	log.Info("Stopping tx manager")
	if err := tm.Service.Stop(); err != nil {
		return err
	}
	if tm.txMemPool.IsPersist() {
		num, err := tm.txMemPool.Save()
		if err != nil {
			log.Error(err.Error())
		} else {
			log.Info(fmt.Sprintf("Mempool persist:%d transactions", num))
		}
	}

	if tm.feeEstimator != nil {
		// Save fee estimator state in the database.
		tm.db.Update(func(tx legacydb.Tx) error {
			metadata := tx.Metadata()
			metadata.Put(mempool.EstimateFeeDatabaseKey, tm.feeEstimator.Save())

			return nil
		})
	}

	return nil
}

func (tm *TxManager) MemPool() model.TxPool {
	return tm.txMemPool
}

func (tm *TxManager) FeeEstimator() model.FeeEstimator {
	if tm.feeEstimator != nil {
		return tm.feeEstimator
	}
	return nil
}

func (tm *TxManager) InitDefaultFeeEstimator() {
	tm.feeEstimator = mempool.NewFeeEstimator(
		mempool.DefaultEstimateFeeMaxRollback,
		mempool.DefaultEstimateFeeMinRegisteredBlocks)
}

func (tm *TxManager) handleNotifyMsg(notification *blockchain.Notification) {
	switch notification.Type {
	case blockchain.BlockConnected:
		blockSlice, ok := notification.Data.([]interface{})
		if !ok {
			log.Warn("Chain connected notification is not a block slice.")
			break
		}

		if len(blockSlice) != 3 {
			log.Warn("Chain connected notification is wrong size slice.")
			break
		}

		block := blockSlice[0].(*types.SerializedBlock)
		txds := []*types.TxDesc{}
		for _, tx := range block.Transactions()[1:] {
			if tm.IsShutdown() {
				return
			}
			tm.MemPool().RemoveTransaction(tx, false)
			tm.MemPool().RemoveDoubleSpends(tx)
			tm.MemPool().RemoveOrphan(tx.Hash())
			tm.ntmgr.TransactionConfirmed(tx)
			acceptedTxs := tm.MemPool().ProcessOrphans(tx.Hash())
			txds = append(txds, acceptedTxs...)
		}
		tm.ntmgr.AnnounceNewTransactions(txds, nil)
		// Register block with the fee estimator, if it exists.
		if tm.FeeEstimator() != nil && blockSlice[1].(bool) {
			err := tm.FeeEstimator().RegisterBlock(block, blockSlice[2].(meerdag.IBlock).GetHeight())

			// If an error is somehow generated then the fee estimator
			// has entered an invalid state. Since it doesn't know how
			// to recover, create a new one.
			if err != nil {
				tm.InitDefaultFeeEstimator()
			}
		}
	case blockchain.BlockDisconnected:
		blockSlice, ok := notification.Data.([]interface{})
		if !ok {
			log.Warn("Chain disconnected notification is not a block slice.")
			break
		}
		// Rollback previous block recorded by the fee estimator.
		if tm.FeeEstimator() != nil {
			tm.FeeEstimator().Rollback(blockSlice[0].(*types.SerializedBlock).Hash())
		}
	}
}

func (tm *TxManager) GetChain() *blockchain.BlockChain {
	return tm.consensus.BlockChain().(*blockchain.BlockChain)
}

func NewTxManager(consensus model.Consensus, ntmgr model.Notify) (*TxManager, error) {
	cfg := consensus.Config()
	sigCache := consensus.SigCache()
	bc := consensus.BlockChain().(*blockchain.BlockChain)
	// mem-pool
	amt, _ := types.NewMeer(uint64(cfg.MinTxFee))
	txC := mempool.Config{
		Policy: mempool.Policy{
			MaxTxVersion:         2,
			DisableRelayPriority: cfg.NoRelayPriority,
			AcceptNonStd:         cfg.AcceptNonStd,
			FreeTxRelayLimit:     cfg.FreeTxRelayLimit,
			MaxOrphanTxs:         cfg.MaxOrphanTxs,
			MaxOrphanTxSize:      mempool.DefaultMaxOrphanTxSize,
			MaxSigOpsPerTx:       blockchain.MaxSigOpsPerBlock / 5,
			MaxTxSize:            int64(cfg.BlockMaxSize - types.MaxBlockHeaderPayload),
			MinRelayTxFee:        *amt,
			TxTimeScope:          cfg.TxTimeScope,
			StandardVerifyFlags: func() (txscript.ScriptFlags, error) {
				return mempool.StandardScriptVerifyFlags()
			},
		},
		ChainParams:      consensus.Params(),
		FetchUtxoView:    bc.FetchUtxoView, //TODO, duplicated dependence of miner
		BlockByHash:      bc.FetchBlockByHash,
		BestHash:         func() *hash.Hash { return &bc.BestSnapshot().Hash },
		BestHeight:       func() uint64 { return uint64(bc.BestSnapshot().GraphState.GetMainHeight()) },
		CalcSequenceLock: bc.CalcSequenceLock,
		SubsidyCache:     bc.FetchSubsidyCache(),
		SigCache:         sigCache,
		PastMedianTime:   func() time.Time { return bc.BestSnapshot().MedianTime },
		IndexManager:     consensus.IndexManager().(*index.Manager),
		BC:               bc,
		DataDir:          cfg.DataDir,
		Expiry:           time.Duration(cfg.MempoolExpiry),
		Persist:          cfg.Persistmempool,
		NoMempoolBar:     cfg.NoMempoolBar,
		Events:           consensus.Events(),
	}
	txMemPool := mempool.New(&txC)
	invalidTx := make(map[hash.Hash]*meerdag.HashSet)
	tm := &TxManager{
		consensus:    consensus,
		indexManager: consensus.IndexManager().(*index.Manager),
		txMemPool:    txMemPool,
		ntmgr:        ntmgr,
		db:           consensus.LegacyDB(),
		invalidTx:    invalidTx,
		enableFeeEst: cfg.Estimatefee}
	consensus.BlockChain().(*blockchain.BlockChain).Subscribe(tm.handleNotifyMsg)
	return tm, nil
}
