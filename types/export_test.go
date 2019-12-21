package types

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestExportImport(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "empty",
			src:  "",
		},
		{
			name: "simple val",
			src: `
				val x Int := [5]
			`,
		},
		{
			name: "inferred type val",
			src: `
				val x := [5]
			`,
		},
		{
			name: "empty type",
			src: `
				type Empty {}
			`,
		},
		{
			name: "simple alias type",
			src: `
				type Integer := Int.
			`,
		},
		{
			name: "param alias type",
			src: `
				type T Pred := (T, Bool) Fun.
			`,
		},
		{
			name: "simple and type",
			src: `
				type Point {x: Float y: Float}
			`,
		},
		{
			name: "param and type",
			src: `
				type (X, Y) Pair {x: X y: Y}
			`,
		},
		{
			name: "simple or type",
			src: `
				type IntOrNil {nil | int: Int}
			`,
		},
		{
			name: "param or type",
			src: `
				type T? {none | some: T}
			`,
		},
		{
			name: "simple virt type",
			src: `
				type FooBarBazer {
					[foo]
					[bar: Int]
					[baz: Float qux: String]
					[+ UInt8]
				}
			`,
		},
		{
			name: "param virt type",
			src: `
				type T Eq {[= T ^Bool]}
			`,
		},
		{
			name: "recursive type",
			src: `
				type Loop { x: Loop& }
			`,
		},
		{
			name: "mutual recursive types",
			src: `
				type LoopA { x: LoopB& }
				type LoopB { y: LoopA& }
			`,
		},
		{
			name: "type constraints",
			src: `
				type HashKey {[hash ^Int]}
				type (K HashKey, V) HashMap {useK: K useV: V}
			`,
		},
		{
			name: "simple 0-ary func",
			src: `
				func [foo]
			`,
		},
		{
			name: "simple n-ary func",
			src: `
				func [foo: _ Int bar: _ Float]
			`,
		},
		{
			name: "simple func with return",
			src: `
				func [foo ^Int]
			`,
		},
		{
			name: "type parm func",
			src: `
				func T [foo ^T]
			`,
		},
		{
			name: "constrained type parm func",
			src: `
				type Fooer {[foo]}
				func (T Fooer) [foo ^T]
			`,
		},
		{
			name: "simple 0-ary meth",
			src: `
				meth Int [foo]
			`,
		},
		{
			name: "simple binray meth",
			src: `
				meth Int [+++ _ Int]
			`,
		},
		{
			name: "simple n-ary meth",
			src: `
				meth Int [foo: _ Int bar: _ Float]
			`,
		},
		{
			name: "type parm meth",
			src: `
				meth Int T [foo ^T]
			`,
		},
		{
			name: "constrained type parm meth",
			src: `
				type Fooer {[foo]}
				meth Int (T Fooer) [foo ^T]
			`,
		},
		{
			name: "type parm receiver",
			src: `
				meth T Array [foo ^T]
			`,
		},
		{
			name: "constrained type parm receiver",
			src: `
				type Fooer {[foo]}
				meth (T Fooer) Array [foo ^T]
			`,
		},
		{
			name: "constrained type parm receiver type parm meth",
			src: `
				type Fooer {[foo]}
				meth (T Fooer) Array U [from: _ U fold: _ (U, T, U) Fun ^U]
			`,
		},
		{
			name: "instantiated parm meth",
			src: `
				type Fooer {[foo]}
				meth (T Fooer) Array U [from: _ U fold: _ (U, T, U) Fun ^U]
				val _ Int := [
					x Int Array := {}.
					x from: 0 fold: [:sum :x | x + sum]
				]
			`,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			p := ast.NewParser("test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Fatalf("failed to parse source: %s", err)
			}
			mod, errs := Check(p.Mod(), Config{})
			if len(errs) > 0 {
				t.Fatalf("failed to check source: %v", errs)
			}
			mod.Name = "test"
			var buf bytes.Buffer
			if err := Write(&buf, mod); err != nil {
				t.Fatalf("failed to write the mod: %v", err)
			}
			got, err := Read(&buf)
			if err != nil {
				t.Fatalf("failed to read the mod: %v", err)
			}
			opts := []cmp.Option{
				cmp.Exporter(func(reflect.Type) bool { return true }),
				cmpopts.IgnoreFields(Mod{}, "SortedVals"),
				cmpopts.IgnoreFields(Val{}, "Locals", "Init"),
				cmpopts.IgnoreFields(Fun{}, "Insts"),
				cmpopts.IgnoreFields(Type{}, "Insts"),
				cmp.FilterPath(func(p cmp.Path) bool {
					return len(p) > 0 && p[len(p)-1].String() == ".AST"
				}, cmp.Ignore()),
			}
			if diff := cmp.Diff(mod, got, opts...); diff != "" {
				t.Errorf("modules do not match:\n%s", diff)
			}
		})
	}
}

func TestWriteReadBool(t *testing.T) {
	t.Parallel()
	tests := []bool{true, false}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
			var buf bytes.Buffer
			writeBool(&buf, test)
			got := readBool(&buf)
			if got != test {
				t.Errorf("got %v, want %v", got, test)
			}
		})
	}
}

func TestWriteReadInt(t *testing.T) {
	t.Parallel()
	tests := []int{math.MinInt32, -1, 0, 1, 10, 50, math.MaxInt32}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
			var buf bytes.Buffer
			writeInt(&buf, test)
			got := readInt(&buf)
			if got != test {
				t.Errorf("got %v, want %v", got, test)
			}
		})
	}
}

func TestWriteReadString(t *testing.T) {
	t.Parallel()
	tests := []string{
		"",
		"1",
		"12",
		"こんにちは、みなさん",
		"\"",
		"\n",
	}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%v", test), func(t *testing.T) {
			var buf bytes.Buffer
			writeString(&buf, test)
			got := readString(&buf)
			if got != test {
				t.Errorf("got %v, want %v", got, test)
			}
		})
	}
}

func TestWriteIntTooSmall(t *testing.T) {
	t.Parallel()
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("expected panic, got nil")
		}
	}()
	writeInt(&bytes.Buffer{}, math.MinInt32-1)
}

func TestWriteIntTooBig(t *testing.T) {
	t.Parallel()
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("expected panic, got nil")
		}
	}()
	writeInt(&bytes.Buffer{}, math.MaxInt32+1)
}
