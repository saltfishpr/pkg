package errors

import (
	"fmt"
	"runtime"

	pkgerrors "github.com/pkg/errors"
)

type StackError interface {
	error
	Stack() string
}

// stack represents a stack of program counters.
type stack []uintptr

func (st *stack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			frames := runtime.CallersFrames(*st)
			for {
				frame, more := frames.Next()
				fmt.Fprintf(s, "\n%s\n\t%s:%d", frame.Function, frame.File, frame.Line)
				if !more {
					break
				}
			}
		}
	}
}

func (st *stack) Stack() string {
	return fmt.Sprintf("%+v", st)
}

func callers(skip int, depth int) *stack {
	if skip < 0 {
		skip = 0
	}
	if depth <= 0 {
		depth = stackDepth
	}
	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip+2, pcs)
	var st stack = pcs[:n]
	return &st
}

func TraceStack(err error) string {
	if err == nil {
		return ""
	}
	var stack string
	for {
		if se, ok := err.(StackError); ok {
			stack = se.Stack()
		}
		if ste, ok := err.(interface{ StackTrace() pkgerrors.StackTrace }); ok {
			st := ste.StackTrace()
			if len(st) > 0 {
				stack = fmt.Sprintf("%+v", st)
			}
		}
		if err = Unwrap(err); err == nil {
			break
		}
	}
	return stack
}
