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
			"#main [max: Int and: Int ^Int]",
		},
		{
			"#sub [max: x Int and: y Int ^Int | ]",
			"#main #sub [max: Int and: Int ^Int]",
		},
		{
			"Int [orMax: x Int ^Int | ]",
			"#main Int [orMax: Int ^Int]",
		},
		{
			"#sub Int [orMax: x Int ^Int | ]",
			"#main #sub Int [orMax: Int ^Int]",
		},
		{
			"Int [add: x Int | ]",
			"#main Int [add: Int]",
		},
		{
			"Int [+ x Int ^Int | ]",
			"#main Int [+ Int ^Int]",
		},
		{
			"Int [neg ^Int | ]",
			"#main Int [neg ^Int]",
		},
		{
			"Int [inc | ]",
			"#main Int [inc]",
		},
		{
			"x := [ 5 ]",
			"#main x",
		},
		{
			"Int := Int32",
			"#main Int := Int32",
		},
		{
			"T IntMap := (Int, T) Map",
			"#main T IntMap := (Int, T) Map",
		},
		{
			"#sub1 #sub2 x := [ 5 ]",
			"#main #sub1 #sub2 x",
		},
		{
			"Point { x: Float y: Int }",
			"#main Point {x: Float y: Int}",
		},
		{
			"#sub1 #sub2 Point { x: Float y: Int }",
			"#main #sub1 #sub2 Point {x: Float y: Int}",
		},
		{
			"#sub1 (X, Y) Pair { x: X y: Y }",
			"#main #sub1 (X, Y) Pair {x: X y: Y}",
		},
		{
			"#sub1 T Vec { data: T Array }",
			"#main #sub1 T Vec {data: T Array}",
		},
		{
			"(K Key, V) Map {}",
			"#main (K Key, V) Map {}",
		},
		{
			"T Opt {some: T, none}",
			"#main T Opt {some: T, none}",
		},
		{
			"T! { error: String, ok: T }",
			"#main T! {error: String, ok: T}",
		},
		{
			"T Ord { [= T& ^Bool] [< T& ^Bool] }",
			"#main T Ord {[= T & ^Bool] [< T & ^Bool]}",
		},
		{
			"Foo { [bar] }",
			"#main Foo {[bar]}",
		},
		{
			"Foo { [bar: Int baz: Float Array] }",
			"#main Foo {[bar: Int baz: Float Array]}",
		},
		{
			"[x: f (String, Int) Map | ]",
			"#main [x: (String, Int) Map]",
		},
		{
			"[do: f [Int, Float, String Array | String] | ]",
			"#main [do: [Int, Float, String Array | String]]",
		},
		{
			"[do: f [Int, Float, String Array] | ]",
			"#main [do: [Int, Float, String Array]]",
		},
		{
			"T [foo: t T | ]",
			"#main T [foo: T]",
		},
		{
			"(K Key, V) [foo: k K bar: v V | ]",
			"#main (K Key, V) [foo: K bar: V]",
		},
		{
			"[x: v Int Array? Vec | ]",
			"#main [x: Int Array ? Vec]",
		},
		{
			"[x: _ #foo #bar #baz Int | ]",
			"#main [x: #main #foo #bar #baz Int]",
		},
		{
			"#nest0 ( #nest1 ( #nest2 Point {} ) )",
			"#main #nest0 #nest1 #nest2 Point {}",
		},
	}
	for _, test := range tests {
		mod, err := parseString(test.in)
		if err != nil {
			t.Errorf("failed to parse %q: %v", test.in, err)
			continue
		}
		got := mod.Defs[0].String()
		if got != test.want {
			t.Errorf("%q.String()= %q, want %q", test.in, got, test.want)
		}
	}
}

func parseString(str string) (*Mod, error) {
	p := NewParser("#main")
	if err := p.Parse("", strings.NewReader(str)); err != nil {
		return nil, err
	}
	return p.Mod(), nil
}
