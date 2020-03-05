// Copyright © 2020 The Pea Authors under an MIT-style license.

package gengo

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"text/template"
)

// Merger merges peago object files into a single, complete Go source file.
type Merger struct {
	TestMod string
	// Profile is whether to enable cpu and mem profiling in the output .go file.
	// When true, the generated program will write cpu.prof and mem.prof files
	// to the current directory when run.
	// These file can be read with go tool pprof.
	Profile bool
	w       io.Writer
	seen    map[string]bool
	inits   []string
	tests   []testFun

	includePrintForTests bool
}

type testFun struct {
	Name string
	Fun  string
}

// NewMerger writes a source header to the io.Writer and returns a new Merger.
func NewMerger(w io.Writer) (*Merger, error) {
	if _, err := io.WriteString(w, header); err != nil {
		return nil, err
	}
	return &Merger{w: w, seen: make(map[string]bool)}, nil
}

// Add adds the definitions from ant io.Reader to the output.
func (m *Merger) Add(r io.Reader) error {
	if m.seen == nil {
		panic("Merger.Add called after Merger.Done")
	}
	for {
		var name string
		var byteSize int64
		_, err := fmt.Fscanf(r, "%d %s\n", &byteSize, &name)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if m.seen[name] {
			if _, err := io.CopyN(ioutil.Discard, r, byteSize); err != nil {
				return err
			}
			continue
		}
		switch {
		case strings.HasPrefix(name, "T"):
			mod, pretty, err := demangleTestName(name)
			if err != nil {
				return err
			}
			if mod == m.TestMod {
				m.tests = append(m.tests, testFun{pretty, name})
			}
		case strings.HasSuffix(name, "init"):
			m.inits = append(m.inits, name)
		}
		m.seen[name] = true
		if _, err := io.CopyN(m.w, r, byteSize); err != nil {
			return err
		}
	}
	return nil
}

// Done writes a source footer to the output;
// it should now be a complete Go source file.
// After calling Done, the Merger can no longer be used.
func (m *Merger) Done() error {
	if m.seen == nil {
		panic("Merger.Done called multiple times")
	}
	m.seen = nil

	if m.includePrintForTests {
		if _, err := io.WriteString(m.w, printForTests); err != nil {
			return err
		}
	}

	t, err := template.New("main").Parse(mainTemplate)
	if err != nil {
		panic(err)
	}
	return t.Execute(m.w, map[string]interface{}{
		"Inits":   m.inits,
		"Tests":   m.tests,
		"Test":    m.TestMod != "",
		"Profile": m.Profile,
	})
}

const header = `package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"sync/atomic"
)

var tokenCounter int64

type retToken int64

func nextToken() retToken {
	return retToken(atomic.AddInt64(&tokenCounter, 1))
}

type panicVal struct {
	msg string
	file string
	line int
	testFile string
	testLine int
}

func recoverTestLoc(file string, line int) {
	switch r := recover().(type) {
	case nil:
		return
	case panicVal:
		r.testFile = file
		r.testLine = line
		panic(r)
	default:
		panic(r)
	}
}

func use(interface{}) {}

func F0___print_3A__(x *[]byte) {
	fmt.Printf("%v", string(*x))
}
`

const mainTemplate = `
{{if  .Test -}}
var exitStatus = 0

func runTest(name string, test func()) {
	fmt.Print("Test ", name, " ")
	defer func() {
		switch r := recover().(type) {
		case nil:
			fmt.Println("ok")
		case panicVal:
			exitStatus = 1
			fmt.Printf("failed\n\t%s:%d: %s\n", r.testFile, r.testLine, r.msg)
		default:
			panic(r)
		}
	}()
	test()
}
{{end -}}

func main() {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		switch r := r.(type) {
		case retToken:
			os.Stderr.WriteString("far return from a different stack\n")
		case panicVal:
			fmt.Fprintf(os.Stderr, "%s:%d: panic: %s\n", r.file, r.line, r.msg)
		default:
			panic(r)
		}
	}()

	if {{.Profile}} {
		f, err := os.Create("cpu.prof")
		if err != nil {
			panic("failed to create cpu profile file: " + err.Error())
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		defer func() {
			f, err := os.Create("mem.prof")
			if err != nil {
				panic("failed to create mem profile file: " + err.Error())
			}
			pprof.WriteHeapProfile(f)
			f.Close()
		}()
	}

	{{range .Inits -}}
	{{.}}()
	{{end -}}
	{{if not .Test -}}
		F0_main__main__()
	{{else -}}
		{{range .Tests -}}
		runTest({{printf "%q" .Name}}, {{.Fun}})
		{{end -}}
		os.Exit(exitStatus)
	{{end -}}
}
`

// These implement the main package's
// 	func T [print: _T]
// assumed in tests.
const printForTests = `
func F1___0_String__print_3A__(x *[]byte) {
	fmt.Printf("%v", string(*x))
}
func F1___0_String__main__print_3A__(x *[]byte) {
	fmt.Printf("%v", string(*x))
}
func F1___1__26____0_String__main__print_3A__(x *[]byte) {
	fmt.Printf("%v", string(*x))
}
func F1___0_Int__main__print_3A__(x int) {
	fmt.Printf("%v", x)
}
func F1___1__26____0_Int__main__print_3A__(x *int) {
	fmt.Printf("%v", *x)
}
func F1___0_Int8__main__print_3A__(x int8) {
	fmt.Printf("%v", x)
}
func F1___0_UInt8__main__print_3A__(x uint8) {
	fmt.Printf("%v", x)
}
func F1___0_UInt__main__print_3A__(x uint) {
	fmt.Printf("%v", x)
}
func F1___0_Float__main__print_3A__(x float64) {
	fmt.Printf("%v", x)
}
func F1___0_Float32__main__print_3A__(x float32) {
	fmt.Printf("%v", x)
}
func F1___0_Bool__main__print_3A__(x uint8) {
	fmt.Printf("%v", x == 1)
}
`
