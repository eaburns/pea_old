package types

import (
	"strings"

	"github.com/eaburns/pea/ast"
)

type scope struct {
	*state
	up *scope

	// One of each of the following fields is non-nil.
	univ     []Def
	mod      *Mod
	file     *file
	typeVar  *Var
	val      *Val
	fun      *Fun
	block    *Block
	variable *Var
}

type file struct {
	ast     *ast.File
	imports []imp
	x       *scope
}

type imp struct {
	path string
	name string
	defs []Def
}

func newUnivScope(x *state) *scope {
	defs, err := x.cfg.Importer.Import(x.cfg, "")
	if err != nil {
		panic(err.Error())
	}
	return &scope{state: x, univ: defs}
}

func (x *scope) new() *scope {
	return &scope{state: x.state, up: x}
}

func (x *scope) function() *Fun {
	switch {
	case x == nil:
		return nil
	case x.fun != nil:
		return x.fun
	default:
		return x.up.function()
	}
}

func (x *scope) findImport(name string) *imp {
	return x._findImport(strings.TrimPrefix(name, "#"))
}

func (x *scope) _findImport(name string) *imp {
	switch {
	case x == nil:
		return nil
	case x.file != nil:
		for i := range x.file.imports {
			if x.file.imports[i].name == name {
				return &x.file.imports[i]
			}
		}
	}
	return x.up._findImport(name)
}

// Returns either *Type or *Var.
func (x *scope) findType(arity int, name string) interface{} {
	switch {
	case x == nil:
		return nil
	case x.typeVar != nil:
		if arity == 0 && x.typeVar.Name == name {
			return x.typeVar
		}
	case x.mod != nil:
		if t := findType(arity, name, x.mod.Defs); t != nil {
			return t
		}
	case x.univ != nil:
		if t := findType(arity, name, x.univ); t != nil {
			return t
		}
	}
	return x.up.findType(arity, name)
}

func (imp *imp) findType(arity int, name string) *Type {
	t := findType(arity, name, imp.defs)
	if t == nil || t.Priv {
		return nil
	}
	return t
}

func findType(arity int, name string, defs []Def) *Type {
	for _, d := range defs {
		if t, ok := d.(*Type); ok && t.Sig.Arity == arity && t.Sig.Name == name {
			return t
		}
	}
	return nil
}

// locals returns the local variable slice of the inner-most Block, Fun, or Val.
func (x *scope) locals() *[]*Var {
	switch {
	case x.fun != nil:
		return &x.fun.Locals
	case x.block != nil:
		return &x.block.Locals
	case x.val != nil:
		return &x.val.Locals
	default:
		// TODO: implement locals() for Block.
		return x.up.locals()
	}
}

// findIdent returns a *Var or *Fun.
// In the *Fun case, the identifier is a unary function in the current module.
func (x *scope) findIdent(name string) interface{} {
	switch {
	case x == nil:
		return nil
	case x.variable != nil && x.variable.Name == name:
		return x.variable
	case x.fun != nil && x.fun.Recv != nil && x.fun.Recv.Type != nil:
		for i := range x.fun.Recv.Type.Fields {
			if f := &x.fun.Recv.Type.Fields[i]; f.Name == name {
				return f
			}
		}
	case x.mod != nil:
		if fun := findIdent(name, x.mod.Defs); fun != nil {
			return fun
		}
	case x.univ != nil:
		if fun := findIdent(name, x.univ); fun != nil {
			return fun
		}
	}
	return x.up.findIdent(name)
}

// findIdent returns either a *Fun that is a unary function or
// a *Var that is a module-level Val.Var.
func findIdent(name string, defs []Def) interface{} {
	for _, def := range defs {
		if fun, ok := def.(*Fun); ok && fun.Recv == nil && fun.Sig.Sel == name {
			return fun
		}
		if val, ok := def.(*Val); ok && val.Var.Name == name {
			return &val.Var
		}
	}
	return nil
}
