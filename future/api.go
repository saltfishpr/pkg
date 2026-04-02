package future

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/saltfishpr/pkg/routine"
)

// Sentinel errors used by the combinators in this package.
var (
	ErrPanic   = errors.New("async panic")
	ErrTimeout = errors.New("future timeout")
)

// Async runs f in a new goroutine (using the package-level executor) and
// returns a [Future] that resolves when f completes. Panics inside f are
// recovered and surfaced as errors wrapping [ErrPanic].
func Async[T any](f func() (T, error)) *Future[T] {
	return Submit(executor, f)
}

// Submit is like [Async] but uses the provided [Executor].
func Submit[T any](e Executor, f func() (T, error)) *Future[T] {
	s := &state[T]{}
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

// Done returns an already-resolved [Future] carrying val with a nil error.
func Done[T any](val T) *Future[T] {
	return Done2(val, nil)
}

// Done2 returns an already-resolved [Future] carrying val and err.
func Done2[T any](val T, err error) *Future[T] {
	s := &state[T]{}
	s.set(val, err)
	return &Future[T]{state: s}
}

// Then chains a callback on f: when f resolves, cb is called with its
// result, and a new [Future] is returned carrying cb's output. This is
// analogous to Promise.then() in JavaScript.
func Then[T any, R any](f *Future[T], cb func(T, error) (R, error)) *Future[R] {
	s := &state[R]{}
	f.state.subscribe(func(val T, err error) {
		rval, rerr := cb(val, err)
		s.set(rval, rerr)
	})
	return &Future[R]{state: s}
}

// AllOf waits for every Future in fs to resolve and collects their values
// into a slice. If any Future fails, the returned Future resolves
// immediately with that error; remaining successes are discarded.
// An empty input yields an already-resolved Future with a nil slice.
func AllOf[T any](fs ...*Future[T]) *Future[[]T] {
	if len(fs) == 0 {
		return Done[[]T](nil)
	}

	var done uint32
	s := &state[[]T]{}
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

// Timeout wraps f with a deadline. If f does not resolve within d, the
// returned Future resolves with [ErrTimeout].
func Timeout[T any](f *Future[T], d time.Duration) *Future[T] {
	var done uint32
	s := &state[T]{}
	timer := time.AfterFunc(d, func() {
		if atomic.CompareAndSwapUint32(&done, 0, 1) {
			var zero T
			s.set(zero, ErrTimeout)
		}
	})
	f.state.subscribe(func(val T, err error) {
		if atomic.CompareAndSwapUint32(&done, 0, 1) {
			s.set(val, err)
			timer.Stop()
		}
	})
	return &Future[T]{state: s}
}

// WithContext races f against ctx. The returned Future resolves with
// whichever completes first: the original Future's result, or the
// context's error.
func WithContext[T any](ctx context.Context, f *Future[T]) *Future[T] {
	var done uint32
	s := &state[T]{}
	routine.GoSafe(func() {
		select {
		case <-ctx.Done():
			if atomic.CompareAndSwapUint32(&done, 0, 1) {
				var zero T
				s.set(zero, ctx.Err())
			}
		case <-s.done:
			return
		}
	})
	f.state.subscribe(func(val T, err error) {
		if atomic.CompareAndSwapUint32(&done, 0, 1) {
			s.set(val, err)
		}
	})
	return &Future[T]{state: s}
}
