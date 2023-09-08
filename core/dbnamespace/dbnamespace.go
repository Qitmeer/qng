// Package dbnamespace contains constants that define the database namespaces
// for the purpose of the blockchain, so that external callers may easily access
// this data.
package dbnamespace

import (
	"encoding/binary"
)

var (
	// ByteOrder is the preferred byte order used for serializing numeric
	// fields for storage in the database.
	ByteOrder = binary.LittleEndian

	// BCDBInfoBucketName is the name of the database bucket used to house
	// global versioning and date information for the blockchain database.
	BCDBInfoBucketName = []byte("dbinfo")

	// ChainStateKeyName is the name of the db key used to store the best
	// chain state.
	ChainStateKeyName = []byte("chainstate")

	// SpendJournalBucketName is the name of the db bucket used to house
	// transactions outputs that are spent in each block.
	SpendJournalBucketName = []byte("spendjournal")

	// UtxoSetBucketName is the name of the db bucket used to house the
	// unspent transaction output set.
	UtxoSetBucketName = []byte("utxoset")

	// IndexTipsBucketName is the name of the db bucket used to house the
	// current tip of each index.
	IndexTipsBucketName = []byte("idxtips")

	//TokenBucketName is the name of the db bucket used to house the token balance state
	//The balance state is updated by the TOKEN_MINT/TOKEN_UNMINT transactions.
	TokenBucketName = []byte("token")
)
