package lfchan

import (
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

const ptrSize = unsafe.Sizeof((*interface{})(nil))

// Chan is a lock free channel that supports concurrent channel operations.
type Chan struct {
	q    []*interface{}
	p    unsafe.Pointer
	size int32
	die  int32
}

// New returns a new channel with the buffer set to 1
func New() *Chan {
	return NewSize(1)
}

// New creates a buffered channel, with minimum length of 1
func NewSize(sz int) *Chan {
	if sz < 1 {
		sz = 1
	}
	ch := &Chan{
		q: make([]*interface{}, sz),
	}
	ch.p = unsafe.Pointer(&ch.q[0])
	return ch
}

// Send adds v to the buffer of the channel and returns true, if the channel is closed it returns false
func (ch *Chan) Send(v interface{}) bool {
	ln, idx := uintptr(len(ch.q))*ptrSize, uintptr(0)
	for atomic.LoadInt32(&ch.die) == 0 {
		p := unsafe.Pointer(uintptr(ch.p) + idx)
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(atomic.LoadPointer(&p)), nil, unsafe.Pointer(&v)) {
			atomic.AddInt32(&ch.size, 1)
			return true
		}
		if idx += ptrSize; idx == ln {
			time.Sleep(time.Millisecond)
			idx = 0
		}
		runtime.Gosched()
	}
	return false
}

// Recv blocks until a value is available and returns v, true, or if the channel is closed and
// the buffer is empty, it will return nil, false
func (ch *Chan) Recv() (interface{}, bool) {
	ln, idx := uintptr(len(ch.q))*ptrSize, uintptr(0)
	for atomic.LoadInt32(&ch.die) == 0 || atomic.LoadInt32(&ch.size) > 0 {
		p := unsafe.Pointer(uintptr(ch.p) + idx)
		if v := atomic.SwapPointer((*unsafe.Pointer)(atomic.LoadPointer(&p)), nil); v != nil {
			atomic.AddInt32(&ch.size, -1)
			return *(*interface{})(v), true
		}
		if idx += ptrSize; idx == ln {
			time.Sleep(time.Millisecond)
			idx = 0
		}
		runtime.Gosched()
	}
	return nil, false
}

//Close marks the channel as closed
func (ch *Chan) Close() {
	atomic.StoreInt32(&ch.die, 1)
}
