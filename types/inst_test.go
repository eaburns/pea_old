package types

import (
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
)

func TestInstCallError(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				val test := [
					foo: 5.
					foo: "string".
					foo: 6.
					foo: "another string".
				]
				func T [foo: _ T |]
			`,
			err: "",
		},
		{
			name: "infer arg expr error",
			src: `
				val test := [ foo: {Unknown | } ]
				func T [foo: _ T |]
			`,
			err: "Unknown not found",
		},
		{
			name: "not all vars bound",
			src: `
				val test := [ foo: 5 ]
				func T [foo: _ Int |]
			`,
			err: "cannot infer",
		},
		{
			name: "return unify fails",
			src: `
				val test String := [ foo ]
				func T [foo ^T Array |]
			`,
			err: "type mismatch",
		},
		{
			name: "param unify fails",
			src: `
				val test := [ foo: "string" ]
				func T [foo: _ T Array |]
			`,
			err: "type mismatch",
		},
		{
			name: "multi-binding type mismatch",
			src: `
				val test Rune := [ foo: "string" ]
				func T [foo: _ T ^T |]
			`,
			err: "have String, want Int32",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestInstCall(t *testing.T) {
	tests := []instTest{
		{
			name: "fully grounded",
			src: `
				val test := [ 1 + 2 ]
			`,
			want: "Int [+ _ Int ^Int]",
		},
		{
			name: "ground receiver subs return",
			src: `
				val test := [ {String Array|} at: 2 ]
			`,
			want: "String Array [at: _ Int ^String&]",
		},
		{
			name: "ground receiver subs parm",
			src: `
				val test := [ {String Array|} at: 2 put: "hello" ]
			`,
			want: "String Array [at: _ Int put: _ String]",
		},
		{
			name: "ground multi-type-param receiver",
			src: `
				val test := [ {(String, Float) Map|} at: "pi" put: 3.14 ]
				type (K, V) Map {}
				meth (K, V) Map [at: _ K put: _ V |]
			`,
			want: "(String, Float) Map [at: _ String put: _ Float]",
		},
		{
			name: "ground imported receiver type",
			src: `
				import "map"
				val test := [ {(String, Float) #map Map|} #map at: "pi" put: 3.14 ]
			`,
			imports: [][2]string{
				{"map", `
					Type (K, V) Map {}
					Meth (K, V) Map [at: _ K put: _ V |]
				`},
			},
			want: "(String, Float) Map [at: _ String put: _ Float]",
		},
		{
			name: "ground fun return type",
			src: `
				val test String := [
					5 < 6 ifTrue: ["hello"] ifFalse: ["goodbye"]
				]
			`,
			want: "Bool [ifTrue: _ String Fun ifFalse: _ String Fun ^String]",
		},
		{
			name: "ground fun parameter type",
			src: `
					val test := [
						foo: "Hello"
					]
					func T [foo: _ T |]
				`,
			want: "[foo: _ String]",
		},
		{
			name: "ground fun parameter complex type",
			src: `
					val test := [
						foo: { String Array Array | }
					]
					func T [foo: _ T Array Array |]
				`,
			want: "[foo: _ String Array Array]",
		},
		{
			name: "map method",
			src: `
					val test String Array := [
						{Int8 Array|} map: [:i Int8 | "foo"]
					]
					meth T Array R [map: _ (T, R) Fun ^R Array |]
				`,
			want: "Int8 Array [map: _ (Int8, String) Fun ^String Array]",
		},
		{
			name: "reduce method",
			src: `
					val test String := [
						{Int8 Array|} init: "hello" fold: [:i Int8 :s String | "foo"]
					]
					meth T Array R [init: _ R fold: _ (T, R, R) Fun ^R |]
				`,
			want: "Int8 Array [init: _ String fold: _ (Int8, String, String) Fun ^String]",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

type instTest struct {
	name string
	// The src must contain a val named test with a call statement.
	// The .String() of the .Fun of the first Msg
	// of the first call statement is compared to want,
	// or the string <nil> if the .Fun is nil.
	src     string
	imports [][2]string
	want    string
	trace   bool
}

func (test instTest) run(t *testing.T) {
	p := ast.NewParser("#test")
	if err := p.Parse("", strings.NewReader(test.src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	cfg := Config{
		Importer: testImporter(test.imports),
		Trace:    test.trace,
	}
	mod, errs := Check(p.Mod(), cfg)
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}
	val := findTestVal(mod)
	if val == nil {
		t.Fatal("val test not found")
	}
	call := firstCallStmt(val)
	if call == nil {
		t.Fatal("call statement not found")
	}
	got := "<nil>"
	if fun := call.Msgs[0].Fun; fun != nil {
		got = fun.String()
	}
	if got != test.want {
		t.Errorf("got %s, want %s", got, test.want)
	}
}

func findTestVal(mod *Mod) *Val {
	for _, def := range mod.Defs {
		if v, ok := def.(*Val); ok && v.Var.Name == "test" {
			return v
		}
	}
	return nil
}

func firstCallStmt(val *Val) *Call {
	for _, stmt := range val.Init {
		if call, ok := stmt.(*Call); ok {
			return call
		}
	}
	return nil
}
