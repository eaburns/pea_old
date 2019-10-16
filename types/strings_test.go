package types

import (
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
)

func TestString(t *testing.T) {
	tests := []struct {
		src     string
		want    string
		imports [][2]string
	}{
		{src: "val x := [5]", want: "x"},
		{src: "Val x := [5]", want: "x"},
		{src: "val x Int32 := [5]", want: "x Int32"},
		{src: "val x Int Array := [{5}]", want: "x Int Array"},
		{src: "type Xyz { }", want: "Xyz"},
		{src: "Type Xyz { }", want: "Xyz"},
		{src: "type X Xyz { }", want: "X Xyz"},
		{src: "type (X, Y, Z) Xyz { }", want: "(X, Y, Z) Xyz"},
		{src: "type (X Foo) Xyz { } type Foo{[foo]}", want: "(X Foo) Xyz"},
		{src: "type (X Foo, Y, Z Foo) Xyz { } type Foo{[foo]}", want: "(X Foo, Y, Z Foo) Xyz"},
		{src: "type X& { }", want: "X&"},
		{src: "type (X, Y, Z)& { }", want: "(X, Y, Z)&"},
		{src: "func [unary |]", want: "[unary]"},
		{src: "func T [unary |]", want: "T [unary]"},
		{src: "func (T Foo) [unary |] type Foo { [xyz] }", want: "(T Foo) [unary]"},
		{src: "Func [unary |]", want: "[unary]"},
		{src: "meth Int [++ abc Int |]", want: "Int [++ abc Int]"},
		{src: "Meth Int [++ abc Int |]", want: "Int [++ abc Int]"},
		{src: "meth Int [+ abc Int ^Int |]", want: "Int [+ abc Int ^Int]"},
		{src: "meth T Array [at: i Int put: t T |]", want: "T Array [at: i Int put: t T]"},
		{src: "meth T Array [foo: x Int |]", want: "T Array [foo: x Int]"},
		{
			src: `
				import "foo"
				meth #foo Abc [bar|]
			`,
			want:    "#foo Abc [bar]",
			imports: [][2]string{{"foo", "Type Abc{}"}},
		},
		{
			src: `
				import "foo"
				meth T #foo Abc [bar|]
			`,
			want:    "T #foo Abc [bar]",
			imports: [][2]string{{"foo", "Type T Abc {}"}},
		},
		{
			src: `
				import "foo"
				meth (T, U) #foo Abc [bar|]
			`,
			want:    "(T, U) #foo Abc [bar]",
			imports: [][2]string{{"foo", "Type (T, U) Abc {}"}},
		},
		{
			src: `
				import "foo"
				meth T #foo ? [bar|]
			`,
			want:    "T #foo ? [bar]",
			imports: [][2]string{{"foo", "Type T ? {}"}},
		},
		{
			src: `
				import "foo"
				meth (T, U) #foo ? [bar|]
			`,
			want:    "(T, U) #foo ? [bar]",
			imports: [][2]string{{"foo", "Type (T, U) ? {}"}},
		},

		// Tests for TypeName.String.
		// These use Val's typename to exercise the code path,
		// since this test framework only does .String() on Defs.
		{src: "val x Int := []", want: "x Int"},
		{src: "val x Float Array := []", want: "x Float Array"},
		{src: "val x Float Array Array := []", want: "x Float Array Array"},
		{src: "val x (Float, String) Pair := [] type (X, Y) Pair{}", want: "x (Float, String) Pair"},
		{src: "val x (Float, String Array) Pair := [] type (X, Y) Pair{}", want: "x (Float, String Array) Pair"},
		{src: "val x Int& := []", want: "x Int&"},
		{src: "val x Int& & & := []", want: "x Int& & &"},
		{src: "val x (Int, Float)! := [] type (X, Y)! {}", want: "x (Int, Float)!"},
		{src: "val x Int& && := [] type T &&{}", want: "x Int& &&"},
		{
			src: `
				import "foo"
				val x #foo Abc := []
			`,
			want:    "x #foo Abc",
			imports: [][2]string{{"foo", "Type Abc {}"}},
		},
		{
			src: `
				import "foo"
				val x Int #foo Abc := []
			`,
			want:    "x Int #foo Abc",
			imports: [][2]string{{"foo", "Type T Abc {}"}},
		},
		{
			src: `
				import "foo"
				val x (Int, String) #foo Abc := []
			`,
			want:    "x (Int, String) #foo Abc",
			imports: [][2]string{{"foo", "Type (T, U) Abc {}"}},
		},
		{
			src: `
				import "foo"
				val x #foo ? := []
			`,
			want:    "x #foo ?",
			imports: [][2]string{{"foo", "Type ? {}"}},
		},
		{
			src: `
				import "foo"
				val x Int #foo ? := []
			`,
			want:    "x Int #foo ?",
			imports: [][2]string{{"foo", "Type T ? {}"}},
		},
		{
			src: `
				import "foo"
				val x (Int, String) #foo ? := []
			`,
			want:    "x (Int, String) #foo ?",
			imports: [][2]string{{"foo", "Type (T, U) ? {}"}},
		},
	}
	for _, test := range tests {
		p := ast.NewParser("#test")
		if err := p.Parse("", strings.NewReader(test.src)); err != nil {
			t.Errorf("failed to parse [%s]: %s,", test.src, err.Error())
			continue
		}
		mod, errs := Check(p.Mod(), Config{
			Importer: testImporter(test.imports),
		})
		if len(errs) > 0 {
			t.Errorf("failed to check [%s]: %v", test.src, errs)
			continue
		}
		got := mod.Defs[0].String()
		if got != test.want {
			t.Errorf("%s\ngot:\n	%s\nexpected:\n	%s", test.src, got, test.want)
			continue
		}
	}
}

func TestFullString(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			"type Xyz { }",
			"type Xyz {}",
		},
		{
			"Type Xyz { }",
			"Type Xyz {}",
		},
		{
			"type X Xyz { }",
			"type X Xyz {}",
		},
		{
			"type Abc := Int.",
			"type Abc := Int.",
		},
		{
			"type Abc := Int Array.",
			"type Abc := Int Array.",
		},
		{
			"type Abc { x0: Int x1: Int }",
			"type Abc { x0: Int x1: Int }",
		},
		{
			"type Abc { x0: Int | x1: Int }",
			"type Abc { x0: Int | x1: Int }",
		},
		{
			"type T? { None | Some: T }",
			"type T? { None | Some: T }",
		},
		{
			"type Abc { [foo] [bar: Int] [baz ^Bool] [= Int ^Bool] }",
			"type Abc { [foo] [bar: Int] [baz ^Bool] [= Int ^Bool] }",
		},
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
		got := mod.Defs[0].(*Type).fullString()
		if got != test.want {
			t.Errorf("%s:\ngot:\n	%s\nexpected:\n	%s", test.src, got, test.want)
			continue
		}
	}
}
