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
// This is executed with text/template.
var univ = `
	type Nil {}

	type T& {}

	type Bool { true | false }
	val true Bool := [ {true} ]
	val false Bool := [ {false} ]

	type String {}
	meth String [ byteSize ^Int |]
	meth String [ atByte: _ Int ^Byte |]
	meth String [ fromByte: _ Int toByte: _ Int ^String |]

	type T Array {}
	meth T Array [ size ^Int |]
	meth T Array [ at: _ Int ^T& |]
	meth T Array [ at: _ Int put: _ T |]
	meth T Array [ from: _ Int to: _ Int ^T Array |]

	type T Fun {}
	meth T Fun [ value ^T |]

	type (T, U) Fun {}
	meth (T, U) Fun [ value: _ T ^U |]

	type (T, U, V) Fun {}
	meth (T, U, V) Fun [ value: _ T value: _ U ^V |]

	type (T, U, V, W) Fun {}
	meth (T, U, V, W) Fun [ value: _ T value: _ U  value: _ V ^W |]

	type (T, U, V, W, X) Fun {}
	meth (T, U, V, W, X) Fun [ value: _ T value: _ U  value: _ V value: _ W ^ X |]

	type Byte := UInt8.
	type Word := UInt.
	type Rune := Int32.

	{{range $_, $t := $.IntTypes}}
		type {{$t}} {}
		meth {{$t}} [ & _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ | _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ not ^{{$t}} |]
		meth {{$t}} [ >> _ Int ^{{$t}} |]
		meth {{$t}} [ << _ Int ^{{$t}} |]
		meth {{$t}} [ neg ^{{$t}} |]
		meth {{$t}} [ + _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ - _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ * _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ / _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ % _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ = _ {{$t}} ^Bool |]
		meth {{$t}} [ != _ {{$t}} ^Bool |]
		meth {{$t}} [ < _ {{$t}} ^Bool |]
		meth {{$t}} [ <= _ {{$t}} ^Bool |]
		meth {{$t}} [ > _ {{$t}} ^Bool |]
		meth {{$t}} [ >= _ {{$t}} ^Bool |]
		{{range $_, $r := $.IntTypes}}
			meth {{$t}} [ as{{$r}} ^{{$r}} |]
		{{end}}
		{{range $_, $r := $.FloatTypes}}
			meth {{$t}} [ as{{$r}} ^{{$r}} |]
		{{end}}
	{{end}}

	{{range $_, $t := $.FloatTypes}}
		type {{$t}} {}
		meth {{$t}} [ neg ^{{$t}} |]
		meth {{$t}} [ + _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ - _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ * _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ / _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ % _ {{$t}} ^{{$t}} |]
		meth {{$t}} [ = _ {{$t}} ^Bool |]
		meth {{$t}} [ != _ {{$t}} ^Bool |]
		meth {{$t}} [ < _ {{$t}} ^Bool |]
		meth {{$t}} [ <= _ {{$t}} ^Bool |]
		meth {{$t}} [ > _ {{$t}} ^Bool |]
		meth {{$t}} [ >= _ {{$t}} ^Bool |]
		{{range $_, $r := $.IntTypes}}
			meth {{$t}} [ as{{$r}} ^{{$r}} |]
		{{end}}
		{{range $_, $r := $.FloatTypes}}
			meth {{$t}} [ as{{$r}} ^{{$r}} |]
		{{end}}
	{{end}}
`

func newUniv(x *state) []Def {
	p := ast.NewParser("")
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
	clearAST(mod.Defs)
	return mod.Defs
}
