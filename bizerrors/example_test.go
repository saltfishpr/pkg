package bizerrors_test

import (
	"fmt"

	"github.com/saltfishpr/pkg/bizerrors"
	"github.com/saltfishpr/pkg/errors"
)

func ExampleNew() {
	err := bizerrors.New(1001, "User not found") // stack trace from here
	fmt.Printf("%+v\n", err)

	// Output:
	// code=1001, message=User not found
	// github.com/lebensoft/lebensoft-starter/pkg/bizerrors_test.ExampleNew
	// 	/Users/wuwenshuo/repos/lebensoft/lebensoft-starter/pkg/bizerrors/example_test.go:12
	// testing.runExample
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/testing/run_example.go:63
	// testing.runExamples
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/testing/example.go:41
	// testing.(*M).Run
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/testing/testing.go:2339
	// main.main
	// 	_testmain.go:59
	// runtime.main
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/runtime/proc.go:285
	// runtime.goexit
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/runtime/asm_arm64.s:1268
	// code=1001, message=User not found
}

func ExampleError_WithCause() {
	err := errors.New("oops") // stack trace from here
	err1 := bizerrors.New(1001, "User not found").WithCause(err)
	fmt.Printf("%+v\n", err1)

	// Output:
	// oops
	// github.com/lebensoft/lebensoft-starter/pkg/bizerrors_test.ExampleError_WithCause
	// 	/Users/wuwenshuo/repos/lebensoft/lebensoft-starter/pkg/bizerrors/example_test.go:35
	// testing.runExample
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/testing/run_example.go:63
	// testing.runExamples
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/testing/example.go:41
	// testing.(*M).Run
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/testing/testing.go:2339
	// main.main
	// 	_testmain.go:59
	// runtime.main
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/runtime/proc.go:285
	// runtime.goexit
	// 	/Users/wuwenshuo/sdk/go1.25.4/src/runtime/asm_arm64.s:1268
	// code=1001, message=User not found
}
