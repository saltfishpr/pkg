package bizerrors

import (
	"runtime"

	"github.com/pkg/errors"
)

// stack holds a captured set of program counters for stack trace generation.
type stack []uintptr

// callers captures up to depth program counters, skipping skip extra frames
// above the caller. It mirrors the behavior of github.com/pkg/errors but
// allows configurable skip and depth.
func callers(skip int, depth int) *stack {
	if skip < 0 {
		skip = 0
	}
	if depth <= 0 {
		depth = 32
	}
	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip+2, pcs)
	var st stack = pcs[:n]
	return &st
}

type (
	// StackTrace is an alias for [errors.StackTrace] so callers do not need
	// to import github.com/pkg/errors directly.
	StackTrace = errors.StackTrace

	// Frame is an alias for [errors.Frame].
	Frame = errors.Frame
)

// StackTrace converts the raw program counters into a [StackTrace]
// compatible with github.com/pkg/errors formatting verbs.
func (s *stack) StackTrace() StackTrace {
	f := make([]Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = Frame((*s)[i])
	}
	return f
}
