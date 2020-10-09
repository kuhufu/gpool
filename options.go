package gpool

type Option func(g *GPool)

func WithCoreNum(num int) Option {
	return func(g *GPool) {
		g.workerNum = num
	}
}

func WithQueueLen(num int) Option {
	return func(g *GPool) {
		g.taskChan = make(chan Task, num)
	}
}
