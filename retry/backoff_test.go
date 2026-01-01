package retry

import (
	"testing"
	"time"
)

func TestFixedBackoff_NextBackoff(t *testing.T) {
	duration := 100 * time.Millisecond
	strategy := FixedBackoff(duration)

	for i := 0; i < 5; i++ {
		if got := strategy.NextBackoff(i); got != duration {
			t.Errorf("FixedBackoff.NextBackoff(%d) = %v, want %v", i, got, duration)
		}
	}
}

func TestLinearBackoff_NextBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	strategy := LinearBackoff(base)

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 300 * time.Millisecond},
		{4, 500 * time.Millisecond},
	}

	for _, tt := range tests {
		if got := strategy.NextBackoff(tt.attempt); got != tt.want {
			t.Errorf("LinearBackoff.NextBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestExponentialBackoff_NextBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	max := 1 * time.Second
	strategy := ExponentialBackoff(base, max)

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond}, // 100 * 2^0 = 100
		{1, 200 * time.Millisecond}, // 100 * 2^1 = 200
		{2, 400 * time.Millisecond}, // 100 * 2^2 = 400
		{3, 800 * time.Millisecond}, // 100 * 2^3 = 800
		{4, 1 * time.Second},        // 100 * 2^4 = 1600 > max -> 1000
		{5, 1 * time.Second},        // 100 * 2^5 = 3200 > max -> 1000
	}

	for _, tt := range tests {
		if got := strategy.NextBackoff(tt.attempt); got != tt.want {
			t.Errorf("ExponentialBackoff.NextBackoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}
