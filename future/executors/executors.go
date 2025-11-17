package executors

type GoExecutor struct{}

func (GoExecutor) Submit(f func()) {
	go f()
}

type PoolExecutor struct {
	sem chan struct{}
}

func NewPoolExecutor(maxWorkers int) *PoolExecutor {
	return &PoolExecutor{
		sem: make(chan struct{}, maxWorkers),
	}
}

func (p *PoolExecutor) Submit(f func()) {
	p.sem <- struct{}{}
	go func() {
		defer func() { <-p.sem }()
		f()
	}()
}
