package daemon

import (
	"errors"
	"sync/atomic"
)

var (
	ErrDaemonStartFailed = errors.New("daemon already started or stopped")
	ErrDaemonStopFailed  = errors.New("daemon not started or already stopped")
)

const (
	DaemonStateInitialized int32 = iota
	DaemonStateStarted
	DaemonStateStopped
)

type BaseDaemon struct {
	state atomic.Int32
}

func (d *BaseDaemon) Start() error {
	if !d.state.CompareAndSwap(DaemonStateInitialized, DaemonStateStarted) {
		return nil
	}
	return ErrDaemonStartFailed
}

func (d *BaseDaemon) Stop() error {
	if !d.state.CompareAndSwap(DaemonStateStarted, DaemonStateStopped) {
		return nil
	}
	return ErrDaemonStopFailed
}
