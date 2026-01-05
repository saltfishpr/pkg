package bizerrors_test

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/saltfishpr/pkg/bizerrors"
)

func ExampleNew() {
	err := bizerrors.New(1001, "User not found") // <- stack top
	fmt.Printf("%+v\n", err)

	// Output:
	// code=1001, message=User not found
	// github.com/saltfishpr/pkg/bizerrors_test.ExampleNew
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:12
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:63
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
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:34
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:63
	// runtime.main
	// 	runtime/proc.go:285
	// runtime.goexit
	// 	runtime/asm_arm64.s:1268
	// code=1001, message=User not found
	// github.com/saltfishpr/pkg/bizerrors_test.ExampleError_WithCause
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:35
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:63
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
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:72
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:63
	// runtime.main
	// 	runtime/proc.go:285
	// runtime.goexit
	// 	runtime/asm_arm64.s:1268
}

func ExampleError_WithStack() {
	err := bizerrors.New(1001, "User not found")
	err = err.WithStack() // <- stack top
	fmt.Printf("%+v\n", err)

	// Output:
	// code=1001, message=User not found
	// github.com/saltfishpr/pkg/bizerrors_test.ExampleError_WithStack
	// 	github.com/saltfishpr/pkg/bizerrors/example_test.go:96
	// testing.runExample
	// 	testing/run_example.go:63
	// testing.runExamples
	// 	testing/example.go:41
	// testing.(*M).Run
	// 	testing/testing.go:2339
	// main.main
	// 	_testmain.go:63
	// runtime.main
	// 	runtime/proc.go:285
	// runtime.goexit
	// 	runtime/asm_arm64.s:1268
}
