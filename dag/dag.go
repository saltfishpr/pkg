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
	ErrDAGNodeExists = errors.New("DAG node is already exists")
	ErrDAGFrozen     = errors.New("DAG is frozen")
	ErrDAGNotFrozen  = errors.New("DAG is not frozen")
	ErrDAGIncomplete = errors.New("DAG is incomplete")
	ErrDAGCyclic     = errors.New("DAG has cycle")
)

type NodeType int

const (
	NodeTypeEntry NodeType = iota
	NodeTypeSimple
	NodeTypeSubDAG
)

type NodeID string

type NodeFunc func(ctx context.Context, deps map[NodeID]any) (any, error)

type Node interface {
	Type() NodeType
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

func (n *EntryNode) Type() NodeType { return NodeTypeEntry }

type SimpleNode struct {
	BaseNode
	run NodeFunc
}

func (n *SimpleNode) Type() NodeType { return NodeTypeSimple }

type SubDAGNode struct {
	BaseNode
	subDag        *DAG
	inputMapping  func(map[NodeID]any) any
	outputMapping func(map[NodeID]any) any
}

func (n *SubDAGNode) Type() NodeType { return NodeTypeSubDAG }

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

type InstantiateOptions struct {
	executor future.Executor
}

type InstantiateOption func(*InstantiateOptions)

func WithExecutor(executor future.Executor) InstantiateOption {
	return func(opts *InstantiateOptions) {
		opts.executor = executor
	}
}

func (d *DAG) Instantiate(input any, opts ...InstantiateOption) (*DAGInstance, error) {
	if !d.frozen {
		return nil, ErrDAGNotFrozen
	}

	options := &InstantiateOptions{
		executor: executors.GoExecutor{},
	}
	for _, opt := range opts {
		opt(options)
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
		switch spec.Type() {
		case NodeTypeEntry:
			node.run = func(_ context.Context, _ map[NodeID]any) (any, error) { return input, nil }
		case NodeTypeSimple:
			n := spec.(*SimpleNode)
			node.run = n.run
		case NodeTypeSubDAG:
			n := spec.(*SubDAGNode)
			node.run = func(ctx context.Context, deps map[NodeID]any) (any, error) {
				var input any = deps
				if n.inputMapping != nil {
					input = n.inputMapping(deps)
				}
				instance, err := n.subDag.Instantiate(input)
				if err != nil {
					return nil, fmt.Errorf("instantiate sub DAG failed: %w", err)
				}
				results, err := instance.Run(ctx)
				if err != nil {
					return nil, fmt.Errorf("run sub DAG failed: %w", err)
				}
				node.subDagInstance = instance
				node.subDagResult = results
				var output any = results
				if n.outputMapping != nil {
					output = n.outputMapping(results)
				}
				return output, nil
			}
		}

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

type NodeInstance struct {
	spec Node

	children []NodeID
	pending  *atomic.Int32
	run      NodeFunc
	promise  *future.Promise[any]
	result   *future.Future[any]

	subDagInstance *DAGInstance
	subDagResult   map[NodeID]any

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
	return future.Then(
		future.AllOf(futures...),
		func(_ []any, err error) (map[NodeID]any, error) {
			if err != nil {
				return nil, err
			}
			results := make(map[NodeID]any)
			for id, node := range d.nodes {
				val, err := node.result.Get()
				if err != nil {
					return nil, fmt.Errorf("node %s failed: %w", id, err)
				}
				results[id] = val
			}
			return results, nil
		},
	)
}

func (d *DAGInstance) runNode(ctx context.Context, id NodeID) {
	node := d.nodes[id]
	node.startTime = time.Now()
	future.CtxSubmit(ctx, d.executor, func(ctx context.Context) (any, error) {
		deps := make(map[NodeID]any)
		for _, depid := range node.spec.Deps() {
			v, err := d.nodes[depid].result.Get()
			if err != nil {
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

func (d *DAGInstance) ToMermaid() string {
	var b strings.Builder
	b.WriteString("graph LR\n")
	d.toMermaid(&b, "", "\t")
	return b.String()
}

func (d *DAGInstance) toMermaid(b *strings.Builder, prefix string, indent string) {
	ids := make([]string, 0, len(d.nodes))
	for id := range d.nodes {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, id := range ids {
		node := d.nodes[NodeID(id)]
		label := prefix + id

		switch node.spec.Type() {
		case NodeTypeEntry:
			_, _ = fmt.Fprintf(b, "%s%s[%q]\n", indent, label, label)
		case NodeTypeSimple:
			_, _ = fmt.Fprintf(b, "%s%s((%q))\n", indent, label, label)
		case NodeTypeSubDAG:
			if node.subDagInstance != nil {
				_, _ = fmt.Fprintf(b, "%ssubgraph %s [Subgraph %s]\n", indent, label, label)
				node.subDagInstance.toMermaid(b, label+".", indent+"\t")
				_, _ = fmt.Fprintf(b, "%send\n", indent)
			}
		}
	}

	for _, id := range ids {
		node := d.nodes[NodeID(id)]
		srcLabel := prefix + id
		for _, dep := range node.spec.Deps() {
			depLabel := prefix + string(dep)
			_, _ = fmt.Fprintf(b, "%s%s --> %s\n", indent, depLabel, srcLabel)
		}
	}
}
