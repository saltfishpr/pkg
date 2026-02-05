package future

import (
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
	once  sync.Once

	val T
	err error

	stack unsafe.Pointer // *callback[T]
}

func (s *state[T]) lazyInit() {
	s.once.Do(func() {
		s.done = make(chan struct{})
	})
}

func (s *state[T]) set(val T, err error) bool {
	if !s.state.CompareAndSwap(stateFree, stateDoing) {
		return false
	}
	s.val = val
	s.err = err

	s.state.CompareAndSwap(stateDoing, stateDone)
	s.lazyInit()
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

func (s *state[T]) get() (T, error) {
	if s.isDone() {
		return s.val, s.err
	}
	s.lazyInit()
	<-s.done
	return s.val, s.err
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

// noCopy 可以添加到首次使用后不得被复制的结构体中。
//
// 详情请参见：https://golang.org/issues/8005#issuecomment-190753527
//
// 注意：由于 Lock 和 Unlock 方法，不得嵌入此结构体。
type noCopy struct{}

// Lock 是一个空操作，由 `go vet` 的 -copylocks 检查器使用。
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
