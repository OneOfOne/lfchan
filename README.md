# lfchan [![GoDoc](http://godoc.org/github.com/OneOfOne/lfchan?status.svg)](http://godoc.org/github.com/OneOfOne/lfchan) [![Build Status](https://travis-ci.org/OneOfOne/lfchan.svg?branch=master)](https://travis-ci.org/OneOfOne/lfchan)
--

Extremely simple lock-free blocking channel implementation.

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
	go ch.Send("hello")
	fmt.Printf("%s world", ch.Recv().(string))
}
```

## Benchmark
```bash
➜ go test -bench=. -benchmem -cpu 1,4,8,32 -benchtime 3s

# 	ch := NewSize(runtime.NumCPU())
BenchmarkLFChan         30000000               168 ns/op              40 B/op          4 allocs/op
BenchmarkLFChan-4       30000000               175 ns/op              45 B/op          4 allocs/op
BenchmarkLFChan-8       20000000               205 ns/op              45 B/op          4 allocs/op
BenchmarkLFChan-32      20000000               201 ns/op              45 B/op          4 allocs/op

# ch := make(chan interface{}, runtime.NumCPU())
BenchmarkChan           50000000               115 ns/op               8 B/op          1 allocs/op
BenchmarkChan-4         20000000               261 ns/op               8 B/op          1 allocs/op
BenchmarkChan-8         20000000               331 ns/op               8 B/op          1 allocs/op
BenchmarkChan-32        10000000               532 ns/op               8 B/op          1 allocs/op

PASS
ok      github.com/OneOfOne/lfchan      51.663s
```

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
