// Copyright © 2020 The Pea Authors under an MIT-style license.

package types

import (
	"bytes"
	"text/template"

	"github.com/eaburns/pea/ast"
)

// MaxValueParms the maximum number of value: parameters
// for Fun value methods.
const MaxValueParms = 4

// univ are the definitions of the universal package.
// This is string executed with text/template.
//
// The basic package will not emit static calls for the methods below.
// Instead it will emit inline the basic.Stmts that implement the method.
// For example 1+1 will not generate a call to the + method of Int;
// it will generate an add operator.
//
// However, we need the ability to emit the actual methods
// so that they can be used in a virtual table for a virtual type.
//
// To this end, the methods below all have a method body
// consisting of a single, static, call back to themselves.
// When emitting the code for these methods,
// the basic package will not actually emit recursive calls,
// but instead will emit the implementation of the method.
var univ = `
	// print: is not part of the language, but a temporary function
	// that will be removed once proper library support is implemented.
	func [print: _ String]

	func [panic: _ String]

	type _& {}

	type Nil {}

	type Bool {true | false}
	func [true ^Bool | ^{true}]
	func [false ^Bool | ^{false}]

	type String {}
	func [newString: a Byte Array ^String | ^newString: a]
	meth String [byteSize ^Int | ^self byteSize]
	meth String [atByte: x Int ^Byte | ^self atByte: x]
	meth String [fromByte: x Int toByte: y Int ^String | ^self fromByte: x toByte: y]

	type _ Array {}
	func T [newArray: s Int init: f (Int, T) Fun ^T Array | ^newArray: s init: f]
	meth _ Array [size ^Int | ^self size]
	meth T Array [at: x Int ^T& | ^self at: x]
	meth T Array [at: x Int put: y T | self at: x put: y]
	meth T Array [from: x Int to: y Int ^T Array | ^self from: x to: y]

	type T Fun {[value ^T]}
	type (T, U) Fun {[value: T ^U]}
	type (T, U, V) Fun {[value: T value: U ^V]}
	type (T, U, V, W) Fun {[value: T value: U  value: V ^W]}
	type (T, U, V, W, X) Fun {[value: T value: U  value: V value: W ^ X]}

	type Byte := UInt8.
	type Word := UInt.
	type Rune := Int32.

	{{range $_, $t := $.IntTypes}}
		type {{$t}} {}
		meth {{$t}} [& x {{$t}} ^{{$t}} | ^self & x]
		meth {{$t}} [| x {{$t}} ^{{$t}} | ^self | x]
		meth {{$t}} [xor: x {{$t}} ^{{$t}} | ^self xor: x]
		meth {{$t}} [not ^{{$t}} | ^self not]
		meth {{$t}} [>> x Int ^{{$t}} | ^self >> x]
		meth {{$t}} [<< x Int ^{{$t}} | ^self << x]
		meth {{$t}} [neg ^{{$t}} | ^self neg]
		meth {{$t}} [+ x {{$t}} ^{{$t}} | ^self + x]
		meth {{$t}} [- x {{$t}} ^{{$t}} | ^self - x]
		meth {{$t}} [* x {{$t}} ^{{$t}} | ^self * x]
		meth {{$t}} [/ x {{$t}} ^{{$t}} | ^self / x]
		meth {{$t}} [% x {{$t}} ^{{$t}} | ^self % x]
		meth {{$t}} [= x {{$t}} ^Bool | ^self = x]
		meth {{$t}} [!= x {{$t}} ^Bool | ^self != x]
		meth {{$t}} [< x {{$t}} ^Bool | ^self < x]
		meth {{$t}} [<= x {{$t}} ^Bool | ^self <= x]
		meth {{$t}} [> x {{$t}} ^Bool | ^self > x]
		meth {{$t}} [>= x {{$t}} ^Bool | ^self >= x]
		{{range $_, $r := $.IntTypes}}
			meth {{$t}} [as{{$r}} ^{{$r}} | ^self as{{$r}}]
		{{end}}
		{{range $_, $r := $.FloatTypes}}
			meth {{$t}} [as{{$r}} ^{{$r}} | ^self as{{$r}}]
		{{end}}
	{{end}}

	{{range $_, $t := $.FloatTypes}}
		type {{$t}} {}
		meth {{$t}} [neg ^{{$t}} | ^self neg]
		meth {{$t}} [+ x {{$t}} ^{{$t}} | ^self + x]
		meth {{$t}} [- x {{$t}} ^{{$t}} | ^self - x]
		meth {{$t}} [* x {{$t}} ^{{$t}} | ^self * x]
		meth {{$t}} [/ x {{$t}} ^{{$t}} | ^self / x]
		meth {{$t}} [= x {{$t}} ^Bool | ^self = x]
		meth {{$t}} [!= x {{$t}} ^Bool | ^self != x]
		meth {{$t}} [< x {{$t}} ^Bool | ^self < x]
		meth {{$t}} [<= x {{$t}} ^Bool | ^self <= x]
		meth {{$t}} [> x {{$t}} ^Bool | ^self > x]
		meth {{$t}} [>= x {{$t}} ^Bool | ^self >= x]
		{{range $_, $r := $.IntTypes}}
			meth {{$t}} [as{{$r}} ^{{$r}} | ^self as{{$r}}]
		{{end}}
		{{range $_, $r := $.FloatTypes}}
			meth {{$t}} [as{{$r}} ^{{$r}} | ^self as{{$r}}]
		{{end}}
	{{end}}
`

func newUniv(x *state) []Def {
	p := ast.NewParserWithLocs("", nil)
	tmp, err := template.New("").Parse(univ)
	if err != nil {
		panic("failed to parse template: " + err.Error())
	}
	var buf bytes.Buffer
	if err := tmp.Execute(&buf, struct {
		IntTypes   []string
		FloatTypes []string
	}{
		IntTypes: []string{
			"Int", "Int8", "Int16", "Int32", "Int64",
			"UInt", "UInt8", "UInt16", "UInt32", "UInt64",
		},
		FloatTypes: []string{
			"Float", "Float32", "Float64",
		},
	}); err != nil {
		panic("failed to execute template: " + err.Error())
	}
	if err := p.Parse("", bytes.NewReader(buf.Bytes())); err != nil {
		panic("parse error in univ: " + err.Error())
	}
	astMod := p.Mod()
	cfg := x.cfg
	cfg.Trace = false
	mod, errs := check(&scope{state: newState(cfg, astMod)}, astMod)
	if len(errs) > 0 {
		panic("check error in univ: " + errs[0].Error())
	}
	return mod.Defs
}
