package lfchan

import (
	"runtime"
	"sync/atomic"
)

// Chan is a lock free channel that supports concurrent channel operations.
type Chan struct {
	q       []AtomicValue
	sendIdx uint32
	recvIdx uint32
	len     int32
	die     int32
}

// New returns a new channel with the buffer set to 1
func New() *Chan {
	return NewSize(1)
}

// NewSize creates a buffered channel, with minimum length of 1
func NewSize(sz int) *Chan {
	if sz < 1 {
		panic("sz < 1")
	}
	ch := &Chan{
		q:       make([]AtomicValue, sz),
		sendIdx: ^uint32(0),
		recvIdx: ^uint32(0),
	}
	return ch
}

// Send adds v to the buffer of the channel and returns true, if the channel is closed it returns false
func (ch *Chan) Send(v interface{}, block bool) bool {
	ncpu, ln, cnt := uint32(runtime.NumCPU()), uint32(len(ch.q)), uint32(0)
	for !ch.Closed() {
		i := atomic.AddUint32(&ch.sendIdx, 1)
		if ch.q[i%ln].CompareAndSwap(nil, v) {
			atomic.AddInt32(&ch.len, 1)
			return true
		}
		if block {
			if i%(ncpu*100) == 0 {
				for i := uint32(0); i < ncpu; i++ {
					runtime.Gosched()
				}
			}
		} else if cnt++; cnt == ln {
			break
		}
		runtime.Gosched()
	}
	return false
}

// Recv blocks until a value is available and returns v, true, or if the channel is closed and
// the buffer is empty, it will return nil, false
func (ch *Chan) Recv(block bool) (interface{}, bool) {
	ncpu, ln, cnt := uint32(runtime.NumCPU()), uint32(len(ch.q)), uint32(0)
	if !block && ch.Len() == 0 { // fast path
		goto EXIT
	}
	for !ch.Closed() || ch.Len() > 0 {
		i := atomic.AddUint32(&ch.recvIdx, 1)
		if v := ch.q[i%ln].Swap(nil); v != nil {
			atomic.AddInt32(&ch.len, -1)
			return v, true
		}
		if block {
			if i%(ncpu*100) == 0 {
				for i := uint32(0); i < ncpu; i++ {
					runtime.Gosched()
				}
			}
		} else if cnt++; cnt == ln {
			break
		}
		runtime.Gosched()
	}
EXIT:
	return nil, false
}

// Close marks the channel as closed
func (ch *Chan) Close() { atomic.StoreInt32(&ch.die, 1) }

// Closed returns true if the channel have been closed
func (ch *Chan) Closed() bool { return atomic.LoadInt32(&ch.die) == 1 }

// Cap returns the size of the internal queue
func (ch *Chan) Cap() int { return len(ch.q) }

// Len returns the number of elements queued
func (ch *Chan) Len() int { return int(atomic.LoadInt32(&ch.len)) }

// Select returns the first available value
func Select(block bool, chans ...*Chan) (interface{}, bool) {
	for {
		for _, ch := range chans {
			if v, ok := ch.Recv(false); ok {
				return v, ok
			}
		}
		if !block {
			return nil, false
		}
		runtime.Gosched()
	}
}
