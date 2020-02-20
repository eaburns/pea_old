// © 2020 the Pea Authors under the MIT license. See AUTHORS for the list of authors.

package types

import (
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
)

func TestString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		src     string
		want    string
		imports [][2]string
	}{
		{src: "val x := [5]", want: "x"},
		{src: "Val x := [5]", want: "x"},
		{src: "val x Int32 := [5]", want: "x Int32"},
		{src: "val x Int Array := [{5}]", want: "x Int Array"},
		{src: "type Xyz {}", want: "#test Xyz"},
		{src: "Type Xyz {}", want: "#test Xyz"},
		{src: "type X Xyz {x: X}", want: "X #test Xyz"},
		{src: "type (X, Y, Z) Xyz {x: X y: Y z: Z}", want: "(X, Y, Z) #test Xyz"},
		{src: "type (X Foo) Xyz {x: X} type Foo{[foo]}", want: "(X Foo) #test Xyz"},
		{
			src:  "type (X Foo, Y, Z Foo) Xyz {x: X y: Y z: Z} type Foo{[foo]}",
			want: "(X Foo, Y, Z Foo) #test Xyz",
		},
		{src: "type X& {x: X}", want: "X #test &"},
		{src: "type (X, Y, Z)& {x: X y: Y z: Z}", want: "(X, Y, Z) #test &"},
		{src: "func [unary |]", want: "[unary]"},
		{src: "func [unary]", want: "[unary]"},
		{src: "func T [unary ^T]", want: "T [unary ^T]"},
		{src: "func (T Foo) [unary ^T] type Foo { [xyz] }", want: "(T Foo) [unary ^T]"},
		{src: "Func [unary |]", want: "[unary]"},
		{src: "meth Int [++ abc Int |]", want: "Int [++ abc Int]"},
		{src: "meth Int [++ abc Int]", want: "Int [++ abc Int]"},
		{src: "Meth Int [++ abc Int |]", want: "Int [++ abc Int]"},
		{src: "meth Int [+ abc Int ^Int]", want: "Int [+ abc Int ^Int]"},
		{src: "meth T Array [at: i Int put: t T |]", want: "T Array [at: i Int put: t T]"},
		{src: "meth _ Array [foo: x Int |]", want: "_ Array [foo: x Int]"},
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
				meth T #foo Abc [bar ^T]
			`,
			want:    "T #foo Abc [bar ^T]",
			imports: [][2]string{{"foo", "Type T Abc {t: T}"}},
		},
		{
			src: `
				import "foo"
				meth (T, U) #foo Abc [bar: _ T baz: _ U]
			`,
			want:    "(T, U) #foo Abc [bar: _ T baz: _ U]",
			imports: [][2]string{{"foo", "Type (T, U) Abc {t: T u: U}"}},
		},
		{
			src: `
				import "foo"
				meth T #foo ? [bar ^T]
			`,
			want:    "T #foo ? [bar ^T]",
			imports: [][2]string{{"foo", "Type T? {t: T}"}},
		},
		{
			src: `
				import "foo"
				meth (_, _) #foo ? [bar|]
			`,
			want:    "(_, _) #foo ? [bar]",
			imports: [][2]string{{"foo", "Type (T, U)? {t: T u: U}"}},
		},

		// Tests for TypeName.String.
		// These use Val's typename to exercise the code path,
		// since this test framework only does .String() on Defs.
		{src: "val x Int := []", want: "x Int"},
		{src: "val x Float Array := []", want: "x Float Array"},
		{src: "val x Float Array Array := []", want: "x Float Array Array"},
		{src: "val x (Float, String) Pair := [] type (X, Y) Pair{x: X y: Y}", want: "x (Float, String) Pair"},
		{src: "val x (Float, String Array) Pair := [] type (X, Y) Pair{x: X y: Y}", want: "x (Float, String Array) Pair"},
		{src: "val x Int& := []", want: "x Int&"},
		{src: "val x Int& & & := []", want: "x Int& & &"},
		{src: "val x (Int, Float)! := [] type (X, Y)! {x: X y: Y}", want: "x (Int, Float)!"},
		{src: "val x Int& && := [] type T&&{t: T}", want: "x Int& &&"},
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
			imports: [][2]string{{"foo", "Type T Abc {t: T}"}},
		},
		{
			src: `
				import "foo"
				val x (Int, String) #foo Abc := []
			`,
			want:    "x (Int, String) #foo Abc",
			imports: [][2]string{{"foo", "Type (T, U) Abc {t: T u: U}"}},
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
			imports: [][2]string{{"foo", "Type T ? {t: T}"}},
		},
		{
			src: `
				import "foo"
				val x (Int, String) #foo ? := []
			`,
			want:    "x (Int, String) #foo ?",
			imports: [][2]string{{"foo", "Type (T, U) ? {t: T u: U}"}},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.want, func(t *testing.T) {
			t.Parallel()
			p := ast.NewParser("/test/test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Errorf("failed to parse [%s]: %s,", test.src, err.Error())
				return
			}
			mod, errs := Check(p.Mod(), Config{
				Importer: testImporter(test.imports),
			})
			if len(errs) > 0 {
				t.Errorf("failed to check [%s]: %v", test.src, errs)
				return
			}
			got := mod.Defs[0].String()
			if got != test.want {
				t.Errorf("%s\ngot:\n	%s\nexpected:\n	%s", test.src, got, test.want)
				return
			}
		})
	}
}

func TestFullString(t *testing.T) {
	t.Parallel()
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
			"type X Xyz {x: X}",
			"type X Xyz { x: X }",
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
		test := test
		t.Run(test.src, func(t *testing.T) {
			p := ast.NewParser("/test/test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Errorf("failed to parse [%s]: %s,", test.src, err.Error())
				return
			}
			mod, errs := Check(p.Mod(), Config{})
			if len(errs) > 0 {
				t.Errorf("failed to check [%s]: %v", test.src, errs)
				return
			}
			got := mod.Defs[0].(*Type).fullString()
			if got != test.want {
				t.Errorf("%s:\ngot:\n	%s\nexpected:\n	%s", test.src, got, test.want)
				return
			}
		})
	}
}
