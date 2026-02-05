// Package future 提供了 Promise-Future 模式的简单 Go 语言实现。
package future

// Promise 提供了一种存储值或错误的机制，该值或错误随后可以通过由 Promise 创建的 Future 异步获取。
// 注意：Promise 对象只能被设置一次。
//
// 每个 Promise 关联一个共享状态，该共享状态包含一些状态信息和一个结果，
// 该结果可能尚未计算、已计算为值（可能为 nil）或已计算为错误。
//
// Promise 是 promise-future 通信通道的"推送"端：在共享状态中存储值的操作
// 与等待该共享状态的任何函数（如 Future.Get）的成功返回同步
// （按照 Go 内存模型的定义）。
//
// Promise 在首次使用后不得被复制。
type Promise[T any] struct {
	state *state[T]
}

// NewPromise 创建一个新的 Promise 对象。
func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{
		state: &state[T]{},
	}
}

// Set 设置 Promise 的值和错误。
// 如果 Promise 已经被设置，则会 panic。
func (p *Promise[T]) Set(val T, err error) {
	if !p.state.set(val, err) {
		panic("promise already satisfied")
	}
}

// SetSafety 设置 Promise 的值和错误，如果已经设置则返回 false。
func (p *Promise[T]) SetSafety(val T, err error) bool {
	return p.state.set(val, err)
}

// Future 返回与 Promise 关联的 Future 对象。
func (p *Promise[T]) Future() *Future[T] {
	return &Future[T]{state: p.state}
}

// IsFree 如果 Promise 尚未设置则返回 true。
func (p *Promise[T]) IsFree() bool {
	return p.state.isFree()
}

// Future 提供了一种访问异步操作结果的机制：
//
// 1. 异步操作（Async 和 Promise）可以为该异步操作的创建者提供一个 Future。
//
// 2. 异步操作的创建者可以使用多种方法来查询、等待或从 Future 中提取值。
// 如果异步操作尚未提供值，这些方法可能会阻塞。
//
// 3. 当异步操作准备向创建者发送结果时，可以通过修改与创建者的 Future 关联的
// 共享状态（例如 Promise.Set）来实现。
//
// Future 还具有注册回调的能力，当异步操作准备向创建者发送结果时将调用该回调。
type Future[T any] struct {
	state *state[T]
}

// Get 返回 Future 的值和错误。
func (f *Future[T]) Get() (T, error) {
	return f.state.get()
}

// Subscribe 注册一个回调，在 Future 完成时调用。
//
// 注意：回调将在与改变 Future 状态相同的 goroutine 中调用。
// 回调中不应包含任何阻塞操作。
func (f *Future[T]) Subscribe(cb func(val T, err error)) {
	f.state.subscribe(cb)
}

// IsDone 如果 Future 已完成则返回 true。
func (f *Future[T]) IsDone() bool {
	return f.state.isDone()
}
