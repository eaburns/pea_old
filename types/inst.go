package types

import (
	"fmt"

	"github.com/eaburns/pea/ast"
)

type typeKey struct {
	// Either mod+name+args or Var.

	mod  string
	name string
	args interface{}

	Var *TypeVar
}

func makeTypeKey(sig *Type) typeKey {
	k := typeKey{mod: sig.Mod, name: sig.Name}
	for i := len(sig.Parms) - 1; i >= 0; i-- {
		k.args = argsKey{
			typ:  typeKey{Var: &sig.Parms[i]},
			next: k.args,
		}
	}
	return k
}

func makeTypeNameKey(mod, name string, args []TypeName) typeKey {
	return typeKey{mod: mod, name: name, args: makeArgsKey(args)}
}

type argsKey struct {
	typ  typeKey
	next interface{}
}

func makeArgsKey(args []TypeName) interface{} {
	if len(args) == 0 {
		return nil
	}
	var tkey typeKey
	switch a := args[0]; {
	case a.Type == nil:
		// This case indicates an error somwhere in the args.
		// The error was reported elsewhere; just use the empty key.
		break
	case a.Type.Var != nil:
		tkey = typeKey{Var: a.Type.Var}
	default:
		tkey = makeTypeNameKey(a.Type.Mod, a.Type.Name, a.Args)
	}
	return argsKey{typ: tkey, next: makeArgsKey(args[1:])}
}

type recvKey struct {
	recvType typeKey
	sel      string
}

func makeRecvKey(fun *Fun, args []TypeName) recvKey {
	r := fun.Recv.Type
	return recvKey{
		recvType: makeTypeNameKey(r.Mod, r.Name, args),
		sel:      fun.Sig.Sel,
	}
}

type funKey struct {
	recv typeKey
	sel  string
	args interface{}
}

func makeFunKey(recv *Recv, sel string, args []TypeName) funKey {
	var recvKey typeKey
	if recv != nil && recv.Type != nil {
		recvKey = makeTypeKey(recv.Type)
	}
	return funKey{
		recv: recvKey,
		sel:  sel,
		args: makeArgsKey(args),
	}
}

func instType(x *scope, typ *Type, args []TypeName) (res *Type, errs []checkError) {
	defer x.tr("instType(%p %s, %v)", typ, typ, args)(&errs)
	defer func() { x.log("inst=%p", res) }()

	if t, ok := x.origTypeDef[typ]; ok {
		x.log("original type: %s (%p)", t, t)
		typ = t
	}

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
		sub := make(map[*TypeVar]TypeName)
		for i := range typ.Parms {
			sub[&typ.Parms[i]] = args[i]
		}
		seen := make(map[*Type]*Type)
		args = subTypeNames(x, seen, sub, typ.Alias.Args)
		typ = typ.Alias.Type
	}
	if len(args) == 0 {
		return typ, nil
	}

	key := makeTypeNameKey(typ.Mod, typ.Name, args)
	if inst, ok := x.typeInsts[key]; ok {
		return inst, nil
	}

	var inst Type
	if file, ok := x.defFiles[typ]; ok {
		x.defFiles[&inst] = file
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

		// Mark our instance as now gathered
		// since it will be subbed from
		// a fully-gathered type definition.
		x.gathered[&inst] = true
	}

	inst = *typ
	x.log("memoizing %s (%p)", inst, &inst)
	x.typeInsts[key] = &inst
	x.origTypeDef[&inst] = typ

	sub := make(map[*TypeVar]TypeName)
	for i := range inst.Parms {
		sub[&inst.Parms[i]] = args[i]
	}

	seen := make(map[*Type]*Type)
	seen[typ] = &inst
	subTypeBody(x, seen, sub, &inst)
	inst.Parms = nil
	inst.Args = args
	return &inst, errs
}

func instRecv(x *scope, recv *Type, fun *Fun) (_ *Fun, errs []checkError) {
	defer x.tr("instRecv(%s, %s)", recv, fun)(&errs)

	sub := make(map[*TypeVar]TypeName)
	if fun.Recv.Type.Parms != nil {
		for i, arg := range recv.Args {
			parm := &fun.Recv.Type.Parms[i]
			sub[parm] = arg
		}
	} else {
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
	}
	if len(errs) > 0 {
		return nil, errs
	}

	key := makeRecvKey(fun, recv.Args)
	if inst := x.recvInsts[key]; inst != nil {
		return inst, errs
	}
	inst := subFun(x, make(map[*Type]*Type), sub, fun)
	x.recvInsts[key] = inst
	// Save the original function definition so that later we can find it
	// in order to substitute the statements of this definition.
	if orig, ok := x.origFunDef[fun]; ok {
		x.origFunDef[inst] = orig
	} else {
		x.origFunDef[inst] = fun
	}
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
			// TODO: Detect unused type vars at function def and emit an error.
			// Currently the error will happen at the callsite,
			// but really this is an error in the def:
			// not all type vars are used.
			x.log("var=%p", tvar)
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

	key := makeFunKey(fun.Recv, fun.Sig.Sel, args)
	if inst := x.funInsts[key]; inst != nil {
		return inst, nil
	}
	inst := subFun(x, make(map[*Type]*Type), sub, fun)
	inst.TParms = nil // all should be subbed
	x.funInsts[key] = inst
	// Save the original function definition so that later we can find it
	// in order to substitute the statements of this definition.
	if orig, ok := x.origFunDef[fun]; ok {
		x.origFunDef[inst] = orig
	} else {
		x.origFunDef[inst] = fun
	}
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
