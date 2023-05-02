package snapshot

import (
	"github.com/Qitmeer/qng/common/hash"
	"sync"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/ethdb"
)

// diskLayer is a low level persistent snapshot built on top of a key-value store.
type diskLayer struct {
	diskdb ethdb.KeyValueStore // Key-value store containing the base snapshot
	cache  *fastcache.Cache    // Cache to avoid hitting the disk for direct access

	root  hash.Hash // Root hash of the base snapshot
	stale bool      // Signals that the layer became stale (state progressed)

	genMarker  []byte        // Marker for the state that's indexed during initial layer generation
	genPending chan struct{} // Notification channel when generation is done (test synchronicity)

	lock sync.RWMutex
}

// Root returns  root hash for which this snapshot was made.
func (dl *diskLayer) Root() hash.Hash {
	return dl.root
}

// Parent always returns nil as there's no layer below the disk.
func (dl *diskLayer) Parent() interface{} {
	return nil
}

// Stale return whether this layer has become stale (was flattened across) or if
// it's still live.
func (dl *diskLayer) Stale() bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return dl.stale
}

func (dl *diskLayer) BlockOrder(order uint64) (uint64, error) {
	return 0, nil
}

func (dl *diskLayer) BlockOrderRLP(order uint64) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return nil, nil
}

func (dl *diskLayer) BlockStatus(id uint64) (interface{}, error) {
	return 0, nil
}

func (dl *diskLayer) BlockStatusRLP(id uint64) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return nil, nil
}
