package gpool

import (
	"log"
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	ByChan = iota
	ByCaller
)

type pool struct {
	coreNum  int32
	maxNum   int32
	curNum   int32
	taskChan chan TaskFunc
	done     chan int
	wg       *sync.WaitGroup
	mu       *sync.Mutex
	policy   int
	closed   bool
}

type TaskFunc func()

func New(coreNum, maxNum, bufLen int, policy int) *pool {
	if maxNum < coreNum {
		maxNum = coreNum
	}

	p := &pool{
		coreNum:  int32(coreNum),
		curNum:   int32(coreNum),
		maxNum:   int32(maxNum),
		taskChan: make(chan TaskFunc, bufLen),
		done:     make(chan int),
		wg:       &sync.WaitGroup{},
		mu:       &sync.Mutex{},
		policy:   policy,
	}
	p.start()
	return p
}

func NewFixed(num, bufLen int) *pool {
	return New(num, num, bufLen, ByChan)
}

func NewDefault() *pool {
	n := runtime.NumCPU()
	return New(n, n, n, ByChan)
}

func (p *pool) start() {
	for i := int32(0); i < p.coreNum; i++ {
		go func() {
		label:
			for {
				select {
				case <-p.done:
					break label
				case t := <-p.taskChan:
					t()
				}
			}
			log.Println("gpool:", "core goroutine exit")
		}()
	}
}

//Close 关闭后执行 Run 方法行为是不可预测的
func (p *pool) Run(taskFunc TaskFunc) {
	p.wg.Add(1)
	task := p.taskWrapper(taskFunc)

	p.mu.Lock()
	defer p.mu.Unlock()
	chanLen := len(p.taskChan)
	chanCap := cap(p.taskChan)
	switch {
	case chanLen < chanCap:
		p.taskChan <- task
	case chanLen == chanCap && p.curNum < p.maxNum:
		//缓冲区已满且当前Go程数量小于 maxNum，则启动新的Go程执行任务
		p.curNum++
		go func() {
			defer atomic.AddInt32(&p.curNum, -1)
			task()
		}()
	case chanLen == chanCap && p.curNum == p.maxNum:
		//调用方执行: 在调用 Run方法 的Go程中执行
		switch p.policy {
		case ByCaller:
			task()
		case ByChan:
			p.taskChan <- task
		}
	}
}

func (p *pool) taskWrapper(taskFunc TaskFunc) TaskFunc {
	return func() {
		defer p.wg.Done()
		taskFunc()
	}
}

func (p *pool) WaitAndClose() {
	p.Wait()
	p.Close()
}

func (p *pool) Wait() {
	p.wg.Wait()
}

// Close 调用之后就不应该再使用该pool
func (p *pool) Close() {
	close(p.done) //通知核心Go程退出
	p.taskChan = nil
	p.closed = true
}
