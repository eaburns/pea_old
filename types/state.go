package types

import (
	"fmt"
	"reflect"

	"github.com/eaburns/pea/ast"
)

// An Opt is an option to the type checker.
type Opt func(*state)

var (
	// Trace enables tracing of the type checker.
	Trace Opt = func(x *state) { x.trace = true }
)

type state struct {
	astMod *ast.Mod

	// IntSize is the size of Int, UInt, and Word in bits (8, 16, 32, or 64).
	IntSize int
	// FloatSize is the size of Float in bits (32 or 64).
	FloatSize int

	trace  bool
	indent string
}

func newState(astMod *ast.Mod, opts ...Opt) *state {
	s := &state{
		astMod:    astMod,
		IntSize:   64,
		FloatSize: 64,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (x *state) loc(n interface{}) ast.Loc {
	switch n := n.(type) {
	case ast.Node:
		return x.astMod.Loc(n)
	case Node:
		return x.astMod.Loc(n.AST())
	default:
		panic("bad type")
	}
}

func (x *state) err(n interface{}, f string, vs ...interface{}) *checkError {
	return &checkError{loc: x.loc(n), msg: fmt.Sprintf(f, vs...)}
}

// The argument to the returned function,
// if non-empty, only the first element of vs is used.
// It must be a either pointer to a slice of types convertable to error,
// or a pointer to a type convertable to error.
func (x *state) tr(f string, vs ...interface{}) func(...interface{}) {
	if !x.trace {
		return func(...interface{}) {}
	}
	x.log(f, vs...)
	olddent := x.indent
	x.indent += "---"
	return func(errs ...interface{}) {
		defer func() { x.indent = olddent }()
		if len(errs) == 0 {
			return
		}
		v := reflect.ValueOf(errs[0])
		if v.IsNil() || v.Elem().Kind() == reflect.Slice && v.Elem().Len() == 0 {
			return
		}
		x.log("%v", v.Elem().Interface())
	}
}

func (x *state) log(f string, vs ...interface{}) {
	if !x.trace {
		return
	}
	fmt.Printf(x.indent)
	fmt.Printf(f, vs...)
	fmt.Println("")
}
