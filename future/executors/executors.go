// Package executors provides built-in [future.Executor] implementations.
package executors

// GoExecutor spawns a new goroutine for every submitted task.
// It is the default executor used by the future package.
type GoExecutor struct{}

// Submit starts f in a new goroutine.
func (GoExecutor) Submit(f func()) {
	go f()
}

// PoolExecutor limits concurrency to a fixed number of workers using a
// semaphore channel. Each submitted task acquires a slot before running
// and releases it on completion.
type PoolExecutor struct {
	sem chan struct{}
}

// NewPoolExecutor creates a [PoolExecutor] that allows at most maxWorkers
// concurrent tasks.
func NewPoolExecutor(maxWorkers int) *PoolExecutor {
	return &PoolExecutor{
		sem: make(chan struct{}, maxWorkers),
	}
}

// Submit blocks until a worker slot is available, then runs f in a new
// goroutine. The slot is released when f returns.
func (p *PoolExecutor) Submit(f func()) {
	p.sem <- struct{}{}
	go func() {
		defer func() { <-p.sem }()
		f()
	}()
}
