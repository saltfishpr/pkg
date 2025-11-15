package executors

type GoExecutor struct{}

func (GoExecutor) Submit(f func()) {
	go f()
}
