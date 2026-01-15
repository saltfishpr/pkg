package routine

import (
	"time"
)

func RunSafe(fn func(), cleanup ...func(r interface{})) {
	defer Recover(cleanup...)

	fn()
}

func GoSafe(fn func(), cleanup ...func(r interface{})) {
	go RunSafe(fn, cleanup...)
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
