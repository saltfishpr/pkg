package bizerrors

import (
	"errors"
	"fmt"
)

type Error struct {
	error   // maybe nil
	*stack  // won't be nil
	code    int32
	message string
	details map[string]string
}

func New(code int32, message string) *Error {
	return &Error{
		stack:   callers(1, 16),
		code:    code,
		message: message,
		details: make(map[string]string),
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("code=%d, message=%s, cause=%v", e.code, e.message, e.error)
}

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

func (e *Error) Unwrap() error {
	return e.error
}

func (e *Error) StackTrace() StackTrace {
	return e.stack.StackTrace()
}

func (e *Error) GetCode() int32 {
	return e.code
}

func (e *Error) GetMessage() string {
	return e.message
}

func (e *Error) GetDetails() map[string]string {
	return e.details
}

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

func (e *Error) WithMessage(message string) *Error {
	return &Error{
		error:   e.error,
		stack:   e.stack,
		code:    e.code,
		message: message,
		details: e.details,
	}
}

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

func (e *Error) WithStack() *Error {
	return &Error{
		error:   e.error,
		stack:   callers(1, 16),
		code:    e.code,
		message: e.message,
		details: e.details,
	}
}

func (e *Error) WithStackSkip(skip int) *Error {
	return &Error{
		error:   e.error,
		stack:   callers(skip+1, 16),
		code:    e.code,
		message: e.message,
		details: e.details,
	}
}

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
