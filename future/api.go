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

var (
	ErrPanic   = errors.New("async panic")
	ErrTimeout = errors.New("future timeout")
)

// Async 异步执行函数并返回 Future。使用默认执行器（goroutine）执行函数。
// 返回的 Future 将在函数完成时被设置结果。
// 如果函数 panic，会捕获并转换为 error（ErrPanic）。
func Async[T any](f func() (T, error)) *Future[T] {
	return Submit(executor, f)
}

// Submit 使用指定的执行器异步执行函数并返回 Future。
// 返回的 Future 将在函数完成时被设置结果。
// 如果函数 panic，会捕获并转换为 error（ErrPanic）。
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

// Done 返回一个已完成的 Future，携带指定的值（error 为 nil）。
func Done[T any](val T) *Future[T] {
	return Done2(val, nil)
}

// Done2 返回一个已完成的 Future，携带指定的值和错误。
func Done2[T any](val T, err error) *Future[T] {
	s := &state[T]{}
	s.set(val, err)
	return &Future[T]{state: s}
}

// Then 在指定的 Future 完成后执行回调函数，并返回一个新的 Future。
// 回调函数接收原 Future 的结果（值和错误），返回转换后的结果。
// 类似 Promise 链式调用。
func Then[T any, R any](f *Future[T], cb func(T, error) (R, error)) *Future[R] {
	s := &state[R]{}
	f.state.subscribe(func(val T, err error) {
		rval, rerr := cb(val, err)
		s.set(rval, rerr)
	})
	return &Future[R]{state: s}
}

// AllOf 等待所有 Future 完成，返回一个包含所有结果的 Future。
// 如果任意一个 Future 失败（error 非 nil），返回的 Future 也会立即设置为该错误。
// 如果传入的 Future 切片为空，返回一个已完成的空结果 Future。
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

// Timeout 为 Future 添加超时控制，返回一个新的 Future。
// 如果原始 Future 在指定时长内完成，返回其结果。
// 如果超时，返回的 Future 将被设置为 ErrTimeout 错误。
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

// WithContext 返回一个 Future，该 Future 在原始 Future 完成或 Context 被取消时完成，取两者中先发生者。
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
		case <-s.done: // s.set will close s.done
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
