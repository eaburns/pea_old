package types

import (
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
)

func TestString(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"val x := [5]", "val x"},
		{"Val x := [5]", "Val x"},
		{"val x Int32 := [5]", "val x Int32"},
		{"val x Int Array := [5]", "val x Int Array"},
		{"type Xyz { }", "type Xyz"},
		{"Type Xyz { }", "Type Xyz"},
		{"type X Xyz { }", "type X Xyz"},
		{"type (X, Y, Z) Xyz { }", "type (X, Y, Z) Xyz"},
		{"type (X Foo) Xyz { } type Foo{[foo]}", "type (X Foo) Xyz"},
		{"type (X Foo, Y, Z Foo) Xyz { } type Foo{[foo]}", "type (X Foo, Y, Z Foo) Xyz"},
		{"type X& { }", "type X&"},
		{"type (X, Y, Z)& { }", "type (X, Y, Z)&"},
		{"func [unary |]", "func [unary]"},
		{"Func [unary |]", "Func [unary]"},
		{"meth Int [++ abc Int |]", "meth Int [++ abc Int]"},
		{"Meth Int [++ abc Int |]", "Meth Int [++ abc Int]"},
		{"meth Int [+ abc Int ^Int |]", "meth Int [+ abc Int ^Int]"},
		{"meth T Array [at: i Int put: t T |]", "meth T Array [at: i Int put: t T]"},
		{"meth T Array [foo: x Int |]", "meth T Array [foo: x Int]"},

		// Tests for TypeName.String.
		// These use Val's typename to exercise the code path,
		// since this test framework only does .String() on Defs.
		{"val x Int := []", "val x Int"},
		{"val x Float Array := []", "val x Float Array"},
		{"val x Float Array Array := []", "val x Float Array Array"},
		{"val x (Float, String) Pair := [] type (X, Y) Pair{}", "val x (Float, String) Pair"},
		{"val x (Float, String Array) Pair := [] type (X, Y) Pair{}", "val x (Float, String Array) Pair"},
		{"val x Int& := []", "val x Int&"},
		{"val x Int& & & := []", "val x Int& & &"},
		{"val x (Int, Float)! := [] type (X, Y)! {}", "val x (Int, Float)!"},
		{"val x Int& && := [] type T &&{}", "val x Int& &&"},
	}
	for _, test := range tests {
		p := ast.NewParser("#test")
		if err := p.Parse("", strings.NewReader(test.src)); err != nil {
			t.Errorf("failed to parse [%s]: %s,", test.src, err.Error())
			continue
		}
		mod, errs := Check(p.Mod(), Config{})
		if len(errs) > 0 {
			t.Errorf("failed to check [%s]: %v", test.src, errs)
			continue
		}
		got := mod.Defs[0].String()
		if got != test.want {
			t.Errorf("got:\n	%s\nexpected:\n	%s", got, test.want)
			continue
		}
	}
}
