package future

import "github.com/saltfishpr/pkg/future/executors"

// Executor 定义了在 go-future 中执行异步任务的抽象。
//
// 默认情况下，go-future 使用标准 Go goroutines（executors.GoExecutor{}）来执行任务。
// 这提供了轻量级的异步执行，没有池化或并发限制。
//
// 您可以使用 SetExecutor 通过 Executor 接口的任何实现来覆盖默认执行器。
// 常见的模式是使用 ExecutorFunc 来包装 goroutine 池，例如：
//
//	pool := ants.NewPool(100)
//	SetExecutor(ExecutorFunc(func(f func()) {
//	    pool.Submit(f)
//	}))
//
// 大多数情况下不需要更改执行器。替换默认执行器可用于限制并发、重用 goroutine 或减少 GC 压力。
//
// 警告：
//   - 对于 RPC 任务或其他可能阻塞的操作，使用池化执行器可能会导致任务排队和性能下降。
//     只有在了解工作负载并进行了彻底的性能测试后，才应覆盖执行器。
//   - 向 SetExecutor 传递 nil 会 panic。
type Executor interface {
	Submit(func())
}

type ExecutorFunc func(func())

func (e ExecutorFunc) Submit(f func()) {
	e(f)
}

var executor Executor = executors.GoExecutor{}

func SetExecutor(e Executor) {
	if e == nil {
		panic("executor is nil")
	}
	executor = e
}
