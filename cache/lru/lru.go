// Package lru 提供了一个并发安全的泛型 LRU 缓存实现。
package lru

import (
	"container/list"
	"sync"
)

// entry 是存储在双向链表中的内部键值对。
type entry[K comparable, V any] struct {
	key   K
	value V
}

// Cache 是一个支持泛型键值类型的并发安全 LRU 缓存。
//
// 它使用双向链表来维护访问顺序，使用哈希映射实现 O(1) 查找。
// 读写互斥锁保证多个 goroutine 的安全并发访问。
type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	capacity int
	ll       *list.List
	items    map[K]*list.Element
	onEvict  func(key K, value V)
}

// Option 用于在构建时配置 Cache。
type Option[K comparable, V any] func(*Cache[K, V])

// WithOnEvict 注册一个回调函数，当条目被驱逐时调用。
// 回调在未持有锁的情况下调用，因此从回调中安全地调用缓存是可行的。
// 但是，驱逐顺序保证仅适用于单个 Put 调用内的通知顺序。
func WithOnEvict[K comparable, V any](fn func(key K, value V)) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.onEvict = fn
	}
}

// New 创建一个新的 LRU Cache，具有给定的最大容量。
// capacity 必须大于 0，否则会 panic。
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

// Get 在缓存中查找键。如果找到，则将条目移到最前面（最近使用）
// 并返回 (value, true)。否则返回零值和 false。
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

// Peek 返回键对应的值，但不更新访问顺序。
// 这对于无副作用地检查缓存很有用。
func (c *Cache[K, V]) Peek(key K) (value V, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if elem, hit := c.items[key]; hit {
		return elem.Value.(*entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

// Put 插入或更新键值对。
//   - 如果键已存在，则更新值并将条目提升到最前面。
//   - 如果缓存已满，则首先驱逐最近最少使用的条目。
//
// 如果驱逐了现有条目以腾出空间，则返回 true。
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

// Delete 删除键对应的条目（如果存在）。如果找到键则返回 true。
func (c *Cache[K, V]) Delete(key K) (ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, hit := c.items[key]; hit {
		c.removeElement(elem)
		return true
	}
	return false
}

// Contains 报告缓存是否包含给定的键，但不更新访问顺序。
func (c *Cache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

// Len 返回缓存中当前条目的数量。
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ll.Len()
}

// Cap 返回缓存的最大容量。
func (c *Cache[K, V]) Cap() int {
	// Capacity 在构建后是不可变的；不需要锁。
	return c.capacity
}

// Keys 返回缓存中所有键的切片，按从最近使用到最近最少使用的顺序排列。
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, c.ll.Len())
	for e := c.ll.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(*entry[K, V]).key)
	}
	return keys
}

// Purge 从缓存中删除所有条目。
func (c *Cache[K, V]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ll.Init()
	c.items = make(map[K]*list.Element, c.capacity)
}

// Resize 更改缓存的容量。如果新容量小于当前条目数，
// 则驱逐最近最少使用的条目。返回被驱逐的条目数。
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

// removeLRU 移除最近最少使用的元素并返回其条目。
func (c *Cache[K, V]) removeLRU() *entry[K, V] {
	back := c.ll.Back()
	if back == nil {
		return nil
	}
	c.removeElement(back)
	return back.Value.(*entry[K, V])
}

// removeElement 从链表和映射中移除一个元素。
func (c *Cache[K, V]) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry[K, V])
	delete(c.items, kv.key)
}
