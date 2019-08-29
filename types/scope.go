package types

import "strings"

type scope struct {
	*state
	up *scope

	// One of each of the following fields is non-nil.
	univ []Def
	mod  *Mod
	file *file
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

func (x *scope) findType(arity int, name string) *Type {
	switch {
	case x == nil:
		return nil
	case x.mod != nil:
		if typ := findType(x.mod.Defs, arity, name); typ != nil {
			return typ
		}
	case x.univ != nil:
		if typ := findType(x.univ, arity, name); typ != nil {
			return typ
		}
	}
	return x.up.findType(arity, name)
}

func (imp *imp) findType(arity int, name string) *Type {
	return findType(imp.defs, arity, name)
}

func findType(defs []Def, arity int, name string) *Type {
	for _, d := range defs {
		if t, ok := d.(*Type); ok && t.Sig.Arity == arity && t.Sig.Name == name {
			return t
		}
	}
	return nil
}
