package lfchan

import (
	"runtime"
	"sync/atomic"
	"testing"
)

func TestLFChan(t *testing.T) {
	ch := New()
	go func() {
		for i := 0; i < 100; i++ {
			ch.Send(i)
		}
		ch.Close()
	}()
	var i int
	for v, ok := ch.Recv(); ok && v != nil; v, ok = ch.Recv() {
		if v.(int) != i {
			t.Fatalf("wanted %v, got %v", i, v)
		}
		i++
	}
}

func BenchmarkLFChan(b *testing.B) {
	var cnt uint64
	ch := NewSize(runtime.NumCPU())
	var total uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch.Send(atomic.AddUint64(&cnt, 1))
			v, _ := ch.Recv()
			atomic.AddUint64(&total, v.(uint64))
		}
	})
}

func BenchmarkChan(b *testing.B) {
	var cnt uint64
	ch := make(chan interface{}, runtime.NumCPU())
	var total uint64
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch <- atomic.AddUint64(&cnt, 1)
			atomic.AddUint64(&total, (<-ch).(uint64))
		}
	})
}
