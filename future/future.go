// Package future provides a lightweight Promise-Future concurrency primitive
// for Go, inspired by Java's CompletableFuture and JavaScript's Promise.
//
// A [Promise] is the write end: it stores a value or error exactly once.
// A [Future] is the read end: it blocks until the result is available, or
// delivers it asynchronously via [Future.Subscribe].
//
// Key combinators:
//   - [Async] / [Submit]: run a function asynchronously and return a Future.
//   - [Then]: chain a transformation on a Future's result.
//   - [AllOf]: fan-in, waiting for all Futures to complete.
//   - [Timeout]: race a Future against a deadline.
//   - [WithContext]: race a Future against context cancellation.
//
// The default executor spawns a goroutine per task ([executors.GoExecutor]).
// Use [SetExecutor] to substitute a pooled executor for back-pressure control.
package future

// Promise is the write-once, producer side of the Promise-Future pair.
//
// Call [Promise.Set] exactly once to resolve the associated [Future]. Setting
// a Promise more than once panics (or use [Promise.SetSafety] for a boolean
// check). A Promise must not be copied after first use.
type Promise[T any] struct {
	state *state[T]
}

// NewPromise creates an unresolved Promise together with its internal shared
// state. Obtain the read side via [Promise.Future].
func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{
		state: &state[T]{},
	}
}

// Set resolves the Promise with val and err. It panics if the Promise has
// already been resolved.
func (p *Promise[T]) Set(val T, err error) {
	if !p.state.set(val, err) {
		panic("promise already satisfied")
	}
}

// SetSafety resolves the Promise and returns true, or returns false if it
// was already resolved. It never panics.
func (p *Promise[T]) SetSafety(val T, err error) bool {
	return p.state.set(val, err)
}

// Future returns the read side associated with this Promise.
func (p *Promise[T]) Future() *Future[T] {
	return &Future[T]{state: p.state}
}

// IsFree reports whether the Promise has not yet been resolved.
func (p *Promise[T]) IsFree() bool {
	return p.state.isFree()
}

// Future is the read-only, consumer side of the Promise-Future pair.
//
// [Future.Get] blocks until the result is available. [Future.Subscribe]
// registers a non-blocking callback that fires once the result is ready;
// the callback runs in the goroutine that resolves the Promise, so it must
// not perform blocking operations.
type Future[T any] struct {
	state *state[T]
}

// Get blocks until the Future is resolved and returns the value and error.
func (f *Future[T]) Get() (T, error) {
	return f.state.get()
}

// Subscribe registers cb to be called when the Future resolves.
//
// The callback executes in the same goroutine that calls [Promise.Set], so
// it must be non-blocking.
func (f *Future[T]) Subscribe(cb func(val T, err error)) {
	f.state.subscribe(cb)
}

// IsDone reports whether the Future has been resolved.
func (f *Future[T]) IsDone() bool {
	return f.state.isDone()
}
