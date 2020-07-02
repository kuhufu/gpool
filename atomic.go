package gpool

import "sync/atomic"

type AtomicInt32 struct {
	v int32
}

func (a *AtomicInt32) Incr(v int) {
	atomic.AddInt32(&a.v, int32(v))
}

func (a *AtomicInt32) Val() int32 {
	return atomic.LoadInt32(&a.v)
}

func (a *AtomicInt32) Set(v int) {
	atomic.StoreInt32(&a.v, int32(v))
}

type AtomicBool struct {
	v int32
}

func (a *AtomicBool) Val() bool {
	return atomic.LoadInt32(&a.v) != 0
}

func (a *AtomicBool) Set(v bool) {
	if v {
		atomic.StoreInt32(&a.v, 1)
		return
	}
	atomic.StoreInt32(&a.v, 0)
}
