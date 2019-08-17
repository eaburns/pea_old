package ast

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"import \"abc\"", "import \"abc\""},
		{"val x := [5]", "val x"},
		{"Val x := [5]", "Val x"},
		{"val x Int32 := [5]", "val x Int32"},
		{"val x Int List := [5]", "val x Int List"},
		{"type Xyz { }", "type Xyz"},
		{"Type Xyz { }", "Type Xyz"},
		{"type X Xyz { }", "type X Xyz"},
		{"type (X, Y, Z) Xyz { }", "type (X, Y, Z) Xyz"},
		{"type (X List) Xyz { }", "type (X List) Xyz"},
		{"type (X List, Y, Z Array) Xyz { }", "type (X List, Y, Z Array) Xyz"},
		{"type X? { }", "type X?"},
		{"type (X, Y, Z)? { }", "type (X, Y, Z)?"},
		{"func [unary |]", "func [unary]"},
		{"Func [unary |]", "Func [unary]"},
		{"meth Int [++ abc Int |]", "meth Int [++ abc Int]"},
		{"Meth Int [++ abc Int |]", "Meth Int [++ abc Int]"},
		{"meth Int [+ abc Int ^Int |]", "meth Int [+ abc Int ^Int]"},
		{"meth T Array [at: i Int put: t T |]", "meth T Array [at: i Int put: t T]"},
		{"meth T Array [foo: x Bar |]", "meth T Array [foo: x Bar]"},

		// Tests for TypeName.String.
		// These use Val's typename to exercise the code path,
		// since this test framework only does .String() on Defs.
		{"val x Int := []", "val x Int"},
		{"val x Float Array := []", "val x Float Array"},
		{"val x Float Array Array := []", "val x Float Array Array"},
		{"val x (Float, String) Pair := []", "val x (Float, String) Pair"},
		{"val x (Float, String Array) Pair := []", "val x (Float, String Array) Pair"},
		{"val x Int? := []", "val x Int?"},
		{"val x Int? ? ? := []", "val x Int? ? ?"},
		{"val x (Int, Float)! := []", "val x (Int, Float)!"},
		{"val x Int? ?? := []", "val x Int? ??"},
	}
	for _, test := range tests {
		p := NewParser("")
		if err := p.Parse("", strings.NewReader(test.src)); err != nil {
			t.Errorf("failed to parse [%s]: %s,", test.src, err.Error())
			continue
		}
		got := p.Mod().Files[0].Defs[0].String()
		if got != test.want {
			t.Errorf("got:\n	%s\nexpected:\n	%s", got, test.want)
			continue
		}
	}
}
