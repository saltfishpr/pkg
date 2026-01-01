package bizerrors_test

import (
	"fmt"

	"github.com/saltfishpr/pkg/bizerrors"
	"github.com/saltfishpr/pkg/errors"
)

func ExampleNew() {
	err := bizerrors.New(1001, "User not found") // <- stack top
	fmt.Printf("%+v\n", err)

	// Output:
	// code=1001, message=User not found
	// github.com/saltfishpr/pkg/bizerrors_test.ExampleNew
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:11
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:59
	// runtime.main
	// 	runtime/proc.go:285
	// runtime.goexit
	// 	runtime/asm_arm64.s:1268
}

func ExampleError_WithCause() {
	err := errors.New("oops") // <- stack top
	err1 := bizerrors.New(1001, "User not found").WithCause(err)
	fmt.Printf("%+v\n", err1)

	// Output:
	// oops
	// github.com/saltfishpr/pkg/bizerrors_test.ExampleError_WithCause
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:33
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:59
	// runtime.main
	// 	runtime/proc.go:285
	// runtime.goexit
	// 	runtime/asm_arm64.s:1268
	// code=1001, message=User not found
	// github.com/saltfishpr/pkg/bizerrors_test.ExampleError_WithCause
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:34
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:59
	// runtime.main
	// 	runtime/proc.go:285
	// runtime.goexit
	// 	runtime/asm_arm64.s:1268
}

func ExampleError_WithCause_whenCauseIsBizError() {
	cause := bizerrors.New(2002, "Database error") // <- stack top
	err := bizerrors.New(1001, "User not found").WithCause(cause)
	fmt.Printf("%+v\n", err)

	// Output:
	// code=2002, message=Database error
	// github.com/saltfishpr/pkg/bizerrors_test.ExampleError_WithCause_whenCauseIsBizError
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:71
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:61
	// runtime.main
	// 	runtime/proc.go:285
	// runtime.goexit
	// 	runtime/asm_arm64.s:1268
}
