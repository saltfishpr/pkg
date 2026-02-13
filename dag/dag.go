// Package dag 提供了一个有向无环图（DAG）执行引擎。
//
// DAG 允许你定义由多个节点及其依赖关系组成的任务图，引擎会自动处理并行执行、
// 依赖管理和错误传播。支持嵌套子图、节点拦截器、自定义执行器等高级特性。
//
// 基本用法：
//
//	d := dag.NewDAG("entry")
//	d.AddNode("double", []dag.NodeID{"entry"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
//	    return deps["entry"].(int) * 2, nil
//	})
//	d.Freeze()
//	instance, _ := d.Instantiate(5)
//	results, _ := instance.Run(context.Background())
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
	ErrNodeSkipped   = errors.New("DAG node is skipped")
)

// NodeID 表示 DAG 中节点的唯一标识符。
type NodeID string

// NodeFunc 是节点执行函数的类型定义。
// 接收 context 和依赖节点结果映射，返回当前节点的结果或错误。
type NodeFunc func(ctx context.Context, deps map[NodeID]any) (any, error)

// NodeFuncInterceptor 用于拦截和包装节点执行函数。
// 可以用于日志记录、性能监控、错误处理等横切关注点。
// 拦截器按添加顺序的逆序执行（类似中间件链）。
type NodeFuncInterceptor func(next NodeFunc) NodeFunc

// Node 表示 DAG 中的一个节点。
type Node interface {
	ID() NodeID
	Deps() []NodeID
}

type baseNode struct {
	id   NodeID
	deps []NodeID
}

func (n *baseNode) ID() NodeID { return n.id }

func (n *baseNode) Deps() []NodeID { return n.deps }

type EntryNode struct {
	baseNode
}

type SimpleNode struct {
	baseNode
	run NodeFunc
}

type SubDAGNode struct {
	baseNode
	subDag        *DAG
	inputMapping  func(map[NodeID]any) any
	outputMapping func(map[NodeID]any) any
}

// DAG 是有向无环图的核心结构，用于定义任务及其依赖关系。
type DAG struct {
	entry  NodeID
	nodes  map[NodeID]Node
	frozen bool
}

// NewDAG 创建一个新的 DAG，并指定入口节点 ID。
func NewDAG(entry NodeID) *DAG {
	dag := &DAG{
		nodes: make(map[NodeID]Node),
	}
	dag.entry = entry
	dag.nodes[entry] = &EntryNode{
		baseNode: baseNode{
			id: entry,
		},
	}
	return dag
}

// AddNode 向 DAG 中添加一个简单节点。
func (d *DAG) AddNode(id NodeID, deps []NodeID, fn NodeFunc) error {
	if d.frozen {
		return ErrDAGFrozen
	}
	if _, exists := d.nodes[id]; exists {
		return ErrDAGNodeExists
	}
	d.nodes[id] = &SimpleNode{
		baseNode: baseNode{
			id:   id,
			deps: deps,
		},
		run: fn,
	}
	return nil
}

// AddSubGraph 向 DAG 中添加一个子图节点。
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
		baseNode: baseNode{
			id:   id,
			deps: deps,
		},
		subDag:        subDag,
		inputMapping:  inputMapping,
		outputMapping: outputMapping,
	}
	return nil
}

// Freeze 冻结 DAG，验证其完整性和无环性。
// 冻结后不能再添加节点或子图。
// 必须在 Instantiate 之前调用。
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
	for _, node := range d.nodes {
		if subDagNode, ok := node.(*SubDAGNode); ok {
			if err := subDagNode.subDag.Freeze(); err != nil {
				return fmt.Errorf("freeze node %s failed: %w", subDagNode.ID(), err)
			}
		}
	}
	d.frozen = true
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

// ToMermaid 生成 DAG 的 Mermaid 流程图表示。
// 仅在 DAG 冻结后可用。
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
	executor     future.Executor
	interceptors []NodeFuncInterceptor
	nodeResults  map[NodeID]any
}

// InstantiateOption 是用于配置 DAG 实例的选项函数类型。
type InstantiateOption func(*instantiateOptions)

// WithExecutor 设置 DAG 实例使用的执行器。
// 默认使用 GoExecutor（基于 goroutine 的并发执行）。
func WithExecutor(executor future.Executor) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.executor = executor
	}
}

// WithNodeFuncInterceptor 添加节点函数拦截器。
// 可添加多个拦截器，按添加顺序的逆序执行。
func WithNodeFuncInterceptor(interceptor NodeFuncInterceptor) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.interceptors = append(opts.interceptors, interceptor)
	}
}

// WithNodeResults 预设节点结果，用于跳过特定节点的执行。
func WithNodeResults(results map[NodeID]any) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.nodeResults = results
	}
}

// Instantiate 创建 DAG 的可执行实例。
// 返回的 DAGInstance 可多次执行，每次执行都是独立的。
func (d *DAG) Instantiate(input any, options ...InstantiateOption) (*DAGInstance, error) {
	if !d.frozen {
		return nil, ErrDAGNotFrozen
	}

	opts := instantiateOptions{
		executor: executors.GoExecutor{},
	}
	for _, option := range options {
		option(&opts)
	}

	results := make(map[NodeID]any)
	results[d.entry] = input
	for id, result := range opts.nodeResults {
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

		run := d.createNodeRunFunc(spec, results, options, node)
		for i := len(opts.interceptors) - 1; i >= 0; i-- {
			run = opts.interceptors[i](run)
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
		executor: opts.executor,
	}, nil
}

func (d *DAG) createNodeRunFunc(spec Node, results map[NodeID]any, options []InstantiateOption, node *NodeInstance) NodeFunc {
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
			instance, err := n.subDag.Instantiate(input, options...)
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

// DAGInstance 是 DAG 的可执行实例。
// 每个实例维护独立的执行状态，可多次运行。
type DAGInstance struct {
	spec  *DAG
	nodes map[NodeID]*NodeInstance

	executor future.Executor
}

// Run 同步执行 DAG 实例，返回所有节点的执行结果。
func (d *DAGInstance) Run(ctx context.Context) (map[NodeID]any, error) {
	return d.RunAsync(ctx).Get()
}

// RunAsync 异步执行 DAG 实例，返回一个 Future。
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
		deps := make(map[NodeID]any, len(node.spec.Deps()))
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
