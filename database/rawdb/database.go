package rawdb

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/cockroachdb/pebble"
	"github.com/olekukonko/tablewriter"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	ldb "github.com/syndtr/goleveldb/leveldb"
)

// freezerdb is a database wrapper that enabled freezer data retrievals.
type freezerdb struct {
	ancientRoot string
	ethdb.KeyValueStore
	*chainFreezer

	readOnly bool
}

// AncientDatadir returns the path of root ancient directory.
func (frdb *freezerdb) AncientDatadir() (string, error) {
	return frdb.ancientRoot, nil
}

// Close implements io.Closer, closing both the fast key-value store as well as
// the slow ancient tables.
func (frdb *freezerdb) Close() error {
	var errs []error
	if err := frdb.chainFreezer.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := frdb.KeyValueStore.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) != 0 {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// Freeze is a helper method used for external testing to trigger and block until
// a freeze cycle completes, without having to sleep for a minute to trigger the
// automatic background run.
func (frdb *freezerdb) Freeze() error {
	if frdb.readOnly {
		return errReadOnly
	}
	// Trigger a freeze cycle and block until it's done
	trigger := make(chan struct{}, 1)
	frdb.chainFreezer.trigger <- trigger
	<-trigger
	return nil
}

// nofreezedb is a database wrapper that disables freezer data retrievals.
type nofreezedb struct {
	ethdb.KeyValueStore
}

// HasAncient returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) HasAncient(kind string, number uint64) (bool, error) {
	return false, errNotSupported
}

// Ancient returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errNotSupported
}

// AncientRange returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) AncientRange(kind string, start, max, maxByteSize uint64) ([][]byte, error) {
	return nil, errNotSupported
}

// Ancients returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) Ancients() (uint64, error) {
	return 0, errNotSupported
}

// Tail returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) Tail() (uint64, error) {
	return 0, errNotSupported
}

// AncientSize returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) AncientSize(kind string) (uint64, error) {
	return 0, errNotSupported
}

// ModifyAncients is not supported.
func (db *nofreezedb) ModifyAncients(func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, errNotSupported
}

// TruncateHead returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) TruncateHead(items uint64) (uint64, error) {
	return 0, errNotSupported
}

// TruncateTail returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) TruncateTail(items uint64) (uint64, error) {
	return 0, errNotSupported
}

// Sync returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) Sync() error {
	return errNotSupported
}

func (db *nofreezedb) ReadAncients(fn func(reader ethdb.AncientReaderOp) error) (err error) {
	// Unlike other ancient-related methods, this method does not return
	// errNotSupported when invoked.
	// The reason for this is that the caller might want to do several things:
	// 1. Check if something is in freezer,
	// 2. If not, check leveldb.
	//
	// This will work, since the ancient-checks inside 'fn' will return errors,
	// and the leveldb work will continue.
	//
	// If we instead were to return errNotSupported here, then the caller would
	// have to explicitly check for that, having an extra clause to do the
	// non-ancient operations.
	return fn(db)
}

// MigrateTable processes the entries in a given table in sequence
// converting them to a new format if they're of an old format.
func (db *nofreezedb) MigrateTable(kind string, convert convertLegacyFn) error {
	return errNotSupported
}

// AncientDatadir returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) AncientDatadir() (string, error) {
	return "", errNotSupported
}

// NewDatabase creates a high level database on top of a given key-value data
// store without a freezer moving immutable chain segments into cold storage.
func NewDatabase(db ethdb.KeyValueStore) ethdb.Database {
	return &nofreezedb{KeyValueStore: db}
}

// resolveChainFreezerDir is a helper function which resolves the absolute path
// of chain freezer by considering backward compatibility.
func resolveChainFreezerDir(ancient string) string {
	// Check if the chain freezer is already present in the specified
	// sub folder, if not then two possibilities:
	// - chain freezer is not initialized
	// - chain freezer exists in legacy location (root ancient folder)
	freezer := filepath.Join(ancient, chainFreezerName)
	if !common.FileExist(freezer) {
		if !common.FileExist(ancient) {
			// The entire ancient store is not initialized, still use the sub
			// folder for initialization.
		} else {
			// Ancient root is already initialized, then we hold the assumption
			// that chain freezer is also initialized and located in root folder.
			// In this case fallback to legacy location.
			freezer = ancient
			log.Info("Found legacy ancient chain path", "location", ancient)
		}
	}
	return freezer
}

// NewDatabaseWithFreezer creates a high level database on top of a given key-
// value data store with a freezer moving immutable chain segments into cold
// storage. The passed ancient indicates the path of root ancient directory
// where the chain freezer can be opened.
func NewDatabaseWithFreezer(db ethdb.KeyValueStore, ancient string, namespace string, readonly bool) (ethdb.Database, error) {
	// Create the idle freezer instance. If the given ancient directory is empty,
	// in-memory chain freezer is used (e.g. dev mode); otherwise the regular
	// file-based freezer is created.
	chainFreezerDir := ancient
	if chainFreezerDir != "" {
		chainFreezerDir = resolveChainFreezerDir(chainFreezerDir)
	}
	frdb, err := newChainFreezer(chainFreezerDir, namespace, readonly)
	if err != nil {
		printChainMetadata(db)
		return nil, err
	}
	// TODO: Data validity check
	// Freezer is consistent with the key-value database, permit combining the two
	if !readonly {
		frdb.wg.Add(1)
		go func() {
			frdb.freeze(db)
			frdb.wg.Done()
		}()
	}
	return &freezerdb{
		ancientRoot:   ancient,
		KeyValueStore: db,
		chainFreezer:  frdb,
	}, nil
}

// NewMemoryDatabase creates an ephemeral in-memory key-value database without a
// freezer moving immutable chain segments into cold storage.
func NewMemoryDatabase() ethdb.Database {
	return NewDatabase(memorydb.New())
}

// NewMemoryDatabaseWithCap creates an ephemeral in-memory key-value database
// with an initial starting capacity, but without a freezer moving immutable
// chain segments into cold storage.
func NewMemoryDatabaseWithCap(size int) ethdb.Database {
	return NewDatabase(memorydb.NewWithCap(size))
}

// NewLevelDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
func NewLevelDBDatabase(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	db, err := leveldb.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	log.Info("Using LevelDB as the backing database")
	return NewDatabase(db), nil
}

const (
	dbPebble  = "pebble"
	dbLeveldb = "leveldb"
)

// PreexistingDatabase checks the given data directory whether a database is already
// instantiated at that location, and if so, returns the type of database (or the
// empty string).
func PreexistingDatabase(path string) string {
	if _, err := os.Stat(filepath.Join(path, "CURRENT")); err != nil {
		return "" // No pre-existing db
	}
	if matches, err := filepath.Glob(filepath.Join(path, "OPTIONS*")); len(matches) > 0 || err != nil {
		if err != nil {
			panic(err) // only possible if the pattern is malformed
		}
		return dbPebble
	}
	return dbLeveldb
}

// OpenOptions contains the options to apply when opening a database.
// OBS: If AncientsDirectory is empty, it indicates that no freezer is to be used.
type OpenOptions struct {
	Type              string // "leveldb" | "pebble"
	Directory         string // the datadir
	AncientsDirectory string // the ancients-dir
	Namespace         string // the namespace for database relevant metrics
	Cache             int    // the capacity(in megabytes) of the data caching
	Handles           int    // number of files to be open simultaneously
	ReadOnly          bool
	// Ephemeral means that filesystem sync operations should be avoided: data integrity in the face of
	// a crash is not important. This option should typically be used in tests.
	Ephemeral bool
}

// openKeyValueDatabase opens a disk-based key-value database, e.g. leveldb or pebble.
//
//	                      type == null          type != null
//	                   +----------------------------------------
//	db is non-existent |  pebble default  |  specified type
//	db is existent     |  from db         |  specified type (if compatible)
func openKeyValueDatabase(o OpenOptions) (ethdb.Database, error) {
	// Reject any unsupported database type
	if len(o.Type) != 0 && o.Type != dbLeveldb && o.Type != dbPebble {
		return nil, fmt.Errorf("unknown db.engine %v", o.Type)
	}
	// Retrieve any pre-existing database's type and use that or the requested one
	// as long as there's no conflict between the two types
	existingDb := PreexistingDatabase(o.Directory)
	if len(existingDb) != 0 && len(o.Type) != 0 && o.Type != existingDb {
		return nil, fmt.Errorf("db.engine choice was %v but found pre-existing %v database in specified data directory", o.Type, existingDb)
	}
	if o.Type == dbPebble || existingDb == dbPebble {
		if PebbleEnabled {
			log.Info("Using pebble as the backing database")
			return NewPebbleDBDatabase(o.Directory, o.Cache, o.Handles, o.Namespace, o.ReadOnly, o.Ephemeral)
		} else {
			return nil, errors.New("db.engine 'pebble' not supported on this platform")
		}
	}
	if o.Type == dbLeveldb || existingDb == dbLeveldb {
		log.Info("Using leveldb as the backing database")
		return NewLevelDBDatabase(o.Directory, o.Cache, o.Handles, o.Namespace, o.ReadOnly)
	}
	// No pre-existing database, no user-requested one either. Default to Pebble
	// on supported platforms and LevelDB on anything else.
	if PebbleEnabled {
		log.Info("Defaulting to pebble as the backing database")
		return NewPebbleDBDatabase(o.Directory, o.Cache, o.Handles, o.Namespace, o.ReadOnly, o.Ephemeral)
	} else {
		log.Info("Defaulting to leveldb as the backing database")
		return NewLevelDBDatabase(o.Directory, o.Cache, o.Handles, o.Namespace, o.ReadOnly)
	}
}

// Open opens both a disk-based key-value database such as leveldb or pebble, but also
// integrates it with a freezer database -- if the AncientDir option has been
// set on the provided OpenOptions.
// The passed o.AncientDir indicates the path of root ancient directory where
// the chain freezer can be opened.
func Open(o OpenOptions) (ethdb.Database, error) {
	kvdb, err := openKeyValueDatabase(o)
	if err != nil {
		return nil, err
	}
	if len(o.AncientsDirectory) == 0 {
		return kvdb, nil
	}
	frdb, err := NewDatabaseWithFreezer(kvdb, o.AncientsDirectory, o.Namespace, o.ReadOnly)
	if err != nil {
		kvdb.Close()
		return nil, err
	}
	return frdb, nil
}

type counter uint64

func (c counter) String() string {
	return fmt.Sprintf("%d", c)
}

func (c counter) Percentage(current uint64) string {
	return fmt.Sprintf("%d", current*100/uint64(c))
}

// stat stores sizes and count for a parameter
type stat struct {
	size  common.StorageSize
	count counter
}

// Add size to the stat and increase the counter by 1
func (s *stat) Add(size common.StorageSize) {
	s.size += size
	s.count++
}

func (s *stat) Size() string {
	return s.size.String()
}

func (s *stat) Count() string {
	return s.count.String()
}

// InspectDatabase traverses the entire database and checks the size
// of all different categories of data.
func InspectDatabase(db ethdb.Database, keyPrefix, keyStart []byte) error {
	it := db.NewIterator(keyPrefix, keyStart)
	defer it.Release()

	var (
		count  int64
		start  = time.Now()
		logged = time.Now()

		// Key-value store statistics
		headers             stat
		bodies              stat
		spendJournal        stat
		utxo                stat
		tokenState          stat
		dagBlock            stat
		blockID             stat
		dagMainChain        stat
		txLookup            stat
		txFullHash          stat
		invalidtxLookup     stat
		invalidtxFullHash   stat
		SnapshotBlockOrder  stat
		SnapshotBlockStatus stat
		addridx             stat

		// Meta- and unaccounted data
		metadata    stat
		unaccounted stat
		// Totals
		total common.StorageSize
	)
	// Inspect key-value database first.
	for it.Next() {
		var (
			key  = it.Key()
			size = common.StorageSize(len(key) + len(it.Value()))
		)
		total += size
		switch {
		case bytes.HasPrefix(key, headerPrefix) && len(key) == (len(headerPrefix)+common.HashLength):
			headers.Add(size)
		case bytes.HasPrefix(key, blockPrefix) && len(key) == (len(blockPrefix)+common.HashLength):
			bodies.Add(size)
		case bytes.HasPrefix(key, spendJournalPrefix) && len(key) == (len(spendJournalPrefix)+common.HashLength):
			spendJournal.Add(size)
		case bytes.HasPrefix(key, utxoPrefix):
			utxo.Add(size)
		case bytes.HasPrefix(key, tokenStatePrefix) && len(key) == (len(tokenStatePrefix)+8):
			tokenState.Add(size)
		case bytes.HasPrefix(key, dagBlockPrefix) && len(key) == (len(dagBlockPrefix)+8):
			dagBlock.Add(size)
		case bytes.HasPrefix(key, blockIDPrefix) && len(key) == (len(blockIDPrefix)+common.HashLength):
			blockID.Add(size)
		case bytes.HasPrefix(key, dagMainChainPrefix) && len(key) == (len(dagMainChainPrefix)+8):
			dagMainChain.Add(size)
		case bytes.HasPrefix(key, txLookupPrefix) && len(key) == (len(txLookupPrefix)+common.HashLength):
			txLookup.Add(size)
		case bytes.HasPrefix(key, txFullHashPrefix) && len(key) == (len(txFullHashPrefix)+common.HashLength):
			txFullHash.Add(size)
		case bytes.HasPrefix(key, invalidtxLookupPrefix) && len(key) == (len(invalidtxLookupPrefix)+common.HashLength):
			invalidtxLookup.Add(size)
		case bytes.HasPrefix(key, invalidtxFullHashPrefix) && len(key) == (len(invalidtxFullHashPrefix)+common.HashLength):
			invalidtxFullHash.Add(size)

		case bytes.HasPrefix(key, SnapshotBlockOrderPrefix) && len(key) == (len(SnapshotBlockOrderPrefix)+8):
			SnapshotBlockOrder.Add(size)
		case bytes.HasPrefix(key, SnapshotBlockStatusPrefix) && len(key) == (len(SnapshotBlockStatusPrefix)+8):
			SnapshotBlockStatus.Add(size)
		case bytes.HasPrefix(key, AddridxPrefix):
			addridx.Add(size)
		default:
			var accounted bool
			for _, meta := range [][]byte{VersionKey, CompressionVersionKey, BlockIndexVersionKey, CreatedKey,
				snapshotDisabledKey, SnapshotRootKey, snapshotJournalKey, snapshotGeneratorKey, snapshotRecoveryKey, snapshotSyncStatusKey,
				badBlockKey, uncleanShutdownKey, bestChainStateKey, dagInfoKey, mainchainTipKey, dagTipsKey, diffAnticoneKey, EstimateFeeDatabaseKey,
				addridxTipKey,
			} {
				if bytes.Equal(key, meta) {
					metadata.Add(size)
					accounted = true
					break
				}
			}
			if !accounted {
				unaccounted.Add(size)
			}
		}
		count++
		if count%1000 == 0 && time.Since(logged) > 8*time.Second {
			log.Info("Inspecting database", "count", count, "elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
	}
	// Display the database statistic of key-value store.
	stats := [][]string{
		{"Key-Value store", "Headers", headers.Size(), headers.Count()},
		{"Key-Value store", "Bodies", bodies.Size(), bodies.Count()},
		{"Key-Value store", "SpendJournal", spendJournal.Size(), spendJournal.Count()},
		{"Key-Value store", "UTXO", utxo.Size(), utxo.Count()},
		{"Key-Value store", "TokenState", tokenState.Size(), tokenState.Count()},
		{"Key-Value store", "DAGBlock", dagBlock.Size(), dagBlock.Count()},
		{"Key-Value store", "BlockID", blockID.Size(), blockID.Count()},
		{"Key-Value store", "DAGMainChain", dagMainChain.Size(), dagMainChain.Count()},
		{"Key-Value store", "TxLookup", txLookup.Size(), txLookup.Count()},
		{"Key-Value store", "TxFullHash", txFullHash.Size(), txFullHash.Count()},
		{"Key-Value store", "InvalidTxLookup", invalidtxLookup.Size(), invalidtxLookup.Count()},
		{"Key-Value store", "InvalidTxFullHash", invalidtxFullHash.Size(), invalidtxFullHash.Count()},
		{"Key-Value store", "SnapshotBlockOrder", SnapshotBlockOrder.Size(), SnapshotBlockOrder.Count()},
		{"Key-Value store", "SnapshotBlockStatus", SnapshotBlockStatus.Size(), SnapshotBlockStatus.Count()},
		{"Key-Value store", "Addridx", addridx.Size(), addridx.Count()},
	}
	// Inspect all registered append-only file store then.
	ancients, err := inspectFreezers(db)
	if err != nil {
		return err
	}
	for _, ancient := range ancients {
		for _, table := range ancient.sizes {
			stats = append(stats, []string{
				fmt.Sprintf("Ancient store (%s)", strings.Title(ancient.name)),
				strings.Title(table.name),
				table.size.String(),
				fmt.Sprintf("%d", ancient.count()),
			})
		}
		total += ancient.size()
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Database", "Category", "Size", "Items"})
	table.SetFooter([]string{"", "Total", total.String(), " "})
	table.AppendBulk(stats)
	table.Render()

	if unaccounted.size > 0 {
		log.Error("Database contains unaccounted data", "size", unaccounted.size, "count", unaccounted.count)
	}
	return nil
}

// printChainMetadata prints out chain metadata to stderr.
func printChainMetadata(db ethdb.KeyValueStore) {
	fmt.Fprintf(os.Stderr, "Chain metadata\n")
	for _, v := range ReadChainMetadata(db) {
		fmt.Fprintf(os.Stderr, "  %s\n", strings.Join(v, ": "))
	}
	fmt.Fprintf(os.Stderr, "\n\n")
}

// ReadChainMetadata returns a set of key/value pairs that contains informatin
// about the database chain status. This can be used for diagnostic purposes
// when investigating the state of the node.
func ReadChainMetadata(db ethdb.KeyValueStore) [][]string {
	pp := func(val *uint64) string {
		if val == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%d (%#x)", *val, *val)
	}
	pp32 := func(val *uint32) string {
		if val == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%d (%#x)", *val, *val)
	}
	data := [][]string{
		{"databaseVersion", pp32(ReadDatabaseVersion(db))},
		{"len(snapshotSyncStatus)", fmt.Sprintf("%d bytes", len(ReadSnapshotSyncStatus(db)))},
		{"snapshotDisabled", fmt.Sprintf("%v", ReadSnapshotDisabled(db))},
		{"snapshotJournal", fmt.Sprintf("%d bytes", len(ReadSnapshotJournal(db)))},
		{"snapshotRecoveryNumber", pp(ReadSnapshotRecoveryNumber(db))},
		{"snapshotRoot", fmt.Sprintf("%v", ReadSnapshotRoot(db))},
	}
	return data
}

func isErrNotFound(err error) bool {
	return err == pebble.ErrNotFound || err == ldb.ErrNotFound || strings.Contains(err.Error(), "not found")
}

func isErrWithoutNotFound(err error) bool {
	if err == nil {
		return false
	}
	return !isErrNotFound(err)
}
