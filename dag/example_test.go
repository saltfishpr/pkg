package dag_test

import (
	"context"
	"fmt"
	"log"

	"github.com/saltfishpr/pkg/dag"
)

// ExampleDAG_simple 演示如何创建和运行一个简单的 DAG
func ExampleDAG_simple() {
	// 创建一个 DAG，entry 节点作为入口
	d := dag.NewDAG("entry")

	// 添加节点 A，依赖于 entry
	_ = d.AddNode("A", []dag.NodeID{"entry"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		input := deps["entry"].(int)
		return input * 2, nil
	})

	// 添加节点 B，依赖于 entry
	_ = d.AddNode("B", []dag.NodeID{"entry"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		input := deps["entry"].(int)
		return input + 10, nil
	})

	// 添加节点 C，依赖于 A 和 B
	_ = d.AddNode("C", []dag.NodeID{"A", "B"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		a := deps["A"].(int)
		b := deps["B"].(int)
		return a + b, nil
	})

	// 冻结 DAG（验证完整性和无环）
	if err := d.Freeze(); err != nil {
		log.Fatal(err)
	}

	// 实例化 DAG 并提供输入
	instance, err := d.Instantiate(5)
	if err != nil {
		log.Fatal(err)
	}

	// 运行 DAG
	results, err := instance.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("A = %d\n", results["A"].(int))
	fmt.Printf("B = %d\n", results["B"].(int))
	fmt.Printf("C = %d\n", results["C"].(int))

	// Output:
	// A = 10
	// B = 15
	// C = 25
}

// ExampleDAG_pipeline 演示如何创建一个流水线式的 DAG
func ExampleDAG_pipeline() {
	d := dag.NewDAG("start")

	// 第一步：验证输入
	_ = d.AddNode("validate", []dag.NodeID{"start"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		input := deps["start"].(string)
		if input == "" {
			return nil, fmt.Errorf("input is empty")
		}
		return input, nil
	})

	// 第二步：转换为大写
	_ = d.AddNode("uppercase", []dag.NodeID{"validate"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		input := deps["validate"].(string)
		return fmt.Sprintf("UPPER: %s", input), nil
	})

	// 第三步：添加前缀
	_ = d.AddNode("prefix", []dag.NodeID{"uppercase"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		input := deps["uppercase"].(string)
		return fmt.Sprintf(">>> %s", input), nil
	})

	if err := d.Freeze(); err != nil {
		log.Fatal(err)
	}

	instance, err := d.Instantiate("hello")
	if err != nil {
		log.Fatal(err)
	}

	results, err := instance.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(results["prefix"].(string))

	// Output:
	// >>> UPPER: hello
}

// ExampleDAG_parallel 演示如何创建并行处理的 DAG
func ExampleDAG_parallel() {
	d := dag.NewDAG("data")

	// 并行处理：计算总和
	_ = d.AddNode("sum", []dag.NodeID{"data"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		numbers := deps["data"].([]int)
		sum := 0
		for _, n := range numbers {
			sum += n
		}
		return sum, nil
	})

	// 并行处理：计算平均值
	_ = d.AddNode("avg", []dag.NodeID{"data"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		numbers := deps["data"].([]int)
		sum := 0
		for _, n := range numbers {
			sum += n
		}
		return float64(sum) / float64(len(numbers)), nil
	})

	// 并行处理：找最大值
	_ = d.AddNode("max", []dag.NodeID{"data"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		numbers := deps["data"].([]int)
		max := numbers[0]
		for _, n := range numbers {
			if n > max {
				max = n
			}
		}
		return max, nil
	})

	// 汇总结果
	_ = d.AddNode("summary", []dag.NodeID{"sum", "avg", "max"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		return fmt.Sprintf("Sum: %d, Avg: %.1f, Max: %d",
			deps["sum"].(int),
			deps["avg"].(float64),
			deps["max"].(int),
		), nil
	})

	if err := d.Freeze(); err != nil {
		log.Fatal(err)
	}

	instance, err := d.Instantiate([]int{1, 2, 3, 4, 5})
	if err != nil {
		log.Fatal(err)
	}

	results, err := instance.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(results["summary"].(string))

	// Output:
	// Sum: 15, Avg: 3.0, Max: 5
}

// ExampleDAG_subDAG 演示如何使用子 DAG
func ExampleDAG_subDAG() {
	// 创建子 DAG：处理单个数字
	subDAG := dag.NewDAG("num")

	_ = subDAG.AddNode("square", []dag.NodeID{"num"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		n := deps["num"].(int)
		return n * n, nil
	})

	_ = subDAG.AddNode("double", []dag.NodeID{"num"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		n := deps["num"].(int)
		return n * 2, nil
	})

	_ = subDAG.AddNode("result", []dag.NodeID{"square", "double"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		square := deps["square"].(int)
		double := deps["double"].(int)
		return square + double, nil
	})

	// 创建主 DAG
	mainDAG := dag.NewDAG("input")

	// 添加子 DAG 节点
	_ = mainDAG.AddSubGraph("process", []dag.NodeID{"input"}, subDAG,
		func(deps map[dag.NodeID]any) any {
			// 输入映射：从主 DAG 的依赖中提取输入
			return deps["input"].(int)
		},
		func(results map[dag.NodeID]any) any {
			// 输出映射：从子 DAG 的结果中提取输出
			return results["result"]
		},
	)

	// 添加后续节点
	_ = mainDAG.AddNode("final", []dag.NodeID{"process"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		processResult := deps["process"].(int)
		return processResult + 100, nil
	})

	if err := mainDAG.Freeze(); err != nil {
		log.Fatal(err)
	}

	instance, err := mainDAG.Instantiate(3)
	if err != nil {
		log.Fatal(err)
	}

	results, err := instance.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Process result: %d\n", results["process"].(int))
	fmt.Printf("Final result: %d\n", results["final"].(int))

	// Output:
	// Process result: 15
	// Final result: 115
}

// ExampleDAGInstance_RunAsync 演示如何异步运行 DAG
func ExampleDAGInstance_RunAsync() {
	d := dag.NewDAG("input")

	_ = d.AddNode("step1", []dag.NodeID{"input"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		return deps["input"].(int) * 2, nil
	})

	_ = d.AddNode("step2", []dag.NodeID{"step1"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		return deps["step1"].(int) + 5, nil
	})

	if err := d.Freeze(); err != nil {
		log.Fatal(err)
	}

	instance, err := d.Instantiate(10)
	if err != nil {
		log.Fatal(err)
	}

	// 异步运行 DAG
	future := instance.RunAsync(context.Background())

	// 可以在这里做其他事情...

	// 等待结果
	results, err := future.Get()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result: %d\n", results["step2"].(int))

	// Output:
	// Result: 25
}

// ExampleDAGInstance_ToMermaid 演示如何生成 Mermaid 图表
func ExampleDAGInstance_ToMermaid() {
	d := dag.NewDAG("start")

	_ = d.AddNode("A", []dag.NodeID{"start"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		return nil, nil
	})

	_ = d.AddNode("B", []dag.NodeID{"start"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		return nil, nil
	})

	_ = d.AddNode("C", []dag.NodeID{"A", "B"}, func(ctx context.Context, deps map[dag.NodeID]any) (any, error) {
		return nil, nil
	})

	if err := d.Freeze(); err != nil {
		log.Fatal(err)
	}

	instance, err := d.Instantiate(nil)
	if err != nil {
		log.Fatal(err)
	}

	// 生成 Mermaid 图表
	mermaid := instance.ToMermaid()
	fmt.Println(mermaid)

	// Output:
	// graph LR
	// 	A(("A"))
	// 	B(("B"))
	// 	C(("C"))
	// 	start["start"]
	// 	start --> A
	// 	start --> B
	// 	A --> C
	// 	B --> C
}
