package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
	"testing"
)

func TestWithStack(t *testing.T) {
	err1 := stderrors.New("oops")
	err2 := WithStack(err1)

	if err2.Error() != "oops" {
		t.Errorf("unexpected error message: %s", err2.Error())
	}

	if Unwrap(err2) != err1 {
		t.Errorf("Unwrap did not return the original error")
	}

	s1 := fmt.Sprintf("%v", err2)
	if s1 != "oops" {
		t.Errorf("unexpected %%v format: %s", s1)
	}

	s2 := fmt.Sprintf("%+v", err2)
	lines2 := strings.Split(s2, "\n")
	if lines2[0] != "oops" {
		t.Errorf("unexpected %%+v format first line: %s", lines2[0])
	}
	if lines2[1] != "github.com/saltfishpr/pkg/errors.TestWithStack" {
		t.Errorf("unexpected %%+v format second line: %s", lines2[1])
	}

	s3 := fmt.Sprintf("%q", err2)
	if s3 != `"oops"` {
		t.Errorf("unexpected %%q format: %s", s3)
	}
}

func TestWithMessage(t *testing.T) {
	err1 := stderrors.New("oops")
	err2 := WithMessage(err1, "additional context")

	if err2.Error() != "additional context: oops" {
		t.Errorf("unexpected error message: %s", err2.Error())
	}

	if Unwrap(err2) != err1 {
		t.Errorf("Unwrap did not return the original error")
	}

	s1 := fmt.Sprintf("%v", err2)
	if s1 != "additional context: oops" {
		t.Errorf("unexpected %%v format: %s", s1)
	}

	s2 := fmt.Sprintf("%+v", err2)
	if s2 != "oops\nadditional context" {
		t.Errorf("unexpected %%+v format: %s", s2)
	}

	s3 := fmt.Sprintf("%q", err2)
	if s3 != `"additional context: oops"` {
		t.Errorf("unexpected %%q format: %s", s3)
	}
}
