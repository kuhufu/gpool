package gpool

import (
	"log"
	"runtime"
	"sync"
)

type GPool struct {
	workerNum     int         //核心 worker 数量
	currWorkerNum AtomicInt32 //当前 worker 数量
	taskChan      chan Task   //任务队列
	closed        AtomicBool  //pool是否已关闭
	currTaskNum   AtomicInt32 //当前任务数

	mu        sync.Mutex
	doneC     chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup //等待所有worker退出
}

type Task struct {
	f     func()
	doneC chan struct{}
}

func (t Task) Do() {
	defer close(t.doneC)
	t.f()
}

func New(opts ...Option) *GPool {
	cpuNum := runtime.NumCPU()
	g := &GPool{
		workerNum: cpuNum,
		taskChan:  make(chan Task, cpuNum*2),
		doneC:     make(chan struct{}),
	}

	for _, opt := range opts {
		opt(g)
	}

	g.wg.Add(g.workerNum)
	for i := 0; i < g.workerNum; i++ {
		go g.startOneWorker()
	}

	return g
}

//startNewCoreRoutine 启动一个 worker
func (p *GPool) startOneWorker() {
	defer func() {
		p.wg.Done()
		log.Println("gpool:", "worker exit")
	}()

	for {
		select {
		case t := <-p.taskChan:
			t.Do()
		case <-p.doneC:
			return
		}
	}
}

func (g *GPool) Do(taskFunc func()) <-chan struct{} {
	if g.closed.Val() {
		log.Println("已关闭")
		return nil
	}

	g.currTaskNum.Incr(1) //任务数加一

	doneC := make(chan struct{})

	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.taskChan) < cap(g.taskChan) {
		g.taskChan <- Task{
			f: func() {
				defer func() {
					g.currTaskNum.Incr(-1)
					close(doneC)
				}()
				taskFunc()
			},
		}
	} else {
		g.currWorkerNum.Incr(1)
		g.wg.Add(1)
		go func() {
			defer func() {
				g.wg.Done()
				g.currTaskNum.Incr(-1)
				close(doneC)
				g.currWorkerNum.Incr(-1)
			}()
			taskFunc()
		}()
	}

	return doneC
}

func (g *GPool) Close() error {
	g.closeOnce.Do(func() {
		g.closed.Set(true)
		close(g.doneC)
		g.wg.Wait()
	})

	return nil
}
