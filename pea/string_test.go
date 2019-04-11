package pea

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		in   string // source; the first Def is tested against want
		want string
	}{
		{
			"#Sub ( Point {} )",
			"submodule: #Main #Sub",
		},
		{
			"#Sub1 #Sub2 #Sub3 ( Point {} )",
			"submodule: #Main #Sub1 #Sub2 #Sub3",
		},
		{
			"import \"foo\"",
			"import: foo",
		},
		{
			"[max: x Int and: y Int ^Int | ]",
			"function: #Main [max: #Main Int and: #Main Int ^#Main Int]",
		},
		{
			"#Sub [max: x Int and: y Int ^Int | ]",
			"function: #Main #Sub [max: #Main Int and: #Main Int ^#Main Int]",
		},
		{
			"Int [orMax: x Int ^Int | ]",
			"function: #Main Int [orMax: #Main Int ^#Main Int]",
		},
		{
			"#Sub Int [orMax: x Int ^Int | ]",
			"function: #Main #Sub Int [orMax: #Main Int ^#Main Int]",
		},
		{
			"Int [add: x Int | ]",
			"function: #Main Int [add: #Main Int]",
		},
		{
			"Int [+ x Int ^Int | ]",
			"function: #Main Int [+ #Main Int ^#Main Int]",
		},
		{
			"Int [neg ^Int | ]",
			"function: #Main Int [neg ^#Main Int]",
		},
		{
			"Int [inc | ]",
			"function: #Main Int [inc]",
		},
		{
			"x := [ 5 ]",
			"variable: #Main x",
		},
		{
			"#Sub1 #Sub2 x := [ 5 ]",
			"variable: #Main #Sub1 #Sub2 x",
		},
		{
			"Point { x: Float y: Int }",
			"struct: #Main Point {x: #Main Float y: #Main Int}",
		},
		{
			"#Sub1 #Sub2 Point { x: Float y: Int }",
			"struct: #Main #Sub1 #Sub2 Point {x: #Main Float y: #Main Int}",
		},
		{
			"#Sub1 (X, Y) Pair { x: X y: Y }",
			"struct: #Main #Sub1 (X, Y) Pair {x: X y: Y}",
		},
		{
			"#Sub1 T Vec { data: T Array }",
			"struct: #Main #Sub1 T Vec {data: T #Main Array}",
		},
		{
			"(K Key, V) Map {}",
			"struct: #Main (K #Main Key, V) Map {}",
		},
		{
			"T Opt {some: T, none}",
			"enum: #Main T Opt {some: T, none}",
		},
		{
			"T! { error: String, ok: T }",
			"enum: #Main T! {error: #Main String, ok: T}",
		},
		{
			"T Ord { [= T& ^Bool] [< T& ^Bool] }",
			"virtual: #Main T Ord {[= T #Main & ^#Main Bool] [< T #Main & ^#Main Bool]}",
		},
		{
			"Foo { [bar] }",
			"virtual: #Main Foo {[bar]}",
		},
		{
			"Foo { [bar: Int baz: Float Array] }",
			"virtual: #Main Foo {[bar: #Main Int baz: #Main Float #Main Array]}",
		},
		{
			"[x: f (String, Int) Map | ]",
			"function: #Main [x: (#Main String, #Main Int) #Main Map]",
		},
		{
			"[do: f [Int, Float, String Array | String] | ]",
			"function: #Main [do: #Main [#Main Int, #Main Float, #Main String #Main Array | #Main String]]",
		},
		{
			"[do: f [Int, Float, String Array] | ]",
			"function: #Main [do: #Main [#Main Int, #Main Float, #Main String #Main Array]]",
		},
		{
			"T [foo: t T | ]",
			"function: #Main T [foo: T]",
		},
		{
			"(K Key, V) [foo: k K bar: v V | ]",
			"function: #Main (K #Main Key, V) [foo: K bar: V]",
		},
		{
			"[x: v Int Array? Vec | ]",
			"function: #Main [x: #Main Int #Main Array #Main ? #Main Vec]",
		},
		{
			"[x: _ #Foo #Bar #Baz Int | ]",
			"function: #Main [x: #Main #Foo #Bar #Baz Int]",
		},
	}
	for _, test := range tests {
		mod, err := parseString(test.in)
		if err != nil {
			t.Errorf("failed to parse %q: %v", test.in, err)
			continue
		}
		got := mod.Files[0].Defs[0].String()
		if got != test.want {
			t.Errorf("%q.String()= %q, want %q", test.in, got, test.want)
		}
	}
}

func TestKey(t *testing.T) {
	tests := []struct {
		in   string // source; the first Def is tested against want
		want string
	}{
		{
			"[foo: t T | ]",
			"#Main foo:",
		},
		{
			"T [foo: t T | ]",
			"#Main foo:",
		},
		{
			"Point [x ^Int | ]",
			"#Main Point x",
		},
		{
			"Point [x ^Int | ]",
			"#Main Point x",
		},
		{
			"Point [x: i Int | ]",
			"#Main Point x:",
		},
		{
			"Point [* p Point ^Point | ]",
			"#Main Point *",
		},
		{
			"(K Key, V) Map [do: f [(K, V) Point] | ]",
			"#Main Map do:",
		},
		{
			"Point { x: Float y: Float }",
			"#Main Point",
		},
		{
			"T Opt {none, some: T}",
			"#Main Opt",
		},
		{
			"T Key { [= T& ^Bool] [hash ^Int64] }",
			"#Main Key",
		},
	}
	for _, test := range tests {
		mod, err := parseString(test.in)
		if err != nil {
			t.Errorf("failed to parse %q: %v", test.in, err)
			continue
		}
		got := mod.Files[0].Defs[0].(interface{ Key() string }).Key()
		if got != test.want {
			t.Errorf("%q.Key()= %q, want %q", test.in, got, test.want)
		}
	}
}

func parseString(str string) (*Mod, error) {
	p := NewParser("#Main")
	if err := p.Parse("", strings.NewReader(str)); err != nil {
		return nil, err
	}
	return p.Mod(), nil
}
