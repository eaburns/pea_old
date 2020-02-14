package types

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
)

func TestInstType(t *testing.T) {
	t.Parallel()
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
			name: "constraint cycle 2",
			src: `
				type (T T Foo) Test {[foo: T] [bar ^T]}
				type (T T Test) Foo {[foo: T] [bar ^T]}
				func [foo ^Int Test]
				meth Int [foo: _ Int]
				meth Int [bar ^Int]
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
			want: "type (Int, String) Fun { [value: Int ^String] }",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if strings.HasPrefix(test.name, "SKIP") {
				t.Skip()
			}
			p := ast.NewParser("/test/test")
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
	t.Parallel()
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
		{
			name: "subbed virtual call",
			src: `
				// The call to bar: will sub the virtual call to Foore foo.
				val test String := [bar: 5]
				meth Int [foo ^String | ^"string"]
				type T Fooer {[foo ^T]}
				func T [bar: f T Fooer ^T | ^f foo]
			`,
			err: "",
		},
		{
			name: "cannot infer type parm not referenced in Fun.Sig",
			src: `
				type T Bar {[bar ^T]}
				// Since T is not referenced in the method parameters
				// or its return value, we cannot infer it.
				// This is an error.
				func (T, S T Bar) [foo: s S ^S | ^s]
				meth Int [bar ^Int | ^self]
				func [main | foo: 5]
			`,
			err: "unable to infer type parameter T",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestInstCall(t *testing.T) {
	t.Parallel()
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
			want: "Int [+ x Int ^Int]",
		},
		{
			name: "ground receiver subs return",
			src: `
				val test := [
					recv String Array := {}.
					recv at: 2
				]
			`,
			want: "String Array [at: x Int ^String&]",
		},
		{
			name: "ground receiver subs parm",
			src: `
				val test := [
					recv String Array := {}.
					recv at: 2 put: "hello"
				]
			`,
			want: "String Array [at: x Int put: y String]",
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
			want: "Bool [ifTrue: x0 String Fun ifFalse: x1 String Fun ^String]",
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
		{
			// This is reproducing a bug the receiver type was not correctly
			// substituting inside the _body_ of an instantiated type.
			name: "reduce method 2",
			src: `
				val test := [
					a Int Array := {1; 2}.
					a from: 0 fold: [:sum :i | i + sum]
				]

				Meth Bool [ifTrue: f Nil Fun | self ifTrue: f ifFalse: []]

				Meth Int [to: e Int do: f (Int, Nil) Fun |
					self <= e ifTrue: [
						f value: self.
						self + 1 to: e do: f
					]
				]

				Meth T Array U [from: u U fold: f (U, T, U) Fun ^U |
					0 to: self size -1 do: [:i |
						u := f value: u value: (self at: i)
					].
					^u
				]

				Func [sum: a Int Array ^Int |
					^a from: 0 fold: [:s :i | s + i]
				]
			`,
			want: "Int Array [from: u Int fold: f (Int, Int, Int) Fun ^Int]",
		},
		{
			name: "paritally-bound parameterized method",
			src: `
				val test := [
					[:i Int | i < 5] from: 1 whileTrue: [:i | i + 1]
				]
				type T BoolFun := (T, Bool) Fun.
				meth T BoolFun [from: t T whileTrue: f (T, T) Fun |
					(self value: t) ifTrue: [
						self from: (f value: t) whileTrue: f
					]
				]
				meth Bool [ifTrue: _ Nil Fun]

			`,
			want: "(Int, Bool) Fun [from: t Int whileTrue: f (Int, Int) Fun]",
		},
		{
			// This test is trying to catch a regression.
			// A previous implementation of Recv instantiation
			// created the substitution map looping over the args.
			// However, that is not sufficient in this case,
			// because the Param's use in the alias target
			// is nested in an argument of the aliased type.
			// We need a full, recursive unify to create the sub map.
			name: "paritally-bound parameterized method with nesting",
			src: `
				val test := [
					[some: 3] whileSome: [:i | i + 1. {}]
				]
				type T OptFun := T? Fun.
				meth T OptFun [whileSome: f (T&, Nil) Fun |
					self value ifSome: [:t |
						f value: t.
						self whileSome: f.
					]
				]
				type T? {none | some: T}
				func T [some: t T ^T? | ^{some: t}]
				meth T? [ifSome: _ (T&, Nil) Fun]
			`,
			want: "Int #test ? Fun [whileSome: f (Int&, Nil) Fun]",
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
			cfg := Config{
				Importer: testImporter(test.imports),
				Trace:    test.trace,
			}
			mod, errs := Check(p.Mod(), cfg)
			if len(errs) > 0 {
				t.Fatalf("failed to check source: %v", errs)
			}
			val := findTestVal(mod, "test")
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

func TestSubStmts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		// We basically just test that none of the panics fire.
		src     string
		imports [][2]string
		trace   bool
	}{
		{
			name: "int",
			src: `
				func (T Int Eq) [foo: t T ^Bool |
					^t = 12
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: 4]
			`,
		},
		{
			name: "float",
			src: `
				func (T Float Eq) [foo: t T ^Bool |
					^t = 12.0
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: 4.1]
			`,
		},
		{
			name: "string",
			src: `
				func (T String Eq) [foo: t T ^Bool |
					^t = "world"
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: "hello"]
				meth String [= _ String & ^Bool]
			`,
		},
		{
			name: "val ident",
			src: `
				val x := [5]
				func (T Int Eq) [foo: t T ^Bool |
					^t = x
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: 4]
			`,
		},
		{
			name: "fun parm ident",
			src: `
				func (T Int Eq) [foo: t T bar: x Int ^Bool |
					^t = x
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: 4 bar: 5]
			`,
		},
		{
			name: "local ident",
			src: `
				func (T Int Eq) [foo: t T ^Bool |
					x := 5.
					^t = x
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: 4]
			`,
		},
		{
			name: "field ident",
			src: `
				meth Test (T Int Eq) [foo: t T ^Bool |
					^t = x
				]
				type Test {x: Int}
				type T Eq {[= T& ^Bool]}
				val _ := [
					t Test := {x: 5}.
					t foo: 4
				]
			`,
		},
		{
			name: "block",
			src: `
				func (T Int Eq) [foo: t T ^Bool |
					^[ :x Int |
						y := x.
						t = y
					] value: 4
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: 4]
			`,
		},
		{
			name: "constructor",
			src: `
				func (T Int Eq) [foo: t T ^Bool |
					x Test := {x: 5}.
					^t = x x
				]
				type T Eq {[= T& ^Bool]}
				type Test {x: Int}
				meth Test [x ^Int | ^x]
				val _ := [foo: 4]
			`,
		},
		{
			name: "call",
			src: `
				func (T Int Eq) [foo: t T ^Bool |
					x := 5.
					^t = (x + 6, - 7, neg).
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: 4]
			`,
		},
		{
			name: "ref convert",
			src: `
				func (T Int Eq) [foo: t T ^Bool |
					tt T& := t.
					^tt = 5
				]
				type T Eq {[= T& ^Bool]}
				val _ := [foo: 4]
			`,
		},
		{
			name: "virt convert",
			src: `
				func (T T Eq) [foo: t T ^T Eq | ^t]
				type T Eq {[= T ^Bool]}
				val _ Int Eq := [foo: 4]
			`,
		},
		{
			name: "instantiating adds a new instance",
			src: `
				func T [foo: t T^ T | ^baz: t]
				func T [bar: t T ^T | ^t]
				func T [baz: t T ^T | ^bar: t]
				val _ Int := [foo: 4]
			`,
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
			cfg := Config{
				Importer: testImporter(test.imports),
				Trace:    test.trace,
			}
			func() {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("panicked: %v", p)
					}
				}()
				if _, errs := Check(p.Mod(), cfg); len(errs) > 0 {
					t.Fatalf("failed to check source: %v", errs)
				}
			}()
		})
	}
}

// Tests that instantiating a function body is able to add a function instance,
// which will then also be correctly instantiated.
func TestRecursiveFunBodyInstantiation(t *testing.T) {
	t.Parallel()
	const src = `
		// Instantiating [foo: Int ^Int] will create a new instance of
		// [baz: Int ^Int], which will create an instance of [bar: Int ^Int].
		// All three of these should be instantiated.
		func T [foo: t T^ T | ^baz: t]
		func T [bar: t T ^T | ^t]
		func T [baz: t T ^T | ^bar: t]
		val _ Int := [foo: 4]
	`
	p := ast.NewParser("/test/test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}

	for _, sel := range [...]string{"foo:", "bar:", "baz:"} {
		f := findTestFun(mod, sel)
		if f == nil {
			t.Errorf("no function %s", sel)
			continue
		}
		var fInt *Fun
		for _, f := range f.Insts {
			if f.TArgs[0].Name == "Int" {
				fInt = f
				break
			}
		}
		if fInt == nil {
			t.Errorf("no function [%s Int ^Int]", sel)
			continue
		}
		if len(fInt.Stmts) == 0 {
			t.Errorf("[%s Int ^Int]'s body is not instantiated", sel)
		}
	}
}

// It is possible that the methods of a type differ between files in the same module.
// This can happen due to different Import statements.
// This tests that instances of the same parameterized function
// differ between calls in different files where the type parameters
// have different methods.
func TestDifferentInstsFromDifferentFiles(t *testing.T) {
	t.Parallel()
	const file0 = `
		type Fooer {[foo]}
		func (T Fooer) [doFoo: f T | f foo]
	`
	const file1 = `
		Import "bar"
		// doFoo should use Int #bar foo in its call to doFoo: 5.
		val file1Val := [doFoo: 5]
	`
	const file2 = `
		Import "baz"
		// doFoo should use Int #baz foo in its call to doFoo: 5.
		val file2Val := [doFoo: 5]
	`
	// The bar and baz modules have different Int [foo] methods.
	imports := [][2]string{
		{"bar", "Meth Int [foo]"},
		{"baz", "Meth Int [foo]"},
	}
	p := ast.NewParser("/test/test")
	for i, src := range [...]string{file0, file1, file2} {
		path := fmt.Sprintf("file%d", i)
		if err := p.Parse(path, strings.NewReader(src)); err != nil {
			t.Fatalf("failed to parse source file%d: %s", i, err)
		}
	}
	cfg := Config{Importer: testImporter(imports)}
	mod, errs := Check(p.Mod(), cfg)
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}

	file1DoFoo := findTestVal(mod, "file1Val").Init[0].(*Call).Msgs[0].Fun
	file1Foo := file1DoFoo.Stmts[0].(*Call).Msgs[0].Fun
	if file1Foo.ModPath != "bar" {
		t.Errorf("expected file1 to call Int #bar foo, got #%s", file1Foo.ModPath)
	}

	file2DoFoo := findTestVal(mod, "file2Val").Init[0].(*Call).Msgs[0].Fun
	file2Foo := file2DoFoo.Stmts[0].(*Call).Msgs[0].Fun
	if file2Foo.ModPath != "baz" {
		t.Errorf("expected file2 to call Int #bar foo, got #%s", file2Foo.ModPath)
	}
}

// Tests that Fun.Insts contains only grounded function instances.
func TestFunInsts_Grounded(t *testing.T) {
	t.Parallel()
	const src = `
		func T [foo: t T^ T | ^t]

		// [foo: U] should not be in Fun.Insts.
		func U [bar: u U | foo: u]

		// [foo: Int] should be in Fun.Insts.
		val _ Int := [foo: 5]
	`
	p := ast.NewParser("/test/test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}
	foo := findTestFun(mod, "foo:")
	if foo == nil {
		t.Fatal("foo: not found")
	}
	if len(foo.Insts) != 1 {
		t.Fatalf("foo: len(Insts)=%d, want 1", len(foo.Insts))
	}
	if got := foo.Insts[0].TArgs[0].Name; got != "Int" {
		t.Errorf("foo: Insts[0] arg is %s, want Int", got)
	}
}

func TestFunInst_DeclInstGetsNilStmts(t *testing.T) {
	t.Parallel()
	const src = `
		Func T [use: _ T]
		val _ := [x := 1. use: x]
	`
	p := ast.NewParser("/test/test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}
	use := findTestFun(mod, "use:")
	if use == nil {
		t.Fatal("use: not found")
	}
	if len(use.Insts) != 1 {
		t.Fatalf("use: len(Insts)=%d, want 1", len(use.Insts))
	}
	if use.Insts[0].Stmts != nil {
		t.Errorf("use.Insts[0].Stmts != nil")
	}
}

// Tests that we properly insert conversions params and returns
// of grounded constraint functions.
func TestFunInsts_ConvertArgsAndReturn(t *testing.T) {
	t.Parallel()
	const src = `
		type Foo {
			[foo: Int bar: Int& ^Int]
		}
		func (T Foo) [baz: t T ^Int |
			// When T is instantiated with Int,
			// We need to to convert he foo: arg to a ref.
			// We need to convert the bar: arg to a value.
			// We need to convert the return to a value.
			^t foo: 5 bar: 6
		]
		meth Int [foo: _ Int& bar: _ Int ^Int&]
		val _ := [baz: 5]
	`
	p := ast.NewParser("/test/test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}
	baz := findTestFun(mod, "baz:")

	var bazInt *Fun
	for _, inst := range baz.Insts {
		if inst.TArgs[0].Name == "Int" {
			bazInt = inst
			break
		}
	}
	if bazInt == nil {
		t.Fatal("[baz: Int] not found")
	}

	ret, ok := bazInt.Stmts[0].(*Ret).Expr.(*Convert)
	if !ok {
		t.Fatalf("foo:bar: ret is a %T, want *Convert", bazInt.Stmts[0].(*Ret).Expr)
	} else if ret.Ref != -1 {
		t.Errorf("foo:bar: ret Ref=%d, want -1", ret.Ref)
	} else if ret.typ.String() != "Int" {
		t.Errorf("foo:bar: ret typ=%s, want Int", ret.typ.String())
	}
	msg := ret.Expr.(*Call).Msgs[0]
	if fooArg, ok := msg.Args[0].(*Convert); !ok {
		t.Errorf("foo: arg is a %T, want *Convert", msg.Args[0])
	} else if fooArg.Ref != 1 {
		t.Errorf("foo: arg Ref=%d, want 1", fooArg.Ref)
	} else if fooArg.typ.String() != "Int&" {
		t.Errorf("foo: arg typ=%s, want Int&", fooArg.typ.String())
	}
	// The convert node in the source should be removed,
	// so that the argument is just the Int value itself.
	if barArg, ok := msg.Args[1].(*Int); !ok {
		t.Errorf("bar: arg is a %T, want *Int", msg.Args[1])
	} else if barArg.typ.String() != "Int" {
		t.Errorf("bar: arg typ=%s, want Int", barArg.typ.String())
	}
}

// Tests that Fun.Insts for a non-parameterized function contains the function def.
func TestFunInsts_NonParamFunInstsContainsDef(t *testing.T) {
	t.Parallel()
	const src = `
		func [foo: i Int ^Int | ^i]
	`
	p := ast.NewParser("/test/test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}
	foo := findTestFun(mod, "foo:")
	if foo == nil {
		t.Fatal("foo: not found")
	}
	if len(foo.Insts) != 1 {
		t.Fatalf("foo: len(Insts)=%d, want 1", len(foo.Insts))
	}
	if foo.Insts[0] != foo {
		t.Errorf("foo: Insts[0] is not the definition")
	}
}

// Tests that Type.Insts for a non-parameterized function contains the type def.
func TestTypeInsts_NonParamTypeInstsContainsDef(t *testing.T) {
	t.Parallel()
	const src = `
		type Foo {x: Int}
	`
	p := ast.NewParser("/test/test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check source: %v", errs)
	}
	foo := findTestType(mod, "Foo")
	if foo == nil {
		t.Fatal("Foo not found")
	}
	if len(foo.Insts) != 1 {
		t.Fatalf("Foo len(Insts)=%d, want 1", len(foo.Insts))
	}
	if foo.Insts[0] != foo {
		t.Errorf("Foo Insts[0] is not the definition")
	}
}

func findTestFun(mod *Mod, sel string) *Fun {
	for _, def := range mod.Defs {
		if f, ok := def.(*Fun); ok && f.Sig.Sel == sel {
			return f
		}
	}
	return nil
}

func findTestVal(mod *Mod, name string) *Val {
	for _, def := range mod.Defs {
		if v, ok := def.(*Val); ok && v.Var.Name == name {
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
