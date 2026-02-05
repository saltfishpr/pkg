package routine

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
)

func Recover(cleanups ...func(r interface{})) {
	if r := recover(); r != nil {
		for _, cleanup := range cleanups {
			cleanup(r)
		}
	}
}

type Recovered struct {
	Value   interface{}
	Callers []uintptr
}

func NewRecovered(skip int, value any) *Recovered {
	var callers [32]uintptr
	n := runtime.Callers(skip+1, callers[:])
	return &Recovered{
		Value:   value,
		Callers: callers[:n],
	}
}

func (p *Recovered) AsError() error {
	if p == nil {
		return nil
	}
	return &RecoveredError{p}
}

type RecoveredError struct {
	*Recovered
}

func (e *RecoveredError) Error() string {
	return fmt.Sprintf("panic: %v\nstacktrace:%+v", e.Value, e.StackTrace())
}

func (e *RecoveredError) StackTrace() errors.StackTrace {
	if e == nil {
		return nil
	}
	frames := make([]errors.Frame, len(e.Callers))
	for i, pc := range e.Callers {
		frames[i] = errors.Frame(pc)
	}
	return frames
}
