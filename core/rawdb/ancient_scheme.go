package rawdb

// The list of table names of chain freezer.
const (
	ChainFreezerBlockTable = "blocks"

	ChainFreezerDAGBlockTable = "dagblocks"
)

// chainFreezerNoSnappy configures whether compression is disabled for the ancient-tables.
// Hashes and difficulties don't compress well.
var chainFreezerNoSnappy = map[string]bool{
	ChainFreezerBlockTable:    false,
	ChainFreezerDAGBlockTable: false,
}

// The list of identifiers of ancient stores.
var (
	chainFreezerName = "chain" // the folder name of chain segment ancient store.
)

// freezers the collections of all builtin freezers.
var freezers = []string{chainFreezerName}
