package lfchan

import (
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

type Chan struct {
	q    []*interface{}
	p    unsafe.Pointer
	size int32
	die  int32
}

func New() *Chan {
	return NewSize(1)
}

func NewSize(sz int) *Chan {
	ch := &Chan{
		q: make([]*interface{}, sz),
	}
	ch.p = unsafe.Pointer(&ch.q[0])
	return ch
}

func (ch *Chan) Send(v interface{}) {
	ln, idx := uintptr(len(ch.q)*8), uintptr(0)
	for atomic.LoadInt32(&ch.die) == 0 {
		p := unsafe.Pointer(uintptr(ch.p) + idx)
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(atomic.LoadPointer(&p)), nil, unsafe.Pointer(&v)) {
			atomic.AddInt32(&ch.size, 1)
			return
		}
		if idx += 8; idx == ln {
			time.Sleep(time.Millisecond)
			idx = 0
		}
		runtime.Gosched()
	}
}

func (ch *Chan) Recv() interface{} {
	ln, idx := uintptr(len(ch.q)*8), uintptr(0)
	for atomic.LoadInt32(&ch.die) == 0 || atomic.LoadInt32(&ch.size) > 0 {
		p := unsafe.Pointer(uintptr(ch.p) + idx)
		if v := atomic.SwapPointer((*unsafe.Pointer)(atomic.LoadPointer(&p)), nil); v != nil {
			atomic.AddInt32(&ch.size, -1)
			return *(*interface{})(v)
		}
		if idx += 8; idx == ln {
			time.Sleep(time.Millisecond)
			idx = 0
		}
		runtime.Gosched()
	}
	return nil
}

func (ch *Chan) Close() {
	atomic.StoreInt32(&ch.die, 1)
}

type waiter uint64

func (w *waiter) wait() {
	if *w++; *w%10 == 0 {
		time.Sleep(time.Millisecond)
	}
	runtime.Gosched()
}
