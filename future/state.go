package future

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// Lifecycle states of a state instance.
const (
	stateFree  uint32 = iota // not yet resolved
	stateDoing               // resolution in progress
	stateDone                // resolved; value and err are final
)

// state is the shared mutable core behind both [Promise] and [Future].
// It stores the result (val + err), a channel for blocking waiters, and a
// lock-free stack of subscriber callbacks.
type state[T any] struct {
	noCopy noCopy

	state atomic.Uint32
	done  chan struct{}
	once  sync.Once

	val T
	err error

	stack unsafe.Pointer // *callback[T]; lock-free Treiber stack
}

// lazyInit creates the done channel on first use, avoiding allocation for
// Futures that are resolved before anyone calls Get.
func (s *state[T]) lazyInit() {
	s.once.Do(func() {
		s.done = make(chan struct{})
	})
}

// set resolves the state exactly once. It stores the value and error, closes
// the done channel, then drains and executes all stacked callbacks.
// Returns false if the state was already resolved.
func (s *state[T]) set(val T, err error) bool {
	if !s.state.CompareAndSwap(stateFree, stateDoing) {
		return false
	}
	s.val = val
	s.err = err

	s.state.CompareAndSwap(stateDoing, stateDone)
	s.lazyInit()
	close(s.done)

	for {
		head := (*callback[T])(atomic.LoadPointer(&s.stack))
		if head == nil {
			break
		}
		if atomic.CompareAndSwapPointer(&s.stack, unsafe.Pointer(head), unsafe.Pointer(head.next)) {
			head.execOnce(val, err)
			head.next = nil
		}
	}

	return true
}

// get blocks until the state is resolved and returns the stored result.
func (s *state[T]) get() (T, error) {
	if s.isDone() {
		return s.val, s.err
	}
	s.lazyInit()
	<-s.done
	return s.val, s.err
}

// subscribe pushes cb onto the lock-free stack. If the state is already
// resolved, cb is invoked immediately in the caller's goroutine.
func (s *state[T]) subscribe(cb func(T, error)) {
	newCb := &callback[T]{f: cb}
	for {
		oldCb := (*callback[T])(atomic.LoadPointer(&s.stack))

		if s.isDone() {
			cb(s.val, s.err)
			return
		}

		newCb.next = oldCb
		if atomic.CompareAndSwapPointer(&s.stack, unsafe.Pointer(oldCb), unsafe.Pointer(newCb)) {
			// The state may have been resolved between the isDone check and
			// the CAS. Double-check to avoid a lost notification.
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

// callback is a singly-linked node in a lock-free Treiber stack of subscriber
// functions. execOnce guarantees at-most-once delivery.
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

// noCopy is embedded to trigger the go vet -copylocks checker.
//
// See https://golang.org/issues/8005#issuecomment-190753527.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
