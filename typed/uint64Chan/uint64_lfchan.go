
package uint64Chan

import (
	"runtime"
	"sync/atomic"
	"time"
)

type innerChan struct {
	q   queue
	len int32
	cap int32
	die uint32
}

type Chan struct {
	*innerChan
}

func New() Chan {
	return NewSize(1)
}

func NewSize(sz int) Chan {
	if sz < 1 {
		panic("sz < 1")
	}
	ch := Chan{&innerChan{
		cap: int32(sz),
	}}
	ch.q.q = make([]qvalue, sz)
	return ch
}

const maxBackoff = time.Millisecond * 10

func (ch Chan) Send(v uint64, block bool) bool {
	if !block && ch.Len() == int(ch.cap) {
		return false
	}
	for chln, ccap := ch.Len(), int(ch.cap); !ch.Closed(); chln = ch.Len() {
		if chln == ccap {
			if !block {
				break
			}
			time.Sleep(time.Microsecond)
			runtime.Gosched()
			continue
		}
		if ch.q.store(v) {
			atomic.AddInt32(&ch.len, 1)
			return true
		}

		if !block {
			break
		}
		runtime.Gosched()
	}
	return false
}

func (ch Chan) Recv(block bool) (uint64, bool) {
	for chln := ch.Len(); chln > 0 || (block && !ch.Closed()); chln = ch.Len() {
		if chln == 0 {
			time.Sleep(time.Microsecond)
			runtime.Gosched()
			continue
		}
		//		i = atomic.AddUint32(&ch.recvIdx, 1)
		if v, ok := ch.q.get(); ok {
			atomic.AddInt32(&ch.len, -1)
			return v, true
		}

		if !block {
			break
		}

		//time.Sleep(time.Millisecond)
		runtime.Gosched()
	}
	return zeroValue, false
}

func (ch Chan) SendOnly() SendOnly { return SendOnly{ch} }

func (ch Chan) RecvOnly() RecvOnly { return RecvOnly{ch} }

func (ch Chan) Close() { atomic.StoreUint32(&ch.die, 1) }

func (ch Chan) Closed() bool { return atomic.LoadUint32(&ch.die) == 1 }

func (ch Chan) Cap() int { return int(ch.cap) }

func (ch Chan) Len() int {
	for {
		if ln := atomic.LoadInt32(&ch.len); ln > -1 {
			return int(ln)
		}
		runtime.Gosched()
	}
}

func SelectSend(block bool, v uint64, chans ...Sender) bool {
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

func SelectRecv(block bool, chans ...Receiver) (uint64, bool) {
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

type SendOnly struct{ c Chan }

func (so SendOnly) Send(v uint64, block bool) bool { return so.c.Send(v, block) }

type Sender interface {
	Send(v uint64, block bool) bool
}

type RecvOnly struct{ c Chan }

func (ro RecvOnly) Recv(block bool) (uint64, bool) { return ro.c.Recv(block) }

type Receiver interface {
	Recv(block bool) (uint64, bool)
}

var (
	_ Sender   = (*Chan)(nil)
	_ Sender   = (*SendOnly)(nil)
	_ Receiver = (*Chan)(nil)
	_ Receiver = (*RecvOnly)(nil)
)

var zeroValue uint64

type qvalue struct {
	v      uint64
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
		if i%100 == 0 {
			time.Sleep(time.Millisecond)
		}
		runtime.Gosched()
	}
}

func (a *queue) unlock() {
	atomic.StoreUint32(&a.sl, 0)
}

func (a *queue) store(newVal uint64) (b bool) {
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

func (a *queue) get() (v uint64, b bool) {
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
