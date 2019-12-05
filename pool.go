package gpool

import (
	"errors"
	"log"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
)

type policy byte

//ByChan 等待taskChan缓冲区出现空位
//ByCaller 调用方执行
//直接丢弃
const (
	ByChan = policy(iota)
	ByCaller
	Discard
)

var (
	ErrTaskDiscard       = errors.New("task has been discarded")
	ErrPoolAlreadyClosed = errors.New("pool already closed")
)

type Pool struct {
	coreNum            int32           //核心Go程数量
	maxNum             int32           //最大Go程数量
	curNum             int32           //当前Go程数量
	taskChan           chan item       //任务通道
	shutdownDirectly   chan struct{}   //通知核心Go程退出
	shutdownGracefully chan struct{}   //通知核心Go程退出
	wg                 *sync.WaitGroup //用于等待所有任务完成
	mu                 *sync.Mutex     //保证正确的选择策略
	policy             policy          //策略
	closed             bool            //pool是否已关闭
	taskNum            int32           //当前任务数
}

//New 灵活的创建 Pool
func New(coreNum, maxNum, bufLen int, policy policy) *Pool {
	if maxNum < coreNum {
		maxNum = coreNum
	}

	p := &Pool{
		coreNum:            int32(coreNum),
		curNum:             int32(coreNum),
		maxNum:             int32(maxNum),
		taskChan:           make(chan item, bufLen),
		shutdownDirectly:   make(chan struct{}),
		shutdownGracefully: make(chan struct{}),
		wg:                 &sync.WaitGroup{},
		mu:                 &sync.Mutex{},
		policy:             policy,
	}
	return p
}

//NewFixed 创建 coreNum == maxNum 的 Pool
func NewFixed(num, bufLen int) *Pool {
	return New(num, num, bufLen, ByChan)
}

//NewDefault 根据当前机器 cpu 核心数创建 Pool
func NewDefault() *Pool {
	n := runtime.NumCPU()
	return New(n, math.MaxInt32, n, ByChan)
}

//Start 启动所有核心 Go 程
func (p *Pool) Start() {
	for i := int32(0); i < p.coreNum; i++ {
		go p.startNewCoreRoutine()
	}
}

//startNewCoreRoutine 启动一个核心 Go 程
func (p *Pool) startNewCoreRoutine() {
label:
	for {
		select {
		case <-p.shutdownDirectly:
			break label
		case t := <-p.taskChan:
			t.run()
			atomic.AddInt32(&p.taskNum, -1)
		case <-p.shutdownGracefully:
			select {
			case t := <-p.taskChan:
				t.run()
			default:
				if p.isEmpty() {
					break
				}
			}
		}
	}
	log.Println("gpool:", "core goroutine exit")
}

func (p *Pool) newRunTask(fn RunTaskFunc) task {
	return &RunTask{
		fn: func() {
			defer p.wg.Done()
			fn()
		},
		done: make(chan interface{}),
	}
}

func (p *Pool) newCallTask(fn CallTaskFunc) task {
	return &CallTask{
		fn: func() interface{} {
			defer p.wg.Done()
			return fn()
		},
		done: make(chan interface{}),
	}
}

func (p *Pool) Run(fn RunTaskFunc) (result Result) {
	return p.execute(p.newRunTask(fn))
}

func (p *Pool) Call(fn CallTaskFunc) (result Result) {
	return p.execute(p.newCallTask(fn))
}

//Run 添加任务
func (p *Pool) execute(taskItem task) (result Result) {
	p.wg.Add(1)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		taskItem.abortWithErr(ErrPoolAlreadyClosed)
		atomic.AddInt32(&p.taskNum, -1)
		return taskItem
	}

	chanLen := len(p.taskChan)
	chanCap := cap(p.taskChan)
	switch {
	case chanLen < chanCap:
		//缓冲区未满，进入缓冲区
		p.taskChan <- taskItem
	case p.curNum < p.maxNum && chanLen == chanCap:
		//缓冲区已满且当前Go程数量小于 maxNum，则启动新的Go程执行任务
		p.curNum++
		go func() {
			taskItem.run()
			atomic.AddInt32(&p.curNum, -1) //Go程计数减一
		}()
	case p.curNum == p.maxNum && chanLen == chanCap:
		//缓冲区已满且已达最大Go程数，这时候采用相关策略
		switch p.policy {
		case ByCaller:
			taskItem.run()
		case ByChan:
			p.taskChan <- taskItem
		case Discard:
			taskItem.discard()
		}
		return taskItem
	}
	return taskItem
}

//Wait 等待全部任务结束
func (p *Pool) Wait() {
	p.wg.Wait()
}

// ShutdownDirectly 不再接受新的task，丢弃排队中的task
func (p *Pool) ShutdownDirectly() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.shutdownDirectly) //通知核心Go程退出
	p.mu.Unlock()

	go p.Empty()
}

//ShutdownGracefully 不再接受新的task，会继续处理已提交的task
func (p *Pool) ShutdownGracefully() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.shutdownGracefully)
	log.Println("shutdown gracefully")
}

// Closed 是否已关闭
func (p *Pool) Closed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.closed
}

func (p *Pool) isEmpty() bool {
	return atomic.LoadInt32(&p.taskNum) == 0
}

// Empty 清空任务队列
func (p *Pool) Empty() {
	for {
		select {
		case t := <-p.taskChan:
			t.discard()
		default:
			if p.isEmpty() {
				return
			}
		}
	}
}
