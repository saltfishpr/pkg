package retry

import (
	"time"
)

// RetryStrategy 定义重试之间的等待时间计算方式。
// attempt 从 0 开始，表示第几次重试（0 表示首次失败后的第一次重试）。
type RetryStrategy interface {
	// NextBackoff 返回在指定尝试次数后应该等待的时长。
	// attempt 从 0 开始计数。
	NextBackoff(attempt int) time.Duration
}

type fixedBackoff time.Duration

// FixedBackoff 返回使用固定间隔的退避策略。
// 每次重试前等待相同的时长。
func FixedBackoff(d time.Duration) fixedBackoff {
	return fixedBackoff(d)
}

func (f fixedBackoff) NextBackoff(attempt int) time.Duration {
	return time.Duration(f)
}

type linearBackoff time.Duration

// LinearBackoff 返回线性增长的退避策略。
// 第 n 次重试等待 (n+1) * d 时长。
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

// ExponentialBackoff 返回指数退避策略。
// 第 n 次重试等待 min(base * 2^n, max) 时长，避免等待时间无限增长。
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
