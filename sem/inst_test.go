package sem

import (
	"strings"
	"testing"

	"github.com/eaburns/pea/syn"
)

func TestInstType(t *testing.T) {
	tests := []struct {
		name string
		// The src must contain a type named Test.
		// We compare the want with all .Inst[i].fullString()
		// and report an error if it is not fonud.
		src     string
		imports [][2]string
		want    string
		trace   bool
	}{
		{
			name: "inst empty type",
			src: `
				type _ Test {}
				func [foo ^Int Test]
			`,
			want: "type Int Test {}",
		},
		{
			name: "inst field",
			src: `
				type T Test {x: T}
				func [foo ^Int Test]
			`,
			want: "type Int Test { x: Int }",
		},
		{
			name: "inst case",
			src: `
				type T Test {x: T | y}
				func [foo ^Int Test]
			`,
			want: "type Int Test { x: Int | y }",
		},
		{
			name: "inst virt",
			src: `
				type T Test {[foo: T]}
				func [foo ^Int Test]
			`,
			want: "type Int Test { [foo: Int] }",
		},
		{
			name: "inst cyclic and-type",
			src: `
				type T Test {x: T next: T Test& y: T}
				func [foo ^Int Test]
			`,
			want: "type Int Test { x: Int next: Int Test& y: Int }",
		},
		{
			name: "inst cyclic or-type",
			src: `
				type T Test {leaf: T | node: T Test&}
				func [foo ^Int Test]
			`,
			want: "type Int Test { leaf: Int | node: Int Test& }",
		},
		{
			name: "inst cyclic virt-type",
			src: `
				type T Test {[foo: T] [bar: T Test&]}
				func [foo ^Int Test]
			`,
			want: "type Int Test { [foo: Int] [bar: Int Test&] }",
		},
		{
			name: "constraint",
			src: `
				type (T T Eq) Test {x: T}
				type T Eq {[= T& ^Bool]}
				func [foo ^Int Test]
			`,
			want: "type Int Test { x: Int }",
		},
		{
			name: "constraint cycle",
			src: `
				type (T T Foo) Test {[foo] [bar]}
				type (T T Test) Foo {[foo] [bar]}
				meth Int [foo]
				meth Int [bar]
				func [foo ^Int Test]
			`,
			want: "type Int Test { [foo] [bar] }",
		},
		{
			// TODO: something isn't substituted correctly in cyclic constraints?
			name: "SKIP: constraint cycle 2",
			src: `
				type (T T Foo) Test {[foo: T] [bar ^T]}
				type (T T Test) Foo {[foo: T] [bar ^T]}
				func [foo ^Int Test]
			`,
			want: "type Int Test { [foo: Int] [bar ^Int] }",
		},
		{
			name: "alias type",
			src: `
				type T Test := T Array.
				func [foo ^Int Test]
			`,
			want: "type Int Array {}",
		},
		{
			name: "alias type with partially bound target type",
			src: `
				type T Test := (T, String) Fun.
				func [foo ^Int Test]
			`,
			want: "type (Int, String) Fun {}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if strings.HasPrefix(test.name, "SKIP") {
				t.Skip()
			}
			p := syn.NewParser("#test")
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
			typ := findTestType(mod, "Test")
			for typ != nil && typ.Alias != nil {
				typ = typ.Alias.Type.Def
			}
			if typ == nil {
				t.Fatal("type Test not found")
			}
			var insts []string
			for _, inst := range typ.Insts {
				got := inst.fullString()
				if got == test.want {
					return
				}
				insts = append(insts, got)
			}
			t.Errorf("got %s, want %s", insts, test.want)
		})
	}
}

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
				func T [foo: _ T]
			`,
			err: "",
		},
		{
			name: "infer arg expr error",
			src: `
				val test := [
					u Unknown := {}.
					foo: u.
				]
				func T [foo: _ T]
			`,
			err: "Unknown not found",
		},
		{
			name: "not all vars bound",
			src: `
				val test := [ foo: 5 ]
				func T [foo: _ Int]
			`,
			err: "T defined and not used",
		},
		{
			name: "return unify fails",
			src: `
				val test String := [ foo ]
				func T [foo ^T Array]
			`,
			err: "type mismatch",
		},
		{
			name: "param unify fails",
			src: `
				val test := [ foo: "string" ]
				func T [foo: _ T Array]
			`,
			err: "type mismatch",
		},
		{
			name: "multi-binding type mismatch",
			src: `
				val test Rune := [ foo: "string" ]
				func T [foo: _ T ^T]
			`,
			err: "have String, want Int32",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestInstCall(t *testing.T) {
	tests := []struct {
		name string
		// The src must contain a val named test with a call statement.
		// The .String() of the .Fun of the first Msg
		// of the first call statement is compared to want,
		// or the string <nil> if the .Fun is nil.
		src     string
		imports [][2]string
		want    string
		trace   bool
	}{
		{
			name: "fully grounded",
			src: `
				val test := [ 1 + 2 ]
			`,
			want: "Int [+ _ Int& ^Int]",
		},
		{
			name: "ground receiver subs return",
			src: `
				val test := [
					recv String Array := {}.
					recv at: 2
				]
			`,
			want: "String Array [at: _ Int ^String&]",
		},
		{
			name: "ground receiver subs parm",
			src: `
				val test := [
					recv String Array := {}.
					recv at: 2 put: "hello"
				]
			`,
			want: "String Array [at: _ Int put: _ String]",
		},
		{
			name: "ground multi-type-param receiver",
			src: `
				val test := [
					recv (String, Float) Map := {}.
					recv at: "pi" put: 3.14
				]
				type (_, _) Map {}
				meth (K, V) Map [at: _ K put: _ V]
			`,
			want: "(String, Float) Map [at: _ String put: _ Float]",
		},
		{
			name: "ground imported receiver type",
			src: `
				import "map"
				val test := [
					recv (String, Float) #map Map := {}.
					recv #map at: "pi" put: 3.14
				]
			`,
			imports: [][2]string{
				{"map", `
					Type (_, _) Map {}
					Meth (K, V) Map [at: _ K put: _ V]
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
				func T [foo: _ T]
			`,
			want: "[foo: _ String]",
		},
		{
			name: "ground fun parameter complex type",
			src: `
				val test := [
					arg String Array Array := {}.
					foo: arg
				]
				func T [foo: _ T Array Array]
			`,
			want: "[foo: _ String Array Array]",
		},
		{
			name: "map method",
			src: `
				val test String Array := [
					recv Int8 Array := {}.
					recv map: [:i Int8 | "foo"]
				]
				meth T Array R [map: _ (T, R) Fun ^R Array]
			`,
			want: "Int8 Array [map: _ (Int8, String) Fun ^String Array]",
		},
		{
			name: "reduce method",
			src: `
				val test String := [
					recv Int8 Array :={}.
					recv init: "hello" fold: [:i Int8 :s String | "foo"]
				]
				meth T Array R [init: _ R fold: _ (T, R, R) Fun ^R]
			`,
			want: "Int8 Array [init: _ String fold: _ (Int8, String, String) Fun ^String]",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := syn.NewParser("#test")
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
		})
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
