package types

import (
	"reflect"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
)

func TestBuiltInMeths(t *testing.T) {
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
				"meth Foo $0 [ifBar: _ $0 Fun ^$0]",
			},
		},
		{
			name: "typeless cases",
			src: `
				type Nums { one, two, three, four }
			`,
			want: []string{
				"meth Nums $0 [ifOne: _ $0 Fun ifTwo: _ $0 Fun ifThree: _ $0 Fun ifFour: _ $0 Fun ^$0]",
			},
		},
		{
			name: "typeed case",
			src: `
				type IntOrString { int: Int, string: String }
			`,
			want: []string{
				"meth IntOrString $0 [ifInt: _ (Int64&, $0) Fun ifString: _ (String&, $0) Fun ^$0]",
			},
		},
		{
			name: "mixed case",
			src: `
				type IntOpt { int: Int, none }
			`,
			want: []string{
				"meth IntOpt $0 [ifInt: _ (Int64&, $0) Fun ifNone: _ $0 Fun ^$0]",
			},
		},
		{
			name: "parameterized or-type receiver",
			src: `
				type T? { none, some: T }
			`,
			want: []string{
				"meth T? $0 [ifNone: _ $0 Fun ifSome: _ (T&, $0) Fun ^$0]",
			},
		},
		{
			name: "a virtual method",
			src: `
				type Foo { [bar: Int baz: String ^Float] }
			`,
			want: []string{
				"meth Foo [bar: _ Int64 baz: _ String ^Float64]",
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
				"meth Foo [bar]",
				"meth Foo [baz: _ Int64]",
				"meth Foo [* _ Foo ^Foo]",
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
				"meth T Eq [= _ T& ^Bool]",
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			p := ast.NewParser("#test")
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
