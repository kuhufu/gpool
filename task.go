package gpool

import (
	"sync"
)

type RunTaskFunc func()
type CallTaskFunc func() interface{}

func NewRunTask(fn func(), wg *sync.WaitGroup) task {
	return &RunTask{
		fn:     fn,
		poolWg: wg,
		done:   make(chan interface{}),
		err:    nil,
	}
}

func NewCallTask(fn func() interface{}, wg *sync.WaitGroup) task {
	return &CallTask{
		fn:     fn,
		poolWg: wg,
		done:   make(chan interface{}, 1),
		err:    nil,
	}
}

type item interface {
	run()
	discard()
	setError(err error)
	closeDone()
}

type Result interface {
	Error() error
	Done() <-chan interface{}
}

type task interface {
	item
	Result
}

type RunTask struct {
	fn     func()
	poolWg *sync.WaitGroup
	done   chan interface{}
	err    error
}

func (t *RunTask) discard() {
	t.err = ErrTaskDiscard
	t.poolWg.Done()
	close(t.done)
}

func (t *RunTask) run() {
	t.fn()
	t.poolWg.Done()
	close(t.done)
}

func (t *RunTask) setError(err error) {
	t.err = err
}

func (t *RunTask) closeDone() {
	close(t.done)
}

func (t *RunTask) Error() error {
	return t.err
}

func (t *RunTask) Done() <-chan interface{} {
	return t.done
}

type CallTask struct {
	fn     func() interface{}
	poolWg *sync.WaitGroup
	done   chan interface{}
	err    error
}

func (t *CallTask) discard() {
	t.err = ErrTaskDiscard
	t.poolWg.Done()
	close(t.done)
}

func (t *CallTask) run() {
	t.done <- t.fn()
	t.poolWg.Done()
	close(t.done)
}

func (t *CallTask) setError(err error) {
	t.err = err
}

func (t *CallTask) closeDone() {
	t.fn()
	t.poolWg.Done()
	close(t.done)
}

func (t *CallTask) Error() error {
	return t.err
}

func (t *CallTask) Done() <-chan interface{} {
	return t.done
}
