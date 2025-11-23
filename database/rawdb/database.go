package rawdb

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Qitmeer/qng/common/util"
	"github.com/cockroachdb/pebble"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	ldb "github.com/syndtr/goleveldb/leveldb"
)

// freezerdb is a database wrapper that enables ancient chain segment freezing.
type freezerdb struct {
	ethdb.KeyValueStore
	*chainFreezer

	readOnly    bool
	ancientRoot string
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

// Ancient returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errNotSupported
}

// AncientRange returns an error as we don't have a backing chain freezer.
func (db *nofreezedb) AncientRange(kind string, start, max, maxByteSize uint64) ([][]byte, error) {
	return nil, errNotSupported
}

// AncientBytes retrieves the value segment of the element specified by the id
// and value offsets.
func (db *nofreezedb) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
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
func (db *nofreezedb) SyncAncient() error {
	return errNotSupported
}

func (db *nofreezedb) ReadAncients(fn func(reader ethdb.AncientReaderOp) error) (err error) {
	// Unlike other ancient-related methods, this method does not return
	// errNotSupported when invoked.
	// The reason for this is that the caller might want to do several things:
	// 1. Check if something is in the freezer,
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
	freezer := filepath.Join(ancient, ChainFreezerName)
	if !common.FileExist(freezer) {
		if !common.FileExist(ancient) || !util.IsNonEmptyDir(ancient) {
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

// resolveChainEraDir is a helper function which resolves the absolute path of era database.
func resolveChainEraDir(chainFreezerDir string, era string) string {
	switch {
	case era == "":
		return filepath.Join(chainFreezerDir, "era")
	case !filepath.IsAbs(era):
		return filepath.Join(chainFreezerDir, era)
	default:
		return era
	}
}

// NewDatabaseWithFreezer creates a high level database on top of a given key-value store.
// The passed ancient indicates the path of root ancient directory where the chain freezer
// can be opened.
//
// Deprecated: use Open.
func NewDatabaseWithFreezer(db ethdb.KeyValueStore, ancient string, namespace string, readonly bool) (ethdb.Database, error) {
	return Open(db, OpenOptions{
		Ancient:          ancient,
		MetricsNamespace: namespace,
		ReadOnly:         readonly,
	})
}

// OpenOptions specifies options for opening the database.
type OpenOptions struct {
	Ancient          string // ancients directory
	Era              string // era files directory
	MetricsNamespace string // prefix added to freezer metric names
	ReadOnly         bool
}

// Open creates a high-level database wrapper for the given key-value store.
func Open(db ethdb.KeyValueStore, opts OpenOptions) (ethdb.Database, error) {
	// Create the idle freezer instance. If the given ancient directory is empty,
	// in-memory chain freezer is used (e.g. dev mode); otherwise the regular
	// file-based freezer is created.
	chainFreezerDir := opts.Ancient
	if chainFreezerDir != "" {
		chainFreezerDir = resolveChainFreezerDir(chainFreezerDir)
	}
	frdb, err := newChainFreezer(chainFreezerDir, opts.Era, opts.MetricsNamespace, opts.ReadOnly)
	if err != nil {
		printChainMetadata(db)
		return nil, err
	}
	// TODO: Data validity check
	// Freezer is consistent with the key-value database, permit combining the two
	if !opts.ReadOnly {
		frdb.wg.Add(1)
		go func() {
			frdb.freeze(db)
			frdb.wg.Done()
		}()
	}
	return &freezerdb{
		ancientRoot:   opts.Ancient,
		KeyValueStore: db,
		chainFreezer:  frdb,
	}, nil
}

// NewMemoryDatabase creates an ephemeral in-memory key-value database without a
// freezer moving immutable chain segments into cold storage.
func NewMemoryDatabase() ethdb.Database {
	return NewDatabase(memorydb.New())
}

const (
	DBPebble  = "pebble"
	DBLeveldb = "leveldb"
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
		return DBPebble
	}
	return DBLeveldb
}

type counter uint64

func (c counter) String() string {
	return fmt.Sprintf("%d", c)
}

func (c counter) Percentage(current uint64) string {
	return fmt.Sprintf("%d", current*100/uint64(c))
}

// stat provides lock-free statistics aggregation using atomic operations
type stat struct {
	size  uint64
	count uint64
}

func (s *stat) empty() bool {
	return atomic.LoadUint64(&s.count) == 0
}

func (s *stat) add(size common.StorageSize) {
	atomic.AddUint64(&s.size, uint64(size))
	atomic.AddUint64(&s.count, 1)
}

func (s *stat) sizeString() string {
	return common.StorageSize(atomic.LoadUint64(&s.size)).String()
}

func (s *stat) countString() string {
	return counter(atomic.LoadUint64(&s.count)).String()
}

// InspectDatabase traverses the entire database and checks the size
// of all different categories of data.
func InspectDatabase(db ethdb.Database, keyPrefix, keyStart []byte) error {
	var (
		start = time.Now()
		count atomic.Int64
		total atomic.Uint64

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
		unaccounted stat

		// This map tracks example keys for unaccounted data.
		// For each unique two-byte prefix, the first unaccounted key encountered
		// by the iterator will be stored.
		unaccountedKeys = make(map[[2]byte][]byte)
		unaccountedMu   sync.Mutex
	)

	inspectRange := func(ctx context.Context, r byte) error {
		var s []byte
		if len(keyStart) > 0 {
			switch {
			case r < keyStart[0]:
				return nil
			case r == keyStart[0]:
				s = keyStart[1:]
			default:
				// entire key range is included for inspection
			}
		}
		it := db.NewIterator(append(keyPrefix, r), s)
		defer it.Release()

		for it.Next() {
			var (
				key  = it.Key()
				size = common.StorageSize(len(key) + len(it.Value()))
			)
			total.Add(uint64(size))
			count.Add(1)

			switch {
			case bytes.HasPrefix(key, headerPrefix) && len(key) == (len(headerPrefix)+common.HashLength):
				headers.add(size)
			case bytes.HasPrefix(key, blockPrefix) && len(key) == (len(blockPrefix)+common.HashLength):
				bodies.add(size)
			case bytes.HasPrefix(key, spendJournalPrefix) && len(key) == (len(spendJournalPrefix)+common.HashLength):
				spendJournal.add(size)
			case bytes.HasPrefix(key, utxoPrefix):
				utxo.add(size)
			case bytes.HasPrefix(key, tokenStatePrefix) && len(key) == (len(tokenStatePrefix)+8):
				tokenState.add(size)
			case bytes.HasPrefix(key, dagBlockPrefix) && len(key) == (len(dagBlockPrefix)+8):
				dagBlock.add(size)
			case bytes.HasPrefix(key, blockIDPrefix) && len(key) == (len(blockIDPrefix)+common.HashLength):
				blockID.add(size)
			case bytes.HasPrefix(key, dagMainChainPrefix) && len(key) == (len(dagMainChainPrefix)+8):
				dagMainChain.add(size)
			case bytes.HasPrefix(key, txLookupPrefix) && len(key) == (len(txLookupPrefix)+common.HashLength):
				txLookup.add(size)
			case bytes.HasPrefix(key, txFullHashPrefix) && len(key) == (len(txFullHashPrefix)+common.HashLength):
				txFullHash.add(size)
			case bytes.HasPrefix(key, invalidtxLookupPrefix) && len(key) == (len(invalidtxLookupPrefix)+common.HashLength):
				invalidtxLookup.add(size)
			case bytes.HasPrefix(key, invalidtxFullHashPrefix) && len(key) == (len(invalidtxFullHashPrefix)+common.HashLength):
				invalidtxFullHash.add(size)

			case bytes.HasPrefix(key, SnapshotBlockOrderPrefix) && len(key) == (len(SnapshotBlockOrderPrefix)+8):
				SnapshotBlockOrder.add(size)
			case bytes.HasPrefix(key, SnapshotBlockStatusPrefix) && len(key) == (len(SnapshotBlockStatusPrefix)+8):
				SnapshotBlockStatus.add(size)
			case bytes.HasPrefix(key, AddridxPrefix):
				addridx.add(size)

			default:
				unaccounted.add(size)
				if len(key) >= 2 {
					prefix := [2]byte(key[:2])
					unaccountedMu.Lock()
					if _, ok := unaccountedKeys[prefix]; !ok {
						unaccountedKeys[prefix] = bytes.Clone(key)
					}
					unaccountedMu.Unlock()
				}
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		return it.Error()
	}

	var (
		eg, ctx = errgroup.WithContext(context.Background())
		workers = runtime.NumCPU()
	)
	eg.SetLimit(workers)

	// Progress reporter
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Info("Inspecting database", "count", count.Load(), "size", common.StorageSize(total.Load()), "elapsed", common.PrettyDuration(time.Since(start)))
			case <-done:
				return
			}
		}
	}()

	// Inspect key-value database in parallel.
	for i := 0; i < 256; i++ {
		eg.Go(func() error { return inspectRange(ctx, byte(i)) })
	}

	if err := eg.Wait(); err != nil {
		close(done)
		return err
	}
	close(done)

	// Display the database statistic of key-value store.
	stats := [][]string{
		{"Key-Value store", "Headers", headers.sizeString(), headers.countString()},
		{"Key-Value store", "Bodies", bodies.sizeString(), bodies.countString()},
		{"Key-Value store", "SpendJournal", spendJournal.sizeString(), spendJournal.countString()},
		{"Key-Value store", "UTXO", utxo.sizeString(), utxo.countString()},
		{"Key-Value store", "TokenState", tokenState.sizeString(), tokenState.countString()},
		{"Key-Value store", "DAGBlock", dagBlock.sizeString(), dagBlock.countString()},
		{"Key-Value store", "BlockID", blockID.sizeString(), blockID.countString()},
		{"Key-Value store", "DAGMainChain", dagMainChain.sizeString(), dagMainChain.countString()},
		{"Key-Value store", "TxLookup", txLookup.sizeString(), txLookup.countString()},
		{"Key-Value store", "TxFullHash", txFullHash.sizeString(), txFullHash.countString()},
		{"Key-Value store", "InvalidTxLookup", invalidtxLookup.sizeString(), invalidtxLookup.countString()},
		{"Key-Value store", "InvalidTxFullHash", invalidtxFullHash.sizeString(), invalidtxFullHash.countString()},
		{"Key-Value store", "SnapshotBlockOrder", SnapshotBlockOrder.sizeString(), SnapshotBlockOrder.countString()},
		{"Key-Value store", "SnapshotBlockStatus", SnapshotBlockStatus.sizeString(), SnapshotBlockStatus.countString()},
		{"Key-Value store", "Addridx", addridx.sizeString(), addridx.countString()},
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
		total.Add(uint64(ancient.size()))
	}

	table := newTableWriter(os.Stdout)
	table.SetHeader([]string{"Database", "Category", "Size", "Items"})
	table.SetFooter([]string{"", "Total", common.StorageSize(total.Load()).String(), fmt.Sprintf("%d", count.Load())})
	table.AppendBulk(stats)
	table.Render()

	if !unaccounted.empty() {
		log.Error("Database contains unaccounted data", "size", unaccounted.sizeString(), "count", unaccounted.countString())
		for _, e := range slices.SortedFunc(maps.Values(unaccountedKeys), bytes.Compare) {
			log.Error(fmt.Sprintf("   example key: %x", e))
		}
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
