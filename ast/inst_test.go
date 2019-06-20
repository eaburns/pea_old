package ast

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eaburns/peggy/peg"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCheckInstErrors(t *testing.T) {
	tests := []checkTest{
		{
			name: "arg count mismatch",
			src: `
				(X, Y, Z) Tuple { x: X y: Y z: Z }
				Bad := (Int, String) Tuple.
			`,
			err: "argument count mismatch",
		},
		{
			name: "imported type arg count mismatch",
			src: `
				import "tuple"
				Bad := (Int, String) Tuple.
			`,
			mods: [][2]string{
				{"tuple", "(X, Y, Z) Tuple { x: X y: Y z: Z }"},
			},
			err: "argument count mismatch",
		},
		{
			name: "built-in type arg count mismatch",
			src:  "Bad := Array.",
			err:  "argument count mismatch",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestInst(t *testing.T) {
	tests := []struct {
		name  string
		def   string
		typ   string
		want  string
		trace bool
	}{
		{
			name: "no type vars",
			def:  "Point { x: Float y: Float }",
			typ:  "Point",
			want: "Point { x: Float y: Float }",
		},
		{
			name: "single type variable",
			def:  "T List { t: T next: T List }",
			typ:  "Int Array List",
			want: "List { t: Int Array next: Int Array List }",
		},
		{
			name: "two type variables",
			def:  "(X, Y) Pair { x: X y: Y }",
			typ:  "(Int, Float) Pair",
			want: "Pair { x: Int y: Float }",
		},
		{
			name: "sub type constraint",
			def: `
				(X, Y X Container) Foo { x: X y: Y }
				T Container { [do] }
				IntArray {}
			`,
			typ: "(Int, IntArray) Foo",
			// We don't see the substitution,
			// since the instantiated Args
			// aren't written in the string output.
			want: "Foo { x: Int y: IntArray }",
		},
		{
			name: "alias",
			def:  "T Block1 := (T, Nil) Fun1.",
			typ:  "Int Block1",
			want: "Block1 := (Int, Nil) Fun1.",
		},
		{
			name: "enum",
			def:  "T Opt { none, some: T }",
			typ:  "Int Array Opt",
			want: "Opt { none, some: Int Array }",
		},
		{
			name: "virt",
			def:  "(X, Y) Virt { [ x: X ^Y ] [ foo ] }",
			typ:  "(Int Array, String) Virt",
			want: "Virt { [ x: Int Array ^String ] [ foo ] }",
		},
		{
			name: "meth sig",
			def: `
				(X, Y) Pair (T Y Key) [ foo: x X bar: y Y ^(X, Y) Pair List | ]
				T Key{ [= T& ^Bool] [hash ^Int64] }
				(X, Y) Pair{}
				T List{}
			`,
			typ:  "(Int Array, String) Pair",
			want: "Pair (T String Key) [ foo: x Int Array bar: y String ^(Int Array, String) Pair List | ]",
		},
		{
			name: "ret",
			def: `
				X List [ toArray ^X Array | ^{ X Array | 5; 6; 6 } ]
				T List {}
			`,
			typ:  "Int List",
			want: "List [ toArray ^Int Array | ^{ Int Array | 5; 6; 6 } ]",
		},
		{
			name: "assign",
			def: `
				X List [ toArray ^X Array | x := { X Array | 5; 6; 6 }. ^x ]
				T List {}
			`,
			typ:  "Int List",
			want: "List [ toArray ^Int Array | x := { Int Array | 5; 6; 6 }. ^x ]",
		},
		{
			name: "call",
			def: `
				X List [ foo | y bar: {X Array|} baz: {X Array|}, qux: {X Array|} ]
				T List {}
			`,
			typ:  "Int List",
			want: "List [ foo | y bar: {Int Array|} baz: {Int Array|}, qux: {Int Array|} ]",
		},
		{
			name: "block",
			def: `
				X List [ foo | [ :x X | {X Array|} ]  ]
				T List {}
			`,
			typ:  "Int List",
			want: "List [ foo | [ :x Int | {Int Array|} ]  ]",
		},
		{
			name: "primitives",
			def: `
				X List [ foo | id. 123. 3.14. 'a'. "string". #xyz foo ]
				T List {}
			`,
			typ:  "Int List",
			want: `List [ foo | id. 123. 3.14. 'a'. "string". #xyz foo ]`,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mod, err := parseMod(test.def)
			if err != nil {
				t.Fatalf("failed to parse mod: %s", err)
			}
			x := newScope(mod)
			if es := checkMod(x, mod); len(es) > 0 {
				t.Fatalf("failed to check the source: %v", es)
			}

			x = &scope{state: x.state, parent: x, def: &Fun{
				ModPath: ModPath{Root: mod.Name},
			}}
			typ, err := parseTypeName(test.typ)
			if err != nil {
				t.Fatalf("failed to parse type name: %s", err)
			}
			if es := checkTypeName(x, &typ); len(es) > 0 {
				t.Fatalf("failed to check the type: %v", es)
			}

			want, err := parseDef(test.want)
			if err != nil {
				t.Fatalf("failed to parse expected def: %s", err)
			}

			got, errs := inst(test.trace, mod, typ)
			if len(errs) > 0 {
				t.Fatalf("failed to inst: %v", convertErrors(errs))
			}
			diff := cmp.Diff(want, got,
				ignoreUnexported,
				// Ignore the type signature.
				// We just want to compare the body of the type/fun.
				cmpopts.IgnoreTypes(TypeSig{}),
				// Ignore TypeName.Type, as it can be recursive.
				cmpopts.IgnoreFields(TypeName{}, "Type"),
				// Fields set by check and not by parsing want.
				cmpopts.IgnoreFields(Fun{},
					"RecvType",
					"Self"),
				cmpopts.IgnoreFields(TypeName{}, "Mod"),
			)
			if diff != "" {
				t.Errorf("got %s, wanted %s\n%s", got, want, diff)
			}
		})
	}
}

func inst(trace bool, mod *Mod, typ TypeName) (Def, []checkError) {
	s := newState(mod)
	s.trace = trace
	x := &scope{state: s}

	switch def := mod.Defs[0].(type) {
	case *Fun:
		return def.instRecv(x, typ)
	case *Type:
		return def.inst(x, typ)
	default:
		panic(fmt.Sprintf("bad def type: %T", def))
	}
}

func TestMemoizeMethInst(t *testing.T) {
	mod, err := parseMod("T Array [ foo: _ T | ]")
	if err != nil {
		t.Fatalf("parseMod(…)=%v, want nil", err)
	}
	intArrayName, err := parseTypeName("Int Array")
	if err != nil {
		t.Fatalf("parseTypeName(Int Array)=%v, want nil", err)
	}
	stringArrayName, err := parseTypeName("String Array")
	if err != nil {
		t.Fatalf("parseTypeName(String Array)=%v, want nil", err)
	}

	fun := mod.Defs[0].(*Fun)
	x := &scope{state: newState(mod)}

	intArray0, errs := fun.instRecv(x, intArrayName)
	if len(errs) > 0 {
		t.Fatalf("inst(Int Array)=%v, want nil", errs)
	}
	stringArray, errs := fun.instRecv(x, stringArrayName)
	if len(errs) > 0 {
		t.Fatalf("inst(String Array)=%v, want nil", errs)
	}
	intArray1, errs := fun.instRecv(x, intArrayName)
	if len(errs) > 0 {
		t.Fatalf("inst(Int Array)=%v, want nil", errs)
	}

	if intArray0 != intArray1 {
		t.Error("intArray0 != intArray1")
	}
	if intArray0 == stringArray {
		t.Error("intArray0 == stringArray")
	}
	if intArray1 == stringArray {
		t.Error("intArray1 == stringArray")
	}
}

func TestMemoizeTypeInst(t *testing.T) {
	mod, err := parseMod("X List { x: X }")
	if err != nil {
		t.Fatalf("parseMod(…)=%v, want nil", err)
	}
	x := newScope(mod)
	if es := checkMod(x, mod); len(es) > 0 {
		t.Fatalf("failed to check the source: %v", es)
	}
	x = &scope{state: x.state, parent: x, def: &Fun{
		ModPath: ModPath{Root: mod.Name},
	}}

	intArrayName, err := parseTypeName("Int Array")
	if err != nil {
		t.Fatalf("parseTypeName(Int Array)=%v, want nil", err)
	}
	if es := checkTypeName(x, &intArrayName); len(es) > 0 {
		t.Fatalf("failed to check intArrayName: %v", es)
	}
	stringArrayName, err := parseTypeName("String Array")
	if err != nil {
		t.Fatalf("parseTypeName(String Array)=%v, want nil", err)
	}
	if es := checkTypeName(x, &stringArrayName); len(es) > 0 {
		t.Fatalf("failed to check stringArrayName: %v", es)
	}

	typ := mod.Defs[0].(*Type)

	intArray0, errs := typ.inst(x, intArrayName)
	if len(errs) > 0 {
		t.Fatalf("inst(Int Array)=%v, want nil", errs)
	}
	stringArray, errs := typ.inst(x, stringArrayName)
	if len(errs) > 0 {
		t.Fatalf("inst(String Array)=%v, want nil", errs)
	}
	intArray1, errs := typ.inst(x, intArrayName)
	if len(errs) > 0 {
		t.Fatalf("inst(Int Array)=%v, want nil", errs)
	}

	if intArray0 != intArray1 {
		t.Error("intArray0 != intArray1")
	}
	if intArray0 == stringArray {
		t.Error("intArray0 == stringArray")
	}
	if intArray1 == stringArray {
		t.Error("intArray1 == stringArray")
	}
}

func parseMod(str string) (*Mod, error) {
	p := NewParser("#test")
	if err := p.Parse("", strings.NewReader(str)); err != nil {
		return nil, err
	}
	return p.Mod(), nil
}

func parseDef(str string) (Def, error) {
	p := _NewParser(str)
	p.data = NewParser("#test")
	if pos, perr := _DefAccepts(p, 0); pos < 0 {
		_, fail := _DefFail(p, 0, perr)
		return nil, peg.SimpleError(str, fail)
	}
	_, defs := _DefAction(p, 0)
	return (*defs)[0], nil
}

func parseTypeName(str string) (TypeName, error) {
	p := _NewParser(str)
	p.data = NewParser("#test")
	if pos, perr := _TypeNameAccepts(p, 0); pos < 0 {
		_, fail := _TypeNameFail(p, 0, perr)
		return TypeName{}, peg.SimpleError(str, fail)
	}
	_, tname := _TypeNameAction(p, 0)
	return *tname, nil
}

var ignoreUnexported = cmpopts.IgnoreUnexported(
	Mod{},
	Import{},
	Fun{},
	Parm{},
	Var{},
	TypeSig{},
	TypeName{},
	Type{},
	MethSig{},
	Ret{},
	Assign{},
	Call{},
	Msg{},
	Ctor{},
	Block{},
	ModPath{},
	Ident{},
	Int{},
	Float{},
	Rune{},
	String{},
)
