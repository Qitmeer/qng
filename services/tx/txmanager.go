package tx

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/message"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/services/index"
	"github.com/Qitmeer/qng/services/mempool"
	vmconsensus "github.com/Qitmeer/qng/vm/consensus"
	"time"
)

type TxManager struct {
	service.Service
	indexManager *index.Manager

	// mempool hold tx that need to be mined into blocks and relayed to other peers.
	txMemPool *mempool.TxPool

	// notify
	ntmgr vmconsensus.Notify

	// db
	db database.DB

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
		tm.db.Update(func(tx database.Tx) error {
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

		if len(blockSlice) != 2 {
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
			err := tm.FeeEstimator().RegisterBlock(block)

			// If an error is somehow generated then the fee estimator
			// has entered an invalid state. Since it doesn't know how
			// to recover, create a new one.
			if err != nil {
				tm.InitDefaultFeeEstimator()
			}
		}
	case blockchain.BlockDisconnected:
		block, ok := notification.Data.(*types.SerializedBlock)
		if !ok {
			log.Warn("Chain disconnected notification is not a block slice.")
			break
		}
		// Rollback previous block recorded by the fee estimator.
		if tm.FeeEstimator() != nil {
			tm.FeeEstimator().Rollback(block.Hash())
		}
	}
}

func (tm *TxManager) GetChain() *blockchain.BlockChain {
	return tm.consensus.BlockChain().(*blockchain.BlockChain)
}

func NewTxManager(consensus model.Consensus, ntmgr vmconsensus.Notify) (*TxManager, error) {
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
				return common.StandardScriptVerifyFlags()
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
		db:           consensus.DatabaseContext(),
		invalidTx:    invalidTx,
		enableFeeEst: cfg.Estimatefee}
	consensus.BlockChain().(*blockchain.BlockChain).Subscribe(tm.handleNotifyMsg)
	return tm, nil
}

func (tm *TxManager) ProcessRawTx(serializedTx []byte, highFees bool) (string, error) {
	msgtx := types.NewTransaction()
	err := msgtx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return "", rpc.RpcDeserializationError("Could not decode Tx: %v",
			err)
	}

	tx := types.NewTx(msgtx)
	for _, v := range msgtx.TxIn {
		fmt.Println(v.PreviousOut.Hash)
		fmt.Println(v.PreviousOut.OutIndex)
	}
	acceptedTxs, err := tm.txMemPool.ProcessTransaction(tx, false,
		false, highFees)
	if err != nil {
		// When the error is a rule error, it means the transaction was
		// simply rejected as opposed to something actually going
		// wrong, so log it as such.  Otherwise, something really did
		// go wrong, so log it as an actual error.  In both cases, a
		// JSON-RPC error is returned to the client with the
		// deserialization error code (to match bitcoind behavior).
		if _, ok := err.(mempool.RuleError); ok {
			err = fmt.Errorf("Rejected transaction %v: %v", tx.Hash(),
				err)
			log.Error("Failed to process transaction", "mempool.RuleError", err)
			txRuleErr, ok := err.(mempool.TxRuleError)
			if ok {
				if txRuleErr.RejectCode == message.RejectDuplicate {
					// return a dublicate tx error
					return "", rpc.RpcDuplicateTxError("%v", err)
				}
			}

			// return a generic rule error
			return "", rpc.RpcRuleError("%v", err)
		}

		log.Error("Failed to process transaction", "err", err)
		err = fmt.Errorf("failed to process transaction %v: %v",
			tx.Hash(), err)
		return "", rpc.RpcDeserializationError("rejected: %v", err)
	}
	tm.ntmgr.AnnounceNewTransactions(acceptedTxs, nil)
	tm.ntmgr.AddRebroadcastInventory(acceptedTxs)
	return tx.Hash().String(), nil
}

func (tm *TxManager) CreateRawTransactionV2(inputs []json.TransactionInput,
	amounts json.AdreesAmount, lockTime *int64) (interface{}, error) {

	// Validate the locktime, if given.
	if lockTime != nil &&
		(*lockTime < 0 || *lockTime > int64(types.MaxTxInSequenceNum)) {
		return nil, rpc.RpcInvalidError("Locktime out of range")
	}

	// Add all transaction inputs to a new transaction after performing
	// some validity checks.
	mtx := types.NewTransaction()
	for _, input := range inputs {
		txHash, err := hash.NewHashFromStr(input.Txid)
		if err != nil {
			return nil, rpc.RpcDecodeHexError(input.Txid)
		}
		prevOut := types.NewOutPoint(txHash, input.Vout)
		txIn := types.NewTxInput(prevOut, []byte{})
		if lockTime != nil && *lockTime != 0 {
			txIn.Sequence = types.MaxTxInSequenceNum - 1
		}
		mtx.AddTxIn(txIn)
	}

	// Add all transaction outputs to the transaction after performing
	// some validity checks.
	for encodedAddr, amount := range amounts {
		// Ensure amount is in the valid range for monetary amounts.
		if amount.Amount <= 0 || amount.Amount > types.MaxAmount {
			return nil, rpc.RpcInvalidError("Invalid amount: 0 >= %v "+
				"> %v", amount, types.MaxAmount)
		}

		err := types.CheckCoinID(types.CoinID(amount.CoinId))
		if err != nil {
			return nil, rpc.RpcInvalidError(err.Error())
		}
		// Decode the provided address.
		addr, err := address.DecodeAddress(encodedAddr)
		if err != nil {
			return nil, rpc.RpcAddressKeyError("Could not decode "+
				"address: %v", err)
		}

		// Ensure the address is one of the supported types and that
		// the network encoded with the address matches the network the
		// server is currently on.
		switch addr.(type) {
		case *address.PubKeyHashAddress:
		case *address.ScriptHashAddress:
		case *address.SecpPubKeyAddress:
		default:
			return nil, rpc.RpcAddressKeyError("Invalid type: %T", addr)
		}
		if !address.IsForNetwork(addr, tm.consensus.Params()) {
			return nil, rpc.RpcAddressKeyError("Wrong network: %v",
				addr)
		}

		// Create a new script which pays to the provided address.
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, rpc.RpcInternalError(err.Error(),
				"Pay to address script")
		}

		txOut := types.NewTxOutput(types.Amount{Value: amount.Amount, Id: types.CoinID(amount.CoinId)}, pkScript)
		mtx.AddTxOut(txOut)
	}

	// Set the Locktime, if given.
	if lockTime != nil {
		mtx.LockTime = uint32(*lockTime)
	}

	// Return the serialized and hex-encoded transaction.  Note that this
	// is intentionally not directly returning because the first return
	// value is a string and it would result in returning an empty string to
	// the client instead of nothing (nil) in the case of an error.
	mtxHex, err := marshal.MessageToHex(mtx)
	if err != nil {
		return nil, err
	}
	return mtxHex, nil
}
