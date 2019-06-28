package gpool

import (
	"log"
	"runtime"
	"sync"
	"sync/atomic"
)

//ByChan 等待taskChan缓冲区出现空位
//ByCaller 调用方执行
const (
	ByChan = policy(iota)
	ByCaller
)

type policy int

type pool struct {
	coreNum  int32           //核心Go程数量
	maxNum   int32           //最大Go程数量
	curNum   int32           //当前Go程数量
	taskChan chan TaskFunc   //任务通道
	done     chan int        //通知核心Go程退出
	wg       *sync.WaitGroup //用于等待所有任务完成
	mu       *sync.Mutex     //保证正确的选择策略
	policy   policy          //策略
	closed   bool            //pool是否已关闭
}

type TaskFunc func()

//New 灵活的创建 pool
func New(coreNum, maxNum, bufLen int, policy policy) *pool {
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

//NewFixed 创建 coreNum == maxNum 的 pool
func NewFixed(num, bufLen int) *pool {
	return New(num, num, bufLen, ByChan)
}

//NewDefault 根据当前机器 cpu 核心数创建 pool
func NewDefault() *pool {
	n := runtime.NumCPU()
	return New(n, n, n, ByChan)
}

//start 启动所有核心 Go 程
func (p *pool) start() {
	for i := int32(0); i < p.coreNum; i++ {
		go p.startNewCoreRoutine()
	}
}

//startNewCoreRoutine 启动一个核心 Go 程
func (p *pool) startNewCoreRoutine() {
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
}

//Run 添加任务
func (p *pool) Run(taskFunc TaskFunc) {
	p.wg.Add(1)
	task := p.taskWrapper(taskFunc)

	p.mu.Lock()
	defer p.mu.Unlock()
	chanLen := len(p.taskChan)
	chanCap := cap(p.taskChan)
	switch {
	case chanLen < chanCap:
		//缓冲区未满，进入缓冲区
		p.taskChan <- task
	case chanLen == chanCap && p.curNum < p.maxNum:
		//缓冲区已满且当前Go程数量小于 maxNum，则启动新的Go程执行任务
		p.curNum++
		go func() {
			defer atomic.AddInt32(&p.curNum, -1)
			task()
		}()
	case chanLen == chanCap && p.curNum == p.maxNum:
		//缓冲区已满且已达最大Go程数，这时候采用相关策略
		switch p.policy {
		case ByCaller:
			task()
		case ByChan:
			p.taskChan <- task
		}
	}
}

//taskWrapper 包装 TaskFunc
func (p *pool) taskWrapper(taskFunc TaskFunc) TaskFunc {
	return func() {
		defer p.wg.Done()
		taskFunc()
	}
}

//WaitAndClose 等待全部任务结束并关闭 pool
func (p *pool) WaitAndClose() {
	p.Wait()
	p.Close()
}

//Wait 等待全部任务结束
func (p *pool) Wait() {
	p.wg.Wait()
}

//Close 关闭 pool，调用之后不应再使用该 pool
func (p *pool) Close() {
	close(p.done) //通知核心Go程退出
	p.taskChan = nil
	p.closed = true
}
