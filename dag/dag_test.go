package dag

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestNewDAG(t *testing.T) {
	d := NewDAG("entry")

	if d.entry != "entry" {
		t.Errorf("expected entry to be 'entry', got %s", d.entry)
	}

	if len(d.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(d.nodes))
	}

	if _, exists := d.nodes["entry"]; !exists {
		t.Error("expected entry node to exist")
	}

	if d.frozen {
		t.Error("expected DAG to not be frozen")
	}
}

func TestDAG_AddNode(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(d.nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(d.nodes))
	}

	node, exists := d.nodes["node1"]
	if !exists {
		t.Error("expected node1 to exist")
	}

	if node.ID() != "node1" {
		t.Errorf("expected node ID to be 'node1', got %s", node.ID())
	}

	if node.Type() != NodeTypeSimple {
		t.Errorf("expected node type to be NodeTypeSimple, got %v", node.Type())
	}
}

func TestDAG_AddNode_Duplicate(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("node1", []NodeID{"entry"}, fn)
	if !errors.Is(err, ErrDAGNodeExists) {
		t.Errorf("expected ErrDAGNodeExists, got %v", err)
	}
}

func TestDAG_AddNode_WhenFrozen(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	err := d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("node1", []NodeID{"entry"}, fn)
	if !errors.Is(err, ErrDAGFrozen) {
		t.Errorf("expected ErrDAGFrozen, got %v", err)
	}
}

func TestDAG_AddSubGraph(t *testing.T) {
	d := NewDAG("entry")
	sub := NewDAG("sub_entry")

	err := d.AddSubGraph(
		"subgraph1",
		[]NodeID{"entry"},
		sub,
		func(deps map[NodeID]any) any { return deps["entry"] },
		func(result map[NodeID]any) any { return result },
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(d.nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(d.nodes))
	}

	node, exists := d.nodes["subgraph1"]
	if !exists {
		t.Error("expected subgraph1 to exist")
	}

	if node.Type() != NodeTypeSubDAG {
		t.Errorf("expected node type to be NodeTypeSubDAG, got %v", node.Type())
	}
}

func TestDAG_AddSubGraph_Duplicate(t *testing.T) {
	d := NewDAG("entry")
	sub := NewDAG("sub_entry")

	err := d.AddSubGraph(
		"subgraph1",
		[]NodeID{"entry"},
		sub,
		nil,
		nil,
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddSubGraph(
		"subgraph1",
		[]NodeID{"entry"},
		sub,
		nil,
		nil,
	)
	if !errors.Is(err, ErrDAGNodeExists) {
		t.Errorf("expected ErrDAGNodeExists, got %v", err)
	}
}

func TestDAG_Freeze(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !d.frozen {
		t.Error("expected DAG to be frozen")
	}
}

func TestDAG_Freeze_AlreadyFrozen(t *testing.T) {
	d := NewDAG("entry")

	err := d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if !errors.Is(err, ErrDAGFrozen) {
		t.Errorf("expected ErrDAGFrozen, got %v", err)
	}
}

func TestDAG_Freeze_Incomplete(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	// Add a node with a missing dependency
	err := d.AddNode("node1", []NodeID{"missing"}, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if !errors.Is(err, ErrDAGIncomplete) {
		t.Errorf("expected ErrDAGIncomplete, got %v", err)
	}
}

func TestDAG_Freeze_Cyclic(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	// Create a cycle: node1 -> node2 -> node3 -> node1
	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("node2", []NodeID{"node1"}, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("node3", []NodeID{"node2"}, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Create the cycle by making node1 depend on node3
	d.nodes["node1"].(*SimpleNode).deps = []NodeID{"node3"}

	err = d.Freeze()
	if !errors.Is(err, ErrDAGCyclic) {
		t.Errorf("expected ErrDAGCyclic, got %v", err)
	}
}

func TestDAG_Instantiate(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["entry"].(int) + 1, nil
	}

	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := d.Instantiate(10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if inst == nil {
		t.Error("expected instance to not be nil")
	}

	if inst.spec != d {
		t.Error("expected instance spec to match DAG")
	}

	if len(inst.nodes) != 2 {
		t.Errorf("expected 2 node instances, got %d", len(inst.nodes))
	}
}

func TestDAG_Instantiate_NotFrozen(t *testing.T) {
	d := NewDAG("entry")

	_, err := d.Instantiate(10)
	if !errors.Is(err, ErrDAGNotFrozen) {
		t.Errorf("expected ErrDAGNotFrozen, got %v", err)
	}
}

func TestDAGInstance_Run_Simple(t *testing.T) {
	d := NewDAG("entry")

	err := d.AddNode("double", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["entry"].(int) * 2, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("add10", []NodeID{"double"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["double"].(int) + 10, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := d.Instantiate(5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	results, err := inst.Run(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if results["entry"].(int) != 5 {
		t.Errorf("expected entry result to be 5, got %v", results["entry"])
	}

	if results["double"].(int) != 10 {
		t.Errorf("expected double result to be 10, got %v", results["double"])
	}

	if results["add10"].(int) != 20 {
		t.Errorf("expected add10 result to be 20, got %v", results["add10"])
	}
}

func TestDAGInstance_Run_ParallelNodes(t *testing.T) {
	d := NewDAG("entry")

	// Create multiple nodes that depend only on entry - they should run in parallel
	err := d.AddNode("node1", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		time.Sleep(50 * time.Millisecond)
		return "result1", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("node2", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		time.Sleep(50 * time.Millisecond)
		return "result2", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("node3", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		time.Sleep(50 * time.Millisecond)
		return "result3", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := d.Instantiate(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	start := time.Now()
	results, err := inst.Run(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// If they ran in parallel, total time should be ~50ms, not ~150ms
	if elapsed > 150*time.Millisecond {
		t.Errorf("expected parallel execution, but took %v", elapsed)
	}

	if results["node1"].(string) != "result1" {
		t.Errorf("expected node1 result to be 'result1', got %v", results["node1"])
	}

	if results["node2"].(string) != "result2" {
		t.Errorf("expected node2 result to be 'result2', got %v", results["node2"])
	}

	if results["node3"].(string) != "result3" {
		t.Errorf("expected node3 result to be 'result3', got %v", results["node3"])
	}
}

func TestDAGInstance_Run_WithError(t *testing.T) {
	d := NewDAG("entry")

	testErr := errors.New("test error")

	err := d.AddNode("failing", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return nil, testErr
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("dependent", []NodeID{"failing"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "should not run", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := d.Instantiate(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = inst.Run(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}

	if !errors.Is(err, testErr) {
		t.Errorf("expected error to contain test error, got %v", err)
	}
}

func TestDAGInstance_Run_ContextCancellation(t *testing.T) {
	d := NewDAG("entry")

	err := d.AddNode("slow", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		select {
		case <-time.After(1 * time.Second):
			return "completed", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := d.Instantiate(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = inst.Run(ctx)
	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestDAGInstance_RunAsync(t *testing.T) {
	ctx := context.Background()
	d := NewDAG("entry")

	err := d.AddNode("node1", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result1", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := d.Instantiate(10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	future := inst.RunAsync(ctx)
	results, err := future.Get(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if results["node1"].(string) != "result1" {
		t.Errorf("expected node1 result to be 'result1', got %v", results["node1"])
	}
}

func TestDAGInstance_SubDAG(t *testing.T) {
	// Create sub DAG
	sub := NewDAG("x")
	err := sub.AddNode("square", []NodeID{"x"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["x"].(int) * deps["x"].(int), nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Create main DAG
	main := NewDAG("input")
	err = main.AddSubGraph(
		"compute",
		[]NodeID{"input"},
		sub,
		func(deps map[NodeID]any) any { return deps["input"] },
		func(result map[NodeID]any) any { return result["square"] },
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = main.AddNode("addTen", []NodeID{"compute"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["compute"].(int) + 10, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = main.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := main.Instantiate(4)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	results, err := inst.Run(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if results["compute"].(int) != 16 {
		t.Errorf("expected compute result to be 16, got %v", results["compute"])
	}

	if results["addTen"].(int) != 26 {
		t.Errorf("expected addTen result to be 26, got %v", results["addTen"])
	}
}

func TestDAGInstance_SubDAG_WithoutMappings(t *testing.T) {
	// Create sub DAG
	sub := NewDAG("x")
	err := sub.AddNode("double", []NodeID{"x"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		input := deps["x"].(map[NodeID]any)
		return input["input"].(int) * 2, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Create main DAG without input/output mappings
	main := NewDAG("input")
	err = main.AddSubGraph(
		"compute",
		[]NodeID{"input"},
		sub,
		nil, // no input mapping
		nil, // no output mapping
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = main.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := main.Instantiate(5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	results, err := inst.Run(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Without output mapping, the result should be the entire sub DAG result map
	computeResult, ok := results["compute"].(map[NodeID]any)
	if !ok {
		t.Errorf("expected compute result to be map[NodeID]any, got %T", results["compute"])
	}

	if computeResult["double"].(int) != 10 {
		t.Errorf("expected double result to be 10, got %v", computeResult["double"])
	}
}

func TestDAGInstance_ToMermaid(t *testing.T) {
	d := NewDAG("entry")

	err := d.AddNode("node1", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result1", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("node2", []NodeID{"node1"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result2", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	mermaid := d.ToMermaid()

	// Check that mermaid diagram contains expected elements
	if mermaid == "" {
		t.Error("expected non-empty mermaid diagram")
	}

	expectedStrings := []string{
		"graph LR",
		`entry["entry"]`,
		`node1(("node1"))`,
		`node2(("node2"))`,
		"entry --> node1",
		"node1 --> node2",
	}

	for _, expected := range expectedStrings {
		if !contains(mermaid, expected) {
			t.Errorf("expected mermaid diagram to contain %q, got:\n%s", expected, mermaid)
		}
	}
}

func TestDAGInstance_ToMermaid_WithSubDAG(t *testing.T) {
	// Create sub DAG
	sub := NewDAG("x")
	err := sub.AddNode("square", []NodeID{"x"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["x"].(int) * deps["x"].(int), nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Create main DAG
	main := NewDAG("input")
	err = main.AddSubGraph(
		"compute",
		[]NodeID{"input"},
		sub,
		func(deps map[NodeID]any) any { return deps["input"] },
		func(result map[NodeID]any) any { return result["square"] },
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = main.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	mermaid := main.ToMermaid()

	expectedStrings := []string{
		"graph LR",
		"subgraph compute [Subgraph compute]",
		"compute.square",
		"compute.x",
		"end",
	}

	for _, expected := range expectedStrings {
		if !contains(mermaid, expected) {
			t.Errorf("expected mermaid diagram to contain %q, got:\n%s", expected, mermaid)
		}
	}
}

func TestNodeTypes(t *testing.T) {
	entry := &EntryNode{}
	if entry.Type() != NodeTypeEntry {
		t.Errorf("expected EntryNode type to be NodeTypeEntry, got %v", entry.Type())
	}

	simple := &SimpleNode{}
	if simple.Type() != NodeTypeSimple {
		t.Errorf("expected SimpleNode type to be NodeTypeSimple, got %v", simple.Type())
	}

	subdag := &SubDAGNode{}
	if subdag.Type() != NodeTypeSubDAG {
		t.Errorf("expected SubDAGNode type to be NodeTypeSubDAG, got %v", subdag.Type())
	}
}

func TestBaseNode(t *testing.T) {
	base := BaseNode{
		id:   "test",
		deps: []NodeID{"dep1", "dep2"},
	}

	if base.ID() != "test" {
		t.Errorf("expected ID to be 'test', got %s", base.ID())
	}

	deps := base.Deps()
	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}

	if deps[0] != "dep1" || deps[1] != "dep2" {
		t.Errorf("expected deps to be ['dep1', 'dep2'], got %v", deps)
	}
}

func TestDAG_ComplexDependencies(t *testing.T) {
	d := NewDAG("entry")

	// Create a diamond-shaped DAG:
	//     entry
	//     /   \
	//   left  right
	//     \   /
	//     merge

	err := d.AddNode("left", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["entry"].(int) + 1, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("right", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["entry"].(int) + 2, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.AddNode("merge", []NodeID{"left", "right"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["left"].(int) + deps["right"].(int), nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = d.Freeze()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	inst, err := d.Instantiate(10)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	results, err := inst.Run(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if results["left"].(int) != 11 {
		t.Errorf("expected left result to be 11, got %v", results["left"])
	}

	if results["right"].(int) != 12 {
		t.Errorf("expected right result to be 12, got %v", results["right"])
	}

	if results["merge"].(int) != 23 {
		t.Errorf("expected merge result to be 23, got %v", results["merge"])
	}
}

func TestDAG_SkipNode(t *testing.T) {
	d := NewDAG("entry")

	_ = d.AddNode("node1", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result1", nil
	})
	_ = d.AddNode("node1-1", []NodeID{"node1"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result1-1", nil
	})

	_ = d.AddNode("node2", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return nil, ErrNodeSkipped
	})
	_ = d.AddNode("node2-1", []NodeID{"node2"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result2-1", nil
	})

	_ = d.Freeze()

	inst, err := d.Instantiate(nil)
	if err != nil {
		t.Fatal(err)
	}

	results, err := inst.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if results["node1"].(string) != "result1" {
		t.Errorf("expected node1 result to be 'result1', got %v", results["node1"])
	}

	if results["node1-1"].(string) != "result1-1" {
		t.Errorf("expected node1-1 result to be 'result1-1', got %v", results["node1-1"])
	}

	if _, exists := results["node2"]; exists {
		t.Errorf("expected node2 to be skipped, but got result %v", results["node2"])
	}

	if _, exists := results["node2-1"]; exists {
		t.Errorf("expected node2-1 to be skipped, but got result %v", results["node2-1"])
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
