package lfchan

import (
	"flag"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var timeCpu = flag.Bool("timeCpu", false, "internal")

func init() {
	log.SetFlags(log.Lshortfile)
}

func TestLFChan(t *testing.T) {
	ch := New()
	var want, got int
	for i := 0; i < 100; i++ {
		go ch.Send(i, true)
		want += i
	}
	for i := 0; i < 100; i++ {
		v, ok := ch.Recv(true)
		if !ok {
			t.Fatal("!ok")
		}
		got += v.(int)
	}
	if want != got {
		t.Fatalf("wanted %v, got %v", want, got)
	}
}

func TestSelect(t *testing.T) {
	var chs [100]Chan
	for i := range chs {
		chs[i] = New()
	}
	for i := range chs {
		if !SelectSend(false, i, chs[:]...) {
			t.Fatalf("couldn't send %d", i)
		}
	}
	for i := range chs {
		if v, ok := SelectRecv(false, chs[:]...); !ok || v != i {
			t.Fatalf("wanted %v, got %v", i, v)
		}
	}
}

func TestFIFO(t *testing.T) {
	const N = 10000
	ch := NewSize(100)

	go func() {
		for i := 0; i < N; i++ {
			ch.Send(i, true)
		}
		ch.Close()
	}()
	for i := 0; i < N; i++ {
		v, ok := ch.Recv(true)
		if !ok {
			t.Fatal("!ok")
		}
		if v.(int) != i {
			t.Fatalf("wanted %d, got %d", i, v)
		}
	}
}

// needs to run with -count 100 to trigger
func TestLen(t *testing.T) {
	const N = 100000
	ch := NewSize(100)
	var wg sync.WaitGroup
	wg.Add(N * 2)
	go func() {
		for i := 0; i < N; i++ {
			go func() {
				defer wg.Done()
				v, ok := ch.Recv(true)
				if !ok {
					t.Fatal("!ok")
				}
				if ln := ch.Len(); ln < 0 {
					t.Fatalf("ch.Len() == %d: %v", ln, v)
				}
			}()
		}
	}()

	go func() {
		for i := 0; i < N; i++ {
			go func(i int) {
				defer wg.Done()
				ch.Send(i, true)
			}(i)
		}
	}()

	wg.Wait()
	ch.Close()
}

func TestLFCPU(t *testing.T) {
	if !*timeCpu {
		t.SkipNow()
	}
	ch := NewSize(1)
	go func() {
		for i := 0; i < 10; i++ {
			ch.Send(i, true)
			time.Sleep(time.Second)
		}
		ch.Close()
	}()
	for v, ok := ch.Recv(true); ok && v != nil; v, ok = ch.Recv(true) {
		t.Log(v)
	}
}

func TestStdCPU(t *testing.T) {
	if !*timeCpu {
		t.SkipNow()
	}
	ch := make(chan interface{}, 1)
	go func() {
		for i := 0; i < 10; i++ {
			ch <- i
			time.Sleep(time.Second)
		}
		close(ch)
	}()
	for v := range ch {
		t.Log(v)
	}
}

func BenchmarkLFChan(b *testing.B) {
	var cnt uint64
	ch := NewSize(100)
	var total uint64
	b.RunParallel(func(pb *testing.PB) {
		var wg sync.WaitGroup
		for pb.Next() {
			ch.Send(atomic.AddUint64(&cnt, 1), true)
			wg.Add(1)
			go func() {
				v, _ := ch.Recv(true)
				atomic.AddUint64(&total, v.(uint64))
				wg.Done()
			}()
		}
		wg.Wait()
	})
}

func BenchmarkChan(b *testing.B) {
	var cnt uint64
	ch := make(chan interface{}, 100)
	var total uint64
	b.RunParallel(func(pb *testing.PB) {
		var wg sync.WaitGroup
		for pb.Next() {
			wg.Add(1)
			ch <- atomic.AddUint64(&cnt, 1)
			go func() {
				atomic.AddUint64(&total, (<-ch).(uint64))
				wg.Done()
			}()
		}
		wg.Wait()
	})
}
