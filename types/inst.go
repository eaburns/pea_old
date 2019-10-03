package types

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
	}

	inst = *typ
	x.log("memoizing %s (%p)", inst, &inst)
	x.typeInsts[key] = &inst

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
