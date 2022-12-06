package utxo

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
)

// txoFlags is a bitmask defining additional information and state for a
// transaction output in a utxo view.
type txoFlags uint8

const (
	// tfCoinBase indicates that a txout was contained in a coinbase tx.
	tfCoinBase txoFlags = 1 << iota

	// tfSpent indicates that a txout is spent.
	tfSpent

	// tfModified indicates that a txout has been modified since it was
	// loaded.
	tfModified
)

// utxoOutput houses details about an individual unspent transaction output such
// as whether or not it is spent, its public key script, and how much it pays.
//
// Standard public key scripts are stored in the database using a compressed
// format. Since the vast majority of scripts are of the standard form, a fairly
// significant savings is achieved by discarding the portions of the standard
// scripts that can be reconstructed.
//
// Also, since it is common for only a specific output in a given utxo entry to
// be referenced from a redeeming transaction, the script and amount for a given
// output is not uncompressed until the first time it is accessed.  This
// provides a mechanism to avoid the overhead of needlessly uncompressing all
// outputs for a given utxo entry at the time of load.
//
// The struct is aligned for memory efficiency.
type UtxoEntry struct {
	amount      types.Amount // The amount of the output.
	pkScript    []byte       // The public key script for the output.
	blockHash   hash.Hash
	packedFlags txoFlags
}

// isModified returns whether or not the output has been modified since it was
// loaded.
func (entry *UtxoEntry) IsModified() bool {
	return entry.packedFlags&tfModified == tfModified
}

func (entry *UtxoEntry) Modified() {
	entry.packedFlags |= tfModified
}

// IsCoinBase returns whether or not the output was contained in a coinbase
// transaction.
func (entry *UtxoEntry) IsCoinBase() bool {
	return entry.packedFlags&tfCoinBase == tfCoinBase
}

func (entry *UtxoEntry) CoinBase() {
	entry.packedFlags |= tfCoinBase
}

// BlockHash returns the hash of the block containing the output.
func (entry *UtxoEntry) BlockHash() *hash.Hash {
	return &entry.blockHash
}

func (entry *UtxoEntry) SetBlockHash(bh *hash.Hash) {
	entry.blockHash = *bh
}

// IsSpent returns whether or not the output has been spent based upon the
// current state of the unspent transaction output view it was obtained from.
func (entry *UtxoEntry) IsSpent() bool {
	return entry.packedFlags&tfSpent == tfSpent
}

// Spend marks the output as spent.  Spending an output that is already spent
// has no effect.
func (entry *UtxoEntry) Spend() {
	// Nothing to do if the output is already spent.
	if entry.IsSpent() {
		return
	}

	// Mark the output as spent and modified.
	entry.packedFlags |= tfSpent | tfModified
}

// Amount returns the amount of the output.
func (entry *UtxoEntry) Amount() types.Amount {
	return entry.amount
}

func (entry *UtxoEntry) SetAmount(amount types.Amount) {
	entry.amount = amount
}

// PkScript returns the public key script for the output.
func (entry *UtxoEntry) PkScript() []byte {
	return entry.pkScript
}

func (entry *UtxoEntry) SetPkScript(pks []byte) {
	entry.pkScript = pks
}

// Clone returns a shallow copy of the utxo entry.
func (entry *UtxoEntry) Clone() *UtxoEntry {
	if entry == nil {
		return nil
	}

	return &UtxoEntry{
		amount:      entry.amount,
		pkScript:    entry.pkScript,
		blockHash:   entry.blockHash,
		packedFlags: entry.packedFlags,
	}
}

func NewUtxoEntry(amount types.Amount, pkScript []byte, blockHash *hash.Hash, isCoinBase bool) *UtxoEntry {
	entry := &UtxoEntry{
		amount:      amount,
		pkScript:    pkScript,
		blockHash:   *blockHash,
		packedFlags: 0,
	}
	if isCoinBase {
		entry.packedFlags |= tfCoinBase
	}
	return entry
}
