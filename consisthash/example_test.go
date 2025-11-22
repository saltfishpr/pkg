package consisthash_test

import (
	"fmt"

	"github.com/saltfishpr/pkg/consisthash"
)

// ExampleRing_basic 展示了最基础的字符串节点用法
func ExampleRing_basic() {
	// 1. 定义 KeyFunc：直接返回字符串本身
	keyFunc := func(node string) string {
		return node
	}

	// 2. 创建哈希环，每个节点 3 个虚拟节点
	r := consisthash.NewRing(3, keyFunc)

	// 3. 添加节点
	r.Add("192.168.1.101", "192.168.1.102", "192.168.1.103")

	// 4. 获取 Key 归属的节点
	keys := []string{"user_123", "order_456", "session_789"}
	for _, k := range keys {
		node, ok := r.Get(k)
		if ok {
			fmt.Printf("Key '%s' is mapped to node: %s\n", k, node)
		}
	}

	// Output:
	// Key 'user_123' is mapped to node: 192.168.1.101
	// Key 'order_456' is mapped to node: 192.168.1.101
	// Key 'session_789' is mapped to node: 192.168.1.102
}

// Server 代表一个后端服务器节点
type Server struct {
	IP   string
	Port int
	Zone string
}

// ExampleRing_struct 展示了如何使用结构体作为节点（泛型优势）
func ExampleRing_struct() {
	// 1. 定义 KeyFunc：使用 IP:Port 作为唯一标识
	keyFunc := func(s *Server) string {
		return fmt.Sprintf("%s:%d", s.IP, s.Port)
	}

	// 2. 创建哈希环
	r := consisthash.NewRing(10, keyFunc)

	// 3. 准备节点数据
	srv1 := &Server{IP: "10.0.0.1", Port: 8080, Zone: "US-West"}
	srv2 := &Server{IP: "10.0.0.2", Port: 8080, Zone: "US-East"}

	r.Add(srv1, srv2)

	// 4. 查找节点
	target, ok := r.Get("request_id_abc")
	if ok {
		// 可以直接访问结构体字段
		fmt.Printf("Request handled by %s (Zone: %s)\n", target.IP, target.Zone)
	}

	// Output:
	// Request handled by 10.0.0.2 (Zone: US-East)
}

// ExampleRing_addRemove 展示节点动态添加和删除
func ExampleRing_addRemove() {
	keyFunc := func(node string) string { return node }
	r := consisthash.NewRing(20, keyFunc)

	// 初始节点
	r.Add("Node_A", "Node_B")

	key := "cache_key_example"
	node1, _ := r.Get(key)
	fmt.Printf("Initial mapping: %s -> %s\n", key, node1)

	// 移除命中的节点，验证重哈希
	fmt.Printf("Removing node: %s\n", node1)
	r.Remove(node1)

	node2, _ := r.Get(key)
	fmt.Printf("After remove:    %s -> %s\n", key, node2)

	// Output:
	// Initial mapping: cache_key_example -> Node_B
	// Removing node: Node_B
	// After remove:    cache_key_example -> Node_A
}
