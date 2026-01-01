package retry

import (
	"time"
)

type RetryStrategy interface {
	// attempt starts from 0
	NextBackoff(attempt int) time.Duration
}

type fixedBackoff time.Duration

func FixedBackoff(d time.Duration) fixedBackoff {
	return fixedBackoff(d)
}

func (f fixedBackoff) NextBackoff(attempt int) time.Duration {
	return time.Duration(f)
}

type linearBackoff time.Duration

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
