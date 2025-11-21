package future

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync/atomic"
)

var (
	ErrPanic   = errors.New("async panic")
	ErrTimeout = errors.New("future timeout")
)

func Async[T any](f func() (T, error)) *Future[T] {
	return Submit(executor, f)
}

func CtxAsync[T any](ctx context.Context, f func(ctx context.Context) (T, error)) *Future[T] {
	return CtxSubmit(ctx, executor, f)
}

func Submit[T any](e Executor, f func() (T, error)) *Future[T] {
	s := newState[T]()
	e.Submit(func() {
		var val T
		var err error
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%w, err=%s, stack=%s", ErrPanic, r, debug.Stack())
			}
			s.set(val, err)
		}()
		val, err = f()
	})
	return &Future[T]{state: s}
}

func CtxSubmit[T any](ctx context.Context, e Executor, f func(ctx context.Context) (T, error)) *Future[T] {
	s := newState[T]()
	e.Submit(func() {
		var val T
		var err error
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%w, err=%s, stack=%s", ErrPanic, r, debug.Stack())
			}
			s.set(val, err)
		}()
		val, err = f(ctx)
	})
	return &Future[T]{state: s}
}

func Done[T any](val T) *Future[T] {
	return Done2(val, nil)
}

func Done2[T any](val T, err error) *Future[T] {
	s := newState[T]()
	s.set(val, err)
	return &Future[T]{state: s}
}

func Then[T any, R any](f *Future[T], cb func(T, error) (R, error)) *Future[R] {
	s := newState[R]()
	f.state.subscribe(func(val T, err error) {
		rval, rerr := cb(val, err)
		s.set(rval, rerr)
	})
	return &Future[R]{state: s}
}

func AllOf[T any](fs ...*Future[T]) *Future[[]T] {
	if len(fs) == 0 {
		return Done[[]T](nil)
	}

	var done uint32
	s := newState[[]T]()
	c := int32(len(fs))
	results := make([]T, len(fs))
	for i, f := range fs {
		i := i
		f.state.subscribe(func(val T, err error) {
			if err != nil {
				if atomic.CompareAndSwapUint32(&done, 0, 1) {
					s.set(nil, err)
				}
			} else {
				results[i] = val
				if atomic.AddInt32(&c, -1) == 0 {
					s.set(results, nil)
				}
			}
		})
	}
	return &Future[[]T]{state: s}
}
