// Package dag provides a directed acyclic graph (DAG) execution engine.
//
// A DAG defines a task graph where each node is a function that depends on
// the results of upstream nodes. The engine handles parallel scheduling,
// dependency resolution, and error propagation automatically.
//
// Advanced features include nested sub-graphs, node-function interceptors
// (middleware), pluggable executors, and Mermaid diagram generation.
//
// Basic usage:
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

// Sentinel errors returned by DAG construction and execution methods.
var (
	ErrDAGNodeExists = errors.New("DAG node already exists")
	ErrDAGFrozen     = errors.New("DAG is frozen")
	ErrDAGNotFrozen  = errors.New("DAG is not frozen")
	ErrDAGIncomplete = errors.New("DAG is incomplete")
	ErrDAGCyclic     = errors.New("DAG has cycle")
	ErrNodeSkipped   = errors.New("DAG node is skipped")
)

// NodeID uniquely identifies a node within a DAG.
type NodeID string

// NodeFunc is the signature of a node's execution function.
// It receives a context and a map of dependency results keyed by [NodeID],
// and returns this node's result or an error.
type NodeFunc func(ctx context.Context, deps map[NodeID]any) (any, error)

// NodeFuncInterceptor wraps a [NodeFunc] to add cross-cutting behavior
// such as logging, metrics, or error handling. Multiple interceptors are
// applied in reverse registration order, forming a middleware chain.
type NodeFuncInterceptor func(next NodeFunc) NodeFunc

// Node is the interface satisfied by every node variant in the DAG.
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

// EntryNode is the root node of a DAG that receives the initial input value.
type EntryNode struct {
	baseNode
}

// SimpleNode is a leaf or intermediate node backed by a [NodeFunc].
type SimpleNode struct {
	baseNode
	run NodeFunc
}

// SubDAGNode embeds a frozen child [DAG] as a single node, optionally
// transforming inputs and outputs through mapping functions.
type SubDAGNode struct {
	baseNode
	subDag        *DAG
	inputMapping  func(map[NodeID]any) any
	outputMapping func(map[NodeID]any) any
}

// DAG is the core directed acyclic graph definition. Nodes and edges are
// added before calling [DAG.Freeze], after which the DAG becomes immutable
// and can be instantiated for execution.
type DAG struct {
	entry  NodeID
	nodes  map[NodeID]Node
	frozen bool
}

// NewDAG creates a new DAG with the given entry node ID. The entry node is
// automatically registered and receives the input value at instantiation.
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

// AddNode registers a [SimpleNode] backed by fn with the given dependencies.
// It returns [ErrDAGFrozen] if the DAG has been frozen, or [ErrDAGNodeExists]
// if a node with the same id already exists.
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

// AddSubGraph registers a [SubDAGNode] that embeds a child DAG.
// inputMapping transforms the parent dependency map into the child's entry
// value; outputMapping transforms the child's result map into the parent
// node's output. Either mapping may be nil for pass-through behavior.
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

// Freeze validates the DAG for completeness (no dangling deps) and acyclicity
// (topological sort), then marks it as immutable. Sub-graphs are frozen
// recursively. It must be called before [DAG.Instantiate].
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

// checkComplete verifies that every declared dependency references an
// existing node.
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

// checkCycle uses Kahn's algorithm (topological sort) to detect cycles.
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

// ToMermaid generates a Mermaid flowchart string representing the DAG.
// It returns an empty string if the DAG has not been frozen.
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

// instantiateOptions holds the resolved configuration for [DAG.Instantiate].
type instantiateOptions struct {
	executor     future.Executor
	interceptors []NodeFuncInterceptor
	nodeResults  map[NodeID]any
}

// InstantiateOption configures a [DAGInstance] created by [DAG.Instantiate].
type InstantiateOption func(*instantiateOptions)

// WithExecutor sets the task executor for the instance.
// The default is [executors.GoExecutor] which runs each node in a new goroutine.
func WithExecutor(executor future.Executor) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.executor = executor
	}
}

// WithNodeFuncInterceptor appends an interceptor to the chain.
// Interceptors execute in reverse registration order (last registered runs
// outermost), similar to HTTP middleware.
func WithNodeFuncInterceptor(interceptor NodeFuncInterceptor) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.interceptors = append(opts.interceptors, interceptor)
	}
}

// WithNodeResults pre-populates node results, causing those nodes to be
// skipped during execution.
func WithNodeResults(results map[NodeID]any) InstantiateOption {
	return func(opts *instantiateOptions) {
		opts.nodeResults = results
	}
}

// Instantiate creates an executable [DAGInstance] from a frozen DAG.
// input is passed to the entry node. The returned instance is independent
// and may be run concurrently with other instances of the same DAG.
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

// createNodeRunFunc builds the [NodeFunc] for a given node spec. Nodes with
// pre-populated results return them immediately; sub-DAG nodes instantiate
// and run their child DAG.
func (d *DAG) createNodeRunFunc(spec Node, results map[NodeID]any, options []InstantiateOption, node *NodeInstance) NodeFunc {
	result, ok := results[spec.ID()]
	if ok {
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

// NodeInstance is the runtime representation of a single node within a
// [DAGInstance]. It tracks pending dependencies, execution timing, and
// the eventual result via a [future.Promise].
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

// DAGInstance is a ready-to-run snapshot of a frozen [DAG] with a specific
// input. Each instance maintains its own execution state and may be run
// independently.
type DAGInstance struct {
	spec  *DAG
	nodes map[NodeID]*NodeInstance

	executor future.Executor
}

// Run synchronously executes the DAG and returns all node results.
func (d *DAGInstance) Run(ctx context.Context) (map[NodeID]any, error) {
	return d.RunAsync(ctx).Get()
}

// RunAsync starts the DAG execution asynchronously and returns a [future.Future]
// that will resolve to the complete result map.
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

// runNode submits a node for asynchronous execution. Once the node completes,
// it decrements the pending count of all children and triggers any child
// whose dependencies are fully satisfied.
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
