package rawdb

import (
	"fmt"
	"github.com/Qitmeer/qng/meerdag"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// freezerRecheckInterval is the frequency to check the key-value database for
	// chain progression that might permit new blocks to be frozen into immutable
	// storage.
	freezerRecheckInterval = time.Minute

	// freezerBatchLimit is the maximum number of blocks to freeze in one batch
	// before doing an fsync and deleting it from the key-value store.
	freezerBatchLimit = 30000
)

// chainFreezer is a wrapper of freezer with additional chain freezing feature.
// The background thread will keep moving ancient chain segments from key-value
// database to flat files for saving space on live database.
type chainFreezer struct {
	// WARNING: The `threshold` field is accessed atomically. On 32 bit platforms, only
	// 64-bit aligned fields can be atomic. The struct is guaranteed to be so aligned,
	// so take advantage of that (https://golang.org/pkg/sync/atomic/#pkg-note-BUG).
	threshold uint64 // Number of recent blocks not to freeze (params.FullImmutabilityThreshold apart from tests)

	*Freezer
	quit    chan struct{}
	wg      sync.WaitGroup
	trigger chan chan struct{} // Manual blocking freeze trigger, test determinism
}

// newChainFreezer initializes the freezer for ancient chain data.
func newChainFreezer(datadir string, namespace string, readonly bool) (*chainFreezer, error) {
	freezer, err := NewChainFreezer(datadir, namespace, readonly)
	if err != nil {
		return nil, err
	}
	return &chainFreezer{
		Freezer:   freezer,
		threshold: params.FullImmutabilityThreshold,
		quit:      make(chan struct{}),
		trigger:   make(chan chan struct{}),
	}, nil
}

// Close closes the chain freezer instance and terminates the background thread.
func (f *chainFreezer) Close() error {
	select {
	case <-f.quit:
	default:
		close(f.quit)
	}
	f.wg.Wait()
	return f.Freezer.Close()
}

// freeze is a background thread that periodically checks the blockchain for any
// import progress and moves ancient data from the fast database into the freezer.
//
// This functionality is deliberately broken off from block importing to avoid
// incurring additional data shuffling delays on block propagation.
func (f *chainFreezer) freeze(db ethdb.KeyValueStore) {
	var (
		backoff   bool
		triggered chan struct{} // Used in tests
		nfdb      = &nofreezedb{KeyValueStore: db}
	)
	timer := time.NewTimer(freezerRecheckInterval)
	defer timer.Stop()

	for {
		select {
		case <-f.quit:
			log.Info("Freezer shutting down")
			return
		default:
		}
		if backoff {
			// If we were doing a manual trigger, notify it
			if triggered != nil {
				triggered <- struct{}{}
				triggered = nil
			}
			select {
			case <-timer.C:
				backoff = false
				timer.Reset(freezerRecheckInterval)
			case triggered = <-f.trigger:
				backoff = false
			case <-f.quit:
				return
			}
		}
		// Retrieve the freezing threshold.
		mt := ReadMainChainTip(nfdb)
		if mt == nil {
			log.Debug("Current full block hash unavailable") // new chain, empty database
			backoff = true
			continue
		}
		mb := ReadDAGBlock(nfdb, *mt)
		if mb == nil {
			log.Debug("Current full block hash unavailable") // new chain, empty database
			backoff = true
			continue
		}
		threshold := atomic.LoadUint64(&f.threshold)
		frozen := atomic.LoadUint64(&f.frozen)
		switch {
		case *mt < threshold:
			log.Debug("Current full block not old enough", "DAG_ID", *mt, "hash", mb.GetHash(), "delay", threshold)
			backoff = true
			continue

		case *mt-threshold <= frozen:
			log.Debug("Ancient blocks frozen already", "DAG_ID", *mt, "hash", mb.GetHash(), "frozen", frozen)
			backoff = true
			continue
		}

		// Seems we have data ready to be frozen, process in usable batches
		var (
			start    = time.Now()
			first, _ = f.Ancients()
			limit    = *mt - threshold
		)
		if limit-first > freezerBatchLimit {
			limit = first + freezerBatchLimit
		}
		ancients, err := f.freezeRange(nfdb, first, limit)
		if err != nil {
			log.Error("Error in block freeze operation", "err", err)
			backoff = true
			continue
		}

		// Batch of blocks have been frozen, flush them before wiping from leveldb
		if err := f.Sync(); err != nil {
			log.Crit("Failed to flush frozen tables", "err", err)
		}

		// Wipe out all data from the active database
		batch := db.NewBatch()
		for i := 0; i < len(ancients); i++ {
			// Always keep the genesis block in active database
			if first+uint64(i) != 0 {
				DeleteBlock(batch, ancients[i].GetHash())
				DeleteDAGBlock(batch, uint64(ancients[i].GetID()))
			}
		}
		if err := batch.Write(); err != nil {
			log.Crit("Failed to delete frozen canonical blocks", "err", err)
		}
		batch.Reset()

		// Log something friendly for the user
		context := []interface{}{
			"blocks", frozen - first, "elapsed", common.PrettyDuration(time.Since(start)), "DAG_ID", frozen - 1,
		}
		if n := len(ancients); n > 0 {
			context = append(context, []interface{}{"hash", ancients[n-1].GetHash()}...)
		}
		log.Debug("Deep froze chain segment", context...)

		// Avoid database thrashing with tiny writes
		if frozen-first < freezerBatchLimit {
			backoff = true
		}
	}
}

func (f *chainFreezer) freezeRange(nfdb *nofreezedb, id, limit uint64) ([]meerdag.IBlock, error) {
	blocks := make([]meerdag.IBlock, 0, limit-id)

	_, err := f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for ; id <= limit; id++ {
			// Retrieve all the components of the canonical block.
			mb := ReadDAGBlock(nfdb, id)
			if mb == nil {
				return fmt.Errorf("canonical hash missing, can't freeze block %d", id)
			}
			block := ReadBlockBaw(nfdb, mb.GetHash())
			if len(block) == 0 {
				return fmt.Errorf("block header missing, can't freeze block %d %s", id, mb.GetHash().String())
			}

			// Write to the batch.
			if err := op.AppendRaw(ChainFreezerBlockTable, id, block); err != nil {
				return fmt.Errorf("can't write hash to Freezer: %v", err)
			}
			if err := op.AppendRaw(ChainFreezerDAGBlockTable, id, mb.Bytes()); err != nil {
				return fmt.Errorf("can't write header to Freezer: %v", err)
			}
			blocks = append(blocks, mb)
		}
		return nil
	})

	return blocks, err
}
