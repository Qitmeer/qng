package lrucache

type LRUIDCache struct {
	cache    map[uint64]interface{}
	capacity int
}

func NewLRUIDCache(capacity int, preallocate bool) *LRUIDCache {
	var cache map[uint64]interface{}
	if preallocate {
		cache = make(map[uint64]interface{}, capacity+1)
	} else {
		cache = make(map[uint64]interface{})
	}
	return &LRUIDCache{
		cache:    cache,
		capacity: capacity,
	}
}

func (c *LRUIDCache) Add(key uint64, value interface{}) {
	c.cache[key] = value

	if len(c.cache) > c.capacity {
		c.evictRandom()
	}
}

func (c *LRUIDCache) Get(key uint64) (interface{}, bool) {
	value, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	return value, true
}

func (c *LRUIDCache) Has(key uint64) bool {
	_, ok := c.cache[key]
	return ok
}

func (c *LRUIDCache) Remove(key uint64) {
	delete(c.cache, key)
}

func (c *LRUIDCache) evictRandom() {
	var keyToEvict uint64
	for key := range c.cache {
		keyToEvict = key
		break
	}
	c.Remove(keyToEvict)
}
