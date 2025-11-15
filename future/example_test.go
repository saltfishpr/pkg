package future

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ExampleNewPromise demonstrates creating and using a Promise
func ExampleNewPromise() {
	promise := NewPromise[string]()
	future := promise.Future()

	go func() {
		time.Sleep(50 * time.Millisecond)
		promise.Set("promise result", nil)
	}()

	result, _ := future.Get()
	fmt.Println(result)
	// Output: promise result
}

// ExamplePromise_Set demonstrates setting a Promise value
func ExamplePromise_Set() {
	promise := NewPromise[int]()
	promise.Set(42, nil)

	result, _ := promise.Future().Get()
	fmt.Println(result)
	// Output: 42
}

// ExamplePromise_Set_panic demonstrates that Set panics when called twice
func ExamplePromise_Set_panic() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Panic caught")
		}
	}()

	promise := NewPromise[int]()
	promise.Set(1, nil)
	promise.Set(2, nil) // This will panic
	// Output: Panic caught
}

// ExamplePromise_SetSafety demonstrates safe setting of a Promise
func ExamplePromise_SetSafety() {
	promise := NewPromise[int]()

	ok1 := promise.SetSafety(42, nil)
	ok2 := promise.SetSafety(100, nil)

	fmt.Println("First set:", ok1)
	fmt.Println("Second set:", ok2)
	result, _ := promise.Future().Get()
	fmt.Println("Result:", result)
	// Output: First set: true
	// Second set: false
	// Result: 42
}

// ExamplePromise_SetSafety_withError demonstrates setting a Promise with an error
func ExamplePromise_SetSafety_withError() {
	promise := NewPromise[string]()
	promise.SetSafety("", errors.New("failed"))

	_, err := promise.Future().Get()
	if err != nil {
		fmt.Println("Error received")
	}
	// Output: Error received
}

// ExamplePromise_Free demonstrates checking if a Promise is free
func ExamplePromise_Free() {
	promise := NewPromise[int]()

	fmt.Println("Before set:", promise.Free())
	promise.Set(42, nil)
	fmt.Println("After set:", promise.Free())
	// Output: Before set: true
	// After set: false
}

// ExamplePromise_Future demonstrates getting a Future from a Promise
func ExamplePromise_Future() {
	promise := NewPromise[string]()
	future := promise.Future()

	go func() {
		promise.Set("async value", nil)
	}()

	result, _ := future.Get()
	fmt.Println(result)
	// Output: async value
}

// ExampleAsync demonstrates basic asynchronous execution
func ExampleAsync() {
	future := Async(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "hello", nil
	})

	result, err := future.Get()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(result)
	// Output: hello
}

// ExampleAsync_withError demonstrates error handling
func ExampleAsync_withError() {
	future := Async(func() (string, error) {
		return "", errors.New("something went wrong")
	})

	_, err := future.Get()
	if err != nil {
		fmt.Println("Error occurred")
	}
	// Output: Error occurred
}

// ExampleCtxAsync demonstrates context-aware asynchronous execution
func ExampleCtxAsync() {
	ctx := context.Background()
	future := CtxAsync(ctx, func(ctx context.Context) (int, error) {
		return 42, nil
	})

	result, _ := future.Get()
	fmt.Println(result)
	// Output: 42
}

// ExampleSubmit demonstrates submitting a task to a custom executor
func ExampleSubmit() {
	future := Submit(executor, func() (int, error) {
		return 100, nil
	})

	result, _ := future.Get()
	fmt.Println(result)
	// Output: 100
}

// ExampleDone demonstrates creating a completed future
func ExampleDone() {
	future := Done("immediate result")
	result, _ := future.Get()
	fmt.Println(result)
	// Output: immediate result
}

// ExampleDone2 demonstrates creating a completed future with error
func ExampleDone2() {
	future := Done2("value", errors.New("error"))
	_, err := future.Get()
	if err != nil {
		fmt.Println("Has error")
	}
	// Output: Has error
}

// ExampleAwait demonstrates awaiting a future result
func ExampleAwait() {
	future := Async(func() (string, error) {
		return "awaited result", nil
	})

	result, _ := Await(future)
	fmt.Println(result)
	// Output: awaited result
}

// ExampleThen demonstrates chaining futures
func ExampleThen() {
	future := Async(func() (int, error) {
		return 10, nil
	})

	mapped := Then(future, func(val int, err error) (string, error) {
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Result: %d", val*2), nil
	})

	result, _ := mapped.Get()
	fmt.Println(result)
	// Output: Result: 20
}

// ExampleThen_errorHandling demonstrates error handling in Then
func ExampleThen_errorHandling() {
	future := Async(func() (int, error) {
		return 0, errors.New("initial error")
	})

	mapped := Then(future, func(val int, err error) (string, error) {
		if err != nil {
			return "handled error", nil
		}
		return fmt.Sprintf("%d", val), nil
	})

	result, _ := mapped.Get()
	fmt.Println(result)
	// Output: handled error
}

// ExampleAllOf demonstrates waiting for multiple futures
func ExampleAllOf() {
	f1 := Async(func() (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 1, nil
	})

	f2 := Async(func() (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 2, nil
	})

	f3 := Async(func() (int, error) {
		time.Sleep(25 * time.Millisecond)
		return 3, nil
	})

	all := AllOf(f1, f2, f3)
	results, _ := all.Get()
	fmt.Println(results)
	// Output: [1 2 3]
}

// ExampleAllOf_withError demonstrates AllOf with error
func ExampleAllOf_withError() {
	f1 := Async(func() (int, error) {
		return 1, nil
	})

	f2 := Async(func() (int, error) {
		return 0, errors.New("failure")
	})

	f3 := Async(func() (int, error) {
		return 3, nil
	})

	all := AllOf(f1, f2, f3)
	_, err := all.Get()
	if err != nil {
		fmt.Println("One future failed")
	}
	// Output: One future failed
}

// ExampleTimeout demonstrates timeout functionality
func ExampleTimeout() {
	future := Async(func() (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "too slow", nil
	})

	timeoutFuture := Timeout(future, 50*time.Millisecond)
	_, err := timeoutFuture.Get()
	if err == ErrTimeout {
		fmt.Println("Timeout occurred")
	}
	// Output: Timeout occurred
}

// ExampleTimeout_success demonstrates successful completion before timeout
func ExampleTimeout_success() {
	future := Async(func() (string, error) {
		time.Sleep(10 * time.Millisecond)
		return "fast enough", nil
	})

	timeoutFuture := Timeout(future, 100*time.Millisecond)
	result, err := timeoutFuture.Get()
	if err == nil {
		fmt.Println(result)
	}
	// Output: fast enough
}

// ExampleUntil demonstrates deadline-based timeout
func ExampleUntil() {
	future := Async(func() (string, error) {
		time.Sleep(200 * time.Millisecond)
		return "delayed", nil
	})

	deadline := time.Now().Add(50 * time.Millisecond)
	untilFuture := Until(future, deadline)
	_, err := untilFuture.Get()
	if err == ErrTimeout {
		fmt.Println("Deadline exceeded")
	}
	// Output: Deadline exceeded
}
