package gengo

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/types"
)

func TestWriteMod(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		src    string
		stdout string
	}{
		{
			name:   "empty",
			src:    "func [main|]",
			stdout: "",
		},
		{
			name:   "int literal",
			src:    "func [main | print: 1]",
			stdout: "1",
		},
		{
			name:   "float literal",
			src:    "func [main | print: 3.14]",
			stdout: "3.14",
		},
		{
			name:   "string literal",
			src:    `func [main | print: "こんにちは、皆さん"]`,
			stdout: "こんにちは、皆さん",
		},
		{
			name:   "bool",
			src:    `func [main | print: true. print: "\n". print: false]`,
			stdout: "true\nfalse",
		},
		{
			name: "int & op",
			src: `
				func [main |
					print: 0 & 0. print: "\n".
					print: 0 & 1. print: "\n".
					print: 1 & 0. print: "\n".
					print: 1 & 1. print: "\n".
					print: 15 & 7. print: "\n".
				]
			`,
			stdout: "0\n0\n0\n1\n7\n",
		},
		{
			name: "int | op",
			src: `
				func [main |
					print: 0 | 0. print: "\n".
					print: 0 | 1. print: "\n".
					print: 1 | 0. print: "\n".
					print: 1 | 1. print: "\n".
					print: 10 | 7. print: "\n".
				]
			`,
			stdout: "0\n1\n1\n1\n15\n",
		},
		{
			name: "int not op",
			src: `
				func [main |
					print: 0 not. print: "\n".
					print: 1 not. print: "\n".
				]
			`,
			stdout: "-1\n-2\n",
		},
		{
			name: "int >> op",
			src: `
				func [main |
					print: 8 >> 0. print: "\n".
					print: 8 >> 1. print: "\n".
					print: 8 >> 2. print: "\n".
					print: 8 >> 3. print: "\n".
					print: 8 >> 4. print: "\n".
				]
			`,
			stdout: "8\n4\n2\n1\n0\n",
		},
		{
			name: "int >> op",
			src: `
				func [main |
					print: 1 << 0. print: "\n".
					print: 1 << 1. print: "\n".
					print: 1 << 2. print: "\n".
					print: 1 << 3. print: "\n".
					print: 1 << 4. print: "\n".
				]
			`,
			stdout: "1\n2\n4\n8\n16\n",
		},
		{
			name: "int neg op",
			src: `
				func [main |
					print: 1 neg. print: "\n".
					print: -1 neg. print: "\n".
					print: 5 neg. print: "\n".
				]
			`,
			stdout: "-1\n1\n-5\n",
		},
		{
			name: "int + op",
			src: `
				func [main |
					print: 1 + 1. print: "\n".
					print: 1 + -1. print: "\n".
					print: 0 + 0. print: "\n".
					print: 2 + 40. print: "\n".
				]
			`,
			stdout: "2\n0\n0\n42\n",
		},
		{
			name: "int - op",
			src: `
				func [main |
					print: 1 - 1. print: "\n".
					print: 1 - -1. print: "\n".
					print: 0 - 0. print: "\n".
					print: 44 - 2. print: "\n".
				]
			`,
			stdout: "0\n2\n0\n42\n",
		},
		{
			name: "int * op",
			src: `
				func [main |
					print: 1 * 1. print: "\n".
					print: 1 * -1. print: "\n".
					print: 1 * 0. print: "\n".
					print: 21 * 2. print: "\n".
				]
			`,
			stdout: "1\n-1\n0\n42\n",
		},
		{
			name: "int / op",
			src: `
				func [main |
					print: 1 / 1. print: "\n".
					print: 1 / -1. print: "\n".
					print: 0 / 1. print: "\n".
					print: 84 / 2. print: "\n".
				]
			`,
			stdout: "1\n-1\n0\n42\n",
		},
		{
			name: "int % op",
			src: `
				func [main |
					print: 1 % 1. print: "\n".
					print: 5 % 2. print: "\n".
				]
			`,
			stdout: "0\n1\n",
		},
		{
			name: "int = op",
			src: `
				func [main |
					print: 1 = 1. print: "\n".
					print: 1 = 2. print: "\n".
				]
			`,
			stdout: "true\nfalse\n",
		},
		{
			name: "int != op",
			src: `
				func [main |
					print: 1 != 1. print: "\n".
					print: 1 != 2. print: "\n".
				]
			`,
			stdout: "false\ntrue\n",
		},
		{
			name: "int < op",
			src: `
				func [main |
					print: 1 < 0. print: "\n".
					print: 1 < 1. print: "\n".
					print: 1 < 2. print: "\n".
				]
			`,
			stdout: "false\nfalse\ntrue\n",
		},
		{
			name: "int <= op",
			src: `
				func [main |
					print: 1 <= 0. print: "\n".
					print: 1 <= 1. print: "\n".
					print: 1 <= 2. print: "\n".
				]
			`,
			stdout: "false\ntrue\ntrue\n",
		},
		{
			name: "int > op",
			src: `
				func [main |
					print: 1 > 0. print: "\n".
					print: 1 > 1. print: "\n".
					print: 1 > 2. print: "\n".
				]
			`,
			stdout: "true\nfalse\nfalse\n",
		},
		{
			name: "int >= op",
			src: `
				func [main |
					print: 1 >= 0. print: "\n".
					print: 1 >= 1. print: "\n".
					print: 1 >= 2. print: "\n".
				]
			`,
			stdout: "true\ntrue\nfalse\n",
		},
		{
			name: "int asFloat",
			src: `
				func [main |
					print: 1 asFloat. print: "\n".
				]
			`,
			stdout: "1\n",
		},
		{
			name: "function return value",
			src: `
				func [main | print: foo]
				func [foo ^String | ^"こんにちは、皆さん"]
			`,
			stdout: "こんにちは、皆さん",
		},
		{
			name: "function int argument",
			src: `
				func [main | foo: 42]
				func [foo: i Int | print: i]
			`,
			stdout: "42",
		},
		{
			name: "function String argument",
			src: `
				func [main | foo: "Hello, World"]
				func [foo: s String | print: s]
			`,
			stdout: "Hello, World",
		},
		{
			name: "simple field value",
			src: `
				func [main |
					p Point := {x: 42 y: 43}.
					p printX.
					print: "\n".
					p printY.
				]
				type Point {x: Int y: Int}
				meth Point [printX | print: x]
				meth Point [printY | print: y]
			`,
			stdout: "42\n43",
		},
		{
			name: "composite field value",
			src: `
				func [main |
					p StringPair := {x: "Hello, " y: "World"}.
					p printX.
					p printY.
				]
				type StringPair {x: String y: String}
				meth StringPair [printX | print: x]
				meth StringPair [printY | print: y]
			`,
			stdout: "Hello, World",
		},
		{
			name: "empty field type",
			src: `
				func [main |
					p Point := {x: 42 y: 43 z: {}}.
					p printX.
					print: "\n".
					p printY.
				]
				type Point {x: Int y: Int z: Nil}
				meth Point [printX | print: x]
				meth Point [printY | print: y]
			`,
			stdout: "42\n43",
		},
		{
			name: "array element",
			src: `
				func [main |
					p Int Array := {42; 43}.
					print: (p at: 0).
					print: "\n".
					print: (p at: 1).
					print: "\n".
					p at: 0 put: 82.
					p at: 1 put: 83.
					print: (p at: 0).
					print: "\n".
					print: (p at: 1).
				]
			`,
			stdout: "42\n43\n82\n83",
		},
		{
			name: "array slice element",
			src: `
				func [main |
					p Int Array := {40; 41; 42; 43; 44; 45}.
					s := p from: 2 to: 3.
					print: s size.
					print: "\n".
					print: (s at: 0).
				]
			`,
			stdout: "1\n42",
		},
		{
			name: "module variable",
			src: `
				func [main |
					print: fourtyTwo.
					print: "\n".
					print: fourtyThree.
				]
				val fourtyTwo := [42]
				val fourtyThree := [43]
			`,
			stdout: "42\n43",
		},
		{
			name: "virtual",
			src: `
				func [main |
					v Printer := 42.
					v print.
					print: "\n".
					v := "Hello, World".
					v print.
				]
				type Printer {[print]}
				meth Int [print | print: self]
				meth String [print | print: self]
			`,
			stdout: "42\nHello, World",
		},
		{
			name: "virtual with parm",
			src: `
				func [main |
					v Printer := 42.
					v printMaybeNl: true.
					v printMaybeNl: false
				]
				type Printer {
					[printMaybeNl: Bool]
				}
				meth Int [printMaybeNl: nl Bool |
					print: self.
					nl ifTrue: [print: "\n"] ifFalse: []
				]
			`,
			stdout: "42\n42",
		},
		{
			name: "virtual with empty type parm",
			src: `
				func [main |
					v Printer := 42.
					v printIgnoredNilParm: {}
				]
				type Printer {
					[printIgnoredNilParm: Nil]
				}
				meth Int [printIgnoredNilParm: n Nil | print: self]
			`,
			stdout: "42",
		},
		{
			name: "virtual with multiple parms",
			src: `
				func [main |
					v Printer := 42.
					v print: "a" then: "b" finally: "c".
				]
				type Printer {
					[print: String then: String finally: String]
				}
				meth Int [print: a String then: b String finally: c String |
					print: self.
					print: "\n".
					print: a.
					print: "\n".
					print: b.
					print: "\n".
					print: c.
				]
			`,
			stdout: "42\na\nb\nc",
		},
		{
			name: "virtual with simple return",
			src: `
				func [main |
					v Printer := 42.
					v print print print
				]
				type Printer {
					[print ^Int]
				}
				meth Int [print ^Int |
					print: self.
					print: "\n".
					^self + 1
				]
			`,
			stdout: "42\n43\n44\n",
		},
		{
			name: "virtual with composite return",
			src: `
				func [main |
					v Printer := 42.
					v print print
				]
				type Printer {
					[print ^String]
				}
				meth Int [print ^String |
					print: self.
					print: "\n".
					^"Hello"
				]
				meth String [print | print: self. ]
			`,
			stdout: "42\nHello",
		},
		{
			// TODO: gengo stack overflows emitting a recursive type
			// like Printer here, whcih has a method referring to itself.
			name: "SKIP: virtual with return recursive type",
			src: `
				func [main |
					v Printer := 42.
					v print print print
				]
				type Printer {
					[print ^Printer]
				}
				meth Int [print ^Printer |
					print: self.
					print: "\n".
					^self + 1
				]
			`,
			stdout: "42\n43\n44\n",
		},
		{
			name: "virtual with parms and return",
			src: `
				func [main |
					v Printer := 42.
					print: (v print: "a" then: "b" return: "c")
				]
				type Printer {
					[print: String then: String return: String ^String]
				}
				meth Int [print: a String then: b String return: c String ^String |
					print: self.
					print: "\n".
					print: a.
					print: "\n".
					print: b.
					print: "\n".
					^c.
				]
			`,
			stdout: "42\na\nb\nc",
		},
		{
			name: "switch simple value",
			src: `
				func [main |
					o Int? := {some: 5}.
					o ifNone: [print: "none"] ifSome: [:i | print: i].
					print: "\n".
					o := {none}.
					o ifNone: [print: "none"] ifSome: [:i | print: i].
				]
				type T? {none | some: T}
			`,
			stdout: "5\nnone",
		},
		{
			name: "switch composite value",
			src: `
				func [main |
					o String? := {some: "Hello"}.
					o ifNone: [print: "none"] ifSome: [:s | print: s].
					print: "\n".
					o := {none}.
					o ifNone: [print: "none"] ifSome: [:s | print: s].
				]
				type T? {none | some: T}
			`,
			stdout: "Hello\nnone",
		},
		{
			name: "for loop",
			src: `
				func [main | 1 to: 10 do: [:i | print: i. print: "\n"]]

				meth Int [to: e Int do: f (Int, Nil) Fun |
					self <= e ifTrue: [
						f value: self.
						self + 1 to: e do: f.
					]
				]

				meth Bool [ifTrue: f Nil Fun | self ifTrue: f ifFalse: []]
			`,
			stdout: "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n",
		},
		{
			name: "loop over virtual array",
			src: `
				func [main |
					point Point := {x: 5 y: 10}.
					ps Printer Array := {42; "Hello, World"; point}.
					ps do: [:p | p print. print: "\n".]
				]

				type Printer {[print]}

				meth Int [print | print: self]
				meth String [print | print: self]

				type Point {x: Int y: Int}

				meth Point [print |
					print: "{x: ". print: x.
					print: " y: ". print: y.
					print: "}"
				]

				meth T Array [do: f (T&, Nil) Fun |
					0 to: self size - 1 do: [:i | f value: (self at: i)]
				]

				meth Int [to: e Int do: f (Int, Nil) Fun |
					self <= e ifTrue: [
						f value: self.
						self + 1 to: e do: f.
					]
				]

				meth Bool [ifTrue: f Nil Fun | self ifTrue: f ifFalse: []]
			`,
			stdout: "42\nHello, World\n{x: 5 y: 10}\n",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if strings.HasPrefix(test.name, "SKIP") {
				t.Skip()
			}
			src := "func T [print: _ T]\n" + test.src
			mod, errs := compile(src)
			if len(errs) > 0 {
				t.Fatalf("failed to compile: %v", errs)
			}
			stdout, err := run(mod)
			if err != nil {
				t.Fatalf("failed to run: %v", err)
			}
			if stdout != test.stdout {
				t.Fatalf("got [%s], want [%s]", stdout, test.stdout)
			}
		})
	}
}

func compile(src string) (*basic.Mod, []error) {
	p := ast.NewParser("main")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		return nil, []error{err}
	}
	typesMod, errs := types.Check(p.Mod(), types.Config{})
	if len(errs) > 0 {
		return nil, errs
	}
	basicMod := basic.Build(typesMod)
	basic.Optimize(basicMod)
	return basicMod, nil
}

func run(mod *basic.Mod) (string, error) {
	f, err := ioutil.TempFile("", "gengo_test_*.go")
	if err != nil {
		return "", err
	}
	WriteMod(f, mod)
	path := f.Name()
	if err := f.Close(); err != nil {
		return "", err
	}
	cmd := exec.Command("go", "run", path)
	var stdOut bytes.Buffer
	cmd.Stdout = &stdOut
	runErr := cmd.Run()
	rmErr := os.Remove(path)
	if runErr != nil {
		return "", runErr
	}
	if rmErr != nil {
		return "", rmErr
	}
	return stdOut.String(), nil
}
