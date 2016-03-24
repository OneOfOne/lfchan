package lfchan

import (
	"sync/atomic"
	"testing"
)

func Test(t *testing.T) {
	ch := New()
	ch.Send(1)
	go ch.Send(2)
	go ch.Send(3)
	go ch.Send(4)
	go ch.Send(5)
	t.Log(ch.Recv(), ch.Recv(), ch.Recv(), ch.Recv(), ch.Recv(), ch)
}

func BenchmarkLFChan(b *testing.B) {
	var cnt uint64
	ch := NewSize(4)
	var total uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch.Send(atomic.AddUint64(&cnt, 1))
			atomic.AddUint64(&total, ch.Recv().(uint64))
		}
	})
}

func BenchmarkChan(b *testing.B) {
	var cnt uint64
	ch := make(chan interface{}, 4)
	var total uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch <- atomic.AddUint64(&cnt, 1)
			atomic.AddUint64(&total, (<-ch).(uint64))
		}
	})
}
