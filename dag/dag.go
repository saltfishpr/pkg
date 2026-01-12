// Package dag 提供了一个有向无环图（DAG）的实现，支持节点的并发执行、子图嵌套和自定义拦截器。
//
// # 使用流程
//
//  1. 创建 DAG：使用 NewDAG() 创建一个新的 DAG，指定入口节点 ID
//  2. 添加节点：使用 AddNode() 或 AddSubGraph() 添加节点
//  3. 冻结 DAG：调用 Freeze() 进行验证（检查完整性和循环）
//  4. 创建实例：使用 Instantiate() 为 DAG 创建可执行实例
//  5. 运行实例：调用 Run() 或 RunAsync() 执行 DAG
//
// # 示例
//
//	dag := dag.NewDAG("entry")
//	dag.AddNode("node1", []dag.NodeID{"entry"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
//		return "result1", nil
//	})
//	dag.AddNode("node2", []dag.NodeID{"node1"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
//		return "result2", nil
//	})
//	if err := dag.Freeze(); err != nil {
//		log.Fatal(err)
//	}
//
//	instance, err := dag.Instantiate("entry_value")
//	if err != nil {
//		log.Fatal(err)
//	}
//	results, err := instance.Run(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # 高级特性
//
//   - 子图：通过 AddSubGraph() 支持 DAG 的嵌套和模块化。
//   - 并发执行：所有无依赖关系的节点会自动并发执行。
//   - Mermaid 图形化：调用 ToMermaid() 生成 Mermaid 格式的 DAG 图表。
//   - 拦截器：使用 WithNodeFuncInterceptor() 为节点执行添加日志、监控等功能。
package dag

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/saltfishpr/pkg/future"
	"github.com/saltfishpr/pkg/future/executors"
)

var (
	ErrDAGNodeExists = errors.New("DAG node already exists")
	ErrDAGFrozen     = errors.New("DAG is frozen")
	ErrDAGNotFrozen  = errors.New("DAG is not frozen")
	ErrDAGIncomplete = errors.New("DAG is incomplete")
	ErrDAGCyclic     = errors.New("DAG has cycle")
	// ErrNodeSkipped 表示节点执行被跳过。
	ErrNodeSkipped = errors.New("DAG node is skipped")
)

type NodeID string

type NodeFunc func(ctx context.Context, deps map[NodeID]any) (any, error)

type NodeFuncInterceptor func(next NodeFunc) NodeFunc

type Node interface {
	ID() NodeID
	Deps() []NodeID
}

type BaseNode struct {
	id   NodeID
	deps []NodeID
}

func (n *BaseNode) ID() NodeID { return n.id }

func (n *BaseNode) Deps() []NodeID { return n.deps }

type EntryNode struct {
	BaseNode
}

type SimpleNode struct {
	BaseNode
	run NodeFunc
}

type SubDAGNode struct {
	BaseNode
	subDag        *DAG
	inputMapping  func(map[NodeID]any) any
	outputMapping func(map[NodeID]any) any
}
type DAG struct {
	entry  NodeID
	nodes  map[NodeID]Node
	frozen bool
}

func NewDAG(entry NodeID) *DAG {
	dag := &DAG{
		nodes: make(map[NodeID]Node),
	}
	dag.entry = entry
	dag.nodes[entry] = &EntryNode{
		BaseNode: BaseNode{
			id: entry,
		},
	}
	return dag
}

func (d *DAG) AddNode(id NodeID, deps []NodeID, fn NodeFunc) error {
	if d.frozen {
		return ErrDAGFrozen
	}
	if _, exists := d.nodes[id]; exists {
		return ErrDAGNodeExists
	}
	d.nodes[id] = &SimpleNode{
		BaseNode: BaseNode{
			id:   id,
			deps: deps,
		},
		run: fn,
	}
	return nil
}

func (d *DAG) AddSubGraph(
	id NodeID, deps []NodeID, subDag *DAG,
	inputMapping func(map[NodeID]any) any,
	outputMapping func(map[NodeID]any) any,
) error {
	if d.frozen {
		return ErrDAGFrozen
	}
	if _, exists := d.nodes[id]; exists {
		return ErrDAGNodeExists
	}
	d.nodes[id] = &SubDAGNode{
		BaseNode: BaseNode{
			id:   id,
			deps: deps,
		},
		subDag:        subDag,
		inputMapping:  inputMapping,
		outputMapping: outputMapping,
	}
	return nil
}

func (d *DAG) Freeze() error {
	if d.frozen {
		return ErrDAGFrozen
	}
	if err := d.checkComplete(); err != nil {
		return err
	}
	if err := d.checkCycle(); err != nil {
		return err
	}
	d.frozen = true
	for _, node := range d.nodes {
		if subDagNode, ok := node.(*SubDAGNode); ok {
			if err := subDagNode.subDag.Freeze(); err != nil {
				return fmt.Errorf("freeze node %s failed: %w", subDagNode.ID(), err)
			}
		}
	}
	return nil
}

func (d *DAG) checkComplete() error {
	for id, node := range d.nodes {
		for _, dep := range node.Deps() {
			if _, ok := d.nodes[dep]; !ok {
				return fmt.Errorf("dependency %s of node %s is not present: %w", dep, id, ErrDAGIncomplete)
			}
		}
	}
	return nil
}

func (d *DAG) checkCycle() error {
	inDegree := make(map[NodeID]int)
	queue := make([]NodeID, 0)
	visited := 0

	for id, node := range d.nodes {
		inDegree[id] = len(node.Deps())
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	children := make(map[NodeID][]NodeID)
	for id, node := range d.nodes {
		for _, dep := range node.Deps() {
			children[dep] = append(children[dep], id)
		}
	}

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		visited++
		for _, v := range children[u] {
			inDegree[v]--
			if inDegree[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	if visited != len(d.nodes) {
		return ErrDAGCyclic
	}
	return nil
}

func (d *DAG) ToMermaid() string {
	if !d.frozen {
		return ""
	}

	var b strings.Builder
	b.WriteString("graph LR\n")
	d.toMermaid(&b, "", "\t")
	return b.String()
}

func (d *DAG) toMermaid(b *strings.Builder, prefix string, indent string) {
	ids := make([]string, 0, len(d.nodes))
	for id := range d.nodes {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, id := range ids {
		node := d.nodes[NodeID(id)]
		label := prefix + id

		switch n := node.(type) {
		case *EntryNode:
			_, _ = fmt.Fprintf(b, "%s%s[%q]\n", indent, label, label)
		case *SimpleNode:
			_, _ = fmt.Fprintf(b, "%s%s((%q))\n", indent, label, label)
		case *SubDAGNode:
			_, _ = fmt.Fprintf(b, "%ssubgraph %s [Subgraph %s]\n", indent, label, label)
			n.subDag.toMermaid(b, label+".", indent+"\t")
			_, _ = fmt.Fprintf(b, "%send\n", indent)
		}
	}

	for _, id := range ids {
		node := d.nodes[NodeID(id)]
		srcLabel := prefix + id
		for _, dep := range node.Deps() {
			dstLabel := prefix + string(dep)
			_, _ = fmt.Fprintf(b, "%s%s --> %s\n", indent, dstLabel, srcLabel)
		}
	}
}

type instantiateOptions struct {
	// executor 用于执行 DAG 实例中的节点
	executor future.Executor
	// interceptors 用来包装节点的执行函数，可以用于日志、监控等场景
	interceptors []NodeFuncInterceptor
	// nodeResults 预设的节点结果
	nodeResults map[NodeID]any
}

type InstantiateOption func(*instantiateOptions)

func WithExecutor(executor future.Executor) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.executor = executor
	}
}

func WithNodeFuncInterceptor(interceptor NodeFuncInterceptor) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.interceptors = append(opts.interceptors, interceptor)
	}
}

func WithNodeResults(results map[NodeID]any) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.nodeResults = results
	}
}

// Instantiate 创建 DAG 的一个实例。
func (d *DAG) Instantiate(input any, opts ...InstantiateOption) (*DAGInstance, error) {
	if !d.frozen {
		return nil, ErrDAGNotFrozen
	}

	options := &instantiateOptions{
		executor: executors.GoExecutor{},
	}
	for _, opt := range opts {
		opt(options)
	}

	results := make(map[NodeID]any)
	results[d.entry] = input
	for id, result := range options.nodeResults {
		results[id] = result
	}

	nodes := make(map[NodeID]*NodeInstance)
	children := make(map[NodeID][]NodeID)
	for id, spec := range d.nodes {
		spec := spec

		promise := future.NewPromise[any]()
		node := &NodeInstance{
			spec:    spec,
			pending: &atomic.Int32{},
			promise: promise,
			result:  promise.Future(),
		}
		node.pending.Store(int32(len(spec.Deps())))

		run := d.createNodeRunFunc(spec, results, opts, node)
		for i := len(options.interceptors) - 1; i >= 0; i-- {
			run = options.interceptors[i](run)
		}
		node.run = run

		nodes[id] = node
		for _, dep := range spec.Deps() {
			children[dep] = append(children[dep], id)
		}
	}

	for id, node := range nodes {
		node.children = children[id]
	}

	return &DAGInstance{
		spec:     d,
		nodes:    nodes,
		executor: options.executor,
	}, nil
}

func (d *DAG) createNodeRunFunc(spec Node, results map[NodeID]any, opts []InstantiateOption, node *NodeInstance) NodeFunc {
	result, ok := results[spec.ID()]
	if ok {
		// 节点已经有值，直接返回
		return func(_ context.Context, _ map[NodeID]any) (any, error) { return result, nil }
	}

	switch n := spec.(type) {
	case *EntryNode:
		return func(_ context.Context, _ map[NodeID]any) (any, error) { return results[n.ID()], nil }
	case *SimpleNode:
		return n.run
	case *SubDAGNode:
		return func(ctx context.Context, deps map[NodeID]any) (any, error) {
			var input any = deps
			if n.inputMapping != nil {
				input = n.inputMapping(deps)
			}
			instance, err := n.subDag.Instantiate(input, opts...)
			if err != nil {
				return nil, fmt.Errorf("instantiate sub DAG failed: %w", err)
			}
			results, err := instance.Run(ctx)
			if err != nil {
				return nil, fmt.Errorf("run sub DAG failed: %w", err)
			}
			node.subDagInstance = instance
			node.subDagResults = results
			var output any = results
			if n.outputMapping != nil {
				output = n.outputMapping(results)
			}
			return output, nil
		}
	default:
		panic("should not happen")
	}
}

type NodeInstance struct {
	spec Node

	children []NodeID
	pending  *atomic.Int32
	run      NodeFunc
	promise  *future.Promise[any]
	result   *future.Future[any]

	subDagInstance *DAGInstance
	subDagResults  map[NodeID]any

	startTime time.Time
	endTime   time.Time
}

type DAGInstance struct {
	spec  *DAG
	nodes map[NodeID]*NodeInstance

	executor future.Executor
}

func (d *DAGInstance) Run(ctx context.Context) (map[NodeID]any, error) {
	return d.RunAsync(ctx).Get()
}

func (d *DAGInstance) RunAsync(ctx context.Context) *future.Future[map[NodeID]any] {
	d.runNode(ctx, d.spec.entry)
	futures := make([]*future.Future[any], 0, len(d.nodes))
	for _, node := range d.nodes {
		futures = append(futures, node.result)
	}
	return future.WithContext(ctx,
		future.Then(future.AllOf(futures...),
			func(_ []any, _ error) (map[NodeID]any, error) {
				results := make(map[NodeID]any)
				for id, node := range d.nodes {
					val, err := node.result.Get()
					if err != nil {
						if errors.Is(err, ErrNodeSkipped) {
							continue
						}
						return nil, fmt.Errorf("node %s failed: %w", id, err)
					}
					results[id] = val
				}
				return results, nil
			},
		),
	)
}

func (d *DAGInstance) runNode(ctx context.Context, id NodeID) {
	node := d.nodes[id]
	node.startTime = time.Now()
	future.Submit(d.executor, func() (any, error) {
		deps := make(map[NodeID]any)
		for _, depid := range node.spec.Deps() {
			v, err := d.nodes[depid].result.Get()
			if err != nil {
				if errors.Is(err, ErrNodeSkipped) {
					return nil, ErrNodeSkipped
				}
				return nil, fmt.Errorf("dep %s failed: %w", depid, err)
			}
			deps[depid] = v
		}
		val, err := node.run(ctx, deps)
		node.endTime = time.Now()
		for _, childID := range node.children {
			if d.nodes[childID].pending.Add(-1) == 0 {
				d.runNode(ctx, childID)
			}
		}
		return val, err
	}).Subscribe(func(val any, err error) {
		node.promise.Set(val, err)
	})
}
