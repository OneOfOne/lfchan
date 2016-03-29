package lfchan

import (
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

var ncpu = runtime.NumCPU()

type innerChan struct {
	q       [][]aValue
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
	n := ncpu
	if sz < n {
		n = sz
	}
	ch := Chan{&innerChan{
		q:       make([][]aValue, n),
		sendIdx: ^uint32(0),
		recvIdx: ^uint32(0),
	}}
	for i := range ch.q {
		ch.q[i] = make([]aValue, sz/n)
	}
	return ch
}

// Send adds v to the buffer of the channel and returns true, if the channel is closed it returns false
func (ch Chan) Send(v interface{}, block bool) bool {
	if !block && ch.Len() == ch.Cap() {
		return false
	}
	qln, ln, cnt := uint32(len(ch.q)), uint32(len(ch.q[0])), uint32(0)
	for !ch.Closed() {
		if ch.Len() == ch.Cap() {
			if !block {
				break
			}
			runtime.Gosched()
			continue
		}
		i := atomic.AddUint32(&ch.sendIdx, 1)
		if ch.q[i%qln][i%ln].CompareAndSwapIfNil(v) {
			atomic.AddUint32(&ch.slen, 1)
			return true
		}
		if block {
			if i%250 == 0 {
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
		return zeroValue, false
	}
	qln, ln, cnt := uint32(len(ch.q)), uint32(len(ch.q[0])), uint32(0)
	for chln := ch.Len(); !ch.Closed() || chln > 0; chln = ch.Len() {
		if chln == 0 {
			if !block {
				break
			}
			runtime.Gosched()
			continue
		}
		i := atomic.AddUint32(&ch.recvIdx, 1)
		if v, ok := ch.q[i%qln][i%ln].SwapWithNil(); ok {
			atomic.AddUint32(&ch.rlen, 1)
			return v, true
		}
		if block {
			if i%250 == 0 {
				pause(1)
			}
		} else if cnt++; cnt == ln {
			break
		}
		runtime.Gosched()
	}
	return zeroValue, false
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
func SelectSend(block bool, v interface{}, chans ...Sender) bool {
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
func SelectRecv(block bool, chans ...Receiver) (interface{}, bool) {
	for {
		for i := range chans {
			if v, ok := chans[i].Recv(false); ok {
				return v, ok
			}
		}
		if !block {
			return zeroValue, false
		}
		pause(1)
	}
}

// SendOnly is a send-only channel.
type SendOnly struct{ c Chan }

// Send is an alias for Chan.Send.
func (so SendOnly) Send(v interface{}, block bool) bool { return so.c.Send(v, block) }

// Sender represents a Chan or SendOnly.
type Sender interface {
	Send(v interface{}, block bool) bool
}

// RecvOnly is a receive-only channel.
type RecvOnly struct{ c Chan }

// Recv is an alias for Chan.Recv.
func (ro RecvOnly) Recv(block bool) (interface{}, bool) { return ro.c.Recv(block) }

// Receiver represents a Chan or RecvOnly.
type Receiver interface {
	Recv(block bool) (interface{}, bool)
}

func pause(p time.Duration) { time.Sleep(time.Millisecond * p) }

var (
	_ Sender   = (*Chan)(nil)
	_ Sender   = (*SendOnly)(nil)
	_ Receiver = (*Chan)(nil)
	_ Receiver = (*RecvOnly)(nil)
)

var zeroValue interface{}

type aValue struct {
	v interface{}
}

func (a *aValue) CompareAndSwapIfNil(newVal interface{}) bool {
	x := unsafe.Pointer(&a.v)
	return atomic.CompareAndSwapPointer((*unsafe.Pointer)(atomic.LoadPointer(&x)), nil, unsafe.Pointer(&newVal))
}

func (a *aValue) SwapWithNil() (interface{}, bool) {
	x := unsafe.Pointer(&a.v)
	if v := atomic.SwapPointer((*unsafe.Pointer)(atomic.LoadPointer(&x)), nil); v != nil {
		return *(*interface{})(v), true
	}
	return zeroValue, false
}
