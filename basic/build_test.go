// Copyright © 2020 The Pea Authors under an MIT-style license.

package basic

import (
	"bufio"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/types"
	"github.com/google/go-cmp/cmp"
)

// Tests the build pass, which generates literal, clumsy code.
// So this is mostly a change detector test to catch regressions.
func TestBuild(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// The name of the Fun to test against want.
		fun string
		// Lines must begin with 4 tabs.
		// Lines beginning with whitespace and then // are ignored.
		// If the first line of want, after trimming
		// is "module" the whole module is compared.
		// Otherwise just the 0th Fun is compared.
		want string
	}{
		{
			name: "return Nil type",
			src: `
				func [foo ^Nil | ^{}]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						return
			`,
		},
		{
			name: "return empty and-type with empty fields",
			src: `
				func [foo ^empty | ^{}]
				type empty {x: Nil y: Nil}
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						return
			`,
		},
		{
			name: "return int literal",
			src: `
				func [foo ^Int | ^123]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 123
						$1 := arg(0)
						store($1, $0)
						return
			`,
		},
		{
			name: "return float literal",
			src: `
				func [foo ^Float | ^3.14]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Float&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 3.14
						$1 := arg(0)
						store($1, $0)
						return
			`,
		},
		{
			name: "return empty module variable",
			src: `
				func [foo ^Nil | ^n]
				val n Nil := [{}]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						return
			`,
		},
		{
			name: "return simple module variable",
			src: `
				func [foo ^Int | ^i]
				val i Int := [5]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := global(i)
						$1 := load($0)
						$2 := arg(0)
						store($2, $1)
						return
			`,
		},
		{
			name: "return composite module variable",
			src: `
				func [foo ^String | ^s]
				val s String := ["Hello, World"]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 String&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := global(s)
						$1 := arg(0)
						copy($1, $0, String)
						return
			`,
		},
		{
			name: "return empty parm",
			src: `
				func [foo: n Nil ^Nil | ^n]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						return
			`,
		},
		{
			name: "return simple parm",
			src: `
				func [foo: i Int ^Int | ^i]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [i] Int
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						$1 := arg(0 [i])
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := arg(1)
						store($3, $2)
						return
			`,
		},
		{
			name: "return composite parm",
			src: `
				func [foo: s String ^String | ^s]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [s] String& (value)
						1 String&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [s])
						$1 := arg(1)
						copy($1, $0, String)
						return
			`,
		},
		{
			name: "return empty local",
			src: `
				func [foo ^Nil | n Nil := {}. ^n]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						// We allocate a Nil, in case we need its address.
						// In this case we don't; the opt pass will remove it.
						$0 := alloc(Nil)
						jmp 1
					1:
						[in: 0] [out:]
						return
			`,
		},
		{
			name: "return simple local",
			src: `
				func [foo ^Int | i := 123. ^i]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out:]
						$1 := 123
						store($0, $1)
						$2 := load($0)
						$3 := arg(0)
						store($3, $2)
						return
			`,
		},
		{
			name: "return composite local",
			src: `
				func [foo ^String | s := "Hello, World". ^s]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 String&
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						$1 := alloc(String)
						jmp 1
					1:
						[in: 0] [out:]
						string($1, string1)
						copy($0, $1, String)
						$2 := arg(0)
						copy($2, $0, String)
						return
			`,
		},
		{
			name: "return simple field",
			src: `
				meth Foo [foo ^Int | ^x]
				type Foo {x: Int}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [self] #test Foo&
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(#test Foo&)
						$1 := arg(0 [self])
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [x]
						$4 := load($3)
						$5 := arg(1)
						store($5, $4)
						return
			`,
		},
		{
			name: "return composite field",
			src: `
				meth Foo [foo ^String | ^x]
				type Foo {x: String}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [self] #test Foo&
						1 String&
					0:
						[in:] [out: 1]
						$0 := alloc(#test Foo&)
						$1 := arg(0 [self])
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [x]
						$4 := arg(1)
						copy($4, $3, String)
						return
				`,
		},
		{
			name: "dead code after return",
			src: `
				func [foo ^Int | ^123. x := 5. ^x]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out:]
						$1 := 123
						$2 := arg(0)
						store($2, $1)
						return
					// Basic block 2 has no incoming edges.
					// It will be removed by the opt pass.
					2:
						[in:] [out:]
						$3 := 5
						store($0, $3)
						$4 := load($0)
						$5 := arg(0)
						store($5, $4)
						return

			`,
		},
		{
			name: "0-ary function call",
			src: `
				func [foo | f]
				func [f]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						call function1()
						return
			`,
		},
		{
			name: "3-ary function call",
			src: `
				func [foo | f: 1 b: 2 b: 3]
				func [f: _ Int b: _ Int b: _ Int]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 1
						$1 := 2
						$2 := 3
						call function1($0, $1, $2)
						return
			`,
		},
		{
			name: "function call with composite arg",
			src: `
				func [foo | f: "Hello, World"]
				func [f: _ String]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						$1 := alloc(String)
						jmp 1
					1:
						[in: 0] [out:]
						string($0, string1)
						copy($1, $0, String)
						call function2($1)
						return
			`,
		},
		{
			name: "function call with empty args",
			src: `
				func [foo | f: 1 b: {} b: {} q: 2]
				func [f: _ Int b: _ Nil b: _ Nil q: _ Int]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 1
						$1 := 2
						call function1($0, $1)
						return
			`,
		},
		{
			name: "0-ary method call",
			src: `
				func [foo | 123 m]
				meth Int [m]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$1 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 123
						store($1, $0)
						call function1($1)
						return
			`,
		},
		{
			name: "3-ary method call",
			src: `
				func [foo | 123 f: 1 b: 2 b: 3]
				meth Int [f: _ Int b: _ Int b: _ Int]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$1 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 123
						store($1, $0)
						$2 := 1
						$3 := 2
						$4 := 3
						call function1($1, $2, $3, $4)
						return
			`,
		},
		{
			name: "empty-type receiver method call",
			src: `
				func [main |
					n Nil := {}.
					n foo.
				]
				meth Nil [foo]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(Nil)
						jmp 1
					1:
						[in: 0] [out:]
						call function1()
						return
			`,
		},
		{
			name: "pass by value",
			src: `
				func [foo: s String& | f: s]
				func [f: s String]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [s] String&
					0:
						[in:] [out: 1]
						$0 := alloc(String&)
						$1 := arg(0 [s])
						store($0, $1)
						$3 := alloc(String)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						copy($3, $2, String)
						call function1($3)
						return
			`,
		},
		{
			name: "pass by reference",
			src: `
				func [foo: s String& | f: s]
				func [f: s String&]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [s] String&
					0:
						[in:] [out: 1]
						$0 := alloc(String&)
						$1 := arg(0 [s])
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						call function1($2)
						return
			`,
		},
		{
			name: "virtual call",
			src: `
				func [foo: v Fooer | v f: 1 b: "Hello" b: 3]
				type Fooer {[f: Int b: String b: Int]}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [v] #test Fooer& (value)
					0:
						[in:] [out: 1]
						$2 := alloc(String)
						$3 := alloc(String)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [v])
						$1 := 1
						string($2, string1)
						copy($3, $2, String)
						$4 := 3
						virt call $0.0 [f:b:b:]($0, $1, $3, $4)
						return
			`,
		},
		{
			name: "return empty function result",
			src: `
				func [foo ^Nil | ^f]
				func [f ^Nil]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						call function1()
						return
			`,
		},
		{
			name: "return simple function result",
			src: `
				func [foo ^Int | ^f]
				func [f ^Int]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out:]
						call function1($0)
						$1 := load($0)
						$2 := arg(0)
						store($2, $1)
						return
			`,
		},
		{
			name: "return complex function result",
			src: `
				func [foo ^String | ^f]
				func [f ^String]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 String&
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						jmp 1
					1:
						[in: 0] [out:]
						call function1($0)
						$1 := arg(0)
						copy($1, $0, String)
						return
			`,
		},
		{
			name: "return block literal",
			src: `
				func [foo ^Int Fun | ^[3]]
			`,
			fun: "",
			want: `
				block1
					parms:
						0 #test $Block0&
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := 3
						$3 := arg(1)
						store($3, $2)
						return
				function0
					parms:
						0 Int Fun&
					0:
						[in:] [out: 1]
						$1 := alloc(#test $Block0)
						$2 := alloc(Int Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0)
						and($1, {$0})
						virt($2, $1, {block1})
						$3 := arg(0)
						copy($3, $2, Int Fun)
						return
				function2
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "binary op",
			src: `
				func [foo: i Int ^Int | ^i + 123]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [i] Int
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						$1 := arg(0 [i])
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := 123
						$4 := $2 + $3
						$5 := arg(1)
						store($5, $4)
						return
			`,
		},
		{
			name: "unary op",
			src: `
				func [foo: i Int ^Int | ^i neg]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [i] Int
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						$1 := arg(0 [i])
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := -$2
						$4 := arg(1)
						store($4, $3)
						return
			`,
		},
		{
			name: "num convert op",
			src: `
				func [foo: i Int ^Float | ^i asFloat]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [i] Int
						1 Float&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						$1 := arg(0 [i])
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := Float($2)
						$4 := arg(1)
						store($4, $3)
						return
			`,
		},
		{
			name: "string atByte",
			src: `
				func [foo: a String ^UInt8 | ^a atByte: 123]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [a] String& (value)
						1 UInt8&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [a])
						$1 := 123
						$2 := $0[$1]
						$3 := arg(1)
						store($3, $2)
						return
			`,
		},
		{
			name: "array simple load",
			src: `
				func [foo: a Int Array ^Int | ^a at: 123]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [a] Int Array& (value)
						1 Int&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [a])
						$1 := 123
						$2 := $0[$1]
						$3 := load($2)
						$4 := arg(1)
						store($4, $3)
						return
			`,
		},
		{
			name: "array composite load",
			src: `
				func [foo: a String Array ^String | ^a at: 123]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [a] String Array& (value)
						1 String&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [a])
						$1 := 123
						$2 := $0[$1]
						$3 := arg(1)
						copy($3, $2, String)
						return
			`,
		},
		{
			name: "array simple store",
			src: `
				func [foo: a Int Array | a at: 123 put: 6]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [a] Int Array& (value)
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [a])
						$1 := 123
						$2 := 6
						$3 := $0[$1]
						store($3, $2)
						return
			`,
		},
		{
			name: "array composite store",
			src: `
				func [foo: a String Array | a at: 123 put: "Hello, World"]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [a] String Array& (value)
					0:
						[in:] [out: 1]
						$2 := alloc(String)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [a])
						$1 := 123
						string($2, string1)
						$3 := $0[$1]
						copy($3, $2, String)
						return
			`,
		},
		{
			name: "string byteSize",
			src: `
				Func [size: a String ^Int | ^a byteSize]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [a] String& (value)
						1 Int&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [a])
						$1 := size($0)
						$2 := arg(1)
						store($2, $1)
						return
			`,
		},
		{
			name: "array size",
			src: `
				Func [size: a Int Array ^Int | ^a size]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [a] Int Array& (value)
						1 Int&
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [a])
						$1 := size($0)
						$2 := arg(1)
						store($2, $1)
						return
			`,
		},
		{
			name: "make array",
			src: `
				func [foo ^Int Array | ^{5; 6; 7}]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int Array&
					0:
						[in:] [out: 1]
						$3 := alloc(Int Array)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						$1 := 6
						$2 := 7
						array($3, {$0, $1, $2})
						$4 := arg(0)
						copy($4, $3, Int Array)
						return
			`,
		},
		{
			name: "make array non-simple element type",
			src: `
				func [foo ^String Array | ^{"a"; "b"}]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 String Array&
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						$1 := alloc(String)
						$2 := alloc(String Array)
						jmp 1
					1:
						[in: 0] [out:]
						string($0, string1)
						string($1, string2)
						array($2, {*$0, *$1})
						$3 := arg(0)
						copy($3, $2, String Array)
						return
			`,
		},
		{
			name: "make empty array",
			src: `
				func [foo ^Int Array | ^{}]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int Array&
					0:
						[in:] [out: 1]
						$0 := alloc(Int Array)
						jmp 1
					1:
						[in: 0] [out:]
						array($0, {})
						$1 := arg(0)
						copy($1, $0, Int Array)
						return
			`,
		},
		{
			name: "make nil array",
			src: `
				func [foo ^Nil Array | ^{{}; {}; {}}]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Nil Array&
					0:
						[in:] [out: 1]
						$0 := alloc(Nil Array)
						jmp 1
					1:
						[in: 0] [out:]
						array($0, {})
						$1 := arg(0)
						copy($1, $0, Nil Array)
						return
			`,
		},
		{
			name: "make slice",
			src: `
				func [foo: a Int Array ^Int Array| ^a from: 5 to: 10]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [a] Int Array& (value)
						1 Int Array&
					0:
						[in:] [out: 1]
						$3 := alloc(Int Array)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [a])
						$1 := 5
						$2 := 10
						slice($3, $0[$1:$2])
						$4 := arg(1)
						copy($4, $3, Int Array)
						return
			`,
		},
		{
			name: "return string",
			src: `
				func [foo ^String | ^"Hello, World!"]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 String&
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						jmp 1
					1:
						[in: 0] [out:]
						string($0, string1)
						$1 := arg(0)
						copy($1, $0, String)
						return
			`,
		},
		{
			name: "make and",
			src: `
				func [foo ^(Int, Int) Pair | ^{x: 5 y: 6}]
				type (X, Y) Pair {x: X y: Y}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 (Int, Int) #test Pair&
					0:
						[in:] [out: 1]
						$2 := alloc((Int, Int) #test Pair)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						$1 := 6
						and($2, {x: $0 y: $1})
						$3 := arg(0)
						copy($3, $2, (Int, Int) #test Pair)
						return
			`,
		},
		{
			name: "make and non-simple field type",
			src: `
				func [foo ^(String, String) Pair | ^{x: "a" y: "b"}]
				type (X, Y) Pair {x: X y: Y}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 (String, String) #test Pair&
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						$1 := alloc(String)
						$2 := alloc((String, String) #test Pair)
						jmp 1
					1:
						[in: 0] [out:]
						string($0, string1)
						string($1, string2)
						and($2, {x: *$0 y: *$1})
						$3 := arg(0)
						copy($3, $2, (String, String) #test Pair)
						return
			`,
		},
		{
			name: "make and with Nil last field",
			src: `
				func [foo ^(Int, Nil) Pair | ^{x: 5 y: {}}]
				type (X, Y) Pair {x: X y: Y}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 (Int, Nil) #test Pair&
					0:
						[in:] [out: 1]
						$1 := alloc((Int, Nil) #test Pair)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						and($1, {x: $0 y: {}})
						$2 := arg(0)
						copy($2, $1, (Int, Nil) #test Pair)
						return
			`,
		},
		{
			name: "make and with Nil first field",
			src: `
				func [foo ^(Nil, Int) Pair | ^{x: {} y: 5}]
				type (X, Y) Pair {x: X y: Y}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 (Nil, Int) #test Pair&
					0:
						[in:] [out: 1]
						$1 := alloc((Nil, Int) #test Pair)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						and($1, {x: {} y: $0})
						$2 := arg(0)
						copy($2, $1, (Nil, Int) #test Pair)
						return
			`,
		},
		{
			name: "make and with Nil middle field",
			src: `
				func [foo ^(Int, Nil, Int) Triple | ^{x: 5 y: {} z: 6}]
				type (X, Y, Z) Triple {x: X y: Y z: Z}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 (Int, Nil, Int) #test Triple&
					0:
						[in:] [out: 1]
						$2 := alloc((Int, Nil, Int) #test Triple)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						$1 := 6
						and($2, {x: $0 y: {} z: $1})
						$3 := arg(0)
						copy($3, $2, (Int, Nil, Int) #test Triple)
						return
			`,
		},
		{
			name: "make or with value",
			src: `
				func [foo ^Int? | ^{some: 5}]
				type T? {none | some: T}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int #test ? &
					0:
						[in:] [out: 1]
						$1 := alloc(Int #test ?)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						or($1, {1=some: $0})
						$2 := arg(0)
						copy($2, $1, Int #test ?)
						return
			`,
		},
		{
			name: "make or with non-simple value",
			src: `
				func [foo ^String? | ^{some: "a"}]
				type T? {none | some: T}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 String #test ? &
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						$1 := alloc(String #test ?)
						jmp 1
					1:
						[in: 0] [out:]
						string($0, string1)
						or($1, {1=some: *$0})
						$2 := arg(0)
						copy($2, $1, String #test ?)
						return
			`,
		},
		{
			name: "make or with nil",
			src: `
				func [foo ^Nil? | ^{some: {}}]
				type T? {none | some: T}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Nil #test ? &
					0:
						[in:] [out: 1]
						$0 := alloc(Nil #test ?)
						jmp 1
					1:
						[in: 0] [out:]
						or($0, {1=some:})
						$1 := arg(0)
						copy($1, $0, Nil #test ?)
						return
			`,
		},
		{
			name: "make or without value",
			src: `
				func [foo ^Int? | ^{none}]
				type T? {none | some: T}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int #test ? &
					0:
						[in:] [out: 1]
						$0 := alloc(Int #test ?)
						jmp 1
					1:
						[in: 0] [out:]
						or($0, {0=none})
						$1 := arg(0)
						copy($1, $0, Int #test ?)
						return
			`,
		},
		{
			name: "make virtual",
			src: `
				func [foo ^Fooer | ^5]
				type Fooer {[foo]}
				meth Int [foo]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 #test Fooer&
					0:
						[in:] [out: 1]
						$1 := alloc(Int)
						$2 := alloc(#test Fooer)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						store($1, $0)
						virt($2, $1, {function1})
						$3 := arg(0)
						copy($3, $2, #test Fooer)
						return
			`,
		},
		{
			name: "dedup same string literals",
			src: `
				func [foo ^Bool | ^"a" = "a"]
				meth String [= _ String& ^Bool]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Bool&
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						$1 := alloc(String)
						$2 := alloc(Bool)
						jmp 1
					1:
						[in: 0] [out:]
						// Both strirgs use string 1.e
						string($0, string1)
						string($1, string1)
						call function2($0, $1, $2)
						$3 := load($2)
						$4 := arg(0)
						store($4, $3)
						return
			`,
		},
		{
			name: "case method returns empty type",
			src: `
				func [foo |
					intOpt Int? := {none}.
					intOpt ifNone: [] ifSome: [:t |]
				]
				type T? {none | some: T}
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(Int #test ?)
						$1 := alloc(Int #test ?)
						$2 := alloc(#test $Block0)
						$3 := alloc(Nil Fun)
						$4 := alloc(#test $Block1)
						$5 := alloc((Int, Nil) Fun)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						or($1, {0=none})
						copy($0, $1, Int #test ?)
						and($2, {})
						virt($3, $2, {block1})
						and($4, {})
						virt($5, $4, {block2})
						$6 := tag($0)
						switch $6 [none 2] [some: 3]
					2:
						[in: 1] [out: 4]
						virt call $3.0($3)
						jmp 4
					3:
						[in: 1] [out: 4]
						$7 := $0.1 [some:]
						$8 := load($7)
						virt call $5.0($5, $8)
						jmp 4
					4:
						[in: 2 3] [out:]
						return
			`,
		},
		{
			name: "case method returns simple type",
			src: `
				func [foo ^Int |
					intOpt Int? := {none}.
					^intOpt ifNone: [0] ifSome: [:t | t]
				]
				type T? {none | some: T}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int #test ?)
						$1 := alloc(Int #test ?)
						$3 := alloc(#test $Block0)
						$4 := alloc(Int Fun)
						$6 := alloc(#test $Block1)
						$7 := alloc((Int, Int) Fun)
						$8 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						or($1, {0=none})
						copy($0, $1, Int #test ?)
						// TODO: blocks that do not far-return needn't capture the return location.
						$2 := arg(0)
						and($3, {$2})
						virt($4, $3, {block1})
						$5 := arg(0)
						and($6, {$5})
						virt($7, $6, {block2})
						$9 := tag($0)
						switch $9 [none 2] [some: 3]
					2:
						[in: 1] [out: 4]
						virt call $4.0($4, $8)
						jmp 4
					3:
						[in: 1] [out: 4]
						$10 := $0.1 [some:]
						$11 := load($10)
						virt call $7.0($7, $11, $8)
						jmp 4
					4:
						[in: 2 3] [out:]
						$12 := load($8)
						$13 := arg(0)
						store($13, $12)
						return
			`,
		},
		{
			name: "case method returns composite type",
			src: `
				func [foo ^String |
					intOpt Int? := {none}.
					^intOpt ifNone: ["Bye"] ifSome: [:t | "Hi"]
				]
				type T? {none | some: T}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 String&
					0:
						[in:] [out: 1]
						$0 := alloc(Int #test ?)
						$1 := alloc(Int #test ?)
						$3 := alloc(#test $Block0)
						$4 := alloc(String Fun)
						$6 := alloc(#test $Block1)
						$7 := alloc((Int, String) Fun)
						$8 := alloc(String)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						or($1, {0=none})
						copy($0, $1, Int #test ?)
						// TODO: blocks that do not far-return needn't capture the return location.
						$2 := arg(0)
						and($3, {$2})
						virt($4, $3, {block1})
						$5 := arg(0)
						and($6, {$5})
						virt($7, $6, {block3})
						$9 := tag($0)
						switch $9 [none 2] [some: 3]
					2:
						[in: 1] [out: 4]
						virt call $4.0($4, $8)
						jmp 4
					3:
						[in: 1] [out: 4]
						$10 := $0.1 [some:]
						$11 := load($10)
						virt call $7.0($7, $11, $8)
						jmp 4
					4:
						[in: 2 3] [out:]
						$12 := arg(0)
						copy($12, $8, String)
						return
			`,
		},
		{
			name: "block return",
			src: `
				func [foo ^Int Fun | ^[5]]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int Fun&
					0:
						[in:] [out: 1]
						$1 := alloc(#test $Block0)
						$2 := alloc(Int Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0)
						and($1, {$0})
						virt($2, $1, {block1})
						$3 := arg(0)
						copy($3, $2, Int Fun)
						return
			`,
		},
		{
			name: "block far return",
			src: `
				func [foo ^Int | [^5]. ^3]
			`,
			fun: "",
			want: `
				block1
					parms:
						0 #test $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := 5
						$3 := load($0)
						$4 := $3.0
						$5 := load($4)
						store($5, $2)
						far return
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						$1 := alloc(#test $Block0)
						$2 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0)
						and($1, {$0})
						virt($2, $1, {block1})
						$3 := 3
						$4 := arg(0)
						store($4, $3)
						return
				function2
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "block nil far return",
			src: `
				func [foo | [^{}]]
			`,
			fun: "",
			want: `
				block1
					parms:
						0 #test $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						far return
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0)
						$1 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out:]
						and($0, {})
						virt($1, $0, {block1})
						return
				function2
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			// This is testing a regression where makeblock
			// inside a nested block,
			// inside a function with no return value
			// would try to access beyond the end of the Type.Fields
			// to set a non-existant capture
			// for the return slot.
			name: "makeblock in a nested block in a Nil return fun",
			src: `
				func [foo | [[3]]]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block1)
						$1 := alloc(Int Fun Fun)
						jmp 1
					1:
						[in: 0] [out:]
						and($0, {})
						virt($1, $0, {block1})
						return
			`,
		},
		{
			name: "block empty capture",
			src: `
				func [foo: n Nil ^Nil Fun | ^[n]]
			`,
			fun: "",
			want: `
				block1
					parms:
						0 #test $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						return
				function0
					parms:
						0 Nil Fun&
					0:
						[in:] [out: 1]
						$1 := alloc(#test $Block0)
						$2 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0)
						and($1, {n: {} $0})
						virt($2, $1, {block1})
						$3 := arg(0)
						copy($3, $2, Nil Fun)
						return
				function2
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "block capture parm",
			src: `
				func [foo: i Int ^Int | [^i]. ^3]
			`,
			fun: "",
			want: `
				block1
					parms:
						0 #test $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [i]
						$4 := load($3)
						$5 := load($4)
						$6 := load($0)
						$7 := $6.1
						$8 := load($7)
						store($8, $5)
						far return
				function0
					parms:
						0 [i] Int
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						$1 := arg(0 [i])
						store($0, $1)
						$3 := alloc(#test $Block0)
						$4 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := arg(1)
						and($3, {i: $0 $2})
						virt($4, $3, {block1})
						$5 := 3
						$6 := arg(1)
						store($6, $5)
						return
				function2
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "block capture local",
			src: `
				func [foo ^Int | i := 5. [^i]. ^3]
			`,
			fun: "",
			want: `
				block1
					parms:
						0 #test $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [i]
						$4 := load($3)
						$5 := load($4)
						$6 := load($0)
						$7 := $6.1
						$8 := load($7)
						store($8, $5)
						far return
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						$3 := alloc(#test $Block0)
						$4 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$1 := 5
						store($0, $1)
						$2 := arg(0)
						and($3, {i: $0 $2})
						virt($4, $3, {block1})
						$5 := 3
						$6 := arg(0)
						store($6, $5)
						return
				function2
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "block capture field",
			src: `
				meth Point [foo ^Int | [^x]. ^3]
				type Point {x: Int y: Int}
			`,
			fun: "",
			want: `
				block1
					parms:
						0 #test $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [x]
						$4 := load($3)
						$5 := load($4)
						$6 := load($0)
						$7 := $6.1
						$8 := load($7)
						store($8, $5)
						far return
				function0
					parms:
						0 [self] #test Point&
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(#test Point&)
						$1 := arg(0 [self])
						store($0, $1)
						$5 := alloc(#test $Block0)
						$6 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [x]
						$4 := arg(1)
						and($5, {x: $3 $4})
						virt($6, $5, {block1})
						$7 := 3
						$8 := arg(1)
						store($8, $7)
						return
				function2
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "block capture nested block parm",
			src: `
				func [foo ^Int | [:i Int | [^i]]. ^3]
			`,
			fun: "",
			want: `
				block2
					parms:
						0 #test $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [i]
						$4 := load($3)
						$5 := load($4)
						$6 := load($0)
						$7 := $6.1
						$8 := load($7)
						store($8, $5)
						far return
				block1
					parms:
						0 #test $Block1&
						1 [i] Int
						2 Nil Fun&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block1&)
						$1 := alloc(Int)
						$2 := arg(0)
						store($0, $2)
						$3 := arg(1 [i])
						store($1, $3)
						$7 := alloc(#test $Block0)
						$8 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$4 := load($0)
						$5 := $4.0
						$6 := load($5)
						and($7, {i: $1 $6})
						virt($8, $7, {block2})
						$9 := arg(2)
						copy($9, $8, Nil Fun)
						return
				function0
					parms:
						0 Int&
					0:
						[in:] [out: 1]
						$1 := alloc(#test $Block1)
						$2 := alloc((Int, Nil Fun) Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0)
						and($1, {$0})
						virt($2, $1, {block1})
						$3 := 3
						$4 := arg(0)
						store($4, $3)
						return
				function3
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "block capture capture",
			src: `
				func [foo: i Int ^Int | [[^i]]. ^3]
			`,
			fun: "",
			want: `
				block2
					parms:
						0 #test $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block0&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [i]
						$4 := load($3)
						$5 := load($4)
						$6 := load($0)
						$7 := $6.1
						$8 := load($7)
						store($8, $5)
						far return
				block1
					parms:
						0 #test $Block1&
						1 Nil Fun&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block1&)
						$1 := arg(0)
						store($0, $1)
						$8 := alloc(#test $Block0)
						$9 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := load($0)
						$3 := $2.0 [i]
						$4 := load($3)
						$5 := load($0)
						$6 := $5.1
						$7 := load($6)
						and($8, {i: $4 $7})
						virt($9, $8, {block2})
						$10 := arg(1)
						copy($10, $9, Nil Fun)
						return
				function0
					parms:
						0 [i] Int
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Int)
						$1 := arg(0 [i])
						store($0, $1)
						$3 := alloc(#test $Block1)
						$4 := alloc(Nil Fun Fun)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := arg(1)
						and($3, {i: $0 $2})
						virt($4, $3, {block1})
						$5 := 3
						$6 := arg(1)
						store($6, $5)
						return
				function3
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "bool is an enum",
			src: `
				func [foo ^Bool | ^{true}]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Bool&
					0:
						[in:] [out: 1]
						$0 := alloc(Bool)
						jmp 1
					1:
						[in: 0] [out:]
						$1 := 1
						store($0, $1)
						$2 := load($0)
						$3 := arg(0)
						store($3, $2)
						return
			`,
		},
		{
			name: "enum type",
			src: `
				func [foo ^Num | ^{four}]
				type Num {zero|one|two|three|four|five|six}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 #test Num&
					0:
						[in:] [out: 1]
						$0 := alloc(#test Num)
						jmp 1
					1:
						[in: 0] [out:]
						$1 := 4
						store($0, $1)
						$2 := load($0)
						$3 := arg(0)
						store($3, $2)
						return
			`,
		},
		{
			name: "bool switch",
			src: `
				func [foo | b Bool := {false}. b ifTrue: [] ifFalse: []]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(Bool)
						$1 := alloc(Bool)
						$4 := alloc(#test $Block0)
						$5 := alloc(Nil Fun)
						$6 := alloc(#test $Block1)
						$7 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						$2 := 0
						store($1, $2)
						$3 := load($1)
						store($0, $3)
						and($4, {})
						virt($5, $4, {block1})
						and($6, {})
						virt($7, $6, {block2})
						$8 := load($0)
						switch $8 [true 2] [false 3]
					2:
						[in: 1] [out: 4]
						virt call $5.0($5)
						jmp 4
					3:
						[in: 1] [out: 4]
						virt call $7.0($7)
						jmp 4
					4:
						[in: 2 3] [out:]
						return
			`,
		},
		{
			name: "enum switch",
			src: `
				func [foo | n Num := {zero}. n ifZero: [] ifOne: []]
				type Num {zero|one}
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(#test Num)
						$1 := alloc(#test Num)
						$4 := alloc(#test $Block0)
						$5 := alloc(Nil Fun)
						$6 := alloc(#test $Block1)
						$7 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						$2 := 0
						store($1, $2)
						$3 := load($1)
						store($0, $3)
						and($4, {})
						virt($5, $4, {block1})
						and($6, {})
						virt($7, $6, {block2})
						$8 := load($0)
						switch $8 [zero 2] [one 3]
					2:
						[in: 1] [out: 4]
						virt call $5.0($5)
						jmp 4
					3:
						[in: 1] [out: 4]
						virt call $7.0($7)
						jmp 4
					4:
						[in: 2 3] [out:]
						return
			`,
		},
		{
			name: "bool returning op",
			src: `
				func [foo | 1 < 2 ifTrue: [] ifFalse: []]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$1 := alloc(Int)
						$5 := alloc(Bool)
						$6 := alloc(#test $Block0)
						$7 := alloc(Nil Fun)
						$8 := alloc(#test $Block1)
						$9 := alloc(Nil Fun)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						$0 := 1
						store($1, $0)
						$2 := load($1)
						$3 := 2
						$4 := $2 < $3
						store($5, $4)
						and($6, {})
						virt($7, $6, {block1})
						and($8, {})
						virt($9, $8, {block2})
						$10 := load($5)
						switch $10 [true 2] [false 3]
					2:
						[in: 1] [out: 4]
						virt call $7.0($7)
						jmp 4
					3:
						[in: 1] [out: 4]
						virt call $9.0($9)
						jmp 4
					4:
						[in: 2 3] [out:]
						return
			`,
		},
		{
			name: "assign to capture",
			src: `
				func [foo ^Int |
					x := 1.
					true ifTrue: [] ifFalse: [ x := 3 ].
					^x
				]
			`,
			fun: "block3",
			want: `
				block3
					parms:
						0 #test $Block1&
					0:
						[in:] [out: 1]
						$0 := alloc(#test $Block1&)
						$1 := arg(0)
						store($0, $1)
						jmp 1
					1:
						[in: 0] [out:]
						$2 := 3
						$3 := load($0)
						$4 := $3.0 [x]
						$5 := load($4)
						store($5, $2)
						return
			`,
		},
		{
			name: "topo sort by calls",
			src: `
				func [foo | bar]
				func [bar | baz]
				func [baz | qux]
				func [qux | baz]
			`,
			fun: "",
			want: `
				function3
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						call function2()
						return
				function2
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						call function3()
						return
				function1
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						call function2()
						return
				function0
					parms:
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						call function1()
						return
				function4
					parms:
					0:
						[in:] [out:]
						return
			`,
		},
		{
			name: "newArray:init simple type",
			src: `
				func [foo ^Int Array | ^newArray: 5 init: [:_ | 0]]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Int Array&
					0:
						[in:] [out: 1]
						$0 := alloc(Int Array)
						$3 := alloc(#test $Block0)
						$4 := alloc((Int, Int) Fun)
						$5 := alloc(Int)
						$9 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out: 2]
						$1 := 5
						$2 := arg(0)
						and($3, {$2})
						virt($4, $3, {block1})
						array($0, $1)
						$6 := 0
						store($5, $6)
						jmp 2
					2:
						[in: 1 3] [out: 3 4]
						$7 := load($5)
						$8 := $7 < $1
						switch $8 [true 3] [false 4]
					3:
						[in: 2] [out: 2]
						virt call $4.0($4, $7, $9)
						$10 := $0[$7]
						$11 := load($9)
						store($10, $11)
						$12 := 1
						$13 := $7 + $12
						store($5, $13)
						jmp 2
					4:
						[in: 2] [out:]
						$14 := arg(0)
						copy($14, $0, Int Array)
						return
			`,
		},
		{
			name: "newArray:init composite type",
			src: `
				func [foo ^String Array | ^newArray: 5 init: [:_ | ""]]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 String Array&
					0:
						[in:] [out: 1]
						$0 := alloc(String Array)
						$3 := alloc(#test $Block0)
						$4 := alloc((Int, String) Fun)
						$5 := alloc(Int)
						$9 := alloc(String)
						jmp 1
					1:
						[in: 0] [out: 2]
						$1 := 5
						$2 := arg(0)
						and($3, {$2})
						virt($4, $3, {block1})
						array($0, $1)
						$6 := 0
						store($5, $6)
						jmp 2
					2:
						[in: 1 3] [out: 3 4]
						$7 := load($5)
						$8 := $7 < $1
						switch $8 [true 3] [false 4]
					3:
						[in: 2] [out: 2]
						virt call $4.0($4, $7, $9)
						$10 := $0[$7]
						copy($10, $9, String)
						$11 := 1
						$12 := $7 + $11
						store($5, $12)
						jmp 2
					4:
						[in: 2] [out:]
						$13 := arg(0)
						copy($13, $0, String Array)
						return
			`,
		},
		{
			name: "panic",
			src: `
				func [foo | panic: "bar"]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(String)
						jmp 1
					1:
						[in: 0] [out:]
						string($0, string1)
						panic($0)
					2:
						[in:] [out:]
						return
			`,
		},
		{
			// This is testing for regression of a bug
			// where building a Convert didn't return a new BBlk,
			// so calling a conversion on a case method paniced.
			name: "conversion with case-receiver",
			src: `
				func [foo | (true ifTrue: [5] ifFalse: [6]) + 1]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(Bool)
						$2 := alloc(Bool)
						$3 := alloc(#test $Block0)
						$4 := alloc(Int Fun)
						$5 := alloc(#test $Block1)
						$6 := alloc(Int Fun)
						$7 := alloc(Int)
						$10 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						call function1($0)
						$1 := load($0)
						store($2, $1)
						and($3, {})
						virt($4, $3, {block2})
						and($5, {})
						virt($6, $5, {block3})
						$8 := load($2)
						switch $8 [true 2] [false 3]
					2:
						[in: 1] [out: 4]
						virt call $4.0($4, $7)
						jmp 4
					3:
						[in: 1] [out: 4]
						virt call $6.0($6, $7)
						jmp 4
					4:
						[in: 2 3] [out:]
						$9 := load($7)
						store($10, $9)
						$11 := load($10)
						$12 := 1
						$13 := $11 + $12
						return
				`,
		},
		{
			name: "MakeVirt of an empty object",
			src: `
				func [main |
					e Empty := {}.
					_ Fooer := e.
				]
				type Fooer {[foo]}
				type Empty {}
				meth Empty [foo]
			`,
			fun: "function0",
			want: `
				function0
					parms:
					0:
						[in:] [out: 1]
						$0 := alloc(#test Empty)
						$1 := alloc(#test Fooer)
						$2 := alloc(#test Fooer)
						jmp 1
					1:
						[in: 0] [out:]
						virt($2, {function1})
						copy($1, $2, #test Fooer)
						return
			`,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if strings.HasPrefix(test.name, "SKIP") {
				t.Skip()
			}
			p := ast.NewParser("/test/test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Fatalf("failed to parse source: %s", err)
			}
			typesMod, errs := types.Check(p.Mod(), types.Config{})
			if len(errs) > 0 {
				t.Fatalf("failed to check the source: %v", errs)
			}
			basicMod := Build(typesMod)
			want := trimLeadingTestIndent(test.want)
			var s strings.Builder
			if test.fun == "" {
				basicMod.buildString(&s, false)
			} else {
				fun := findTestFun(basicMod, test.fun)
				fun.buildString(&s, false)
			}
			got := strings.TrimSpace(s.String())
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("got\n%s\nexpected\n%s\ndiff\n%s", got, want, diff)
			}
		})
	}
}

func findTestFun(mod *Mod, name string) *Fun {
	for _, fun := range mod.Funs {
		if fun.name() == name {
			return fun
		}
	}
	panic(fmt.Sprintf("fun %s not found", name))
}

func trimLeadingTestIndent(src string) string {
	var s strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(src))
	for scanner.Scan() {
		if strings.HasPrefix(strings.TrimSpace(scanner.Text()), "//") {
			continue
		}
		s.WriteString(strings.TrimPrefix(scanner.Text(), "\t\t\t\t"))
		s.WriteRune('\n')

	}
	if err := scanner.Err(); err != nil {
		panic(err.Error())
	}
	return strings.TrimSpace(s.String())
}

func TestHasFarRet(t *testing.T) {
	tests := []struct {
		src    string
		has    []string
		hasOpt []string
	}{
		{
			src:    `func [foo |]`,
			has:    nil,
			hasOpt: nil,
		},
		{
			src:    `func [foo ^Int | ^5]`,
			has:    nil,
			hasOpt: nil,
		},
		{
			src:    `func [foo ^Int | ^[1 + 1] value]`,
			has:    nil,
			hasOpt: nil,
		},
		{
			src:    `func [foo ^Int | [^1 + 1] value. ^2]`,
			has:    []string{"function0"},
			hasOpt: nil,
		},
		{
			src:    `func [foo ^Int | [[[^1 + 1] value] value] value. ^2]`,
			has:    []string{"function0"},
			hasOpt: nil,
		},
		{
			src: `
				func [foo ^Int |
					x := [^3].
					true ifTrue: [x := [^2]] ifFalse: [].
					x value.
					^2.
				]`,
			has: []string{"function0"},
			// Since the blocks calls cannot be inlined,
			// these keep the far ret even after optimization.
			hasOpt: []string{"function0"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.src, func(t *testing.T) {
			p := ast.NewParser("#test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			typesMod, errs := types.Check(p.Mod(), types.Config{})
			if len(errs) > 0 {
				t.Fatalf("failed to check: %v", errs)
			}
			basicMod := Build(typesMod)

			var has []string
			for _, f := range basicMod.Funs {
				if f.CanFarRet {
					has = append(has, f.name())
				}
			}
			sort.Strings(has)
			if !reflect.DeepEqual(test.has, has) {
				t.Errorf("unopt: got %v, want %v", has, test.has)
			}

			Optimize(basicMod)
			has = nil
			for _, f := range basicMod.Funs {
				if f.CanFarRet {
					has = append(has, f.name())
				}
			}
			sort.Strings(has)
			if !reflect.DeepEqual(test.hasOpt, has) {
				t.Errorf("opt: got %v, want %v", has, test.has)
			}
		})
	}
}
