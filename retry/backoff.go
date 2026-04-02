package retry

import (
	"time"
)

// RetryStrategy computes the delay before each retry attempt.
type RetryStrategy interface {
	// NextBackoff returns the duration to wait before the given attempt.
	// attempt is zero-based: 0 means the pause after the first failure.
	NextBackoff(attempt int) time.Duration
}

type fixedBackoff time.Duration

// FixedBackoff returns a strategy that waits d between every retry.
func FixedBackoff(d time.Duration) fixedBackoff {
	return fixedBackoff(d)
}

func (f fixedBackoff) NextBackoff(attempt int) time.Duration {
	return time.Duration(f)
}

type linearBackoff time.Duration

// LinearBackoff returns a strategy whose delay grows linearly:
// wait = (attempt + 1) * d.
func LinearBackoff(d time.Duration) linearBackoff {
	return linearBackoff(d)
}

func (l linearBackoff) NextBackoff(attempt int) time.Duration {
	return time.Duration(l) * time.Duration(attempt+1)
}

type exponentialBackoff struct {
	baseDuration time.Duration
	maxDuration  time.Duration
}

// ExponentialBackoff returns a strategy whose delay doubles each attempt:
// wait = min(base * 2^attempt, max).
func ExponentialBackoff(baseDuration time.Duration, maxDuration time.Duration) *exponentialBackoff {
	return &exponentialBackoff{
		baseDuration: baseDuration,
		maxDuration:  maxDuration,
	}
}

func (e *exponentialBackoff) NextBackoff(attempt int) time.Duration {
	d := e.baseDuration * time.Duration(1<<attempt)
	if d > e.maxDuration {
		return e.maxDuration
	}
	return d
}
