package routine

import (
	"context"
	"time"
)

func RunSafe(fn func(), cleanup ...func(r interface{})) {
	defer Recover(cleanup...)

	fn()
}

func RunSafeCtx(ctx context.Context, fn func(ctx context.Context), cleanup ...func(ctx context.Context, r interface{})) {
	defer RecoverCtx(ctx, cleanup...)

	fn(ctx)
}

func GoSafe(fn func(), cleanup ...func(r interface{})) {
	go RunSafe(fn, cleanup...)
}

func GoSafeCtx(ctx context.Context, fn func(ctx context.Context), cleanup ...func(ctx context.Context, r interface{})) {
	go RunSafeCtx(ctx, fn, cleanup...)
}

func RunWithTimeout(fn func(), timeout time.Duration) bool {
	done := make(chan struct{})

	GoSafe(func() {
		fn()
		close(done)
	})

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func RunWithTimeoutCtx(ctx context.Context, fn func(ctx context.Context), timeout time.Duration) bool {
	done := make(chan struct{})

	GoSafeCtx(ctx, func(ctx context.Context) {
		fn(ctx)
		close(done)
	})

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
