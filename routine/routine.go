package routine

import (
	"time"
)

// RunSafe 同步执行函数 fn，自动捕获并恢复 panic。
//
// 如果 fn 发生 panic，会依次调用 cleanup 函数（如果提供），panic 值会作为参数传递。
// panic 不会向上传播，调用者可以继续执行。
//
// 示例：
//
//	routine.RunSafe(func() {
//	    // 可能 panic 的代码
//	}, func(r interface{}) {
//	    // panic 恢复时的清理逻辑
//	})
func RunSafe(fn func(), cleanup ...func(r interface{})) {
	defer Recover(cleanup...)

	fn()
}

// GoSafe 在新的 goroutine 中异步执行函数 fn，自动捕获并恢复 panic。
//
// 如果 fn 发生 panic，会依次调用 cleanup 函数（如果提供），panic 值会作为参数传递。
// panic 不会导致程序崩溃，也不会向上传播。
//
// 示例：
//
//	routine.GoSafe(func() {
//	    // 在 goroutine 中执行的代码，即使 panic 也不会影响主程序
//	})
func GoSafe(fn func(), cleanup ...func(r interface{})) {
	go RunSafe(fn, cleanup...)
}

// RunWithTimeout 在新 goroutine 中执行函数 fn，并等待其完成或超时。
//
// 如果 fn 在超时时间内完成，返回 true；否则返回 false。
// 注意：即使超时，fn 仍会继续在后台执行，不会被取消。
//
// 参数：
//   - fn: 要执行的函数
//   - timeout: 超时时间
//
// 返回：
//   - true: fn 在超时前完成
//   - false: fn 超时
//
// 示例：
//
//	success := routine.RunWithTimeout(func() {
//	    // 需要限制执行时间的任务
//	}, 5 * time.Second)
//	if !success {
//	    log.Println("任务执行超时")
//	}
func RunWithTimeout(fn func(), timeout time.Duration) bool {
	done := make(chan struct{})

	GoSafe(func() {
		fn()
		close(done)
	})

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
