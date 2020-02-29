// Copyright © 2020 The Pea Authors under an MIT-style license.

package types

import "github.com/eaburns/pea/ast"

type scope struct {
	*state
	up *scope

	// One of each of the following fields is non-nil.
	univ     []Def
	mod      *Mod
	file     *file
	def      Def
	typeVar  *Type
	val      *Val
	fun      *Fun
	block    *Block
	variable *Var
}

type file struct {
	ast     *ast.File
	imports []imp
	x       *scope

	// Fun instances due to calls from in this file.
	funInsts []*Fun
}

type imp struct {
	ast  ast.Node
	used bool
	all  bool
	path string
	name string
	defs []Def
}

func newUnivScope(x *state) *scope {
	defs, err := x.cfg.Importer.Import(x.cfg, x.astMod.Locs, "")
	if err != nil {
		panic(err.Error())
	}
	return &scope{state: x, univ: defs}
}

func (x *scope) new() *scope {
	return &scope{state: x.state, up: x}
}

func use(x *scope, def Def, loc ast.Node) {
	if _, ok := x.defFiles[def]; !ok {
		return
	}
	if fun, ok := def.(*Fun); ok {
		def = fun.Def
	}
	x.use(def, loc)
}

func (x *scope) use(def Def, loc ast.Node) {
	if x.def == nil {
		x.up.use(def, loc)
		return
	}
	uses := x.initDeps[def]
	for _, w := range uses {
		if w.def == x.def {
			return
		}
	}
	x.initDeps[def] = append(uses, witness{x.def, loc})
}

func (x *scope) curFile() *file {
	switch {
	case x == nil:
		return nil
	case x.file != nil:
		return x.file
	default:
		return x.up.curFile()
	}
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
		return x.up.locals()
	}
}

func findImport(x *scope, mod *ast.ModTag) (*imp, *checkError) {
	imp := x.findImport(mod.Text)
	if imp == nil {
		return nil, x.err(mod, "module %s not found", mod.Text)
	}
	return imp, nil
}

func (x *scope) findImport(name string) *imp {
	switch {
	case x == nil:
		return nil
	case x.file != nil:
		for i := range x.file.imports {
			imp := &x.file.imports[i]
			if imp.name == name {
				imp.used = true
				return imp
			}
		}
	}
	return x.up.findImport(name)
}

func refTypeDef(x *scope) *Type {
	for {
		if x.univ != nil {
			return findTypeInDefs(1, "&", x.univ)
		}
		x = x.up
	}
}

func findType(x *scope, loc ast.Node, mod *ast.ModTag, arity int, name string) (typ *Type, err *checkError) {
	if mod != nil {
		imp, err := findImport(x, mod)
		if err != nil {
			return nil, err
		}
		if t := imp.findType(arity, name); t != nil {
			return t, nil
		}
		if arity == 0 {
			return nil, x.err(loc, "type %s %s not found", mod.Text, name)
		}
		return nil, x.err(loc, "type (%d) %s %s not found", arity, mod.Text, name)
	}
	t, err := x.findType(loc, arity, name)
	if err != nil {
		return nil, err
	}
	if t != nil {
		return t, nil
	}
	if arity == 0 {
		return nil, x.err(loc, "type %s not found", name)
	}
	return nil, x.err(loc, "type (%d)%s not found", arity, name)
}

func (x *scope) findType(loc ast.Node, arity int, name string) (*Type, *checkError) {
	switch {
	case x == nil:
		return nil, nil
	case x.typeVar != nil:
		if arity == 0 && x.typeVar.Name == name {
			return x.typeVar, nil
		}
	case x.file != nil:
		switch t, err := x.file.findType(loc, arity, name); {
		case err != nil:
			return nil, err
		case t != nil:
			return t, nil
		}
	case x.mod != nil:
		if t := findTypeInDefs(arity, name, x.mod.Defs); t != nil {
			return t, nil
		}
	case x.univ != nil:
		if t := findTypeInDefs(arity, name, x.univ); t != nil {
			return t, nil
		}
	}
	return x.up.findType(loc, arity, name)
}

func (f *file) findType(loc ast.Node, arity int, name string) (*Type, *checkError) {
	var ts []*Type
	var imps []*imp
	for i := range f.imports {
		imp := &f.imports[i]
		if !imp.all {
			continue
		}
		if t := imp.findType(arity, name); t != nil {
			imp.used = true
			ts = append(ts, t)
			imps = append(imps, imp)
		}
	}
	if len(ts) == 0 {
		return nil, nil
	}
	if len(ts) == 1 {
		return ts[0], nil
	}
	err := f.x.err(loc, "ambiguous type (%d)%s)", arity, name)
	for _, imp := range imps {
		note(err, "imported from %s at %s", imp.name, f.x.loc(imp.ast))
	}
	return nil, err
}

func (imp *imp) findType(arity int, name string) *Type {
	t := findTypeInDefs(arity, name, imp.defs)
	if t == nil || t.Priv {
		return nil
	}
	return t
}

func findTypeInDefs(arity int, name string, defs []Def) *Type {
	for _, d := range defs {
		if t, ok := d.(*Type); ok && t.Arity == arity && t.Name == name {
			return t
		}
	}
	return nil
}

// findIdent returns a *Var or *Fun.
// In the *Fun case, the identifier is a unary function in the current module.
func findIdent(x *scope, loc ast.Node, mod *ast.ModTag, name string) (id interface{}, err *checkError) {
	defer func() {
		switch id := id.(type) {
		case *Var:
			if id.Val != nil {
				use(x, id.Val, loc)
			}
		case *Fun:
			use(x, id, loc)
		}
	}()
	if mod != nil {
		imp, err := findImport(x, mod)
		if err != nil {
			return nil, err
		}
		if id := imp.findIdent(name); id != nil {
			return id, nil
		}
		return nil, x.err(loc, "identifier %s %s not found", mod.Text, name)
	}
	id, err = x.findIdent(loc, name)
	if err != nil {
		return nil, err
	}
	if id == nil {
		return nil, x.err(loc, "identifier %s not found", name)
	}
	return id, nil
}

// findIdent returns a *Var or *Fun.
// In the *Fun case, the identifier is a unary function in the current module.
func (x *scope) findIdent(loc ast.Node, name string) (interface{}, *checkError) {
	switch {
	case x == nil:
		return nil, nil
	case x.variable != nil && x.variable.Name == name:
		return x.variable, nil
	case x.fun != nil && x.fun.Recv != nil && x.fun.Recv.Type != nil:
		for i := range x.fun.Recv.Type.Fields {
			if f := &x.fun.Recv.Type.Fields[i]; f.Name == name {
				return f, nil
			}
		}
	case x.file != nil:
		switch f, err := x.file.findIdent(loc, name); {
		case err != nil:
			return nil, err
		case f != nil:
			return f, nil
		}
	case x.mod != nil:
		if fun := findIdentInDefs(name, x.mod.Defs); fun != nil {
			return fun, nil
		}
	case x.univ != nil:
		if fun := findIdentInDefs(name, x.univ); fun != nil {
			return fun, nil
		}
	}
	return x.up.findIdent(loc, name)
}

func (f *file) findIdent(loc ast.Node, name string) (interface{}, *checkError) {
	var ids []interface{}
	var imps []*imp
	for i := range f.imports {
		imp := &f.imports[i]
		if !imp.all {
			continue
		}
		if id := imp.findIdent(name); id != nil {
			imp.used = true
			ids = append(ids, id)
			imps = append(imps, imp)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) == 1 {
		return ids[0], nil
	}
	err := f.x.err(loc, "ambiguous identifier %s", name)
	for _, imp := range imps {
		note(err, "imported from %s at %s", imp.name, f.x.loc(imp.ast))
	}
	return nil, err
}

// findIdent returns either a *Fun that is a unary function or
// a *Var that is a module-level Val.Var.
func (imp *imp) findIdent(name string) interface{} {
	for _, def := range imp.defs {
		if def.priv() {
			continue
		}
		if fun, ok := def.(*Fun); ok && fun.Recv == nil && fun.Sig.Sel == name {
			return fun
		}
		if val, ok := def.(*Val); ok && val.Var.Name == name {
			return &val.Var
		}
	}
	return nil
}

// findIdentInDefs returns either a *Fun that is a unary function or
// a *Var that is a module-level Val.Var.
func findIdentInDefs(name string, defs []Def) interface{} {
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

func findFun(x *scope, loc ast.Node, recv *Type, mod *ast.ModTag, sel string) (fun *Fun, err *checkError) {
	defer func() {
		if fun != nil {
			use(x, fun, loc)
		}
	}()
	var modName string
	if mod != nil {
		imp, err := findImport(x, mod)
		if err != nil {
			return nil, err
		}
		if fun := imp.findFun(recv, sel); fun != nil {
			return fun, nil
		}
		modName = mod.Text + " "
	} else {
		fun, err := x.findFun(loc, recv, sel)
		if err != nil {
			return nil, err
		}
		if fun != nil {
			return fun, nil
		}
		if fun = findDefModMeth(x, recv, sel); fun != nil {
			return fun, nil
		}
	}
	if recv == nil {
		return nil, x.err(loc, "function %s%s not found", modName, sel)
	}
	return nil, x.err(loc, "method %s %s%s not found", recv, modName, sel)
}

func findDefModMeth(x *scope, recv *Type, sel string) *Fun {
	if recv == nil {
		return nil
	}
	imp := x.findImport("#" + recv.ModPath)
	if imp == nil {
		return nil
	}
	return imp.findFun(recv, sel)
}

func (x *scope) findFun(loc ast.Node, recv *Type, sel string) (*Fun, *checkError) {
	switch {
	case x == nil:
		return nil, nil
	case x.file != nil:
		fun, err := x.file.findFun(loc, recv, sel)
		if err != nil {
			return nil, err
		}
		if fun != nil {
			return fun, nil
		}
	case x.mod != nil:
		if f := findFunInDefs(recv, sel, x.mod.Defs); f != nil {
			return f, nil
		}
	case x.univ != nil:
		if f := findFunInDefs(recv, sel, x.univ); f != nil {
			return f, nil
		}
	}
	return x.up.findFun(loc, recv, sel)
}

func (f *file) findFun(loc ast.Node, recv *Type, sel string) (*Fun, *checkError) {
	var funs []*Fun
	var imps []*imp
	for i := range f.imports {
		imp := &f.imports[i]
		if !imp.all {
			continue
		}
		if fun := imp.findFun(recv, sel); fun != nil {
			imp.used = true
			funs = append(funs, fun)
			imps = append(imps, imp)
		}
	}
	if len(funs) == 0 {
		return nil, nil
	}
	if len(funs) == 1 {
		return funs[0], nil
	}
	var err *checkError
	if recv == nil {
		err = f.x.err(loc, "ambiguous function %s", sel)
	} else {
		err = f.x.err(loc, "ambiguous method %s", sel)
	}
	for _, imp := range imps {
		note(err, "imported from %s at %s", imp.name, f.x.loc(imp.ast))
	}
	return nil, err
}

func (imp *imp) findFun(recv *Type, sel string) *Fun {
	f := findFunInDefs(recv, sel, imp.defs)
	if f == nil || f.Priv {
		return nil
	}
	return f
}

func findFunInDefs(recv *Type, sel string, defs []Def) *Fun {
	for _, def := range defs {
		switch fun, ok := def.(*Fun); {
		case !ok:
			continue
		case fun.Recv != nil && fun.Recv.Type == nil:
			continue
		case (recv == nil) != (fun.Recv == nil):
			continue
		case fun.Sig.Sel != sel:
			continue
		case recv == nil:
			return fun
		case recv.Arity == fun.Recv.Type.Arity &&
			recv.Name == fun.Recv.Type.Name:
			return fun
		}
	}
	return nil
}

func markCapture(x *scope, vr *Var) (captured bool) {
	if vr.Val != nil || vr.Case != nil {
		// We don't need to capture module-level variables.
		// We should never be called with vr.Case != nil;
		// it is not an identifier.
		return false
	}
	for x != nil {
		if x.block == nil {
			x = x.up
			continue
		}
		if vr.BlkParm == x.block || vr.Local == &x.block.Locals {
			// We found the definiting block.
			return captured
		}
		addCapture(x.block, vr)
		captured = true
		x = x.up
	}
	return captured
}

func addCapture(b *Block, vr *Var) {
	for _, c := range b.Captures {
		if c == vr {
			return
		}
	}
	b.Captures = append(b.Captures, vr)
}
