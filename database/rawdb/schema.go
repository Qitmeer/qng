package rawdb

import (
	"encoding/binary"
	"github.com/Qitmeer/qng/common/hash"
)

// The fields below define the low level database schema prefixing.
var (
	// VersionKeyName is the name of the database key used to house
	// the database version.  It is itself under the BCDBInfoBucketName
	// bucket.
	VersionKey = []byte("version")

	// CompressionVersionKeyName is the name of the database key
	// used to house the database compression version.  It is itself under
	// the BCDBInfoBucketName bucket.
	CompressionVersionKey = []byte("compver")

	// BlockIndexVersionKeyName is the name of the database key
	// used to house the database block index version.  It is itself under
	// the BCDBInfoBucketName bucket.
	BlockIndexVersionKey = []byte("bidxver")

	// CreatedKeyName is the name of the database key used to house
	// date the database was created.  It is itself under the
	// BCDBInfoBucketName bucket.
	CreatedKey = []byte("created")

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

	// Best chain state
	bestChainStateKey = []byte("chainstate")

	// base
	headerPrefix       = []byte("h") // headerPrefix + hash -> header
	blockPrefix        = []byte("b") // blockPrefix + hash -> block
	spendJournalPrefix = []byte("j") // spendJournalPrefix + hash -> SpentTxOuts data
	utxoPrefix         = []byte("u") // utxoPrefix + outpoint data -> UtxoEntry data
	tokenStatePrefix   = []byte("t") // tokenStatePrefix + id (uint64 big endian) -> tokenState data
	// dag
	// DagInfoKey is the name of the db bucket used to house the
	// dag information
	dagInfoKey = []byte("daginfo")

	dagBlockPrefix = []byte("d") // dagBlockPrefix + id (uint64 big endian) -> dag block
	blockIDPrefix  = []byte("i") // block hash -> block id.

	mainchainTipKey    = []byte("MainChainTip") // main chain tip id
	dagMainChainPrefix = []byte("m")            // dagMainChainPrefix + id (uint64 big endian) -> 0

	dagTipsKey      = []byte("dagtips") // main,tip,... ...
	diffAnticoneKey = []byte("diffanticone")
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

// headerKey = headerPrefix + hash
func headerKey(hash *hash.Hash) []byte {
	return append(headerPrefix, hash.Bytes()...)
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

// spendJournalKey = spendJournalPrefix + hash
func spendJournalKey(hash *hash.Hash) []byte {
	return append(spendJournalPrefix, hash.Bytes()...)
}

// utxoKey = utxoPrefix + outpoint data
func utxoKey(opd []byte) []byte {
	return append(utxoPrefix, opd...)
}

// spendJournalKey = tokenStatePrefix + hash
func tokenStateKey(id uint64) []byte {
	return append(tokenStatePrefix, encodeBlockID(id)...)
}

// dagMainChainKey = dagMainChainPrefix + id (uint64 big endian)
func dagMainChainKey(id uint64) []byte {
	return append(dagMainChainPrefix, encodeBlockID(id)...)
}
