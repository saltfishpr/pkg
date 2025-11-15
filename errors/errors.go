package errors

import (
	stderrors "errors"
	"fmt"
	"io"
)

const stackDepth = 32

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
func New(message string) error {
	return WithStackSkip(stderrors.New(message), 1)
}

func Errorf(format string, args ...interface{}) error {
	return WithStackSkip(fmt.Errorf(format, args...), 1)
}

// WithStack annotates err with a stack trace at the point WithStack was called.
// If err is nil, WithStack returns nil.
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return &withStack{
		err,
		callers(1, stackDepth),
	}
}

// WithStackSkip annotates err with a stack trace at the point WithStackSkip was called.
// The skip parameter is the number of stack frames to skip.
// If err is nil, WithStackSkip returns nil.
func WithStackSkip(err error, skip int) error {
	if err == nil {
		return nil
	}
	return &withStack{
		err,
		callers(skip+1, stackDepth),
	}
}

type withStack struct {
	error
	*stack
}

// Unwrap provides compatibility for Go 1.13 error chains.
func (w *withStack) Unwrap() error { return w.error }

func (w *withStack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v", w.error)
			w.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	}
}

// WithMessage annotates err with a new message.
// If err is nil, WithMessage returns nil.
func WithMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		error: err,
		msg:   message,
	}
}

// WithMessagef annotates err with the format specifier.
// If err is nil, WithMessagef returns nil.
func WithMessagef(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		error: err,
		msg:   fmt.Sprintf(format, args...),
	}
}

type withMessage struct {
	error
	msg string
}

func (w *withMessage) Error() string { return w.msg + ": " + w.error.Error() }

// Unwrap provides compatibility for Go 1.13 error chains.
func (w *withMessage) Unwrap() error { return w.error }

func (w *withMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", w.error)
			io.WriteString(s, w.msg)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	}
}
