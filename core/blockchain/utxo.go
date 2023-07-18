// Copyright (c) 2017-2018 The qitmeer developers
package blockchain

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/meerdag"
)

func (bc *BlockChain) IsInvalidOut(entry *utxo.UtxoEntry) bool {
	if entry == nil {
		return true
	}
	if entry.BlockHash().IsEqual(&hash.ZeroHash) {
		return false
	}
	node := bc.BlockDAG().GetBlock(entry.BlockHash())
	if node != nil {
		if !node.GetState().GetStatus().KnownInvalid() {
			return false
		}
	}
	return true
}

func (bc *BlockChain) FetchInputUtxos(db legacydb.DB, block *types.SerializedBlock, view *utxo.UtxoViewpoint) error {
	return bc.fetchInputUtxos(block, view)
}

// fetchInputUtxos loads utxo details about the input transactions referenced
// by the transactions in the given block into the view from the database as
// needed.  In particular, referenced entries that are earlier in the block are
// added to the view and entries that are already in the view are not modified.
// TODO, revisit the usage on the parent block
func (bc *BlockChain) fetchInputUtxos(block *types.SerializedBlock, view *utxo.UtxoViewpoint) error {
	// Build a map of in-flight transactions because some of the inputs in
	// this block could be referencing other transactions earlier in this
	// block which are not yet in the chain.
	txInFlight := map[hash.Hash]int{}
	transactions := block.Transactions()
	for i, tx := range transactions {
		if tx.IsDuplicate ||
			types.IsTokenTx(tx.Tx) ||
			types.IsCrossChainImportTx(tx.Tx) ||
			types.IsCrossChainVMTx(tx.Tx) {
			continue
		}
		txInFlight[*tx.Hash()] = i
	}

	// Loop through all of the transaction inputs (except for the coinbase
	// which has no inputs) collecting them into sets of what is needed and
	// what is already known (in-flight).
	txNeededSet := make(map[types.TxOutPoint]struct{})
	for i, tx := range transactions[1:] {
		if tx.IsDuplicate {
			continue
		}
		if types.IsTokenTx(tx.Tx) && !types.IsTokenMintTx(tx.Tx) {
			continue
		}
		if types.IsCrossChainImportTx(tx.Tx) {
			continue
		}
		if types.IsCrossChainVMTx(tx.Tx) {
			continue
		}

		for txInIdx, txIn := range tx.Transaction().TxIn {
			if txInIdx == 0 && types.IsTokenMintTx(tx.Tx) {
				continue
			}
			// It is acceptable for a transaction input to reference
			// the output of another transaction in this block only
			// if the referenced transaction comes before the
			// current one in this block.  Add the outputs of the
			// referenced transaction as available utxos when this
			// is the case.  Otherwise, the utxo details are still
			// needed.
			//
			// NOTE: The >= is correct here because i is one less
			// than the actual position of the transaction within
			// the block due to skipping the coinbase.
			originHash := &txIn.PreviousOut.Hash
			if inFlightIndex, ok := txInFlight[*originHash]; ok &&
				i >= inFlightIndex {

				originTx := transactions[inFlightIndex]
				view.AddTxOuts(originTx, block.Hash())
				continue
			}

			// Don't request entries that are already in the view
			// from the database.
			if _, ok := view.Entries()[txIn.PreviousOut]; ok {
				continue
			}

			txNeededSet[txIn.PreviousOut] = struct{}{}
		}
	}
	err := view.FetchUtxosMain(bc.consensus.DatabaseContext(), txNeededSet)
	if err != nil {
		return err
	}
	bc.FilterInvalidOut(view)
	// Request the input utxos from the database.
	return nil

}

func (bc *BlockChain) FilterInvalidOut(view *utxo.UtxoViewpoint) {
	for outpoint, entry := range view.Entries() {
		if !bc.IsInvalidOut(entry) {
			continue
		}
		view.RemoveEntry(outpoint)
	}
}

// FetchUtxoView loads utxo details about the input transactions referenced by
// the passed transaction from the point of view of the end of the main chain.
// It also attempts to fetch the utxo details for the transaction itself so the
// returned view can be examined for duplicate unspent transaction outputs.
//
// This function is safe for concurrent access however the returned view is NOT.
func (b *BlockChain) FetchUtxoView(tx *types.Tx) (*utxo.UtxoViewpoint, error) {
	// Create a set of needed transactions based on those referenced by the
	// inputs of the passed transaction.  Also, add the passed transaction
	// itself as a way for the caller to detect duplicates that are not
	// fully spent.
	neededSet := make(map[types.TxOutPoint]struct{})
	prevOut := types.TxOutPoint{Hash: *tx.Hash()}
	for txOutIdx := range tx.Tx.TxOut {
		prevOut.OutIndex = uint32(txOutIdx)
		neededSet[prevOut] = struct{}{}
	}
	if !tx.Tx.IsCoinBase() {
		for _, txIn := range tx.Tx.TxIn {
			neededSet[txIn.PreviousOut] = struct{}{}
		}
	}

	// Request the utxos from the point of view of the end of the main
	// chain.
	view := utxo.NewUtxoViewpoint()
	view.SetViewpoints(b.GetMiningTips(meerdag.MaxPriority))
	b.ChainRLock()
	err := view.FetchUtxosMain(b.consensus.DatabaseContext(), neededSet)
	b.ChainRUnlock()
	if err != nil {
		return view, err
	}
	b.FilterInvalidOut(view)
	return view, err
}

// FetchUtxoEntry loads and returns the unspent transaction output entry for the
// passed hash from the point of view of the end of the main chain.
//
// NOTE: Requesting a hash for which there is no data will NOT return an error.
// Instead both the entry and the error will be nil.  This is done to allow
// pruning of fully spent transactions.  In practice this means the caller must
// check if the returned entry is nil before invoking methods on it.
//
// This function is safe for concurrent access however the returned entry (if
// any) is NOT.
func (b *BlockChain) FetchUtxoEntry(outpoint types.TxOutPoint) (*utxo.UtxoEntry, error) {
	b.ChainRLock()
	defer b.ChainRUnlock()

	entry, err := utxo.DBFetchUtxoEntry(b.consensus.DatabaseContext(), outpoint)
	if err != nil {
		return nil, err
	}
	if b.IsInvalidOut(entry) {
		entry = nil
	}
	return entry, nil
}

func (b *BlockChain) dbPutUtxoView(view *utxo.UtxoViewpoint) error {
	for outpoint, entry := range view.Entries() {
		// No need to update the database if the entry was not modified.
		if entry == nil || !entry.IsModified() {
			continue
		}

		// Remove the utxo entry if it is spent.
		if entry.IsSpent() {
			key := utxo.OutpointKey(outpoint)
			err := b.consensus.DatabaseContext().DeleteUtxo(*key)
			utxo.RecycleOutpointKey(key)
			if err != nil {
				return err
			}
			if b.Acct != nil {
				err = b.Acct.Apply(false, &outpoint, entry)
				if err != nil {
					log.Error(err.Error())
				}
			}
			continue
		}

		// Serialize and store the utxo entry.
		serialized, err := utxo.SerializeUtxoEntry(entry)
		if err != nil {
			return err
		}
		key := utxo.OutpointKey(outpoint)
		err = b.consensus.DatabaseContext().PutUtxo(*key, serialized)
		// NOTE: The key is intentionally not recycled here since the
		// database interface contract prohibits modifications.  It will
		// be garbage collected normally when the database is done with
		// it.
		if err != nil {
			return err
		}

		if b.Acct != nil {
			err = b.Acct.Apply(true, &outpoint, entry)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}

	return nil
}

func (b *BlockChain) dbPutUtxoViewByBlock(block *types.SerializedBlock) error {
	view := utxo.NewUtxoViewpoint()
	view.SetViewpoints([]*hash.Hash{block.Hash()})
	for _, tx := range block.Transactions() {
		view.AddTxOuts(tx, block.Hash())
	}
	return b.dbPutUtxoView(view)
}
