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

// Ring 表示一致性哈希环，通过虚拟节点实现负载均衡。
// 添加和删除节点时只会影响相邻节点的数据迁移，最小化重组影响。
// 所有操作都是线程安全的。
type Ring[T any] struct {
	hashFunc HashFunc   // 自定义哈希算法，默认使用 FNV-1a
	keyFunc  KeyFunc[T] // 节点到唯一标识的转换函数
	replicas int        // 每个物理节点对应的虚拟节点数，增加可提高数据分布均匀性

	mu      sync.RWMutex
	keys    []uint64     // 排序后的虚拟节点哈希环，用于二分查找
	hashMap map[uint64]T // 哈希值到物理节点的映射
}

// RingOption 配置 Ring 的函数选项。
type RingOption[T any] func(*Ring[T])

// WithHashFunc 设置自定义哈希函数。默认使用 FNV-1a 算法。
func WithHashFunc[T any](hf HashFunc) RingOption[T] {
	return func(r *Ring[T]) {
		r.hashFunc = hf
	}
}

// NewRing 创建一个空的一致性哈希环。
// replicas 是每个物理节点生成的虚拟节点数量，通常设置为 150-200 可获得较好的分布均匀性。
// keyFunc 用于从节点类型 T 中提取唯一标识符，作为哈希输入。
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

// Add 将节点添加到哈希环。每个节点会生成 replicas 个虚拟节点以分散负载。
// 发生哈希冲突时跳过该虚拟节点，确保数据完整性。
func (r *Ring[T]) Add(nodes ...T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, node := range nodes {
		nodeKey := r.keyFunc(node)

		for i := 0; i < r.replicas; i++ {
			virtualKey := r.generateVirtualKey(nodeKey, i)
			hash := r.hashFunc([]byte(virtualKey))

			// 哈希冲突时跳过，避免覆盖已有映射
			if _, exists := r.hashMap[hash]; exists {
				continue
			}

			r.keys = append(r.keys, hash)
			r.hashMap[hash] = node
		}
	}

	// 重新排序以保持二分查找正确性
	sort.Slice(r.keys, func(i, j int) bool {
		return r.keys[i] < r.keys[j]
	})
}

// Remove 从哈希环中移除节点及其所有虚拟节点。
func (r *Ring[T]) Remove(node T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodeKey := r.keyFunc(node)

	// 先收集所有待删除的哈希值，避免在遍历 keys 时修改 map
	hashesToRemove := make(map[uint64]struct{})
	for i := 0; i < r.replicas; i++ {
		virtualKey := r.generateVirtualKey(nodeKey, i)
		hash := r.hashFunc([]byte(virtualKey))

		hashesToRemove[hash] = struct{}{}
		delete(r.hashMap, hash)
	}

	// 过滤掉已删除的哈希值
	newKeys := r.keys[:0]
	for _, k := range r.keys {
		if _, exists := hashesToRemove[k]; !exists {
			newKeys = append(newKeys, k)
		}
	}
	r.keys = newKeys
}

// Get 根据给定 key 查找哈希环上顺时针方向的第一个节点。
// 返回节点和 true；如果环为空，返回零值和 false。
func (r *Ring[T]) Get(key string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.keys) == 0 {
		var zero T
		return zero, false
	}

	hash := r.hashFunc([]byte(key))

	// 二分查找顺时针方向第一个虚拟节点
	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= hash
	})

	// 回绕到环的起点
	if idx == len(r.keys) {
		idx = 0
	}

	targetHash := r.keys[idx]
	return r.hashMap[targetHash], true
}

// generateVirtualKey 通过拼接索引和 key 生成虚拟节点的唯一标识。
// 使用 @ 分隔符确保不同索引不会产生相同的哈希结果。
func (r *Ring[T]) generateVirtualKey(key string, idx int) string {
	return strconv.Itoa(idx) + "@" + key
}
