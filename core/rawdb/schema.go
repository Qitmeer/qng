package rawdb

import (
	"encoding/binary"
	"github.com/Qitmeer/qng/common/hash"
)

// The fields below define the low level database schema prefixing.
var (
	// databaseVersionKey tracks the current database version.
	databaseVersionKey = []byte("DatabaseVersion")

	// snapshotDisabledKey flags that the snapshot should not be maintained due to initial sync.
	snapshotDisabledKey = []byte("SnapshotDisabled")

	// SnapshotRootKey tracks the hash of the last snapshot.
	SnapshotRootKey = []byte("SnapshotRoot")

	// snapshotJournalKey tracks the in-memory diff layers across restarts.
	snapshotJournalKey = []byte("SnapshotJournal")

	// snapshotGeneratorKey tracks the snapshot generation marker across restarts.
	snapshotGeneratorKey = []byte("SnapshotGenerator")

	// snapshotRecoveryKey tracks the snapshot recovery marker across restarts.
	snapshotRecoveryKey = []byte("SnapshotRecovery")

	// snapshotSyncStatusKey tracks the snapshot sync status across restarts.
	snapshotSyncStatusKey = []byte("SnapshotSyncStatus")

	// badBlockKey tracks the list of bad blocks seen by local
	badBlockKey = []byte("InvalidBlock")

	// uncleanShutdownKey tracks the list of local crashes
	uncleanShutdownKey = []byte("Unclean-shutdown") // config prefix for the db

	// base
	blockPrefix = []byte("b") // blockPrefix + hash -> block
	// dag
	dagBlockPrefix = []byte("d") // dagBlockPrefix + id (uint64 big endian) -> dag block
	blockIDPrefix  = []byte("i") // block hash -> block id.

	mainchainTipKey = []byte("MainChainTip") // main chain tip id

	// index
	txLookupPrefix = []byte("l") // txLookupPrefix + hash -> transaction lookup metadata

	// snapshot
	SnapshotBlockOrderPrefix  = []byte("o") // SnapshotBlockOrderPrefix + block order -> block id
	SnapshotBlockStatusPrefix = []byte("s") // SnapshotBlockStatusPrefix + block id -> block status
)

// encodeBlockID encodes a block id as big endian uint64
func encodeBlockID(id uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, id)
	return enc
}

// blockKey = blockPrefix + hash
func blockKey(hash *hash.Hash) []byte {
	return append(blockPrefix, hash.Bytes()...)
}

// dagBlockKey = dagBlockPrefix + id (uint64 big endian)
func dagBlockKey(id uint64) []byte {
	return append(dagBlockPrefix, encodeBlockID(id)...)
}

// blockIDKey = blockIDPrefix + hash
func blockIDKey(hash *hash.Hash) []byte {
	return append(blockIDPrefix, hash.Bytes()...)
}

// txLookupKey = txLookupPrefix + hash
func txLookupKey(hash *hash.Hash) []byte {
	return append(txLookupPrefix, hash.Bytes()...)
}

func blockOrderKey(order uint64) []byte {
	return append(SnapshotBlockOrderPrefix, encodeBlockID(order)...)
}
