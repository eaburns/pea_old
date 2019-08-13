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
		// The val is just needed for syntax, this is testing just the Sub.
		{"#foo _ := [3]", "#foo"},
		{"import \"abc\"", "import \"abc\""},
		{"x := [5]", "x"},
		{"x Int32 := [5]", "x Int32"},
		{"x Int List := [5]", "x Int List"},
		{"Type { }", "Type"},
		{"X Type { }", "X Type"},
		{"(X, Y, Z) Type { }", "(X, Y, Z) Type"},
		{"(X List) Type { }", "(X List) Type"},
		{"(X List, Y, Z Array) Type { }", "(X List, Y, Z Array) Type"},
		{"X? { }", "X?"},
		{"(X, Y, Z)? { }", "(X, Y, Z)?"},
		{"[unary |]", "[unary]"},
		{"Int [++ abc Int |]", "Int [++ abc Int]"},
		{"Int [+ abc Int ^Int |]", "Int [+ abc Int ^Int]"},
		{"T Array [at: i Int put: t T |]", "T Array [at: i Int put: t T]"},
		{"T Array [foo: x Bar |]", "T Array [foo: x Bar]"},

		// Tests for TypeName.String.
		// These use Val's typename to exercise the code path,
		// since this test framework only does .String() on Defs.
		{"x Int := []", "x Int"},
		{"x Float Array := []", "x Float Array"},
		{"x Float Array Array := []", "x Float Array Array"},
		{"x (Float, String) Pair := []", "x (Float, String) Pair"},
		{"x (Float, String Array) Pair := []", "x (Float, String Array) Pair"},
		{"x Int? := []", "x Int?"},
		{"x Int? ? ? := []", "x Int? ? ?"},
		{"x (Int, Float)! := []", "x (Int, Float)!"},
		{"x Int? ?? := []", "x Int? ??"},
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
