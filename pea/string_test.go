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
			"import \"foo\"",
			"import foo",
		},
		{
			"[max: x Int and: y Int ^Int | ]",
			"[max: Int and: Int ^Int]",
		},
		{
			"#Sub [max: x Int and: y Int ^Int | ]",
			"#Sub [max: Int and: Int ^Int]",
		},
		{
			"Int [orMax: x Int ^Int | ]",
			"Int [orMax: Int ^Int]",
		},
		{
			"#Sub Int [orMax: x Int ^Int | ]",
			"#Sub Int [orMax: Int ^Int]",
		},
		{
			"Int [add: x Int | ]",
			"Int [add: Int]",
		},
		{
			"Int [+ x Int ^Int | ]",
			"Int [+ Int ^Int]",
		},
		{
			"Int [neg ^Int | ]",
			"Int [neg ^Int]",
		},
		{
			"Int [inc | ]",
			"Int [inc]",
		},
		{
			"x := [ 5 ]",
			"x",
		},
		{
			"Int := Int32",
			"Int := Int32",
		},
		{
			"T IntMap := (Int, T) Map",
			"T IntMap := (Int, T) Map",
		},
		{
			"#Sub1 #Sub2 x := [ 5 ]",
			"#Sub1 #Sub2 x",
		},
		{
			"Point { x: Float y: Int }",
			"Point {x: Float y: Int}",
		},
		{
			"#Sub1 #Sub2 Point { x: Float y: Int }",
			"#Sub1 #Sub2 Point {x: Float y: Int}",
		},
		{
			"#Sub1 (X, Y) Pair { x: X y: Y }",
			"#Sub1 (X, Y) Pair {x: X y: Y}",
		},
		{
			"#Sub1 T Vec { data: T Array }",
			"#Sub1 T Vec {data: T Array}",
		},
		{
			"(K Key, V) Map {}",
			"(K Key, V) Map {}",
		},
		{
			"T Opt {some: T, none}",
			"T Opt {some: T, none}",
		},
		{
			"T! { error: String, ok: T }",
			"T! {error: String, ok: T}",
		},
		{
			"T Ord { [= T& ^Bool] [< T& ^Bool] }",
			"T Ord {[= T & ^Bool] [< T & ^Bool]}",
		},
		{
			"Foo { [bar] }",
			"Foo {[bar]}",
		},
		{
			"Foo { [bar: Int baz: Float Array] }",
			"Foo {[bar: Int baz: Float Array]}",
		},
		{
			"[x: f (String, Int) Map | ]",
			"[x: (String, Int) Map]",
		},
		{
			"[do: f [Int, Float, String Array | String] | ]",
			"[do: [Int, Float, String Array | String]]",
		},
		{
			"[do: f [Int, Float, String Array] | ]",
			"[do: [Int, Float, String Array]]",
		},
		{
			"T [foo: t T | ]",
			"T [foo: T]",
		},
		{
			"(K Key, V) [foo: k K bar: v V | ]",
			"(K Key, V) [foo: K bar: V]",
		},
		{
			"[x: v Int Array? Vec | ]",
			"[x: Int Array ? Vec]",
		},
		{
			"[x: _ #Foo #Bar #Baz Int | ]",
			"[x: #Foo #Bar #Baz Int]",
		},
		{
			"#Nest0 ( #Nest1 ( #Nest2 Point {} ) )",
			"#Nest0 #Nest1 #Nest2 Point {}",
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

func parseString(str string) (*Mod, error) {
	p := NewParser("#Main")
	if err := p.Parse("", strings.NewReader(str)); err != nil {
		return nil, err
	}
	return p.Mod(), nil
}
