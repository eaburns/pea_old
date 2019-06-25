package ast

import (
	"fmt"
	"math/big"
	"strings"
)

// An Opt is an option to the type checker.
type Opt func(*state)

var (
	// Trace enables tracing of the type checker.
	Trace Opt = func(x *state) { x.trace = true }
	// Dump writes the definition tree to stdout after checking.
	Dump = func(x *state) { x.dump = true }
	// Word64 makes Word and Uint aliases to Uint64, and Int an alias to Int64.
	Word64 = func(x *state) { x.wordSize = 64 }
	// Word32 makes Word and Uint aliases to Uint32, and Int an alias to Int32.
	Word32 = func(x *state) { x.wordSize = 32 }
)

// Check type-checks the module:
// It imports definitions from imported modules,
// sets fields on the AST with the resolution of references,
// and instantiates uses of type-parameterized definitions.
func Check(mod *Mod, opts ...Opt) []error {
	x := newScope(mod, opts...)
	errs := checkMod(x, mod)
	if x.state.dump {
		dump(x.state.mods, "")
	}
	return convertErrors(errs)
}

func checkMod(x *scope, mod *Mod) (errs []checkError) {
	defer x.tr("checkMod(%s)", mod.Name)(errs)
	errs = collect(x, nil, []string{mod.Name}, mod)

	// First check types, since expression checking assumes
	// that type definitions have been fully checked
	// and have non-nil TypeName.Type values.
	for _, def := range mod.Defs {
		errs = append(errs, checkDef(x, def)...)
	}

	// Check instantiated defs too.
	// However, here we ignore the errors,
	// because any errors must have already
	// been reported above.
	for _, def := range x.typeInsts {
		if typ, ok := def.(*Type); ok {
			checkDef(x, typ)
		}
	}
	for _, def := range x.methInsts {
		if fun, ok := def.(*Fun); ok {
			checkDef(x, fun)
		}
	}

	for _, def := range mod.Defs {
		errs = append(errs, checkDefStmts(x, def)...)
	}
	// Check instantiated fun statements too.
	// Again, ignore errors, since they must have
	// been reported above already.
	for _, fun := range x.funInsts {
		checkDefStmts(x, fun)
	}

	return errs
}

func checkDef(x *scope, def Def) (errs []checkError) {
	defer x.tr("checkDef(%s)", defName(def))(&errs)
	x = &scope{state: x.state, parent: x, def: def}
	switch def := def.(type) {
	case *Import:
		return nil // handled elsewhere
	case *Fun:
		return checkFun(x, def)
	case *Var:
		return checkVar(x, def)
	case *Type:
		return checkType(x, def)
	default:
		panic(fmt.Sprintf("impossible Def type %T", def))
	}
}

func checkDefStmts(x *scope, def Def) (errs []checkError) {
	defer x.tr("checkDefStmts(%s)", def.Name())(&errs)

	x = &scope{state: x.state, parent: x, def: def}
	switch def := def.(type) {
	case *Fun:
		errs = append(errs, checkFunStmts(x, def)...)
	case *Var:
		errs = append(errs, checkVarStmts(x, def)...)
	}
	return errs
}

func checkVar(x *scope, vr *Var) (errs []checkError) {
	defer x.tr("checkVar(%s)", vr.Ident)(&errs)
	if vr.Type != nil {
		errs = append(errs, checkTypeName(x, vr.Type)...)
	}
	// TODO: checkVar only checks the type if the name is explicit.
	// There should be a pass after all checkDefs
	// to check the var def statements in topological order,
	// setting the inferred types.
	return errs
}

func checkVarStmts(x *scope, vr *Var) (errs []checkError) {
	defer x.tr("checkVarStmts(%s)", vr.Ident)(&errs)
	if ss, es := checkStmts(x, vr.Val, vr.Type); len(es) > 0 {
		errs = append(errs, es...)
	} else {
		vr.Val = ss
	}
	if vr.Type == nil {
		typ := builtInType(x, "Nil")
		if len(vr.Val) > 0 {
			if expr, ok := vr.Val[len(vr.Val)-1].(Expr); ok && expr.ExprType() != nil {
				typ = expr.ExprType()
			}
		}
		vr.Type = typ
	}
	return errs
}

func checkFun(x *scope, fun *Fun) (errs []checkError) {
	defer x.tr("checkFun(%s)", fun)(&errs)

	if fun.Recv != nil {
		errs = append(errs, checkRecvSig(x, fun)...)
		x = fun.Recv.x

		if len(fun.Recv.Parms) > 0 && len(fun.Recv.Args) == 0 {
			// TODO: checkFun for param receivers is only partially implemented.
			// We should create stub type arguments, instantiate the fun,
			// then check the instance.
		}
	}
	if len(fun.TypeParms) > 0 && len(fun.TypeArgs) == 0 {
		// TODO: checkFun for parameterized funs is unimplemented.
		// We should create stub type arguments, instantiate the fun,
		// then check the instance.
	}
	for i := range fun.TypeParms {
		p := &fun.TypeParms[i]
		x = x.push(p.Name, p)
		x.typeVars[p] = makeTypeVarType(p)
		if p.Type != nil {
			errs = append(errs, checkTypeName(x, p.Type)...)
		}
	}
	fun.x = x

	seen := make(map[string]*Parm)
	if fun.Recv != nil && fun.RecvType != nil {
		fun.Self = &Parm{
			location: fun.Recv.location,
			Name:     "self",
			Type:     typeName(fun.RecvType),
		}
		seen[fun.Self.Name] = fun.Self
		x = x.push(fun.Self.Name, fun.Self)
	}

	for i := range fun.Parms {
		p := &fun.Parms[i]
		switch prev := seen[p.Name]; {
		case prev != nil:
			err := x.err(p, "parameter %s is redefined", p.Name)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		case p.Name == "_" || p.Name == "":
			break
		default:
			seen[p.Name] = p
			x = x.push(p.Name, p)
		}
		errs = append(errs, checkTypeName(x, p.Type)...)
	}
	if fun.Ret != nil {
		errs = append(errs, checkTypeName(x, fun.Ret)...)
	}

	return errs
}

func checkRecvSig(x *scope, fun *Fun) (errs []checkError) {
	defer x.tr("checkRecvSig(%s)", fun)(&errs)

	var def Def
	defOrImport := x.mods.find(fun.Mod(), fun.Recv.Name)
	switch d := defOrImport.(type) {
	case builtin:
		def = d.Def
	case imported:
		def = d.Def
	case Def:
		def = d
	case nil:
		err := x.err(fun, "type %s is undefined", fun.Recv.Name)
		errs = append(errs, *err)
	}

	var typ *Type
	if def != nil {
		var ok bool
		if typ, ok = def.(*Type); !ok {
			err := x.err(fun, "got %s, expected a type", def.kind())
			addDefNotes(err, x, defOrImport)
			errs = append(errs, *err)
		}
	}

	if typ != nil {
		if len(fun.Recv.Parms) != len(typ.Sig.Parms) {
			err := x.err(fun, "parameter count mismatch: got %d, expected %d",
				len(fun.Recv.Parms), len(typ.Sig.Parms))
			addDefNotes(err, x, defOrImport)
			errs = append(errs, *err)
		}
		for i := range typ.Virts {
			v := &typ.Virts[i]
			if v.Sel == fun.Sel {
				err := x.err(fun, "method %s is redefined", fun.Sel)
				note(err, "previous definition is a virtual method")
				addDefNotes(err, x, defOrImport)
				errs = append(errs, *err)
			}
		}
		fun.RecvType = typ
	}

	for i := range fun.Recv.Parms {
		p := &fun.Recv.Parms[i]
		x = x.push(p.Name, p)
		x.typeVars[p] = makeTypeVarType(p)
		if p.Type != nil {
			errs = append(errs, checkTypeName(x, p.Type)...)
		}
	}
	fun.Recv.x = x
	return errs
}

func makeTypeVarType(p *Parm) *Type {
	var virts []MethSig
	// TODO: makeTypeVar should set virtuals from the constraint, if any.
	loc := location{start: p.Start(), end: p.End()}
	return &Type{
		location: loc,
		Sig:      TypeSig{location: loc, Name: p.Name},
		Virts:    virts,
	}
}

func checkFunStmts(x *scope, fun *Fun) (errs []checkError) {
	defer x.tr("checkFunStmts(%s)", fun)(&errs)

	if fun.Recv != nil && len(fun.Recv.Parms) > 0 && len(fun.Recv.Args) == 0 {
		// TODO: checkFunStmts for param receivers is only partially implemented.
		// We should create stub type arguments, instantiate the fun,
		// then check the instance.
		x.log("skipping lifted function")
		return errs
	}
	if len(fun.TypeParms) > 0 && len(fun.TypeArgs) == 0 {
		// TODO: checkFunStmts for parameterized funs is unimplemented.
		// We should create stub type arguments, instantiate the fun,
		// then check the instance.
		x.log("skipping lifted function")
		return errs
	}

	seen := make(map[string]*Parm)
	if fun.Self != nil {
		seen[fun.Self.Name] = fun.Self
		x = x.push(fun.Self.Name, fun.Self)
	}
	for i := range fun.Parms {
		p := &fun.Parms[i]
		if seen[p.Name] == nil && p.Name != "_" {
			seen[p.Name] = p
			x = x.push(p.Name, p)
		}
	}
	if ss, es := checkStmts(x, fun.Stmts, nil); len(es) > 0 {
		errs = append(errs, es...)
	} else {
		fun.Stmts = ss
	}
	return errs
}

func checkType(x *scope, typ *Type) (errs []checkError) {
	defer x.tr("checkType(%s)", typ)(&errs)

	if len(typ.Sig.Parms) > 0 && len(typ.Sig.Args) == 0 {
		for i := range typ.Sig.Parms {
			p := &typ.Sig.Parms[i]
			x = x.push(p.Name, p)
			x.typeVars[p] = makeTypeVarType(p)
			if p.Type != nil {
				errs = append(errs, checkTypeName(x, p.Type)...)
			}
		}
		typ.Sig.x = x
		// TODO: checkType for param types is only partially implemented.
		// We should create stub type arguments, instantiate the type,
		// then check the instance.
	}

	switch {
	case typ.Alias != nil:
		errs = append(errs, checkAliasType(x, typ)...)
	case typ.Fields != nil:
		errs = append(errs, checkFields(x, typ.Fields)...)
	case typ.Cases != nil:
		errs = append(errs, checkCases(x, typ.Cases)...)
	case typ.Virts != nil:
		errs = append(errs, checkVirts(x, typ.Virts)...)
	}
	return errs
}

func checkAliasType(x *scope, typ *Type) (errs []checkError) {
	defer x.tr("checkAliasType(%s)", typ)(&errs)
	if _, ok := x.aliasCycles[typ]; ok {
		// This alias is already found to be on a cycle.
		// The error is returned at the root of the cycle.
		return errs
	}
	if x.onAliasPath[typ] {
		markAliasCycle(x, x.aliasPath, typ)
		// The error is returned at the root of the cycle.
		return errs
	}

	x.onAliasPath[typ] = true
	x.aliasPath = append(x.aliasPath, typ)
	defer func() {
		delete(x.onAliasPath, typ)
		x.aliasPath = x.aliasPath[:len(x.aliasPath)-1]
	}()

	if errs = checkTypeName(x, typ.Alias); len(errs) > 0 {
		return errs
	}
	// checkTypeName can make a recursive call to checkAliasType.
	// In the case of an alias cycle, that call would have added the error
	// to the aliasCycles map for the root alias definition.
	// We return any such error here.
	if err := x.aliasCycles[typ]; err != nil {
		errs = append(errs, *err)
	}
	return errs
}

func markAliasCycle(x *scope, path []*Type, alias *Type) {
	err := x.err(alias, "type alias cycle")
	i := len(x.aliasPath) - 1
	for x.aliasPath[i] != alias {
		i--
	}
	for ; i < len(x.aliasPath); i++ {
		note(err, "%s at %s", defName(x.aliasPath[i]), x.loc(x.aliasPath[i]))
		// We only want to report the cycle error for one definition.
		// Mark all other nodes on the cycle with a nil error.
		x.aliasCycles[x.aliasPath[i]] = nil
	}
	note(err, "%s at %s", defName(alias), x.loc(alias))
	x.aliasCycles[alias] = err
}

func checkFields(x *scope, ps []Parm) (errs []checkError) {
	defer x.tr("checkFields(…)")(&errs)
	seen := make(map[string]*Parm)
	for i := range ps {
		p := &ps[i]
		if prev := seen[p.Name]; prev != nil {
			err := x.err(p, "field %s is redefined", p.Name)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[p.Name] = p
		}
		errs = append(errs, checkTypeName(x, p.Type)...)
	}
	return errs
}

func checkCases(x *scope, ps []Parm) (errs []checkError) {
	defer x.tr("checkCases(…)")(&errs)
	seen := make(map[string]*Parm)
	for i := range ps {
		p := &ps[i]
		lower := strings.ToLower(p.Name)
		if prev := seen[lower]; prev != nil {
			err := x.err(p, "case %s is redefined", lower)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[lower] = p
		}
		if p.Type != nil {
			errs = append(errs, checkTypeName(x, p.Type)...)
		}
	}
	return errs
}

func checkVirts(x *scope, sigs []MethSig) (errs []checkError) {
	defer x.tr("checkVirts(…)")(&errs)
	seen := make(map[string]*MethSig)
	for i := range sigs {
		sig := &sigs[i]
		if prev, ok := seen[sig.Sel]; ok {
			err := x.err(sig, "virtual method %s is redefined", sig.Sel)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[sig.Sel] = sig
		}
		for j := range sig.Parms {
			errs = append(errs, checkTypeName(x, &sig.Parms[j])...)
		}
		if sig.Ret != nil {
			errs = append(errs, checkTypeName(x, sig.Ret)...)
		}
	}
	return errs
}

func checkTypeName(x *scope, name *TypeName) (errs []checkError) {
	defer x.tr("checkTypeName(%s)", name)(&errs)

	if name.Var {
		def := x.find(name.Name)
		if def == nil {
			err := x.err(name, "type variable %s is undefined", name.Name)
			errs = append(errs, *err)
			return errs
		}
		p, ok := def.(*Parm)
		if !ok || x.typeVars[p] == nil {
			panic(fmt.Sprintf("impossible: %s not a type var", name.Name))
		}
		name.Type = x.typeVars[p]
		return errs
	}

	for i := range name.Args {
		errs = append(errs, checkTypeName(x, &name.Args[i])...)
	}

	if name.Mod == nil {
		name.Mod = x.modPath()
	}

	var def Def
	defOrImport := x.mods.find(*name.Mod, name.Name)
	switch d := defOrImport.(type) {
	case builtin:
		def = d.Def
	case imported:
		def = d.Def
	case Def:
		def = d
	case nil:
		err := x.err(name, "type %s is undefined", name)
		errs = append(errs, *err)
		return errs
	}

	typ, ok := def.(*Type)
	if !ok {
		err := x.err(name, "got %s, expected a type", def.kind())
		addDefNotes(err, x, defOrImport)
		return append(errs, *err)
	}
	x.log("found type %s (%p)", typ, typ)

	if len(typ.Sig.Parms) > 0 {
		name.Type = typ
		var es []checkError
		if typ, es = typ.inst(x, *name); len(es) > 0 {
			err := x.err(name, "%s cannot be instantiated", name)
			err.cause = es
			return append(errs, *err)
		}
	}

	if typ != nil && typ.Alias != nil {
		if checkAliasType(x, typ) != nil {
			// Return nil, because the error is reported
			// by the call to checkAliasType from its definition.
			return nil
		}
		x.log("setting type of %s to alias %s (%p) of %s",
			name, typ.Alias.Type, typ.Alias.Type, typ.Alias)
		name.Type = typ.Alias.Type
	} else {
		x.log("setting type of %s to %s (%p) ", name, typ, typ)
		name.Type = typ
	}
	return errs
}

func checkStmts(x *scope, stmts []Stmt, blockRes *TypeName) (_ []Stmt, errs []checkError) {
	defer x.tr("checkStmts(…)")(&errs)

	var out []Stmt
	for i := range stmts {
		switch stmt := stmts[i].(type) {
		case *Ret:
			if es := checkRet(x, stmt); len(es) > 0 {
				errs = append(errs, es...)
			}
			out = append(out, stmt)
		case *Assign:
			var ss []Stmt
			var es []checkError
			if x, ss, es = checkAssign(x, stmt); len(es) > 0 {
				out = append(out, stmt)
				errs = append(errs, es...)
			} else {
				out = append(out, ss...)
			}
		case Expr:
			var infer *TypeName
			if i == len(stmts)-1 {
				infer = blockRes
			}
			if expr, es := checkExpr(x, stmt, infer); len(es) > 0 {
				out = append(out, stmt)
				errs = append(errs, es...)
			} else {
				out = append(out, expr)
			}
		}
	}

	return out, errs
}

func checkRet(x *scope, ret *Ret) (errs []checkError) {
	defer x.tr("checkRet(…)")(&errs)

	var infer *TypeName
	if fun := x.fun(); fun == nil {
		err := x.err(ret, "return outside of a method")
		errs = append(errs, *err)
	} else {
		infer = fun.Ret
	}
	if expr, es := checkExpr(x, ret.Val, infer); len(es) > 0 {
		errs = append(errs, es...)
	} else {
		ret.Val = expr
	}
	return errs
}

func checkAssign(x *scope, as *Assign) (_ *scope, _ []Stmt, errs []checkError) {
	defer x.tr("checkAssign(…)")(&errs)

	if es := checkAssignCount(x, as); len(es) > 0 {
		errs = append(errs, es...)
		return x, []Stmt{as}, errs
	}
	x, ss, ass, es := splitAssign(x, as)
	if len(es) > 0 {
		errs = append(errs, es...)
	}
	x1 := x
	for _, as := range ass {
		if x1, es = checkAssign1(x, x1, as); len(es) > 0 {
			errs = append(errs, es...)
		}
	}
	return x1, ss, errs
}

func checkAssignCount(x *scope, as *Assign) (errs []checkError) {
	defer x.tr("checkAssignCount(…)")(&errs)
	if len(as.Vars) == 1 {
		return nil
	}
	c, ok := as.Val.(Call)
	if ok && len(c.Msgs) == len(as.Vars) {
		return nil
	}
	got := 1
	if ok {
		got = len(c.Msgs)
	}
	// This is best-effort to report any errors,
	// but since the assignment mismatches
	// the infer type must always be nil.
	if _, es := checkExpr(x, as.Val, nil); len(es) > 0 {
		errs = append(errs, es...)
	}
	for _, p := range as.Vars {
		if p.Type != nil {
			errs = append(errs, checkTypeName(x, p.Type)...)
		}
	}
	err := x.err(as, "assignment count mismatch: got %d, expected %d",
		got, len(as.Vars))
	errs = append(errs, *err)
	return errs
}

func splitAssign(x *scope, as *Assign) (_ *scope, ss []Stmt, ass []*Assign, errs []checkError) {
	defer x.tr("splitAssign(…)")(&errs)

	if len(as.Vars) == 1 {
		return x, []Stmt{as}, []*Assign{as}, nil
	}

	call := as.Val.(Call) // must, because checkAssignCount was OK
	recv := call.Recv
	if expr, ok := recv.(Expr); ok {
		tmp := x.newID()
		loc := location{start: expr.Start(), end: expr.End()}
		p := &Parm{location: loc, Name: tmp}
		a := &Assign{Vars: []*Parm{p}, Val: expr}
		if e, es := checkExpr(x, expr, nil); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			a.Val = e
		}
		ss = append(ss, a)
		locals := x.locals() // cannot be nil
		*locals = append(*locals, p)
		x = x.push(tmp, p)
		recv = Ident{location: loc, Text: tmp}
	}
	for i, p := range as.Vars {
		as := &Assign{
			Vars: []*Parm{p},
			Val: Call{
				location: call.Msgs[i].location,
				Recv:     recv,
				Msgs:     []Msg{call.Msgs[i]},
			},
		}
		ss = append(ss, as)
		ass = append(ass, as)
	}
	return x, ss, ass, errs
}

func checkAssign1(x, x1 *scope, as *Assign) (_ *scope, errs []checkError) {
	defer x.tr("checkAssign1(%s)", as.Vars[0].Name)(&errs)

	vr := as.Vars[0]
	def, _ := x.find(vr.Name).(*Parm)
	if def == nil {
		locals := x.locals() // cannot be nil
		*locals = append(*locals, vr)
		x1 = x1.push(vr.Name, vr)
		def = vr
	}
	if vr.Type != nil {
		errs = append(errs, checkTypeName(x, vr.Type)...)
		if vr != def {
			err := x.err(vr, "%s is redefined", vr.Name)
			note(err, "previous definition is at %s", x.loc(def))
			errs = append(errs, *err)
		}
	}
	as.Vars[0] = def

	var infer *TypeName
	if vr.Type != nil {
		infer = vr.Type
	}

	if expr, es := checkExpr(x, as.Val, infer); len(es) > 0 {
		errs = append(errs, es...)
	} else {
		as.Val = expr
	}
	if vr.Type == nil && as.Val.ExprType() != nil {
		vr.Type = as.Val.ExprType()
	}
	return x1, errs
}

func typeName(typ *Type) *TypeName {
	mp := typ.Mod()
	var args []TypeName
	if typ.Sig.Args != nil {
		for i := range typ.Sig.Parms {
			args = append(args, typ.Sig.Args[&typ.Sig.Parms[i]])
		}
	}
	return &TypeName{
		location: typ.location,
		Mod:      &mp,
		Name:     typ.Sig.Name,
		Args:     args,
		Type:     typ,
	}
}

// checkExpr checks the expression.
// If infer is non-nil, and the expression type is convertable to infer,
// a conversion node is added.
// If infer is non-nil, and the experssion type is not convertable to infer,
// a type-mismatch error is returned.
func checkExpr(x *scope, expr Expr, want *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("checkExpr(got=%s, want=%s)", debugExprTypeString(expr), want)(&errs)
	if expr, errs = expr.check(x, want); len(errs) > 0 {
		return expr, errs
	}
	got := expr.ExprType()
	if want == nil || want.Type == nil || got == nil || got.Type == nil {
		return expr, errs
	}
	if want.Type == got.Type {
		return expr, errs
	}
	x.log("want.Type=%p, got=%p", want.Type, got)

	gotRefs, gotBase := refBaseType(got.Type)
	wantRefs, wantBase := refBaseType(want.Type)
	if gotBase == wantBase {
		for gotRefs > wantRefs {
			t := got.Type.Sig.Args[&got.Type.Sig.Parms[0]]
			expr = Ctor{
				location: location{start: expr.Start(), end: expr.End()},
				Type:     t,
				Sel:      t.Name,
				Args:     []Expr{expr},
			}
			gotRefs--
		}
		for gotRefs < wantRefs {
			got = builtInType(x, "&", *got)
			expr = Ctor{
				location: location{start: expr.Start(), end: expr.End()},
				Type:     *got,
				Sel:      got.Name,
				Args:     []Expr{expr},
			}
			gotRefs++
		}
		return expr, errs
	}

	err := x.err(expr, "got type %s, wanted %s",
		typeStringForUser(got),
		typeStringForUser(want))
	if len(want.Type.Virts) > 0 {
		expr, es := convertVirtual(x, expr, want)
		if len(es) == 0 {
			return expr, errs
		}
		note(err, "%s does not implement %s",
			typeStringForUser(got),
			typeStringForUser(want))
		err.cause = es
	}
	return expr, append(errs, *err)
}

func convertVirtual(x *scope, expr Expr, want *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("convertVirtual(got=%s, want=%s)", debugExprTypeString(expr), want)(&errs)
	ctor := Ctor{
		location: location{start: expr.Start(), end: expr.End()},
		Type:     *want,
		Args:     []Expr{expr},
	}
	for _, v := range want.Type.Virts {
		fun, err := findFun(x, expr, expr, v.Sel)
		if err != nil {
			errs = append(errs, *err)
			continue
		}
		x.log("found %v\n", fun)
		x.log("want %v\n", v)
		if !parmsMatch(x, fun, v) {
			err := x.err(expr, "%s has the wrong type", v.Sel)
			funForString := *fun
			funForString.Recv = nil
			note(err, "got  %s", &funForString)
			note(err, "want %s", methSigStringForUser(v))
			errs = append(errs, *err)
			continue
		}
		ctor.Virts = append(ctor.Virts, fun)
	}
	if len(errs) > 0 {
		return expr, errs
	}
	return ctor, nil
}

func parmsMatch(x *scope, got *Fun, want MethSig) bool {
	defer x.tr("checkParmMatch(got=%s, want=%s)", got, want)()
	switch {
	case got.Ret == nil && want.Ret == nil:
		// ok
	case got.Ret != nil && want.Ret != nil:
		if got.Ret.Type != want.Ret.Type {
			x.log("different return types (%v != %v)",
				got.Ret.Type, want.Ret.Type)
			return false
		}
	default:
		x.log("one returns, one does not")
		return false
	}
	if len(want.Parms) != len(got.Parms) {
		panic("impossible") // same selector, same number of parms
	}
	for i := range want.Parms {
		w := &want.Parms[i]
		g := &got.Parms[i]
		if w.Type != g.Type.Type {
			x.log("parameter %d type mismatch", i)
			return false
		}
	}
	return true
}

func debugExprTypeString(expr Expr) string {
	if expr == nil {
		return "expr=nil"
	}
	if expr.ExprType() == nil {
		return "type=nil"
	}
	return typeStringForUser(expr.ExprType())
}

func (n Call) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Call.check(…)")(&errs)

	switch recv := n.Recv.(type) {
	case Expr:
		if expr, es := recv.check(x, nil); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Recv = expr
		}
	case nil:
		n.Recv = *x.modPath()
	case ModPath:
		// nothing to do
	}

	if n.Recv == nil {
		// We can't find the messages,
		// so just do best-effort error reporting
		// by checking the message arguments.
		for _, msg := range n.Msgs {
			for _, arg := range msg.Args {
				_, es := checkExpr(x, arg, nil)
				errs = append(errs, es...)
			}
		}
		return n, errs
	}

	for i := range n.Msgs {
		var in *TypeName
		if i == len(n.Msgs)-1 {
			in = infer
		}
		errs = append(errs, checkMsg(x, &n, &n.Msgs[i], in)...)
	}

	return n, errs
}

func checkMsg(x *scope, call *Call, msg *Msg, infer *TypeName) (errs []checkError) {
	defer x.tr("checkMsg(%s, infer=%s)", msg.Sel, infer)(&errs)

	fun, err := findFun(x, call, call.Recv, msg.Sel)
	if fun == nil || err != nil {
		if err != nil {
			errs = append(errs, *err)
		}
		// Best-effort checking of the arguments.
		for i := range msg.Args {
			if arg, es := msg.Args[i].check(x, nil); len(es) > 0 {
				errs = append(errs, es...)
			} else {
				msg.Args[i] = arg
			}
		}
		return errs
	}
	if fun.Recv != nil && len(fun.Recv.Parms) > 0 {
		if recv, ok := call.Recv.(Expr); ok && recv.ExprType() != nil {
			var es []checkError
			recvType := *recv.ExprType()
			if fun, es = fun.instRecv(x, recvType); len(es) > 0 {
				err := x.err(recvType, "%s cannot be instantiated", recvType)
				err.cause = es
				return append(errs, *err)
			}
		}
	}
	if len(fun.TypeParms) == 0 {
		return checkGroundedMsg(x, msg, fun, infer)
	}
	return checkLiftedMsg(x, msg, fun, infer)
}

func checkGroundedMsg(x *scope, msg *Msg, fun *Fun, infer *TypeName) (errs []checkError) {
	defer x.tr("checkGroundedMsg(%s, infer=%s)", msg.Sel, infer)(&errs)

	for i := range msg.Args {
		var in *TypeName
		if i < len(fun.Parms) {
			in = fun.Parms[i].Type
		}

		if arg, es := checkExpr(x, msg.Args[i], in); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			msg.Args[i] = arg
		}
	}
	if fun.Ret == nil {
		msg.Type = builtInType(x, "Nil")
	} else {
		msg.Type = fun.Ret
	}
	msg.Fun = fun
	return errs
}

func checkLiftedMsg(x *scope, msg *Msg, fun *Fun, infer *TypeName) (errs []checkError) {
	defer x.tr("checkLiftedMsg(%s, infer=%s)", msg.Sel, infer)(&errs)

	if errs = inferArgTypes(x, msg, fun, infer); len(errs) > 0 {
		// This is terminal; do best-effort checking
		// of all yet-to-be-checked arguments.
		for i := range fun.Parms {
			if hasVar(*fun.Parms[i].Type) {
				continue
			}
			arg, es := checkExpr(x, msg.Args[i], fun.Parms[i].Type)
			if len(es) > 0 {
				errs = append(errs, es...)
			} else {
				msg.Args[i] = arg
			}
		}
		return errs
	}

	fun = fun.sub(fun.x, fun.TypeArgs)
	if prev := x.funInsts[fun.String()]; prev == nil {
		x.funInsts[fun.String()] = fun
	} else {
		fun = prev
	}
	msg.Fun = fun

	for i := range fun.Parms {
		arg, es := checkExpr(x, msg.Args[i], fun.Parms[i].Type)
		if len(es) > 0 {
			errs = append(errs, es...)
			continue
		}
		msg.Args[i] = arg
	}

	return errs
}

// inferArgTypes checks the argument for any parameter with a type variable,
// unifies the type variables with the resulting expression types,
// and populates fun.TypeArgs with a *Parm→TypeName mapping,
// and fun.x with a scope from type variable names to *Parm.
func inferArgTypes(x *scope, msg *Msg, fun *Fun, infer *TypeName) (errs []checkError) {
	defer x.tr("inferArgTypes(%s, infer=%s)", msg.Sel, infer)(&errs)

	bind := make(map[string]TypeName)
	if fun.Ret != nil && hasVar(*fun.Ret) {
		in := infer
		if in == nil {
			in = builtInType(x, "Nil")
		}
		if err := unify(x, fun.Ret, in, bind); err != nil {
			errs = append(errs, *err)
		}
	}
	for i := range fun.Parms {
		p := &fun.Parms[i]
		if !hasVar(*p.Type) {
			continue
		}
		arg, es := msg.Args[i].check(x, nil)
		if len(es) > 0 {
			errs = append(errs, es...)
			continue
		}
		msg.Args[i] = arg
		if arg.ExprType() != nil {
			typ := arg.ExprType()
			typ.location = location{start: arg.Start(), end: arg.End()}
			for i := range typ.Args {
				typ.Args[i].location = typ.location
			}
			if err := unify(x, p.Type, typ, bind); err != nil {
				errs = append(errs, *err)
			}
		}
	}
	fun.x = x
	fun.TypeArgs = make(map[*Parm]TypeName)
	for i := range fun.TypeParms {
		p := &fun.TypeParms[i]
		n, ok := bind[p.Name]
		if !ok {
			x.log("p=%#v\n", p)
			x.log("bind=%v\n", bind)
			err := x.err(msg, "cannot infer type of type variable %s", p.Name)
			errs = append(errs, *err)
			continue
		}
		x.typeVars[p] = makeTypeVarType(p)
		fun.x = fun.x.push(p.Name, p)
		fun.TypeArgs[p] = n
		if p.Type != nil {
			errs = append(errs, checkTypeName(x, p.Type)...)
		}

	}
	return errs
}

func unify(x *scope, pat, typ *TypeName, bind map[string]TypeName) (err *checkError) {
	defer x.tr("unify(pat=%s, typ=%s)", pat, typ)(err)

	// The caller ensures this.
	if typ.Var || typ.Type == nil {
		panic(fmt.Sprintf("impossible: typ.Var=%v, typ.Type=%p", typ.Var, typ.Type))
	}

	if pat.Var {
		x.log("type variable %s", pat.Name)
		prev, ok := bind[pat.Name]
		if !ok {
			x.log("binding %s=%s", pat.Name, typ)
			bind[pat.Name] = *typ
			return nil
		}
		x.log("prev=%s (%p), typ=%s (%p)", prev, prev.Type, typ, typ.Type)
		if typ.Type != prev.Type {
			err = unifyError(x, &prev, typ)
			note(err, "%s bound to %s at %s", pat.Name, prev, x.loc(prev))
			return err
		}
		return nil
	}
	typRoot := rootUninstTypeDef(x, typ)
	patRoot := rootUninstTypeDef(x, pat)
	if typRoot != patRoot {
		// TODO: unify could implement interface inference.
		return unifyError(x, pat, typ)
	}
	if len(pat.Args) != len(typ.Args) {
		// They are the same Mod and same Name;
		// they must have the same number of args.
		panic("impossible")
	}
	var errs []checkError
	for i := range pat.Args {
		if err := unify(x, &pat.Args[i], &typ.Args[i], bind); err != nil {
			errs = append(errs, *err)
		}
	}
	if len(errs) > 0 {
		return unifyError(x, pat, typ, errs...)
	}
	return nil
}

// rootUninstTypeDef returns the type definition, following aliases.
// This is the uninstantiated definition.
// If there are errors looking up the Type, nil is returned;
// it is assumed that errors are reported elsewhere.
func rootUninstTypeDef(x *scope, name *TypeName) *Type {
	seen := make(map[Def]bool)
	for {
		var def Def
		defOrImport := x.mods.find(*name.Mod, name.Name)
		switch d := defOrImport.(type) {
		case builtin:
			def = d.Def
		case imported:
			def = d.Def
		case Def:
			def = d
		case nil:
			return nil
		}
		typ, ok := def.(*Type)
		if !ok {
			return nil
		}
		if typ.Alias != nil {
			if seen[def] {
				return nil // alias cycle
			}
			seen[def] = true
			name = typ.Alias
			continue
		}
		return typ
	}
}

func unifyError(x *scope, pat, typ *TypeName, cause ...checkError) *checkError {
	err := x.err(typ, "type %s cannot unify with %s",
		typeStringForUser(typ),
		typeStringForUser(pat))
	if len(cause) > 0 {
		err.cause = cause
	}
	return err
}

func hasVar(name TypeName) bool {
	if name.Var {
		return true
	}
	for _, a := range name.Args {
		if hasVar(a) {
			return true
		}
	}
	return false
}

func findFun(x *scope, loc, recv Node, sel string) (_ *Fun, err *checkError) {
	defer x.tr("findFun(%s)", sel)(err)

	var mps []*ModPath
	var name string
	var funMeth string
	switch recv := recv.(type) {
	case ModPath:
		mps = []*ModPath{&recv}
		name = sel
		funMeth = "function"
	case Expr:
		typ := recv.ExprType()
		if typ == nil || typ.Type == nil {
			return nil, nil
		}
		mps = []*ModPath{&typ.Type.ModPath, x.modPath()}
		name = typ.Type.Sig.Name + " " + sel
		funMeth = "method"
	default:
		panic(fmt.Sprintf("impossible receiver type: %T", recv))
	}
	var defOrImports []interface{}
	seen := make(map[interface{}]bool)
	for _, mp := range mps {
		x.log("looking for %s: [%s] %s", funMeth, mp, name)
		switch d := x.mods.find(*mp, name); {
		case d == nil:
			continue
		case seen[d]:
			continue
		default:
			x.log("found")
			seen[d] = true
			defOrImports = append(defOrImports, d)
		}
	}
	switch len(defOrImports) {
	case 1:
		break // good
	case 0:
		return nil, x.err(loc, "%s undefined", name)
	default:
		err := x.err(loc, "ambiguous call")
		for _, d := range defOrImports {
			addDefNotes(err, x, d)
		}
		return nil, err
	}
	var def Def
	switch d := defOrImports[0].(type) {
	case builtin:
		def = d.Def
	case imported:
		def = d.Def
	case Def:
		def = d
	default:
		panic(fmt.Sprintf("bad definition type: %T", d))
	}
	fun, ok := def.(*Fun)
	if !ok {
		err := x.err(loc, "got %s, expected a %s", def.kind(), funMeth)
		addDefNotes(err, x, defOrImports[0])
		return nil, err
	}
	return fun, nil
}

func (n Ctor) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Ctor.check(infer=%s)", infer)(&errs)

	errs = append(errs, checkTypeName(x, &n.Type)...)
	switch typ := n.Type.Type; {
	case typ == nil:
		break
	case typ.Alias != nil:
		panic("impossible") // aliases should already be forwarded
	case checkRefConvert(x, &n):
		break // ok
	case isAry(typ):
		errs = append(errs, checkAryCtor(x, &n)...)
	case len(typ.Cases) > 0:
		errs = append(errs, checkOrCtor(x, &n)...)
	case len(typ.Virts) > 0:
		errs = append(errs, checkVirtCtor(x, &n)...)
	case isBuiltInType(typ):
		switch {
		case !isRefType(typ):
			err := x.err(n, "built-in type %s cannot be constructed", typ.Name())
			errs = append(errs, *err)
		case len(n.Args) > 1 || strings.ContainsRune(n.Sel, ':'):
			err := x.err(n, "malformed reference conversion")
			errs = append(errs, *err)
		}
		for _, expr := range n.Args {
			_, es := checkExpr(x, expr, nil)
			errs = append(errs, es...)
		}
	default:
		errs = append(errs, checkAndCtor(x, &n)...)
	}

	return n, errs
}

func checkRefConvert(x *scope, n *Ctor) bool {
	defer x.tr("isRefConvert(…)")()

	if n.Type.Type == nil || len(n.Args) != 1 || strings.ContainsRune(n.Sel, ':') {
		x.log("n.Type.Type=%p, len(n.Args)=%d", n.Type.Type, len(n.Args))
		return false
	}
	expr, es := checkExpr(x, n.Args[0], nil)
	if len(es) > 0 {
		x.log("errors in checkExpr; not ref convert")
		return false
	}
	exprType := expr.ExprType()
	if exprType == nil || exprType.Type == nil {
		x.log("expr type is nil; not ref convert")
		return false
	}
	_, wantBase := refBaseType(n.Type.Type)
	_, gotBase := refBaseType(exprType.Type)
	if wantBase != gotBase {
		return false
	}
	n.Args[0] = expr
	return true
}

func refBaseType(typ *Type) (int, *Type) {
	var i int
	for isRefType(typ) {
		i++
		typ = typ.Sig.Args[&typ.Sig.Parms[0]].Type
	}
	return i, typ
}

func isRefType(typ *Type) bool {
	return isBuiltInType(typ) && typ.Sig.Name == "&"
}

func isAry(t *Type) bool {
	return isBuiltInType(t) && t.Sig.Name == "Array"
}

func isBuiltInType(t *Type) bool {
	return t.ModPath.Root == ""
}

func checkAryCtor(x *scope, n *Ctor) (errs []checkError) {
	defer x.tr("checkAryCtor(%s)", n.Type.Type.Sig.Name)(&errs)

	typ := n.Type.Type
	t := typ.Sig.Args[&typ.Sig.Parms[0]]
	elmType := &t
	x.log("elmType=%s", elmType)

	for i := range n.Args {
		if expr, es := checkExpr(x, n.Args[i], elmType); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Args[i] = expr
		}
	}
	return errs
}

func checkAndCtor(x *scope, n *Ctor) (errs []checkError) {
	defer x.tr("checkAndCtor(%s)", n.Type.Type.Sig.Name)(&errs)

	typ := n.Type.Type
	var sel string
	var inferTypes []*TypeName
	for _, p := range typ.Fields {
		sel += p.Name + ":"
		inferTypes = append(inferTypes, p.Type)
	}
	if sel != n.Sel {
		err := x.err(n, "bad and-type constructor: got %s, expected %s", n.Sel, sel)
		errs = append(errs, *err)
	}
	if n.Sel == "" && len(n.Args) > 0 {
		err := x.err(n, "bad and-type constructor: Nil with non-nil expression")
		errs = append(errs, *err)
	}
	if len(inferTypes) != len(n.Args) {
		inferTypes = make([]*TypeName, len(n.Args))
	}
	for i := range n.Args {
		if expr, es := checkExpr(x, n.Args[i], inferTypes[i]); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Args[i] = expr
		}
	}
	return errs
}

func checkOrCtor(x *scope, n *Ctor) (errs []checkError) {
	defer x.tr("checkOrCtor(%s)", n.Type.Type.Sig.Name)(&errs)
	typ := n.Type.Type
	var cas *Parm
	x.log("selector: [%s]", n.Sel)
	for i := range typ.Cases {
		c := &typ.Cases[i]
		name := c.Name
		if c.Type != nil {
			name += ":"
		}
		x.log("considering case: [%s]", name)
		if name == n.Sel {
			cas = c
		}
	}
	if cas == nil {
		err := x.err(n, "bad or-type constructor: no case %s", n.Sel)
		errs = append(errs, *err)
	}
	if cas != nil && cas.Type == nil {
		// The arg was just the case label.
		n.Args = nil
	}

	// Currently n.Args can never be >1 if cas != nil,
	// (cas.Name can contain no more than one :, and n.Sel has one : per-arg).
	// but we still want to check all args to at least report their errors.
	for i := range n.Args {
		var inferType *TypeName
		if i == 0 && cas != nil && cas.Type != nil {
			inferType = cas.Type
		}
		if expr, es := checkExpr(x, n.Args[i], inferType); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Args[i] = expr
		}
	}
	return errs
}

func checkVirtCtor(x *scope, n *Ctor) (errs []checkError) {
	defer x.tr("checkVirtCtor(%s)", n.Type.Type.Sig.Name)(&errs)

	infer := &n.Type
	if n.Sel != "" {
		infer = nil
		err := x.err(n, "a virtual conversion cannot have a selector")
		errs = append(errs, *err)
	}
	if len(n.Args) != 1 {
		infer = nil
		err := x.err(n, "a virtual conversion must have exactly one argument")
		errs = append(errs, *err)
	}

	for i := range n.Args {
		if expr, es := checkExpr(x, n.Args[i], infer); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Args[i] = expr
		}
	}

	// TODO: checkVirtCtor should verify that the type is convertable.

	return errs
}

func (n Block) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Block.check(infer=%s)", infer)(&errs)

	for i := range n.Parms {
		p := &n.Parms[i]
		if p.Type != nil {
			errs = append(errs, checkTypeName(x, p.Type)...)
		}
		x = x.push(p.Name, p)
	}

	resInfer := inferBlock(x, &n, infer)
	var resultType *TypeName
	if resInfer != nil {
		resultType = resInfer
	}
	if isNil(resInfer) {
		resInfer = nil
	}

	x = &scope{state: x.state, parent: x, block: &n}
	switch ss, es := checkStmts(x, n.Stmts, resInfer); {
	case es != nil:
		errs = append(errs, es...)
	case resultType == nil:
		n.Stmts = ss
		if t := lastStmtExprType(n.Stmts); t != nil && t.Type != nil {
			resultType = t
		}
	}
	if resultType == nil { // still nil?
		resultType = builtInType(x, "Nil")
	}

	var typeArgs []TypeName
	for _, p := range n.Parms {
		if p.Type == nil {
			err := x.err(p, "cannot infer block parameter type")
			errs = append(errs, *err)
			continue
		}
		typeArgs = append(typeArgs, *p.Type)
	}
	typeArgs = append(typeArgs, *resultType)

	if max := len(funTypeParms); len(n.Parms) > max {
		err := x.err(n, "too many block parameters (max %d)", max)
		errs = append(errs, *err)
		return n, errs
	}

	// The following condition can only be true
	// if there were un-inferable parameter types,
	// thus len(errs)>0, so it's OK for n.Type to be nil;
	// the type check will not be error-free.
	if len(typeArgs) == len(n.Parms)+1 {
		n.Type = builtInType(x, fmt.Sprintf("Fun%d", len(n.Parms)), typeArgs...)
	}
	return n, errs
}

func isNil(t *TypeName) bool {
	return t != nil && t.Type != nil && isBuiltInType(t.Type) && t.Type.Sig.Name == "Nil"
}

func inferBlock(x *scope, n *Block, infer *TypeName) *TypeName {
	if infer == nil || infer.Type == nil || infer.Type.ModPath.Root != "" ||
		!strings.HasPrefix(infer.Type.Sig.Name, "Fun") ||
		len(n.Parms) != len(infer.Type.Sig.Parms)-1 {
		return nil
	}
	sig := infer.Type.Sig
	for i := range n.Parms {
		if n.Parms[i].Type == nil {
			p := &sig.Parms[i]
			t := sig.Args[p]
			n.Parms[i].Type = &t
		}
	}
	p := &sig.Parms[len(sig.Parms)-1]
	t := sig.Args[p]
	return &t
}

func lastStmtExprType(ss []Stmt) *TypeName {
	if len(ss) == 0 {
		return nil
	}
	if e, ok := ss[len(ss)-1].(Expr); ok {
		return e.ExprType()
	}
	return nil
}

func (n Ident) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Ident.check(%s, infer=%s)", n.Text, infer)(&errs)
	if parm, ok := x.find(n.Text).(*Parm); ok {
		n.Parm = parm
		return n, errs
	}

	if fun := x.fun(); fun != nil && fun.RecvType != nil {
		if f := findField(fun.RecvType, n.Text); f != nil {
			n.Parm = f
			n.RecvType = fun.RecvType
			return n, errs
		}
	}

	def, defOrImport := findDef(x, n.Text)
	if def == nil {
		err := x.err(n, "undefined: %s", n.Text)
		errs = append(errs, *err)
		return n, errs
	}

	if v, ok := def.(*Var); ok {
		n.Var = v
		return n, errs
	}

	if fun, ok := def.(*Fun); ok {
		if len(fun.Parms) > 0 || fun.Recv != nil {
			// The name is just an ident,
			// it cannot have a receiver or params.
			panic("impossible")
		}
		loc := n.location
		call := &Call{
			location: loc,
			Msgs:     []Msg{{location: loc, Sel: n.Text}},
		}
		expr, es := call.check(x, infer)
		return expr, append(errs, es...)
	}

	err := x.err(n, "got %s, expected a variable or 0-ary function", def.kind())
	addDefNotes(err, x, defOrImport)
	errs = append(errs, *err)
	return n, errs
}

func findDef(x *scope, name string) (def Def, defOrImport interface{}) {
	defOrImport = x.mods.find(*x.modPath(), name)
	switch d := defOrImport.(type) {
	case nil:
		return nil, nil
	case builtin:
		return d.Def, defOrImport
	case imported:
		return d.Def, defOrImport
	case Def:
		return d, defOrImport
	default:
		panic(fmt.Sprintf("impossible definition type %T", def))
	}
}

func findField(typ *Type, name string) *Parm {
	for i := range typ.Fields {
		f := &typ.Fields[i]
		if f.Name == name {
			return f
		}
	}
	return nil
}

func (n Int) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Int.check(%s, infer=%s)", n.Text, infer)(&errs)

	n.Val = big.NewInt(0)
	if _, ok := n.Val.SetString(n.Text, 10); !ok {
		// If the int is syntactiacally valid,
		// it should be a valid Go int too.
		panic("impossible")
	}

	n.Type = *builtInType(x, "Int")
	if ok, signed, bits := isInt(infer); ok {
		x.log("isInt(%s): signed=%v, bits=%v", infer.Type.Sig.Name, signed, bits)
		n.Type = *infer
		n.BitLen = bits
		n.Signed = signed

		var zero big.Int
		if !signed && n.Val.Cmp(&zero) < 0 {
			err := x.err(n, "%s cannot represent %s: negative unsigned", infer.Name, n.Text)
			errs = append(errs, *err)
			return n, errs
		}
		if signed {
			bits-- // sign bit
		}
		min := big.NewInt(-(1 << uint(bits)))
		x.log("bits=%d, val.BitLen()=%d, val=%v, min=%v",
			bits, n.Val.BitLen(), n.Val, min)
		if n.Val.BitLen() > bits && (!signed || n.Val.Cmp(min) != 0) {
			err := x.err(n, "%s cannot represent %s: overflow", infer.Name, n.Text)
			errs = append(errs, *err)
			return n, errs
		}
		return n, nil
	}
	if isFloat(infer) {
		return Float{location: n.location, Text: n.Text}.check(x, infer)
	}
	return n, nil
}

func (n Float) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Float.check(…)")(&errs)

	n.Val = big.NewFloat(0)
	if _, _, err := n.Val.Parse(n.Text, 10); err != nil {
		// If the float is syntactiacally valid,
		// it should be a valid Go float too.
		panic("impossible: " + err.Error())
	}

	n.Type = *builtInType(x, "Float")
	if isFloat(infer) {
		x.log("isFloat(%s)", infer.Type.Sig.Name)
		n.Type = *infer
		return n, nil
	}
	if ok, _, _ := isInt(infer); ok {
		x.log("isInt(%s)", infer.Type.Sig.Name)
		var i big.Int
		if _, acc := n.Val.Int(&i); acc != big.Exact {
			err := x.err(n, "%s cannot represent %s: truncation", infer.Name, n.Text)
			errs = append(errs, *err)
			return n, errs
		}
		return Int{location: n.location, Text: i.String()}.check(x, infer)
	}
	return n, nil
}

func isFloat(t *TypeName) bool {
	if t == nil || t.Type == nil || t.Type.ModPath.Root != "" {
		return false
	}
	return t.Type.Sig.Name == "Float32" || t.Type.Sig.Name == "Float64"
}

func isInt(t *TypeName) (ok bool, signed bool, bits int) {
	if t == nil || t.Type == nil || t.Type.ModPath.Root != "" {
		return false, false, 0
	}
	if n, _ := fmt.Sscanf(t.Type.Sig.Name, "Uint%d", &bits); n == 1 {
		return true, false, bits
	}
	if n, _ := fmt.Sscanf(t.Type.Sig.Name, "Int%d", &bits); n == 1 {
		return true, true, bits
	}
	return false, false, 0
}

func (n Rune) check(x *scope, _ *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Rune.check(…)")(&errs)
	// TODO: Rune.check should error on invalid unicode codepoints.
	n.Type = *builtInType(x, "Rune")
	return n, nil
}

func (n String) check(x *scope, _ *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("String.check(…)")(&errs)
	n.Type = *builtInType(x, "String")
	return n, nil
}

func builtInType(x *scope, name string, args ...TypeName) *TypeName {
	tn := TypeName{
		Mod:  &ModPath{},
		Name: name,
		Args: args,
	}
	if errs := checkTypeName(x, &tn); len(errs) > 0 {
		panic(fmt.Sprintf("impossible error: %v", errs))
	}
	return &tn
}
