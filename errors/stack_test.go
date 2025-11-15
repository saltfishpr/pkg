package errors

import (
	"fmt"
	"strings"
	"testing"
)

func Test_stack_Format(t *testing.T) {
	st := callers(0, 10)
	s := fmt.Sprintf("%+v", st)
	lines := strings.Split(s, "\n")
	if lines[0] != "" {
		t.Errorf("expected first line to be empty, got %q", lines[0])
	}
	if lines[1] != "github.com/saltfishpr/pkg/errors.Test_stack_Format" {
		t.Errorf("expected function name to be Test_stack_Format, got %q", lines[1])
	}
}
