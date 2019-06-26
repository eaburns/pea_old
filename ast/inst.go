package ast

import (
	"fmt"
	"strings"
)

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
	// The key for a function instance is the modpath of the caller
	// plus the signature of the function itself. The caller modpath
	// ensures that each unique set of available methods on the
	// type arguments will get a unique instance. Currently,
	// the method set in the same modpath is unique.
	//
	// TODO: fun inst keys should be computed using the type parameter methods, not the modpath.
	// Modpath is sufficient, but not necessary to determine unique sets of type argument methods. Two different modpaths could (and likely will) have the same set of methods; with the current approach, they will get different instances. This is wasteful. We should do the lookups to see, for each constraint method, which Fun* is actually going to be used. Then make the key unique based on this information. Then we will only have different instances if the actual methods will differ.
	key := [2]string{x.modPath().String(), fun.String()}
	if prev := x.funInsts[key]; prev == nil {
		x.funInsts[key] = fun
		if x.discardInsts == 0 {
			x.log("adding a function inst")
			x.mod.Insts = append(x.mod.Insts, fun)
		}
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

func (n Fun) instRecv(x *scope, typ TypeName) (_ *Fun, errs []checkError) {
	defer x.tr("Fun.instRecv(%s, %s)", n.Name(), typ)(errs)
	x = x.root()
	if n.Recv == nil {
		return &n, nil
	}

	sig, errs := instTypeSig(x, &n, *n.Recv, typ)
	if len(errs) > 0 {
		return &n, errs
	}
	n.Recv = &sig
	nInst := n.sub(n.Recv.x, n.Recv.Args)
	return nInst, nil
}

func (n Fun) sub(x *scope, sub map[*Parm]TypeName) *Fun {
	n.TypeParms = subParms(x, sub, n.TypeParms)
	n.Parms = subParms(x, sub, n.Parms)
	if n.Ret != nil {
		n.Ret = subTypeName(x, sub, *n.Ret)
	}
	n.Stmts = subStmts(x, sub, n.Stmts)
	return &n
}

func (n Type) inst(x *scope, typ TypeName) (_ *Type, errs []checkError) {
	defer x.tr("Type.inst(%s, %s)", n.Name(), typ)(errs)
	x = x.root()

	key, ok := typeNameKey(typ)
	if !ok {
		x.log("bad key")
		return nil, nil // error reported elsewhere
	}
	x.log("looking for memoized type [%s]", key)
	switch typeOrErrs := x.typeInsts[key].(type) {
	case nil:
		x.log("not found")
		break
	case *Type:
		x.log("found %p", typeOrErrs)
		return typeOrErrs, nil
	case []checkError:
		x.log("found errors")
		return nil, typeOrErrs
	}

	// Memoize the type before substituting,
	// so that recursive calls to inst within substitution
	// will simply refer to this type.
	x.log("memoizing %p", &n)
	x.typeInsts[key] = &n

	n.Sig, errs = instTypeSig(x, &n, n.Sig, typ)
	if len(errs) != 0 {
		x.log("memoizing errors")
		x.typeInsts[key] = errs
		return &n, errs
	}
	if x.discardInsts == 0 {
		x.log("adding a type inst")
		x.mod.Insts = append(x.mod.Insts, &n)
	}
	switch {
	case n.Alias != nil:
		n.Alias = subTypeName(n.Sig.x, n.Sig.Args, *n.Alias)
	case n.Fields != nil:
		n.Fields = subParms(n.Sig.x, n.Sig.Args, n.Fields)
	case n.Cases != nil:
		n.Cases = subParms(n.Sig.x, n.Sig.Args, n.Cases)
	case n.Virts != nil:
		n.Virts = subMethSigs(n.Sig.x, n.Sig.Args, n.Virts)
	}
	return &n, nil
}

func typeNameKey(n TypeName) (string, bool) {
	var s strings.Builder
	if !buildTypeNameKey(n, &s) {
		return "", false
	}
	return s.String(), true
}

func buildTypeNameKey(n TypeName, s *strings.Builder) bool {
	if n.Type == nil {
		return false
	}
	if len(n.Args) > 0 {
		s.WriteRune('(')
		for _, a := range n.Args {
			if !buildTypeNameKey(a, s) {
				return false
			}
		}
		s.WriteRune(')')
	}
	s.WriteString(n.Type.Name())
	return true
}

func typeOK(n TypeName) bool {
	if n.Type == nil {
		return false
	}
	for _, a := range n.Args {
		if !typeOK(a) {
			return false
		}
	}
	return true
}

func instTypeSig(x *scope, def Def, sig TypeSig, name TypeName) (_ TypeSig, errs []checkError) {
	defer x.tr("instTypeSig(%s, %s)", sig, name)(errs)
	ps := sig.Parms
	if len(ps) != len(name.Args) {
		err := x.err(name, "argument count mismatch: got %d, expected %d",
			len(name.Args), len(ps))
		addDefNotes(err, x, x.mods.find(*name.Mod, name.Name))
		errs = append(errs, *err)
		return sig, errs
	}

	// TODO: instTypeSig doesn't check type constraints.

	sig.x = x
	sig.Args = make(map[*Parm]TypeName)
	sig.Parms = make([]Parm, len(sig.Parms))
	for i := range ps {
		p := &sig.Parms[i]
		*p = ps[i]
		if p.Type != nil {
			p.Type = subTypeName(x, sig.Args, *p.Type)
		}
		sig.x = sig.x.push(p.Name, p)
		sig.Args[p] = name.Args[i]
	}
	return sig, nil
}

func (n Call) sub(x *scope, sub map[*Parm]TypeName) Expr {
	defer x.tr("Call.sub(…)")()
	switch r := n.Recv.(type) {
	case nil:
		break
	case Expr:
		n.Recv = r.sub(x, sub)
	case ModPath:
		n.Recv = r.sub(x, sub)
	}
	var msgs []Msg
	for _, m := range n.Msgs {
		m.Args = subExprs(x, sub, m.Args)
		msgs = append(msgs, m)
	}
	n.Msgs = msgs
	return n
}

func (n Ctor) sub(x *scope, sub map[*Parm]TypeName) Expr {
	defer x.tr("Ctor.sub(…)")()
	n.Type = *subTypeName(x, sub, n.Type)
	n.Args = subExprs(x, sub, n.Args)
	return n
}

func (n Block) sub(x *scope, sub map[*Parm]TypeName) Expr {
	defer x.tr("Block.sub(…)")()
	n.Parms = subParms(x, sub, n.Parms)
	n.Stmts = subStmts(x, sub, n.Stmts)
	return n
}

func (n ModPath) sub(*scope, map[*Parm]TypeName) Node { return n }

func (n Ident) sub(x *scope, sub map[*Parm]TypeName) Expr { return n }

func (n Int) sub(*scope, map[*Parm]TypeName) Expr { return n }

func (n Float) sub(*scope, map[*Parm]TypeName) Expr { return n }

func (n Rune) sub(*scope, map[*Parm]TypeName) Expr { return n }

func (n String) sub(*scope, map[*Parm]TypeName) Expr { return n }

func subMethSigs(x *scope, sub map[*Parm]TypeName, in []MethSig) []MethSig {
	defer x.tr("subMethSigs(…)")()
	out := make([]MethSig, len(in))
	for i := range in {
		out[i] = in[i]
		if out[i].Ret != nil {
			out[i].Ret = subTypeName(x, sub, *out[i].Ret)
		}
		if out[i].Parms == nil {
			continue
		}
		parms := make([]TypeName, len(out[i].Parms))
		for j := range out[i].Parms {
			parms[j] = *subTypeName(x, sub, out[i].Parms[j])
		}
		out[i].Parms = parms
	}
	return out
}

func subParms(x *scope, sub map[*Parm]TypeName, in []Parm) []Parm {
	defer x.tr("subParms()")()
	if in == nil {
		return nil
	}
	out := make([]Parm, len(in))
	for i, p := range in {
		if p.Type != nil {
			p.Type = subTypeName(x, sub, *p.Type)
		}
		out[i] = p
	}
	return out
}

// subTypeName always returns non-nil.
func subTypeName(x *scope, sub map[*Parm]TypeName, n TypeName) *TypeName {
	defer x.tr("subTypeName(%v, %s)", sub, n)()
	if n.Var {
		d := x.find(n.Name)
		if d == nil {
			x.log("no definition")
			return &n
		}
		p, ok := d.(*Parm)
		if !ok {
			x.log("non-param definition")
			return &n
		}
		if s, ok := sub[p]; ok {
			x.log("sub %s → %s", n.Name, s)
			return &s
		}
		x.log("no sub")
		return &n
	}
	if n.Args == nil {
		return &n
	}
	args := make([]TypeName, len(n.Args))
	for i := range n.Args {
		args[i] = *subTypeName(x, sub, n.Args[i])
	}
	n.Args = args

	if n.Type != nil {
		if typ, es := n.Type.inst(x, n); len(es) > 0 {
			panic(fmt.Sprintf("impossible: %s, %s, %v", n.Type, n, es))
		} else {
			n.Type = typ
		}
	}

	return &n
}

func subStmts(x *scope, sub map[*Parm]TypeName, in []Stmt) []Stmt {
	defer x.tr("subStmts()")()
	if in == nil {
		return nil
	}
	out := make([]Stmt, len(in))
	for i := range in {
		switch s := in[i].(type) {
		case *Ret:
			up := x.tr("sub ret")
			s.Val = s.Val.sub(x, sub)
			out[i] = s
			up()
		case *Assign:
			up := x.tr("sub assign")
			s.Val = s.Val.sub(x, sub)
			out[i] = s
			up()
		case Expr:
			out[i] = s.sub(x, sub)
		default:
			panic(fmt.Sprintf("impossible statement %T", s))
		}
	}
	return out
}

func subExprs(x *scope, sub map[*Parm]TypeName, in []Expr) []Expr {
	defer x.tr("subExprs()")()
	if in == nil {
		return nil
	}
	out := make([]Expr, len(in))
	for i, e := range in {
		out[i] = e.sub(x, sub)
	}
	return out
}
