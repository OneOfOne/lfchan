# lfchan [![GoDoc](http://godoc.org/github.com/OneOfOne/lfchan?status.svg)](http://godoc.org/github.com/OneOfOne/lfchan) [![Build Status](https://travis-ci.org/OneOfOne/lfchan.svg?branch=master)](https://travis-ci.org/OneOfOne/lfchan)
--

A scalable lock-free channel.

- Supports graceful closing.
- Supports blocking and non-blocking operations.
- Supports select.
- Scales with the number of cores.

## Install

	go get github.com/OneOfOne/lfchan

## Usage

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

## Generate a typed channel

### Generate the package:
``` bash
go run "$GOPATH/src/github.com/OneOfOne/lfchan/gen.go" type [pkgName]

# example
go run "$GOPATH/src/github.com/OneOfOne/lfchan/gen.go" *node nodeChan

```

### Use it in your code:

```go
import (
	"fmt"

	"github.com/YOU/nodeChan"
)

func main() {
	ch := nodeChan.New() // or
	// ch := nodeChan.NewSize(10) // buffered channel
	go ch.Send(&node{1}, true)
	fmt.Printf("%#+v", ch.Recv(true))
}

```

**Warning** currently, typed channels can't handle zero value primitve types correctly,
for example it can't handle sending 0 on an int channel.

## Benchmark
```bash
➜ go test -bench=. -benchmem -cpu 1,4,8,32 -benchtime 3s

# ch := NewSize(100)
BenchmarkLFChan         20000000               292 ns/op               8 B/op          1 allocs/op
BenchmarkLFChan-4       20000000               202 ns/op               8 B/op          1 allocs/op
BenchmarkLFChan-8       30000000               161 ns/op               8 B/op          1 allocs/op
BenchmarkLFChan-32      20000000               215 ns/op               8 B/op          1 allocs/op

# ch := make(chan interface{}, 100)
BenchmarkChan           10000000               371 ns/op               8 B/op          1 allocs/op
BenchmarkChan-4         10000000               378 ns/op               8 B/op          1 allocs/op
BenchmarkChan-8         10000000               506 ns/op               8 B/op          1 allocs/op
BenchmarkChan-32        10000000               513 ns/op               8 B/op          1 allocs/op

PASS
ok      github.com/OneOfOne/lfchan      39.461s
```

## FAQ
### Why are you using `runtime.Gosched`?

- Sadly, it is the only clean way to release the scheduler in a tight loop, Go doesn't provide any other way to yield,
`time.Sleep` causes random allocations at times.
[`sync/atomic.Value`](https://github.com/golang/go/blob/master/src/sync/atomic/value.go#L57) has access to internal
funcs which can control the scheduler, however user code can't do that.

### Isn't using a spinlock bad for the CPU?

- Yes and no, thanks to `runtime.Gosched` usage in my tight loops, lfchan actually uses less CPU than std chans.

## License

Apache v2.0 (see [LICENSE](https://github.com/OneOfOne/lfchan/blob/master/LICENSE) file).

Copyright 2016-2016 Ahmed <[OneOfOne](https://github.com/OneOfOne/)> W.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
