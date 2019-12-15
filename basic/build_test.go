package basic

import (
	"bufio"
	"fmt"
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
						0 [self] Foo&
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Foo&)
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
						0 [self] Foo&
						1 String&
					0:
						[in:] [out: 1]
						$0 := alloc(Foo&)
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
				func [foo: v Fooer | v f: 1 b: 2 b: 3]
				type Fooer {[f: Int b: Int b: Int]}
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 [v] Fooer& (value)
					0:
						[in:] [out: 1]
						jmp 1
					1:
						[in: 0] [out:]
						$0 := arg(0 [v])
						$1 := 1
						$2 := 2
						$3 := 3
						virt call $0.0 [f:b:b:]($0, $1, $2, $3)
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
						0 $Block0&
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc($Block0&)
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
						$1 := alloc($Block0)
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
						0 (Int, Int) Pair&
					0:
						[in:] [out: 1]
						$2 := alloc((Int, Int) Pair)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						$1 := 6
						and($2, {x: $0 y: $1})
						$3 := arg(0)
						copy($3, $2, (Int, Int) Pair)
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
						0 (Int, Nil) Pair&
					0:
						[in:] [out: 1]
						$1 := alloc((Int, Nil) Pair)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						and($1, {x: $0 y: {}})
						$2 := arg(0)
						copy($2, $1, (Int, Nil) Pair)
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
						0 (Nil, Int) Pair&
					0:
						[in:] [out: 1]
						$1 := alloc((Nil, Int) Pair)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						and($1, {x: {} y: $0})
						$2 := arg(0)
						copy($2, $1, (Nil, Int) Pair)
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
						0 (Int, Nil, Int) Triple&
					0:
						[in:] [out: 1]
						$2 := alloc((Int, Nil, Int) Triple)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						$1 := 6
						and($2, {x: $0 y: {} z: $1})
						$3 := arg(0)
						copy($3, $2, (Int, Nil, Int) Triple)
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
						0 Int? &
					0:
						[in:] [out: 1]
						$1 := alloc(Int?)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						or($1, {1=some: $0})
						$2 := arg(0)
						copy($2, $1, Int?)
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
						0 Nil? &
					0:
						[in:] [out: 1]
						$0 := alloc(Nil?)
						jmp 1
					1:
						[in: 0] [out:]
						or($0, {1=some:})
						$1 := arg(0)
						copy($1, $0, Nil?)
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
						0 Int? &
					0:
						[in:] [out: 1]
						$0 := alloc(Int?)
						jmp 1
					1:
						[in: 0] [out:]
						or($0, {0=none})
						$1 := arg(0)
						copy($1, $0, Int?)
						return
			`,
		},
		{
			name: "make virt",
			src: `
				func [foo ^Fooer | ^5]
				type Fooer {[foo]}
				meth Int [foo]
			`,
			fun: "function0",
			want: `
				function0
					parms:
						0 Fooer&
					0:
						[in:] [out: 1]
						$1 := alloc(Int)
						$2 := alloc(Fooer)
						jmp 1
					1:
						[in: 0] [out:]
						$0 := 5
						store($1, $0)
						virt($2, $1, {function1})
						$3 := arg(0)
						copy($3, $2, Fooer)
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
						$0 := alloc(Int?)
						$1 := alloc(Int?)
						$2 := alloc($Block0)
						$3 := alloc(Nil Fun)
						$4 := alloc($Block1)
						$5 := alloc((Int&, Nil) Fun)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						or($1, {0=none})
						copy($0, $1, Int?)
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
						virt call $5.0($5, $7)
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
						$0 := alloc(Int?)
						$1 := alloc(Int?)
						$3 := alloc($Block0)
						$4 := alloc(Int Fun)
						$6 := alloc($Block1)
						$7 := alloc((Int&, Int) Fun)
						$8 := alloc(Int)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						or($1, {0=none})
						copy($0, $1, Int?)
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
						virt call $7.0($7, $10, $8)
						jmp 4
					4:
						[in: 2 3] [out:]
						$11 := load($8)
						$12 := arg(0)
						store($12, $11)
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
						$0 := alloc(Int?)
						$1 := alloc(Int?)
						$3 := alloc($Block0)
						$4 := alloc(String Fun)
						$6 := alloc($Block1)
						$7 := alloc((Int&, String) Fun)
						$8 := alloc(String)
						jmp 1
					1:
						[in: 0] [out: 2 3]
						or($1, {0=none})
						copy($0, $1, Int?)
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
						virt call $7.0($7, $10, $8)
						jmp 4
					4:
						[in: 2 3] [out:]
						$11 := arg(0)
						copy($11, $8, String)
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
						$1 := alloc($Block0)
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
						0 $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc($Block0&)
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
						$1 := alloc($Block0)
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
						0 $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc($Block0&)
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
						$1 := alloc($Block0)
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
						0 $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc($Block0&)
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
						$3 := alloc($Block0)
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
						0 $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc($Block0&)
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
						$3 := alloc($Block0)
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
						0 $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc($Block0&)
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
						0 [self] Point&
						1 Int&
					0:
						[in:] [out: 1]
						$0 := alloc(Point&)
						$1 := arg(0 [self])
						store($0, $1)
						$5 := alloc($Block0)
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
						0 $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc($Block0&)
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
						0 $Block1&
						1 [i] Int
						2 Nil Fun&
					0:
						[in:] [out: 1]
						$0 := alloc($Block1&)
						$1 := alloc(Int)
						$2 := arg(0)
						store($0, $2)
						$3 := arg(1 [i])
						store($1, $3)
						$7 := alloc($Block0)
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
						$1 := alloc($Block1)
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
						0 $Block0&
					0:
						[in:] [out: 1]
						$0 := alloc($Block0&)
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
						0 $Block1&
						1 Nil Fun&
					0:
						[in:] [out: 1]
						$0 := alloc($Block1&)
						$1 := arg(0)
						store($0, $1)
						$8 := alloc($Block0)
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
						$3 := alloc($Block1)
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
			`,
		},
		{
			name: "bool is an enum",
			src: `
				func [foo ^Bool | ^{true}]
			`,
			fun: "",
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
			fun: "",
			want: `
				function0
					parms:
						0 Num&
					0:
						[in:] [out: 1]
						$0 := alloc(Num)
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
						$4 := alloc($Block0)
						$5 := alloc(Nil Fun)
						$6 := alloc($Block1)
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
						$0 := alloc(Num)
						$1 := alloc(Num)
						$4 := alloc($Block0)
						$5 := alloc(Nil Fun)
						$6 := alloc($Block1)
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
						$6 := alloc($Block0)
						$7 := alloc(Nil Fun)
						$8 := alloc($Block1)
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
						0 $Block1&
					0:
						[in:] [out: 1]
						$0 := alloc($Block1&)
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
			`,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if strings.HasPrefix(test.name, "SKIP") {
				t.Skip()
			}
			p := ast.NewParser("#test")
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
