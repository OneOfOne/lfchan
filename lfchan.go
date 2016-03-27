package lfchan

import (
	"runtime"
	"sync/atomic"
	"time"
)

type innerChan struct {
	q       []AtomicValue
	sendIdx uint32
	recvIdx uint32
	slen    uint32
	rlen    uint32
	die     uint32
}

// Chan is a lock free channel that supports concurrent channel operations.
type Chan struct {
	*innerChan
}

// New returns a new channel with the buffer set to 1
func New() Chan {
	return NewSize(1)
}

// NewSize creates a buffered channel, with minimum length of 1
func NewSize(sz int) Chan {
	if sz < 1 {
		panic("sz < 1")
	}
	return Chan{&innerChan{
		q:       make([]AtomicValue, sz),
		sendIdx: ^uint32(0),
		recvIdx: ^uint32(0),
	}}
}

// Send adds v to the buffer of the channel and returns true, if the channel is closed it returns false
func (ch Chan) Send(v interface{}, block bool) bool {
	if !block && ch.Len() == ch.Cap() {
		return false
	}
	ncpu, ln, cnt := uint32(runtime.GOMAXPROCS(0)), uint32(len(ch.q)), uint32(0)
	for !ch.Closed() {
		if ch.Len() == ch.Cap() {
			if !block {
				return false
			}
			runtime.Gosched()
			continue
		}
		i := atomic.AddUint32(&ch.sendIdx, 1)
		if ch.q[i%ln].CompareAndSwap(nilValue, v) {
			atomic.AddUint32(&ch.slen, 1)
			return true
		}
		if block {
			if i%(ncpu*100) == 0 {
				pause(1)
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
func (ch Chan) Recv(block bool) (interface{}, bool) {
	if !block && ch.Len() == 0 { // fast path
		return nilValue, false
	}
	ncpu, ln, cnt := uint32(runtime.GOMAXPROCS(0)), uint32(len(ch.q)), uint32(0)
	for !ch.Closed() || ch.Len() > 0 {
		if ch.Len() == 0 {
			if !block {
				return nil, false
			}
			runtime.Gosched()
			continue
		}
		i := atomic.AddUint32(&ch.recvIdx, 1)
		if v := ch.q[i%ln].Swap(nilValue); v != nilValue {
			atomic.AddUint32(&ch.rlen, 1)
			return v, true
		}
		if block {
			if i%(ncpu*100) == 0 {
				pause(1)
			}
		} else if cnt++; cnt == ln {
			break
		}
		runtime.Gosched()
	}
	return nilValue, false
}

// SendOnly returns a send-only channel.
func (ch Chan) SendOnly() SendOnly { return SendOnly{ch} }

// RecvOnly returns a receive-only channel.
func (ch Chan) RecvOnly() RecvOnly { return RecvOnly{ch} }

// Close marks the channel as closed
func (ch Chan) Close() { atomic.StoreUint32(&ch.die, 1) }

// Closed returns true if the channel have been closed
func (ch Chan) Closed() bool { return atomic.LoadUint32(&ch.die) == 1 }

// Cap returns the size of the internal queue
func (ch Chan) Cap() int { return len(ch.q) }

// Len returns the number of elements queued
func (ch Chan) Len() int { return int(atomic.LoadUint32(&ch.slen) - atomic.LoadUint32(&ch.rlen)) }

// SelectSend sends v to the first available channel, if block is true, it blocks until a channel a accepts the value.
// returns false if all channels were full and block is false.
func SelectSend(block bool, v interface{}, chans ...Chan) bool {
	for {
		for i := range chans {
			if ok := chans[i].Send(v, false); ok {
				return ok
			}
		}
		if !block {
			return false
		}
		pause(1)
	}
}

// SelectRecv returns the first available value from chans, if block is true, it blocks until a value is available.
// returns nil, false if all channels were empty and block is false.
func SelectRecv(block bool, chans ...Chan) (interface{}, bool) {
	for {
		for i := range chans {
			if v, ok := chans[i].Recv(false); ok {
				return v, ok
			}
		}
		if !block {
			return nilValue, false
		}
		pause(1)
	}
}

// SendOnly is a send-only channel.
type SendOnly struct{ c Chan }

// Send is an alias for Chan.Send.
func (so SendOnly) Send(v interface{}, block bool) bool { return so.c.Send(v, block) }

// SelectSendOnly sends v to the first available channel, if block is true, it blocks until a channel a accepts the value.
// returns false if all channels were full and block is false.
func SelectSendOnly(block bool, v interface{}, chans ...SendOnly) bool {
	for {
		for i := range chans {
			if ok := chans[i].Send(v, false); ok {
				return ok
			}
		}
		if !block {
			return false
		}
		pause(1)
	}
}

// RecvOnly is a receive-only channel.
type RecvOnly struct{ c Chan }

// Recv is an alias for Chan.Recv.
func (ro RecvOnly) Recv(block bool) (interface{}, bool) { return ro.c.Recv(block) }

// SelectRecvOnly returns the first available value from chans, if block is true, it blocks until a value is available.
// returns nil, false if all channels were empty and block is false.
func SelectRecvOnly(block bool, chans ...RecvOnly) (interface{}, bool) {
	for {
		for i := range chans {
			if v, ok := chans[i].Recv(false); ok {
				return v, ok
			}
		}
		if !block {
			return nilValue, false
		}
		pause(1)
	}
}

func pause(p time.Duration) { time.Sleep(time.Millisecond * p) }
