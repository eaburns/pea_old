package pea

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eaburns/peggy/peg"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestInst(t *testing.T) {
	tests := []struct {
		name string
		def  string
		typ  string
		want string
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
			name: "fun sig",
			def:  "(X, Y) Pair (T Y Key) [ foo: x X bar: y Y ^(X, Y) Pair List | ]",
			typ:  "(Int Array, String) Pair",
			want: "Pair (T String Key) [ foo: x Int Array bar: y String ^(Int Array, String) Pair List | ]",
		},
		{
			name: "ret",
			def:  "X List [ toArray ^X Array | ^{ X Array | 5; 6; 6 } ]",
			typ:  "Int List",
			want: "List [ toArray ^Int Array | ^{ Int Array | 5; 6; 6 } ]",
		},
		{
			name: "assign",
			def:  "X List [ toArray ^X Array | x := { X Array | 5; 6; 6 }. ^x ]",
			typ:  "Int List",
			want: "List [ toArray ^Int Array | x := { Int Array | 5; 6; 6 }. ^x ]",
		},
		{
			name: "call",
			def:  "X List [ foo | y bar: {X Array|} baz: {X Array|}, qux: {X Array|} ]",
			typ:  "Int List",
			want: "List [ foo | y bar: {Int Array|} baz: {Int Array|}, qux: {Int Array|} ]",
		},
		{
			name: "block",
			def:  "X List [ foo | [ :x X | {X Array|} ]  ]",
			typ:  "Int List",
			want: "List [ foo | [ :x Int | {Int Array|} ]  ]",
		},
		{
			name: "primitives",
			def:  "X List [ foo | id. 123. 3.14. 'a'. `string`. #xyz foo ]",
			typ:  "Int List",
			want: "List [ foo | id. 123. 3.14. 'a'. `string`. #xyz foo ]",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mod, err := parseMod(test.def)
			if err != nil {
				t.Fatalf("failed to parse mod: %s", err)
			}
			typ, err := parseTypeName(test.typ)
			if err != nil {
				t.Fatalf("failed to parse type name: %s", err)
			}
			want, err := parseDef(test.want)
			if err != nil {
				t.Fatalf("failed to parse expected def: %s", err)
			}
			got, errs := inst(mod, typ)
			if len(errs) > 0 {
				t.Fatalf("failed to inst: %v", convertErrors(errs))
			}
			diff := cmp.Diff(want, got,
				ignoreUnexported,
				// Ignore the type signature.
				// We just want to compare the body of the type/fun.
				cmpopts.IgnoreTypes(TypeSig{}),
			)
			if diff != "" {
				t.Errorf("got %s, wanted %s\n%s", got, want, diff)
			}
		})
	}
}

func inst(mod *Mod, typ TypeName) (Def, []checkError) {
	s := newState(mod)
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
