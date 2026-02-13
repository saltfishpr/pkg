// Package routine 提供安全的 goroutine 执行和 panic 恢复工具。
//
// 主要功能：
//   - RunSafe/GoSafe: 自动捕获 panic 的同步/异步函数执行
//   - RunWithTimeout: 带超时控制的函数执行
//   - Recover: panic 恢复和堆栈跟踪
//
// 使用场景：
//   - 需要在 goroutine 中执行可能 panic 的函数时，使用 GoSafe 避免程序崩溃
//   - 需要限制函数执行时间时，使用 RunWithTimeout
//   - 需要捕获并处理 panic 时，配合 Recovered/RecoveredError 使用
package routine
