// Package daemon provides a lightweight lifecycle primitive for long-running
// services.
//
// Embed [BaseDaemon] into a service struct to gain atomic, CAS-based Start
// and Stop methods that enforce a strict Initialized → Started → Stopped
// state machine. Calling Start or Stop out of order returns a descriptive
// sentinel error.
package daemon

import (
	"errors"
	"sync/atomic"
)

// Sentinel errors returned by [BaseDaemon.Start] and [BaseDaemon.Stop].
var (
	ErrDaemonStartFailed = errors.New("daemon already started or stopped")
	ErrDaemonStopFailed  = errors.New("daemon not started or already stopped")
)

// Daemon lifecycle states.
const (
	DaemonStateInitialized int32 = iota
	DaemonStateStarted
	DaemonStateStopped
)

// BaseDaemon is an embeddable struct that tracks a service's lifecycle
// through three states: Initialized → Started → Stopped.
// State transitions are atomic and safe for concurrent use.
type BaseDaemon struct {
	state atomic.Int32
}

// Start transitions from Initialized to Started.
// It returns [ErrDaemonStartFailed] if the daemon has already been started
// or stopped.
func (d *BaseDaemon) Start() error {
	if !d.state.CompareAndSwap(DaemonStateInitialized, DaemonStateStarted) {
		return ErrDaemonStartFailed
	}
	return nil
}

// Stop transitions from Started to Stopped.
// It returns [ErrDaemonStopFailed] if the daemon has not been started or
// has already been stopped.
func (d *BaseDaemon) Stop() error {
	if !d.state.CompareAndSwap(DaemonStateStarted, DaemonStateStopped) {
		return ErrDaemonStopFailed
	}
	return nil
}
