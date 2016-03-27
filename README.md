# lfchan [![GoDoc](http://godoc.org/github.com/OneOfOne/lfchan?status.svg)](http://godoc.org/github.com/OneOfOne/lfchan) [![Build Status](https://travis-ci.org/OneOfOne/lfchan.svg?branch=master)](https://travis-ci.org/OneOfOne/lfchan)
--

A scalable lock-free channel.

- Supports graceful closing.
- Supports blocking and non-blocking operations.
- Supports select.
- Scales with the number of cores.

# Install

	go get github.com/OneOfOne/lfchan

# Usage

```go
import (
	"fmt"

	"github.com/OneOfOne/lfchan"
)

func main() {
	ch := lfchan.New() // or
	// ch := lfchan.NewSize(10) // buffered channel
	go ch.Send("hello", true)
	fmt.Printf("%s world", ch.Recv(true).(string))
}
```

# Generate a typed channel

## Generate the package:
``` bash
go run "$GOPATH/src/github.com/OneOfOne/lfchan/gen.go" type [pkgName]

# example
go run "$GOPATH/src/github.com/OneOfOne/lfchan/gen.go" *Node nodeChan

```

## Use it in your code:

```go
import (
	"fmt"

	"github.com/YOU/nodeChan"
)

func main() {
	ch := nodeChan.New() // or
	// ch := nodeChan.NewSize(10) // buffered channel
	go ch.Send(&Node{1}, true)
	for n, ok := ch.Recv(true); ok; n, ok = ch.Recv(true) {
		// handle n
	}
}

```

**Warning** currently, typed channels can't handle zero value primitve types correctly,
for example it can't handle sending 0 on an int channel.

# Known issues

- <strike>Under high concurrency, ch.Len() can return -1 (issue [#2](https://github.com/OneOfOne/lfchan/issues/2))</strike>
Fixed by commit [bdddd90](https://github.com/OneOfOne/lfchan/commit/bdddd904676fc8368064cc2eb21efaa4384cd2db).

# Benchmark
```bash
# ch := NewSize(100)
BenchmarkLFChan         50000000               344 ns/op               8 B/op          1 allocs/op
BenchmarkLFChan-4       50000000               275 ns/op               8 B/op          1 allocs/op
BenchmarkLFChan-8       50000000               298 ns/op               8 B/op          1 allocs/op

# ch := make(chan interface{}, 100)
BenchmarkChan           50000000               382 ns/op               8 B/op          1 allocs/op
BenchmarkChan-4         50000000               389 ns/op               8 B/op          1 allocs/op
BenchmarkChan-8         30000000               500 ns/op               8 B/op          1 allocs/op
PASS
ok      github.com/OneOfOne/lfchan      128.469s
```

# FAQ

## Why are you using `runtime.Gosched`?

- Sadly, it is the only clean way to release the scheduler in a tight loop, Go doesn't provide any other way to yield,
`time.Sleep` causes random allocations at times.
[`sync/atomic.Value`](https://github.com/golang/go/blob/master/src/sync/atomic/value.go#L57) has access to internal
funcs which can control the scheduler, however user code can't do that.

## Isn't using a spinlock bad for the CPU?

- Yes and no, using the spinlock and few sleeps in the code makes it very efficient even under idle conditions.

# License

This project is released under the Apache v2. licence. See [LICENCE](LICENCE) for more details.
