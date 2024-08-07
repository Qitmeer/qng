// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2017-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package mempool

import (
	"container/list"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	mmeer "github.com/Qitmeer/qng/consensus/model/meer"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/message"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerevm/meer"
)

// TxPool is used as a source of transactions that need to be mined into blocks
// and relayed to other peers.  It is safe for concurrent access from multiple
// peers.
type TxPool struct {
	// The following variables must only be used atomically.
	lastUpdated int64 // last time pool was updated.

	procmtx sync.RWMutex
	cfg     Config

	mtx       sync.RWMutex
	pool      map[hash.Hash]*TxDesc
	outpoints map[types.TxOutPoint]*types.Tx

	orphansmtx    sync.RWMutex
	orphans       map[hash.Hash]*types.Tx
	orphansByPrev map[hash.Hash]map[hash.Hash]*types.Tx

	pennyTotal    float64 // exponentially decaying total for penny spends.
	lastPennyUnix int64   // unix time of last ``penny spend''

	dirty      atomic.Bool
	pruneDirty atomic.Bool
}

// New returns a new memory pool for validating and storing standalone
// transactions until they are mined into a block.
func New(cfg *Config) *TxPool {
	tp := &TxPool{
		cfg:           *cfg,
		pool:          make(map[hash.Hash]*TxDesc),
		orphans:       make(map[hash.Hash]*types.Tx),
		orphansByPrev: make(map[hash.Hash]map[hash.Hash]*types.Tx),
		outpoints:     make(map[types.TxOutPoint]*types.Tx),
	}
	tp.pruneDirty.Store(false)
	return tp
}

// TxDesc is a descriptor containing a transaction in the mempool along with
// additional metadata.
type TxDesc struct {
	types.TxDesc

	// StartingPriority is the priority of the transaction when it was added
	// to the pool.
	StartingPriority float64
}

// TxDescs returns a slice of descriptors for all the transactions in the pool.
// The descriptors are to be treated as read only.
//
// This function is safe for concurrent access.
func (mp *TxPool) TxDescs(all bool) []*types.TxDesc {
	mp.mtx.RLock()
	descs := []*types.TxDesc{}
	for _, desc := range mp.pool {
		descs = append(descs, &desc.TxDesc)
	}
	mp.mtx.RUnlock()
	if !all {
		return descs
	}
	etxs, _, err := mp.cfg.BC.MeerChain().(*meer.MeerChain).MeerPool().GetTxs()
	if err != nil {
		log.Error(err.Error())
		return descs
	}

	for _, tx := range etxs {
		txDesc := &types.TxDesc{
			Tx:     tx,
			Added:  roughtime.Now(),
			Height: mp.GetMainHeight(),
		}
		descs = append(descs, txDesc)
	}

	return descs
}

// removeTransaction is the internal function which implements the public
// RemoveTransaction.  See the comment for RemoveTransaction for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeTransaction(theTx *types.Tx, removeRedeemers bool) {
	tx := theTx.Transaction()
	txHash := theTx.Hash()
	if removeRedeemers {
		// Remove any transactions which rely on this one.
		for i := uint32(0); i < uint32(len(tx.TxOut)); i++ {
			outpoint := types.NewOutPoint(txHash, i)
			mp.mtx.RLock()
			txRedeemer, exists := mp.outpoints[*outpoint]
			mp.mtx.RUnlock()
			if exists {
				mp.removeTransaction(txRedeemer, true)
			}
		}
	}

	// Remove the transaction if needed.
	mp.mtx.RLock()
	txDesc, exists := mp.pool[*txHash]
	mp.mtx.RUnlock()
	if exists {
		// Remove unconfirmed address index entries associated with the
		// transaction if enabled.
		// TODO address index
		if aIndex := mp.cfg.IndexManager.AddrIndex(); aIndex != nil {
			aIndex.RemoveUnconfirmedTx(txHash)
		}
		// Mark the referenced outpoints as unspent by the pool.
		mp.mtx.Lock()
		for _, txIn := range txDesc.Tx.Transaction().TxIn {
			delete(mp.outpoints, txIn.PreviousOut)
		}
		delete(mp.pool, *txHash)
		mp.mtx.Unlock()

		// stats daily tx count
		if mp.LastUpdated().Day() != time.Now().Day() {
			newDailyTxCount.Update(1)
		}
		atomic.StoreInt64(&mp.lastUpdated, roughtime.Now().Unix())
		log.Trace(fmt.Sprintf("TxPool:remove tx %s", txHash))
	}
}

// RemoveTransaction removes the passed transaction from the mempool. When the
// removeRedeemers flag is set, any transactions that redeem outputs from the
// removed transaction will also be removed recursively from the mempool, as
// they would otherwise become orphans.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveTransaction(tx *types.Tx, removeRedeemers bool) {
	// Protect concurrent access.
	if opreturn.IsMeerEVMTx(tx.Tx) {
		if mp.cfg.BC.IsShutdown() {
			return
		}
		err := mp.cfg.BC.MeerChain().(*meer.MeerChain).MeerPool().RemoveTx(tx)
		if err != nil {
			log.Error(err.Error())
		}
	} else {
		mp.procmtx.Lock()
		defer mp.procmtx.Unlock()
		mp.removeTransaction(tx, removeRedeemers)
	}
}

// RemoveDoubleSpends removes all transactions which spend outputs spent by the
// passed transaction from the memory pool.  Removing those transactions then
// leads to removing all transactions which rely on them, recursively.  This is
// necessary when a block is connected to the main chain because the block may
// contain transactions which were previously unknown to the memory pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveDoubleSpends(tx *types.Tx) {
	if types.IsCrossChainVMTx(tx.Tx) ||
		types.IsCrossChainImportTx(tx.Tx) ||
		types.IsTokenNewTx(tx.Tx) ||
		types.IsTokenRenewTx(tx.Tx) ||
		types.IsTokenInvalidateTx(tx.Tx) ||
		types.IsTokenValidateTx(tx.Tx) {
		return
	}
	// Protect concurrent access.
	mp.procmtx.Lock()
	for _, txIn := range tx.Transaction().TxIn {
		mp.mtx.RLock()
		txRedeemer, ok := mp.outpoints[txIn.PreviousOut]
		mp.mtx.RUnlock()
		if ok {
			if !txRedeemer.Hash().IsEqual(tx.Hash()) {
				mp.removeTransaction(txRedeemer, true)
			}
		}
	}
	mp.procmtx.Unlock()
}

// addTransaction adds the passed transaction to the memory pool.  It should
// not be called directly as it doesn't perform any validation.  This is a
// helper for maybeAcceptTransaction.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) addTransaction(utxoView *utxo.UtxoViewpoint,
	tx *types.Tx, height uint64, fee int64) *TxDesc {
	// Add the transaction to the pool and mark the referenced outpoints
	// as spent by the pool.
	msgTx := tx.Transaction()
	txD := &TxDesc{
		TxDesc: types.TxDesc{
			Tx:       tx,
			Added:    roughtime.Now(),
			Height:   int64(height), //todo: fix type conversion
			Fee:      fee,
			FeePerKB: fee * 1000 / int64(tx.Tx.SerializeSize()),
		},
	}

	if utxoView != nil {
		txD.StartingPriority = CalcPriority(msgTx, utxoView, height, mp.cfg.BC)
	}

	if !types.IsCrossChainVMTx(tx.Tx) {
		mp.mtx.Lock()
		mp.pool[*tx.Hash()] = txD
		mp.mtx.Unlock()
	}

	if !types.IsCrossChainVMTx(tx.Tx) &&
		!types.IsCrossChainImportTx(tx.Tx) &&
		!types.IsTokenNewTx(tx.Tx) &&
		!types.IsTokenRenewTx(tx.Tx) &&
		!types.IsTokenInvalidateTx(tx.Tx) &&
		!types.IsTokenValidateTx(tx.Tx) {

		mp.mtx.Lock()
		for txIdx, txIn := range msgTx.TxIn {
			if txIdx == 0 && (types.IsTokenMintTx(tx.Tx) || types.IsTokenUnmintTx(tx.Tx)) {
				continue
			}
			mp.outpoints[txIn.PreviousOut] = tx
		}
		mp.mtx.Unlock()
	}
	if mp.LastUpdated().Day() == time.Now().Day() {
		newDailyTxCount.Inc(1)
	} else {
		newDailyTxCount.Update(1)
	}

	atomic.StoreInt64(&mp.lastUpdated, roughtime.Now().Unix())

	// Add unconfirmed address index entries associated with the transaction
	// if enabled.
	if aIndex := mp.cfg.IndexManager.AddrIndex(); aIndex != nil && utxoView != nil {
		mp.AddUnconfirmedTx(tx, utxoView)
	}

	// Record this tx for fee estimation if enabled.
	if mp.cfg.FeeEstimator != nil {
		mp.cfg.FeeEstimator.ObserveTransaction(txD)
	}

	mp.dirty.Store(true)

	return txD
}

func (mp *TxPool) TriggerDirty() {
	if mp.LastUpdated().Day() == time.Now().Day() {
		newDailyTxCount.Inc(1)
	} else {
		newDailyTxCount.Update(1)
	}

	atomic.StoreInt64(&mp.lastUpdated, roughtime.Now().Unix())

	mp.dirty.Store(true)
}

// Call addTransaction
func (mp *TxPool) AddTransaction(tx *types.Tx, height uint64, fee int64) {
	mp.addTransaction(nil, tx, height, fee)
}

// maybeAcceptTransaction is the internal function which implements the public
// MaybeAcceptTransaction.  See the comment for MaybeAcceptTransaction for
// more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) maybeAcceptTransaction(tx *types.Tx, isNew, rateLimit, allowHighFees bool) ([]*hash.Hash, *TxDesc, error) {
	msgTx := tx.Transaction()
	txHash := tx.Hash()
	start := time.Now()
	if mp.LastUpdated().Day() == time.Now().Day() {
		newDailyAllTxCount.Inc(1)
	} else {
		newDailyAllTxCount.Update(1)
	}
	// Don't accept the transaction if it already exists in the pool.  This
	// applies to orphan transactions as well.  This check is intended to
	// be a quick check to weed out duplicates.
	if mp.HaveTransaction(txHash) {
		str := fmt.Sprintf("already have transaction %v", txHash)
		return nil, nil, txRuleError(message.RejectDuplicate, str)
	}

	if !mp.cfg.BC.IsValidTxType(types.DetermineTxType(tx.Tx)) {
		str := fmt.Sprintf("%s is not support transaction type.", types.DetermineTxType(tx.Tx).String())
		return nil, nil, txRuleError(message.RejectNonstandard, str)
	}
	// Perform preliminary sanity checks on the transaction.  This makes
	// use of chain which contains the invariant rules for what
	// transactions are allowed into blocks.
	err := blockchain.CheckTransactionSanity(tx, mp.cfg.ChainParams, nil, mp.cfg.BC)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}
	mempoolCheckTransactionSanity.Update(time.Now().Sub(start))
	start = time.Now()
	// A standalone transaction must not be a coinbase transaction.
	if tx.Tx.IsCoinBase() {
		str := fmt.Sprintf("transaction %v is an individual coinbase",
			txHash)
		return nil, nil, txRuleError(message.RejectInvalid, str)
	}

	// Don't accept transactions with a lock time after the maximum int32
	// value for now.  This is an artifact of older bitcoind clients which
	// treated this field as an int32 and would treat anything larger
	// incorrectly (as negative).
	if msgTx.LockTime > math.MaxInt32 {
		str := fmt.Sprintf("transaction %v has a lock time after "+
			"2038 which is not accepted yet", txHash)
		return nil, nil, txRuleError(message.RejectNonstandard, str)
	}

	// Get the current height of the main chain.  A standalone transaction
	// will be mined into the next block at best, so its height is at least
	// one more than the current height.
	nextBlockHeight := mp.cfg.BestHeight() + 1

	// Don't accept transactions that will be expired as of the next block.
	if blockchain.IsExpired(tx, nextBlockHeight) {
		str := fmt.Sprintf("transaction %v expired at height %d",
			txHash, msgTx.Expire)
		return nil, nil, txRuleError(message.RejectInvalid, str)
	}

	if !tx.Tx.ValidTime(mp.cfg.Policy.TxTimeScope) {
		str := fmt.Sprintf("transaction %v with tx time %s is invalid",
			txHash.String(), tx.Tx.Timestamp.Format(time.RFC3339))
		return nil, nil, txRuleError(message.RejectTxTimestamp, str)
	}

	// Don't allow non-standard transactions if the mempool config forbids
	// their acceptance and relaying.
	medianTime := mp.cfg.PastMedianTime()
	if !mp.cfg.Policy.AcceptNonStd {
		err := checkTransactionStandard(tx, nextBlockHeight,
			medianTime, mp.cfg.Policy.MinRelayTxFee,
			mp.cfg.Policy.MaxTxVersion)
		if err != nil {
			// Attempt to extract a reject code from the error so
			// it can be retained.  When not possible, fall back to
			// a non standard error.
			rejectCode, found := extractRejectCode(err)
			if !found {
				rejectCode = message.RejectNonstandard
			}
			str := fmt.Sprintf("transaction %v is not standard: %v",
				txHash, err)
			return nil, nil, txRuleError(rejectCode, str)
		}
	}
	mempoolCheckTransactionStandard.Update(time.Now().Sub(start))
	start = time.Now()
	// The transaction may not use any of the same outputs as other
	// transactions already in the pool as that would ultimately result in a
	// double spend.  This check is intended to be quick and therefore only
	// detects double spends within the transaction pool itself.  The
	// transaction could still be double spending coins from the main chain
	// at this point.  There is a more in-depth check that happens later
	// after fetching the referenced transaction inputs from the main chain
	// which examines the actual spend data and prevents double spends.
	err = mp.checkPoolDoubleSpend(tx)
	if err != nil {
		return nil, nil, err
	}
	mempoolCheckPoolDoubleSpend.Update(time.Now().Sub(start))
	start = time.Now()
	// Verify crypto signatures for each input and reject the transaction if
	// any don't verify.
	flags, err := mp.cfg.Policy.StandardVerifyFlags()
	if err != nil {
		return nil, nil, err
	}
	// Don't allow transactions with fees too low to get into a mined block.
	serializedSize := int64(msgTx.SerializeSize())

	if serializedSize >= mp.cfg.Policy.MaxTxSize {
		return nil, nil, fmt.Errorf("serialized transaction is too big for pool - got %d, max %d", serializedSize, mp.cfg.Policy.MaxTxSize)
	}

	minFee := calcMinRequiredTxRelayFee(serializedSize,
		mp.cfg.Policy.MinRelayTxFee)

	if types.IsTokenTx(tx.Tx) {

		utxoView := utxo.NewUtxoViewpoint()
		if types.IsTokenMintTx(tx.Tx) {
			utxoView, err = mp.fetchInputUtxos(tx)
			if err != nil {
				return nil, nil, err
			}
			pkscript, err := mp.cfg.BC.GetCurTokenOwners(tx.Tx.TxOut[0].Amount.Id)
			if err != nil {
				return nil, nil, err
			}
			utxoView.AddTokenTxOut(tx.Tx.TxIn[0].PreviousOut, pkscript)

			err = mp.cfg.BC.CheckTokenTransactionInputs(tx, utxoView)
			if err != nil {
				return nil, nil, err
			}
		} else {
			utxoView.AddTokenTxOut(tx.Tx.TxIn[0].PreviousOut, nil)
		}

		err = blockchain.ValidateTransactionScripts(tx, utxoView, flags,
			mp.cfg.SigCache, int64(nextBlockHeight))
		if err != nil {
			if cerr, ok := err.(blockchain.RuleError); ok {
				return nil, nil, chainRuleError(cerr)
			}
			return nil, nil, err
		}

		// Add to transaction pool.
		txD := mp.addTransaction(utxoView, tx, nextBlockHeight, 0)

		log.Debug("Accepted transaction", "txHash", txHash, "pool size", len(mp.pool))

		return nil, txD, nil
	} else if types.IsCrossChainImportTx(tx.Tx) {
		if mp.cfg.BC.HasTx(txHash) {
			return nil, nil, fmt.Errorf("Already have transaction %v", txHash)
		}
		itx, err := mmeer.NewImportTx(tx.Tx)
		if err != nil {
			return nil, nil, err
		}
		pks, err := itx.GetPKScript()
		if err != nil {
			return nil, nil, err
		}
		utxoView := utxo.NewUtxoViewpoint()
		utxoView.AddTokenTxOut(tx.Tx.TxIn[0].PreviousOut, pks)
		vtsTx, err := itx.GetTransactionForEngine()
		if err != nil {
			return nil, nil, err
		}
		err = blockchain.ValidateTransactionScripts(types.NewTx(vtsTx), utxoView, flags,
			mp.cfg.SigCache, int64(nextBlockHeight))
		if err != nil {
			if cerr, ok := err.(blockchain.RuleError); ok {
				return nil, nil, chainRuleError(cerr)
			}
			return nil, nil, err
		}
		fee, err := mp.cfg.BC.MeerVerifyTx(itx, utxoView)
		if err != nil {
			return nil, nil, err
		}
		if fee < minFee {
			str := fmt.Sprintf("transaction %v has %v fees which is under the required amount of %v, tx size is %v bytes, policy-rate is %v/KB.", txHash,
				fee, minFee, serializedSize, mp.cfg.Policy.MinRelayTxFee.Value)
			return nil, nil, txRuleError(message.RejectInsufficientFee, str)
		}
		if !allowHighFees {
			maxFee := calcMinRequiredTxRelayFee(serializedSize*maxRelayFeeMultiplier,
				mp.cfg.Policy.MinRelayTxFee)
			if fee > maxFee {
				err = fmt.Errorf("transaction %v has %v fee which is above the "+
					"allowHighFee check threshold amount of %v (= %v byte * %v/kB * %v)", txHash,
					fee, maxFee, serializedSize, mp.cfg.Policy.MinRelayTxFee.Value, maxRelayFeeMultiplier)
				return nil, nil, err
			}
		}

		// Add to transaction pool.
		txD := mp.addTransaction(utxoView, tx, nextBlockHeight, fee)

		log.Debug(fmt.Sprintf("Accepted import transaction ,txHash(qng):%s ,pool size:%d , fee:%d", txHash, len(mp.pool), fee))
		return nil, txD, nil
	} else if opreturn.IsMeerEVMTx(tx.Tx) {
		return nil, nil, fmt.Errorf("Unsupported this MeerEVMTx %v", txHash)
	}
	// Fetch all of the unspent transaction outputs referenced by the inputs
	// to this transaction.  This function also attempts to fetch the
	// transaction itself to be used for detecting a duplicate transaction
	// without needing to do a separate lookup.
	utxoView, err := mp.fetchInputUtxos(tx)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}

	// Don't allow the transaction if it exists in the main chain and is not
	// not already fully spent.
	prevOut := types.TxOutPoint{Hash: *txHash}
	for txOutIdx := range tx.Tx.TxOut {
		prevOut.OutIndex = uint32(txOutIdx)
		entry := utxoView.LookupEntry(prevOut)
		if entry != nil && !entry.IsSpent() {
			return nil, nil, txRuleError(message.RejectDuplicate,
				"transaction already exists")
		}
		utxoView.RemoveEntry(prevOut)
	}

	// Transaction is an orphan if any of the inputs don't exist.
	var missingParents []*hash.Hash
	for _, txIn := range msgTx.TxIn {

		log.Trace("Looking up UTXO", "txIn", txIn, "PrevOutput", &txIn.PreviousOut.Hash)
		entry := utxoView.LookupEntry(txIn.PreviousOut)
		if entry == nil || entry.IsSpent() {
			// Must make a copy of the hash here since the iterator
			// is replaced and taking its address directly would
			// result in all of the entries pointing to the same
			// memory location and thus all be the final hash.
			hashCopy := txIn.PreviousOut.Hash
			missingParents = append(missingParents, &hashCopy)
		}
	}
	mempoolLookSpent.Update(time.Now().Sub(start))
	start = time.Now()
	if len(missingParents) > 0 {
		return missingParents, nil, nil
	}

	// Don't allow the transaction into the mempool unless its sequence
	// lock is active, meaning that it'll be allowed into the next block
	// with respect to its defined relative lock times.
	seqLock, err := mp.cfg.CalcSequenceLock(tx, utxoView)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}
	// TODO fix type conversion
	if !blockchain.SequenceLockActive(seqLock, int64(nextBlockHeight), medianTime) {
		return nil, nil, txRuleError(message.RejectNonstandard,
			"transaction sequence locks on inputs not met")
	}

	// Perform several checks on the transaction inputs using the invariant
	// rules in chain for what transactions are allowed into blocks.
	// Also returns the fees associated with the transaction which will be
	// used later.  The fraud proof is not checked because it will be
	// filled in by the miner.
	txFees, err := mp.cfg.BC.CheckTransactionInputs(tx, utxoView) //TODO fix type conversion
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}
	if _, exist := txFees[types.MEERA]; !exist {
		str := fmt.Sprintf("transaction %v must contain at least the utxo of base coin (MEER)", txHash)
		return nil, nil, txRuleError(message.RejectInvalid, str)
	}
	mempoolCheckTransactionInputs.Update(time.Now().Sub(start))
	start = time.Now()
	// Don't allow transactions with non-standard inputs if the mempool config
	// forbids their acceptance and relaying.
	if !mp.cfg.Policy.AcceptNonStd {
		err := checkInputsStandard(tx, utxoView)
		if err != nil {
			// Attempt to extract a reject code from the error so
			// it can be retained.  When not possible, fall back to
			// a non standard error.
			rejectCode, found := extractRejectCode(err)
			if !found {
				rejectCode = message.RejectNonstandard
			}
			str := fmt.Sprintf("transaction %v has a non-standard "+
				"input: %v", txHash, err)
			return nil, nil, txRuleError(rejectCode, str)
		}
	}

	// NOTE: if you modify this code to accept non-standard transactions,
	// you should add code here to check that the transaction does a
	// reasonable number of ECDSA signature verifications.

	// Don't allow transactions with an excessive number of signature
	// operations which would result in making it impossible to mine.  Since
	// the coinbase address itself can contain signature operations, the
	// maximum allowed signature operations per transaction is less than
	// the maximum allowed signature operations per block.
	numSigOps, err := blockchain.CountP2SHSigOps(tx, false, utxoView)
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}

	numSigOps += blockchain.CountSigOps(tx)
	if numSigOps > mp.cfg.Policy.MaxSigOpsPerTx {
		str := fmt.Sprintf("transaction %v has too many sigops: %d > %d",
			txHash, numSigOps, mp.cfg.Policy.MaxSigOpsPerTx)
		return nil, nil, txRuleError(message.RejectNonstandard, str)
	}
	mempoolCountSigOps.Update(time.Now().Sub(start))
	start = time.Now()
	txFee := types.Amount{Id: types.MEERA, Value: 0}
	if txFees != nil {
		txFee.Value = txFees[txFee.Id]
	}

	if txFee.Value < minFee {
		str := fmt.Sprintf("transaction %v has %v fees which "+
			"is under the required amount of %v, tx size is %v bytes, policy-rate is %v/byte.", txHash,
			txFee, minFee, serializedSize, mp.cfg.Policy.MinRelayTxFee.Value/1000)
		return nil, nil, txRuleError(message.RejectInsufficientFee, str)
	}

	// Require that free transactions have sufficient priority to be mined
	// in the next block.  Transactions which are being added back to the
	// memory pool from blocks that have been disconnected during a reorg
	// are exempted.
	//
	// This applies to non-stake transactions only.
	if isNew && !mp.cfg.Policy.DisableRelayPriority && txFee.Value < minFee {

		currentPriority := CalcPriority(msgTx, utxoView,
			nextBlockHeight, mp.cfg.BC)

		if currentPriority <= MinHighPriority {
			str := fmt.Sprintf("transaction %v has insufficient "+
				"priority (%g <= %g)", txHash,
				currentPriority, MinHighPriority)
			return nil, nil, txRuleError(message.RejectInsufficientFee, str)
		}
	}

	// Free-to-relay transactions are rate limited here to prevent
	// penny-flooding with tiny transactions as a form of attack.
	// This applies to non-stake transactions only.
	if rateLimit && txFee.Value < minFee {
		nowUnix := roughtime.Now().Unix()
		// Decay passed data with an exponentially decaying ~10 minute
		// window.
		mp.pennyTotal *= math.Pow(1.0-1.0/600.0,
			float64(nowUnix-mp.lastPennyUnix))
		mp.lastPennyUnix = nowUnix

		// Are we still over the limit?
		if mp.pennyTotal >= mp.cfg.Policy.FreeTxRelayLimit*10*1000 {
			str := fmt.Sprintf("transaction %v has been rejected "+
				"by the rate limiter due to low fees", txHash)
			return nil, nil, txRuleError(message.RejectInsufficientFee, str)
		}
		oldTotal := mp.pennyTotal

		mp.pennyTotal += float64(serializedSize)
		log.Trace("rate limit: curTotal %v, nextTotal: %v, "+
			"limit %v", oldTotal, mp.pennyTotal,
			mp.cfg.Policy.FreeTxRelayLimit*10*1000)
	}

	// Check whether allowHighFees is set to false (default), if so, then make
	// sure the current fee is sensible.  If people would like to avoid this
	// check then they can AllowHighFees = true
	if !allowHighFees {
		maxFee := calcMinRequiredTxRelayFee(serializedSize*maxRelayFeeMultiplier,
			mp.cfg.Policy.MinRelayTxFee)

		mrtf := types.Amount{Id: txFee.Id, Value: mp.cfg.Policy.MinRelayTxFee.Value}
		if txFee.Value > maxFee {
			err = fmt.Errorf("transaction %v has %v fee which is above the "+
				"allowHighFee check threshold amount of %v (= %v byte * %v/kB * %v)", txHash,
				txFee.Value, maxFee, serializedSize, mrtf.Format(types.AmountAtom), maxRelayFeeMultiplier)
			return nil, nil, err
		}
	}

	// Verify crypto signatures for each input and reject the transaction if
	// any don't verify.

	err = blockchain.ValidateTransactionScripts(tx, utxoView, flags,
		mp.cfg.SigCache, int64(nextBlockHeight))
	if err != nil {
		if cerr, ok := err.(blockchain.RuleError); ok {
			return nil, nil, chainRuleError(cerr)
		}
		return nil, nil, err
	}

	// Add to transaction pool.
	txD := mp.addTransaction(utxoView, tx, nextBlockHeight, txFee.Value)

	log.Debug("Accepted transaction", "txHash", txHash, "pool size", len(mp.pool))

	return nil, txD, nil
}

// fetchInputUtxos loads utxo details about the input transactions referenced by
// the passed transaction.  First, it loads the details form the viewpoint of
// the main chain, then it adjusts them based upon the contents of the
// transaction pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) fetchInputUtxos(tx *types.Tx) (*utxo.UtxoViewpoint, error) {
	utxoView, err := mp.cfg.FetchUtxoView(tx)
	if err != nil {
		return nil, err
	}

	// Attempt to populate any missing inputs from the transaction pool.
	for _, txIn := range tx.Tx.TxIn {
		prevOut := &txIn.PreviousOut
		entry := utxoView.LookupEntry(*prevOut)
		if entry != nil && !entry.IsSpent() {
			continue
		}

		if poolTxDesc, exists := mp.pool[prevOut.Hash]; exists {
			// AddTxOut ignores out of range index values, so it is
			// safe to call without bounds checking here.
			utxoView.AddTxOut(poolTxDesc.Tx, prevOut.OutIndex, &hash.ZeroHash)
		}
	}

	return utxoView, nil
}

// ProcessTransaction is the main workhorse for handling insertion of new
// free-standing transactions into the memory pool.  It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, orphan transaction handling, and insertion into the memory pool.
//
// It returns a slice of transactions added to the mempool.  When the
// error is nil, the list will include the passed transaction itself along
// with any additional orphan transaactions that were added as a result of
// the passed one being accepted.
//
// This function is safe for concurrent access.
func (mp *TxPool) ProcessTransaction(tx *types.Tx, allowOrphan, rateLimit, allowHighFees bool) ([]*types.TxDesc, error) {
	// Protect concurrent access.
	start := time.Now()
	mp.procmtx.Lock()
	defer mp.procmtx.Unlock()
	var err error
	defer func() {
		if err != nil {
			log.Trace("Failed to process transaction", "tx", tx.Hash(), "err", err.Error())
		}
	}()

	// Potentially accept the transaction to the memory pool.
	var missingParents []*hash.Hash
	missingParents, txD, err := mp.maybeAcceptTransaction(tx, true, rateLimit,
		allowHighFees)
	if err != nil {
		return nil, err
	}
	mempoolMaybeAcceptTransaction.Update(time.Now().Sub(start))
	// If len(missingParents) == 0 then we know the tx is NOT an orphan.
	if len(missingParents) == 0 {
		// Accept any orphan transactions that depend on this
		// transaction (they are no longer orphans if all inputs are
		// now available) and repeat for those accepted transactions
		// until there are no more.
		newTxs := mp.processOrphans(tx.Hash())
		acceptedTxs := []*types.TxDesc{}

		// Add the parent transaction first so remote nodes
		// do not add orphans.
		acceptedTxs = append(acceptedTxs, &txD.TxDesc)
		for _, td := range newTxs {
			acceptedTxs = append(acceptedTxs, &td.TxDesc)
		}

		return acceptedTxs, nil
	}

	// The transaction is an orphan (has inputs missing).  Reject
	// it if the flag to allow orphans is not set.
	if !allowOrphan {
		// Only use the first missing parent transaction in
		// the error message.
		//
		// NOTE: RejectDuplicate is really not an accurate
		// reject code here, but it matches the reference
		// implementation and there isn't a better choice due
		// to the limited number of reject codes.  Missing
		// inputs is assumed to mean they are already spent
		// which is not really always the case.
		str := fmt.Sprintf("orphan transaction %v references "+
			"outputs of unknown or fully-spent "+
			"transaction %v", tx.Hash(), missingParents[0])
		return nil, txRuleError(message.RejectDuplicate, str)
	}

	// Potentially add the orphan transaction to the orphan pool.
	err = mp.maybeAddOrphan(tx)
	return nil, err
}

// maybeAddOrphan potentially adds an orphan to the orphan pool.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) maybeAddOrphan(tx *types.Tx) error {
	// Ignore orphan transactions that are too large.  This helps avoid
	// a memory exhaustion attack based on sending a lot of really large
	// orphans.  In the case there is a valid transaction larger than this,
	// it will ultimtely be rebroadcast after the parent transactions
	// have been mined or otherwise received.
	//
	// Note that the number of orphan transactions in the orphan pool is
	// also limited, so this equates to a maximum memory used of
	// mp.cfg.Policy.MaxOrphanTxSize * mp.cfg.Policy.MaxOrphanTxs (which is ~5MB
	// using the default values at the time this comment was written).
	serializedLen := tx.Transaction().SerializeSize()
	if serializedLen > mp.cfg.Policy.MaxOrphanTxSize {
		str := fmt.Sprintf("orphan transaction size of %d bytes is "+
			"larger than max allowed size of %d bytes",
			serializedLen, mp.cfg.Policy.MaxOrphanTxSize)
		return txRuleError(message.RejectNonstandard, str)
	}

	// Add the orphan if the none of the above disqualified it.
	mp.addOrphan(tx)

	return nil
}

// MaybeAcceptTransaction is the main workhorse for handling insertion of new
// free-standing transactions into a memory pool.  It includes functionality
// such as rejecting duplicate transactions, ensuring transactions follow all
// rules, orphan transaction handling, and insertion into the memory pool.  The
// isOrphan parameter can be nil if the caller does not need to know whether
// or not the transaction is an orphan.
//
// This function is safe for concurrent access.
func (mp *TxPool) MaybeAcceptTransaction(tx *types.Tx, isNew, rateLimit bool) ([]*hash.Hash, error) {
	// Protect concurrent access.
	mp.procmtx.Lock()
	hashes, _, err := mp.maybeAcceptTransaction(tx, isNew, rateLimit, true)
	mp.procmtx.Unlock()

	return hashes, err
}

// removeOrphan is the internal function which implements the public
// RemoveOrphan.  See the comment for RemoveOrphan for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) removeOrphan(txHash *hash.Hash) {
	mp.orphansmtx.Lock()
	defer mp.orphansmtx.Unlock()
	// Nothing to do if passed tx is not an orphan.
	tx, exists := mp.orphans[*txHash]
	if !exists {
		return
	}

	// Remove the reference from the previous orphan index.
	for _, txIn := range tx.Transaction().TxIn {
		originTxHash := txIn.PreviousOut.Hash
		if orphans, exists := mp.orphansByPrev[originTxHash]; exists {
			delete(orphans, *tx.Hash())

			// Remove the map entry altogether if there are no
			// longer any orphans which depend on it.
			if len(orphans) == 0 {
				delete(mp.orphansByPrev, originTxHash)
			}
		}
	}

	// Remove the transaction from the orphan pool.
	delete(mp.orphans, *txHash)
	log.Trace(fmt.Sprintf("Removing orphan transaction %v", txHash))
}

// RemoveOrphan removes the passed orphan transaction from the orphan pool and
// previous orphan index.
//
// This function is safe for concurrent access.
func (mp *TxPool) RemoveOrphan(tx *types.Tx) {
	if types.IsTokenTx(tx.Tx) ||
		types.IsCrossChainImportTx(tx.Tx) ||
		opreturn.IsMeerEVMTx(tx.Tx) {
		return
	}
	mp.removeOrphan(tx.Hash())
}

// processOrphans is the internal function which implements the public
// ProcessOrphans.  See the comment for ProcessOrphans for more details.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) processOrphans(h *hash.Hash) []*TxDesc {
	var acceptedTxns []*TxDesc

	// Start with processing at least the passed hash.
	processHashes := list.New()
	processHashes.PushBack(h)
	for processHashes.Len() > 0 {
		// Pop the first hash to process.
		firstElement := processHashes.Remove(processHashes.Front())
		processHash := firstElement.(*hash.Hash)

		// Look up all orphans that are referenced by the transaction we
		// just accepted.  This will typically only be one, but it could
		// be multiple if the referenced transaction contains multiple
		// outputs.  Skip to the next item on the list of hashes to
		// process if there are none.
		mp.orphansmtx.RLock()
		orphans, exists := mp.orphansByPrev[*processHash]
		mp.orphansmtx.RUnlock()
		if !exists || orphans == nil {
			continue
		}

		for _, tx := range orphans {
			// Remove the orphan from the orphan pool.  Current
			// behavior requires that all saved orphans with
			// a newly accepted parent are removed from the orphan
			// pool and potentially added to the memory pool, but
			// transactions which cannot be added to memory pool
			// (including due to still being orphans) are expunged
			// from the orphan pool.
			//
			// TODO(jrick): The above described behavior sounds
			// like a bug, and I think we should investigate
			// potentially moving orphans to the memory pool, but
			// leaving them in the orphan pool if not all parent
			// transactions are known yet.
			orphanHash := tx.Hash()

			// Potentially accept the transaction into the
			// transaction pool.
			missingParents, txD, err := mp.maybeAcceptTransaction(tx,
				true, true, true)
			if err != nil {
				// TODO: Remove orphans that depend on this
				// failed transaction.
				log.Debug("Unable to move orphan transaction "+
					"%v to mempool: %v", tx.Hash(), err)
				mp.removeOrphan(orphanHash)
				continue
			}

			if len(missingParents) > 0 {
				continue
			}

			// Add this transaction to the list of transactions
			// that are no longer orphans.
			acceptedTxns = append(acceptedTxns, txD)
			mp.removeOrphan(orphanHash)
			// Add this transaction to the list of transactions to
			// process so any orphans that depend on this one are
			// handled too.
			//
			// TODO(jrick): In the case that this is still an orphan,
			// we know that any other transactions in the orphan
			// pool with this orphan as their parent are still
			// orphans as well, and should be removed.  While
			// recursively calling removeOrphan and
			// maybeAcceptTransaction on these transactions is not
			// wrong per se, it is overkill if all we care about is
			// recursively removing child transactions of this
			// orphan.
			processHashes.PushBack(orphanHash)
		}
	}

	return acceptedTxns
}

// addOrphan adds an orphan transaction to the orphan pool.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) addOrphan(tx *types.Tx) {
	// Nothing to do if no orphans are allowed.
	if mp.cfg.Policy.MaxOrphanTxs <= 0 {
		return
	}
	mp.orphansmtx.Lock()
	defer mp.orphansmtx.Unlock()

	mp.orphans[*tx.Hash()] = tx
	for _, txIn := range tx.Tx.TxIn {
		originTxHash := txIn.PreviousOut.Hash
		if _, exists := mp.orphansByPrev[originTxHash]; !exists {
			mp.orphansByPrev[originTxHash] =
				make(map[hash.Hash]*types.Tx)
		}
		mp.orphansByPrev[originTxHash][*tx.Hash()] = tx
	}

	log.Debug(fmt.Sprintf("Stored orphan transaction %v (total: %d)", tx.Hash(),
		len(mp.orphans)))
}

// ProcessOrphans determines if there are any orphans which depend on the passed
// transaction hash (it is possible that they are no longer orphans) and
// potentially accepts them to the memory pool.  It repeats the process for the
// newly accepted transactions (to detect further orphans which may no longer be
// orphans) until there are no more.
//
// It returns a slice of transactions added to the mempool.  A nil slice means
// no transactions were moved from the orphan pool to the mempool.
//
// This function is safe for concurrent access.
func (mp *TxPool) ProcessOrphans(tx *types.Tx) []*types.TxDesc {
	if opreturn.IsMeerEVMTx(tx.Tx) {
		return nil
	}
	mp.procmtx.Lock()
	acceptedTxns := mp.processOrphans(tx.Hash())
	mp.procmtx.Unlock()
	if len(acceptedTxns) <= 0 {
		return nil
	}
	acceptedTxnsT := []*types.TxDesc{}
	for _, td := range acceptedTxns {
		acceptedTxnsT = append(acceptedTxnsT, &td.TxDesc)
	}
	return acceptedTxnsT
}

// FetchTransaction returns the requested transaction from the transaction pool.
// This only fetches from the main transaction pool and does not include
// orphans.
//
// This function is safe for concurrent access.
func (mp *TxPool) FetchTransaction(txHash *hash.Hash) (*types.Tx, error) {
	// Protect concurrent access.
	mp.mtx.RLock()
	txDesc, exists := mp.pool[*txHash]
	mp.mtx.RUnlock()

	if exists {
		return txDesc.Tx, nil
	}
	er := fmt.Errorf("transaction is not in the pool")
	etxs, _, err := mp.cfg.BC.MeerChain().(*meer.MeerChain).MeerPool().GetTxs()
	if err != nil {
		return nil, er
	}

	for _, tx := range etxs {
		th := tx.Hash()
		if txHash.IsEqual(th) {
			return tx, nil
		}
	}
	return nil, er
}

func (mp *TxPool) FetchTransactions(txHashs []*hash.Hash) ([]*types.Tx, error) {
	// Protect concurrent access.
	result := []*types.Tx{}

	mp.mtx.RLock()
	for _, txh := range txHashs {
		txDesc, exists := mp.pool[*txh]
		if !exists {
			continue
		}
		result = append(result, txDesc.Tx)
	}
	mp.mtx.RUnlock()

	er := fmt.Errorf("transaction is not in the pool")
	etxs, _, err := mp.cfg.BC.MeerChain().(*meer.MeerChain).MeerPool().GetTxs()
	if err != nil {
		return nil, er
	}

	etxsM := map[string]*types.Tx{}
	for i := 0; i < len(etxs); i++ {
		etxsM[etxs[i].Hash().String()] = etxs[i]
	}

	for _, txh := range txHashs {
		tx, exists := etxsM[txh.String()]
		if !exists {
			continue
		}
		result = append(result, tx)
	}
	return result, nil
}

// HaveAllTransactions returns whether or not all of the passed transaction
// hashes exist in the mempool.
//
// This function is safe for concurrent access.
func (mp *TxPool) HaveAllTransactions(hashes []hash.Hash) bool {
	etxs, _, _ := mp.cfg.BC.MeerChain().(*meer.MeerChain).MeerPool().GetTxs()
	mp.mtx.RLock()
	inPool := true
	for _, h := range hashes {
		if _, exists := mp.pool[h]; !exists {

			isinep := false
			if len(etxs) > 0 {
				for _, tx := range etxs {
					th := tx.Hash()
					if h.IsEqual(th) {
						isinep = true
						break
					}
				}
			}

			if !isinep {
				inPool = false
				break
			}
		}
	}
	mp.mtx.RUnlock()
	return inPool
}

// haveTransaction returns whether or not the passed transaction already exists
// in the main pool or in the orphan pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) haveTransactionAll(hash *hash.Hash) bool {
	return mp.isTransactionInPool(hash, true) || mp.isOrphanInPool(hash)
}

// HaveTransaction returns whether or not the passed transaction already exists
// in the main pool or in the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) HaveTransaction(hash *hash.Hash) bool {
	start := time.Now()
	defer mempoolHaveTransaction.Update(time.Now().Sub(start))
	return mp.haveTransaction(hash)
}

func (mp *TxPool) haveTransaction(hash *hash.Hash) bool {
	// Protect concurrent access.
	haveTx := mp.isTransactionInPool(hash, false) || mp.isOrphanInPool(hash)
	return haveTx
}

// isTransactionInPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function MUST be called with the mempool lock held (for reads).
// all: include evm tx
func (mp *TxPool) isTransactionInPool(hash *hash.Hash, all bool) bool {
	mp.mtx.RLock()
	_, exists := mp.pool[*hash]
	mp.mtx.RUnlock()

	if exists {
		return true
	}
	if !all {
		return false
	}
	return mp.cfg.BC.MeerChain().(*meer.MeerChain).MeerPool().HasTx(hash)
}

// IsTransactionInPool returns whether or not the passed transaction already
// exists in the main pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) IsTransactionInPool(hash *hash.Hash, all bool) bool {
	// Protect concurrent access.
	return mp.isTransactionInPool(hash, all)
}

// isOrphanInPool returns whether or not the passed transaction already exists
// in the orphan pool.
//
// This function MUST be called with the mempool lock held (for reads).
func (mp *TxPool) isOrphanInPool(hash *hash.Hash) bool {
	mp.orphansmtx.RLock()
	_, exists := mp.orphans[*hash]
	mp.orphansmtx.RUnlock()
	if exists {
		return true
	}
	return false
}

// IsOrphanInPool returns whether or not the passed transaction already exists
// in the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) IsOrphanInPool(hash *hash.Hash) bool {
	// Protect concurrent access.
	return mp.isOrphanInPool(hash)
}

// LastUpdated returns the last time a transaction was added to or removed from
// the main pool.  It does not include the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) LastUpdated() time.Time {
	return time.Unix(atomic.LoadInt64(&mp.lastUpdated), 0)
}

// MiningDescs returns a slice of mining descriptors for all the transactions
// in the pool.
//
// This is part of the mining.TxSource interface implementation and is safe for
// concurrent access as required by the interface contract.
func (mp *TxPool) MiningDescs() []*types.TxDesc {
	mp.mtx.RLock()
	descs := []*types.TxDesc{}
	for _, desc := range mp.pool {
		descs = append(descs, &desc.TxDesc)
	}
	mp.mtx.RUnlock()

	etxs, _, err := mp.cfg.BC.MeerChain().(*meer.MeerChain).MeerPool().GetTxs()
	if err != nil {
		log.Error(err.Error())
		return descs
	}

	for _, tx := range etxs {
		txDesc := &types.TxDesc{
			Tx:     tx,
			Added:  roughtime.Now(),
			Height: mp.GetMainHeight(),
		}
		descs = append(descs, txDesc)
	}
	return descs
}

// pruneExpiredTx prunes expired transactions from the mempool that may no longer
// be able to be included into a block.
//
// This function MUST be called with the mempool lock held (for writes).
func (mp *TxPool) DoPruneExpiredTx() {
	nextBlockHeight := mp.cfg.BestHeight() + 1

	mp.procmtx.Lock()
	defer mp.procmtx.Unlock()
	for _, tx := range mp.pool {
		if blockchain.IsExpired(tx.Tx, nextBlockHeight) {
			log.Debug(fmt.Sprintf("Pruning expired transaction %v from the mempool",
				tx.Tx.Hash()))
			mp.removeTransaction(tx.Tx, true)
		}
	}
	mp.pruneDirty.Store(false)
}

// PruneExpiredTx prunes expired transactions from the mempool that may no longer
// be able to be included into a block.
//
// This function is safe for concurrent access.
func (mp *TxPool) PruneExpiredTx() {
	mp.pruneDirty.Store(true)
}

// Count returns the number of transactions in the main pool.  It does not
// include the orphan pool.
//
// This function is safe for concurrent access.
func (mp *TxPool) Count() int {
	mp.mtx.RLock()
	count := len(mp.pool)
	mp.mtx.RUnlock()

	count += int(mp.cfg.BC.MeerChain().(*meer.MeerChain).MeerPool().GetSize())

	return count
}

func (mp *TxPool) GetConfig() *Config {
	return &mp.cfg
}

func (mp *TxPool) GetMainHeight() int64 {
	return int64(mp.cfg.BestHeight() + 1)
}

func (mp *TxPool) IsSupportVMTx() bool {
	return mp.cfg.BC.IsValidTxType(types.TxTypeCrossChainVM)
}

func (mp *TxPool) AddUnconfirmedTx(tx *types.Tx, utxoView *utxo.UtxoViewpoint) {
	var pkScripts [][]byte
	msgTx := tx.Transaction()
	for _, txIn := range msgTx.TxIn {
		entry := utxoView.LookupEntry(txIn.PreviousOut)
		if entry == nil {
			// Ignore missing entries.  This should never happen
			// in practice since the function comments specifically
			// call out all inputs must be available.
			continue
		}
		pkScript := entry.PkScript()
		//txType := entry.TransactionType()
		pkScripts = append(pkScripts, pkScript)
	}

	// Index addresses of all created outputs.
	for _, txOut := range msgTx.TxOut {
		pkScripts = append(pkScripts, txOut.PkScript)
	}
	mp.cfg.IndexManager.AddrIndex().AddUnconfirmedTx(tx, pkScripts)
}

func (mp *TxPool) Dirty() bool {
	return mp.dirty.Load()
}

func (mp *TxPool) CleanDirty() {
	mp.dirty.Store(false)
}

func (mp *TxPool) PruneDirty() bool {
	return mp.pruneDirty.Load()
}
