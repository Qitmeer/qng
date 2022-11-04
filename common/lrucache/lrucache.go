package lrucache

import (
	"github.com/Qitmeer/qng/common/hash"
)

// LRUCache is a least-recently-used cache for any type
// that's able to be indexed by Hash
type LRUCache struct {
	cache    map[hash.Hash]interface{}
	capacity int
}

// New creates a new LRUCache
func New(capacity int, preallocate bool) *LRUCache {
	var cache map[hash.Hash]interface{}
	if preallocate {
		cache = make(map[hash.Hash]interface{}, capacity+1)
	} else {
		cache = make(map[hash.Hash]interface{})
	}
	return &LRUCache{
		cache:    cache,
		capacity: capacity,
	}
}

// Add adds an entry to the LRUCache
func (c *LRUCache) Add(key *hash.Hash, value interface{}) {
	c.cache[*key] = value

	if len(c.cache) > c.capacity {
		c.evictRandom()
	}
}

// Get returns the entry for the given key, or (nil, false) otherwise
func (c *LRUCache) Get(key *hash.Hash) (interface{}, bool) {
	value, ok := c.cache[*key]
	if !ok {
		return nil, false
	}
	return value, true
}

// Has returns whether the LRUCache contains the given key
func (c *LRUCache) Has(key *hash.Hash) bool {
	_, ok := c.cache[*key]
	return ok
}

// Remove removes the entry for the the given key. Does nothing if
// the entry does not exist
func (c *LRUCache) Remove(key *hash.Hash) {
	delete(c.cache, *key)
}

func (c *LRUCache) evictRandom() {
	var keyToEvict hash.Hash
	for key := range c.cache {
		keyToEvict = key
		break
	}
	c.Remove(&keyToEvict)
}
