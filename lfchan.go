package lfchan

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type Chan struct {
	q   []*interface{}
	m   sync.Mutex
	idx uint32
	die uint32
}

func New() *Chan {
	return NewSize(1)
}

func NewSize(sz int) *Chan {
	return &Chan{
		q: make([]*interface{}, sz),
	}
}

func (ch *Chan) Send(v interface{}) {
	var (
		w  waiter
		ln = uint32(len(ch.q))
	)
	for atomic.LoadUint32(&ch.die) == 0 {
		p := unsafe.Pointer(&ch.q[atomic.LoadUint32(&ch.idx)%ln])
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(atomic.LoadPointer(&p)), nil, unsafe.Pointer(&v)) {
			return
		}
		w.wait()
	}
}

func (ch *Chan) Recv() interface{} {
	var (
		w  waiter
		ln = uint32(len(ch.q))
	)
	for atomic.LoadUint32(&ch.die) == 0 {
		p := unsafe.Pointer(&ch.q[atomic.AddUint32(&ch.idx, 1)%ln])
		if v := atomic.SwapPointer((*unsafe.Pointer)(atomic.LoadPointer(&p)), nil); v != nil {
			return *(*interface{})(v)
		}
		w.wait()
	}
	return nil
}

func (ch *Chan) Close() {
	atomic.StoreUint32(&ch.die, 1)
}

type waiter uint64

func (w *waiter) wait() {
	if *w++; *w%100 == 0 {
		time.Sleep(time.Millisecond)
	}
	runtime.Gosched()
}
