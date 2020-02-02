package gengo

import (
	"bytes"
	"errors"
	"fmt"
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
		name    string
		src     string
		stdout  string
		stderr  string
		imports [][2]string
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
			name: "string element",
			src: `
				func [main |
					print: ("Hello" atByte: 2).
				]
			`,
			stdout: "108", // l
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
			// This is to catch a regression
			// where non-inlined module variable init functions
			// were not being generated.
			name: "module variable init calls a function",
			src: `
				func [main |
					print: fourtyTwo.
					print: "\n".
					print: fourtyThree.
				]
				// The calls here prevent inlining.
				val fourtyTwo := [self: 42]
				val fourtyThree := [self: 43]
				func [self: i Int ^Int | ^i]
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
			name: "virtual with return recursive type",
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
		{
			name: "recursive type",
			src: `
				func [main |
					abc String List := cons: "a" and: (cons: "b" and: (cons: "c" and: nil)).
					abc do: [:s | print: s. print: "\n"]
				]

				type T List {nil | elm: T Elm&}

				type T Elm {data: T next: T List}
				meth T Elm [data ^T& | ^data]
				meth T Elm [next ^T List | ^next]

				func T [nil ^T List | ^{nil}]

				func T [cons: t T and: ts T List ^T List | ^{elm: {data: t next: ts}}]

				meth _ List [size ^Int |
					^self ifNil: [0] ifElm: [:e | 1 + e next size]
				]

				meth T List [do: f (T&, Nil) Fun |
					self ifNil: [] ifElm: [:e |
						f value: e data.
						e next do: f.
					]
				]
			`,
			stdout: "a\nb\nc\n",
		},
		{
			name: "or type enum",
			src: `
				type Enum {one | two | three}
				func [main |
					x Enum := {one}.
					x ifOne: [print: 1] ifTwo: [print: 2] ifThree: [print: 3].
				]
			`,
			stdout: "1",
		},
		{
			name: "imported val",
			src: `
				import "/test/sayhi"
				func [main |
					print: #sayhi helloWorld
				]
			`,
			imports: [][2]string{
				{
					"/test/sayhi",
					`Val helloWorld := ["Hello, World"]`,
				},
			},
			stdout: "Hello, World",
		},
		{
			name: "same val name different imports",
			src: `
				import "/test/sayhi"
				import "/test/sayhi2"
				func [main |
					print: #sayhi helloWorld.
					print: "\n".
					print: #sayhi2 helloWorld
				]
			`,
			imports: [][2]string{
				{
					"/test/sayhi",
					`Val helloWorld := ["Hello, World"]`,
				},
				{
					"/test/sayhi2",
					`Val helloWorld := ["こんにちは、皆さん"]`,
				},
			},
			stdout: "Hello, World\nこんにちは、皆さん",
		},
		{
			name: "imported func",
			src: `
				import "/test/sayhi"
				func [main |
					print: #sayhi helloWorld
				]
			`,
			imports: [][2]string{
				{
					"/test/sayhi",
					`Func [helloWorld ^String | ^"Hello, World"]`,
				},
			},
			stdout: "Hello, World",
		},
		{
			name: "same func selector different imports",
			src: `
				import "/test/sayhi"
				import "/test/sayhi2"
				func [main |
					print: #sayhi helloWorld.
					print: "\n".
					print: #sayhi2 helloWorld
				]
			`,
			imports: [][2]string{
				{
					"/test/sayhi",
					`Func [helloWorld ^String | ^"Hello, World"]`,
				},
				{
					"/test/sayhi2",
					`Func [helloWorld ^String | ^"こんにちは、皆さん"]`,
				},
			},
			stdout: "Hello, World\nこんにちは、皆さん",
		},
		{
			name: "imported type",
			src: `
				import "/test/point"
				func [main |
					p #point Point := {x: 5 y: 42}.
					print: p #point x.
					print: "\n".
					print: p #point y
				]
			`,
			imports: [][2]string{
				{
					"/test/point",
					`
					Type Point {x: Int y: Int}
					Meth Point [x ^Int | ^x]
					Meth Point [y ^Int | ^y]
					`,
				},
			},
			stdout: "5\n42",
		},
		{
			name: "same type name different imports",
			src: `
				import "/test/point"
				import "/test/point2"
				func [main |
					p #point Point := {x: 5 y: 42}.
					print: p #point x.
					print: "\n".
					print: p #point y.
					print: "\n".
					q #point2 Point := {x: 5.1 y: 42.2}.
					print: q #point2 x.
					print: "\n".
					print: q #point2 y
				]
			`,
			imports: [][2]string{
				{
					"/test/point",
					`
					Type Point {x: Int y: Int}
					Meth Point [x ^Int | ^x]
					Meth Point [y ^Int | ^y]
					`,
				},
				{
					"/test/point2",
					`
					Type Point {x: Float y: Float}
					Meth Point [x ^Float | ^x]
					Meth Point [y ^Float | ^y]
					`,
				},
			},
			stdout: "5\n42\n5.1\n42.2",
		},
		{
			name: "dedup type instances across imports",
			src: `
				Import "/test/opt"
				import "/test/a"
				import "/test/b"
				func [main |
					#a one ifNone: [] ifSome: [:i | print: i].
					print: "\n".
					#b two ifNone: [] ifSome: [:i | print: i].
				]
			`,
			imports: [][2]string{
				{
					"/test/opt",
					`
					Type T? {none | some: T}
					`,
				},
				{
					"/test/a",
					`
					Import "/test/opt"
					Func [one ^Int? | ^{some: 1}]
					`,
				},
				{
					"/test/b",
					`
					Import "/test/opt"
					Func [two ^Int? | ^{some: 2}]
					`,
				},
			},
			stdout: "1\n2",
		},
		{
			name: "dedup func instances across imports",
			src: `
				import "/test/a"
				import "/test/b"
				func [main |
					print: #a one.
					print: "\n".
					print: #b two.
				]
			`,
			imports: [][2]string{
				{
					"/test/yourself",
					`
					Func T [yourself: t T ^T | ^t]
					`,
				},
				{
					"/test/a",
					`
					Import "/test/yourself"
					Func [one ^Int | ^#yourself yourself: 1]
					`,
				},
				{
					"/test/b",
					`
					Import "/test/yourself"
					Func [two ^Int | ^#yourself yourself: 2]
					`,
				},
			},
			stdout: "1\n2",
		},
		{
			name: "modify simple-type self",
			src: `
				func [main |
					i := 0.
					i increment.
					i increment.
					i increment.
					print: i.
				]
				meth Int [increment | self := self + 1]
			`,
			stdout: "3",
		},
		{
			name: "modify composite-type self",
			src: `
				func [main |
					s := "abc".
					s set: "xyz".
					print: s.
				]
				meth String [set: s String | self := s]
			`,
			stdout: "xyz",
		},
		{
			name: "modify simple-type ref argument",
			src: `
				func [main |
					i := 0.
					increment: i.
					increment: i.
					increment: i.
					print: i.
				]
				func [increment: i Int& | i increment]
				meth Int [increment | self := self + 1]
			`,
			stdout: "3",
		},
		{
			name: "modify composite-type ref argument",
			src: `
				func [main |
					s := "abc".
					set: s to: "xyz".
					print: s.
				]
				func [set: s String& to: t String | s set: t]
				meth String [set: s String | self := s]
			`,
			stdout: "xyz",
		},
		{
			name: "modify simple-type array element",
			src: `
				func [main |
					is Int Array := {0; 0; 0}.
					(is at: 1) increment.
					(is at: 1) increment.
					(is at: 1) increment.
					print: (is at: 1).
				]
				meth Int [increment | self := self + 1]
			`,
			stdout: "3",
		},
		{
			name: "modify composite-type array element",
			src: `
				func [main |
					ss String Array := {"abc"; "abc"; "abc"}.
					(ss at: 1) set: "xyz".
					print: (ss at: 1).
				]
				meth String [set: s String | self := s]
			`,
			stdout: "xyz",
		},
		{
			name: "non-const Fun",
			src: `
				func [main |
					x := [1].
					x := [2].
					print: x value
				]
			`,
			stdout: "2",
		},
		{
			name: "non-const Fun changed by method",
			src: `
				func [main |
					x := [1].
					x set: [2].
					print: x value
				]
				meth T Fun [set: f T Fun |
					print: "\n". // prevents inlining set:
					self := f
				]
			`,
			stdout: "\n2",
		},
		{
			name: "make virtual of built-in case method",
			src: `
				func [main |
					iopt Int? := {some: 5}.
					v (Int, Float) IfNoneIfSomer := iopt.
					print: (v ifNone: [3.14] ifSome: [:i | i asFloat])
				]
				type (T, U) IfNoneIfSomer {
					[ifNone: U Fun ifSome: (T&, U) Fun ^U]
				}
				type T? {none | some: T}
			`,
			stdout: "5",
		},
		{
			name: "make virtual of built-in String byteSize method",
			src: `
				func [main |
					s := "Hello".
					v ByteSizer := s.
					print: v byteSize
				]
				type ByteSizer {[byteSize ^Int]}
			`,
			stdout: "5",
		},
		{
			// String atByte is broken in a few places
			name: "SKIP: make virtual of built-in String atByte method",
			src: `
				func [main |
					s := "Hello".
					v AtByter := s.
					print: (v atByte: 1) asInt.
				]
				type AtByter {[atByte: Int ^Byte]}
			`,
			stdout: "101", // e
		},
		{
			name: "make virtual of built-in Array size method",
			src: `
				func [main |
					ints Int Array := {5; 6; 7}.
					v Sizer := ints.
					print: v size
				]
				type Sizer {[size ^Int]}
			`,
			stdout: "3",
		},
		{
			name: "make virtual of built-in Array at: method",
			src: `
				func [main |
					ints Int Array := {5; 6; 7}.
					v Ater := ints.
					print: (v at: 1)
				]
				type Ater {[at: Int ^Int&]}
			`,
			stdout: "6",
		},
		{
			name: "make virtual of built-in Array at:put: method",
			src: `
				func [main |
					ints Int Array := {5; 6; 7}.
					v AtPuter := ints.
					v at: 1 put: 42.
					print: (ints at: 1)
				]
				type AtPuter {[at: Int put: Int]}
			`,
			stdout: "42",
		},
		{
			name: "make virtual of built-in Array from:to: method",
			src: `
				func [main |
					ints Int Array := {5; 6; 7}.
					v FromToer := ints.
					x := v from: 1 to: 2.
					print: (x at: 0)
				]
				type FromToer {[from: Int to: Int ^Int Array]}
			`,
			stdout: "6",
		},
		{
			name: "make virtual of integer methods",
			src: `
				func [main |
					v Num := 42 asInt8.		// 0010 1010 = 42
					print: v & 42. print: "_".	// 42
					print: v | 1. print: "_".	// 43
					print: v not. print: "_".	// -43
					print: v >> 1. print: "_".	// 0001 0101 = 21
					print: v << 1. print: "_".	// 0101 0100 = 88
					print: v neg. print: "_".	// -42
					print: v + 1. print: "_".	// 43
					print: v - 1. print: "_".	// 41
					print: v * 2. print: "_".	// 84
					print: v / 2. print: "_".	// 21
					print: v % 2. print: "_".	// 0
					print: v = 42. print: "_".	// true
					print: v != 42. print: "_".	// false
					print: v < 42. print: "_".	// false
					print: v <= 42. print: "_".	// true
					print: v > 42. print: "_".	// false
					print: v >= 42. print: "_".	// true
					print: v asUInt. print: "_".	// 42
					print: v asFloat. print: "_".// 42
				]
				type Num {
					[& Int8 ^Int8]
					[| Int8 ^Int8]
					[not ^Int8]
					[>> Int ^Int8]
					[<< Int ^Int8]
					[neg ^Int8]
					[+ Int8 ^Int8]
					[- Int8 ^Int8]
					[* Int8 ^Int8]
					[/ Int8 ^Int8]
					[% Int8 ^Int8]
					[= Int8 ^Bool]
					[!= Int8 ^Bool]
					[< Int8 ^Bool]
					[<= Int8 ^Bool]
					[> Int8 ^Bool]
					[>= Int8 ^Bool]
					[asUInt ^UInt]
					[asFloat ^Float]
				}
			`,
			stdout: "42_43_-43_21_84_-42_43_41_84_21_0_true_false_false_true_false_true_42_42_",
		},
		{
			name: "make virtual of float methods",
			src: `
				func [main |
					v Num := 42.25 asFloat32.
					print: v neg. print: "_".	// -42.25
					print: v + 1. print: "_".	// 43.25
					print: v - 1. print: "_".	// 41.25
					print: v * 2. print: "_".	// 84.5
					print: v / 2. print: "_".	// 21.125
					print: v = 42.25. print: "_".	// true
					print: v != 42.25. print: "_".	// false
					print: v < 42.25. print: "_".	// false
					print: v <= 42.25. print: "_".	// true
					print: v > 42.25. print: "_".	// false
					print: v >= 42.25. print: "_".	// true
					print: v asUInt. print: "_".	// 42
					print: v asFloat. print: "_".// 42.25
				]
				type Num {
					[neg ^Float32]
					[+ Float32 ^Float32]
					[- Float32 ^Float32]
					[* Float32 ^Float32]
					[/ Float32 ^Float32]
					[= Float32 ^Bool]
					[!= Float32 ^Bool]
					[< Float32 ^Bool]
					[<= Float32 ^Bool]
					[> Float32 ^Bool]
					[>= Float32 ^Bool]
					[asUInt ^UInt]
					[asFloat ^Float]
				}
			`,
			stdout: "-42.25_43.25_41.25_84.5_21.125_true_false_false_true_false_true_42_42.25_",
		},
		{
			name: "unary associativity",
			src: `
				func [main |
					"" print1 print2 print3
				]
				meth String [print1 ^String | print: 1. ^""]
				meth String [print2 ^String | print: 2. ^""]
				meth String [print3 ^String | print: 3. ^""]
			`,
			stdout: "123",
		},
		{
			// This test is to catch a bug regression
			// found writing the initial implemention
			// of the "unary associativity" test above.
			// There was a crash with a nil receiver.
			name: "call on nil receiver",
			src: `
				func [main |
					"" print1 print2 print3
				]
				meth String [print1 | print: 1]
				meth Nil [print2 | print: 2]
				meth Nil [print3 | print: 3]
			`,
			stdout: "123",
		},
		{
			name: "binary associativity",
			src: `
				func [main |
					"" % 1 % 2 % 3
				]
				meth String [% i Int ^String | print: i. ^""]
			`,
			stdout: "123",
		},
		{
			name: "binary associativity 2",
			src: `
				func [main |
					print: 1 - 5 + 6
				]
			`,
			stdout: "2",
		},
		{
			name: "binary and unary precedence",
			src: `
				func [main |
					"" print1 % 3
				]
				meth String [print1 ^Int | print: 1. ^2]
				meth Int [% i Int | print: self. print: i]
			`,
			stdout: "123",
		},
		{
			name: "far return probably inlined",
			src: `
				func [main | print: num]
				func [num ^Int | [^42] value. ^43]
			`,
			stdout: "42",
		},
		{
			name: "nil far return",
			src: `
				func [main | print: ([^{}] value)]
			`,
			stdout: "",
		},
		{
			name: "far far return probably inlined",
			src: `
				func [main |print: num]
				func [num ^Int | value3: [^42]. ^43]
				func [value3: f Nil Fun | value2: f]
				func [value2: f Nil Fun | value1: f]
				func [value1: f Nil Fun | f value]
			`,
			stdout: "42",
		},
		{
			name: "far return probably not inlined",
			src: `
				func [main | print: num]
				func [num ^Int |
					f := [^43].
					true ifTrue: [f := [^42]] ifFalse: [].
					f value.
					^44
				]
			`,
			stdout: "42",
		},
		{
			name: "far far return probably not inlined",
			src: `
				func [main |print: num]
				func [num ^Int |
					f := [^43].
					true ifTrue: [f := [^42]] ifFalse: [].
					value3: f.
					^44
				]
				func [value3: f Nil Fun | value2: f]
				func [value2: f Nil Fun | value1: f]
				func [value1: f Nil Fun | f value]
			`,
			stdout: "42",
		},
		{
			name: "far return on different stack",
			src: `
				val f Nil Fun := [[]]

				func [main |
					// TODO: allow unused function returns.
					// There is a bug in gengo that will complain
					// if we don't use the return of foo here.
					// So just print it for now so that it's used.
					print: foo.
					f value.
				]

				func [foo ^Int |
					f := [^42].
					^0.
				]
			`,
			stdout: "0",
			stderr: "far return from a different stack\n",
		},
		{
			name: "binary tree traversal far return",
			src: `
				func [main |
					t Tree := {
						x: 5
						left: (x: 1 left: (x: 0) right: (x: 2))
						right: (x: 43 left: (x: 42) right: (x: 44))
					}.
					(greaterThan40: t)
						ifNone: [print: "none"]
						ifSome: [:i | print: i].
				]

				func [greaterThan40: t Tree& ^Int? |
					t do: [:i | i > 40 ifTrue: [^some: i] ifFalse: []].
					^none
				]

				type T? {none | some: T}
				func T [none ^T? | ^{none}]
				func T [some: t T ^T? | ^{some: t}]

				type Tree {x: Int left: Tree& ? right: Tree& ?}

				func [x: i Int ^Tree& ? |
					^some: {x: i left: none right: none}
				]

				func [x: i Int left: l Tree& ? right: r Tree& ? ^Tree& ? |
					^some: {x: i left: l right: r}
				]

				meth Tree [do: f (Int, Nil) Fun |
					left ifNone: [] ifSome: [:l | l do: f].
					f value: x.
					right ifNone: [] ifSome: [:l | l do: f].
				]
			`,
			stdout: "42",
		},
		{
			name: "do not inline far return function",
			src: `
				val f Nil Fun := [[]]

				func [main |
					print: foo
				]

				func [foo ^Int |
					f := [^0].
					f := [^42].
					f value.
					^0
				]
			`,
			stdout: "42",
		},
		{
			name: "newArray:init: empty array",
			src: `
				func [main |
					ary := newArray: 0 init: [:i Int | i].
					print: ary size. print: "\n".
				]
			`,
			stdout: "0\n",
		},
		{
			name: "newArray:init: simple type",
			src: `
				func [main |
					ary := newArray: 3 init: [:i Int | i].
					print: ary size. print: "\n".
					print: (ary at: 0). print: "\n".
					print: (ary at: 1). print: "\n".
					print: (ary at: 2). print: "\n".
				]
			`,
			stdout: "3\n0\n1\n2\n",
		},
		{
			name: "newArray:init: composite type",
			src: `
				func [main |
					ary := newArray: 3 init: [:i Int | pt Point := {x: i y: i}. pt].
					print: ary size. print: "\n".
					(ary at: 0) print. print: "\n".
					(ary at: 1) print. print: "\n".
					(ary at: 2) print. print: "\n".
				]

				type Point {x: Int y: Int}

				meth Point [print |
					print: "{x: ". print: x. print: " y: ". print: y. print: "}"
				]
			`,
			stdout: "3\n{x: 0 y: 0}\n{x: 1 y: 1}\n{x: 2 y: 2}\n",
		},
		{
			name: "newString",
			src: `
				func [main |
					print: (newString: {104; 101; 108; 108; 111; 10}).
				]
			`,
			stdout: "hello\n",
		},
		{
			name: "newString is immutable",
			src: `
				func [main |
					data Byte Array := {104; 101; 108; 108; 111; 10}.
					str := newString: data.
					data at: 0 put: 0.
					data at: 1 put: 0.
					data at: 2 put: 0.
					data at: 3 put: 0.
					data at: 4 put: 0.
					print: str.
				]
			`,
			stdout: "hello\n",
		},
		{
			name: "newString empty",
			src: `
				func [main |
					print: (newString: {}).
				]
			`,
			stdout: "",
		},
		{
			name: "newString unicode",
			src: `
				func [main |
					print: (newString: {226; 152; 186}).
				]
			`,
			stdout: "☺",
		},
		{
			name: "panic",
			src: `									// 1
				func [main |						// 2
					foo.							// 3
					print: "this is not printed".	// 4
				]									// 5
													// 6
				func [foo |							// 7
					panic: "boo"					// 8
				]
			`,
			stderr: ":8: panic: boo\n",
		},
		{
			name: "ignore code after a panic",
			src: `									// 1
				func [main |						// 2
					panic: "boo".					// 3
					print: "this is not printed".
				]
			`,
			stderr: ":3: panic: boo\n",
		},
		{
			name: "return-ending block can have any result type",
			src: `
				func [main | print: foo]
				func [foo ^Int |
					// This is a String Fun because it ends in a return.
					bar: [^42].
					^0.
				]
				func [bar: f String Fun | print: f value]
			`,
			stdout: "42",
		},
		{
			name: "panic-ending block can have any result type",
			src: `							// 1
				func [main | print: foo]	// 2
				func [foo ^Int |			// 3
					bar: [panic: "oh no"].	// 4
					^0.
				]
				func [bar: f String Fun | print: f value]
			`,
			stderr: ":4: panic: oh no\n",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if strings.HasPrefix(test.name, "SKIP") {
				t.Skip()
			}
			src := test.src + "\nfunc T [print: _ T]\n"
			mods, errs := compileAll(src, test.imports...)
			if len(errs) > 0 {
				t.Fatalf("failed to compile: %v", errs)
			}
			stdout, stderr, err := run(mods)
			if err != nil {
				t.Fatalf("failed to run: %v", err)
			}
			if stdout != test.stdout {
				t.Errorf("stdout: got [%s], want [%s]", stdout, test.stdout)
			}
			if stderr != test.stderr {
				t.Errorf("stderr: got [%s], want [%s]", stderr, test.stderr)
			}
		})
	}
}

func check(modPath, src string, imports ...[2]string) (*types.Mod, []error) {
	p := ast.NewParser(modPath)
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		return nil, []error{err}
	}
	return types.Check(p.Mod(), types.Config{
		Importer: testImporter(imports),
	})
}

func compile(modPath, src string, imports ...[2]string) (*basic.Mod, []error) {
	typesMod, errs := check(modPath, src, imports...)
	if len(errs) > 0 {
		return nil, errs
	}
	basicMod := basic.Build(typesMod)
	basic.Optimize(basicMod)
	return basicMod, nil
}

func compileAll(src string, imports ...[2]string) ([]*basic.Mod, []error) {
	mod, errs := compile("main", src, imports...)
	if len(errs) > 0 {
		return nil, errs
	}
	mods := []*basic.Mod{mod}
	for _, imp := range imports {
		mod, errs = compile(imp[0], imp[1], imports...)
		if len(errs) > 0 {
			return nil, errs
		}
		mods = append(mods, mod)
	}
	return mods, nil
}

func run(mods []*basic.Mod) (string, string, error) {
	f, err := ioutil.TempFile("", "gengo_test_*.go")
	if err != nil {
		return "", "", err
	}
	merger, err := NewMerger(f)
	if err != nil {
		return "", "", err
	}
	merger.includePrintForTests = true
	for _, mod := range mods {
		var b bytes.Buffer
		if err := WriteMod(&b, mod); err != nil {
			return "", "", err
		}
		if err := merger.Add(&b); err != nil {
			return "", "", err
		}
	}
	if err := merger.Done(); err != nil {
		return "", "", err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return "", "", err
	}
	cmd := exec.Command("go", "run", path)
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	runErr := cmd.Run()
	rmErr := os.Remove(path)
	if runErr != nil {
		return "", "", runErr
	}
	if rmErr != nil {
		return "", "", rmErr
	}
	return stdOut.String(), stdErr.String(), nil
}

type testImporter [][2]string

func (imports testImporter) Import(cfg types.Config, path string) ([]types.Def, error) {
	for i := range imports {
		if imports[i][0] != path {
			continue
		}
		src := imports[i][1]
		p := ast.NewParser(path)
		if err := p.Parse(path, strings.NewReader(src)); err != nil {
			return nil, fmt.Errorf("failed to parse import: %s", err)
		}
		cfg.Trace = false
		mod, errs := types.Check(p.Mod(), cfg)
		if len(errs) > 0 {
			return nil, fmt.Errorf("failed to check import: %s", errs)
		}
		setMod(path, mod.Defs)
		return mod.Defs, nil
	}
	return nil, errors.New("not found")
}

func setMod(path string, defs []types.Def) {
	for _, def := range defs {
		switch def := def.(type) {
		case *types.Val:
			def.ModPath = path
		case *types.Fun:
			def.ModPath = path
		case *types.Type:
			def.ModPath = path
		default:
			panic(fmt.Sprintf("impossible type: %T", def))
		}
	}
}
