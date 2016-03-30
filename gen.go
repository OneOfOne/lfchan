// +build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// TODO: implement sync/atomic for primtive types
// TODO: clean this ugly mess up

var (
	files = [...]string{
		"$GOPATH/src/github.com/OneOfOne/lfchan/lfchan.go",
	}

	cwd = func() string { v, _ := os.Getwd(); return filepath.Base(v) }()

	isMain = flag.Bool("main", false, `should the files be under package main, only used if -o is set to ".".
	if not set then they will be under currentDirName.`)
	genTest = flag.Bool("t", false, "generate a test")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() == 0 || flag.NArg() > 2 {
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(log.Lshortfile)
	log.SetPrefix("lfchan: ")
}

func main() {
	var (
		typ, typName = flag.Arg(0), flag.Arg(1)
		importPkg    string
		pkgName      string
		fnamePre     string
		replaces     []string
		mkdir        = true
		replQ        = strings.NewReplacer("*", "", "[]", "")
	)
	if typ == "" || typ == "*" {
		log.Fatal("must pass a type")
	}
	if idx := strings.LastIndex(typ, "."); idx > 0 {
		fnamePre = strings.ToLower(typ[idx+1:]) + "_"
		importPkg = replQ.Replace(typ[:idx])
		if sidx := strings.LastIndex(typ, "/"); idx > 0 {
			typ = typ[sidx+1:]
		} else {
			typ = typ[idx:]
		}
		if typName != "." || importPkg != cwd {
			replaces = append(replaces, "import (", fmt.Sprintf("import (\n\t%q\n", importPkg))
		}
	} else {
		fnamePre = replQ.Replace(strings.ToLower(typ)) + "_"
	}

	if typName == "" {
		if typ[0] == '*' {
			typName = typ[1:]
		} else {
			typName = typ
		}
		if idx := strings.LastIndex(typName, "."); idx > 0 {
			typName = typName[idx+1:]
		}
		typName = strings.ToLower(typName) + "Chan"
		pkgName = typName
	} else if typName == "." {
		mkdir = false
		typName = cwd
		pkgName = "./"
		name := filepath.Base(typ)
		if len(name) > 0 {
			if idx := strings.LastIndex(name, "."); idx > 0 {
				name = name[idx+1:]
			}
			name = replQ.Replace(name)
			r, sz := utf8.DecodeRuneInString(name)
			name = strings.ToUpper(string(r)) + name[sz:]
		}
		replaces = append(replaces, " New(", fmt.Sprintf(" new%sChan(", name))
		replaces = append(replaces, " NewSize(", fmt.Sprintf(" newSize%sChan(", name))
	} else {
		pkgName = typName
	}

	//	log.Fatalf("%q %q %q %q", typ, typName, importPkg, fnamePre)

	if mkdir {
		if err := os.MkdirAll(typName, 0755); err != nil {
			log.Fatalf("os.MkdirAll(%q, 0755): %v", typName, err)
		}
	}

	replaces = append(replaces, "interface{}", typ)
	if *isMain {
		replaces = append(replaces, "package lfchan", "package main")
	} else {
		replaces = append(replaces, "package lfchan", "package "+replQ.Replace(filepath.Base(typName)))
	}

	//log.Fatalf("%q", replaces)

	repl := strings.NewReplacer(replaces...)
	log.Printf("creating %s", typName)
	for _, fn := range files {
		f, err := os.Open(os.ExpandEnv(fn))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		of, err := os.Create(filepath.Join(pkgName, fnamePre+filepath.Base(fn)))
		if err != nil {
			log.Fatal(err)
		}
		b := bufio.NewScanner(f)
		for b.Scan() {
			ln := b.Text()
			if strings.HasPrefix(ln, "/") { // strip comments
				continue
			}
			ln = repl.Replace(ln)
			fmt.Fprintln(of, ln)
		}
		of.Close()
	}

	if *genTest {
		if err := ioutil.WriteFile(filepath.Join(pkgName, fnamePre+"lfchan_test.go"), []byte(repl.Replace(testCode)), 0644); err != nil {
			log.Fatal(err)
		}
		out, err := exec.Command("go", "test", "-run=Chan", "./...").CombinedOutput()
		if err != nil {
			log.Fatalf("error running tests: %s %v", out, err)
		}
	}
}

func arg(idx int) string {
	if len(os.Args) <= idx {
		return ""
	}
	return os.Args[idx]
}

const testCode = `package lfchan

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

func TestChan(t *testing.T) {
	if reflect.TypeOf((*interface{})(nil)).Elem().Kind() == reflect.Interface {
		t.Skip("interfaces aren't supported by this test.")
	}
	var (
		N = 10000
		iv reflect.Value
		typ  = reflect.TypeOf(zeroValue)
		zero = reflect.Zero(typ).Interface().(interface{})
		ch = NewSize(100)
	)
	if testing.Short() {
		N = 1000
	}
	go ch.Send(zero, true)
	if v, ok := ch.Recv(true); !ok || !reflect.DeepEqual(v, zero) {
		t.Fatal("!ok || v != zero")
	}
	for {
		var ok bool
		if iv, ok = quick.Value(typ, rand.New(rand.NewSource(43))); !ok {
			t.Logf("!ok creating a random value")
			return
		}
		if iv.Kind() == reflect.Ptr && iv.IsNil() {
			continue
		}
		break
	}
	rv, ok := iv.Interface().(interface{})
	if !ok {
		t.Fatal("wrong value type")
	}
	t.Logf("test value (%T): %v", rv, rv)
	go func() {
		for i := 0; i < N; i++ {
			ch.Send(rv, true)
		}
		ch.Close()
	}()
	for i := 0; i < N; i++ {
		v, ok := ch.Recv(true)
		if !ok {
			t.Fatal("!ok")
		}
		if !reflect.DeepEqual(v, rv) {
			t.Fatalf("wanted %%v, got %%v", rv, v)
		}
	}
}
`

const usage = `Usage: %s [options] type [pkgName or . to embed]

Examples:
	# creates the needed files to create a chan for []*node in the current package
	#  and use basename(cwd) as the package name.
	$ go run gen.go "[]*node" .

	# creates the needed files to create a chan for []*node in the current package
	#  and use main as the package name.
	$ go run gen.go -main "[]*node" .

	$ go run gen.go string internal/stringChan # creates internal/stringChan sub-package

	$ go run gen.go string # creates stringChan sub-package
`
