package future

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	stateFree uint64 = iota
	stateDoing
	stateDone
)

const stateDelta = 1 << 32

const (
	maskCounter = 1<<32 - 1
	maskState   = 1<<34 - 1
)

func isFree(st uint64) bool {
	return ((st & maskState) >> 32) == stateFree
}

func isDone(st uint64) bool {
	return ((st & maskState) >> 32) == stateDone
}

type state[T any] struct {
	noCopy noCopy

	state atomic.Uint64  // high 30 bits are flags, mid 2 bits are state, low 32 bits are waiter count.
	stack unsafe.Pointer // *callback[T]
	sema  uint32

	val T
	err error
}

func (s *state[T]) set(val T, err error) bool {
	for {
		st := s.state.Load()
		if !isFree(st) {
			return false
		}
		// state: free -> doing
		if s.state.CompareAndSwap(st, st+stateDelta) {
			s.val = val
			s.err = err

			// state: doing -> done
			st = s.state.Add(stateDelta)
			// wake up all waiters
			for w := st & maskCounter; w > 0; w-- {
				runtime_Semrelease(&s.sema, false, 0)
			}
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
	}
}

func (s *state[T]) get() (T, error) {
	for {
		st := s.state.Load()
		if isDone(st) {
			return s.val, s.err
		}
		// add a waiter atomically
		if s.state.CompareAndSwap(st, st+1) {
			runtime_Semacquire(&s.sema) // wait to be notified
			if !isDone(s.state.Load()) {
				panic("sync: notified before state is done")
			}
			return s.val, s.err
		}
	}
}

func (s *state[T]) subscribe(cb func(T, error)) {
	newCb := &callback[T]{f: cb}
	// push newCb onto the stack
	for {
		oldCb := (*callback[T])(atomic.LoadPointer(&s.stack))

		if isDone(s.state.Load()) {
			cb(s.val, s.err)
			return
		}

		newCb.next = oldCb
		if atomic.CompareAndSwapPointer(&s.stack, unsafe.Pointer(oldCb), unsafe.Pointer(newCb)) {
			// stack may be nil, the execution logic in set will skip, so double check here
			if isDone(s.state.Load()) {
				newCb.execOnce(s.val, s.err)
			}
			return
		}
	}
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
