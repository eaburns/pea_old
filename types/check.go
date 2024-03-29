// Copyright © 2020 The Pea Authors under an MIT-style license.

package types

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/eaburns/pea/ast"
)

// Config are configuration parameters for the type checker.
type Config struct {
	// IntSize is the bit size of the Int, UInt, and Word alias types.
	// It must be a valid int size: 8, 16, 32, or 64 (default=64).
	IntSize int
	// Importer is used for importing modules.
	// The default importer reads packages from the local file system.
	Importer Importer
	// Trace is whether to enable debug tracing.
	Trace bool
}

// Check type-checks an AST and returns the type-checked tree or errors.
func Check(astMod *ast.Mod, cfg Config) (*Mod, []error) {
	x := newUnivScope(newDefaultState(cfg, astMod))
	mod, errs := check(x, astMod)
	if len(errs) > 0 {
		return nil, convertErrors(errs)
	}
	return mod, nil
}

func check(x *scope, astMod *ast.Mod) (_ *Mod, errs []checkError) {
	defer x.tr("check(%s)", astMod.Path)(&errs)

	isUniv := x.univ == nil

	mod := &Mod{
		AST:  astMod,
		Path: astMod.Path,
	}

	// Checking happens in multiple passes.
	// Sorry, but some of the passes need more explanation
	// than is convenient to put in their function-call name.
	// Explanations below.

	// Make place-holder nodes for all module-level defs.
	// These have just enough information to lookup by name.
	// This also happens to pull in file-level imports,
	// since we need to hook up defs to their defining file anyway.
	mod.Defs, errs = makeDefs(x, mod, astMod.Files, isUniv)
	if isUniv {
		// In this case, we are checking the univ mod.
		// We've only now just gathered the defs, so set them in the state.
		x.univ = mod.Defs
	}

	// Check duplicates, except method duplicates.
	// We cannot check method duplicates yet,
	// because we have not yet resolved alias types;
	// we just know the names of things.
	errs = append(errs, checkDups(x, mod.Defs)...)

	// Constructs sem Tree nodes for all ast Nodes
	// of the definition "headers" — all but statements.
	// While doing this, we link TypeName.Type
	// to its definition and report any not-found errors.
	errs = append(errs, gatherDefs(x, mod.Defs)...)

	// Now that all definitions have all of their nodes
	// (except for Statement sub-trees),
	// we instantiate TypeName.Type for any parameterized type,
	// and resolve aliases to their target type.
	// After this pass, all TypeName.Types that are non-nil (non-errors)
	// will point to a Type with len(Args)==len(Parms),
	// with any type variables corresponding to a Param
	// substituted with the TypeName of the corresponding Arg.
	// Finally, TypeName.Type will never point to an alias.
	// If the TypeName names an alias, it's .Type will be the resolved type.
	errs = append(errs, instDefTypes(x, mod.Defs)...)

	mod.Defs = append(mod.Defs, builtInMeths(x, mod.Defs)...)
	if isUniv {
		// In this case, we are checking the univ mod.
		// Add the additional built-in defs to the state.
		x.univ = mod.Defs
	}

	// Now that we have resolved types and added any built-in methods,
	// we can report duplicate methods.
	errs = append(errs, checkDupMeths(x, mod.Defs)...)

	// This pass calls check on all the type names
	// in the def "header", reporting errors if the type cannot be instatiated
	// (because it's arguments do not satisfy the constraints).
	// It also finally builds the statement subtrees from the AST nodes.
	// Statement subtrees are built fully-linked:
	// 	to instantiated types,
	// 	identifiers pointing to their variables (tracking uses for init cycles),
	// 	calls pointing to their instantiated function/method.
	// Type conversions are added where needed, and
	// errors are reported for failure to gather and links statements
	// or for type mismatches.
	// Phew!
	errs = append(errs, checkDefs(x, mod.Defs)...)

	errs = append(errs, checkInitCycles(x, mod, mod.Defs)...)

	if len(errs) > 0 {
		return nil, errs
	}

	// At this point, all errors must have been checked.
	// If there were no errors, then there will be no errors
	// instantiating function bodies.
	// The key to this pass is to do method lookup
	// for substituted parameterized types.
	// All methods must exist, or we would have errored above.
	instFunBodies(x.state)

	rmLiftedFunInsts(mod.Defs)

	errs = append(errs, checkUnusedImports(x)...)

	mod.IntType = builtInType(x, "Int")
	mod.BoolType = builtInType(x, "Bool")
	mod.ByteType = builtInType(x, "UInt8")

	return mod, errs
}

func makeDefs(x *scope, mod *Mod, files []ast.File, isUniv bool) ([]Def, []checkError) {
	var defs []Def
	var errs []checkError
	for i := range files {
		file := &file{ast: &files[i]}
		x.files = append(x.files, file)

		fileX := x.new()
		fileX.file = file
		file.x = fileX.new()
		file.x.mod = mod

		errs = append(errs, imports(x, file)...)
		for _, astDef := range file.ast.Defs {
			def := makeDef(x, astDef, isUniv)
			defs = append(defs, def)
			x.defFiles[def] = file
		}
	}
	return defs, errs
}

func makeDef(x *scope, astDef ast.Def, isUniv bool) Def {
	switch astDef := astDef.(type) {
	case *ast.Val:
		val := &Val{
			AST:     astDef,
			ModPath: x.astMod.Path,
			Priv:    astDef.Priv(),
			Var: Var{
				AST:  &astDef.Var,
				Name: astDef.Var.Name,
			},
		}
		val.Var.Val = val
		return val
	case *ast.Fun:
		fun := &Fun{
			AST:     astDef,
			ModPath: x.astMod.Path,
			Priv:    astDef.Priv(),
			Test:    astDef.Test,
			Sig: FunSig{
				AST: &astDef.Sig,
				Sel: astDef.Sig.Sel,
			},
		}
		fun.Def = fun
		if isUniv {
			fun.BuiltIn = builtInFunTag[fun.Sig.Sel]
			if fun.BuiltIn == 0 {
				panic("impossible: " + fun.Sig.Sel)
			}
		}
		return fun
	case *ast.Type:
		typ := &Type{
			AST:     astDef,
			ModPath: x.astMod.Path,
			Priv:    astDef.Priv(),
			Arity:   len(astDef.Sig.Parms),
			Name:    astDef.Sig.Name,
		}
		typ.Def = typ
		if isUniv && astDef.Alias == nil {
			typ.BuiltIn = builtInTypeTag[typ.Name]
			if typ.BuiltIn == 0 {
				panic("impossible: " + typ.Name)
			}
		}
		return typ
	default:
		panic(fmt.Sprintf("impossible type %T", astDef))
	}
}

func imports(x *scope, file *file) []checkError {
	var errs []checkError
	for i := range file.ast.Imports {
		astImp := &file.ast.Imports[i]
		p := astImp.Path[1 : len(astImp.Path)-1] // trim "
		x.log("importing %s", p)
		defs, err := x.cfg.Importer.Import(x.cfg, x.astMod.Locs, p)
		if err != nil {
			errs = append(errs, *x.err(astImp, err.Error()))
			continue
		}
		file.imports = append(file.imports, imp{
			ast:  astImp,
			all:  astImp.All,
			path: p,
			name: modName(p),
			defs: defs,
		})
	}
	return errs
}

// checkDups returns redefinition errors for types, vals, and funs.
// It doesn't check duplicate methods.
func checkDups(x *scope, defs []Def) (errs []checkError) {
	defer x.tr("checkDups")(&errs)

	seen := make(map[string]Def)
	seenTypes := make(map[string]*Type)
	for _, def := range defs {
		var id string
		switch def := def.(type) {
		case *Val:
			id = def.Var.Name
		case *Type:
			id = def.Name
			tid := fmt.Sprintf("(%d)%s", def.Arity, def.Name)
			if prev, ok := seenTypes[tid]; ok {
				err := x.err(def, "type %s redefined", tid)
				note(err, "previous definition is at %s", x.loc(prev))
				errs = append(errs, *err)
				continue
			}
			seenTypes[tid] = def
			if prev, ok := seen[id]; ok {
				if _, ok := prev.(*Type); ok {
					// Multiple defs of the same type name are OK
					// as long as their arity is different.
					continue
				}
			}
		case *Fun:
			// Defer checking duplicate methods until receiver types are resolved.
			if astFun, ok := def.AST.(*ast.Fun); ok && astFun.Recv != nil {
				continue
			}
			id = def.Sig.Sel
		default:
			panic(fmt.Sprintf("impossible type %T", def))
		}
		if prev, ok := seen[id]; ok {
			err := x.err(def, "%s redefined", id)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
			continue
		}
		x.log("id=%s", id)
		seen[id] = def
	}
	return errs
}

func checkDupMeths(x *scope, defs []Def) []checkError {
	var errs []checkError
	seen := make(map[string]Def)
	for _, def := range defs {
		fun, ok := def.(*Fun)
		if !ok || fun.Recv == nil || fun.Recv.Type == nil {
			continue
		}
		recv := fun.Recv.Type
		key := recv.name() + " " + fun.Sig.Sel
		if prev, ok := seen[key]; ok {
			err := x.err(def, "method %s %s redefined", recv, fun.Sig.Sel)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[key] = def
		}
	}
	return errs
}

func checkInitCycles(x *scope, mod *Mod, defs []Def) (errs []checkError) {
	defer x.tr("checkInitCycles()")(&errs)

	seen := make(map[Def]bool)
	onPath := make(map[*Val]bool)
	var path []witness

	var check func(Def)
	check = func(def Def) {
		defer x.tr("checkInitCycles(%s)", def)(&errs)
		val, isVal := def.(*Val)
		if isVal && onPath[val] {
			err := x.err(val, "initialization cycle")
			for i := len(path) - 1; i >= 0; i-- {
				var next string
				if i == 0 {
					next = val.Var.Name
				} else {
					next = funOrValName(path[i-1].def)
				}
				cur := funOrValName(path[i].def)
				note(err, "%s: %s uses %s", x.loc(path[i].loc), cur, next)
			}
			errs = append(errs, *err)
			return
		}
		if seen[def] {
			return
		}
		seen[def] = true
		if isVal {
			onPath[val] = true
			defer func() { onPath[val] = false }()
		}
		for _, w := range x.initDeps[def] {
			path = append(path, w)
			check(w.def)
			path = path[:len(path)-1]
		}
		if isVal {
			mod.SortedVals = append(mod.SortedVals, val)
		}
	}
	for _, def := range defs {
		if val, ok := def.(*Val); ok && !seen[def] {
			check(val)
		}
	}
	sorted := mod.SortedVals
	n := len(sorted)
	for i := 0; i < n/2; i++ {
		sorted[i], sorted[n-i-1] = sorted[n-i-1], sorted[i]
	}
	return errs
}

func funOrValName(def Def) string {
	switch d := def.(type) {
	case *Val:
		return d.Var.Name
	case *Fun:
		return d.Sig.Sel
	default:
		panic("impossible")
	}
}

func checkUnusedImports(x *scope) (errs []checkError) {
	defer x.tr("checkUnusedImports()")(&errs)

	for _, file := range x.files {
		for i := range file.imports {
			imp := &file.imports[i]
			if imp.used {
				continue
			}
			err := x.err(imp.ast, "%s imported and not used", imp.path)
			errs = append(errs, *err)
		}
	}
	return errs
}

func checkDefs(x *scope, defs []Def) []checkError {
	var errs []checkError
	for _, def := range defs {
		errs = append(errs, checkDef(x, def)...)
	}
	return errs
}

func checkDef(x *scope, def Def) []checkError {
	if !x.gathered[def] {
		// This is a built-in method, with no AST and nothing to check.
		return nil
	}
	if x.checked[def] {
		return nil
	}
	x.checked[def] = true
	file, ok := x.defFiles[def]
	if !ok {
		panic("impossible")
	}
	x = file.x.new()
	x.def = def

	switch def := def.(type) {
	case *Val:
		return checkVal(x, def)
	case *Fun:
		return checkFun(x, def)
	case *Type:
		return checkType(x, def)
	default:
		panic(fmt.Sprintf("impossible type: %T", def))
	}
}

func checkVal(x *scope, def *Val) (errs []checkError) {
	defer x.tr("checkVal(%s)", def.name())(&errs)
	if def.Var.TypeName != nil {
		errs = append(errs, checkTypeName(x, def.Var.TypeName)...)
		def.Var.typ = def.Var.TypeName.Type
	}

	x = x.new()
	x.val = def

	var es []checkError
	def.Init, es = checkStmts(x, def.Var.Type(), def.AST.Init)

	if def.Var.Type() == nil {
		def.Var.typ = builtInType(x, "Nil")
		if len(def.Init) > 0 {
			if expr, ok := def.Init[len(def.Init)-1].(Expr); ok {
				def.Var.typ = expr.Type()
			}
		}
	}

	return append(errs, es...)
}

func checkFun(x *scope, def *Fun) (errs []checkError) {
	defer x.tr("checkFun(%s)", def.name())(&errs)

	if (def.Recv == nil || len(def.Recv.Parms) == 0) && len(def.TParms) == 0 {
		def.Insts = []*Fun{def}
	}

	x, errs = checkRecv(x, def.Recv)

	for i := range def.TParms {
		parm := &def.TParms[i]
		for j := range parm.Ifaces {
			iface := &parm.Ifaces[j]
			errs = append(errs, checkTypeName(x, iface)...)
		}
		x = x.new()
		x.typeVar = parm.Type
	}

	x = x.new()
	x.fun = def
	for i := range def.Sig.Parms {
		parm := &def.Sig.Parms[i]
		errs = append(errs, checkTypeName(x, parm.TypeName)...)
		x = x.new()
		x.variable = parm
	}

	var es []checkError
	def.Stmts, es = checkStmts(x, nil, def.AST.(*ast.Fun).Stmts)
	errs = append(errs, es...)
	if stmts := def.AST.(*ast.Fun).Stmts; len(stmts) == 0 && stmts != nil {
		def.Stmts = []Stmt{}
	}

	errs = append(errs, checkFunRet(x, def)...)

	if def.Recv != nil {
		for i := range def.Recv.Parms {
			tvar := &def.Recv.Parms[i]
			if tvar.Name != "_" && !x.tvarUse[tvar] {
				err := x.err(tvar, "%s defined and not used", tvar.Name)
				errs = append(errs, *err)
			}
		}
	}
	for i := range def.TParms {
		tvar := &def.TParms[i]
		if !x.tvarUse[tvar] {
			err := x.err(tvar, "%s defined and not used", tvar.Name)
			errs = append(errs, *err)
		}
	}
	return errs
}

func checkFunRet(x *scope, fun *Fun) []checkError {
	if fun.Sig.Ret == nil {
		addNilRet(x, fun)
		return nil
	}
	errs := checkTypeName(x, fun.Sig.Ret)
	fun.Sig.typ = fun.Sig.Ret.Type
	if fun.Stmts == nil {
		return errs
	}
	if n := len(fun.Stmts); n == 0 || !isRet(fun.Stmts[n-1]) {
		err := x.err(fun, "missing return at the end of %s", fun.Sig.Sel)
		errs = append(errs, *err)
	}
	return errs
}

func addNilRet(x *scope, fun *Fun) {
	fun.Sig.typ = builtInType(x, "Nil")
	if fun.Stmts == nil {
		return
	}
	nilRef := builtInType(x, "&", *makeTypeName(builtInType(x, "Nil")))
	fun.Stmts = append(fun.Stmts, &Ret{Expr: deref(&Ctor{typ: nilRef})})
}

func checkRecv(x *scope, recv *Recv) (_ *scope, errs []checkError) {
	if recv == nil {
		return x, nil
	}

	defer x.tr("checkRecv(%s)", recv.Type)(&errs)

	for i := range recv.Parms {
		parm := &recv.Parms[i]
		for j := range parm.Ifaces {
			iface := &parm.Ifaces[j]
			errs = append(errs, checkTypeName(x, iface)...)
		}
		x = x.new()
		x.typeVar = parm.Type
	}
	if isRef(recv.Type) {
		err := x.err(recv, "invalid receiver type: cannot add a method to &")
		errs = append(errs, *err)
	}
	return x, errs
}

func isRet(s Stmt) bool {
	_, ok := s.(*Ret)
	return ok
}

func checkType(x *scope, def *Type) (errs []checkError) {
	defer x.tr("checkType(%s)", def)(&errs)

	if len(def.Parms) == 0 {
		def.Insts = []*Type{def}
	}

	for i := range def.Parms {
		for j := range def.Parms[i].Ifaces {
			iface := &def.Parms[i].Ifaces[j]
			errs = append(errs, checkTypeName(x, iface)...)
		}
	}

	var es []checkError
	switch {
	case def.Alias != nil:
		es = checkTypeName(x, def.Alias)
	case def.Fields != nil:
		es = checkFields(x, def.Fields)
	case def.Cases != nil:
		es = checkCases(x, def.Cases)
	case def.Virts != nil:
		es = checkVirts(x, def.Virts)
	}
	return append(errs, es...)
}

func checkFields(x *scope, fields []Var) []checkError {
	var errs []checkError
	seen := make(map[string]*Var)
	for i := range fields {
		field := &fields[i]
		if prev, ok := seen[field.Name]; ok {
			err := x.err(field, "field %s redefined", field.Name)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[field.Name] = field
		}
		errs = append(errs, checkTypeName(x, field.TypeName)...)
	}
	return errs
}

func checkCases(x *scope, cases []Var) []checkError {
	var errs []checkError
	seen := make(map[string]*Var)
	for i := range cases {
		cas := &cases[i]
		lower := strings.ToLower(cas.Name)
		if prev, ok := seen[lower]; ok {
			err := x.err(cas, "case %s redefined", prev.Name)
			if prev.Name != cas.Name {
				note(err, "cases cannot differ in only capitalization")
			}
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[lower] = cas
		}
		if cas.TypeName != nil {
			errs = append(errs, checkTypeName(x, cas.TypeName)...)
		}
	}
	return errs
}

func checkVirts(x *scope, virts []FunSig) []checkError {
	var errs []checkError
	seen := make(map[string]*FunSig)
	for i := range virts {
		virt := &virts[i]
		if prev, ok := seen[virt.Sel]; ok {
			err := x.err(virt, "virtual method %s redefined", virt.Sel)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[virt.Sel] = virt
		}
		for i := range virt.Parms {
			parm := &virt.Parms[i]
			errs = append(errs, checkTypeName(x, parm.TypeName)...)
		}
		if virt.Ret != nil {
			errs = append(errs, checkTypeName(x, virt.Ret)...)
		}
	}
	return errs
}

func checkTypeName(x *scope, name *TypeName) (errs []checkError) {
	defer x.tr("checkTypeName(%s)", name)(&errs)

	errs = instTypeName(x, name)
	if name.Type == nil {
		return nil
	}

	for i := range name.Type.Args {
		arg := &name.Type.Args[i]
		parm := &name.Type.Parms[i]
		if arg.Type == nil {
			continue
		}
		for _, iface := range parm.Ifaces {
			if iface.Type == nil {
				continue
			}
			_, es := findVirts(x, arg.AST, arg.Type, iface.Type.Virts, true)
			if len(es) == 0 {
				continue
			}
			err := x.err(arg, "type %s does not implement %s (%s)", arg.Type, parm.Type, iface)
			err.cause = es
			errs = append(errs, *err)
		}
	}
	return errs
}

func checkStmts(x *scope, want *Type, astStmts []ast.Stmt) (_ []Stmt, errs []checkError) {
	defer x.tr("gatherStmts(want=%s)", want)(&errs)

	switch {
	case astStmts == nil:
		return nil, nil
	case len(astStmts) == 0:
		return []Stmt{nilLiteral(x)}, nil
	}

	var stmts []Stmt
	for i, astStmt := range astStmts {
		switch astStmt := astStmt.(type) {
		case *ast.Ret:
			ret, es := checkRet(x, astStmt)
			errs = append(errs, es...)
			stmts = append(stmts, ret)
		case *ast.Assign:
			var ss []Stmt
			var es []checkError
			x, ss, es = checkAssign(x, astStmt)
			errs = append(errs, es...)
			stmts = append(stmts, ss...)
		case ast.Expr:
			var expr Expr
			var es []checkError
			if i < len(astStmts)-1 {
				expr, es = checkExpr(x, nil, astStmt)
				errs = append(errs, es...)
				stmts = append(stmts, expr)
				continue
			}

			// This is the trailing, result expression of a Block or Val.
			// We need to handle a Nil want-type specially.
			// If we want a Nil, and the expression is not Nil convertable,
			// we insert a trailing {} constructor.
			// Otherwise just convert the type as normal.
			expr, es = _checkExpr(x, want, astStmt)
			errs = append(errs, es...)
			if isNil(want) && !isNilConvertable(x, expr.Type()) {
				stmts = append(stmts, expr)
				stmts = append(stmts, nilLiteral(x))
				continue
			}
			expr, err := convertExpr(x, want, expr)
			if err != nil {
				errs = append(errs, *err)
			}
			stmts = append(stmts, expr)
		default:
			panic(fmt.Sprintf("impossible type: %T", astStmt))
		}
	}
	errs = append(errs, checkUnusedLocals(x)...)
	return stmts, errs
}

func checkUnusedLocals(x *scope) []checkError {
	var errs []checkError
	for _, loc := range *x.locals() {
		if loc.Name != "_" && loc.AST != nil && !x.localUse[loc] {
			err := x.err(loc, "%s declared and not used", loc.Name)
			errs = append(errs, *err)
		}
	}
	return errs
}

func isNilConvertable(x *scope, typ *Type) bool {
	_, base := refBaseType(x, typ)
	return isNil(base)
}

func nilLiteral(x *scope) *Convert {
	nilType := builtInType(x, "Nil")
	return &Convert{
		Expr: &Ctor{
			typ: builtInType(x, "&", *makeTypeName(nilType)),
		},
		Ref: -1,
		typ: nilType,
	}
}

func checkRet(x *scope, astRet *ast.Ret) (_ *Ret, errs []checkError) {
	defer x.tr("checkRet(…)")(&errs)

	var want *Type
	if fun := x.function(); fun == nil {
		err := x.err(astRet, "return outside of a function or method")
		errs = append(errs, *err)
	} else if fun.Sig.Ret != nil {
		want = fun.Sig.Ret.Type
	} else {
		want = builtInType(x, "Nil")
	}
	expr, es := checkExpr(x, want, astRet.Expr)
	return &Ret{AST: astRet, Expr: expr}, append(errs, es...)
}

func checkAssign(x *scope, astAss *ast.Assign) (_ *scope, _ []Stmt, errs []checkError) {
	defer x.tr("checkAssign(…)")(&errs)

	x, vars, newLocal, errs := checkAssignVars(x, astAss)

	if len(vars) == 1 {
		var es []checkError
		assign := &Assign{AST: astAss, Var: vars[0]}
		assign.Expr, es = checkExpr(x, vars[0].Type(), astAss.Expr)
		if newLocal[0] && vars[0].TypeName == nil {
			vars[0].typ = assign.Expr.Type()
		}
		errs = append(errs, es...)
		return x, []Stmt{assign}, errs
	}

	var stmts []Stmt
	astCall, ok := astAss.Expr.(*ast.Call)
	if !ok || len(astCall.Msgs) != len(vars) {
		got := 1
		if ok {
			got = len(astCall.Msgs)
		}
		err := x.err(astAss, "assignment count mismatch: got %d, want %d", got, len(vars))
		errs = append(errs, *err)
		expr, es := checkExpr(x, nil, astAss.Expr)
		errs = append(errs, es...)
		stmts = append(stmts, &Assign{
			AST:  astAss,
			Var:  vars[0],
			Expr: expr,
		})
		for i := 1; i < len(vars); i++ {
			stmts = append(stmts, &Assign{
				AST:  astAss,
				Var:  vars[i],
				Expr: nil,
			})
		}
		return x, stmts, errs
	}

	recv, es := checkExpr(x, nil, astCall.Recv)
	recvType := recv.Type()
	errs = append(errs, es...)
	loc := x.locals()
	tmp := &Var{
		Name:  x.newID(),
		Local: loc,
		Index: len(*loc),
		typ:   recvType,
	}
	*loc = append(*loc, tmp)
	x = x.new()
	x.variable = tmp
	stmts = append(stmts, &Assign{Var: tmp, Expr: recv})
	for i := range vars {
		var infer *Type
		if vars[i].TypeName != nil {
			infer = vars[i].TypeName.Type
		}
		msg, es := checkMsg(x, infer, recvType, &astCall.Msgs[i])
		errs = append(errs, es...)
		call := &Call{
			AST: astCall,
			Recv: &Ident{
				Text: tmp.Name,
				Var:  tmp,
				typ:  builtInType(x, "&", *makeTypeName(recvType)),
			},
			Msgs: []Msg{msg},
		}
		if newLocal[i] && vars[i].TypeName == nil {
			vars[i].typ = call.Type()
		}
		stmts = append(stmts, &Assign{AST: astAss, Var: vars[i], Expr: call})
	}
	return x, stmts, errs
}

func checkAssignVars(x *scope, astAss *ast.Assign) (*scope, []*Var, []bool, []checkError) {
	var errs []checkError
	vars := make([]*Var, len(astAss.Vars))
	newLocal := make([]bool, len(astAss.Vars))
	for i := range astAss.Vars {
		astVar := &astAss.Vars[i]

		var typ *Type
		var typName *TypeName
		if astVar.Type != nil {
			var es []checkError
			typName, es = gatherTypeName(x, astVar.Type)
			errs = append(errs, es...)
			errs = append(errs, instTypeName(x, typName)...)
			typ = typName.Type
		}

		var found interface{}
		// If the Type is specified, this is always a new definition;
		// there is nothing to find.
		if astVar.Type == nil {
			var err *checkError
			// We call scope.findIdent here, since there cannot be a mod tag.
			// Also we do not want an error on not-found, scope.findIdent
			// doesn't error in this case, but findIdent does.
			if found, err = x.findIdent(astVar, astVar.Name); err != nil {
				errs = append(errs, *err)
			}
		}
		switch found := found.(type) {
		case nil:
			x.log("adding local %s", astVar.Name)
			loc := x.locals()
			vr := &Var{
				AST:      astVar,
				Name:     astVar.Name,
				TypeName: typName,
				Local:    loc,
				Index:    len(*loc),
				typ:      typ,
			}
			*loc = append(*loc, vr)
			x = x.new()
			x.variable = vr
			vars[i] = vr
			newLocal[i] = true
		case *Var:
			x.log("found var %s", found.Name)
			if found.Val != nil {
				x.use(found.Val, astAss)
			}
			markCapture(x, found)
			if astVar.Type != nil {
				err := x.err(astVar, "%s redefined", astVar.Name)
				note(err, "previous definition at %s", x.loc(found))
				errs = append(errs, *err)
			}
			vars[i] = found
		case *Fun:
			err := x.err(astVar, "assignment to a function")
			note(err, "%s is defined at %s", found.Sig.Sel, x.loc(found))
			errs = append(errs, *err)
			vars[i] = &Var{
				AST:      astVar,
				Name:     astVar.Name,
				TypeName: typName,
				typ:      typ,
			}
		default:
			panic(fmt.Sprintf("impossible type: %T", found))
		}
	}
	return x, vars, newLocal, errs
}

func checkExprs(x *scope, astExprs []ast.Expr) ([]Expr, []checkError) {
	var errs []checkError
	exprs := make([]Expr, len(astExprs))
	for i, expr := range astExprs {
		var es []checkError
		exprs[i], es = checkExpr(x, nil, expr)
		errs = append(errs, es...)
	}
	return exprs, errs
}

func checkExpr(x *scope, infer *Type, astExpr ast.Expr) (expr Expr, errs []checkError) {
	defer x.tr("checkExpr(infer=%s)", infer)(&errs)

	expr, errs = _checkExpr(x, infer, astExpr)
	if len(errs) > 0 {
		return expr, errs
	}
	expr, err := convertExpr(x, infer, expr)
	if err != nil {
		errs = append(errs, *err)
	}
	return expr, errs
}

func convertExpr(x *scope, want *Type, expr Expr) (_ Expr, err *checkError) {
	defer x.tr("convertExpr(want=%s, have=%s)", want, expr.Type())(&err)

	if expr.Type() == nil || want == nil || isPanic(expr) {
		return expr, nil
	}

	have := expr.Type()
	x.log("have %s (%p)", have, have)
	x.log("want %s (%p)", want, want)
	if cvt, ok := expr.(*Convert); ok && cvt.Ref != 0 {
		// We will recompute any reference conversions here,
		// so strip any incoming ones.
		expr = cvt.Expr
	}
	haveI, haveBase := refBaseType(x, expr.Type())
	wantI, wantBase := refBaseType(x, want)
	x.log("have base %s (%p)", haveBase, haveBase)
	x.log("want base %s (%p)", wantBase, wantBase)
	if haveBase == wantBase {
		for haveI > wantI {
			expr = deref(expr)
			haveI--
		}
		for haveI < wantI {
			expr = ref(x, expr)
			haveI++
		}
		return expr, nil
	}

	if len(want.Virts) > 0 {
		funs, es := findVirts(x, expr.ast(), haveBase, wantBase.Virts, false)
		if len(es) > 0 {
			err = x.err(expr.ast(), "type %s does not implement %s", have, want)
			err.cause = es
			return expr, err
		}
		if haveI != 0 {
			expr = deref(expr)
		}
		return &Convert{Expr: expr, Virts: funs, typ: want}, nil
	}

	err = x.err(expr, "type mismatch: have %s, want %s", have, want)
	if have.Var != nil && want.Var != nil && have.Name == want.Name {
		if have.AST != nil {
			note(err, "have type %s defined at %s", have, x.loc(have))
		} else {
			note(err, "have type %s is from a built-in definiton", have)
		}
		if want.AST != nil {
			note(err, "want type %s defined at %s", want, x.loc(want))
		} else {
			note(err, "want type %s is from a built-in definiton", want)
		}
	}
	return expr, err
}

func refBaseType(x *scope, typ *Type) (int, *Type) {
	var i int
	for isRef(typ) {
		i++
		typ = typ.Args[0].Type
	}
	return i, typ
}

func ref(x *scope, expr Expr) Expr {
	if cvt, ok := expr.(*Convert); ok && cvt.Ref == -1 {
		return cvt.Expr
	}
	var typ *Type
	if t := expr.Type(); t != nil {
		typ = builtInType(x, "&", *makeTypeName(t))
	}
	return &Convert{Expr: expr, Ref: 1, typ: typ}
}

func deref(expr Expr) Expr {
	if cvt, ok := expr.(*Convert); ok && cvt.Ref == 1 {
		return cvt.Expr
	}
	typ := expr.Type().Args[0].Type
	return &Convert{Expr: expr, Ref: -1, typ: typ}
}

func findVirts(x *scope, loc ast.Node, recv *Type, virts []FunSig, allowConvert bool) (funs []*Fun, errs []checkError) {
	defer x.tr("findVirts(%s, allowConvert=%v)", recv, allowConvert)(&errs)

	funs = make([]*Fun, len(virts))
	for i := range virts {
		fun, es := findVirt(x, loc, recv, &virts[i], allowConvert)
		if len(es) > 0 {
			errs = append(errs, es...)
		} else {
			funs[i] = fun
		}
	}
	return funs, errs
}

func findVirt(x *scope, loc ast.Node, recv *Type, want *FunSig, allowConvert bool) (fun *Fun, errs []checkError) {
	defer x.tr("findVirt(%s, %s, allowConvert=%v)", recv, want.Sel, allowConvert)(&errs)

	var ret *Type
	if want.Ret != nil {
		ret = want.Ret.Type
	}
	argTypes := funSigArgTypes{loc: loc, sig: want}
	fun, errs = findFunInst(x, loc, ret, recv, nil, want.Sel, argTypes)
	if len(errs) > 0 {
		return nil, errs
	}

	// Make a copy and remove the self parameter.
	funSig := fun.Sig
	funSig.Parms = funSig.Parms[1:]

	switch {
	case !allowConvert && funSigEq(&funSig, want):
		fallthrough
	case allowConvert && funSigConvert(x, &funSig, want):
		return fun, nil
	}

	// Clear the parameter names for printing the error note.
	for i := range funSig.Parms {
		funSig.Parms[i].Name = ""
	}
	var gotWhere string
	if w := x.loc(fun.AST); w != nil {
		gotWhere = fmt.Sprintf(" from %s", w)
	}
	var wantWhere string
	if w := x.loc(want.AST); w != nil {
		wantWhere = fmt.Sprintf(" from %s", w)
	}
	err := x.err(loc, "wrong type for method %s", want.Sel)
	err.notes = []string{
		fmt.Sprintf("have %s%s", funSig, gotWhere),
		fmt.Sprintf("want %s%s", want, wantWhere),
	}
	return nil, append(errs, *err)
}

func funSigEq(a, b *FunSig) bool {
	if a.Sel != b.Sel || len(a.Parms) != len(b.Parms) || (a.Ret == nil) != (b.Ret == nil) {
		return false
	}
	for i := range a.Parms {
		if a.Parms[i].Type() != b.Parms[i].Type() {
			return false
		}
	}
	return a.Ret == nil || a.Ret.Type == b.Ret.Type
}

func funSigConvert(x *scope, a, b *FunSig) bool {
	if a.Sel != b.Sel || len(a.Parms) != len(b.Parms) || (a.Ret == nil) != (b.Ret == nil) {
		return false
	}
	for i := range a.Parms {
		_, aTyp := refBaseType(x, a.Parms[i].Type())
		_, bTyp := refBaseType(x, b.Parms[i].Type())
		if aTyp != bTyp {
			return false
		}
	}
	if a.Ret == nil {
		return true
	}
	_, aRet := refBaseType(x, a.Ret.Type)
	_, bRet := refBaseType(x, b.Ret.Type)
	return aRet == bRet
}

func _checkExpr(x *scope, infer *Type, astExpr ast.Expr) (Expr, []checkError) {
	switch astExpr := astExpr.(type) {
	case *ast.Call:
		return checkCall(x, infer, astExpr)
	case *ast.Ctor:
		return checkCtor(x, infer, astExpr)
	case *ast.Block:
		return checkBlock(x, infer, astExpr)
	case *ast.Ident:
		return checkIdent(x, infer, astExpr)
	case *ast.Int:
		return checkInt(x, infer, astExpr, astExpr.Text)
	case *ast.Float:
		return checkFloat(x, infer, astExpr, astExpr.Text)
	case *ast.Rune:
		return checkRune(x, infer, astExpr)
	case *ast.String:
		return checkString(x, astExpr)
	default:
		panic(fmt.Sprintf("impossible type: %T", astExpr))
	}
}

func checkCall(x *scope, infer *Type, astCall *ast.Call) (_ *Call, errs []checkError) {
	defer x.tr("checkCall(infer=%s)", infer)(&errs)

	call := &Call{
		AST:  astCall,
		Msgs: make([]Msg, len(astCall.Msgs)),
	}

	var recv Expr
	var recvBaseType *Type
	if astCall.Recv != nil {
		switch recv, errs = checkExpr(x, nil, astCall.Recv); {
		case recv.Type() == nil:
			x.log("call receiver check error")
			// There was a receiver, but we don't know it's type.
			// That error was reported elsewhere, but we can't continue here.
			// Do best-effort checking of the message arguments.
			for i := range astCall.Msgs {
				astMsg := &astCall.Msgs[i]
				call.Msgs[i] = Msg{
					AST: astMsg,
					Mod: modString(astMsg.Mod),
					Sel: astMsg.Sel,
				}
				var es []checkError
				call.Msgs[i].Args, es = checkExprs(x, astMsg.Args)
				errs = append(errs, es...)
			}
			return call, errs
		case isRef(recv.Type()) && isRef(recv.Type().Args[0].Type):
			for isRef(recv.Type().Args[0].Type) {
				recv = deref(recv)
			}
		case !isRef(recv.Type()):
			recv = ref(x, recv)
		}
		if !isRef(recv.Type()) || isRef(recv.Type().Args[0].Type) {
			panic("impossible")
		}
		recvBaseType = recv.Type().Args[0].Type
		call.Recv = recv
	}
	for i := range astCall.Msgs {
		var es []checkError
		call.Msgs[i], es = checkMsg(x, infer, recvBaseType, &astCall.Msgs[i])
		errs = append(errs, es...)
	}
	return call, errs
}

func checkMsg(x *scope, infer, recv *Type, astMsg *ast.Msg) (_ Msg, errs []checkError) {
	defer x.tr("checkMsg(infer=%s, %s, %s)", infer, recv, astMsg.Sel)(&errs)

	msg := Msg{
		AST:  astMsg,
		Mod:  modString(astMsg.Mod),
		Sel:  astMsg.Sel,
		Args: make([]Expr, len(astMsg.Args)),
	}
	es := findMsgFun(x, infer, recv, &msg)
	errs = append(errs, es...)
	if msg.Fun == nil {
		// findMsgFun failed; best-effort check the arguments.
		msg.Args, es = checkExprs(x, astMsg.Args)
		return msg, append(errs, es...)
	}
	// We don't check whether msg.Fun is a test here,
	// because tests can currently only be unary functions.
	// Unary function calls are handled as a special case
	// in checkIdent.
	parms := msg.Fun.Sig.Parms
	if msg.Fun.Recv != nil {
		parms = parms[1:]
	}
	for i, astArg := range astMsg.Args {
		if msg.Args[i] != nil {
			// This arg was already checked
			// in order to inst fun type parameters.
			continue
		}
		var es []checkError
		typ := parms[i].Type()
		msg.Args[i], es = checkExpr(x, typ, astArg)
		errs = append(errs, es...)
	}

	if msg.Fun.Sig.Ret == nil {
		msg.typ = builtInType(x, "Nil")
	} else {
		msg.typ = msg.Fun.Sig.Ret.Type
	}

	return msg, errs
}

func findMsgFun(x *scope, infer, recv *Type, msg *Msg) (errs []checkError) {
	defer x.tr("findMsgFun(infer=%s, %s, %s)", infer, recv, msg.name())(&errs)

	var mod *ast.ModTag
	if msg.Mod != "" {
		switch astMsg := msg.AST.(type) {
		case *ast.Msg:
			mod = astMsg.Mod
		case *ast.Ident:
			mod = astMsg.Mod
		default:
			panic(fmt.Sprintf("impossible type: %T", msg.AST))
		}
	}
	msg.Fun, errs = findFunInst(x, msg.AST, infer, recv, mod, msg.Sel, msg)
	return errs
}

func findFunInst(x *scope, loc ast.Node, infer, recv *Type, mod *ast.ModTag, sel string, argTypes argTypes) (fun *Fun, errs []checkError) {
	defer x.tr("findFunInst(infer=%s, %s, %s)", infer, recv, sel)(&errs)

	if recv != nil && recv.Var != nil {
		if mod == nil {
			switch r, f, err := findIfaceMeth(x, loc, sel, recv.Var.Ifaces); {
			case err != nil:
				return nil, append(errs, *err)
			case f != nil:
				recv, fun = r, f
			}
		}
		if fun == nil {
			err := x.err(loc, "method %s %s not found", recv, sel)
			return nil, append(errs, *err)
		}
	} else {
		var err *checkError
		if fun, err = findFun(x, loc, recv, mod, sel); err != nil {
			return nil, append(errs, *err)
		}
	}
	x.log("found %s", fun)
	return instRecvAndFun(x, loc, recv, infer, fun, argTypes)
}

func findIfaceMeth(x *scope, loc ast.Node, sel string, ifaces []TypeName) (*Type, *Fun, *checkError) {
	for _, iface := range ifaces {
		if iface.Type == nil || !hasVirt(iface.Type, sel) {
			continue
		}
		switch fun, err := x.findFun(loc, iface.Type, sel); {
		case err != nil:
			return nil, nil, err
		case fun != nil:
			return iface.Type, fun, nil
		}
	}
	return nil, nil, nil
}

func hasVirt(typ *Type, sel string) bool {
	for _, virt := range typ.Virts {
		if virt.Sel == sel {
			return true
		}
	}
	return false
}

func checkCtor(x *scope, infer *Type, astCtor *ast.Ctor) (_ Expr, errs []checkError) {
	defer x.tr("checkCtor(infer=%s)", infer)(&errs)

	typ := infer
	ctor := &Ctor{AST: astCtor}
	switch {
	case typ == nil:
		errs = append(errs, *x.err(ctor, "cannot infer constructor type"))
	case typ.Alias != nil:
		panic("impossible alias")
	case typ.Priv && x.defFiles[typ.Def] == nil && !isBuiltInType(typ):
		errs = append(errs, *x.err(ctor, "cannot construct unexported type %s", typ))
	case isAry(typ):
		errs = append(errs, checkAryCtor(x, typ, ctor)...)
	case isRef(typ):
		nRef, baseType := refBaseType(x, typ)
		expr, es := checkCtor(x, baseType, astCtor)
		errs = append(errs, es...)
		// Start from 1, becasue checkCtor is already 1 reference.
		for i := 1; i < nRef; i++ {
			expr = ref(x, expr)
		}
		return expr, errs
	case typ.Cases != nil:
		errs = append(errs, checkOrCtor(x, typ, ctor)...)
	case typ.Virts != nil:
		errs = append(errs, *x.err(astCtor, "cannot construct virtual type %s", typ))
	case isBuiltInType(typ) && !isNil(typ):
		errs = append(errs, *x.err(astCtor, "cannot construct built-in type %s", typ))
	default:
		errs = append(errs, checkAndCtor(x, typ, ctor)...)
	}
	if typ != nil {
		ctor.typ = builtInType(x, "&", *makeTypeName(typ))
	}
	return ctor, errs
}

func checkAryCtor(x *scope, aryType *Type, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkAryCtor(%s)", aryType)(&errs)
	elmType := aryType.Args[0].Type
	ctor.Args = make([]Expr, len(ctor.AST.Args))
	for i, expr := range ctor.AST.Args {
		var es []checkError
		ctor.Args[i], es = checkExpr(x, elmType, expr)
		errs = append(errs, es...)
	}
	return errs
}

func checkOrCtor(x *scope, orType *Type, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkOrCtor(%s)", orType)(&errs)

	sel, arg, ok := disectOrCtorArg(ctor.AST)
	if !ok {
		err := x.err(ctor, "malformed %s constructor", orType)
		return append(errs, *err)
	}

	ctor.Case = findCase(orType, sel)
	if ctor.Case == nil {
		err := x.err(ctor, "case %s not found", sel)
		errs = append(errs, *err)
		var es []checkError
		ctor.Args, es = checkExprs(x, ctor.AST.Args)
		return append(errs, es...)
	}
	c := &orType.Cases[*ctor.Case]

	if c.TypeName == nil {
		if arg != nil {
			panic("impossible")
		}
		return errs
	}

	expr, es := checkExpr(x, c.Type(), arg)
	ctor.Args = []Expr{expr}
	return append(errs, es...)
}

func disectOrCtorArg(ctor *ast.Ctor) (string, ast.Expr, bool) {
	if len(ctor.Args) != 1 {
		return "", nil, false
	}
	if id, ok := ctor.Args[0].(*ast.Ident); ok {
		return id.Text, nil, true
	}
	call, ok := ctor.Args[0].(*ast.Call)
	if !ok || len(call.Msgs) != 1 || call.Msgs[0].Mod != nil || len(call.Msgs[0].Args) != 1 {
		return "", nil, false
	}
	return call.Msgs[0].Sel, call.Msgs[0].Args[0], true
}

func findCase(typ *Type, name string) *int {
	for i := range typ.Cases {
		if typ.Cases[i].Name == name {
			return &i
		}
	}
	return nil
}

func checkAndCtor(x *scope, andType *Type, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkAndCtor(%s)", andType)(&errs)

	if len(ctor.AST.Args) == 0 {
		return errs
	}
	call, ok := ctor.AST.Args[0].(*ast.Call)
	if !ok || len(ctor.AST.Args) > 1 || call.Recv != nil || len(call.Msgs) != 1 {
		err := x.err(ctor, "malformed %s constructor", andType)
		return append(errs, *err)
	}

	astArgs := make([]ast.Expr, len(andType.Fields))
	fieldNames := strings.Split(call.Msgs[0].Sel, ":")
	for i, astArg := range call.Msgs[0].Args {
		fieldName := fieldNames[i]
		field := findField(andType, fieldName)
		if field < 0 {
			err := x.err(astArg, "unknown field: %s", fieldName)
			errs = append(errs, *err)
			continue
		}
		if prev := astArgs[field]; prev != nil {
			err := x.err(astArg, "duplicate field: %s", fieldName)
			note(err, "previous at %s", x.loc(prev))
			errs = append(errs, *err)
			continue
		}
		astArgs[field] = astArg
	}

	ctor.Args = make([]Expr, len(andType.Fields))
	for i := range andType.Fields {
		field := &andType.Fields[i]
		if astArgs[i] == nil {
			err := x.err(ctor, "missing field: %s", field.Name)
			errs = append(errs, *err)
			continue
		}
		var es []checkError
		ctor.Args[i], es = checkExpr(x, field.Type(), astArgs[i])
		errs = append(errs, es...)
	}
	return errs
}

func findField(typ *Type, name string) int {
	for i := range typ.Fields {
		if typ.Fields[i].Name == name {
			return i
		}
	}
	return -1
}

func checkBlock(x *scope, infer *Type, astBlock *ast.Block) (_ *Block, errs []checkError) {
	defer x.tr("checkBlock(infer=%s)", infer)(&errs)

	var resInfer *Type
	parmInfer := make([]*Type, len(astBlock.Parms))
	if isFun(infer) {
		resInfer = infer.Args[len(infer.Args)-1].Type
		n := len(infer.Args)
		if n > len(astBlock.Parms) {
			n = len(astBlock.Parms)
		}
		for i := 0; i < n; i++ {
			parmInfer[i] = infer.Args[i].Type
		}
	}

	blk := &Block{
		AST:   astBlock,
		Parms: make([]Var, len(astBlock.Parms)),
	}

	for i := range astBlock.Parms {
		astParm := &astBlock.Parms[i]
		parm := &blk.Parms[i]
		parm.AST = astParm
		parm.Name = astParm.Name
		if astParm.Type == nil {
			if parmInfer[i] == nil {
				err := x.err(parm, "cannot infer block parameter type")
				errs = append(errs, *err)
			}
			parm.typ = parmInfer[i]
			continue
		}
		var es []checkError
		parm.TypeName, es = gatherTypeName(x, astParm.Type)
		errs = append(errs, es...)
		errs = append(errs, checkTypeName(x, parm.TypeName)...)
		parm.typ = parm.TypeName.Type
	}

	x = x.new()
	x.block = blk
	for i := range blk.Parms {
		parm := &blk.Parms[i]
		parm.BlkParm = blk
		parm.Index = i
		x = x.new()
		x.variable = parm
	}

	var es []checkError
	blk.Stmts, es = checkStmts(x, resInfer, astBlock.Stmts)
	errs = append(errs, es...)

	if len(blk.Parms) >= MaxValueParms {
		err := x.err(astBlock, "too many block parameters: got %d, max %d",
			len(astBlock.Parms), MaxValueParms)
		errs = append(errs, *err)
		return blk, errs
	}

	typeArgs := make([]TypeName, len(blk.Parms)+1)
	for i := range blk.Parms {
		parm := &blk.Parms[i]
		if parm.Type() == nil {
			// TODO: this is an error.
			// We cannot figure out the block param type?
			return blk, errs
		}
		if parm.TypeName != nil {
			typeArgs[i] = *parm.TypeName
			continue
		}
		parmTyp := parm.Type()
		typeArgs[i] = TypeName{
			AST:  &astBlock.Parms[i],
			Mod:  modName(parmTyp.ModPath),
			Name: parmTyp.Name,
			Args: parmTyp.Args,
			Type: parmTyp,
		}
	}

	var resType *Type
	// checkStmts inserts a {}, so len(blk.Stmts) is always >0.
	switch last := blk.Stmts[len(blk.Stmts)-1]; {
	case isRet(last) || isPanic(last):
		if resInfer == nil {
			resType = builtInType(x, "Nil")
		} else {
			resType = resInfer
		}

	case isExpr(last):
		expr, _ := last.(Expr)
		resType = expr.Type()
		if resType == nil {
			// TODO: this is an error.
			// We cannot figure out the block result type.
			// test:
			//	f Foo := xyz: [f].
			// we may not have checked the type of f yet
			// when checking the block statements.
			return blk, errs
		}

	default:
		resType = builtInType(x, "Nil")
		blk.Stmts = append(blk.Stmts, nilLiteral(x))
	}
	typeArgs[len(typeArgs)-1] = TypeName{
		AST:  astBlock,
		Mod:  modName(resType.ModPath),
		Name: resType.Name,
		Args: resType.Args,
		Type: resType,
	}
	blk.typ = builtInType(x, "Fun", typeArgs...)
	blk.BlockType = makeBlockType(x, blk)
	return blk, errs
}

func isExpr(stmt Stmt) bool {
	_, ok := stmt.(Expr)
	return ok
}

func isPanic(stmt Stmt) bool {
	call, ok := stmt.(*Call)
	if !ok || len(call.Msgs) == 0 {
		return false
	}
	msg := &call.Msgs[len(call.Msgs)-1]
	return msg.Fun != nil && msg.Fun.BuiltIn == PanicFunc
}

func checkIdent(x *scope, infer *Type, astIdent *ast.Ident) (_ Expr, errs []checkError) {
	defer x.tr("checkIdent(infer=%s, %s)", infer, astIdent.Text)(&errs)

	ident := &Ident{AST: astIdent, Text: astIdent.Text}

	found, err := findIdent(x, astIdent, astIdent.Mod, astIdent.Text)
	if err != nil {
		return ident, append(errs, *err)
	}
	switch vr := found.(type) {
	case *Var:
		ident.Var = vr
		switch {
		case vr.Val != nil:
			// Recursively check the Val to make sure it's type is inferred.
			errs = append(errs, checkDef(x, vr.Val)...)
		case vr.Local != nil:
			x.localUse[vr] = true
		}
		if vr.Type() == nil {
			return ident, errs
		}
		ident.Capture = markCapture(x, vr)
		// Idents are references to their underlying value.
		ident.typ = vr.Type().Ref()
		return deref(ident), errs
	case *Fun:
		defer x.tr("check ident msg(infer=%s, nil, %s)", infer, astIdent.Text)(&errs)
		call := &Call{
			AST: astIdent,
			Msgs: []Msg{{
				AST: astIdent,
				Mod: modString(astIdent.Mod),
				Sel: astIdent.Text,
			}},
		}
		msg := &call.Msgs[0]
		es := findMsgFun(x, infer, nil, msg)
		errs = append(errs, es...)
		switch {
		case msg.Fun == nil:
			return call, errs
		case msg.Fun.Test:
			err := x.err(astIdent, "tests cannot be called")
			errs = append(errs, *err)
			return call, errs
		case msg.Fun.Sig.Ret == nil:
			msg.typ = builtInType(x, "Nil")
		default:
			msg.typ = msg.Fun.Sig.Ret.Type
		}
		return call, errs
	default:
		panic(fmt.Sprintf("impossible type: %T", vr))
	}
}

func checkInt(x *scope, infer *Type, AST ast.Expr, text string) (_ Expr, errs []checkError) {
	defer x.tr("checkInt(infer=%s, %s)", infer, text)(&errs)

	n := &Int{AST: AST, Val: new(big.Int)}
	x.log("parsing int [%s]", text)
	if _, ok := n.Val.SetString(text, 0); !ok {
		panic("malformed int")
	}
	switch {
	case isAnyFloat(infer):
		return checkFloat(x, infer, AST, text)
	case isAnyInt(infer):
		n.typ = infer
	default:
		n.typ = builtInType(x, "Int")
	}
	if err := checkIntBounds(x, AST, n.typ, n.Val); err != nil {
		errs = append(errs, *err)
	}
	return n, errs
}

func checkIntBounds(x *scope, n interface{}, t *Type, i *big.Int) *checkError {
	signed, bits := disectIntType(x.cfg, t)
	x.log("signed=%v, bits=%v", signed, bits)
	if !signed && i.Cmp(&big.Int{}) < 0 {
		return x.err(n, "type %s cannot represent %s: negative unsigned", t, i)
	}
	min := big.NewInt(-(1 << uint(bits)))
	x.log("val=%v, val.BitLen()=%d, min=%v", i, i.BitLen(), min)
	if i.BitLen() > bits && (!signed || i.Cmp(min) != 0) {
		return x.err(n, "type %s cannot represent %s: overflow", t, i)
	}
	return nil
}

func checkFloat(x *scope, infer *Type, AST ast.Expr, text string) (_ Expr, errs []checkError) {
	defer x.tr("checkFloat(infer=%s, %s)", infer, text)(&errs)

	n := &Float{AST: AST, Val: new(big.Float)}
	if _, _, err := n.Val.Parse(text, 10); err != nil {
		panic("malformed float")
	}
	switch {
	case isAnyInt(infer):
		var i big.Int
		if _, acc := n.Val.Int(&i); acc != big.Exact {
			err := x.err(AST, "type %s cannot represent %s: truncation", infer.name(), text)
			errs = append(errs, *err)
		}
		expr, es := checkInt(x, infer, AST, i.String())
		return expr, append(errs, es...)
	case isAnyFloat(infer):
		n.typ = infer
	default:
		n.typ = builtInType(x, "Float")
	}
	return n, errs
}

func checkRune(x *scope, infer *Type, AST *ast.Rune) (_ Expr, errs []checkError) {
	defer x.tr("checkRune(infer=%s, %s)", infer, AST.Text)(&errs)
	n := &Int{
		AST: AST,
		Val: big.NewInt(int64(AST.Rune)),
	}
	switch {
	case isAnyFloat(infer):
		return checkFloat(x, infer, AST, n.Val.String())
	case isAnyInt(infer):
		n.typ = infer
	default:
		n.typ = builtInType(x, "Int32")
	}
	if err := checkIntBounds(x, AST, n.typ, n.Val); err != nil {
		errs = append(errs, *err)
	}
	return n, errs
}

func checkString(x *scope, astString *ast.String) (*String, []checkError) {
	defer x.tr("checkString(%s)", astString.Text)()
	return &String{
		AST:  astString,
		Data: astString.Data,
		typ:  builtInType(x, "String"),
	}, nil
}

func modString(m *ast.ModTag) string {
	if m == nil {
		return ""
	}
	return m.Text
}
