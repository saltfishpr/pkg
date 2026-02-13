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

// RetryOption 配置 Do 函数的重试行为。
type RetryOption func(*retryOptions)

// WithMaxAttempts 设置最大重试次数（包括首次尝试）。
// 默认为 3 次。
func WithMaxAttempts(maxAttempts int) RetryOption {
	return func(opts *retryOptions) {
		opts.maxAttempts = maxAttempts
	}
}

// WithRetryStrategy 设置重试的退避策略。
// 默认为 100ms 的固定退避。
func WithRetryStrategy(strategy RetryStrategy) RetryOption {
	return func(opts *retryOptions) {
		opts.retryStrategy = strategy
	}
}

// WithShouldRetryFunc 设置判断函数，决定是否应该重试某个错误。
// 返回 false 会立即终止重试并返回该错误。
// 默认会对所有错误进行重试。
func WithShouldRetryFunc(fn func(err error) bool) RetryOption {
	return func(opts *retryOptions) {
		opts.shouldRetry = fn
	}
}

// Do 执行函数 f，失败时按配置的策略重试。
//
// 支持通过 context 取消重试；会在每次尝试前和等待期间检查 ctx 状态。
// 如果 f 返回 nil error，立即返回结果；否则根据配置决定是否重试。
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
		// 每次尝试前检查 Context，避免已取消时仍执行函数
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

		// 最后一次尝试失败后无需等待，直接退出
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
