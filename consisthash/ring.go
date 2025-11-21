package consisthash

import (
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
)

// HashFunc 定义哈希函数的类型
type HashFunc func(data []byte) uint64

// KeyFunc 定义将节点转换为字符串的函数，用于生成哈希 Key
type KeyFunc[T any] func(node T) string

// Ring 是一个线程安全的一致性哈希环结构
type Ring[T any] struct {
	hashFunc HashFunc   // 哈希算法
	keyFunc  KeyFunc[T] // 获取节点唯一标识的函数
	replicas int        // 每个真实节点的虚拟节点数量

	mu      sync.RWMutex
	keys    []uint64     // 已排序的虚拟节点哈希值切片
	hashMap map[uint64]T // 虚拟节点哈希值 -> 真实节点数据
}

type RingOption[T any] func(*Ring[T])

func WithHashFunc[T any](hf HashFunc) RingOption[T] {
	return func(r *Ring[T]) {
		r.hashFunc = hf
	}
}

// NewRing 创建一个新的哈希环
func NewRing[T any](replicas int, keyFunc KeyFunc[T], opts ...RingOption[T]) *Ring[T] {
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
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Add 向环中添加节点
func (r *Ring[T]) Add(nodes ...T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, node := range nodes {
		nodeKey := r.keyFunc(node)

		// 为每个节点创建 replicas 个虚拟节点
		for i := 0; i < r.replicas; i++ {
			virtualKey := r.generateVirtualKey(nodeKey, i)
			hash := r.hashFunc([]byte(virtualKey))

			// 处理哈希冲突
			if _, exists := r.hashMap[hash]; exists {
				continue // 跳过冲突的虚拟节点
			}

			// 将哈希值加入切片
			r.keys = append(r.keys, hash)
			// 建立哈希值到真实节点的映射
			r.hashMap[hash] = node
		}
	}

	// 每次添加后，对哈希环上的点进行排序，以便进行二分查找
	sort.Slice(r.keys, func(i, j int) bool {
		return r.keys[i] < r.keys[j]
	})
}

// Remove 从环中移除指定的节点
func (r *Ring[T]) Remove(node T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 1. 获取该节点的唯一标识
	nodeKey := r.keyFunc(node)

	// 2. 找出该节点所有的虚拟节点哈希值，并在 Map 中删除
	// 使用一个临时的 set 来存储需要移除的 hash 值，以便后续过滤切片
	hashesToRemove := make(map[uint64]struct{})
	for i := 0; i < r.replicas; i++ {
		virtualKey := r.generateVirtualKey(nodeKey, i)
		hash := r.hashFunc([]byte(virtualKey))

		hashesToRemove[hash] = struct{}{}
		delete(r.hashMap, hash)
	}

	// 3. 从排序切片 r.keys 中移除这些哈希值
	newKeys := r.keys[:0]
	for _, k := range r.keys {
		// 如果 k 不在待删除列表中，则保留
		if _, exists := hashesToRemove[k]; !exists {
			newKeys = append(newKeys, k)
		}
	}
	r.keys = newKeys
}

// Get 根据输入的数据项 Key，顺时针查找最近的节点
func (r *Ring[T]) Get(key string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.keys) == 0 {
		var zero T
		return zero, false // 环为空
	}

	// 1. 计算输入 Key 的哈希值
	hash := r.hashFunc([]byte(key))

	// 2. 二分查找：找到第一个 Hash 值 >= keyHash 的虚拟节点索引
	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= hash
	})

	// 3. 如果 idx == len(keys)，需要回绕到环的起点
	if idx == len(r.keys) {
		idx = 0
	}

	targetHash := r.keys[idx]
	return r.hashMap[targetHash], true
}

// generateVirtualKey 生成虚拟节点 Key
func (r *Ring[T]) generateVirtualKey(key string, idx int) string {
	return strconv.Itoa(idx) + "@" + key
}
