package types

import (
	"fmt"

	"github.com/eaburns/pea/ast"
)

func instType(x *scope, typ *Type, args []TypeName) (res *Type, errs []checkError) {
	defer x.tr("instType(%p %s, %v)", typ, typ, args)(&errs)
	defer func() { x.log("inst=%p", res) }()

	// We access typ.Alias and typ.Sig.Parms.
	// Both of these must be cycle free to guarantee
	// that they are populated by this call.
	// TODO: check typ.Sig.Parms cycle.
	if es := gatherDef(x, typ); es != nil {
		return nil, append(errs, es...)
	}

	if typ.Alias != nil {
		if typ.Alias.Type == nil {
			return nil, errs // error reported elsewhere
		}
		sub := subMap(typ.Parms, args)
		args = subTypeNames(x, map[*Type]*Type{}, sub, typ.Alias.Args)
		typ = typ.Alias.Type
		x.log("using alias type %s %p", typ, typ)
	}
	if len(args) == 0 {
		return typ, nil
	}

	// Instantiate using the original definition.
	typ = typ.Def

	for _, inst := range typ.Insts {
		if typeNamesEq(inst.Args, args) {
			x.log("found existing instance %p", inst)
			return inst, errs
		}
	}

	inst := new(Type)
	if file, ok := x.defFiles[typ]; ok {
		// The type was defined within this module.
		// It may not be fully gathered; we need to gather our new instance.
		//
		// Further, this call to gatherDef must make a complete *Type.
		// The only way an incomplete *Type would be made
		// is if we are currently gathering &inst previously on the call stack
		// and gatherDef returns true because x.gathered[&inst]=true.
		// However, if this were the case, x.typeInsts[key] above
		// would have had an entry, and we would have never gotten here.
		//
		// Lastly, call gatherDef, not gatherType, because gatherDef
		// fixes the scope to file-scope and does alias cycle checking.
		es := gatherDef(x, typ)
		errs = append(errs, es...)
		x.defFiles[inst] = file
		x.gathered[inst] = true
	}
	x.log("new instance %p", inst)
	*inst = *typ
	inst.Args = args
	inst.Insts = nil
	// add to typ.Insts before subTypeBody, so recursive insts find this inst.
	typ.Insts = append(typ.Insts, inst)
	sub := subMap(typ.Parms, args)
	subTypeBody(x, map[*Type]*Type{typ: inst}, sub, inst)
	return inst, errs
}

func instRecv(x *scope, recv *Type, fun *Fun) (_ *Fun, errs []checkError) {
	defer x.tr("instRecv(%s, %s)", recv, fun)(&errs)

	var sub map[*TypeVar]TypeName
	if fun.Recv.Type.Args != nil {
		sub = make(map[*TypeVar]TypeName)
		for i, arg := range recv.Args {
			switch parm := fun.Recv.Type.Args[i].Type; {
			case parm == nil || arg.Type == nil:
				continue
			case parm.Var != nil:
				sub[parm.Var] = arg
			case parm != arg.Type:
				err := x.err(arg, "type mismatch: have %s, want %s", arg.Type, parm)
				errs = append(errs, *err)
			}
		}
	} else {
		sub = subMap(fun.Recv.Type.Parms, recv.Args)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	for _, inst := range fun.Def.Insts {
		if len(fun.Def.TParms) > 0 && len(inst.TParms) == 0 {
			// This is a fully-instantiated function.
			// We only want a receiver-instantiated instance.
			continue
		}
		if typeNamesEq(inst.Recv.Args, recv.Args) {
			return inst, errs
		}
	}

	inst := subFun(x, make(map[*Type]*Type), sub, fun)
	inst.Def = fun.Def
	fun.Def.Insts = append(fun.Def.Insts, inst)
	inst.Insts = nil
	inst.Recv.Args = recv.Args
	return inst, errs
}

// instFun returns the *Fun instance; on error the *Fun is nil.
func instFun(x *scope, infer *Type, fun *Fun, msg *Msg) (_ *Fun, errs []checkError) {
	defer x.tr("instFun(infer=%s, %s, %s)", infer, fun, msg.Sel)(&errs)

	sub, errs := unifyFunTParms(x, infer, fun, msg)
	if len(errs) > 0 {
		return nil, errs
	}

	var notes []string
	args := make([]TypeName, len(fun.TParms))
	for i := range fun.TParms {
		tvar := &fun.TParms[i]
		var ok bool
		if args[i], ok = sub[tvar]; !ok {
			// TODO: Detect unused type vars at fun def and emit an error.
			// Currently the error will happen at the callsite,
			// but really this is an error in the def:
			// not all type vars are used.
			notes = append(notes, fmt.Sprintf("cannot infer type of %s", tvar.Name))
		}
	}
	if len(notes) > 0 {
		err := x.err(msg, "cannot infer type parameters of %s", msg.Sel)
		note(err, "%s", fun)
		err.notes = append(err.notes, notes...)
		errs = append(errs, *err)
		return nil, errs
	}

	for _, inst := range fun.Def.Insts {
		if (fun.Recv == nil || fun.Recv.Type == inst.Recv.Type) &&
			typeNamesEq(inst.TArgs, args) {
			return inst, errs
		}
	}

	inst := subFun(x, make(map[*Type]*Type), sub, fun)
	inst.Def = fun.Def
	fun.Def.Insts = append(fun.Def.Insts, inst)
	inst.TParms = nil // all should be subbed
	inst.TArgs = args
	return inst, nil
}

// unifyFunTParms sets msg.Args for each arg passed to
// a fun param with a type variable in its type.
// The rest of msg.Args are left nil.
func unifyFunTParms(x *scope, infer *Type, fun *Fun, msg *Msg) (sub map[*TypeVar]TypeName, errs []checkError) {
	defer x.tr("unifyFunTParms(infer=%s, %s, %s)", infer, fun, msg.Sel)(&errs)
	defer func() { x.log("sub=%s", subDebugString(sub)) }()

	sub = make(map[*TypeVar]TypeName)
	tparms := make(map[*TypeVar]bool)
	for i := range fun.TParms {
		tparms[&fun.TParms[i]] = true
	}

	if fun.Sig.Ret != nil && fun.Sig.Ret.Type != nil && infer != nil && hasTParm(tparms, fun.Sig.Ret) {
		x.log("unify return")
		// TODO: Expr.Type() should return a TypeName.
		// Until then, create a transient TypeName so unify
		// has a locatable node to use for error reporting.
		inferName := makeTypeName(infer)
		inferName.AST = msg.AST
		if err := unify(x, fun.Sig.Ret, inferName, tparms, sub); err != nil {
			errs = append(errs, *err)
		}
	}

	parms := fun.Sig.Parms
	if fun.Recv != nil {
		parms = parms[1:]
	}
	seen := make(map[*Type]*Type)
	for i := range parms {
		parm := &parms[i]
		x.log("%s parm %d %s %s", msg.Sel, i, parm.Name, parm.TypeName)
		tname := subTypeName(x, seen, sub, parm.TypeName)
		x.log("subbed name: %s", tname)
		if !hasTParm(tparms, tname) {
			continue
		}
		// The type assertion to *ast.Msg is OK,
		// since Msg.AST is only not a *ast.Msg
		// for a 0-ary function call, but this has args.
		arg, es := checkExpr(x, nil, msg.AST.(*ast.Msg).Args[i])
		errs = append(errs, es...)
		msg.Args[i] = arg
		if arg.Type() == nil {
			continue
		}
		argTypeName := makeTypeName(arg.Type())
		argTypeName.AST = arg.ast()
		if err := unify(x, tname, argTypeName, tparms, sub); err != nil {
			errs = append(errs, *err)
		}
	}
	return sub, errs
}

func hasTParm(tparms map[*TypeVar]bool, name *TypeName) bool {
	if name.Type == nil {
		return false
	}
	if name.Type.Var != nil && tparms[name.Type.Var] {
		return true
	}
	for i := range name.Args {
		if hasTParm(tparms, &name.Args[i]) {
			return true
		}
	}
	return false
}

func unify(x *scope, pat, typ *TypeName, tparms map[*TypeVar]bool, sub map[*TypeVar]TypeName) (err *checkError) {
	defer x.tr("unify(%s, %s, sub=%s)", pat, typ, subDebugString(sub))(err)

	if tparms[pat.Type.Var] {
		x.log("parm %s", pat.Type)
		prev, ok := sub[pat.Type.Var]
		if !ok {
			x.log("binding %s (%p) → %s (%p)", pat.Type, pat.Type.Var, typ, typ.Type)
			sub[pat.Type.Var] = *typ
			return nil
		}
		x.log("prev=%s", prev)
		if prev.Type != typ.Type {
			err = x.err(typ, "cannot bind %s to %s: already bound", typ, pat.Name)
			note(err, "previous binding to %s at %s", prev, x.loc(prev))
			return err
		}
		return nil
	}

	if pat.Type.Mod != typ.Type.Mod ||
		pat.Type.Name != typ.Type.Name ||
		pat.Type.Arity != typ.Type.Arity {
		return x.err(typ, "type mismatch: have %s, want %s", typ.name(), pat.name())
	}
	var errs []checkError
	for i := range pat.Type.Args {
		patArg := &pat.Type.Args[i]
		typArg := &typ.Type.Args[i]
		if e := unify(x, patArg, typArg, tparms, sub); e != nil {
			if e.cause != nil {
				errs = append(errs, e.cause...)
			} else {
				errs = append(errs, *e)
			}
		}
	}
	if len(errs) > 0 {
		err = x.err(typ, "%s cannot unify with %s", typ, pat)
		err.cause = errs
		return err
	}
	return nil
}

func typeNamesEq(as, bs []TypeName) bool {
	if len(as) != len(bs) {
		return false
	}
	for i := range as {
		if as[i].Type != bs[i].Type {
			return false
		}
	}
	return true
}

func subMap(parms []TypeVar, args []TypeName) map[*TypeVar]TypeName {
	sub := make(map[*TypeVar]TypeName)
	for i := range parms {
		sub[&parms[i]] = args[i]
	}
	return sub
}
