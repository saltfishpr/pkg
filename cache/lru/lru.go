// Package lru provides a concurrency-safe, generic LRU (Least Recently Used)
// cache backed by a doubly-linked list and a hash map.
//
// The cache supports O(1) get, put, and delete operations. All methods are
// safe for concurrent use. An optional eviction callback can be registered
// via [WithOnEvict]; the callback is invoked outside the lock so it may
// safely call back into the cache.
package lru

import (
	"container/list"
	"sync"
)

// entry is the internal key-value pair stored in the linked list.
type entry[K comparable, V any] struct {
	key   K
	value V
}

// Cache is a concurrency-safe LRU cache with generic key and value types.
//
// It uses a doubly-linked list to track access order and a hash map for
// O(1) lookups. A [sync.RWMutex] guards all internal state.
type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	capacity int
	ll       *list.List
	items    map[K]*list.Element
	onEvict  func(key K, value V)
}

// Option configures a [Cache] at construction time.
type Option[K comparable, V any] func(*Cache[K, V])

// WithOnEvict registers a callback that fires when an entry is evicted.
// The callback runs outside the cache lock, so calling cache methods from
// within the callback is safe. Eviction ordering is only guaranteed within
// a single [Cache.Put] call.
func WithOnEvict[K comparable, V any](fn func(key K, value V)) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.onEvict = fn
	}
}

// New creates a new LRU cache that holds at most capacity entries.
// It panics if capacity is not positive.
func New[K comparable, V any](capacity int, opts ...Option[K, V]) *Cache[K, V] {
	if capacity <= 0 {
		panic("lru: capacity must be positive")
	}
	c := &Cache[K, V]{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[K]*list.Element, capacity),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Get looks up key and, on a hit, promotes the entry to the front (most
// recently used) and returns (value, true). On a miss it returns the zero
// value and false.
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, hit := c.items[key]; hit {
		c.ll.MoveToFront(elem)
		return elem.Value.(*entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

// Peek returns the value for key without updating access order.
func (c *Cache[K, V]) Peek(key K) (value V, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if elem, hit := c.items[key]; hit {
		return elem.Value.(*entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

// Put inserts or updates the key-value pair.
//
// If the key already exists its value is updated and the entry is promoted
// to the front. If the cache is at capacity the least recently used entry
// is evicted first. The return value reports whether an eviction occurred.
func (c *Cache[K, V]) Put(key K, value V) (evicted bool) {
	var evictedEntry *entry[K, V]

	c.mu.Lock()
	if elem, hit := c.items[key]; hit {
		c.ll.MoveToFront(elem)
		elem.Value.(*entry[K, V]).value = value
		c.mu.Unlock()
		return false
	}

	if c.ll.Len() >= c.capacity {
		evictedEntry = c.removeLRU()
		evicted = true
	}

	ent := &entry[K, V]{key: key, value: value}
	elem := c.ll.PushFront(ent)
	c.items[key] = elem
	c.mu.Unlock()

	if evictedEntry != nil && c.onEvict != nil {
		c.onEvict(evictedEntry.key, evictedEntry.value)
	}
	return evicted
}

// Delete removes the entry for key, returning true if the key was present.
func (c *Cache[K, V]) Delete(key K) (ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, hit := c.items[key]; hit {
		c.removeElement(elem)
		return true
	}
	return false
}

// Contains reports whether key is in the cache without updating access order.
func (c *Cache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

// Len returns the current number of entries in the cache.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ll.Len()
}

// Cap returns the maximum capacity of the cache.
func (c *Cache[K, V]) Cap() int {
	// Capacity is immutable after construction; no lock needed.
	return c.capacity
}

// Keys returns all keys ordered from most recently used to least recently used.
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, c.ll.Len())
	for e := c.ll.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(*entry[K, V]).key)
	}
	return keys
}

// Purge removes all entries from the cache.
func (c *Cache[K, V]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ll.Init()
	c.items = make(map[K]*list.Element, c.capacity)
}

// Resize changes the cache capacity. If the new capacity is smaller than the
// current size, the excess least-recently-used entries are evicted. It returns
// the number of evicted entries.
func (c *Cache[K, V]) Resize(newCapacity int) int {
	if newCapacity <= 0 {
		panic("lru: capacity must be positive")
	}

	var evictedEntries []*entry[K, V]

	c.mu.Lock()
	c.capacity = newCapacity
	for c.ll.Len() > c.capacity {
		evictedEntries = append(evictedEntries, c.removeLRU())
	}
	c.mu.Unlock()

	if c.onEvict != nil {
		for _, ent := range evictedEntries {
			c.onEvict(ent.key, ent.value)
		}
	}
	return len(evictedEntries)
}

// removeLRU removes the back (least recently used) element and returns its entry.
func (c *Cache[K, V]) removeLRU() *entry[K, V] {
	back := c.ll.Back()
	if back == nil {
		return nil
	}
	c.removeElement(back)
	return back.Value.(*entry[K, V])
}

// removeElement removes an element from both the linked list and the map.
func (c *Cache[K, V]) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry[K, V])
	delete(c.items, kv.key)
}
