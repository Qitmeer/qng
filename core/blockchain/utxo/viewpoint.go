package utxo

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

// UtxoViewpoint represents a view into the set of unspent transaction outputs
// from a specific point of view in the chain.  For example, it could be for
// the end of the main chain, some point in the history of the main chain, or
// down a side chain.
//
// The unspent outputs are needed by other transactions for things such as
// script validation and double spend prevention.
type UtxoViewpoint struct {
	entries    map[types.TxOutPoint]*UtxoEntry
	viewpoints []*hash.Hash
}

func (view *UtxoViewpoint) AddEntry(outpoint types.TxOutPoint, entry *UtxoEntry) {
	view.entries[outpoint] = entry
}

func (view *UtxoViewpoint) RemoveEntry(outpoint types.TxOutPoint) {
	delete(view.entries, outpoint)
}

func (view *UtxoViewpoint) Clean() {
	view.entries = map[types.TxOutPoint]*UtxoEntry{}
}

// Entries returns the underlying map that stores of all the utxo entries.
func (view *UtxoViewpoint) Entries() map[types.TxOutPoint]*UtxoEntry {
	return view.entries
}

func (view *UtxoViewpoint) GetEntry(outpoint types.TxOutPoint) *UtxoEntry {
	return view.entries[outpoint]
}

func (view *UtxoViewpoint) AddTxOut(tx *types.Tx, txOutIdx uint32, blockHash *hash.Hash) {
	if types.IsCrossChainExportTx(tx.Tx) {
		if txOutIdx == 0 {
			return
		}
	}
	// Can't add an output for an out of bounds index.
	if txOutIdx >= uint32(len(tx.Tx.TxOut)) {
		return
	}

	// Update existing entries.  All fields are updated because it's
	// possible (although extremely unlikely) that the existing entry is
	// being replaced by a different transaction with the same hash.  This
	// is allowed so long as the previous transaction is fully spent.
	prevOut := types.TxOutPoint{Hash: *tx.Hash(), OutIndex: txOutIdx}
	txOut := tx.Tx.TxOut[txOutIdx]
	view.addTxOut(prevOut, txOut, tx.Tx.IsCoinBase(), blockHash)
}

// AddTxOuts adds all outputs in the passed transaction which are not provably
// unspendable to the view.  When the view already has entries for any of the
// outputs, they are simply marked unspent.  All fields will be updated for
// existing entries since it's possible it has changed during a reorg.
func (view *UtxoViewpoint) AddTxOuts(tx *types.Tx, blockHash *hash.Hash) {
	// Loop all of the transaction outputs and add those which are not
	// provably unspendable.
	isCoinBase := tx.Tx.IsCoinBase()
	prevOut := types.TxOutPoint{Hash: *tx.Hash()}
	for txOutIdx, txOut := range tx.Tx.TxOut {
		if txOutIdx == 0 && types.IsCrossChainExportTx(tx.Tx) {
			continue
		}
		// Update existing entries.  All fields are updated because it's
		// possible (although extremely unlikely) that the existing
		// entry is being replaced by a different transaction with the
		// same hash.  This is allowed so long as the previous
		// transaction is fully spent.
		prevOut.OutIndex = uint32(txOutIdx)
		view.addTxOut(prevOut, txOut, isCoinBase, blockHash)
	}
}

func (view *UtxoViewpoint) addTxOut(outpoint types.TxOutPoint, txOut *types.TxOutput, isCoinBase bool, blockHash *hash.Hash) {
	// Don't add provably unspendable outputs.
	if txscript.IsUnspendable(txOut.PkScript) {
		return
	}

	// Update existing entries.  All fields are updated because it's
	// possible (although extremely unlikely) that the existing entry is
	// being replaced by a different transaction with the same hash.  This
	// is allowed so long as the previous transaction is fully spent.
	entry := view.LookupEntry(outpoint)
	if entry == nil {
		entry = new(UtxoEntry)
		view.entries[outpoint] = entry
	}

	entry.amount = txOut.Amount
	entry.pkScript = txOut.PkScript
	entry.blockHash = *blockHash
	entry.packedFlags = tfModified
	if isCoinBase {
		entry.packedFlags |= tfCoinBase
	}
}

func (view *UtxoViewpoint) AddTokenTxOut(outpoint types.TxOutPoint, pkscript []byte) {
	entry := view.LookupEntry(outpoint)
	if entry == nil {
		entry = new(UtxoEntry)
		view.entries[outpoint] = entry
	}
	if len(pkscript) <= 0 {
		pkscript = params.ActiveNetParams.Params.TokenAdminPkScript
	}
	txOut := &types.TxOutput{PkScript: pkscript}
	entry.amount = txOut.Amount
	entry.pkScript = txOut.PkScript
	entry.packedFlags = tfModified
}

// Viewpoints returns the hash of the viewpoint block in the chain the view currently
// respresents.
func (view *UtxoViewpoint) Viewpoints() []*hash.Hash {
	return view.viewpoints
}

// SetViewpoints sets the hash of the viewpoint block in the chain the view currently
// respresents.
func (view *UtxoViewpoint) SetViewpoints(views []*hash.Hash) {
	view.viewpoints = views
}

// FetchUtxosMain fetches unspent transaction output data about the provided
// set of transactions from the point of view of the end of the main chain at
// the time of the call.
//
// Upon completion of this function, the view will contain an entry for each
// requested transaction.  Fully spent transactions, or those which otherwise
// don't exist, will result in a nil entry in the view.
func (view *UtxoViewpoint) FetchUtxosMain(db model.DataBase, outpoints map[types.TxOutPoint]struct{}) error {
	// Nothing to do if there are no requested hashes.
	if len(outpoints) == 0 {
		return nil
	}

	// Load the unspent transaction output information for the requested set
	// of transactions from the point of view of the end of the main chain.
	//
	// NOTE: Missing entries are not considered an error here and instead
	// will result in nil entries in the view.  This is intentionally done
	// since other code uses the presence of an entry in the store as a way
	// to optimize spend and unspend updates to apply only to the specific
	// utxos that the caller needs access to.
	for outpoint := range outpoints {
		entry, err := dbFetchUtxoEntry(db, outpoint)
		if err != nil {
			return err
		}
		if entry == nil {
			continue
		}
		view.entries[outpoint] = entry
	}
	return nil
}

// LookupEntry returns information about a given transaction according to the
// current state of the view.  It will return nil if the passed transaction
// hash does not exist in the view or is otherwise not available such as when
// it has been disconnected during a reorg.
func (view *UtxoViewpoint) LookupEntry(outpoint types.TxOutPoint) *UtxoEntry {
	entry, ok := view.entries[outpoint]
	if !ok {
		return nil
	}

	return entry
}

// fetchUtxos loads the unspent transaction outputs for the provided set of
// outputs into the view from the database as needed unless they already exist
// in the view in which case they are ignored.
func (view *UtxoViewpoint) fetchUtxos(db model.DataBase, outpoints map[types.TxOutPoint]struct{}) error {
	// Nothing to do if there are no requested outputs.
	if len(outpoints) == 0 {
		return nil
	}

	// Filter entries that are already in the view.
	neededSet := make(map[types.TxOutPoint]struct{})
	for outpoint := range outpoints {
		// Already loaded into the current view.
		if _, ok := view.entries[outpoint]; ok {
			continue
		}

		neededSet[outpoint] = struct{}{}
	}

	// Request the input utxos from the database.
	return view.FetchUtxosMain(db, neededSet)
}

// commit prunes all entries marked modified that are now fully spent and marks
// all entries as unmodified.
func (view *UtxoViewpoint) Commit() {
	for outpoint, entry := range view.entries {
		if entry == nil || (entry.IsModified() && entry.IsSpent()) {
			delete(view.entries, outpoint)
			continue
		}

		entry.packedFlags ^= tfModified
	}
}

// NewUtxoViewpoint returns a new empty unspent transaction output view.
func NewUtxoViewpoint() *UtxoViewpoint {
	return &UtxoViewpoint{
		entries: make(map[types.TxOutPoint]*UtxoEntry),
	}
}
