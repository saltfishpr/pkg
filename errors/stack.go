package errors

import (
	"fmt"
	"runtime"
)

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
