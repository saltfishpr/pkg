package future

import (
	"errors"
	"fmt"
	"testing"
)

type executorTestError struct{}

func (*executorTestError) Error() string { return "executor unavailable" }

func TestRejected(t *testing.T) {
	cause := &executorTestError{}
	err := rejected(fmt.Errorf("submit: %w", cause))

	if !errors.Is(err, ErrExecutorRejected) {
		t.Error("rejection does not match ErrExecutorRejected")
	}
	if !errors.Is(err, cause) {
		t.Error("rejection does not preserve the executor error")
	}
	var got *executorTestError
	if !errors.As(err, &got) || got != cause {
		t.Error("rejection does not expose the executor error type")
	}
	if want := "future: executor rejected the task: submit: executor unavailable"; err.Error() != want {
		t.Errorf("error message = %q, want %q", err.Error(), want)
	}
}
