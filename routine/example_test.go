package routine_test

import (
	"fmt"
	"time"

	"github.com/saltfishpr/pkg/routine"
)

// ExampleRunSafe 演示 RunSafe 的用法 - 同步执行函数并自动恢复 panic
func ExampleRunSafe() {
	routine.RunSafe(func() {
		fmt.Println("执行任务...")
		panic("出错了!")
	})

	fmt.Println("程序继续执行")

	// Output:
	// 执行任务...
	// 程序继续执行
}

// ExampleRunSafe_withCleanup 演示 RunSafe 带 cleanup 函数的用法
func ExampleRunSafe_withCleanup() {
	routine.RunSafe(func() {
		panic("发生 panic")
	}, func(r interface{}) {
		fmt.Printf("清理资源: %v\n", r)
	})

	// Output:
	// 清理资源: 发生 panic
}

// ExampleGoSafe 演示 GoSafe 的用法 - 异步执行 goroutine 并自动恢复 panic
func ExampleGoSafe() {
	done := make(chan struct{})

	routine.GoSafe(func() {
		fmt.Println("goroutine 执行任务")
		panic("goroutine 出错了")
		close(done)
	})

	<-done
	fmt.Println("主程序继续执行")

	// Output:
	// goroutine 执行任务
	// 主程序继续执行
}

// ExampleGoSafe_multiple 演示启动多个安全的 goroutine
func ExampleGoSafe_multiple() {
	done := make(chan struct{})

	for i := 0; i < 3; i++ {
		routine.GoSafe(func() {
			fmt.Println("工作线程运行中...")
			close(done)
		})
	}

	<-done
	<-done
	<-done
	fmt.Println("所有工作完成")

	// Output:
	// 工作线程运行中...
	// 工作线程运行中...
	// 工作线程运行中...
	// 所有工作完成
}

// ExampleRunWithTimeout_success 演示 RunWithTimeout 在任务正常完成时返回 true
func ExampleRunWithTimeout_success() {
	success := routine.RunWithTimeout(func() {
		fmt.Println("任务执行中...")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("任务完成")
	}, 1*time.Second)

	fmt.Printf("任务成功: %v\n", success)

	// Output:
	// 任务执行中...
	// 任务完成
	// 任务成功: true
}

// ExampleRunWithTimeout_timeout 演示 RunWithTimeout 在任务超时时返回 false
func ExampleRunWithTimeout_timeout() {
	success := routine.RunWithTimeout(func() {
		time.Sleep(2 * time.Second)
		fmt.Println("这个不应该打印")
	}, 100*time.Millisecond)

	fmt.Printf("任务成功: %v\n", success)

	// Output:
	// 任务成功: false
}

// ExampleNewRecovered 演示 Recovered 和 RecoveredError 的用法
func ExampleNewRecovered() {
	defer func() {
		if r := recover(); r != nil {
			recovered := routine.NewRecovered(1, r)
			if err := recovered.AsError(); err != nil {
				fmt.Printf("捕获到错误: %v\n", err)
			}
		}
	}()

	panic("手动触发 panic")
}
