package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDo(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		ctx := context.Background()
		calls := 0
		f := func() (string, error) {
			calls++
			return "success", nil
		}

		res, err := Do(ctx, f)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res != "success" {
			t.Errorf("expected result 'success', got %q", res)
		}
		if calls != 1 {
			t.Errorf("expected 1 call, got %d", calls)
		}
	})

	t.Run("success after retries", func(t *testing.T) {
		ctx := context.Background()
		calls := 0
		f := func() (string, error) {
			calls++
			if calls < 3 {
				return "", errors.New("fail")
			}
			return "success", nil
		}

		// Use a very short backoff for testing speed
		res, err := Do(ctx, f, WithMaxAttempts(5), WithRetryStrategy(FixedBackoff(1*time.Millisecond)))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res != "success" {
			t.Errorf("expected result 'success', got %q", res)
		}
		if calls != 3 {
			t.Errorf("expected 3 calls, got %d", calls)
		}
	})

	t.Run("failure after max attempts", func(t *testing.T) {
		ctx := context.Background()
		calls := 0
		expectedErr := errors.New("fail")
		f := func() (string, error) {
			calls++
			return "", expectedErr
		}

		_, err := Do(ctx, f, WithMaxAttempts(3), WithRetryStrategy(FixedBackoff(1*time.Millisecond)))
		if err != expectedErr {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
		if calls != 3 {
			t.Errorf("expected 3 calls, got %d", calls)
		}
	})

	t.Run("should retry logic", func(t *testing.T) {
		ctx := context.Background()
		calls := 0
		retryErr := errors.New("retry me")
		fatalErr := errors.New("fatal")

		f := func() (string, error) {
			calls++
			if calls == 1 {
				return "", retryErr
			}
			return "", fatalErr
		}

		shouldRetry := func(err error) bool {
			return err == retryErr
		}

		_, err := Do(ctx, f, WithMaxAttempts(5), WithShouldRetryFunc(shouldRetry), WithRetryStrategy(FixedBackoff(1*time.Millisecond)))
		if err != fatalErr {
			t.Fatalf("expected error %v, got %v", fatalErr, err)
		}
		if calls != 2 {
			t.Errorf("expected 2 calls (1 retry + 1 fatal), got %d", calls)
		}
	})

	t.Run("context cancelled before start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		calls := 0
		f := func() (string, error) {
			calls++
			return "success", nil
		}

		_, err := Do(ctx, f)
		if err != context.Canceled {
			t.Fatalf("expected context canceled error, got %v", err)
		}
		if calls != 0 {
			t.Errorf("expected 0 calls, got %d", calls)
		}
	})

	t.Run("context cancelled during retry wait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		f := func() (string, error) {
			calls++
			if calls == 1 {
				// Cancel context after first failure to interrupt the wait
				time.AfterFunc(10*time.Millisecond, cancel)
				return "", errors.New("fail")
			}
			return "success", nil
		}

		// Use a long backoff to ensure we are waiting when context is cancelled
		_, err := Do(ctx, f, WithMaxAttempts(3), WithRetryStrategy(FixedBackoff(200*time.Millisecond)))
		if err != context.Canceled {
			t.Fatalf("expected context canceled error, got %v", err)
		}
		if calls != 1 {
			t.Errorf("expected 1 call, got %d", calls)
		}
	})
}
