package gpool

type Option func(g *GPool)

func WithCoreNum(num int) Option {
	return func(g *GPool) {
		g.coreNum = num
	}
}

func WithMaxNum(num int) Option {
	return func(g *GPool) {
		g.maxNum = num
	}
}
