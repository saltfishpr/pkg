package bizerrors

import (
	"runtime"

	"github.com/pkg/errors"
)

type stack []uintptr

// callers copied from pkg/errors, but with configurable skip and depth.
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
	StackTrace = errors.StackTrace
	Frame      = errors.Frame
)

// StackTrace 兼容 pkg/errors 包.
func (s *stack) StackTrace() StackTrace {
	f := make([]Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = Frame((*s)[i])
	}
	return f
}
