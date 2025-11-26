package routine

import (
	"context"
	"time"
)

func RunSafe(fn func()) {
	defer Recover()

	fn()
}

func RunSafeCtx(ctx context.Context, fn func(ctx context.Context)) {
	defer RecoverCtx(ctx)

	fn(ctx)
}

func GoSafe(fn func()) {
	go RunSafe(fn)
}

func GoSafeCtx(ctx context.Context, fn func(ctx context.Context)) {
	go RunSafeCtx(ctx, fn)
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
