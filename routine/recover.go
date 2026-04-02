package routine

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
)

// Recover catches a panic and invokes every cleanup function with the
// recovered value. It is intended for use in a defer statement:
//
//	defer routine.Recover(func(r interface{}) {
//	    log.Printf("panic: %v", r)
//	})
func Recover(cleanups ...func(r interface{})) {
	if r := recover(); r != nil {
		for _, cleanup := range cleanups {
			cleanup(r)
		}
	}
}

// Recovered holds the value from a recovered panic along with the call
// stack at the point of the panic. Convert it to an error with [Recovered.AsError].
type Recovered struct {
	Value   interface{} // the value passed to panic()
	Callers []uintptr   // program counters at the panic site
}

// NewRecovered captures the current call stack and pairs it with the panic
// value. skip controls how many extra frames to omit (pass 1 to skip
// NewRecovered itself).
func NewRecovered(skip int, value any) *Recovered {
	var callers [32]uintptr
	n := runtime.Callers(skip+1, callers[:])
	return &Recovered{
		Value:   value,
		Callers: callers[:n],
	}
}

// AsError converts p to a [RecoveredError]. It returns nil when p is nil.
func (p *Recovered) AsError() error {
	if p == nil {
		return nil
	}
	return &RecoveredError{p}
}

// RecoveredError wraps a [Recovered] value as an error with a full stack
// trace. It implements the StackTrace interface from github.com/pkg/errors.
type RecoveredError struct {
	*Recovered
}

// Error returns a human-readable message containing the panic value and
// the stack trace.
func (e *RecoveredError) Error() string {
	return fmt.Sprintf("panic: %v\nstacktrace:%+v", e.Value, e.StackTrace())
}

// StackTrace returns a [errors.StackTrace] suitable for %+v formatting.
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
