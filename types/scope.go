package types

import (
	"strings"

	"github.com/eaburns/pea/ast"
)

type scope struct {
	*state
	up *scope

	// One of each of the following fields is non-nil.
	univ    []Def
	mod     *Mod
	file    *file
	typeVar *Var
	fun     *Fun
	local   *Assign
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
	var defs []Def
	switch {
	case x == nil:
		return nil
	case x.typeVar != nil:
		if arity == 0 && x.typeVar.Name == name {
			return x.typeVar
		}
	case x.mod != nil:
		defs = x.mod.Defs
	case x.univ != nil:
		defs = x.univ
	}
	for _, d := range defs {
		if t, ok := d.(*Type); ok && t.Sig.Arity == arity && t.Sig.Name == name {
			return t
		}
	}
	return x.up.findType(arity, name)
}

func (imp *imp) findType(arity int, name string) *Type {
	for _, d := range imp.defs {
		if t, ok := d.(*Type); ok && !t.Priv() && t.Sig.Arity == arity && t.Sig.Name == name {
			return t
		}
	}
	return nil

}
