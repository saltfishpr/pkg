package future

import "github.com/saltfishpr/pkg/future/executors"

// Executor abstracts task submission so the scheduling strategy can be
// replaced without changing call-site code.
//
// The default executor is [executors.GoExecutor], which spawns a goroutine
// per task. Override it with [SetExecutor] to use a bounded worker pool for
// back-pressure or goroutine reuse.
//
// Caution: a pool-based executor may cause tasks to queue under load,
// especially for blocking RPC calls. Only override after profiling confirms
// a benefit.
type Executor interface {
	Submit(func())
}

// ExecutorFunc is an adapter that lets an ordinary function satisfy the
// [Executor] interface.
type ExecutorFunc func(func())

// Submit calls the underlying function.
func (e ExecutorFunc) Submit(f func()) {
	e(f)
}

// executor is the package-level default used by [Async].
var executor Executor = executors.GoExecutor{}

// SetExecutor replaces the package-level executor used by [Async].
// Passing nil panics.
func SetExecutor(e Executor) {
	if e == nil {
		panic("executor is nil")
	}
	executor = e
}
