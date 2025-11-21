package future

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	stateFree uint32 = iota
	stateDoing
	stateDone
)

type state[T any] struct {
	noCopy noCopy

	state atomic.Uint32
	done  chan struct{}

	val T
	err error

	stack unsafe.Pointer // *callback[T]
}

func newState[T any]() *state[T] {
	return &state[T]{
		done: make(chan struct{}),
	}
}

func (s *state[T]) set(val T, err error) bool {
	if !s.state.CompareAndSwap(stateFree, stateDoing) {
		return false
	}
	s.val = val
	s.err = err

	s.state.CompareAndSwap(stateDoing, stateDone)
	close(s.done)

	// execute all callbacks
	for {
		head := (*callback[T])(atomic.LoadPointer(&s.stack))
		if head == nil {
			break
		}
		// stack = head.next
		if atomic.CompareAndSwapPointer(&s.stack, unsafe.Pointer(head), unsafe.Pointer(head.next)) {
			head.execOnce(val, err)
			head.next = nil
		}
	}

	return true
}

func (s *state[T]) get(ctx context.Context) (T, error) {
	if s.isDone() {
		return s.val, s.err
	}

	select {
	case <-ctx.Done():
		var value T
		return value, ctx.Err()
	case <-s.done:
		return s.val, s.err
	}
}

func (s *state[T]) subscribe(cb func(T, error)) {
	newCb := &callback[T]{f: cb}
	// push newCb onto the stack
	for {
		oldCb := (*callback[T])(atomic.LoadPointer(&s.stack))

		if s.isDone() {
			cb(s.val, s.err)
			return
		}

		newCb.next = oldCb
		if atomic.CompareAndSwapPointer(&s.stack, unsafe.Pointer(oldCb), unsafe.Pointer(newCb)) {
			// stack may be nil, the execution logic in set will skip, so double check here
			if s.isDone() {
				newCb.execOnce(s.val, s.err)
			}
			return
		}
	}
}

func (s *state[T]) isFree() bool {
	return s.state.Load() == stateFree
}

func (s *state[T]) isDone() bool {
	return s.state.Load() == stateDone
}

type callback[T any] struct {
	once sync.Once

	f    func(T, error)
	next *callback[T]
}

func (cb *callback[T]) execOnce(val T, err error) {
	cb.once.Do(func() {
		cb.f(val, err)
	})
}

// noCopy may be added to structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
