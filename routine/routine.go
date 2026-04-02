// Package routine provides panic-safe helpers for running functions
// synchronously, in goroutines, or with a timeout.
//
// Core functions:
//   - [RunSafe]: run a function synchronously, recovering any panic.
//   - [GoSafe]: same as RunSafe but in a new goroutine.
//   - [RunWithTimeout]: run a function in a goroutine with a deadline.
//   - [Recover]: a defer-friendly panic catcher with cleanup callbacks.
//
// [Recovered] and [RecoveredError] capture the panic value together with a
// full stack trace compatible with github.com/pkg/errors formatting.
package routine

import (
	"time"
)

// RunSafe executes fn synchronously, recovering any panic. Optional cleanup
// functions receive the panic value for logging or resource release.
func RunSafe(fn func(), cleanup ...func(r interface{})) {
	defer Recover(cleanup...)

	fn()
}

// GoSafe starts fn in a new goroutine and recovers any panic, preventing
// the process from crashing. Optional cleanup functions are called with the
// panic value.
func GoSafe(fn func(), cleanup ...func(r interface{})) {
	go RunSafe(fn, cleanup...)
}

// RunWithTimeout starts fn in a new goroutine and waits up to timeout for
// it to finish. It returns true if fn completes in time, or false on
// timeout.
//
// Note: a timed-out fn is not cancelled and will continue running in the
// background until it returns naturally.
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
