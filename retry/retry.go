package retry

import (
	"context"
	"time"
)

type RetryOptions struct {
	maxAttempts   int
	retryStrategy RetryStrategy
	shouldRetry   func(err error) bool
}

type RetryOption func(*RetryOptions)

func WithMaxAttempts(maxAttempts int) RetryOption {
	return func(opts *RetryOptions) {
		opts.maxAttempts = maxAttempts
	}
}

func WithRetryStrategy(strategy RetryStrategy) RetryOption {
	return func(opts *RetryOptions) {
		opts.retryStrategy = strategy
	}
}

func WithShouldRetryFunc(fn func(err error) bool) RetryOption {
	return func(opts *RetryOptions) {
		opts.shouldRetry = fn
	}
}

func defaultRetryOptions() *RetryOptions {
	return &RetryOptions{
		maxAttempts:   3,
		retryStrategy: FixedBackoff(100 * time.Millisecond),
		shouldRetry: func(err error) bool {
			return true
		},
	}
}

func Do[T any](ctx context.Context, f func() (T, error), options ...RetryOption) (T, error) {
	opts := defaultRetryOptions()
	for _, option := range options {
		option(opts)
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
			if attempt > 0 {
				// TODO 考虑记录重试成功的日志
			}
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
