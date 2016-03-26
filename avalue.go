package lfchan

import (
	"runtime"
	"sync/atomic"
)

// AtomicValue is an atomic value using a spinlock
type AtomicValue struct {
	v    interface{}
	lock uint32
}

func (a *AtomicValue) Lock() {
	for !atomic.CompareAndSwapUint32(&a.lock, 0, 1) {
		runtime.Gosched()
	}
}

func (a *AtomicValue) Unlock() {
	atomic.StoreUint32(&a.lock, 0)
}

func (a *AtomicValue) Store(v interface{}) {
	a.Lock()
	a.v = v
	a.Unlock()
}

func (a *AtomicValue) Load() interface{} {
	a.Lock()
	v := a.v
	a.Unlock()
	return v
}

func (a *AtomicValue) CompareAndSwap(oval, nval interface{}) bool {
	var b bool
	a.Lock()
	if b = a.v == oval; b {
		a.v = nval
	}
	a.Unlock()
	return b
}

func (a *AtomicValue) Swap(nval interface{}) interface{} {
	var v interface{}
	a.Lock()
	v, a.v = a.v, nval
	a.Unlock()
	return v
}
