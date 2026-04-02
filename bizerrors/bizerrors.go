// Package bizerrors provides structured business errors with numeric codes,
// human-readable messages, and automatic stack trace capture.
//
// Each [Error] carries a code, a message, an optional key-value detail map,
// and a captured call stack compatible with [github.com/pkg/errors].
// All With* methods return a shallow copy, so error definitions can be safely
// shared as package-level sentinels and specialized per request:
//
//	var ErrUserNotFound = bizerrors.New(1001, "user not found")
//
//	// Later, in a handler:
//	return ErrUserNotFound.WithCause(err).WithDetailPair("uid", uid)
//
// Use [FromError] to extract a *Error from an arbitrary error chain, leveraging
// the standard [errors.As] semantics.
package bizerrors

import (
	"errors"
	"fmt"
)

// Error is a business error that carries a numeric code, a human-readable
// message, an optional cause, structured detail pairs, and a call stack.
//
// All With* methods return a new copy; the receiver is never mutated,
// so sentinel errors can be defined at package level and safely reused.
type Error struct {
	error   // optional underlying cause
	*stack  // always non-nil; captured at creation
	code    int32
	message string
	details map[string]string
}

// New creates an [Error] with the given code and message, capturing the
// current call stack.
func New(code int32, message string) *Error {
	return &Error{
		stack:   callers(1, 16),
		code:    code,
		message: message,
		details: make(map[string]string),
	}
}

// Error returns a string representation including code, message, and cause.
func (e *Error) Error() string {
	return fmt.Sprintf("code=%d, message=%s, cause=%v", e.code, e.message, e.error)
}

// Format implements [fmt.Formatter].
//
//   - %v  – same as %s (code + message + cause).
//   - %+v – verbose: prints the cause (if any) with its own stack, then
//     code + message followed by the full stack trace.
//   - %s  – short string returned by [Error.Error].
//   - %q  – quoted string.
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			if e.error != nil {
				fmt.Fprintf(s, "%+v\n", e.error)
			}
			fmt.Fprintf(s, "code=%d, message=%s", e.code, e.message)
			e.stack.StackTrace().Format(s, verb)
			return
		}
		fallthrough
	case 's':
		fmt.Fprintf(s, "%s", e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// Unwrap returns the underlying cause, enabling [errors.Is] and [errors.As]
// to traverse the error chain.
func (e *Error) Unwrap() error {
	return e.error
}

// StackTrace returns the captured call stack as a [StackTrace] compatible
// with github.com/pkg/errors.
func (e *Error) StackTrace() StackTrace {
	return e.stack.StackTrace()
}

// GetCode returns the business error code.
func (e *Error) GetCode() int32 {
	return e.code
}

// GetMessage returns the human-readable error message.
func (e *Error) GetMessage() string {
	return e.message
}

// GetDetails returns the key-value detail map.
func (e *Error) GetDetails() map[string]string {
	return e.details
}

// WithCause returns a copy of e with the given cause attached.
// If cause is nil, e itself is returned.
// If cause is already a *Error, that error is returned directly to preserve
// the original business context.
func (e *Error) WithCause(cause error) *Error {
	if cause == nil {
		return e
	}
	var res *Error
	if errors.As(cause, &res) {
		return res
	}
	return &Error{
		error:   cause,
		stack:   e.stack,
		code:    e.code,
		message: e.message,
		details: e.details,
	}
}

// WithMessage returns a copy of e with a replaced message.
func (e *Error) WithMessage(message string) *Error {
	return &Error{
		error:   e.error,
		stack:   e.stack,
		code:    e.code,
		message: message,
		details: e.details,
	}
}

// WithDetails returns a copy of e whose detail map is the merge of the
// existing details and extra. Keys in extra overwrite collisions.
func (e *Error) WithDetails(extra map[string]string) *Error {
	newDetails := make(map[string]string, len(e.details)+len(extra))
	for k, v := range e.details {
		newDetails[k] = v
	}
	for k, v := range extra {
		newDetails[k] = v
	}
	return &Error{
		error:   e.error,
		stack:   e.stack,
		code:    e.code,
		message: e.message,
		details: newDetails,
	}
}

// WithDetailPair returns a copy of e with one additional detail key-value pair.
func (e *Error) WithDetailPair(key string, value string) *Error {
	details := make(map[string]string, len(e.details)+1)
	for k, v := range e.details {
		details[k] = v
	}
	details[key] = value
	return &Error{
		error:   e.error,
		stack:   e.stack,
		code:    e.code,
		message: e.message,
		details: details,
	}
}

// WithStack returns a copy of e with a freshly captured stack trace,
// useful when re-raising a sentinel error from a new call site.
func (e *Error) WithStack() *Error {
	return &Error{
		error:   e.error,
		stack:   callers(1, 16),
		code:    e.code,
		message: e.message,
		details: e.details,
	}
}

// WithStackSkip is like [Error.WithStack] but lets the caller skip additional
// frames, which is helpful when wrapping through helper functions.
func (e *Error) WithStackSkip(skip int) *Error {
	return &Error{
		error:   e.error,
		stack:   callers(skip+1, 16),
		code:    e.code,
		message: e.message,
		details: e.details,
	}
}

// FromError extracts a *Error from err's chain using [errors.As].
// It returns nil when err is nil or does not contain a *Error.
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	var bizErr *Error
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return nil
}
