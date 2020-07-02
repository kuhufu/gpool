package gpool

import (
	"log"
	"runtime"
	"sync"
)

type GPool struct {
	coreNum  int         //核心Go程数量
	maxNum   int         //最大Go程数量
	curNum   AtomicInt32 //当前Go程数量
	taskChan chan Task   //任务通道
	closed   AtomicBool  //pool是否已关闭
	taskNum  AtomicInt32 //当前任务数

	mu        sync.Mutex
	doneC     chan struct{}
	closeOnce sync.Once
}

type Task struct {
	f func()
}

func New(opts ...Option) *GPool {
	cpuNum := runtime.NumCPU()
	g := &GPool{
		coreNum:  cpuNum,
		maxNum:   cpuNum,
		taskChan: make(chan Task, cpuNum*2),
		doneC:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(g)
	}

	g.start()

	return g
}

//start 启动所有核心 Go 程
func (p *GPool) start() {
	for i := 0; i < p.coreNum; i++ {
		go p.startOneCoreRoutine()
	}
}

//startNewCoreRoutine 启动一个核心 Go 程
func (p *GPool) startOneCoreRoutine() {
label:
	for {
		select {
		case t := <-p.taskChan:
			t.f()
		case <-p.doneC:
			break label
		}
	}
	log.Println("gpool:", "core goroutine exit")
}

func (g *GPool) Do(f func()) <-chan struct{} {
	if g.closed.Val() {
		log.Println("已关闭")
		return nil
	}

	g.taskNum.Incr(1)

	doneC := make(chan struct{})

	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.taskChan) < cap(g.taskChan) {
		g.taskChan <- Task{
			f: func() {
				defer func() {
					g.taskNum.Incr(-1)
					close(doneC)
				}()
				f()
			},
		}
	} else {
		g.curNum.Incr(1)
		go func() {
			defer func() {
				g.taskNum.Incr(-1)
				close(doneC)
				g.curNum.Incr(-1)
			}()
			f()
		}()
	}

	return doneC
}

func (g *GPool) Close() {
	g.closeOnce.Do(func() {
		g.closed.Set(true)
		close(g.doneC)
	})
}
