package routine

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
)

// Recover 捕获 panic 并调用清理函数。
//
// 应该在 defer 语句中使用。如果发生 panic，会依次调用 cleanups 函数，
// 将 panic 值作为参数传递给每个清理函数。
//
// 示例：
//
//	defer routine.Recover(func(r interface{}) {
//	    log.Printf("发生 panic: %v", r)
//	})
func Recover(cleanups ...func(r interface{})) {
	if r := recover(); r != nil {
		for _, cleanup := range cleanups {
			cleanup(r)
		}
	}
}

// Recovered 存储从 panic 中恢复的值及其调用栈。
//
// 用于捕获 panic 的完整上下文，包括 panic 值和堆栈跟踪。
// 可以通过 AsError 方法转换为 error 类型，获得符合 pkg/errors 错误链接口的堆栈信息。
type Recovered struct {
	Value   interface{} // panic 时传递的值
	Callers []uintptr   // panic 发生时的调用栈
}

// NewRecovered 创建一个 Recovered 实例，捕获当前调用栈。
//
// 参数：
//   - skip: 要跳过的调用栈帧数，通常设置为 1 以跳过 NewRecovered 自身
//   - value: panic 的值（从 recover() 返回）
//
// 返回包含完整调用栈信息的 Recovered 实例。
//
// 示例：
//
//	defer func() {
//	    if r := recover(); r != nil {
//	        recovered := routine.NewRecovered(1, r)
//	        // 使用 recovered 处理错误
//	    }
//	}()
func NewRecovered(skip int, value any) *Recovered {
	// 分配固定大小的数组存储调用栈，避免堆分配
	var callers [32]uintptr
	n := runtime.Callers(skip+1, callers[:])
	return &Recovered{
		Value:   value,
		Callers: callers[:n],
	}
}

// AsError 将 Recovered 转换为 error 类型。
//
// 如果 r 为 nil，返回 nil。否则返回 RecoveredError，其 Error() 方法
// 包含 panic 值和完整的堆栈跟踪。
//
// 返回的错误实现了 pkg/errors 的 StackTrace 接口，可以配合
// %+v 格式化打印完整堆栈。
func (p *Recovered) AsError() error {
	if p == nil {
		return nil
	}
	return &RecoveredError{p}
}

// RecoveredError 实现了 error 接口的 panic 错误，包含完整的堆栈跟踪。
//
// 嵌入 Recovered 以复用其字段，符合 pkg/errors 的 StackTrace 接口约定。
type RecoveredError struct {
	*Recovered
}

// Error 返回 panic 的详细信息，包括值和堆栈跟踪。
func (e *RecoveredError) Error() string {
	return fmt.Sprintf("panic: %v\nstacktrace:%+v", e.Value, e.StackTrace())
}

// StackTrace 返回符合 pkg/errors.StackTrace 接口的堆栈跟踪。
//
// 可以通过 fmt.Sprintf("%+v", err) 打印完整的调用堆栈信息。
func (e *RecoveredError) StackTrace() errors.StackTrace {
	if e == nil {
		return nil
	}
	// 将原始调用栈转换为 pkg/errors.Frame 类型
	frames := make([]errors.Frame, len(e.Callers))
	for i, pc := range e.Callers {
		frames[i] = errors.Frame(pc)
	}
	return frames
}
