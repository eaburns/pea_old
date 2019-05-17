package pea

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/eaburns/pretty"
)

func TestImportError(t *testing.T) {
	tests := []checkTest{
		{
			name: "import not found",
			src:  `import "nothing"`,
			err:  "error importing nothing: not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestRedefError(t *testing.T) {
	tests := []checkTest{
		{
			name: "empty",
			src:  "",
			err:  "",
		},
		{
			name: "no redefinition",
			src:  "Abc{} Xyz{}",
			err:  "",
		},
		{
			name: "type redefined",
			src:  "Xyz{} Xyz{}",
			err:  "Xyz is redefined(.|\n)*previous definition",
		},
		{
			name: "fun redefined",
			src:  "[foo: _ Int32 |] [foo: _ Int32 |]",
			err:  "foo: is redefined(.|\n)*previous definition",
		},
		{
			name: "fun redefined with different param types",
			src:  "[foo: _ Int32 |] [foo: _ String |]",
			err:  "foo: is redefined(.|\n)*previous definition",
		},
		{
			name: "fun with different arity is ok",
			src:  "[foo: _ Int32 |] [foo |] [foo: _ Int32 bar: _ String |]",
			err:  "",
		},
		{
			name: "same-name fun and method are not redefined",
			src:  "[foo: _ Int32 |] String [foo: _ Int32 |]",
			err:  "",
		},
		{
			name: "same-name methods are not redefined",
			src:  "Int [foo: _ Int32 |] String [foo: _ String |]",
			err:  "",
		},
		{
			name: "var redefined",
			src:  "Xyz := [5] Xyz := [6]",
			err:  "Xyz is redefined",
		},
		{
			name: "fun redefines a type",
			src:  "Xyz{} [Xyz |]",
			err:  "Xyz is redefined",
		},
		{
			name: "var redefines a type",
			src:  "Xyz{} Xyz := [5]",
			err:  "Xyz is redefined",
		},
		{
			name: "type redefineds a fun",
			src:  "[Xyz |] Xyz{}",
			err:  "Xyz is redefined",
		},
		{
			name: "type redefineds a var",
			src:  "Xyz := [5] Xyz {}",
			err:  "Xyz is redefined",
		},
		{
			name: "fun redefines a var",
			src:  "Xyz := [5] [Xyz |]",
			err:  "Xyz is redefined",
		},
		{
			name: "var redefines a fun",
			src:  "[Xyz |] Xyz := [5]",
			err:  "Xyz is redefined",
		},
		{
			name: "built-in overridden",
			src:  "Int32 {}",
			err:  "",
		},
		{
			name: "import overridden",
			src:  `import "xyz" Xyz{}`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "",
		},
		{
			name: "redefined by import",
			src:  `Xyz{} import "xyz"`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "imported definition xyz Xyz is redefined",
		},
		{
			name: "import redefined by import",
			src:  `import "xyz" import "xyz"`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "imported definition xyz Xyz is redefined(.|\n)*previous definition imported",
		},
		{
			name: "same def in different submods is ok",
			src: `
				#one Xyz{}
				#two Xyz{}
			`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "",
		},
		{
			name: "same import in different submods is ok",
			src: `
				#one import "xyz"
				#two import "xyz"
			`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "",
		},
		{
			name: "redef in a submod",
			src: `
				#one Xyz{}
				#one Xyz{}
			`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "Xyz is redefined",
		},
		{
			name: "virtual method redef",
			src: `
				Foo { [bar ^Int] }
				Foo [bar ^Int | ^ 1]
			`,
			err: "method bar is redefined",
		},
		{
			name: "multiple redefs",
			src: `
				Xyz{} Abc{} Xyz{} Abc{}
				Cde{}
				[Cde |]
			`,
			err: "Xyz is redefined(.|\n)*Abc is redefined(.|\n)*Cde is redefined",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckVar(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src:  "v Int32 := [5]",
			err:  "",
		},
		{
			name: "ok nil",
			src:  "v := []",
			err:  "",
		},
		{
			name: "undef type",
			src:  "v Undef := [5]",
			err:  "Undef is undefined",
		},
		{
			name: "bad statement",
			src:  "v Undef := [{Int8 Array | 257}]",
			err:  "Int8 cannot represent 257: overflow",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckFun(t *testing.T) {
	tests := []checkTest{
		{
			name: "redefined param",
			src:  "[foo: x Int bar: x Int |]",
			err:  "x is redefined",
		},
		{
			name: "_ is not redefined",
			src:  "[foo: _ Int bar: _ Int |]",
			err:  "",
		},
		{
			name: "bad parameter type",
			src:  "[foo: _ Undef |]",
			err:  "Undef is undefined",
		},
		{
			name: "bad return type",
			src:  "[foo ^Undef |]",
			err:  "Undef is undefined",
		},
		{
			name: "undef recv type",
			src:  "Undef [foo |]",
			err:  "Undef is undefined",
		},
		{
			name: "non-type receiver",
			src: `
				Var [foo |]
				Var := [5]
			`,
			err: "got variable, expected a type",
		},
		{
			name: "undef recv type constraint",
			src:  "(T Undef) Array [foo |]",
			err:  "Undef is undefined",
		},
		{
			name: "non-type recv type constraint",
			src: `
				(X Var) Array [foo |]
				Var := [5]
			`,
			err: "got variable, expected a type",
		},
		{
			name: "bad recv param count",
			src:  "(T, U) Array [foo |]",
			err:  "parameter count mismatch",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckTypeSig(t *testing.T) {
	tests := []checkTest{
		{
			name: "undef constraint",
			src:  "(T Undef) Foo { x: T }",
			err:  "Undef is undefined",
		},
		{
			name: "non-type constraint",
			src: `
				(T Var) Foo { x: T }
				Var := [5]
			`,
			err: "got variable, expected a type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckTypeName(t *testing.T) {
	tests := []checkTest{
		{
			name: "undefined type name arg",
			src:  "[foo ^Undef Array | ]",
			err:  "Undef is undefined",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckTypeAlias(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src: `
				Abc := Int.
				Def := Abc.
				Ghi := Def.
			`,
			err: "",
		},
		{
			name: "undefined",
			src: `
				Abc := Undef.
			`,
			err: "Undef is undefined",
		},
		{
			name: "alias of bad type",
			src: `
				Abc := Def.
				Def := Undef.
			`,
			err: "Undef is undefined",
		},
		{
			name: "2-cycle",
			src: `
				Abc := Def.
				Def := Abc.
			`,
			err: "type alias cycle",
		},
		{
			name: "larger cycle",
			src: `
				Abc := Def.
				Def := Ghi.
				Ghi := Jkl.
				Jkl := Mno.
				Mno := Abc.
			`,
			err: "type alias cycle",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckFields(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src:  "Point { x: Float y: Float }",
			err:  "",
		},
		{
			name: "undefined",
			src:  "Point { x: Undef y: Float }",
			err:  "Undef is undefined",
		},
		{
			name: "field redefined",
			src:  "Point { x: Float x: Float }",
			err:  "field x is redefined",
		},
		{
			name: "non-type",
			src: `
				[ someFunc | ]
				Point { x: someFunc }
			`,
			err: "got function, expected a type",
		},
		/* // TODO: test a built-in non-type error if there is ever a built-in that is not a type.
		{
			name: "built-in non-type",
			src: `
				Point { x: someBuiltin }
			`,
			err: "got function, expected a type",
		},
		*/
		{
			name: "imported non-type",
			src: `
				import "other"
				Point { x: someFunc }
			`,
			mods: [][2]string{
				{"other", "[ someFunc | ]"},
			},
			err: "got function, expected a type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckCases(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src:  "IntOpt { none, some: Int }",
			err:  "",
		},
		{
			name: "undefined",
			src:  "UndefOpt { none, some: Undef }",
			err:  "Undef is undefined",
		},
		{
			name: "typeless case redefined",
			src:  "Opt { none, none }",
			err:  "case none is redefined",
		},
		{
			name: "typed case redefined",
			src:  "Opt { some: Int, some: Int }",
			err:  "case some is redefined",
		},
		{
			name: "typed and typeless case redefined",
			src:  "Opt { none, none: Int }",
			err:  "case none is redefined",
		},
		{
			name: "case capitalization redefined",
			src:  "Opt { none, None }",
			err:  "case none is redefined",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckVirts(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src:  "Reader { [readInto: Byte Array ^Int64] }",
			err:  "",
		},
		{
			name: "undefined parameter type",
			src:  "Reader { [readInto: Undef ^Int64] }",
			err:  "Undef is undefined",
		},
		{
			name: "undefined return type",
			src:  "Reader { [readInto: Byte Array ^Undef] }",
			err:  "Undef is undefined",
		},
		{
			name: "method signature redefined",
			src:  "IntSeq { [at: Int ^Int] [at: Int ^Int] }",
			err:  "virtual method at: is redefined",
		},
		{
			name: "not redefined",
			src: `
				IntSeq {
					[at ^Int]
					[at: Int ^Int]
					[at: Int at: Int ^Int]
					[at: Int put: Int]
				}
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestRet(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src:  "[ foo ^Int | ^5 ]",
			err:  "",
		},
		/*
			// TODO: test bad return expression when expression checking is implemented.
			{
				name: "bad expression",
				src:  "[ foo ^Int | ^{ Undef | } ]",
				err:  "Undef is undefined",
			},
			// TODO: test return outside of a method error when Var checking is implemented.
			{
				name: "outside method",
				src:  "x := [ ^12 ]",
				err:  "return outside of a method",
			},
		*/
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestAssign(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src: `
				[ foo: x Int |
					w := 3.1415.
					x := 5.
					y String := "".
					z Byte Array := {Byte Array | 1}.
					a, b Int, c := 5 neg, neg, neg.
				]`,
			err: "",
		},
		{
			name: "bad type",
			src:  "[ foo | x Undef := `` ]",
			err:  "Undef is undefined",
		},
		{
			name: "param redef",
			src:  "[ foo: x Int | x String := `` ]",
			err:  "x is redefined",
		},
		{
			name: "local redef",
			src:  "[ foo | x String := ``. x Int := 5 ]",
			err:  "x is redefined",
		},
		{
			name: "too few vals",
			src:  "[ foo | x, y := 5 ]",
			err:  "assignment count mismatch: got 1, expected 2",
		},
		{
			name: "too many vals",
			src:  "[ foo | x, y := 5 neg, neg, neg ]",
			err:  "assignment count mismatch: got 3, expected 2",
		},
		{
			name: "assign mismatch still checks type names",
			src:  "[ foo | x, y Undef := 5 ]",
			err:  "Undef is undefined",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCtor(t *testing.T) {
	tests := []checkTest{
		{
			name: "bad type",
			src:  "[foo | { Undef | 5; 6 } ]",
			err:  "Undef is undefined",
		},
		{
			name: "empty array",
			src:  "[foo | { Int Array | } ]",
			err:  "",
		},
		{
			name: "bad array expression",
			src:  "[foo | { Int8 Array | 257 } ]",
			err:  "Int8 cannot represent 257: overflow",
		},
		{
			name: "bad built-in type selector",
			src: `
				[foo| { Int | 123 }]
			`,
			err: "built-in type Int64 cannot be constructed",
		},
		{
			name: "good and-type selector",
			src: `
				[foo| { Point | x: 5; y: 6 }]
				Point { x: Int y: Int }
			`,
			err: "",
		},
		{
			name: "and-type nil",
			src: `
				[foo| { Empty | }]
				Empty { }
			`,
			err: "",
		},
		{
			name: "and-type array args",
			src: `
				[foo| { Empty | 123 }]
				Empty { }
			`,
			err: "bad and-type constructor: Nil with non-nil expression",
		},
		{
			name: "bad and-type selector",
			src: `
				[foo| { Point | a: 5; b: 6 }]
				Point { x: Int y: Int }
			`,
			err: "bad and-type constructor: got a:b:, expected x:y:",
		},
		{
			name: "bad and-type expression",
			src: `
				[foo| { Point | x: 257; y: 6 }]
				Point { x: Int8 y: Int8 }
			`,
			err: "Int8 cannot represent 257: overflow",
		},
		{
			name: "good or-type typeless selector",
			src: `
				[foo| { IntOpt | none }]
				// TODO: change enum , to ; to be consistent with arrays?
				IntOpt { none, some: Int }
			`,
			err: "",
		},
		{
			name: "good or-type typeed selector",
			src: `
				[foo| { IntOpt | some: 5 }]
				IntOpt { none, some: Int }
			`,
			err: "",
		},
		{
			name: "bad or-type selector",
			src: `
				[foo| { IntOpt | oopsy: 5 }]
				IntOpt { none, some: Int }
			`,
			err: "bad or-type constructor: no case oopsy:",
		},
		{
			name: "bad or-type expression",
			src: `
				[foo| { IntOpt | some: 257 }]
				IntOpt { none, some: Int8 }
			`,
			err: "Int8 cannot represent 257: overflow",
		},
		{
			name: "conversion with a selector",
			src: `
				[foo| { Reader | some: 257 }]
				Reader { [read ^Byte Array] }
			`,
			err: "a virtual conversion cannot have a selector",
		},
		{
			name: "conversion with a multiple args",
			src: `
				[foo| { Reader | 257; 258 }]
				Reader { [read ^Byte Array] }
			`,
			err: "a virtual conversion must have exactly one argument",
		},
		{
			name: "bad conversion expression",
			src: `
					[foo| { Reader | { Int8 Array | 257 } }]
					Reader { [read ^Byte Array] }
				`,
			err: "Int8 cannot represent 257: overflow",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestBlock(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok no parms",
			src:  `[ foo | x := [ 5 ] ]`,
			err:  "",
		},
		{
			name: "ok type-specified parm",
			src:  `[ foo | x := [ :n Int | n + 3 ] ]`,
			err:  "",
		},
		{
			name: "ok type-inferred parm",
			src:  `[ foo | x (Int, Nil) Fun1 := [ :n | n + 3 ] ]`,
			err:  "",
		},
		{
			name: "param count mismatch",
			src:  `[ foo | x Int Fun0 := [ :n | n + 3 ] ]`,
			err:  "cannot infer block parameter type",
		},
		{
			name: "too many parameters",
			src:  `[ foo | x := [ :n Int :o Int :p Int :q Int :r Int :s Int | n + 3 ] ]`,
			err:  "too many block parameters",
		},
		{
			name: "bad statement",
			src:  `[ foo | x := [ y Int := 3.14 ] ]`,
			err:  "Int cannot represent 3.14: truncation",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestIntLit(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src: `[ foo |
					a := 3.
					b Int := 3.
					c Int8 := 3.
					d Int16 := 3.
					e Int32 := 3.
					f Int64 := 3.
					g := -3.
					h Int := -3.
					i Int8 := -3.
					j Int16 := -3.
					k Int32 := -3.
					l Int64 := -3.
					m Uint := 3.
					n Uint8 := 3.
					o Uint16 := 3.
					p Uint32 := 3.
					q Uint64 := 3.
					r Float := 3.
					s Float32 := 3.
					t Float64 := 3.
				]`,
			err: "",
		},
		{
			name: "Int8 ok",
			src: `[foo |
					a Int8 := -128.
					b Int8 := -127.
					c Int8 := -1.
					d Int8 := 0.
					e Int8 := -1.
					f Int8 := 126.
					g Int8 := 127.
				]`,
			err: "",
		},
		overflowTest("Int8", "-1000"),
		overflowTest("Int8", "-129"),
		overflowTest("Int8", "128"),
		overflowTest("Int8", "1000"),
		{
			name: "Int16 ok",
			src: `[foo |
					a Int16 := -32768.
					b Int16 := -32767.
					c Int16 := -1.
					d Int16 := 0.
					e Int16 := -1.
					f Int16 := 32766.
					g Int16 := 32767.
				]`,
			err: "",
		},
		overflowTest("Int16", "-10000000"),
		overflowTest("Int16", "-32769"),
		overflowTest("Int16", "32768"),
		overflowTest("Int16", "10000000"),
		{
			name: "Int32 ok",
			src: `[foo |
					a Int32 := -2147483648.
					b Int32 := -2147483647.
					c Int32 := -1.
					d Int32 := 0.
					e Int32 := -1.
					f Int32 := 2147483646.
					g Int32 := 2147483647.
				]`,
			err: "",
		},
		overflowTest("Int32", "-100000000000000"),
		overflowTest("Int32", "-2147483649"),
		overflowTest("Int32", "2147483648"),
		overflowTest("Int32", "100000000000000"),
		{
			name: "Int64 ok",
			src: `[foo |
					a Int64 := -9223372036854775808.
					b Int64 := -9223372036854775807.
					c Int64 := -1.
					d Int64 := 0.
					e Int64 := -1.
					f Int64 := 9223372036854775806.
					g Int64 := 9223372036854775807.
				]`,
			err: "",
		},
		overflowTest("Int64", "-100000000000000000000000"),
		overflowTest("Int64", "-9223372036854775809"),
		overflowTest("Int64", "9223372036854775808"),
		overflowTest("Int64", "100000000000000000000000"),
		{
			name: "Uint8 ok",
			src: `[foo |
					a Uint8 := 0.
					b Uint8 := 1.
					c Uint8 := 100.
					d Uint8 := 254.
					e Uint8 := 255.
				]`,
			err: "",
		},
		{
			name: "Uint8 negative",
			src:  "[foo | x Uint8 := -1 ]",
			err:  "Uint8 cannot represent -1: negative unsigned",
		},
		overflowTest("Uint8", "256"),
		overflowTest("Uint8", "10000"),
		{
			name: "Uint16 ok",
			src: `[foo |
					a Uint16 := 0.
					b Uint16 := 1.
					c Uint16 := 100.
					d Uint16 := 65534.
					e Uint16 := 65535.
				]`,
			err: "",
		},
		{
			name: "Uint16 negative",
			src:  "[foo | x Uint16 := -1 ]",
			err:  "Uint16 cannot represent -1: negative unsigned",
		},
		overflowTest("Uint16", "65536"),
		overflowTest("Uint16", "1000000"),
		{
			name: "Uint32 ok",
			src: `[foo |
					a Uint32 := 0.
					b Uint32 := 1.
					c Uint32 := 100.
					d Uint32 := 4294967294.
					e Uint32 := 4294967295.
				]`,
			err: "",
		},
		{
			name: "Uint32 negative",
			src:  "[foo | x Uint32 := -1 ]",
			err:  "Uint32 cannot represent -1: negative unsigned",
		},
		overflowTest("Uint32", "4294967296"),
		overflowTest("Uint32", "10000000000000"),
		{
			name: "Uint64 ok",
			src: `[foo |
					a Uint64 := 0.
					b Uint64 := 1.
					c Uint64 := 100.
					d Uint64 := 18446744073709551615.
					e Uint64 := 18446744073709551615.
				]`,
			err: "",
		},
		{
			name: "Uint64 negative",
			src:  "[foo | x Uint64 := -1 ]",
			err:  "Uint64 cannot represent -1: negative unsigned",
		},
		overflowTest("Uint64", "18446744073709551616"),
		overflowTest("Uint64", "100000000000000000000000"),
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func overflowTest(typ, val string) checkTest {
	return checkTest{
		name: fmt.Sprintf("%s %s overflow", typ, val),
		src:  fmt.Sprintf("[foo | x %s := %s]", typ, val),
		err:  fmt.Sprintf("%s cannot represent %s: overflow", typ, val),
	}
}

func TestFloatLit(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src: `[ foo |
					w := 3.1415.
					x Float := 3.1415.
					y Float32 := 3.1415.
					z Float64 := 3.1415.
					a Int := 3.00000.
					b Int := -3.00000.
					c Uint := 3.00000.
				]`,
			err: "",
		},
		{
			name: "bad truncation",
			src:  "[foo | x Int := 3.14]",
			err:  "Int cannot represent 3.14: truncation",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

type checkTest struct {
	name  string
	src   string
	mods  [][2]string
	err   string // regexp, "" means no error
	trace bool
	dump  bool
}

func (test checkTest) run(t *testing.T) {
	mod, err := parseString(test.src)
	if err != nil {
		t.Fatalf("failed to parse %q: %v", test.src, err)
	}
	if test.dump {
		t.Log("mod:\n", pretty.String(mod))
	}
	opts := []Opt{testImporter(test.mods)}
	if test.trace {
		opts = append(opts, Trace)
	}
	switch errs := Check(mod, opts...); {
	case test.err == "" && len(errs) == 0:
		break // good
	case test.err == "" && len(errs) > 0:
		t.Errorf("got\n%v\nexpected nil", errs)
	case test.err != "" && len(errs) == 0:
		t.Errorf("got nil, expected matching %q", test.err)
	case !regexp.MustCompile(test.err).MatchString(fmt.Sprintf("%v", errs)):
		t.Errorf("got\n%v,\nexpected matching %q", errs, test.err)
	}
}

func testImporter(mods [][2]string) Opt {
	return func(x *state) {
		x.importer = func(name string) (*Mod, error) {
			for _, m := range mods {
				if m[0] != name {
					continue
				}
				p := NewParser(m[0])
				err := p.Parse("", strings.NewReader(m[1]))
				return p.Mod(), err
			}
			return nil, errors.New("not found")
		}
	}
}
