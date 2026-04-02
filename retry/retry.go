// Package retry provides a generic retry loop with pluggable back-off
// strategies and context-aware cancellation.
//
// Basic usage:
//
//	result, err := retry.Do(ctx, func() (string, error) {
//	    return apiCall()
//	})
//
// With options:
//
//	result, err := retry.Do(ctx, f,
//	    retry.WithMaxAttempts(5),
//	    retry.WithRetryStrategy(retry.ExponentialBackoff(100*time.Millisecond, time.Second)),
//	    retry.WithShouldRetryFunc(func(err error) bool {
//	        return isTransientError(err)
//	    }),
//	)
//
// Built-in back-off strategies:
//   - [FixedBackoff]: constant delay between retries.
//   - [LinearBackoff]: linearly increasing delay.
//   - [ExponentialBackoff]: exponential increase capped at a maximum.
package retry

import (
	"context"
	"time"
)

// retryOptions holds the resolved configuration for a [Do] call.
type retryOptions struct {
	maxAttempts   int
	retryStrategy RetryStrategy
	shouldRetry   func(err error) bool
}

// RetryOption configures the retry behavior of [Do].
type RetryOption func(*retryOptions)

// WithMaxAttempts sets the total number of attempts (initial call + retries).
// The default is 3.
func WithMaxAttempts(maxAttempts int) RetryOption {
	return func(opts *retryOptions) {
		opts.maxAttempts = maxAttempts
	}
}

// WithRetryStrategy sets the back-off strategy between attempts.
// The default is a 100 ms fixed back-off.
func WithRetryStrategy(strategy RetryStrategy) RetryOption {
	return func(opts *retryOptions) {
		opts.retryStrategy = strategy
	}
}

// WithShouldRetryFunc registers a predicate that decides whether a given
// error is retryable. Returning false short-circuits the loop immediately.
// The default retries on every error.
func WithShouldRetryFunc(fn func(err error) bool) RetryOption {
	return func(opts *retryOptions) {
		opts.shouldRetry = fn
	}
}

// Do calls f up to maxAttempts times, pausing between failures according to
// the configured [RetryStrategy].
//
// Before each attempt and during the back-off wait, ctx is checked for
// cancellation. If f returns a nil error, its result is returned immediately.
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
