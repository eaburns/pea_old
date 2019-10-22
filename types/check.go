package types

import (
	"fmt"
	"math/big"
	"path"
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
	defer x.tr("check(%s)", astMod.Name)(&errs)

	isUniv := x.univ == nil

	mod := &Mod{AST: astMod}
	x = x.new()
	x.mod = mod

	mod.Defs, errs = makeDefs(x, astMod.Files)
	errs = append(errs, checkDups(x, mod.Defs)...)
	errs = append(errs, gatherDefs(x, mod.Defs)...)
	if isUniv {
		// In this case, we are checking the univ mod.
		// We've only now just gathered the defs, so set them in the state.
		x.up.univ = mod.Defs
	}
	mod.Defs = append(mod.Defs, builtInMeths(x, mod.Defs)...)
	if isUniv {
		// In this case, we are checking the univ mod.
		// Add the additional built-in defs to the state.
		x.up.univ = mod.Defs
	}
	errs = append(errs, checkDupMeths(x, mod.Defs)...)
	errs = append(errs, checkDefs(x, mod.Defs)...)

	return mod, errs
}

func makeDefs(x *scope, files []ast.File) ([]Def, []checkError) {
	var defs []Def
	var errs []checkError
	for i := range files {
		file := &file{ast: &files[i]}
		file.x = x.new()
		file.x.file = file
		errs = append(errs, imports(x.state, file)...)
		for _, astDef := range file.ast.Defs {
			def := makeDef(astDef)
			defs = append(defs, def)
			x.defFiles[def] = file
		}
	}
	return defs, errs
}

func imports(x *state, file *file) []checkError {
	var errs []checkError
	for _, astImp := range file.ast.Imports {
		p := astImp.Path[1 : len(astImp.Path)-1] // trim "
		x.log("importing %s", p)
		defs, err := x.cfg.Importer.Import(x.cfg, p)
		if err != nil {
			errs = append(errs, *x.err(astImp, err.Error()))
			continue
		}
		file.imports = append(file.imports, imp{
			path: p,
			name: path.Base(p),
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
	types := make(map[string]Def)
	for _, def := range defs {
		var id string
		switch def := def.(type) {
		case *Val:
			id = def.Var.Name
		case *Type:
			id = def.Name
			tid := fmt.Sprintf("(%d)%s", def.Arity, def.Name)
			if prev, ok := types[tid]; ok {
				err := x.err(def, "type %s redefined", tid)
				note(err, "previous definition is at %s", x.loc(prev))
				errs = append(errs, *err)
				continue
			}
			types[tid] = def
			if _, ok := seen[id].(*Type); ok {
				// Multiple defs of the same type name are OK
				// as long as their arity is different.
				continue
			}
		case *Fun:
			if astFun, ok := def.AST.(*ast.Fun); ok && astFun.Recv != nil {
				continue // check dup methods separately.
			}
			id = def.Sig.Sel
		default:
			panic(fmt.Sprintf("impossible type %T", def))
		}
		if prev, ok := seen[id]; ok {
			err := x.err(def, "%s redefined", id)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		}
		seen[id] = def
	}
	return errs
}

func makeDef(astDef ast.Def) Def {
	switch astDef := astDef.(type) {
	case *ast.Val:
		val := &Val{
			AST:  astDef,
			Priv: astDef.Priv(),
			Var: Var{
				AST:  &astDef.Var,
				Name: astDef.Var.Name,
			},
		}
		val.Var.Val = val
		return val
	case *ast.Fun:
		fun := &Fun{
			AST:  astDef,
			Priv: astDef.Priv(),
			Sig: FunSig{
				AST: &astDef.Sig,
				Sel: astDef.Sig.Sel,
			},
		}
		fun.Def = fun
		return fun
	case *ast.Type:
		typ := &Type{
			AST:   astDef,
			Priv:  astDef.Priv(),
			Arity: len(astDef.Sig.Parms),
			Name:  astDef.Sig.Name,
		}
		typ.Def = typ
		return typ
	default:
		panic(fmt.Sprintf("impossible type %T", astDef))
	}
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
			err := x.err(def, "method %s redefined", key)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[key] = def
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
	file, ok := x.defFiles[def]
	if !ok {
		panic("impossible")
	}
	x = file.x

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
	def.Init, es = checkStmts(x, def.Var.typ, def.AST.Init)

	if def.Var.typ == nil {
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
	if def.Recv != nil {
		for i := range def.Recv.Parms {
			parm := &def.Recv.Parms[i]
			for j := range parm.Ifaces {
				iface := &parm.Ifaces[j]
				errs = append(errs, checkTypeName(x, iface)...)
			}
			x = x.new()
			x.typeVar = parm.Type
		}
		if isRef(x, def.Recv.Type) {
			err := x.err(def.Recv, "invalid receiver type: cannot add a method to &")
			errs = append(errs, *err)
		}
	}
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

	if def.Sig.Ret != nil {
		errs = append(errs, checkTypeName(x, def.Sig.Ret)...)

		// TODO: check missing return for non-decl funcs with no statements.
		// We currently have no way to diferentiate a declaration and a function with no statements.
		if len(def.Stmts) > 0 && !isRet(def.Stmts[len(def.Stmts)-1]) {
			err := x.err(def, "missing return at the end of %s", def.name())
			errs = append(errs, *err)
		}
	}
	return errs
}

func isRet(s Stmt) bool {
	_, ok := s.(*Ret)
	return ok
}

func checkType(x *scope, def *Type) (errs []checkError) {
	for i := range def.Parms {
		for j := range def.Parms[i].Ifaces {
			iface := &def.Parms[i].Ifaces[j]
			errs = append(errs, checkTypeName(x, iface)...)
		}
	}
	defer x.tr("checkType(%s)", def.name())(&errs)
	switch {
	case def.Alias != nil:
		errs = checkTypeName(x, def.Alias)
	case def.Fields != nil:
		errs = checkFields(x, def.Fields)
	case def.Cases != nil:
		errs = checkCases(x, def.Cases)
	case def.Virts != nil:
		errs = checkVirts(x, def.Virts)
	}
	return errs
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
		if prev, ok := seen[cas.Name]; ok {
			err := x.err(cas, "case %s redefined", cas.Name)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[cas.Name] = cas
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
			_, es := findVirts(x, arg.AST, arg.Type, iface.Type.Virts)
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
			if i == len(astStmts)-1 {
				expr, es = checkExpr(x, want, astStmt)
			} else {
				expr, es = checkExpr(x, nil, astStmt)
			}
			errs = append(errs, es...)
			stmts = append(stmts, expr)
		default:
			panic(fmt.Sprintf("impossible type: %T", astStmt))
		}
	}
	return stmts, errs
}

func checkRet(x *scope, astRet *ast.Ret) (_ *Ret, errs []checkError) {
	defer x.tr("checkRet(…)")(&errs)

	var want *Type
	if fun := x.function(); fun == nil {
		err := x.err(astRet, "return outside of a function or method")
		errs = append(errs, *err)
	} else if fun.Sig.Ret != nil {
		want = fun.Sig.Ret.Type
	}
	expr, es := checkExpr(x, want, astRet.Val)
	return &Ret{AST: astRet, Val: expr}, append(errs, es...)
}

func checkAssign(x *scope, astAss *ast.Assign) (_ *scope, _ []Stmt, errs []checkError) {
	defer x.tr("checkAssign(…)")(&errs)

	x, vars, newLocal, errs := checkAssignVars(x, astAss)

	if len(vars) == 1 {
		var es []checkError
		assign := &Assign{AST: astAss, Var: vars[0]}
		assign.Expr, es = checkExpr(x, vars[0].typ, astAss.Expr)
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
		typ:   recvType,
		Local: loc,
		Index: len(*loc),
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
			AST:  astCall,
			Recv: &Ident{Text: tmp.Name, Var: tmp},
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
			typ = typName.Type
			errs = append(errs, es...)
		}

		var found interface{}
		if astVar.Type == nil {
			// If the Type is specified, this is always a new definition.
			found = x.findIdent(astVar.Name)
		}
		switch found := found.(type) {
		case nil:
			x.log("adding local %s", astVar.Name)
			loc := x.locals()
			vr := &Var{
				AST:      astVar,
				Name:     astVar.Name,
				TypeName: typName,
				typ:      typ,
				Local:    loc,
				Index:    len(*loc),
			}
			*loc = append(*loc, vr)
			x = x.new()
			x.variable = vr
			vars[i] = vr
			newLocal[i] = true
		case *Var:
			if !found.isSelf() {
				if astVar.Type != nil {
					err := x.err(astVar, "%s redefined", astVar.Name)
					note(err, "previous definition at %s", x.loc(found))
					errs = append(errs, *err)
				}
				vars[i] = found
				break
			}
			err := x.err(astVar, "cannot assign to self")
			errs = append(errs, *err)
			vars[i] = &Var{
				AST:      astVar,
				Name:     astVar.Name,
				TypeName: typName,
				typ:      typ,
			}
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

	if expr, errs = _checkExpr(x, infer, astExpr); len(errs) > 0 {
		return expr, errs
	}
	if expr.Type() == nil {
		return expr, errs
	}
	if infer == nil {
		return expr, errs
	}
	x.log("have %s (%p)", expr.Type(), expr.Type())
	x.log("want %s (%p)", infer, infer)

	gotI, got := refBaseType(x, expr.Type())
	wantI, want := refBaseType(x, infer)
	x.log("have base %s (%p)", got, got)
	x.log("want base %s (%p)", want, want)
	if got == want && gotI != wantI {
		return &Convert{Expr: expr, Ref: wantI - gotI, typ: want}, errs
	}

	if got != want && len(want.Virts) > 0 {
		funs, es := findVirts(x, astExpr, got, want.Virts)
		if len(es) == 0 {
			return &Convert{Expr: expr, Virts: funs, typ: want}, errs
		}
		err := x.err(astExpr, "type %s does not implement %s", got, want)
		err.cause = es
		errs = append(errs, *err)
	}

	if got != want {
		err := x.err(expr, "type mismatch: have %s, want %s", expr.Type(), infer)
		if got.Var != nil && want.Var != nil && got.Name == want.Name {
			if got.AST != nil {
				note(err, "have type %s defined at %s", got, x.loc(got))
			} else {
				note(err, "have type %s is from a built-in definiton", got)
			}
			if want.AST != nil {
				note(err, "want type %s defined at %s", want, x.loc(want))
			} else {
				note(err, "want type %s is from a built-in definiton", want)
			}
		}
		errs = append(errs, *err)
	}
	return expr, errs
}

func refBaseType(x *scope, typ *Type) (int, *Type) {
	var i int
	for isRef(x, typ) {
		i++
		typ = typ.Args[0].Type
	}
	return i, typ
}

func findVirts(x *scope, loc ast.Node, recv *Type, virts []FunSig) (funs []*Fun, errs []checkError) {
	defer x.tr("findVirts(%s %v)", recv, virts)(&errs)

	funs = make([]*Fun, len(virts))
	for i, want := range virts {
		var ret *Type
		if want.Ret != nil {
			ret = want.Ret.Type
		}
		argTypes := funSigArgTypes{loc: loc, sig: &want}
		fun, es := findFunInst(x, ret, recv, nil, want.Sel, argTypes)
		if fun == nil {
			err := x.err(loc, "method %s %s not found", recv, want.Sel)
			err.cause = es
			errs = append(errs, *err)
			continue
		}

		// Make a copy and remove the self parameter.
		funSig := fun.Sig
		funSig.Parms = funSig.Parms[1:]

		if funSigEq(&funSig, &want) {
			funs[i] = fun
			continue
		}
		// Clear the parameter names for printing the error note.
		for i := range funSig.Parms {
			funSig.Parms[i].Name = ""
		}
		var where string
		if fun.AST != nil {
			where = fmt.Sprintf(", defined at %s", x.loc(fun.AST))
		}
		err := x.err(loc, "wrong type for method %s", want.Sel)
		err.notes = []string{
			fmt.Sprintf("wrong type for method %s", want.Sel),
			fmt.Sprintf("	have %s%s", funSig, where),
			fmt.Sprintf("	want %s", want),
		}
		errs = append(errs, *err)
	}
	return funs, errs
}

func funSigEq(a, b *FunSig) bool {
	if a.Sel != b.Sel || len(a.Parms) != len(b.Parms) || (a.Ret == nil) != (b.Ret == nil) {
		return false
	}
	for i := range a.Parms {
		aParm := &a.Parms[i]
		bParm := &b.Parms[i]
		if aParm.typ != bParm.typ {
			return false
		}
	}
	return a.Ret == nil || a.Ret.Type == b.Ret.Type
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
		return checkRune(x, astExpr)
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
	var recvType *Type
	if astCall.Recv != nil {
		recv, errs = checkExpr(x, nil, astCall.Recv)
		recvType = recv.Type()
		switch {
		case recvType == nil:
			x.log("call receiver check error")
			// There was a receiver, but we don't know it's type.
			// That error was reported elsewhere, but we can't continue here.
			// Do best-effort checking of the message arguments.
			for i := range astCall.Msgs {
				astMsg := &astCall.Msgs[i]
				call.Msgs[i] = Msg{
					AST: astMsg,
					Mod: identString(astMsg.Mod),
					Sel: astMsg.Sel,
				}
				var es []checkError
				call.Msgs[i].Args, es = checkExprs(x, astMsg.Args)
				errs = append(errs, es...)
			}
			return call, errs
		case isRef(x, recvType) && isRef(x, recvType.Args[0].Type):
			r := &Convert{Expr: recv, Ref: -1}
			for isRef(x, recvType.Args[0].Type) {
				r.Ref--
				recvType = recvType.Args[0].Type
			}
			recv = r
		case !isRef(x, recvType):
			recv = &Convert{Expr: recv, Ref: 1}
			recvType = builtInType(x, "&", *makeTypeName(recvType))
		}
		if !isRef(x, recvType) || isRef(x, recvType.Args[0].Type) {
			panic("impossible")
		}
		recvType = recvType.Args[0].Type
	}
	for i := range astCall.Msgs {
		var es []checkError
		call.Msgs[i], es = checkMsg(x, infer, recvType, &astCall.Msgs[i])
		errs = append(errs, es...)
	}

	lastMsg := &call.Msgs[len(call.Msgs)-1]
	if lastMsg.Fun == nil {
		return call, errs
	}
	if lastMsg.Fun.Sig.Ret == nil {
		call.typ = builtInType(x, "Nil")
		return call, errs
	}
	call.typ = lastMsg.Fun.Sig.Ret.Type
	return call, errs
}

func checkMsg(x *scope, infer, recv *Type, astMsg *ast.Msg) (_ Msg, errs []checkError) {
	defer x.tr("checkMsg(infer=%s, %s, %s)", infer, recv, astMsg.Sel)(&errs)

	msg := Msg{
		AST:  astMsg,
		Mod:  identString(astMsg.Mod),
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
		typ := parms[i].typ
		msg.Args[i], es = checkExpr(x, typ, astArg)
		errs = append(errs, es...)
	}
	return msg, errs
}

func findMsgFun(x *scope, infer, recv *Type, msg *Msg) (errs []checkError) {
	x.tr("findMsgFun(infer=%s, %s, %s)", infer, recv, msg.name())(&errs)

	var mod *ast.Ident
	var modName string
	if msg.Mod != "" {
		// msg.ast must be an *ast.Msg,
		// since the only other case is ast.Ident,
		// which is only for in-module function calls,
		// and this is not in-module.
		mod = msg.AST.(*ast.Msg).Mod
		modName = mod.Text + " "
	}
	fun, es := findFunInst(x, infer, recv, mod, msg.Sel, msg)
	if fun == nil {
		var err *checkError
		if recv == nil {
			err = x.err(msg, "function %s%s not found", modName, msg.Sel)
		} else {
			err = x.err(msg, "method %s %s%s not found", recv, modName, msg.Sel)
		}
		err.cause = es
		errs = append(errs, *err)
	}
	msg.Fun = fun
	return errs
}

func findFunInst(x *scope, infer, recv *Type, mod *ast.Ident, sel string, argTypes argTypes) (fun *Fun, errs []checkError) {
	x.tr("findFunInst(infer=%s, %s, %s)", infer, recv, sel)(&errs)

	switch {
	case recv != nil && recv.Var != nil:
		for _, iface := range recv.Var.Ifaces {
			if iface.Type == nil {
				continue
			}
			fun = x.findFun(iface.Type, sel)
			recv = iface.Type
		}
	case mod != nil:
		imp := x.findImport(mod.Text)
		if imp == nil {
			err := x.err(mod, "module %s not found", mod.Text)
			return nil, []checkError{*err}
		}
		fun = imp.findFun(recv, sel)
	default:
		fun = x.findFun(recv, sel)
	}
	if fun == nil {
		return nil, nil
	}
	if recv != nil && recv.Var == nil && recv != fun.Recv.Type {
		if fun, errs = instRecv(x, recv, fun); len(errs) > 0 {
			return nil, errs
		}
	}
	if len(fun.TParms) > 0 {
		if fun, errs = instFun(x, infer, fun, argTypes); len(errs) > 0 {
			return nil, errs
		}
	}
	return fun, nil
}

func checkCtor(x *scope, infer *Type, astCtor *ast.Ctor) (_ *Ctor, errs []checkError) {
	defer x.tr("checkCtor(infer=%s)", infer)(&errs)

	ctor := &Ctor{AST: astCtor, typ: infer}
	switch {
	case ctor.typ == nil:
		err := x.err(ctor, "cannot infer constructor type")
		errs = append(errs, *err)
	case ctor.typ.Alias != nil:
		// This should have already been resolved by gatherTypeName.
		panic("impossible alias")
	case isAry(x, ctor.typ):
		errs = append(errs, checkAryCtor(x, ctor)...)
	case ctor.typ.Cases != nil:
		errs = append(errs, checkOrCtor(x, ctor)...)
	case ctor.typ.Virts != nil:
		err := x.err(astCtor, "cannot construct virtual type %s", ctor.typ)
		errs = append(errs, *err)
	case isBuiltIn(x, ctor.typ) && !isNil(x, ctor.typ):
		err := x.err(astCtor, "cannot construct built-in type %s", ctor.typ)
		errs = append(errs, *err)
	default:
		errs = append(errs, checkAndCtor(x, ctor)...)
	}
	return ctor, errs
}

func checkAryCtor(x *scope, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkAryCtor(%s)", ctor.typ)(&errs)
	want := ctor.typ.Args[0].Type
	ctor.Args = make([]Expr, len(ctor.AST.Args))
	for i, expr := range ctor.AST.Args {
		var es []checkError
		ctor.Args[i], es = checkExpr(x, want, expr)
		errs = append(errs, es...)
	}
	return errs
}

func checkOrCtor(x *scope, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkOrCtor(%s)", ctor.typ)(&errs)

	sel, arg, ok := disectOrCtorArg(ctor.AST)
	if !ok {
		err := x.err(ctor, "malformed %s constructor", ctor.typ)
		return append(errs, *err)
	}

	ctor.Case = findCase(ctor.typ, sel)
	if ctor.Case == nil {
		err := x.err(ctor, "case %s not found", sel)
		errs = append(errs, *err)
		var es []checkError
		ctor.Args, es = checkExprs(x, ctor.AST.Args)
		return append(errs, es...)
	}
	c := &ctor.typ.Cases[*ctor.Case]

	if c.TypeName == nil {
		if arg != nil {
			panic("impossible")
		}
		return errs
	}

	expr, es := checkExpr(x, c.typ, arg)
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

func checkAndCtor(x *scope, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkAndCtor(%s)", ctor.typ)(&errs)

	if len(ctor.AST.Args) == 0 {
		return errs
	}
	call, ok := ctor.AST.Args[0].(*ast.Call)
	if !ok || len(ctor.AST.Args) > 1 || call.Recv != nil || len(call.Msgs) != 1 {
		err := x.err(ctor, "malformed %s constructor", ctor.typ)
		return append(errs, *err)
	}

	astArgs := make([]ast.Expr, len(ctor.typ.Fields))
	fieldNames := strings.Split(call.Msgs[0].Sel, ":")
	for i, astArg := range call.Msgs[0].Args {
		fieldName := fieldNames[i]
		field := findField(ctor.typ, fieldName)
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

	ctor.Args = make([]Expr, len(ctor.typ.Fields))
	for i := range ctor.typ.Fields {
		field := &ctor.typ.Fields[i]
		if astArgs[i] == nil {
			err := x.err(ctor, "missing field: %s", field.Name)
			errs = append(errs, *err)
			continue
		}
		var es []checkError
		ctor.Args[i], es = checkExpr(x, field.typ, astArgs[i])
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
	if isFun(x, infer) {
		x.log("is a fun")
		resInfer = infer.Args[len(infer.Args)-1].Type
		n := len(infer.Args)
		if n > len(astBlock.Parms) {
			n = len(astBlock.Parms)
		}
		for i := 0; i < n; i++ {
			parmInfer[i] = infer.Args[i].Type
		}
	} else {
		x.log("is not a fun")
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
		parm.typ = parm.TypeName.Type
		errs = append(errs, es...)
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
		err := x.err(astBlock, "too many block parameters: got %, max %d",
			len(astBlock.Parms), MaxValueParms)
		errs = append(errs, *err)
		return blk, errs
	}

	typeArgs := make([]TypeName, len(blk.Parms)+1)
	for i := range blk.Parms {
		parm := &blk.Parms[i]
		if parm.typ == nil {
			return blk, errs
		}
		if parm.TypeName != nil {
			typeArgs[i] = *parm.TypeName
			continue
		}
		typeArgs[i] = TypeName{
			AST:  &astBlock.Parms[i],
			Mod:  parm.typ.Mod,
			Name: parm.typ.Name,
			Args: parm.typ.Args,
			Type: parm.typ,
		}
	}

	resType := builtInType(x, "Nil")
	if n := len(blk.Stmts); n > 0 {
		if expr, ok := blk.Stmts[n-1].(Expr); ok {
			resType = expr.Type()
		}
	}
	if resType == nil {
		return blk, errs
	}
	typeArgs[len(typeArgs)-1] = TypeName{
		AST:  astBlock,
		Mod:  resType.Mod,
		Name: resType.Name,
		Args: resType.Args,
		Type: resType,
	}
	blk.typ = builtInType(x, "Fun", typeArgs...)
	return blk, errs
}

func checkIdent(x *scope, infer *Type, astIdent *ast.Ident) (_ Expr, errs []checkError) {
	defer x.tr("checkIdent(infer=%s, %s)", infer, astIdent.Text)(&errs)

	ident := &Ident{AST: astIdent, Text: astIdent.Text}
	switch vr := x.findIdent(astIdent.Text).(type) {
	case nil:
		err := x.err(astIdent, "identifier %s not found", astIdent.Text)
		errs = append(errs, *err)
	case *Var:
		ident.Var = vr
	case *Fun:
		defer x.tr("checkMsg(infer=%s, %s, %s)", infer, nil, astIdent.Text)(&errs)
		msg := Msg{AST: astIdent, Sel: astIdent.Text}
		es := findMsgFun(x, infer, nil, &msg)
		errs = append(errs, es...)
		call := &Call{AST: astIdent, Msgs: []Msg{msg}}
		if msg.Fun == nil {
			return call, errs
		}
		if msg.Fun.Sig.Ret == nil {
			call.typ = builtInType(x, "Nil")
		} else {
			call.typ = msg.Fun.Sig.Ret.Type
		}
		return call, errs
	default:
		panic(fmt.Sprintf("impossible type: %T", vr))
	}
	return ident, errs
}

func checkInt(x *scope, infer *Type, AST ast.Expr, text string) (_ Expr, errs []checkError) {
	defer x.tr("checkInt(infer=%s, %s)", infer, text)(&errs)

	if isFloat(x, infer) {
		return checkFloat(x, infer, AST, text)
	}
	var i big.Int
	x.log("parsing int [%s]", text)
	if _, ok := i.SetString(text, 0); !ok {
		panic("malformed int")
	}
	typ := builtInType(x, "Int")
	if isInt(x, infer) {
		typ = infer
	}
	if err := checkIntBounds(x, AST, typ, &i); err != nil {
		errs = append(errs, *err)
	}
	return &Int{AST: AST, Val: &i, typ: typ}, errs
}

func checkIntBounds(x *scope, n interface{}, t *Type, i *big.Int) *checkError {
	signed, bits := disectIntType(x, t)
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

func disectIntType(x *scope, typ *Type) (bool, int) {
	switch typ {
	case builtInType(x, "Int"):
		return true, x.cfg.IntSize
	case builtInType(x, "Int8"):
		return true, 7
	case builtInType(x, "Int16"):
		return true, 15
	case builtInType(x, "Int32"):
		return true, 31
	case builtInType(x, "Int64"):
		return true, 63
	case builtInType(x, "UInt"):
		return false, x.cfg.IntSize
	case builtInType(x, "UInt8"):
		return false, 8
	case builtInType(x, "UInt16"):
		return false, 16
	case builtInType(x, "UInt32"):
		return false, 32
	case builtInType(x, "UInt64"):
		return false, 64
	default:
		panic(fmt.Sprintf("impossible int type: %T", typ))
	}
}

func checkFloat(x *scope, infer *Type, AST ast.Expr, text string) (_ Expr, errs []checkError) {
	defer x.tr("checkFloat(infer=%s, %s)", infer, text)(&errs)

	var f big.Float
	if _, _, err := f.Parse(text, 10); err != nil {
		panic("malformed float")
	}
	if isInt(x, infer) {
		var i big.Int
		if _, acc := f.Int(&i); acc != big.Exact {
			err := x.err(AST, "type %s cannot represent %s: truncation", infer.name(), text)
			errs = append(errs, *err)
		}
		expr, es := checkInt(x, infer, AST, i.String())
		return expr, append(errs, es...)
	}
	typ := builtInType(x, "Float")
	if isFloat(x, infer) {
		typ = infer
	}
	return &Float{AST: AST, Val: &f, typ: typ}, errs
}

func isInt(x *scope, typ *Type) bool {
	switch {
	case typ == nil:
		return false
	default:
		return false
	case typ == builtInType(x, "Int") ||
		typ == builtInType(x, "Int8") ||
		typ == builtInType(x, "Int16") ||
		typ == builtInType(x, "Int32") ||
		typ == builtInType(x, "Int64") ||
		typ == builtInType(x, "UInt") ||
		typ == builtInType(x, "UInt8") ||
		typ == builtInType(x, "UInt16") ||
		typ == builtInType(x, "UInt32") ||
		typ == builtInType(x, "UInt64"):
		return true
	}
}

func isFloat(x *scope, typ *Type) bool {
	switch {
	case typ == nil:
		return false
	default:
		return false
	case typ == builtInType(x, "Float") ||
		typ == builtInType(x, "Float32") ||
		typ == builtInType(x, "Float64"):
		return true
	}
}

func checkRune(x *scope, astRune *ast.Rune) (*Int, []checkError) {
	defer x.tr("checkRune(%s)", astRune.Text)()
	return &Int{
		AST: astRune,
		Val: big.NewInt(int64(astRune.Rune)),
		typ: builtInType(x, "Int32"),
	}, nil
}

func checkString(x *scope, astString *ast.String) (*String, []checkError) {
	defer x.tr("checkString(%s)", astString.Text)()
	return &String{
		AST:  astString,
		Data: astString.Data,
		typ:  builtInType(x, "String"),
	}, nil
}

func identString(id *ast.Ident) string {
	if id == nil {
		return ""
	}
	return id.Text
}
