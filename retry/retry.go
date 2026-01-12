package retry

import (
	"context"
	"time"
)

type retryOptions struct {
	maxAttempts   int
	retryStrategy RetryStrategy
	shouldRetry   func(err error) bool
}

type RetryOption func(*retryOptions)

func WithMaxAttempts(maxAttempts int) RetryOption {
	return func(opts *retryOptions) {
		opts.maxAttempts = maxAttempts
	}
}

func WithRetryStrategy(strategy RetryStrategy) RetryOption {
	return func(opts *retryOptions) {
		opts.retryStrategy = strategy
	}
}

func WithShouldRetryFunc(fn func(err error) bool) RetryOption {
	return func(opts *retryOptions) {
		opts.shouldRetry = fn
	}
}

func Do[T any](ctx context.Context, f func() (T, error), options ...RetryOption) (T, error) {
	opts := retryOptions{
		maxAttempts:   3,
		retryStrategy: FixedBackoff(100 * time.Millisecond),
		shouldRetry: func(err error) bool {
			return true
		},
	}
	for _, option := range options {
		option(&opts)
	}

	var zero T
	var lastErr error
	for attempt := 0; attempt < opts.maxAttempts; attempt++ {
		// 执行前检查 Context 是否已取消
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		result, err := f()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if opts.shouldRetry != nil && !opts.shouldRetry(err) {
			break
		}

		// 如果是最后一次尝试，则不再等待
		if attempt == opts.maxAttempts-1 {
			break
		}

		duration := opts.retryStrategy.NextBackoff(attempt)
		select {
		case <-time.After(duration):
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}

	return zero, lastErr
}
