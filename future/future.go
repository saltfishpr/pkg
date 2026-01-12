// Package future provides a simple implementation of Promise-Future pattern in Go.
// Inspired by https://github.com/jizhuozhi/go-future
package future

// Promise The Promise provides a facility to store a value or an error that is later acquired asynchronously via a Future
// created by the Promise. Note that the Promise object is meant to be set only once.
//
// Each Promise is associated with a shared state, which contains some state information and a result which may be not yet evaluated,
// evaluated to a value (possibly nil) or evaluated to an error.
//
// The Promise is the "push" end of the promise-future communication channel: the operation that stores a value in the shared state
// synchronizes-with (as defined in Go's memory model) the successful return from any function that is waiting on the shared state
// (such as Future.Get).
//
// A Promise must not be copied after first use.
type Promise[T any] struct {
	state *state[T]
}

// NewPromise creates a new Promise object.
func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{
		state: newState[T](),
	}
}

// Set sets the value and error of the Promise.
// It panics if the Promise is already satisfied.
func (p *Promise[T]) Set(val T, err error) {
	if !p.state.set(val, err) {
		panic("promise already satisfied")
	}
}

// SetSafety sets the value and error of the Promise, and it will return false if already set.
func (p *Promise[T]) SetSafety(val T, err error) bool {
	return p.state.set(val, err)
}

// Future returns a Future object associated with the Promise.
func (p *Promise[T]) Future() *Future[T] {
	return &Future[T]{state: p.state}
}

// IsFree returns true if the Promise is not set.
func (p *Promise[T]) IsFree() bool {
	return p.state.isFree()
}

// Future The Future provides a mechanism to access the result of asynchronous operations:
//
// 1. An asynchronous operation (Async and Promise) can provide a Future to the creator of that asynchronous operation.
//
// 2. The creator of the asynchronous operation can then use a variety of methods to query, wait for, or extract a value from the Future.
// These methods may block if the asynchronous operation has not yet provided a value.
//
// 3. When the asynchronous operation is ready to send a result to the creator, it can do so by modifying shared state (e.g. Promise.Set)
// that is linked to the creator's std::future.
//
// The Future also has the ability to register a callback to be called when the asynchronous operation is ready to send a result to the creator.
type Future[T any] struct {
	state *state[T]
}

// Get returns the value and error of the Future.
func (f *Future[T]) Get() (T, error) {
	return f.state.get()
}

// Subscribe registers a callback to be called when the Future is done.
//
// NOTE: The callback will be called in goroutine that is the same as the goroutine which changed Future state.
// The callback should not contain any blocking operations.
func (f *Future[T]) Subscribe(cb func(val T, err error)) {
	f.state.subscribe(cb)
}

// IsDone returns true if the Future is done.
func (f *Future[T]) IsDone() bool {
	return f.state.isDone()
}
