package rawdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"path/filepath"
)

// The list of table names of chain freezer.
const (
	ChainFreezerHeaderTable = "headers"

	ChainFreezerBlockTable = "blocks"

	ChainFreezerDAGBlockTable = "dagblocks"
)

// chainFreezerNoSnappy configures whether compression is disabled for the ancient-tables.
var chainFreezerNoSnappy = map[string]bool{
	ChainFreezerHeaderTable:   false,
	ChainFreezerBlockTable:    false,
	ChainFreezerDAGBlockTable: false,
}

const (
	// stateHistoryTableSize defines the maximum size of freezer data files.
	stateHistoryTableSize = 2 * 1000 * 1000 * 1000

	// stateHistoryAccountIndex indicates the name of the freezer state history table.
	stateHistoryMeta         = "history.meta"
	stateHistoryAccountIndex = "account.index"
	stateHistoryStorageIndex = "storage.index"
	stateHistoryAccountData  = "account.data"
	stateHistoryStorageData  = "storage.data"
)

var stateFreezerNoSnappy = map[string]bool{
	stateHistoryMeta:         true,
	stateHistoryAccountIndex: false,
	stateHistoryStorageIndex: false,
	stateHistoryAccountData:  false,
	stateHistoryStorageData:  false,
}

// The list of identifiers of ancient stores.
var (
	chainFreezerName = "chain" // the folder name of chain segment ancient store.
	stateFreezerName = "state" // the folder name of reverse diff ancient store.
)

// freezers the collections of all builtin freezers.
var freezers = []string{chainFreezerName}

// NewStateFreezer initializes the ancient store for state history.
//
//   - if the empty directory is given, initializes the pure in-memory
//     state freezer (e.g. dev mode).
//   - if non-empty directory is given, initializes the regular file-based
//     state freezer.
func NewStateFreezer(ancientDir string, readOnly bool) (ethdb.ResettableAncientStore, error) {
	if ancientDir == "" {
		return NewMemoryFreezer(readOnly, stateFreezerNoSnappy), nil
	}
	return newResettableFreezer(filepath.Join(ancientDir, stateFreezerName), "state", readOnly, stateHistoryTableSize, stateFreezerNoSnappy)
}
