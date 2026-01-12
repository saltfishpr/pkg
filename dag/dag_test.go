package dag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestNewDAG(t *testing.T) {
	d := NewDAG("entry")
	assert.Equal(t, NodeID("entry"), d.entry)
	assert.Len(t, d.nodes, 1)
	assert.Contains(t, d.nodes, NodeID("entry"))
	assert.False(t, d.frozen)
}

func TestDAG_AddNode(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	require.NoError(t, err)
	assert.Len(t, d.nodes, 2)

	node, exists := d.nodes["node1"]
	require.True(t, exists)
	assert.Equal(t, NodeID("node1"), node.ID())
}

func TestDAG_AddNode_Duplicate(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	require.NoError(t, err)

	err = d.AddNode("node1", []NodeID{"entry"}, fn)
	assert.ErrorIs(t, err, ErrDAGNodeExists)
}

func TestDAG_AddNode_WhenFrozen(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	err := d.Freeze()
	require.NoError(t, err)

	err = d.AddNode("node1", []NodeID{"entry"}, fn)
	assert.ErrorIs(t, err, ErrDAGFrozen)
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
	require.NoError(t, err)
	assert.Len(t, d.nodes, 2)

	node, exists := d.nodes["subgraph1"]
	require.True(t, exists)
	assert.Equal(t, NodeID("subgraph1"), node.ID())
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
	require.NoError(t, err)

	err = d.AddSubGraph(
		"subgraph1",
		[]NodeID{"entry"},
		sub,
		nil,
		nil,
	)
	assert.ErrorIs(t, err, ErrDAGNodeExists)
}

func TestDAG_Freeze(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	assert.True(t, d.frozen)
}

func TestDAG_Freeze_AlreadyFrozen(t *testing.T) {
	d := NewDAG("entry")

	err := d.Freeze()
	require.NoError(t, err)

	err = d.Freeze()
	assert.ErrorIs(t, err, ErrDAGFrozen)
}

func TestDAG_Freeze_Incomplete(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	// Add a node with a missing dependency
	err := d.AddNode("node1", []NodeID{"missing"}, fn)
	require.NoError(t, err)

	err = d.Freeze()
	assert.ErrorIs(t, err, ErrDAGIncomplete)
}

func TestDAG_Freeze_Cyclic(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result", nil
	}

	// Create a cycle: node1 -> node2 -> node3 -> node1
	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	require.NoError(t, err)

	err = d.AddNode("node2", []NodeID{"node1"}, fn)
	require.NoError(t, err)

	err = d.AddNode("node3", []NodeID{"node2"}, fn)
	require.NoError(t, err)

	// Create the cycle by making node1 depend on node3
	d.nodes["node1"].(*SimpleNode).deps = []NodeID{"node3"}

	err = d.Freeze()
	assert.ErrorIs(t, err, ErrDAGCyclic)
}

func TestDAG_Instantiate(t *testing.T) {
	d := NewDAG("entry")

	fn := func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["entry"].(int) + 1, nil
	}

	err := d.AddNode("node1", []NodeID{"entry"}, fn)
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	inst, err := d.Instantiate(10)
	require.NoError(t, err)
	require.NotNil(t, inst)

	assert.Equal(t, d, inst.spec)
	assert.Len(t, inst.nodes, 2)
}

func TestDAG_Instantiate_NotFrozen(t *testing.T) {
	d := NewDAG("entry")

	_, err := d.Instantiate(10)
	assert.ErrorIs(t, err, ErrDAGNotFrozen)
}

func TestDAGInstance_Run_Simple(t *testing.T) {
	d := NewDAG("entry")

	err := d.AddNode("double", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["entry"].(int) * 2, nil
	})
	require.NoError(t, err)

	err = d.AddNode("add10", []NodeID{"double"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["double"].(int) + 10, nil
	})
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	inst, err := d.Instantiate(5)
	require.NoError(t, err)

	results, err := inst.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 5, results["entry"].(int))
	assert.Equal(t, 10, results["double"].(int))
	assert.Equal(t, 20, results["add10"].(int))
}

func TestDAGInstance_Run_ParallelNodes(t *testing.T) {
	d := NewDAG("entry")

	// Create multiple nodes that depend only on entry - they should run in parallel
	err := d.AddNode("node1", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		time.Sleep(50 * time.Millisecond)
		return "result1", nil
	})
	require.NoError(t, err)

	err = d.AddNode("node2", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		time.Sleep(50 * time.Millisecond)
		return "result2", nil
	})
	require.NoError(t, err)

	err = d.AddNode("node3", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		time.Sleep(50 * time.Millisecond)
		return "result3", nil
	})
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	inst, err := d.Instantiate(nil)
	require.NoError(t, err)

	start := time.Now()
	results, err := inst.Run(context.Background())
	elapsed := time.Since(start)

	require.NoError(t, err)

	// If they ran in parallel, total time should be ~50ms, not ~150ms
	assert.Less(t, elapsed, 150*time.Millisecond)

	assert.Equal(t, "result1", results["node1"].(string))
	assert.Equal(t, "result2", results["node2"].(string))
	assert.Equal(t, "result3", results["node3"].(string))
}

func TestDAGInstance_Run_WithError(t *testing.T) {
	d := NewDAG("entry")

	testErr := errors.New("test error")

	err := d.AddNode("failing", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return nil, testErr
	})
	require.NoError(t, err)

	err = d.AddNode("dependent", []NodeID{"failing"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "should not run", nil
	})
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	inst, err := d.Instantiate(nil)
	require.NoError(t, err)

	_, err = inst.Run(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, testErr)
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
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	inst, err := d.Instantiate(nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = inst.Run(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestDAGInstance_RunAsync(t *testing.T) {
	ctx := context.Background()
	d := NewDAG("entry")

	err := d.AddNode("node1", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result1", nil
	})
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	inst, err := d.Instantiate(10)
	require.NoError(t, err)

	future := inst.RunAsync(ctx)
	results, err := future.Get()
	require.NoError(t, err)

	assert.Equal(t, "result1", results["node1"].(string))
}

func TestDAGInstance_SubDAG(t *testing.T) {
	// Create sub DAG
	sub := NewDAG("x")
	err := sub.AddNode("square", []NodeID{"x"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["x"].(int) * deps["x"].(int), nil
	})
	require.NoError(t, err)

	// Create main DAG
	main := NewDAG("input")
	err = main.AddSubGraph(
		"compute",
		[]NodeID{"input"},
		sub,
		func(deps map[NodeID]any) any { return deps["input"] },
		func(result map[NodeID]any) any { return result["square"] },
	)
	require.NoError(t, err)

	err = main.AddNode("addTen", []NodeID{"compute"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["compute"].(int) + 10, nil
	})
	require.NoError(t, err)

	err = main.Freeze()
	require.NoError(t, err)

	inst, err := main.Instantiate(4)
	require.NoError(t, err)

	results, err := inst.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 16, results["compute"].(int))
	assert.Equal(t, 26, results["addTen"].(int))
}

func TestDAGInstance_SubDAG_WithoutMappings(t *testing.T) {
	// Create sub DAG
	sub := NewDAG("x")
	err := sub.AddNode("double", []NodeID{"x"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		input := deps["x"].(map[NodeID]any)
		return input["input"].(int) * 2, nil
	})
	require.NoError(t, err)

	// Create main DAG without input/output mappings
	main := NewDAG("input")
	err = main.AddSubGraph(
		"compute",
		[]NodeID{"input"},
		sub,
		nil, // no input mapping
		nil, // no output mapping
	)
	require.NoError(t, err)

	err = main.Freeze()
	require.NoError(t, err)

	inst, err := main.Instantiate(5)
	require.NoError(t, err)

	results, err := inst.Run(context.Background())
	require.NoError(t, err)

	// Without output mapping, the result should be the entire sub DAG result map
	computeResult, ok := results["compute"].(map[NodeID]any)
	require.True(t, ok)
	assert.Equal(t, 10, computeResult["double"].(int))
}

func TestDAGInstance_ToMermaid(t *testing.T) {
	d := NewDAG("entry")

	err := d.AddNode("node1", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result1", nil
	})
	require.NoError(t, err)

	err = d.AddNode("node2", []NodeID{"node1"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return "result2", nil
	})
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	mermaid := d.ToMermaid()

	// Check that mermaid diagram contains expected elements
	require.NotEmpty(t, mermaid)

	expectedStrings := []string{
		"graph LR",
		`entry["entry"]`,
		`node1(("node1"))`,
		`node2(("node2"))`,
		"entry --> node1",
		"node1 --> node2",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, mermaid, expected)
	}
}

func TestDAGInstance_ToMermaid_WithSubDAG(t *testing.T) {
	// Create sub DAG
	sub := NewDAG("x")
	err := sub.AddNode("square", []NodeID{"x"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["x"].(int) * deps["x"].(int), nil
	})
	require.NoError(t, err)

	// Create main DAG
	main := NewDAG("input")
	err = main.AddSubGraph(
		"compute",
		[]NodeID{"input"},
		sub,
		func(deps map[NodeID]any) any { return deps["input"] },
		func(result map[NodeID]any) any { return result["square"] },
	)
	require.NoError(t, err)

	err = main.Freeze()
	require.NoError(t, err)

	mermaid := main.ToMermaid()

	expectedStrings := []string{
		"graph LR",
		"subgraph compute [Subgraph compute]",
		"compute.square",
		"compute.x",
		"end",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, mermaid, expected)
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
	require.NoError(t, err)

	err = d.AddNode("right", []NodeID{"entry"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["entry"].(int) + 2, nil
	})
	require.NoError(t, err)

	err = d.AddNode("merge", []NodeID{"left", "right"}, func(ctx context.Context, deps map[NodeID]any) (any, error) {
		return deps["left"].(int) + deps["right"].(int), nil
	})
	require.NoError(t, err)

	err = d.Freeze()
	require.NoError(t, err)

	inst, err := d.Instantiate(10)
	require.NoError(t, err)

	results, err := inst.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 11, results["left"].(int))
	assert.Equal(t, 12, results["right"].(int))
	assert.Equal(t, 23, results["merge"].(int))
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
	require.NoError(t, err)

	results, err := inst.Run(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "result1", results["node1"].(string))
	assert.Equal(t, "result1-1", results["node1-1"].(string))
	assert.NotContains(t, results, "node2")
	assert.NotContains(t, results, "node2-1")
}
