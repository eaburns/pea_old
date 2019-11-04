package sem

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pretty"
)

func TestBugRegressions(t *testing.T) {
	tests := []errorTest{
		{
			name: "1",
			src: `
				// There is a parameterized type as
				// the field of a parameterized type.
				// When instantiating T Foo,
				// T Bar will have already been instantiated.
				// We should still re-substitute it with the new T.
				type T Foo { data: T Bar Array }
				type T Bar { x: T }
				meth T Foo [ blah: t T |
					data at: 5 put: {x: t}
				]
			`,
			err: "",
		},
		{
			name: "2",
			src: `
				func T [newArray: _ Int init: _ (Int, T) Fun ^T Array]
				type (X, Y) Bucket := (X, Y) Elem Array.
				type (X, Y) Elem {x: X y: Y}
				type (X, Y) Table {data: (X, Y) Bucket Array}
				meth (_, _) Table [foo |
					data := newArray: 100 init: [:_ | {}].
				]
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

// This tests that Call.Recv is set to the receiver,
// and that if it is a conversion it's type is set.
// This is to catch a regression from a previous bug fix.
func TestCallRecvIsSet(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// want is the Recv.Type().String() of the first call
		// in the body of the val named test
		want  string
		trace bool
	}{
		{
			name: "no ref conversion",
			src: `
				val test := [
					x Int& := 5.
					x foo.
				]
				meth Int [foo]
			`,
			want: "Int&",
		},
		{
			name: "add ref conversion",
			src: `
				val test := [
					5 foo.
				]
				meth Int [foo]
			`,
			want: "Int&",
		},
		{
			name: "remove ref conversion",
			src: `
				val test := [
					x Int& & & & := 5.
					x foo.
				]
				meth Int [foo]
			`,
			want: "Int&",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := ast.NewParser("#test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Fatalf("failed to parse source: %s", err)
			}
			mod, errs := Check(p.Mod(), Config{Trace: test.trace})
			if len(errs) > 0 {
				t.Fatalf("failed to check source: %v", errs)
			}
			val := findTestVal(mod)
			if val == nil {
				t.Fatal("val test not found")
			}
			var call *Call
			for _, stmt := range val.Init {
				var ok bool
				if call, ok = stmt.(*Call); ok {
					break
				}
			}
			if call == nil {
				t.Fatal("no call")
			}
			if call.Recv == nil {
				t.Fatal("nil recv")
			}
			if got := call.Recv.Type().String(); got != test.want {
				t.Errorf("got %s, want %s", got, test.want)
			}
		})
	}
}

func TestImportError(t *testing.T) {
	tests := []errorTest{
		{
			name: "no import",
			src: `
				import "missing"
			`,
			err: "not found",
		},
		{
			name: "unused import",
			src: `
				import "foo"
			`,
			imports: [][2]string{
				{"foo", ""},
			},
			err: "foo imported and not used",
		},
		{
			name: "unused Import",
			src: `
				Import "foo"
			`,
			imports: [][2]string{
				{"foo", ""},
			},
			err: "foo imported and not used",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestPrivate(t *testing.T) {
	tests := []errorTest{
		{
			name: "private type",
			src: `
				import "in"
				type Test := #in Private.
			`,
			imports: [][2]string{
				{"in", `
					type Private {}
				`},
			},
			err: "type #in Private not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestRedefError(t *testing.T) {
	tests := []errorTest{
		{
			name: "val and val",
			src: `
				val Abc := [5]
				val Abc := [6]
			`,
			err: "Abc redefined",
		},
		{
			name: "val and type",
			src: `
				val Abc := [5]
				type Abc {}
			`,
			err: "Abc redefined",
		},
		{
			name: "val and unary func",
			src: `
				val Abc := [5]
				func [Abc ^Int]
			`,
			err: "Abc redefined",
		},
		{
			name: "type and val",
			src: `
				type Abc {}
				val Abc := [6]
			`,
			err: "Abc redefined",
		},
		{
			name: "type and type",
			src: `
				type Abc {}
				type Abc {}
			`,
			err: "Abc redefined",
		},
		{
			name: "type and different arity type is OK",
			src: `
				type Abc {}
				type _ Abc {}
				type (_, _) Abc {}
			`,
			err: "",
		},
		{
			name: "type and same arity type",
			src: `
				type Abc {}
				type _ Abc {}
				type _ Abc {}
			`,
			err: "\\(1\\)Abc redefined",
		},
		{
			name: "type and unary func",
			src: `
				type Abc {}
				func [Abc ^Int]
			`,
			err: "Abc redefined",
		},
		{
			name: "unary func and val",
			src: `
				func [Abc ^Float]
				val Abc := [6]
			`,
			err: "Abc redefined",
		},
		{
			name: "unary func and type",
			src: `
				func [Abc ^Float]
				type Abc {}
			`,
			err: "Abc redefined",
		},
		{
			name: "unary func and unary func",
			src: `
				func [Abc ^Float]
				func [Abc ^Int]
			`,
			err: "Abc redefined",
		},
		{
			name: "nary func and nary func",
			src: `
				func [foo: _ Int bar: _ Float]
				func [foo: _ Int bar: _ Float]
			`,
			err: "foo:bar: redefined",
		},
		{
			name: "nary func and different nary func is OK",
			src: `
				func [foo: _ Int bar: _ Float]
				func [foo: _ Int bar: _ Float baz: _ String]
				func [bar: _ Int foo: _ Float]
			`,
			err: "",
		},
		{
			name: "no redef with imported",
			src: `
				import "xyz"
				Val Abc := [#xyz Abc]
			`,
			imports: [][2]string{
				{
					"xyz",
					`
						Val Abc := [6]
					`,
				},
			},
			err: "",
		},
		{
			name: "method redefinition",
			src: `
				meth Int64 [foo |]
				meth Int64 [foo |]
			`,
			err: "method Int64 foo redefined",
		},
		{
			name: "method redefinition through an alias",
			src: `
				meth Int32 [foo |]
				meth Rune [foo |]
			`,
			err: "method Int32 foo redefined",
		},
		{
			name: "method redefinition through multiple aliases",
			src: `
				meth Rune [foo |]
				meth Abc [foo |]
				type Abc := Def.
				type Def := Ghi.
				type Ghi := Int32.
			`,
			err: "method Int32 foo redefined",
		},
		{
			name: "binary method redefinition",
			src: `
				meth Int [@@ _ Int |]
				meth Int [@@ _ Int |]
			`,
			err: "method Int @@ redefined",
		},
		{
			name: "nary method redefinition",
			src: `
				meth Int [foo: _ String bar: _ Int |]
				meth Int [foo: _ String bar: _ Int |]
			`,
			err: "method Int foo:bar: redefined",
		},
		{
			name: "method redefinition with different param types",
			src: `
				meth Int [foo: _ Int bar: _ Float |]
				meth Int [foo: _ String bar: _ Int |]
			`,
			err: "method Int foo:bar: redefined",
		},
		{
			name: "method redefinition with receiver type params",
			src: `
				meth T Array [foo: _ String bar: _ Int |]
				meth U Array [foo: _ String bar: _ Int |]
			`,
			err: "method \\(1\\)Array foo:bar: redefined",
		},
		{
			name: "method not redefined when differing receiver",
			src: `
				type _ Map {}
				meth _ Map [foo]
				type (_, _) Map {}
				meth (_, _) Map [foo]
			`,
			err: "",
		},
		{
			name: "method not redefined on same-name type from different imports",
			src: `
				import "foo"
				import "bar"
				meth #foo Abc [baz |]
				meth #bar Abc [baz |]
			`,
			imports: [][2]string{
				{"foo", "Type Abc {}"},
				{"bar", "Type Abc {}"},
			},
			err: "",
		},
		{
			name: "built-in case method redefined",
			src: `
				meth T? [ifNone: _ Int ifSome: _ String |]
				type T? { none | some: T }
			`,
			err: "method \\(1\\)\\? ifNone:ifSome: redefined",
		},
		{
			name: "virtual method redefined",
			src: `
				meth Foo [bar |]
				type Foo { [bar] }
			`,
			err: "method Foo bar redefined",
		},
		{
			name: "type field",
			src: `
				type Test { a: Int a: Float }
			`,
			err: "field a redefined",
		},
		{
			name: "type case",
			src: `
				type Test { a | a }
			`,
			err: "case a redefined",
		},
		{
			name: "type case:",
			src: `
				type Test { a: Int | a: Float }
			`,
			err: "case a: redefined",
		},
		{
			name: "type case not redefined with case:",
			src: `
				type Test { a | a: Float }
			`,
			err: "",
		},
		{
			name: "type virt unary selector",
			src: `
				type Test { [foo] [foo] }
			`,
			err: "virtual method foo redefined",
		},
		{
			name: "type virt binary selector",
			src: `
				type Test { [* Int] [* String] }
			`,
			err: "virtual method \\* redefined",
		},
		{
			name: "type virt n-ary selector",
			src: `
				type Test { [foo: Int bar: Float] [foo: String bar: Rune] }
			`,
			err: "virtual method foo:bar: redefined",
		},
		{
			name: "Import val redef OK",
			src: `
				Import "foo"
				Import "bar"

				// to rid unused import errors
				val useImports := [ #foo x. #bar x. ]
			`,
			imports: [][2]string{
				{
					"foo",
					"Val x := [5]",
				},
				{
					"bar",
					"Val x := [5]",
				},
			},
			err: "",
		},
		{
			name: "Import type redef OK",
			src: `
				Import "foo"
				Import "bar"

				// to rid unused import errors
				func [useFoo ^#foo Point]
				func [useBar ^#bar Point]
			`,
			imports: [][2]string{
				{
					"foo",
					"Type Point {x: Int y: Int}",
				},
				{
					"bar",
					"Type Point {x: String y: Float}",
				},
			},
			err: "",
		},
		{
			name: "Import type versus value redef OK",
			src: `
				Import "foo"
				Import "bar"

				// to rid unused import errors
				func [useFoo | #foo Point]
				func [useBar ^#bar Point]
			`,
			imports: [][2]string{
				{
					"foo",
					"Val Point := [5]",
				},
				{
					"bar",
					"Type Point {x: String y: Float}",
				},
			},
			err: "",
		},
		{
			name: "Import non-exported, no refed",
			src: `
				Import "foo"
				Import "bar"

				// to rid unused import errors
				func [useFoo | #foo Abc]
				func [useBar ^#bar Def]
			`,
			imports: [][2]string{
				{
					"foo",
					`
						val Point := [5]
						Val Abc := [5]
						type Def {}
					`,
				},
				{
					"bar",
					`
						type Point {x: String y: Float}
						val Abc := [5]
						Type Def {}
					`,
				},
			},
			err: "",
		},
		{
			name: "Import method redef OK",
			src: `
				Import "foo"
				Import "bar"

				// to rid unused import errors
				val useImports := [
					5 #foo foo: "" bar: 5.
					5 #bar foo: "" bar: 5.
				]
			`,
			imports: [][2]string{
				{
					"foo",
					"Meth Int [foo: _ String bar: _ Int]",
				},
				{
					"bar",
					"Meth Int [foo: _ String bar: _ Int]",
				},
			},
			err: "",
		},
		{
			name: "Import method non-dup with diff recv",
			src: `
				Import "foo"
				Import "bar"

				// to rid unused import errors
				val useImports := [
					5 #foo foo: "" bar: 5.
					5.0 #bar foo: "" bar: 5.
				]
			`,
			imports: [][2]string{
				{
					"foo",
					"Meth Int [foo: _ String bar: _ Int]",
				},
				{
					"bar",
					"Meth Float [foo: _ String bar: _ Int]",
				},
			},
			err: "",
		},
		{
			name: "Import method non-dup with diff recv, but same name",
			src: `
				Import "foo"
				Import "bar"

				// to rid unused import errors
				val useImports := [#foo useMe. #bar useMe.]
			`,
			imports: [][2]string{
				{
					"foo",
					`
						Import "baz"
						// #baz Test
						Meth Test [foo: _ String bar: _ Int]
						Val useMe := [4]
					`,
				},
				{
					"bar",
					`
						Import "qux"
						// #qux Test
						Meth Test [foo: _ String bar: _ Int]
						Val useMe := [4]
					`,
				},
				{
					"baz",
					"Type Test {}",
				},
				{
					"qux",
					"Type Test {}",
				},
			},
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestValDef(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				val _ Int := [5]
			`,
			err: "",
		},
		{
			name: "type not found",
			src: `
				val _ Unknown := [5]
			`,
			err: "type Unknown not found",
		},
		{
			name: "type constraints not met",
			src: `
				val _ Int Test := [5]
				type (_ Foo) Test {}
				type Foo {[foo]}
			`,
			err: "method Int foo not found",
		},
		{
			name: "type mismatch",
			src: `
				val _ Int := ["string"]
			`,
			err: "type mismatch: have String, want Int",
		},
		{
			name: "infer simple type",
			src: `
				val x := ["string"]
				val y Int := [x]
			`,
			err: "type mismatch: have String, want Int",
		},
		{
			name: "infer complex type",
			src: `
				val x := [foo]
				val y Int := [x]
				func [foo ^Int Array Array]
			`,
			err: "type mismatch: have Int Array Array, want Int",
		},
		{
			name: "infer nil: no statements",
			src: `
				val x := []
				val y Int := [x]
			`,
			err: "type mismatch: have Nil, want Int",
		},
		{
			name: "infer nil: final statement",
			src: `
				val x := [_ := 5]
				val y Int := [x]
			`,
			err: "type mismatch: have Nil, want Int",
		},
		{
			name: "statement error",
			src: `
				val x := [z := {}. z]
			`,
			err: "cannot infer constructor type",
		},
		{
			name: "simple 1 cycle",
			src: `
				val x := [x]
			`,
			err: "initialization cycle",
		},
		{
			name: "simple 2 cycle",
			src: `
				val x := [y]
				val y := [x]
			`,
			err: "initialization cycle",
		},
		{
			name: "simple 3 cycle",
			src: `
				val x := [5 + y]
				val y := [y / 6 + z neg]
				val z := [true ifTrue: [x] ifFalse: [4]]
			`,
			err: "initialization cycle",
		},
		{
			name: "1 cycle through calls",
			src: `
				val x := [foo]
				func [foo | x]
			`,
			err: "initialization cycle",
		},
		{
			name: "3 cycle through calls",
			src: `
				val x String := [foo: 5]
				func [foo: _ Int ^String| ^y]
				val y String := [bar]
				func [bar ^String | ^5 baz: 1]
				meth Int [baz: _ String ^String | ^z]
				val z String := [x]
			`,
			err: "initialization cycle",
		},
		{
			name: "recursive functions are not a cycle",
			src: `
				val x := [foo]
				func [foo | bar]
				func [bar | foo]
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestFuncDef(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				func [foo |]
			`,
			err: "",
		},
		{
			name: "unknown param type",
			src: `
				func [foo: _ Unknown]
			`,
			err: "type Unknown not found",
		},
		{
			name: "unknown return type",
			src: `
				func [foo ^Unknown]
			`,
			err: "type Unknown not found",
		},
		{
			name: "param type constraint not met",
			src: `
				func [foo: _ Int Test]
				type (_ Foor) Test {}
				type Foor {[foo]}
			`,
			err: "method Int foo not found",
		},
		{
			name: "return type constraint not met",
			src: `
				func [foo ^Int Test]
				type (_ Foor) Test {}
				type Foor {[foo]}
			`,
			err: "method Int foo not found",
		},
		{
			name: "return type constraint not met",
			src: `
				func [foo ^Int Test]
				type (_ Foor) Test {}
				type Foor {[foo]}
			`,
			err: "method Int foo not found",
		},
		{
			name: "missing return",
			src: `
				func [foo ^Int | _ := 5]
			`,
			err: "missing return at the end of foo",
		},
		{
			name: "missing return: no statemets",
			src: `
				func [foo ^Int |]
			`,
			err: "missing return at the end of foo",
		},
		{
			name: "no missing return for decl",
			src: `
				func [foo ^Int]
			`,
			err: "",
		},
		{
			name: "type parameter: OK",
			src: `
				func T [foo ^T]
			`,
			err: "",
		},
		{
			name: "constrained type parameter: OK",
			src: `
				func (T Fooer) [foo ^T]
				type Fooer {[foo]}
			`,
			err: "",
		},
		{
			name: "constrained type parameter: bad constraint",
			src: `
				func (T T Fooer) [xyz ^T]
				type (_ Barer) Fooer {[foo]}
				type Barer {[bar]}
			`,
			err: "method T bar not found",
		},
		{
			name: "unused type variable",
			src: `
				func T [xyz]
			`,
			err: "T defined and not used",
		},
		{
			name: "illegal _ type variable",
			src: `
				func _ [xyz]
			`,
			err: "illegal function type variable name",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestMethDef(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				meth Int [foo |]
			`,
			err: "",
		},
		{
			name: "unknown receiver type",
			src: `
				meth Unknown [foo]
			`,
			err: "type Unknown not found",
		},
		{
			name: "receiver constraint not met",
			src: `
				meth (T T Fooer) Test [foo]
				type _ Test {}
				type (_ Barer) Fooer {[foo]}
				type Barer {[bar]}
			`,
			err: "method T bar not found",
		},
		{
			name: "reference receiver is not allowed",
			src: `
				meth T& [foo]
			`,
			err: "cannot add a method to &",
		},
		{
			name: "self has expected type",
			src: `
				meth _ Array [foo | _ Int := self ]
			`,
			err: "have _ Array&, want Int",
		},
		{
			name: "alias receiver with bound type arg",
			src: `
				meth IntArray [foo | _ String := self at: 0 ]
				type IntArray := Int Array.
			`,
			err: "have Int&, want String",
		},
		{
			name: "unused type variable",
			src: `
				meth Int T [xyz]
			`,
			err: "T defined and not used",
		},
		{
			name: "illegal _ type variable",
			src: `
				meth Int _ [xyz]
			`,
			err: "illegal function type variable name",
		},
		{
			name: "unused receiver type variable",
			src: `
				meth T Array [xyz]
			`,
			err: "T defined and not used",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestTypeDefParms(t *testing.T) {
	tests := []errorTest{
		{
			name: "constraint ok",
			src: `
				type (K K Key, V) Map {keys: K Array vals: V Array}
				type T Key {[= T& ^Bool]}
			`,
			err: "",
		},
		{
			name: "constraint not met",
			src: `
				type (T T Fooer) Test {f: T}
				type (_ Barer) Fooer {[foo]}
				type Barer {[bar]}
			`,
			err: "method T bar not found",
		},
		{
			name: "unused type variable",
			src: `
				type T Test {}
			`,
			err: "T defined and not used",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestAliasTypeDef(t *testing.T) {
	tests := []errorTest{
		{
			name: "target not found",
			src: `
				type Abc := NotFound.
			`,
			err: "NotFound not found",
		},
		{
			name: "imported not found",
			src: `
				type Abc := #notFound Xyz.
			`,
			err: "#notFound not found",
		},
		{
			name: "imported type not found",
			src: `
				import "xyz"
				type Abc := #xyz NotFound.
			`,
			imports: [][2]string{
				{"xyz", ""},
			},
			err: "NotFound not found",
		},
		{
			name: "arg count mismatch",
			src: `
				type Abc := Int Int.
			`,
			err: "\\(1\\)Int not found",
		},
		{
			name: "no cycle",
			src: `
				type Abc := Int.
			`,
			err: "",
		},
		{
			name: "no cycle, import",
			src: `
				import "xyz"
				type Abc := #xyz Xyz.
			`,
			imports: [][2]string{
				{
					"xyz",
					`
						Type Xyz {}
					`,
				},
			},
			err: "",
		},
		{
			name: "1 cycle",
			src: `
				type Abc := Abc.
			`,
			err: "type alias cycle",
		},
		{
			name: "2 cycle",
			src: `
				type AbcXyz := Abc.
				type Abc := AbcXyz.
			`,
			err: "type alias cycle",
		},
		{
			name: "3 cycle",
			src: `
				type AbcXyz := AbcXyz123.
				type Abc := AbcXyz.
				type AbcXyz123 := Abc.
			`,
			err: "type alias cycle",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestAndTypeDef(t *testing.T) {
	tests := []errorTest{
		{
			name: "empty ok",
			src: `
				type Test {}
			`,
			err: "",
		},
		{
			name: "ok",
			src: `
				type Test {x: Int y: Int}
			`,
			err: "",
		},
		{
			name: "type param ok",
			src: `
				type (X, Y) Pair {x: X y: Y}
			`,
			err: "",
		},
		{
			name: "unknown field type",
			src: `
				type Test {f: Unknown}
			`,
			err: "type Unknown not found",
		},
		{
			name: "field redefined",
			src: `
				type Test {f: Int f: Int}
			`,
			err: "field f redefined",
		},
		{
			name: "fields differing in capitalization are OK",
			src: `
				type Test {field: Int Field: String}
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestOrTypeDef(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				type Test {a | b: Int | c: String | d}
			`,
			err: "",
		},
		{
			name: "type param OK",
			src: `
				type T Test {a | b: T}
			`,
			err: "",
		},
		{
			name: "unknown type",
			src: `
				type Test {a: Unknown}
			`,
			err: "type Unknown not found",
		},
		{
			name: "ident case redefined",
			src: `
				type T Test {a | a}
			`,
			err: "case a redefined",
		},
		{
			name: "identC case redefined",
			src: `
				type _ Test {a: Int | a: Float}
			`,
			err: "case a: redefined",
		},
		{
			name: "ident and identC cases are OK",
			src: `
				type _ Test {a | a: Float}
			`,
			err: "",
		},
		{
			name: "cases differ only in capitalization",
			src: `
				type _ Test {abc: Int | Abc: Float}
			`,
			err: "case abc: redefined",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestVirtTypeDef(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				type Test {[foo] [bar: Int] [baz ^Bool]}
			`,
			err: "",
		},
		{
			name: "type param ok",
			src: `
				type T Test {[foo: T]}
			`,
			err: "",
		},
		{
			name: "unknown param type",
			src: `
				type Test {[foo: Unknown]}
			`,
			err: "type Unknown not found",
		},
		{
			name: "unknown return type",
			src: `
				type Test {[foo ^Unknown]}
			`,
			err: "type Unknown not found",
		},
		{
			name: "method redefined",
			src: `
				type Test {[foo] [foo]}
			`,
			err: "method foo redefined",
		},
		{
			name: "complex method redefined",
			src: `
				type Test {
					[foo: Int bar: String baz: String]
					[foo: String bar: Float baz: Int&]
				}
			`,
			err: "method foo:bar:baz: redefined",
		},
		{
			name: "methods not redefined",
			src: `
				type Test {
					[foo: Int bar: String baz: String]
					[foo: String baz: Int& bar: Float]
					[foo: Int]
					[foo]
					[foo: Int BAR: String baz: String]
				}
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestTypeName(t *testing.T) {
	tests := []errorTest{
		{
			name: "simple type",
			src: `
				val _ Int := [5]
			`,
			err: "",
		},
		{
			name: "unconstrained parameterized type",
			src: `
				val _ Int Test := [{}]
				type _ Test {}
			`,
			err: "",
		},
		{
			name: "unconstrained nested parameterized type",
			src: `
				val _ Int Test Test Test Test := [{}]
				type _ Test {}
			`,
			err: "",
		},
		{
			name: "constraint: OK",
			src: `
				val _ Int Test := [{}]
				type (_ Fooer) Test {}
				type Fooer {[foo]}
				meth Int [foo]
			`,
			err: "",
		},
		{
			name: "constraint: no method",
			src: `
				val _ Int Test := [{}]
				type (_ Fooer) Test {}
				type Fooer {[foo]}
			`,
			err: "method Int foo not found",
		},
		{
			name: "constraint: unexpected return",
			src: `
				val _ Int Test := [{}]
				type (_ Fooer) Test {}
				type Fooer {[foo]}
				meth Int [foo ^Bool|]
			`,
			err: "wrong type for method foo",
		},
		{
			name: "constraint: missing return",
			src: `
				val _ Int Test := [{}]
				type (_ Fooer) Test {}
				type Fooer {[foo ^Bool]}
				meth Int [foo|]
			`,
			err: "wrong type for method foo",
		},
		{
			name: "constraint: mismatching param type",
			src: `
				val _ Int Test := [{}]
				type (_ Fooer) Test {}
				type Fooer {[foo: Int]}
				meth Int [foo: _ String|]
			`,
			err: "wrong type for method foo",
		},
		{
			name: "parameterized constraint",
			src: `
				val _ Int Test := [{}]
				type (T T Eq) Test {}
				type X Eq {[eq: X& ^Bool]}
				meth Int [eq: _ Int& ^Bool]
			`,
			err: "",
		},
		{
			name: "constrained constraint: OK",
			src: `
				val _ Int Test := [{}]
				type (T T Foo) Test {}
				type (_ Bar) Foo {[foo] [bar]}
				type Bar {[bar]}
				meth Int [foo]
				meth Int [bar]
			`,
			err: "",
		},
		{
			name: "constrained constraint: unsatisfied",
			src: `
				val _ Int Test := [{}]
				// Foo doesn't implement Bar,
				// so T==Foo can't be an argument to Foo.
				type (T T Foo) Test {}
				type (_ Bar) Foo {[foo]}
				type Bar {[bar]}
				meth Int [foo]
				meth Int [bar]
			`,
			err: "method T bar not found",
		},
		{
			name: "alias type",
			src: `
				val _ Test := [{}]
				type Test := Test1.
				type Test1 := (Rune, OtherString) Map.
				type OtherString := String.
				type (_, _) Map {}
			`,
			err: "",
		},
		{
			name: "multiple constraints: OK",
			src: `
				val _ (Int, String) Test := [{}]
				type (_ Fooer, _ Barer) Test {}
				type Fooer {[foo]}
				type Barer {[bar]}
				meth Int [foo|]
				meth String [bar|]
			`,
			err: "",
		},
		{
			name: "multiple constraints: second not met",
			src: `
				val _ (Int, String) Test := [{}]
				type (_ Fooer, _ Barer) Test {}
				type Fooer {[foo]}
				type Barer {[bar]}
				meth Int [foo|]
			`,
			err: "method String bar not found",
		},
		{
			name: "imported type name",
			src: `
				import "foo"
				val x #foo Test := [{x: 5}]
			`,
			imports: [][2]string{
				{"foo", "Type Test {x: Int}"},
			},
			err: "",
		},
		{
			name: "imported private type name not found",
			src: `
				import "foo"
				val x #foo Test := [{x: 5}]
			`,
			imports: [][2]string{
				{"foo", "type Test {x: Int}"},
			},
			err: "type #foo Test not found",
		},
		{
			name: "imported type name not found with out mod tag",
			src: `
				import "foo"
				val x Test := [{x: 5}]
			`,
			imports: [][2]string{
				{"foo", "Type Test {x: Int}"},
			},
			err: "type Test not found",
		},
		{
			name: "Imported type name",
			src: `
				Import "foo"
				val x Test := [{x: 5}]
			`,
			imports: [][2]string{
				{"foo", "Type Test {x: Int}"},
			},
			err: "",
		},
		{
			name: "Imported private type name not found",
			src: `
				Import "foo"
				val x Test := [{x: 5}]
			`,
			imports: [][2]string{
				{"foo", "type Test {x: Int}"},
			},
			err: "type Test not found",
		},
		{
			name: "Imported type name with mod tag",
			src: `
				Import "foo"
				val x #foo Test := [{x: 5}]
			`,
			imports: [][2]string{
				{"foo", "Type Test {x: Int}"},
			},
			err: "",
		},
		{
			name: "ambiguous type",
			src: `
				Import "foo"
				Import "bar"
				val x Test := [{x: 5}]
			`,
			imports: [][2]string{
				{"foo", "Type Test {x: Int}"},
				{"bar", "Type Test {x: Float}"},
			},
			err: "ambiguous type",
		},
		{
			name: "unambiguous type 1",
			src: `
				Import "foo"
				Import "bar"
				val x #foo Test := [{x: 5}]
				val _ String := [x]
			`,
			imports: [][2]string{
				{"foo", "Type Test {x: Int}"},
				{"bar", "Type Test {x: Float}"},
			},
			err: "have #foo Test, want String",
		},
		{
			name: "unambiguous type 2",
			src: `
				Import "foo"
				Import "bar"
				val x #bar Test := [{x: 5}]
				val _ String := [x]
			`,
			imports: [][2]string{
				{"foo", "Type Test {x: Int}"},
				{"bar", "Type Test {x: Float}"},
			},
			err: "have #bar Test, want String",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestRetError(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				func [one ^Int | ^1]
			`,
			err: "",
		},
		{
			name: "return outside function",
			src: `
				val x := [^5]
			`,
			err: "return outside of a function or method",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestAssignError(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				Val x := [
					_ Int := 1.
					_ := 2.
				]
			`,
			err: "",
		},
		{
			name: "ok multi-assign",
			src: `
				Val x := [
					_ Int, _, _ := 1 neg, neg, neg.
				]
			`,
			err: "",
		},
		{
			name: "count mismatch: no call",
			src: `
				Val x := [
					_, _, _ := 1.
				]
			`,
			err: "assignment count mismatch: got 1, want 3",
		},
		{
			name: "count mismatch: too few messages",
			src: `
				Val x := [
					_, _, _ := 1 neg, neg
				]
			`,
			err: "assignment count mismatch: got 2, want 3",
		},
		{
			name: "count mismatch: too many messages",
			src: `
				Val x := [
					_, _, _ := 1 neg, neg, neg, neg
				]
			`,
			err: "assignment count mismatch: got 4, want 3",
		},
		{
			name: "bad type name",
			src: `
				Val x := [
					_ Unknown := 1
				]
			`,
			err: "type Unknown not found",
		},
		{
			name: "bad type name and argument count mismatch",
			src: `
				Val x := [
					_, _ Unknown := 1
				]
			`,
			err: "assignment count mismatch: got 1, want 2(.|\n)*type Unknown not found",
		},
		{
			name: "bad assign to a function",
			src: `
				Val x := [
					foo := 1
				]
				func [foo | ]
			`,
			err: "assignment to a function",
		},
		{
			name: "assign to self",
			src: `
				meth Int [ foo | self := 5 ]
			`,
			err: "cannot assign to self",
		},
		{
			name: "assign to shadowed self",
			src: `
				meth Int [ foo: self Int | self := 5 ]
			`,
			err: "",
		},
		{
			name: "shadow a local",
			src: `
				meth Int [foo |
					x Int := 5.
					x String := "hello".
					x := "hello".
					x := 5.
					_ := x.
				]
			`,
			err: "have Int, want String",
		},
		{
			name: "shadow a parm",
			src: `
				meth Int [foo: x Int |
					x String := "hello".
					_ := x.
				]
			`,
			err: "",
		},
		{
			name: "shadow a val",
			src: `
				val x Int := [ 5 ]
				meth Int [foo |
					x String := "hello".
					_ := x.
				]
			`,
			err: "",
		},
		{
			name: "unused in val",
			src: `
				val _ Int := [
					x := 5.
					x := 6.
				]
			`,
			err: "x declared and not used",
		},
		{
			name: "unused in fun",
			src: `
				func [foo |
					x := 5.
					x := 6.
				]
			`,
			err: "x declared and not used",
		},
		{
			name: "unused in block",
			src: `
				func [foo |
					[
						x := 5.
						x := 6.
					]
				]
			`,
			err: "x declared and not used",
		},
		{
			name: "nested use",
			src: `
				func [foo |
					x := 5.
					[[[[[x]]]]]
				]
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestAssignConvert(t *testing.T) {
	tests := []errorTest{
		{
			name: "add a ref",
			src: `
				val _ := [
					x Int := 5.
					_ Int & := x.
				]
			`,
			err: "",
		},
		{
			name: "add a multiple refs",
			src: `
				val _ := [
					x Int := 5.
					_ Int & & & & := x.
				]
			`,
			err: "",
		},
		{
			name: "remove a ref",
			src: `
				val _ := [
					x Int & := 5.
					_ Int := x.
				]
			`,
			err: "",
		},
		{
			name: "remove multiple refs",
			src: `
				val _ := [
					x Int & & & & & := 5.
					_ Int := x.
				]
			`,
			err: "",
		},
		{
			name: "interface conversion",
			src: `
				val _ := [
					x Int := 5.
					_ Int Eq := x.
				]
				type T Eq { [= T& ^Bool] }
			`,
			err: "",
		},
		{
			name: "interface arg type mismatch",
			src: `
				val _ := [
					x Int := 5.
					_ Float Eq := x.
				]
				type T Eq { [= T& ^Bool] }
			`,
			err: "Int does not implement Float Eq",
		},
		{
			name: "interface return type mismatch",
			src: `
				val _ := [
					x Int := 5.
					_ Eq := x.
				]
				type Eq { [= T& ^Int] }
			`,
			err: "Int does not implement Eq",
		},
		{
			name: "interface got a return want none",
			src: `
				val _ := [
					x Int := 5.
					_ Eq := x.
				]
				type Eq { [= T&] }
			`,
			err: "Int does not implement Eq",
		},
		{
			name: "interface want return got none",
			src: `
				val _ := [
					x Int := 5.
					_ Eq := x.
				]
				meth Int [ === _ T&]
				type Eq { [=== T& ^Bool] }
			`,
			err: "Int does not implement Eq",
		},
		{
			name: "deref then interface conversions",
			src: `
				val _ := [
					x Int & & := 5.
					_ Int Eq := x.
				]
				type T Eq { [= T& ^Bool] }
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestMethError(t *testing.T) {
	tests := []errorTest{
		{
			name: "reference receiver",
			src: `
				meth T& [foo]
			`,
			err: "invalid receiver type",
		},
		{
			name: "alias to a ref",
			src: `
				meth Xyz [foo]
				type Xyz := Int & & &.
			`,
			err: "invalid receiver type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestFuncHasNoSelf(t *testing.T) {
	tests := []errorTest{
		{
			name: "no self",
			src: `
				func [foo | self]
			`,
			err: "self not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestAssignToNewVariable(t *testing.T) {
	const src = `
		val x := [
			a := 5.
			use: a.
		]
		func [use: _ Int]
	`
	p := ast.NewParser("#test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check the source: %v", errs)
	}

	val := mod.Defs[0].(*Val)
	if len(val.Locals) != 1 {
		t.Fatalf("got %d locals, expected 1: %v", len(val.Locals), val.Locals)
	}
	l := val.Locals[0]
	assign0 := val.Init[0].(*Assign)
	if assign0.Var != l {
		t.Errorf("assign0.Van (%p) != val.Locals[0] (%p)", assign0.Var, l)
	}
	if l.typ == nil || l.typ.Name != "Int" || l.typ.Mod != "" {
		t.Errorf("got %v, expected Int", l.typ)
	}
}

func TestAssignToExistingVariable(t *testing.T) {
	const src = `
		val x := [
			a := 5.
			a := 6.
			use: a.
		]
		func [use: _ Int]
	`
	p := ast.NewParser("#test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check the source: %v", errs)
	}

	val := mod.Defs[0].(*Val)
	if len(val.Locals) != 1 {
		t.Fatalf("got %d locals, expected 1: %v", len(val.Locals), val.Locals)
	}
	l := val.Locals[0]
	assign0 := val.Init[0].(*Assign)
	assign1 := val.Init[1].(*Assign)
	if assign0.Var != l {
		t.Errorf("assign0.Van (%p) != val.Locals[0] (%p)", assign0.Var, l)
	}
	if assign1.Var != l {
		t.Errorf("assign1.Van (%p) != val.Locals[0] (%p)", assign1.Var, l)
	}
}

func TestCall(t *testing.T) {
	tests := []errorTest{
		{
			name: "function not found",
			src: `
				val x := [ foo: 5 bar: 6 ]
			`,
			err: "function foo:bar: not found",
		},
		{
			name: "function found",
			src: `
				val x := [ foo: 5 bar: 6 ]
				func [ foo: _ Int bar: _ Int | ]
			`,
			err: "",
		},
		{
			name: "method not found",
			src: `
				val x := [ 5 foo: 5 bar: 6 ]
			`,
			err: "method Int foo:bar: not found",
		},
		{
			name: "method found",
			src: `
				val x := [ 5 foo: 5 bar: 6 ]
				meth Int [foo: _ Int bar: _ Int]
			`,
			err: "",
		},
		{
			name: "method found from ref receiver",
			src: `
				val x := [
					x Int& := 12.
					x foo: 5 bar: 6
				]
				meth Int [foo: _ Int bar: _ Int]
			`,
			err: "",
		},
		{
			name: "method found from multi-ref receiver",
			src: `
				val x := [
					x Int& & & & := 12.
					x foo: 5 bar: 6
				]
				meth Int [foo: _ Int bar: _ Int]
			`,
			err: "",
		},
		{
			name: "method on bad type is not found",
			src: `
				val x := [ 5 foo: 5 bar: 6 ]
				meth Bad [foo: _ Int bar: _ Int]
			`,
			err: "method Int foo:bar: not found",
		},
		{
			name: "module not found",
			src: `
				val x := [ 5 #notfound foo: 5 bar: 6 ]
			`,
			err: "module #notfound not found",
		},
		{
			name: "other module function call",
			src: `
				import "found"
				val x := [ #found foo: 5 bar: 6 ]
			`,
			imports: [][2]string{
				{"found", `
					Func [foo: _ Int bar: _ Int]
				`},
			},
			err: "",
		},
		{
			name: "other module method call",
			src: `
				import "found"
				val x := [ 5 #found foo: 5 bar: 6 ]
			`,
			imports: [][2]string{
				{"found", `
					Meth Int [foo: _ Int bar: _ Int]
				`},
			},
			err: "",
		},
		{
			name: "private function not found",
			src: `
				import "found"
				val x := [ #found foo: 5 bar: 6 ]
			`,
			imports: [][2]string{
				{"found", `
					func [foo: _ Int bar: _ Int]
				`},
			},
			err: "function #found foo:bar: not found",
		},
		{
			name: "private method not found",
			src: `
				import "found"
				val x := [ 5 #found foo: 5 bar: 6 ]
			`,
			imports: [][2]string{
				{"found", `
					meth Int [foo: _ Int bar: _ Int]
				`},
			},
			err: "method Int #found foo:bar: not found",
		},
		{
			name: "type variable method not found",
			src: `
				meth T Array [ test: t T | t foo: 5 bar: 6 ]
			`,
			err: "method T foo:bar: not found",
		},
		{
			name: "type variable method found",
			src: `
				meth (T FooBarer) Array [ test: t T | t foo: 5 bar: 6 ]
				type FooBarer { [foo: Int bar: Int] }
			`,
			err: "",
		},
		{
			name: "function call does not find a method",
			src: `
				val x := [ foo: 5 bar: 6 ]
				meth Int [foo: _ Int bar: _ Int]
			`,
			err: "function foo:bar: not found",
		},
		{
			name: "method call does not find a function",
			src: `
				val x := [ 5 foo: 5 bar: 6 ]
				func [foo: _ Int bar: _ Int]
			`,
			err: "method Int foo:bar: not found",
		},
		{
			name: "imported func",
			src: `
				import "foo"
				val _ String := [#foo foo]
			`,
			imports: [][2]string{
				{"foo", "Func [foo ^Int]"},
			},
			err: "have Int, want String",
		},
		{
			name: "imported priv func not found",
			src: `
				import "foo"
				val _ String := [#foo foo]
			`,
			imports: [][2]string{
				{"foo", "func [foo ^Int]"},
			},
			err: "identifier #foo foo not found",
		},
		{
			name: "imported func not found without mod tag",
			src: `
				import "foo"
				val _ String := [foo]
			`,
			imports: [][2]string{
				{"foo", "Func [foo ^Int]"},
			},
			err: "identifier foo not found",
		},
		{
			name: "Imported func",
			src: `
				Import "foo"
				val _ String := [foo]
			`,
			imports: [][2]string{
				{"foo", "Func [foo ^Int]"},
			},
			err: "have Int, want String",
		},
		{
			name: "Imported priv func not found",
			src: `
				Import "foo"
				val _ String := [foo]
			`,
			imports: [][2]string{
				{"foo", "func [foo ^Int]"},
			},
			err: "identifier foo not found",
		},
		{
			name: "Imported func with mod tag",
			src: `
				Import "foo"
				val _ String := [#foo foo]
			`,
			imports: [][2]string{
				{"foo", "Func [foo ^Int]"},
			},
			err: "have Int, want String",
		},
		{
			name: "imported meth",
			src: `
				import "foo"
				val _ String := [5 #foo foo]
			`,
			imports: [][2]string{
				{"foo", "Meth Int [foo ^Int]"},
			},
			err: "have Int, want String",
		},
		{
			name: "imported priv meth not found",
			src: `
				import "foo"
				val _ String := [5 #foo foo]
			`,
			imports: [][2]string{
				{"foo", "meth Int [foo ^Int]"},
			},
			err: "method Int #foo foo not found",
		},
		{
			name: "imported meth not found without mod tag",
			src: `
				import "foo"
				val _ String := [5 foo]
			`,
			imports: [][2]string{
				{"foo", "Meth Int [foo ^Int]"},
			},
			err: "method Int foo not found",
		},
		{
			name: "Imported meth",
			src: `
				Import "foo"
				val _ String := [5 foo]
			`,
			imports: [][2]string{
				{"foo", "Meth Int [foo ^Int]"},
			},
			err: "have Int, want String",
		},
		{
			name: "Imported priv meth not found",
			src: `
				Import "foo"
				val _ String := [5 foo]
			`,
			imports: [][2]string{
				{"foo", "meth Int [foo ^Int]"},
			},
			err: "method Int foo not found",
		},
		{
			name: "Imported func with mod tag",
			src: `
				Import "foo"
				val _ String := [5 #foo foo]
			`,
			imports: [][2]string{
				{"foo", "Meth Int [foo ^Int]"},
			},
			err: "have Int, want String",
		},
		{
			name: "ambiguous method call",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [5 foo]
			`,
			imports: [][2]string{
				{"foo", "Meth Int [foo ^Int]"},
				{"bar", "Meth Int [foo ^Float]"},
			},
			err: "ambiguous method",
		},
		{
			name: "unambiguous method call 1",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [5 #foo foo]
			`,
			imports: [][2]string{
				{"foo", "Meth Int [foo ^Int]"},
				{"bar", "Meth Int [foo ^Float]"},
			},
			err: "have Int, want String",
		},
		{
			name: "unambiguous method call 2",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [5 #bar foo]
			`,
			imports: [][2]string{
				{"foo", "Meth Int [foo ^Int]"},
				{"bar", "Meth Int [foo ^Float]"},
			},
			err: "have Float, want String",
		},
		{
			name: "ambiguous function call",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [foo: 5]
			`,
			imports: [][2]string{
				{"foo", "Func [foo: _ Int ^Int]"},
				{"bar", "Func [foo: _ Float ^Float]"},
			},
			err: "ambiguous function",
		},
		{
			name: "unambiguous function call 1",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [#foo foo: 5]
			`,
			imports: [][2]string{
				{"foo", "Func [foo: _ Int ^Int]"},
				{"bar", "Func [foo: _ Float ^Float]"},
			},
			err: "have Int, want String",
		},
		{
			name: "unambiguous function call 2",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [#bar foo: 5]
			`,
			imports: [][2]string{
				{"foo", "Func [foo: _ Int ^Int]"},
				{"bar", "Func [foo: _ Float ^Float]"},
			},
			err: "have Float, want String",
		},
		{
			name: "virtual call",
			src: `
				val _ String := [
					x Fooer := 1.
					x foo
				]
				type Fooer {[foo ^Int]}
				meth Int [foo ^Int]
			`,
			err: "have Int, want String",
		},
		{
			name: "static call on virtual type",
			src: `
				val _ String := [
					x Fooer := 1.
					x bar
				]
				type Fooer {[foo]}
				meth Int [foo]
				meth Fooer [bar ^Int]
			`,
			err: "have Int, want String",
		},
		{
			name: "method not found on virtual type",
			src: `
				val _ String := [
					x Fooer := 1.
					x baz
				]
				type Fooer {[foo]}
				meth Int [foo]
				meth Fooer [bar ^Int]
			`,
			err: "method Fooer baz not found",
		},
		{
			name: "type var call",
			src: `
				Func (T Fooer) [bar: t T ^String |
					^t foo
				]
				type Fooer {[foo ^Int]}
			`,
			err: "have Int, want String",
		},
		{
			name: "type var does not call static method of interface",
			src: `
				Func (T Fooer) [bar: t T ^String |
					^t bar
				]
				type Fooer {[foo]}
				meth Fooer [bar ^Int]
			`,
			err: "method T bar not found",
		},
		{
			name: "cannot call virtual of unexported type",
			src: `
				import "foo"
				val _ := [(#foo new) #foo foo]
			`,
			imports: [][2]string{
				{
					"foo",
					`
						Type Foo := foo.
						Func [new ^Foo]
						type foo {[foo ^Int]}
					`,
				},
			},
			err: "method #foo foo #foo foo not found",
		},
		{
			name: "can call virtual of exported type",
			src: `
				import "foo"
				val _ String := [(#foo new) #foo foo]
			`,
			imports: [][2]string{
				{
					"foo",
					`
						Type Foo {[foo ^Int]}
						Func [new ^Foo]
					`,
				},
			},
			err: "have Int, want String",
		},
		{
			name: "cannot call case method of unexported type",
			src: `
				import "foo"
				val _ := [(#foo new) #foo ifA: [] ifB: []]
			`,
			imports: [][2]string{
				{
					"foo",
					`
						Type Foo := foo.
						Func [new ^Foo]
						type foo {a | b}
					`,
				},
			},
			err: "method #foo foo #foo ifA:ifB: not found",
		},
		{
			name: "can call case method of exported type",
			src: `
				import "foo"
				val _ := [(#foo new) #foo ifA: [] ifB: []]
			`,
			imports: [][2]string{
				{
					"foo",
					`
						Type Foo {a | b}
						Func [new ^Foo]
					`,
				},
			},
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCallInstRecvType(t *testing.T) {
	tests := []errorTest{
		{
			name: "inst receiver",
			src: `
				val x := [
					recv Rune FloatFoo := {}.

					(recv at: 0) + 5.

					// But Int32 has no xyz: so this will error and say Int32.
					(recv at: 0) xyz: 2
				]
				type (X, Y) Foo { }
				type T FloatFoo := (T, Float) Foo.
				meth T FloatFoo [at: _ Float ^T]
			`,
			err: "Int32 xyz: not found",
		},
		{
			name: "recv type arg mismatch",
			src: `
				val x := [
					recv Int Array := {}.
					recv foo
				]
				type FloatArray := Float Array.
				meth FloatArray [foo]
			`,
			err: "type mismatch: have Int, want Float",
		},
		{
			name: "recv type arg error",
			src: `
				val x := [
					recv Unknown Array := {}.
					recv foo
				]
				type FloatArray := Float Array.
				meth FloatArray [foo]
			`,
			err: "Unknown not found",
		},
		{
			name: "inst built-in type receiver",
			src: `
				val x := [
					recv Rune Array := {}.
					// If we instantiated Rune Array correctly,
					// then the at: method should return Int
					// and we will succssfully find the + method.
					// If we didn't instantiated Int Array, + should fail.
					(recv at: 0) + 5.

					// But Int32 has no xyz: so this will error and say Int32.
					(recv at: 0) xyz: 2
				]
				`,
			err: "Int32 xyz: not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCtor(t *testing.T) {
	tests := []errorTest{
		{
			name: "bad type",
			src: `
				val _ Unknown := [ {5} ]
			`,
			err: "cannot infer constructor type",
		},
		{
			name: "disallowed built-in type",
			src: `
				val _ Int := [ {5} ]
			`,
			err: "cannot construct built-in type Int",
		},
		{
			name: "alias is OK",
			src: `
				val _ Int Ary := [ {5} ]
				type T Ary := T Array.
			`,
			err: "",
		},
		{
			name: "array OK",
			src: `
				val _ Int Array := [ {1; 2; 3} ]
			`,
			err: "",
		},
		{
			name: "array element error",
			src: `
				val _ Int Array := [ {1; 2; 3.14} ]
			`,
			err: "Int cannot represent 3.14",
		},
		{
			name: "or-type constructor",
			src: `
				type T? { none | some: T }
				val x Int? := [ {none} ]
				val y Int? := [ {some: 5} ]
			`,
			err: "",
		},
		{
			name: "or-type malformed no arguments",
			src: `
				type T? { none | some: T }
				val _ Int? := [ {} ]
			`,
			err: "malformed Int\\? constructor",
		},
		{
			name: "or-type malformed single argument",
			src: `
				type T? { none | some: T }
				val _ Int? := [ {5} ]
			`,
			err: "malformed Int\\? constructor",
		},
		{
			name: "or-type malformed too many arguments",
			src: `
				type T? { none | some: T }
				val _ Int? := [ {a: 5; b: 6; c: 7} ]
			`,
			err: "malformed Int\\? constructor",
		},
		{
			name: "or-type case not found",
			src: `
				type T? { none | some: T }
				val _ Int? := [ {noCase} ]
			`,
			err: "case noCase not found",
		},
		{
			name: "or-type case: not found",
			src: `
				type T? { none | some: T }
				val _ Int? := [ {noCase: 4} ]
			`,
			err: "case noCase: not found",
		},
		{
			name: "and-type OK",
			src: `
				val _ (Int, Float) Pair := [ {x: 5 y: 2} ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "",
		},
		{
			name: "and-type reordered fields OK",
			src: `
				val _ (Int, Float) Pair := [ {y: 5 x: 2} ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "",
		},
		{
			name: "nil constructor OK",
			src: `
				val _ Nil := [ {} ]
			`,
			err: "",
		},
		{
			name: "and-type empty OK",
			src: `
				val _ Empty := [ {} ]
				type Empty {}
			`,
			err: "",
		},
		{
			name: "and-type multiple arguments",
			src: `
				val _ (Int, Float) Pair := [ {x: 5; 6; 7} ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "malformed \\(Int, Float\\) Pair constructor",
		},
		{
			name: "and-type non-call",
			src: `
				val _ (Int, Float) Pair := [ { 12 } ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "malformed \\(Int, Float\\) Pair constructor",
		},
		{
			name: "and-type cascade",
			src: `
				val _ (Int, Float) Pair := [ { 12 x; y; z } ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "malformed \\(Int, Float\\) Pair constructor",
		},
		{
			name: "and-type method call",
			src: `
				val _ (Int, Float) Pair := [ { 12 x: 12 y: 12 } ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "malformed \\(Int, Float\\) Pair constructor",
		},
		{
			name: "and-type duplicate field",
			src: `
				val _ (Int, Float) Pair := [ {x: 6 y: 7 x: 8} ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "duplicate field: x",
		},
		{
			name: "and-type missing field",
			src: `
				val _ (Int, Float) Pair := [ {x: 6} ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "missing field: y",
		},
		{
			name: "and-type unknown field",
			src: `
				val _ (Int, Float) Pair := [ {x: 5 y: 6 z: 7} ]
				type (X, Y) Pair { x: X y: Y }
			`,
			err: "unknown field: z",
		},
		{
			name: "virt-type disallowed",
			src: `
				val _ Fooer := [ {5} ]
				meth Int [foo: _ Int ^Int|]
				type Fooer { [foo: Int ^Int] }
			`,
			err: "cannot construct virtual type Fooer",
		},
		{
			name: "cannot construct exported type",
			src: `
				import "foo"
				val _ #foo Test := [{x: 5}]
			`,
			imports: [][2]string{
				{"foo", "Type Test {x: Int}"},
			},
			err: "",
		},
		{
			name: "cannot construct unexported type",
			src: `
				import "foo"
				val _ #foo Test := [{x: 5}]
			`,
			imports: [][2]string{
				{
					"foo",
					`
						Type Test := test.
						type test {x: Int}
					`,
				},
			},
			err: "cannot construct unexported type #foo test",
		},
		{
			name: "cannot construct unexported param type",
			src: `
				import "foo"
				val _ Int #foo Test := [{x: 5}]
			`,
			imports: [][2]string{
				{
					"foo",
					`
						Type T Test := T test.
						type T test {x: T}
					`,
				},
			},
			err: "cannot construct unexported type Int #foo test",
		},
		{
			name: "can construct in-module unexported type",
			src: `
				val _ Test := [{x: 5}]
				type Test {x: Int}
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestIdent(t *testing.T) {
	tests := []errorTest{
		{
			name: "not found",
			src: `
				val x := [
					unknown
				]
			`,
			err: "unknown not found",
		},
		{
			name: "param",
			src: `
				func [foo: x Int |
					_ String := x
				]
			`,
			err: "have Int, want String",
		},
		{
			name: "local",
			src: `
				func [foo |
					x Int := 5.
					_ String := x
				]
			`,
			err: "have Int, want String",
		},
		{
			name: "field",
			src: `
				meth Test [foo |
					_ String := x
				]
				type Test {x: Int}
			`,
			err: "have Int, want String",
		},
		{
			name: "self",
			src: `
				meth Int [foo |
					_ String := self
				]
			`,
			err: "have Int&, want String",
		},
		{
			name: "val",
			src: `
				val x Int := [5]
				val _ String := [x]
			`,
			err: "have Int, want String",
		},
		{
			name: "imported val",
			src: `
				import "foo"
				val _ String := [#foo x]
			`,
			imports: [][2]string{{
				"foo", "Val x Int := [5]",
			}},
			err: "have Int, want String",
		},
		{
			name: "import not found",
			src: `
				val _ String := [#foo x]
			`,
			err: "module #foo not found",
		},
		{
			name: "private imported val not found",
			src: `
				import "foo"
				val _ String := [#foo x]
			`,
			imports: [][2]string{{
				"foo", "val x Int := [5]",
			}},
			err: "identifier #foo x not found",
		},
		{
			name: "imported val not found with mod tag",
			src: `
				import "foo"
				val _ String := [x]
			`,
			imports: [][2]string{{
				"foo", "Val x Int := [5]",
			}},
			err: "identifier x not found",
		},
		{
			name: "Imported val",
			src: `
				Import "foo"
				val _ String := [x]
			`,
			imports: [][2]string{{
				"foo", "Val x Int := [5]",
			}},
			err: "have Int, want String",
		},
		{
			name: "private Imported val not found",
			src: `
				Import "foo"
				val _ String := [x]
			`,
			imports: [][2]string{{
				"foo", "val x Int := [5]",
			}},
			err: "identifier x not found",
		},
		{
			name: "private Imported val with mod tag",
			src: `
				Import "foo"
				val _ String := [#foo x]
			`,
			imports: [][2]string{{
				"foo", "Val x Int := [5]",
			}},
			err: "have Int, want String",
		},
		{
			name: "ambiguous ident",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [x]
			`,
			imports: [][2]string{
				{"foo", "Val x Int := [5]"},
				{"bar", "Val x Float := [5]"},
			},
			err: "ambiguous identifier",
		},
		{
			name: "unambiguous ident 1",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [#foo x]
			`,
			imports: [][2]string{
				{"foo", "Val x Int := [5]"},
				{"bar", "Val x Float := [5]"},
			},
			err: "have Int, want String",
		},
		{
			name: "unambiguous ident 2",
			src: `
				Import "foo"
				Import "bar"
				val _ String := [#bar x]
			`,
			imports: [][2]string{
				{"foo", "Val x Int := [5]"},
				{"bar", "Val x Float := [5]"},
			},
			err: "have Float, want String",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestIdentLookup(t *testing.T) {
	const src = `
		meth Test [ foo: ignore0 Int bar: parmVar Int |
			ignore1 := 5. 	// 0
			localVar := 5. 	// 1
			localVar.		// 2
			parmVar.		// 3
			modVar.		// 4
			fieldVar.		// 5
			unaryFun.	// 6
			self.			// 7
			use: ignore1.
		]
		val modVar := [5]
		type Test { ignore: Float fieldVar: Int }
		func [unaryFun]
		func [use: _ Int]
	`
	p := ast.NewParser("#test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check the source: %v", errs)
	}
	stmts := mod.Defs[0].(*Fun).Stmts

	// Statement 2 is a local variable from Statement 1.
	localVar := stmts[1].(*Assign).Var
	if stmts[2].(*Ident).Var != localVar {
		t.Errorf("localVar (%p) != %p", stmts[2].(*Ident).Var, localVar)
	}
	if stmts[2].(*Ident).Var.Index != 1 {
		t.Errorf("localVar.Index (%d) != 1", stmts[2].(*Ident).Var.Index)
	}

	// Statement 3 is a parameter access.
	fun := mod.Defs[0].(*Fun)
	// parms are: 0=self, 1=ignore0, 2=parmVar
	parmVar := &fun.Sig.Parms[2]
	if stmts[3].(*Ident).Var != parmVar {
		t.Errorf("parmVar (%p) != %p", stmts[3].(*Ident).Var, parmVar)
	}
	if stmts[3].(*Ident).Var.FunParm != fun {
		t.Errorf("fun (%p) != %p", stmts[3].(*Ident).Var.FunParm, fun)
	}
	if stmts[3].(*Ident).Var.Index != 2 {
		t.Errorf("parmVar .Index(%d) != 2", stmts[3].(*Ident).Var.Index)
	}

	// Statement 4 is a module-level value access.
	val := mod.Defs[1].(*Val)
	modVar := &val.Var
	if stmts[4].(*Ident).Var != modVar {
		t.Errorf("modVar (%p) != %p", stmts[4].(*Ident).Var, modVar)
	}
	if stmts[4].(*Ident).Var.Val != val {
		t.Errorf("val (%p) != %p", stmts[4].(*Ident).Var.Val, val)
	}

	// Statement 5 is a struct field access.
	typ := mod.Defs[2].(*Type)
	fieldVar := &typ.Fields[1]
	if stmts[5].(*Ident).Var != fieldVar {
		t.Errorf("fieldVar (%p) != %p", stmts[5].(*Ident).Var, fieldVar)
	}
	if stmts[5].(*Ident).Var.Field != typ {
		t.Errorf("type (%p) != %p", stmts[5].(*Ident).Var.Field, typ)
	}
	if stmts[5].(*Ident).Var.Index != 1 {
		t.Errorf("field (%d) != 1", stmts[5].(*Ident).Var.Index)
	}

	// Statement 6 is a uary function call.
	if _, ok := stmts[6].(*Call); !ok {
		t.Errorf("unaryFun is not a call")
	}
	// Statement 7 is an access to self.
	fun = mod.Defs[0].(*Fun)
	parmVar = &fun.Sig.Parms[0]
	if stmts[7].(*Ident).Var != parmVar {
		t.Errorf("parmVar (%p) != %p", stmts[7].(*Ident).Var, parmVar)
	}
	if stmts[7].(*Ident).Var.FunParm != fun {
		t.Errorf("fun (%p) != %p", stmts[7].(*Ident).Var.FunParm, fun)
	}
	if stmts[7].(*Ident).Var.Index != 0 {
		t.Errorf("parmVar .Index(%d) != 0", stmts[7].(*Ident).Var.Index)
	}
}

func TestAssignToField(t *testing.T) {
	const src = `
		meth Point [ foo |
			x := 5.
			y := 6.
		]
		type Point { x: Int y: Int }
	`
	p := ast.NewParser("#test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	mod, errs := Check(p.Mod(), Config{})
	if len(errs) > 0 {
		t.Fatalf("failed to check the source: %v", errs)
	}

	fun := mod.Defs[0].(*Fun)
	typ := mod.Defs[1].(*Type)
	if len(fun.Locals) != 0 {
		t.Fatalf("got %d locals, expected 0: %v", len(fun.Locals), fun.Locals)
	}
	assign0 := fun.Stmts[0].(*Assign)
	assign1 := fun.Stmts[1].(*Assign)

	if assign0.Var != &typ.Fields[0] {
		t.Errorf("assign0.Var (%p) != &typ.Fields[0] (%p)",
			assign0.Var, &typ.Fields[0])
	}
	if assign0.Var.Field != typ {
		t.Errorf("assign0.Var.Field (%p) != Point (%p)", assign0.Var.Field, typ)
	}
	if assign0.Var.Index != 0 {
		t.Errorf("assign0.Var.Index (%d) != 0", assign0.Var.Index)
	}
	if assign1.Var != &typ.Fields[1] {
		t.Errorf("assign1.Var (%p) != &typ.Fields[1] (%p)",
			assign0.Var, &typ.Fields[1])
	}
	if assign1.Var.Field != typ {
		t.Errorf("assign1.Var.Field (%p) != Point (%p)", assign1.Var.Field, typ)
	}
	if assign1.Var.Index != 1 {
		t.Errorf("assign1.Var.Index (%d) != 1", assign1.Var.Index)
	}
}

func TestBlockLiteralError(t *testing.T) {
	tests := []errorTest{
		{
			name: "no infer type",
			src: `
				val x := [ [ :a :b :c | a + b + c ] ]
			`,
			err: "cannot infer block parameter type",
		},
		{
			name: "non-Fun infer type",
			src: `
				val x Int := [ [ :a :b :c | a + b + c ] ]
			`,
			err: "cannot infer block parameter type",
		},
		{
			name: "too many parameters",
			src: `
				val x (Int, Int, Int, Int, Int, String) Fun := [
					[ :a :b :c :d :e | a + b + c + d + e ]
				]
			`,
			err: "too many block parameters",
		},
		{
			name: "found overrides infer",
			src: `
				val x (Int64, Int32, Float, String) Fun := [
					[ :a Int8 :b String :c Float32 | 5 ]
				]
			`,
			err: "have Int, want String",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestBlockTypeInference(t *testing.T) {
	// The first def must be a *Val.
	// The first def's last statement must be a block literal.
	tests := []struct {
		name  string
		src   string
		want  string
		trace bool
	}{
		{
			name: "infer result from above",
			src: `
				val x Int64 Fun := [ [5] ]
			`,
			want: "Int64 Fun",
		},
		{
			name: "infer result from below",
			src: `
				val x := [ ["string"] ]
			`,
			want: "String Fun",
		},
		{
			name: "infer args from above",
			src: `
				val x (Int64, Int32, Float, String) Fun := [
					[ :a :b :c | "string" ]
				]
			`,
			want: "(Int64, Int32, Float, String) Fun",
		},
		{
			name: "infer args from below",
			src: `
				val x := [
					[ :a Int :b Int32 :c Float | "string" ]
				]
			`,
			want: "(Int, Int32, Float, String) Fun",
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
				t.Fatalf("failed to check the source: %v", errs)
			}
			val := mod.Defs[0].(*Val)
			blk, ok := val.Init[len(val.Init)-1].(*Block)
			if !ok {
				t.Fatalf("not a block, but a %T", val.Init[len(val.Init)-1])
			}
			if blk.typ.String() != test.want {
				t.Errorf("got %s, wanted %s", blk.typ.String(), test.want)
			}
		})
	}
}

func TestIntLit(t *testing.T) {
	tests := []errorTest{
		{
			name: "bad type",
			src:  "val a Unknown := [5]",
			err:  "type Unknown not found",
		},
		{
			name: "ok",
			src: `
				val a := [3]
				val b Int := [3]
				val c Int8 := [3]
				val d Int16 := [3]
				val e Int32 := [3]
				val f Int64 := [3]
				val g := [-3]
				val h Int := [-3]
				val i Int8 := [-3]
				val j Int16 := [-3]
				val k Int32 := [-3]
				val l Int64 := [-3]
				val m UInt := [3]
				val n UInt8 := [3]
				val o UInt16 := [3]
				val p UInt32 := [3]
				val q UInt64 := [3]
				val r Float := [3]
				val s Float32 := [3]
				val t Float64 := [3]
			`,
			err: "",
		},
		{
			name: "Int8 ok",
			src: `
				val a Int8 := [-128]
				val b Int8 := [-127]
				val c Int8 := [-1]
				val d Int8 := [0]
				val e Int8 := [-1]
				val f Int8 := [126]
				val g Int8 := [127]
			`,
			err: "",
		},
		overflowTest("Int8", "-1000"),
		overflowTest("Int8", "-129"),
		overflowTest("Int8", "128"),
		overflowTest("Int8", "1000"),
		{
			name: "Int16 ok",
			src: `
				val a Int16 := [-32768]
				val b Int16 := [-32767]
				val c Int16 := [-1]
				val d Int16 := [0]
				val e Int16 := [-1]
				val f Int16 := [32766]
				val g Int16 := [32767]
			`,
			err: "",
		},
		overflowTest("Int16", "-10000000"),
		overflowTest("Int16", "-32769"),
		overflowTest("Int16", "32768"),
		overflowTest("Int16", "10000000"),
		{
			name: "Int32 ok",
			src: `
				val a Int32 := [-2147483648]
				val b Int32 := [-2147483647]
				val c Int32 := [-1]
				val d Int32 := [0]
				val e Int32 := [-1]
				val f Int32 := [2147483646]
				val g Int32 := [2147483647]
			`,
			err: "",
		},
		overflowTest("Int32", "-100000000000000"),
		overflowTest("Int32", "-2147483649"),
		overflowTest("Int32", "2147483648"),
		overflowTest("Int32", "100000000000000"),
		{
			name: "Int64 ok",
			src: `
				val a Int64 := [-9223372036854775808]
				val b Int64 := [-9223372036854775807]
				val c Int64 := [-1]
				val d Int64 := [0]
				val e Int64 := [-1]
				val f Int64 := [9223372036854775806]
				val g Int64 := [9223372036854775807]
			`,
			err: "",
		},
		overflowTest("Int64", "-100000000000000000000000"),
		overflowTest("Int64", "-9223372036854775809"),
		overflowTest("Int64", "9223372036854775808"),
		overflowTest("Int64", "100000000000000000000000"),
		{
			name: "UInt8 ok",
			src: `
				val a UInt8 := [0]
				val b UInt8 := [1]
				val c UInt8 := [100]
				val d UInt8 := [254]
				val e UInt8 := [255]
			`,
			err: "",
		},
		{
			name: "UInt8 negative",
			src:  "val x UInt8 := [-1]",
			err:  "UInt8 cannot represent -1: negative unsigned",
		},
		overflowTest("UInt8", "256"),
		overflowTest("UInt8", "10000"),
		{
			name: "UInt16 ok",
			src: `
				val a UInt16 := [0]
				val b UInt16 := [1]
				val c UInt16 := [100]
				val d UInt16 := [65534]
				val e UInt16 := [65535]
			`,
			err: "",
		},
		{
			name: "UInt16 negative",
			src:  "val x UInt16 := [-1]",
			err:  "UInt16 cannot represent -1: negative unsigned",
		},
		overflowTest("UInt16", "65536"),
		overflowTest("UInt16", "1000000"),
		{
			name: "UInt32 ok",
			src: `
				val a UInt32 := [0]
				val b UInt32 := [1]
				val c UInt32 := [100]
				val d UInt32 := [4294967294]
				val e UInt32 := [4294967295]
			`,
			err: "",
		},
		{
			name: "UInt32 negative",
			src:  "val x UInt32 := [-1]",
			err:  "UInt32 cannot represent -1: negative unsigned",
		},
		overflowTest("UInt32", "4294967296"),
		overflowTest("UInt32", "10000000000000"),
		{
			name: "UInt64 ok",
			src: `
				val a UInt64 := [0]
				val b UInt64 := [1]
				val c UInt64 := [100]
				val d UInt64 := [18446744073709551615]
				val e UInt64 := [18446744073709551615]
			`,
			err: "",
		},
		{
			name: "UInt64 negative",
			src:  "val x UInt64 := [-1]",
			err:  "UInt64 cannot represent -1: negative unsigned",
		},
		overflowTest("UInt64", "18446744073709551616"),
		overflowTest("UInt64", "100000000000000000000000"),
		{
			name: "rune lit",
			src:  "val x := ['a']",
			err:  "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func overflowTest(typ, val string) errorTest {
	return errorTest{
		name: fmt.Sprintf("%s %s overflow", typ, val),
		src:  fmt.Sprintf("val x %s := [%s]", typ, val),
		err:  fmt.Sprintf("%s cannot represent %s: overflow", typ, val),
	}
}

func TestFloatLit(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				val w := [3.1415]
				val x Float := [3.1415]
				val y Float32 := [3.1415]
				val z Float64 := [3.1415]
				val a Int := [3.00000]
				val b Int := [-3.00000]
				val c UInt := [3.00000]
			`,
			err: "",
		},
		{
			name: "bad truncation",
			src:  "val x Int := [3.14]",
			err:  "Int cannot represent 3.14: truncation",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

// TestTypeInstSub tests type instantiation substitution.
func TestTypeInstSub(t *testing.T) {
	// The test setup expects src to have an alias type named Test.
	// The expectation is that Test.Alias.Type!=nil
	// and Test.Alias.Type.String()==want.
	tests := []struct {
		name  string
		src   string
		want  string
		trace bool
	}{
		{
			name: "no type vars",
			src: `
				type Test := Int64.
			`,
			want: "type Int64 {}",
		},
		{
			name: "follow alias",
			src: `
				type Test := Abc.
				type Abc := Def.
				type Def := Ghi.
				type Ghi := Int32.
			`,
			want: "type Int32 {}",
		},
		{
			name: "sub type parm",
			src: `
				type Test := Rune List.
				type T List { data: T | next: T List? }
				type T ? { none | some: T }
			`,
			want: "type Rune List { data: Rune | next: Rune List? }",
		},
		{
			name: "sub type parms",
			src: `
				type Test := (Rune, String) Pair.
				type (X, Y) Pair { x: X y: Y }
			`,
			want: "type (Rune, String) Pair { x: Rune y: String }",
		},
		{
			name: "sub only some type parms",
			src: `
				type T Test := (Rune, T) Pair.
				type (X, Y) Pair { x: X y: Y }
			`,
			want: "type (Rune, T) Pair { x: Rune y: T }",
		},
		{
			name: "sub alias",
			src: `
				type Test := Rune DifferentArray.
				type T DifferentArray := T Array.
			`,
			want: "type Rune Array {}",
		},
		{
			name: "sub fields",
			src: `
				type Test := (Rune, String) Pair.
				type (X, Y) Pair { x: X y: Y }
			`,
			want: "type (Rune, String) Pair { x: Rune y: String }",
		},
		{
			name: "sub cases",
			src: `
				type Test := Rune?.
				type T? { none | some: T }
			`,
			want: "type Rune? { none | some: Rune }",
		},
		{
			name: "sub virts",
			src: `
				type Test := Rune Eq.
				type T Eq { [= T& ^Bool] }
			`,
			want: "type Rune Eq { [= Rune& ^Bool] }",
		},
		{
			name: "recursive type",
			src: `
				type Test := Rune List.
				type T List { data: T& next: T List? }
				type T? { none | some: T }
			`,
			want: "type Rune List { data: Rune& next: Rune List? }",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			p := ast.NewParser("#test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Fatalf("failed to parse source: %s", err)
			}
			cfg := Config{
				Importer: testImporter(nil),
				Trace:    test.trace,
			}
			mod, errs := Check(p.Mod(), cfg)
			if len(errs) > 0 {
				t.Fatalf("failed to check source: %s", errs)
			}
			testType := findTestType(mod, "Test")
			if testType == nil {
				t.Fatalf("no test type")
			}
			if testType.Alias == nil {
				t.Fatalf("test type is not an alias")
			}
			if got := testType.Alias.Type.fullString(); got != test.want {
				t.Errorf("got:\n%s\nwanted:\n%s", got, test.want)
			}
		})
	}
}

// TestTypeInstMemo tests that the same type instances point to the same objects.
func TestTypeInstMemo(t *testing.T) {
	// The test setup expects src to have two alias types, Test0 and Test1.
	// The expectation is that Test0.Alias.Type==Test1.Alias.Type.
	tests := []struct {
		name    string
		src     string
		imports [][2]string
		trace   bool
	}{
		{
			name: "basic types",
			src: `
				type Test0 := Int64.
				type Test1 := Int64.
			`,
		},
		{
			name: "basic follow alias",
			src: `
				type Test0 := Int.
				type Test1 := Int.
			`,
		},
		{
			name: "basic follow different alias",
			src: `
				type Test0 := Rune.
				type Test1 := Abc.
				type Abc := Int32.
			`,
		},
		{
			name: "inst built-in type",
			src: `
				type Test0 := Int64 Array.
				type Test1 := Int64 Array.
			`,
		},
		{
			name: "inst built-in type with aliases",
			src: `
				type Test0 := Rune Array.
				type Test1 := Abc Array.
				type Abc := Int32.
			`,
		},
		{
			name: "multiple type parms",
			src: `
				type Test0 := (Int64, String) Map.
				type Test1 := (Int64, String) Map.
				type (_, _) Map {}
			`,
		},
		{
			name: "multiple type parms and aliases",
			src: `
				type Test0 := (Int32, String) Map.
				type Test1 := Abc.
				type Abc := (Rune, OtherString) Map.
				type OtherString := String.
				type (_, _) Map {}
			`,
		},
		{
			name: "imported type",
			src: `
				import "bar"
				import "foo"
				type Test0 := #bar IntStringMap.
				type Test1 := #foo IntStringMap.
			`,
			imports: [][2]string{
				{"foo", `
					import "map"
					Type IntStringMap := (Rune, String) #map Map.
				`},
				{"bar", `
					import "map"
					Type IntStringMap := (Rune, String) #map Map.
				`},
				{"map", `
					Type (_, _) Map {}
				`},
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
			cfg := Config{
				Importer: testImporter(test.imports),
				Trace:    test.trace,
			}
			mod, errs := Check(p.Mod(), cfg)
			if len(errs) > 0 {
				t.Fatalf("failed to check source: %s", errs)
			}
			testType0 := findTestType(mod, "Test0")
			if testType0 == nil {
				t.Fatalf("no test type")
			}
			if testType0.Alias == nil {
				t.Fatalf("test type is not an alias")
			}
			testType1 := findTestType(mod, "Test1")
			if testType1 == nil {
				t.Fatalf("no test type")
			}
			if testType1.Alias == nil {
				t.Fatalf("test type is not an alias")
			}
			if testType0.Alias.Type != testType1.Alias.Type {
				t.Logf("Test0=%s\nTest1=%s",
					pretty.String(testType0.Alias.Type),
					pretty.String(testType1.Alias.Type))
				t.Errorf("Test0=%p, Test1=%p",
					testType0.Alias.Type, testType1.Alias.Type)
			}
		})
	}
}

func findTestType(mod *Mod, name string) *Type {
	for _, def := range mod.Defs {
		if typ, ok := def.(*Type); ok && typ.Name == name {
			return typ
		}
	}
	return nil
}

type errorTest struct {
	name    string
	src     string
	imports [][2]string
	err     string // regexp, "" means no error
	trace   bool
}

func (test errorTest) run(t *testing.T) {
	if strings.HasPrefix(test.name, "SKIP:") {
		t.Skip()
	}
	p := ast.NewParser("#test")
	if err := p.Parse("", strings.NewReader(test.src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	cfg := Config{
		Importer: testImporter(test.imports),
		Trace:    test.trace,
	}
	switch _, errs := Check(p.Mod(), cfg); {
	case test.err == "" && len(errs) == 0:
		return
	case test.err == "" && len(errs) > 0:
		t.Errorf("got %v, expected nil", errs)
	case test.err != "" && len(errs) == 0:
		t.Errorf("got nil, expected matching %s", test.err)
	default:
		err := fmt.Sprintf("%v", errs)
		if !regexp.MustCompile(test.err).MatchString(err) {
			t.Errorf("got %v, expected matching %s", errs, test.err)
		}
	}
}

type testImporter [][2]string

func (imports testImporter) Import(cfg Config, path string) ([]Def, error) {
	for i := range imports {
		if imports[i][0] != path {
			continue
		}
		src := imports[i][1]
		p := ast.NewParser(path)
		if err := p.Parse(path, strings.NewReader(src)); err != nil {
			return nil, fmt.Errorf("failed to parse import: %s", err)
		}
		cfg.Trace = false
		mod, errs := Check(p.Mod(), cfg)
		if len(errs) > 0 {
			return nil, fmt.Errorf("failed to check import: %s", errs)
		}
		setMod(path, mod.Defs)
		return mod.Defs, nil
	}
	return nil, errors.New("not found")
}
