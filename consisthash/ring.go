// Package consisthash implements a generic, concurrency-safe consistent
// hashing ring with virtual nodes.
//
// Consistent hashing distributes keys across a ring of nodes so that adding
// or removing a node only remaps keys in its immediate neighborhood,
// minimizing data migration. Each physical node is replicated into multiple
// virtual nodes to improve distribution uniformity.
//
// Basic usage:
//
//	r := consisthash.NewRing(150, func(s string) string { return s })
//	r.Add("10.0.0.1", "10.0.0.2", "10.0.0.3")
//	node, ok := r.Get("my-cache-key")
package consisthash

import (
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
)

// HashFunc computes a 64-bit hash from the given bytes.
type HashFunc func(data []byte) uint64

// KeyFunc extracts a unique string identifier from a node of type T.
// The returned string is used as the hash input for virtual node placement.
type KeyFunc[T any] func(node T) string

// Ring is a consistent hashing ring that maps arbitrary string keys to
// physical nodes of type T via virtual nodes.
//
// All methods are safe for concurrent use.
type Ring[T any] struct {
	hashFunc HashFunc
	keyFunc  KeyFunc[T]
	replicas int // number of virtual nodes per physical node

	mu      sync.RWMutex
	keys    []uint64     // sorted virtual-node hashes for binary search
	hashMap map[uint64]T // virtual-node hash → physical node
}

// RingOption configures a [Ring] at construction time.
type RingOption[T any] func(*Ring[T])

// WithHashFunc overrides the hash function. The default is FNV-1a (64-bit).
func WithHashFunc[T any](hf HashFunc) RingOption[T] {
	return func(r *Ring[T]) {
		r.hashFunc = hf
	}
}

// NewRing creates an empty consistent hashing ring.
//
// replicas controls how many virtual nodes are created per physical node;
// values between 150 and 200 generally yield a good key distribution.
// keyFunc extracts a unique identifier from each node for hashing.
func NewRing[T any](replicas int, keyFunc KeyFunc[T], options ...RingOption[T]) *Ring[T] {
	r := &Ring[T]{
		hashFunc: func(data []byte) uint64 {
			f := fnv.New64a()
			f.Write(data)
			return f.Sum64()
		},
		keyFunc:  keyFunc,
		replicas: replicas,
		hashMap:  make(map[uint64]T),
	}
	for _, option := range options {
		option(r)
	}
	return r
}

// Add inserts one or more nodes into the ring. Each node produces replicas
// virtual nodes. Hash collisions with existing virtual nodes are silently
// skipped to avoid overwriting.
func (r *Ring[T]) Add(nodes ...T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, node := range nodes {
		nodeKey := r.keyFunc(node)

		for i := 0; i < r.replicas; i++ {
			virtualKey := r.generateVirtualKey(nodeKey, i)
			hash := r.hashFunc([]byte(virtualKey))

			if _, exists := r.hashMap[hash]; exists {
				continue
			}

			r.keys = append(r.keys, hash)
			r.hashMap[hash] = node
		}
	}

	sort.Slice(r.keys, func(i, j int) bool {
		return r.keys[i] < r.keys[j]
	})
}

// Remove deletes a node and all of its virtual nodes from the ring.
func (r *Ring[T]) Remove(node T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodeKey := r.keyFunc(node)

	hashesToRemove := make(map[uint64]struct{})
	for i := 0; i < r.replicas; i++ {
		virtualKey := r.generateVirtualKey(nodeKey, i)
		hash := r.hashFunc([]byte(virtualKey))

		hashesToRemove[hash] = struct{}{}
		delete(r.hashMap, hash)
	}

	newKeys := r.keys[:0]
	for _, k := range r.keys {
		if _, exists := hashesToRemove[k]; !exists {
			newKeys = append(newKeys, k)
		}
	}
	r.keys = newKeys
}

// Get finds the first node clockwise from the hash of key on the ring.
// It returns the node and true, or the zero value and false if the ring
// is empty.
func (r *Ring[T]) Get(key string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.keys) == 0 {
		var zero T
		return zero, false
	}

	hash := r.hashFunc([]byte(key))

	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= hash
	})

	// Wrap around to the beginning of the ring.
	if idx == len(r.keys) {
		idx = 0
	}

	targetHash := r.keys[idx]
	return r.hashMap[targetHash], true
}

// generateVirtualKey builds a deterministic string for the i-th virtual node
// of a physical node identified by key. The "@" separator ensures that
// different replica indices never collide.
func (r *Ring[T]) generateVirtualKey(key string, idx int) string {
	return strconv.Itoa(idx) + "@" + key
}
