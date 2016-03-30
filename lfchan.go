package lfchan

import (
	"runtime"
	"sync/atomic"
	"time"
)

type innerChan struct {
	q       []queue
	sendIdx uint32
	recvIdx uint32
	len     int32
	cap     int32
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

// NewSize creates a buffered channel, with minimum length of 1, may adjust the size to fit better in the internal queue.
func NewSize(sz int) Chan {
	if sz < 1 {
		panic("sz < 1")
	}
	n := runtime.NumCPU()
	if sz < n {
		n = sz
	}
	if sz%n != 0 {
		sz += (sz % n)
	}
	ch := Chan{&innerChan{
		q:       make([]queue, n),
		sendIdx: ^uint32(0),
		recvIdx: ^uint32(0),
		cap:     int32(sz),
	}}
	for i := range ch.q {
		ch.q[i].q = make([]qvalue, sz/n)
	}
	return ch
}

const maxBackoff = time.Millisecond * 10

// Send adds v to the buffer of the channel and returns true, if the channel is closed it returns false
func (ch Chan) Send(v interface{}, block bool) bool {
	if !block && ch.Len() == int(ch.cap) {
		return false
	}
	var (
		qln  = uint32(len(ch.q))
		ccap = int(ch.cap)
		//backoff = time.Millisecond
		i   uint32
		cnt int
	)
	for chln := ch.Len(); !ch.Closed(); chln = ch.Len() {
		if chln == ccap {
			goto CHECK
		}
		i = atomic.AddUint32(&ch.sendIdx, 1)
		if ch.q[i%qln].store(v) {
			//log.Println(i, i%qln, v)
			atomic.AddInt32(&ch.len, 1)
			return true
		}

	CHECK:
		if chln == ccap && !block {
			break
		} else if cnt++; cnt == ccap {
			if !block {
				break
			}
			time.Sleep(time.Millisecond)
		}
		runtime.Gosched()
	}
	return false
}

// Recv blocks until a value is available and returns v, true, or if the channel is closed and
// the buffer is empty, it will return nil, false
func (ch Chan) Recv(block bool) (interface{}, bool) {
	var (
		qln  = uint32(len(ch.q))
		ccap = int(ch.cap)
		//backoff = time.Millisecond
		i   uint32
		cnt int
	)
	for chln := ch.Len(); chln > 0 || (block && !ch.Closed()); chln = ch.Len() {
		if chln == 0 {
			goto CHECK
		}
		i = atomic.AddUint32(&ch.recvIdx, 1)
		if v, ok := ch.q[i%qln].get(); ok {
			atomic.AddInt32(&ch.len, -1)
			return v, true
		}

	CHECK:
		if chln == 0 && !block {
			break
		} else if cnt++; cnt == ccap {
			if !block {
				break
			}
			cnt = 0
			time.Sleep(time.Millisecond)
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
func (ch Chan) Cap() int { return int(ch.cap) }

// Len returns the number of elements queued
func (ch Chan) Len() int {
	for {
		if ln := atomic.LoadInt32(&ch.len); ln > -1 {
			return int(ln)
		}
		runtime.Gosched()
	}
}

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
		time.Sleep(time.Millisecond)
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
		time.Sleep(time.Millisecond)
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

var (
	_ Sender   = (*Chan)(nil)
	_ Sender   = (*SendOnly)(nil)
	_ Receiver = (*Chan)(nil)
	_ Receiver = (*RecvOnly)(nil)
)

var zeroValue interface{}

type qvalue struct {
	v      interface{}
	hasVal bool
}
type queue struct {
	q    []qvalue
	sl   uint32
	len  uint32
	rIdx uint32
	sIdx uint32
}

func (a *queue) lock() {
	for i := uint(0); !atomic.CompareAndSwapUint32(&a.sl, 0, 1); i++ {
		if i%1000 == 0 {
			time.Sleep(time.Millisecond)
		}
		runtime.Gosched()
	}
}

func (a *queue) unlock() {
	atomic.StoreUint32(&a.sl, 0)
}

func (a *queue) store(newVal interface{}) (b bool) {
	ln := uint32(len(a.q))
	a.lock()
	if a.len == ln {
		goto DIE
	}
	for i := uint32(0); i < ln; i++ {
		qv := &a.q[a.sIdx%ln]
		if b = !qv.hasVal; b {
			qv.v, qv.hasVal = newVal, true
			a.len++
			a.sIdx++
			break
		}
		a.sIdx++
	}
DIE:
	a.unlock()
	return
}

func (a *queue) get() (v interface{}, b bool) {
	a.lock()
	if a.len == 0 {
		goto DIE
	}
	for i, ln := uint32(0), uint32(len(a.q)); i < ln; i++ {
		qv := &a.q[a.rIdx%ln]
		if v, b = qv.v, qv.hasVal; b {
			qv.v, qv.hasVal = zeroValue, false
			a.rIdx++
			a.len--
			break
		}
		a.rIdx++
	}
DIE:
	a.unlock()
	return v, b
}
