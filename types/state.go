package types

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/eaburns/pea/ast"
)

type state struct {
	astMod     *ast.Mod
	cfg        Config
	defFiles   map[Def]*file
	gathered   map[Def]bool
	aliasStack []*Type

	insts     []Def
	typeInsts map[interface{}]*Type

	indent string
}

func newState(cfg Config, astMod *ast.Mod) *state {
	x := &state{
		astMod:    astMod,
		cfg:       cfg,
		defFiles:  make(map[Def]*file),
		gathered:  make(map[Def]bool),
		typeInsts: make(map[interface{}]*Type),
	}
	setConfigDefaults(x)
	return x
}

func setConfigDefaults(x *state) {
	switch x.cfg.IntSize {
	case 0:
		x.cfg.IntSize = 64
	case 8, 16, 32, 64:
		break
	default:
		panic("bad IntSize " + strconv.Itoa(x.cfg.IntSize))
	}
	switch x.cfg.FloatSize {
	case 0:
		x.cfg.FloatSize = 64
	case 32, 64:
		break
	default:
		panic("bad FloatSize " + strconv.Itoa(x.cfg.FloatSize))
	}
	if x.cfg.Importer == nil {
		x.cfg.Importer = &dirImporter{}
	}
	if _, ok := x.cfg.Importer.(*importer); !ok {
		x.cfg.Importer = newImporter(x, x.astMod.Name, x.cfg.Importer)
	}
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
	if !x.cfg.Trace {
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
	if !x.cfg.Trace {
		return
	}
	fmt.Printf(x.indent)
	fmt.Printf(f, vs...)
	fmt.Println("")
}
