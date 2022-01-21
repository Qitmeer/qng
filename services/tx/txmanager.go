package tx

import (
	"fmt"
	"github.com/Qitmeer/qng-core/common/hash"
	"github.com/Qitmeer/qng-core/config"
	"github.com/Qitmeer/qng-core/consensus"
	"github.com/Qitmeer/qng-core/core/event"
	"github.com/Qitmeer/qng-core/core/types"
	"github.com/Qitmeer/qng-core/database"
	"github.com/Qitmeer/qng-core/engine/txscript"
	"github.com/Qitmeer/qng-core/meerdag"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/services/blkmgr"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/services/index"
	"github.com/Qitmeer/qng/services/mempool"
	"time"
)

type TxManager struct {
	service.Service

	bm *blkmgr.BlockManager
	// tx index
	txIndex *index.TxIndex

	// addr index
	addrIndex *index.AddrIndex
	// mempool hold tx that need to be mined into blocks and relayed to other peers.
	txMemPool *mempool.TxPool

	// notify
	ntmgr consensus.Notify

	// db
	db database.DB

	//invalidTx hash->block hash
	invalidTx map[hash.Hash]*meerdag.HashSet

	// The fee estimator keeps track of how long transactions are left in
	// the mempool before they are mined into blocks.
	feeEstimator *mempool.FeeEstimator

	enableFeeEst bool
}

func (tm *TxManager) Start() error {
	log.Info("Starting tx manager")
	if err := tm.Service.Start(); err != nil {
		return err
	}

	err := tm.txMemPool.Load()
	if err != nil {
		log.Error(err.Error())
	}
	return tm.initFeeEstimator()
}

func (tm *TxManager) initFeeEstimator() error {
	if !tm.enableFeeEst {
		return nil
	}
	// Search for a FeeEstimator state in the database. If none can be found
	// or if it cannot be loaded, create a new one.
	tm.db.Update(func(tx database.Tx) error {
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
	if tm.feeEstimator == nil || tm.feeEstimator.LastKnownHeight() != int32(tm.bm.GetChain().BestSnapshot().GraphState.GetMainHeight()) {
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
		tm.db.Update(func(tx database.Tx) error {
			metadata := tx.Metadata()
			metadata.Put(mempool.EstimateFeeDatabaseKey, tm.feeEstimator.Save())

			return nil
		})
	}

	return nil
}

func (tm *TxManager) MemPool() consensus.TxPool {
	return tm.txMemPool
}

func (tm *TxManager) FeeEstimator() consensus.FeeEstimator {
	return tm.feeEstimator
}

func (tm *TxManager) InitDefaultFeeEstimator() {
	tm.feeEstimator = mempool.NewFeeEstimator(
		mempool.DefaultEstimateFeeMaxRollback,
		mempool.DefaultEstimateFeeMinRegisteredBlocks)
}

func NewTxManager(bm *blkmgr.BlockManager, txIndex *index.TxIndex,
	addrIndex *index.AddrIndex, cfg *config.Config, ntmgr consensus.Notify,
	sigCache *txscript.SigCache, db database.DB, events *event.Feed) (*TxManager, error) {
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
			MinRelayTxFee:        *amt,
			StandardVerifyFlags: func() (txscript.ScriptFlags, error) {
				return common.StandardScriptVerifyFlags()
			},
		},
		ChainParams:      bm.ChainParams(),
		FetchUtxoView:    bm.GetChain().FetchUtxoView, //TODO, duplicated dependence of miner
		BlockByHash:      bm.GetChain().FetchBlockByHash,
		BestHash:         func() *hash.Hash { return &bm.GetChain().BestSnapshot().Hash },
		BestHeight:       func() uint64 { return uint64(bm.GetChain().BestSnapshot().GraphState.GetMainHeight()) },
		CalcSequenceLock: bm.GetChain().CalcSequenceLock,
		SubsidyCache:     bm.GetChain().FetchSubsidyCache(),
		SigCache:         sigCache,
		PastMedianTime:   func() time.Time { return bm.GetChain().BestSnapshot().MedianTime },
		AddrIndex:        addrIndex,
		BD:               bm.GetChain().BlockDAG(),
		BC:               bm.GetChain(),
		DataDir:          cfg.DataDir,
		Expiry:           time.Duration(cfg.MempoolExpiry),
		Persist:          cfg.Persistmempool,
		NoMempoolBar:     cfg.NoMempoolBar,
		Events:           events,
	}
	txMemPool := mempool.New(&txC)
	invalidTx := make(map[hash.Hash]*meerdag.HashSet)
	return &TxManager{bm: bm, txIndex: txIndex, addrIndex: addrIndex, txMemPool: txMemPool, ntmgr: ntmgr, db: db, invalidTx: invalidTx, enableFeeEst: cfg.Estimatefee}, nil
}
