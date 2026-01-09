package knownbots

import (
	"container/list"
	"sync"
)

// LRU is an LRU cache for failed RDNS lookups.
// It provides fast rejection of IP addresses that have already
// been verified as invalid bot sources.
type LRU struct {
	mu    sync.RWMutex
	limit int
	items map[string]*list.Element
	list  *list.List
}

// entry holds an LRU cache entry.
type lruEntry struct {
	key string
}

// NewLRU creates a new LRU cache with the specified limit.
func NewLRU(limit int) *LRU {
	return &LRU{
		limit: limit,
		items: make(map[string]*list.Element),
		list:  list.New(),
	}
}

// Add inserts a key into the cache. If the key already exists,
// it moves the entry to the front. If the cache is full,
// the least recently used entry is evicted.
func (l *LRU) Add(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if el, ok := l.items[key]; ok {
		// Key exists, move to front
		l.list.MoveToFront(el)
		return
	}

	// Add new entry
	el := l.list.PushFront(&lruEntry{key: key})
	l.items[key] = el

	// Evict least recently used if over limit
	for l.list.Len() > l.limit {
		el := l.list.Back()
		if el == nil {
			break
		}
		entry := el.Value.(*lruEntry)
		delete(l.items, entry.key)
		l.list.Remove(el)
	}
}

// Contains returns true if the key exists in the cache.
// Uses read lock for better concurrency.
func (l *LRU) Contains(key string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	_, ok := l.items[key]
	return ok
}
