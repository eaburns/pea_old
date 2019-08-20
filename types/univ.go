package types

import (
	"bytes"
	"text/template"

	"github.com/eaburns/pea/ast"
)

// univ are the definitions of the universal package.
// This is executed with text/template.
// {{IntSize}} is the bit-size of the Int, UInt, and Word type aliases.
// {{FloatSize}} is the bit-size of the Float type alias.
var univ = `
	type Nil {}

	type T& {}

	type Bool { true, false }
	val true := [ {Bool | true} ]
	val false := [ {Bool | true} ]

	type String {}
	meth String [ byteSize ^Int |]
	meth String [ atByte: _ Int ^Byte |]
	meth String [ fromByte: _ Int toByte: _ Int ^String |]

	type T Array {}
	meth T Array [ size ^Int |]
	meth T Array [ at: _ Int ^Byte |]
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

	type Int8 {}
	meth Int8 [ & _ Int8 ^Int8 |]
	meth Int8 [ | _ Int8 ^Int8 |]
	meth Int8 [ not ^Int8 |]
	meth Int8 [ >> _ UInt ^Int8 |]
	meth Int8 [ << _ UInt ^Int8 |]
	meth Int8 [ neg ^Int8 |]
	meth Int8 [ + _ Int8 ^Int8 |]
	meth Int8 [ - _ Int8 ^Int8 |]
	meth Int8 [ * _ Int8 ^Int8 |]
	meth Int8 [ / _ Int8 ^Int8 |]
	meth Int8 [ % _ Int8 ^Int8 |]
	meth Int8 [ = _ Int8 ^Bool |]
	meth Int8 [ != _ Int8 ^Bool |]
	meth Int8 [ < _ Int8 ^Bool |]
	meth Int8 [ <= _ Int8 ^Bool |]
	meth Int8 [ > _ Int8 ^Bool |]
	meth Int8 [ >= _ Int8 ^Bool |]
	meth Int8 [ asInt8 ^Int8 |]
	meth Int8 [ asInt16 ^Int16 |]
	meth Int8 [ asInt32 ^Int32 |]
	meth Int8 [ asInt64 ^Int64 |]
	meth Int8 [ asInt ^Int |]
	meth Int8 [ asUInt8 ^UInt8 |]
	meth Int8 [ asUInt16 ^Uint16 |]
	meth Int8 [ asUInt32 ^UInt32 |]
	meth Int8 [ asUInt64 ^UInt64 |]
	meth Int8 [ asUInt ^UInt |]
	meth Int8 [ asByte ^Byte |]
	meth Int8 [ asWord ^Word |]
	meth Int8 [ asFloat32 ^Float |]
	meth Int8 [ asFloat64 ^Float64 |]
	meth Int8 [ asFloat ^Float |]

	type Int16 {}
	meth Int16 [ & _ Int16 ^Int16 |]
	meth Int16 [ | _ Int16 ^Int16 |]
	meth Int16 [ not ^Int16 |]
	meth Int16 [ >> _ UInt ^Int16 |]
	meth Int16 [ << _ UInt ^Int16 |]
	meth Int16 [ neg ^Int16 |]
	meth Int16 [ + _ Int16 ^Int16 |]
	meth Int16 [ - _ Int16 ^Int16 |]
	meth Int16 [ * _ Int16 ^Int16 |]
	meth Int16 [ / _ Int16 ^Int16 |]
	meth Int16 [ % _ Int16 ^Int16 |]
	meth Int16 [ = _ Int16 ^Bool |]
	meth Int16 [ != _ Int16 ^Bool |]
	meth Int16 [ < _ Int16 ^Bool |]
	meth Int16 [ <= _ Int16 ^Bool |]
	meth Int16 [ > _ Int16 ^Bool |]
	meth Int16 [ >= _ Int16 ^Bool |]
	meth Int16 [ asInt8 ^Int8 |]
	meth Int16 [ asInt16 ^Int16 |]
	meth Int16 [ asInt32 ^Int32 |]
	meth Int16 [ asInt64 ^Int64 |]
	meth Int16 [ asInt ^Int |]
	meth Int16 [ asUInt8 ^UInt8 |]
	meth Int16 [ asUInt16 ^Uint16 |]
	meth Int16 [ asUInt32 ^UInt32 |]
	meth Int16 [ asUInt64 ^UInt64 |]
	meth Int16 [ asUInt ^UInt |]
	meth Int16 [ asByte ^Byte |]
	meth Int16 [ asWord ^Word |]
	meth Int16 [ asFloat32 ^Float |]
	meth Int16 [ asFloat64 ^Float64 |]
	meth Int16 [ asFloat ^Float |]

	type Int32 {}
	meth Int32 [ & _ Int32 ^Int32 |]
	meth Int32 [ | _ Int32 ^Int32 |]
	meth Int32 [ not ^Int32 |]
	meth Int32 [ >> _ UInt ^Int32 |]
	meth Int32 [ << _ UInt ^Int32 |]
	meth Int32 [ neg ^Int32 |]
	meth Int32 [ + _ Int32 ^Int32 |]
	meth Int32 [ - _ Int32 ^Int32 |]
	meth Int32 [ * _ Int32 ^Int32 |]
	meth Int32 [ / _ Int32 ^Int32 |]
	meth Int32 [ % _ Int32 ^Int32 |]
	meth Int32 [ = _ Int32 ^Bool |]
	meth Int32 [ != _ Int32 ^Bool |]
	meth Int32 [ < _ Int32 ^Bool |]
	meth Int32 [ <= _ Int32 ^Bool |]
	meth Int32 [ > _ Int32 ^Bool |]
	meth Int32 [ >= _ Int32 ^Bool |]
	meth Int32 [ asInt8 ^Int8 |]
	meth Int32 [ asInt16 ^Int16 |]
	meth Int32 [ asInt32 ^Int32 |]
	meth Int32 [ asInt64 ^Int64 |]
	meth Int32 [ asInt ^Int |]
	meth Int32 [ asUInt8 ^UInt8 |]
	meth Int32 [ asUInt16 ^Uint16 |]
	meth Int32 [ asUInt32 ^UInt32 |]
	meth Int32 [ asUInt64 ^UInt64 |]
	meth Int32 [ asUInt ^UInt |]
	meth Int32 [ asByte ^Byte |]
	meth Int32 [ asWord ^Word |]
	meth Int32 [ asFloat32 ^Float |]
	meth Int32 [ asFloat64 ^Float64 |]
	meth Int32 [ asFloat ^Float |]

	type Int64 {}
	meth Int64 [ & _ Int64 ^Int64 |]
	meth Int64 [ | _ Int64 ^Int64 |]
	meth Int64 [ not ^Int64 |]
	meth Int64 [ >> _ UInt ^Int64 |]
	meth Int64 [ << _ UInt ^Int64 |]
	meth Int64 [ neg ^Int64 |]
	meth Int64 [ + _ Int64 ^Int64 |]
	meth Int64 [ - _ Int64 ^Int64 |]
	meth Int64 [ * _ Int64 ^Int64 |]
	meth Int64 [ / _ Int64 ^Int64 |]
	meth Int64 [ % _ Int64 ^Int64 |]
	meth Int64 [ = _ Int64 ^Bool |]
	meth Int64 [ != _ Int64 ^Bool |]
	meth Int64 [ < _ Int64 ^Bool |]
	meth Int64 [ <= _ Int64 ^Bool |]
	meth Int64 [ > _ Int64 ^Bool |]
	meth Int64 [ >= _ Int64 ^Bool |]
	meth Int64 [ asInt8 ^Int8 |]
	meth Int64 [ asInt16 ^Int16 |]
	meth Int64 [ asInt32 ^Int32 |]
	meth Int64 [ asInt64 ^Int64 |]
	meth Int64 [ asInt ^Int |]
	meth Int64 [ asUInt8 ^UInt8 |]
	meth Int64 [ asUInt16 ^Uint16 |]
	meth Int64 [ asUInt32 ^UInt32 |]
	meth Int64 [ asUInt64 ^UInt64 |]
	meth Int64 [ asUInt ^UInt |]
	meth Int64 [ asByte ^Byte |]
	meth Int64 [ asWord ^Word |]
	meth Int64 [ asFloat32 ^Float |]
	meth Int64 [ asFloat64 ^Float64 |]
	meth Int64 [ asFloat ^Float |]

	type Int := Int{{ .IntSize }}.

	type UInt8 {}
	meth UInt8 [ & _ UInt8 ^UInt8 |]
	meth UInt8 [ | _ UInt8 ^UInt8 |]
	meth UInt8 [ not ^UInt8 |]
	meth UInt8 [ >> _ UInt ^UInt8 |]
	meth UInt8 [ << _ UInt ^UInt8 |]
	meth UInt8 [ neg ^UInt8 |]
	meth UInt8 [ + _ UInt8 ^UInt8 |]
	meth UInt8 [ - _ UInt8 ^UInt8 |]
	meth UInt8 [ * _ UInt8 ^UInt8 |]
	meth UInt8 [ / _ UInt8 ^UInt8 |]
	meth UInt8 [ % _ UInt8 ^UInt8 |]
	meth UInt8 [ = _ UInt8 ^Bool |]
	meth UInt8 [ != _ UInt8 ^Bool |]
	meth UInt8 [ < _ UInt8 ^Bool |]
	meth UInt8 [ <= _ UInt8 ^Bool |]
	meth UInt8 [ > _ UInt8 ^Bool |]
	meth UInt8 [ >= _ UInt8 ^Bool |]
	meth UInt8 [ asInt8 ^Int8 |]
	meth UInt8 [ asInt16 ^Int16 |]
	meth UInt8 [ asInt32 ^Int32 |]
	meth UInt8 [ asInt64 ^Int64 |]
	meth UInt8 [ asInt ^Int |]
	meth UInt8 [ asUInt8 ^UInt8 |]
	meth UInt8 [ asUInt16 ^Uint16 |]
	meth UInt8 [ asUInt32 ^UInt32 |]
	meth UInt8 [ asUInt64 ^UInt64 |]
	meth UInt8 [ asUInt ^UInt |]
	meth UInt8 [ asByte ^Byte |]
	meth UInt8 [ asWord ^Word |]
	meth UInt8 [ asFloat32 ^Float |]
	meth UInt8 [ asFloat64 ^Float64 |]
	meth UInt8 [ asFloat ^Float |]

	type UInt16 {}
	meth UInt16 [ & _ UInt16 ^UInt16 |]
	meth UInt16 [ | _ UInt16 ^UInt16 |]
	meth UInt16 [ not ^UInt16 |]
	meth UInt16 [ >> _ UInt ^UInt16 |]
	meth UInt16 [ << _ UInt ^UInt16 |]
	meth UInt16 [ neg ^UInt16 |]
	meth UInt16 [ + _ UInt16 ^UInt16 |]
	meth UInt16 [ - _ UInt16 ^UInt16 |]
	meth UInt16 [ * _ UInt16 ^UInt16 |]
	meth UInt16 [ / _ UInt16 ^UInt16 |]
	meth UInt16 [ % _ UInt16 ^UInt16 |]
	meth UInt16 [ = _ UInt16 ^Bool |]
	meth UInt16 [ != _ UInt16 ^Bool |]
	meth UInt16 [ < _ UInt16 ^Bool |]
	meth UInt16 [ <= _ UInt16 ^Bool |]
	meth UInt16 [ > _ UInt16 ^Bool |]
	meth UInt16 [ >= _ UInt16 ^Bool |]
	meth UInt16 [ asInt8 ^Int8 |]
	meth UInt16 [ asInt16 ^Int16 |]
	meth UInt16 [ asInt32 ^Int32 |]
	meth UInt16 [ asInt64 ^Int64 |]
	meth UInt16 [ asInt ^Int |]
	meth UInt16 [ asUInt8 ^UInt8 |]
	meth UInt16 [ asUInt16 ^Uint16 |]
	meth UInt16 [ asUInt32 ^UInt32 |]
	meth UInt16 [ asUInt64 ^UInt64 |]
	meth UInt16 [ asUInt ^UInt |]
	meth UInt16 [ asByte ^Byte |]
	meth UInt16 [ asWord ^Word |]
	meth UInt16 [ asFloat32 ^Float |]
	meth UInt16 [ asFloat64 ^Float64 |]
	meth UInt16 [ asFloat ^Float |]

	type UInt32 {}
	meth UInt32 [ & _ UInt32 ^UInt32 |]
	meth UInt32 [ | _ UInt32 ^UInt32 |]
	meth UInt32 [ not ^UInt32 |]
	meth UInt32 [ >> _ UInt ^UInt32 |]
	meth UInt32 [ << _ UInt ^UInt32 |]
	meth UInt32 [ neg ^UInt32 |]
	meth UInt32 [ + _ UInt32 ^UInt32 |]
	meth UInt32 [ - _ UInt32 ^UInt32 |]
	meth UInt32 [ * _ UInt32 ^UInt32 |]
	meth UInt32 [ / _ UInt32 ^UInt32 |]
	meth UInt32 [ % _ UInt32 ^UInt32 |]
	meth UInt32 [ = _ UInt32 ^Bool |]
	meth UInt32 [ != _ UInt32 ^Bool |]
	meth UInt32 [ < _ UInt32 ^Bool |]
	meth UInt32 [ <= _ UInt32 ^Bool |]
	meth UInt32 [ > _ UInt32 ^Bool |]
	meth UInt32 [ >= _ UInt32 ^Bool |]
	meth UInt32 [ asInt8 ^Int8 |]
	meth UInt32 [ asInt16 ^Int16 |]
	meth UInt32 [ asInt32 ^Int32 |]
	meth UInt32 [ asInt64 ^Int64 |]
	meth UInt32 [ asInt ^Int |]
	meth UInt32 [ asUInt8 ^UInt8 |]
	meth UInt32 [ asUInt16 ^Uint16 |]
	meth UInt32 [ asUInt32 ^UInt32 |]
	meth UInt32 [ asUInt64 ^UInt64 |]
	meth UInt32 [ asUInt ^UInt |]
	meth UInt32 [ asByte ^Byte |]
	meth UInt32 [ asWord ^Word |]
	meth UInt32 [ asFloat32 ^Float |]
	meth UInt32 [ asFloat64 ^Float64 |]
	meth UInt32 [ asFloat ^Float |]

	type UInt64 {}
	meth UInt64 [ & _ UInt64 ^UInt64 |]
	meth UInt64 [ | _ UInt64 ^UInt64 |]
	meth UInt64 [ not ^UInt64 |]
	meth UInt64 [ >> _ UInt ^UInt64 |]
	meth UInt64 [ << _ UInt ^UInt64 |]
	meth UInt64 [ neg ^UInt64 |]
	meth UInt64 [ + _ UInt64 ^UInt64 |]
	meth UInt64 [ - _ UInt64 ^UInt64 |]
	meth UInt64 [ * _ UInt64 ^UInt64 |]
	meth UInt64 [ / _ UInt64 ^UInt64 |]
	meth UInt64 [ % _ UInt64 ^UInt64 |]
	meth UInt64 [ = _ UInt64 ^Bool |]
	meth UInt64 [ != _ UInt64 ^Bool |]
	meth UInt64 [ < _ UInt64 ^Bool |]
	meth UInt64 [ <= _ UInt64 ^Bool |]
	meth UInt64 [ > _ UInt64 ^Bool |]
	meth UInt64 [ >= _ UInt64 ^Bool |]
	meth UInt64 [ asInt8 ^Int8 |]
	meth UInt64 [ asInt16 ^Int16 |]
	meth UInt64 [ asInt32 ^Int32 |]
	meth UInt64 [ asInt64 ^Int64 |]
	meth UInt64 [ asInt ^Int |]
	meth UInt64 [ asUInt8 ^UInt8 |]
	meth UInt64 [ asUInt16 ^Uint16 |]
	meth UInt64 [ asUInt32 ^UInt32 |]
	meth UInt64 [ asUInt64 ^UInt64 |]
	meth UInt64 [ asUInt ^UInt |]
	meth UInt64 [ asByte ^Byte |]
	meth UInt64 [ asWord ^Word |]
	meth UInt64 [ asFloat32 ^Float |]
	meth UInt64 [ asFloat64 ^Float64 |]
	meth UInt64 [ asFloat ^Float |]

	type UInt := UInt{{.IntSize}}.

	type Byte := UInt8.

	type Word := Uint.

	type Float32 {}
	meth Float32 [ neg ^Float32 |]
	meth Float32 [ + _ Float32 ^Float32 |]
	meth Float32 [ - _ Float32 ^Float32 |]
	meth Float32 [ * _ Float32 ^Float32 |]
	meth Float32 [ / _ Float32 ^Float32 |]
	meth Float32 [ % _ Float32 ^Float32 |]
	meth Float32 [ = _ Float32 ^Bool |]
	meth Float32 [ != _ Float32 ^Bool |]
	meth Float32 [ < _ Float32 ^Bool |]
	meth Float32 [ <= _ Float32 ^Bool |]
	meth Float32 [ > _ Float32 ^Bool |]
	meth Float32 [ >= _ Float32 ^Bool |]
	meth Float32 [ asInt8 ^Int8 |]
	meth Float32 [ asInt16 ^Int16 |]
	meth Float32 [ asInt32 ^Int32 |]
	meth Float32 [ asInt64 ^Int64 |]
	meth Float32 [ asInt ^Int |]
	meth Float32 [ asUInt8 ^UInt8 |]
	meth Float32 [ asUInt16 ^Uint16 |]
	meth Float32 [ asUInt32 ^UInt32 |]
	meth Float32 [ asUInt64 ^UInt64 |]
	meth Float32 [ asUInt ^UInt |]
	meth Float32 [ asByte ^Byte |]
	meth Float32 [ asWord ^Word |]
	meth Float32 [ asFloat32 ^Float |]
	meth Float32 [ asFloat64 ^Float64 |]
	meth Float32 [ asFloat ^Float |]

	type Float64 {}
	meth Float64 [ neg ^Float64 |]
	meth Float64 [ + _ Float64 ^Float64 |]
	meth Float64 [ - _ Float64 ^Float64 |]
	meth Float64 [ * _ Float64 ^Float64 |]
	meth Float64 [ / _ Float64 ^Float64 |]
	meth Float64 [ % _ Float64 ^Float64 |]
	meth Float64 [ = _ Float64 ^Bool |]
	meth Float64 [ != _ Float64 ^Bool |]
	meth Float64 [ < _ Float64 ^Bool |]
	meth Float64 [ <= _ Float64 ^Bool |]
	meth Float64 [ > _ Float64 ^Bool |]
	meth Float64 [ >= _ Float64 ^Bool |]
	meth Float64 [ asInt8 ^Int8 |]
	meth Float64 [ asInt16 ^Int16 |]
	meth Float64 [ asInt32 ^Int32 |]
	meth Float64 [ asInt64 ^Int64 |]
	meth Float64 [ asInt ^Int |]
	meth Float64 [ asUInt8 ^UInt8 |]
	meth Float64 [ asUInt16 ^Uint16 |]
	meth Float64 [ asUInt32 ^UInt32 |]
	meth Float64 [ asUInt64 ^UInt64 |]
	meth Float64 [ asUInt ^UInt |]
	meth Float64 [ asByte ^Byte |]
	meth Float64 [ asWord ^Word |]
	meth Float64 [ asFloat32 ^Float |]
	meth Float64 [ asFloat64 ^Float64 |]
	meth Float64 [ asFloat ^Float |]

	type Float := Float{{.FloatSize}}.
`

func newUniv(x *state) Import {
	p := ast.NewParser("")
	tmp, err := template.New("").Parse(univ)
	if err != nil {
		panic("failed to parse template: " + err.Error())
	}
	var buf bytes.Buffer
	if err := tmp.Execute(&buf, x); err != nil {
		panic("failed to execute template: " + err.Error())
	}
	if err := p.Parse("", bytes.NewReader(buf.Bytes())); err != nil {
		panic("parse error in univ: " + err.Error())
	}
	defs, errs := gather(x, make(map[string]Def), p.Mod().Files[0].Defs)
	if len(errs) > 0 {
		panic("check error in univ: " + errs[0].Error())
	}
	return Import{Defs: defs}
}
