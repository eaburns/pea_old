package types

import (
	"reflect"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
)

func TestBuiltInMeths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		src   string
		want  []string
		trace bool
	}{
		{
			name: "no built-in",
			src: `
				type Foo { bar: Int baz: Float }
			`,
			want: nil,
		},
		{
			name: "typeless case",
			src: `
				type Foo { bar }
			`,
			want: []string{
				"Foo $0 [ifBar: _ $0 Fun ^$0]",
			},
		},
		{
			name: "typeless cases",
			src: `
				type Nums { one | two | three | four }
			`,
			want: []string{
				"Nums $0 [ifOne: _ $0 Fun ifTwo: _ $0 Fun ifThree: _ $0 Fun ifFour: _ $0 Fun ^$0]",
			},
		},
		{
			name: "typed case",
			src: `
				type IntOrString { int: Int | string: String }
			`,
			want: []string{
				"IntOrString $0 [ifInt: _ (Int&, $0) Fun ifString: _ (String&, $0) Fun ^$0]",
			},
		},
		{
			name: "mixed case",
			src: `
				type IntOpt { int: Int | none }
			`,
			want: []string{
				"IntOpt $0 [ifInt: _ (Int&, $0) Fun ifNone: _ $0 Fun ^$0]",
			},
		},
		{
			name: "parameterized or-type receiver",
			src: `
				type T? { none | some: T }
			`,
			want: []string{
				"T? $0 [ifNone: _ $0 Fun ifSome: _ (T&, $0) Fun ^$0]",
			},
		},
		{
			name: "a virtual method",
			src: `
				type Foo { [bar: Int baz: String ^Float] }
			`,
			want: []string{
				"#test Foo [bar: _ Int baz: _ String ^Float]",
			},
		},
		{
			name: "virtual methods",
			src: `
				type Foo {
					[bar]
					[baz: Int]
					[* Foo ^Foo]
				}
			`,
			want: []string{
				"#test Foo [bar]",
				"#test Foo [baz: _ Int]",
				"#test Foo [* _ Foo ^Foo]",
			},
		},
		{
			name: "parameterized virtual-type receiver",
			src: `
				type T Eq {
					[= T& ^Bool]
				}
			`,
			want: []string{
				"T #test Eq [= _ T& ^Bool]",
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			p := ast.NewParser("/test/test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Fatalf("failed to parse source: %s", err)
			}
			mod, errs := Check(p.Mod(), Config{Trace: test.trace})
			if len(errs) > 0 {
				t.Fatalf("failed to check source: %v", errs)
			}
			var got []string
			for _, def := range mod.Defs {
				fun, ok := def.(*Fun)
				if !ok {
					continue
				}
				if _, ok := fun.AST.(*ast.Fun); ok {
					continue
				}
				got = append(got, fun.String())
			}
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("got %v, expected %v", got, test.want)
			}
		})
	}
}

func TestCaseMethod(t *testing.T) {
	t.Parallel()
	tests := []errorTest{
		{
			name: "correct param types",
			src: `
				val x String := [
					true ifTrue: ["string"] ifFalse: ["string"]
				]
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestBlockType(t *testing.T) {
	t.Parallel()
	src := `
		val test (Int, Int) Fun := [
			x := 5.
			y := 6.
			z := 7.
			[ :a Int |
				x := x + y.
				z := z + a.
				x + y + z
			]
		]
	`
	p := ast.NewParser("/test/test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}
	blk := findTestVal(mod, "test").Init[3].(*Block)
	got := blk.BlockType.fullString()
	const want = "type $Block0 { x: Int& y: Int& z: Int& }"
	if got != want {
		t.Errorf("got block type %s, want %s", got, want)
	}
}
