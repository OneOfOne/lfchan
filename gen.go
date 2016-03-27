// +build ignore

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TODO: implement sync/atomic for primtive types

var files = [...]string{
	"$GOPATH/src/github.com/OneOfOne/lfchan/avalue.go",
	"$GOPATH/src/github.com/OneOfOne/lfchan/lfchan.go",
}

func main() {
	log.SetFlags(log.Lshortfile)
	log.SetPrefix("lfchan gen:")
	typ, typName := arg(1), arg(2)
	if typ == "" || typ == "*" {
		log.Fatal("must pass a type")
	}
	if typName == "" {
		if typ[0] == '*' {
			typName = typ[1:]
		} else {
			typName = typ
		}
	}
	if err := os.MkdirAll(typName, 0755); err != nil {
		log.Fatalf("os.MkdirAll(%q, 0755): %v", typName, err)
	}
	repl := strings.NewReplacer("interface{}", typ, "package lfchan", "package "+filepath.Base(typName))
	log.Printf("creating %s", typName)
	for _, fn := range files {
		f, err := os.Open(os.ExpandEnv(fn))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		of, err := os.Create(filepath.Join(typName, filepath.Base(fn)))
		if err != nil {
			log.Fatal(err)
		}
		b := bufio.NewScanner(f)
		for b.Scan() {
			ln := repl.Replace(b.Text())
			fmt.Fprintln(of, ln)
		}
		of.Close()
	}
	if err := ioutil.WriteFile(filepath.Join(typName, "chan_test.go"), []byte(fmt.Sprintf(testCode, typName, typ, typ)), 0644); err != nil {
		log.Fatal(err)
	}
	out, err := exec.Command("go", "test", "./"+typName+"/...").CombinedOutput()
	if err != nil {
		log.Fatalf("error running tests: %s %v", out, err)
	}
}

func arg(idx int) string {
	if len(os.Args) <= idx {
		return ""
	}
	return os.Args[idx]
}

const testCode = `package %s

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

func Test(t *testing.T) {
	var (
		iv reflect.Value
		typ  = reflect.TypeOf(nilValue)
		zero = reflect.Zero(typ).Interface().(%s)
	)
	for {
		var ok bool
		if iv, ok = quick.Value(typ, rand.New(rand.NewSource(43))); !ok {
			t.SkipNow()
		}
		if iv.Kind() == reflect.Ptr && iv.IsNil() {
			continue
		}
		break
	}
	rv, ok := iv.Interface().(%s)
	if !ok {
		t.SkipNow()
	}
	ch := New()
	go func() {
		for i := 0; i < 100; i++ {
			ch.Send(rv, true)
		}
		ch.Send(zero, true)
		ch.Close()
	}()
	for i := 0; i < 100; i++ {
		v, ok := ch.Recv(true)
		if !ok {
			t.Fatal("!ok")
		}
		if v != rv {
			t.Fatalf("wanted %%v, got %%v", rv, v)
		}
	}
	if v, ok := ch.Recv(true); !ok || v != zero {
		t.Fatal("!ok || v != zero")
	}
}
`
