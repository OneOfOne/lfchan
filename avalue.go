package lfchan

import (
	"runtime"
	"sync/atomic"
)

// TODO handle primitve types
var nilValue interface{}

// AtomicValue is an atomic value using a spinlock
type aValue struct {
	v      interface{}
	lk     uint32
	hasVal bool
}

func (a *aValue) lock() {
	for !atomic.CompareAndSwapUint32(&a.lk, 0, 1) {
		runtime.Gosched()
	}
}

func (a *aValue) unlock() { atomic.StoreUint32(&a.lk, 0) }

// Store atomically sets the current value.
func (a *aValue) Store(v interface{}) {
	a.lock()
	a.v = v
	a.unlock()
}

// Load atomically returns the current value.
func (a *aValue) Load() interface{} {
	a.lock()
	v := a.v
	a.unlock()
	return v
}

func (a *aValue) CompareAndSwapIfNil(newVal interface{}) bool {
	var b bool
	a.lock()
	if b = !a.hasVal; b {
		a.v, a.hasVal = newVal, true
	}
	a.unlock()
	return b
}
func (a *aValue) SwapWithNil() (interface{}, bool) {
	var (
		v  interface{}
		ok bool
	)
	a.lock()
	v, a.v, ok, a.hasVal = a.v, nilValue, a.hasVal, false
	a.unlock()
	return v, ok
}
