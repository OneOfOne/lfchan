package lfchan

import (
	"runtime"
	"sync/atomic"
)

// TODO handle primitve types
var nilValue interface{}

// AtomicValue is an atomic value using a spinlock
type aValue struct {
	v  interface{}
	lk uint32
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

// CompareAndSwap atomically compares oldVal to the current value and replaces it with newVal if it's the same,
// returns true if it was successfully replaced.
func (a *aValue) CompareAndSwap(oldVal, newVal interface{}) bool {
	var b bool
	a.lock()
	if b = a.v == oldVal; b {
		a.v = newVal
	}
	a.unlock()
	return b
}

// Swap atomically swaps the current value with newVal and returns the old value.
func (a *aValue) Swap(newVal interface{}) interface{} {
	var v interface{}
	a.lock()
	v, a.v = a.v, newVal
	a.unlock()
	return v
}
