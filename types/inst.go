package types

import (
	"fmt"

	"github.com/eaburns/pea/ast"
)

func instDefTypes(x *scope, defs []Def) []checkError {
	var errs []checkError
	for _, def := range defs {
		switch def := def.(type) {
		case *Val:
			errs = append(errs, instValTypes(x, def)...)
		case *Fun:
			errs = append(errs, instFunTypes(x, def)...)
		case *Type:
			errs = append(errs, instTypeTypes(x, def)...)
		default:
			panic(fmt.Sprintf("impossible type: %T", def))
		}
	}
	return errs
}

func instValTypes(x *scope, val *Val) (errs []checkError) {
	defer x.tr("instValTypes(%s)", val)(&errs)
	if val.Var.TypeName != nil {
		errs = instTypeName(x, val.Var.TypeName)
		val.Var.typ = val.Var.TypeName.Type
	}
	return errs
}

func instFunTypes(x *scope, fun *Fun) (errs []checkError) {
	defer x.tr("instFunTypes(%s)", fun)(&errs)
	if fun.Recv != nil {
		errs = append(errs, instRecvTypes(x, fun.Recv)...)
		// Set self parameter type.
		selfBaseTypeName := TypeName{
			AST:  fun.Recv.AST,
			Mod:  fun.Recv.Mod,
			Name: fun.Recv.Name,
			Type: fun.Recv.Type,
		}
		if fun.Recv.Type != nil {
			selfBaseTypeName.Args = fun.Recv.Type.Args
		}
		selfType := builtInType(x, "&", selfBaseTypeName)
		if fun.Sig.Parms[0].Name != "self" {
			panic("impossible")
		}
		fun.Sig.Parms[0].TypeName = makeTypeName(selfType)
		fun.Sig.Parms[0].typ = selfType
	}
	errs = append(errs, instTypeParamTypes(x, fun.TParms)...)
	errs = append(errs, instFunSigTypes(x, &fun.Sig)...)
	return errs
}

func instRecvTypes(x *scope, recv *Recv) (errs []checkError) {
	defer x.tr("instRecvTypes(%s)", recv.Name)(&errs)
	errs = append(errs, instTypeParamTypes(x, recv.Parms)...)
	errs = append(errs, instTypeNames(x, recv.Args)...)

	if recv.Type == nil {
		return errs
	}

	errs = append(errs, instTypeTypes(x, recv.Type)...)

	args := make([]TypeName, len(recv.Parms))
	for i := range recv.Parms {
		parm := &recv.Parms[i]
		args[i] = TypeName{
			AST:  parm.AST,
			Name: parm.Name,
			Type: parm.Type,
		}
	}
	var es []checkError
	recv.Type, es = instType(x, recv.Type, args)
	x.log("recv.Type=%s", recv.Type)
	errs = append(errs, es...)
	x.log("instantiated recv type %s", recv.Type)

	return errs
}

func instTypeParamTypes(x *scope, parms []TypeVar) (errs []checkError) {
	defer x.tr("instTypeParamTypes(…)")(&errs)
	for i := range parms {
		vr := &parms[i]
		for j := range vr.Ifaces {
			iface := &vr.Ifaces[j]
			errs = append(errs, instTypeName(x, iface)...)
		}
	}
	return errs
}

func instFunSigTypes(x *scope, sig *FunSig) (errs []checkError) {
	defer x.tr("instFunSigTypes(%s)", sig.Sel)(&errs)
	errs = append(errs, instVarTypes(x, sig.Parms)...)
	if sig.Ret != nil {
		errs = append(errs, instTypeName(x, sig.Ret)...)
	}
	return errs
}

func instVarTypes(x *scope, vars []Var) (errs []checkError) {
	defer x.tr("instVarTypes(…)")(&errs)
	for i := range vars {
		v := &vars[i]
		x.log("instantiating var %s", v.Name)
		if v.TypeName == nil {
			continue
		}
		errs = append(errs, instTypeName(x, v.TypeName)...)
		v.typ = v.TypeName.Type
		x.log("var %s type %s", v.Name, v.Type())
	}
	return errs
}

func instTypeTypes(x *scope, typ *Type) (errs []checkError) {
	defer x.tr("instTypeTypes(%s)", typ)(&errs)

	if x.insted[typ] {
		return nil
	}
	x.insted[typ] = true

	errs = append(errs, instTypeParamTypes(x, typ.Parms)...)

	switch {
	case typ.Var != nil:
		// nothing to do
	case typ.Alias != nil:
		errs = append(errs, instTypeName(x, typ.Alias)...)
	case len(typ.Fields) > 0:
		errs = append(errs, instVarTypes(x, typ.Fields)...)
	case len(typ.Cases) > 0:
		errs = append(errs, instVarTypes(x, typ.Cases)...)
	case len(typ.Virts) > 0:
		for i := range typ.Virts {
			errs = append(errs, instFunSigTypes(x, &typ.Virts[i])...)
		}
	}
	return errs
}

func instTypeNames(x *scope, names []TypeName) (errs []checkError) {
	defer x.tr("instTypeNames(…)")(&errs)
	for i := range names {
		errs = append(errs, instTypeName(x, &names[i])...)
	}
	return errs
}

func instTypeName(x *scope, name *TypeName) (errs []checkError) {
	defer x.tr("instTypeName(%s)", name)(&errs)

	for i := range name.Args {
		errs = append(errs, instTypeName(x, &name.Args[i])...)
	}
	if name.Type == nil || len(name.Type.Args) > 0 {
		return errs
	}
	errs = append(errs, instTypeTypes(x, name.Type)...)
	var es []checkError
	name.Type, es = instType(x, name.Type, name.Args)
	return append(errs, es...)
}

func instType(x *scope, typ *Type, args []TypeName) (res *Type, errs []checkError) {
	defer x.tr("instType(%p %s, %v)", typ, typ, args)(&errs)
	defer func() { x.log("inst: %s (%p)", res, res) }()

	if typ.Alias != nil {
		if typ.Alias.Type == nil {
			return nil, errs // error reported elsewhere
		}
		sub := newSubMap(typ.Parms, args)
		errs = append(errs, instTypeName(x, typ.Alias)...)
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
	x.log("new instance %p", inst)
	*inst = *typ
	inst.Args = args
	inst.Insts = nil
	// add to typ.Insts before subTypeBody, so recursive insts find this inst.
	typ.Insts = append(typ.Insts, inst)
	sub := newSubMap(typ.Parms, args)
	subTypeBody(x, map[*Type]*Type{typ: inst}, sub, inst)
	x.log("subed: %s", inst.fullString())
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
		sub = newSubMap(fun.Recv.Type.Parms, recv.Args)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	file := x.curFile()
	for _, inst := range file.funInsts {
		if inst.Def != fun.Def {
			continue
		}
		if len(inst.TArgs) > 0 {
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
	inst.Recv.Type = recv
	inst.Recv.Args = recv.Args
	file.funInsts = append(file.funInsts, inst)
	fun.Def.Insts = append(fun.Def.Insts, inst)
	x.funTodo = append(x.funTodo, funFile{fun: inst, file: file})
	return inst, errs
}

type argTypes interface {
	ast() ast.Node
	arg(*scope, int) (*Type, ast.Node, []checkError)
}

func (m *Msg) arg(x *scope, i int) (*Type, ast.Node, []checkError) {
	// The type assertion to *ast.Msg is OK,
	// since Msg.AST is only not a *ast.Msg
	// for a 0-ary function call, but this has args.
	arg, errs := checkExpr(x, nil, m.AST.(*ast.Msg).Args[i])
	m.Args[i] = arg
	return arg.Type(), arg.ast(), errs
}

type funSigArgTypes struct {
	loc ast.Node
	sig *FunSig
}

func (s funSigArgTypes) ast() ast.Node { return s.loc }

func (s funSigArgTypes) arg(x *scope, i int) (*Type, ast.Node, []checkError) {
	return s.sig.Parms[i].Type(), s.sig.Parms[i].AST, nil
}

// instFun returns the *Fun instance; on error the *Fun is nil.
func instFun(x *scope, infer *Type, fun *Fun, argTypes argTypes) (_ *Fun, errs []checkError) {
	defer x.tr("instFun(infer=%s, %s)", infer, fun)(&errs)

	sub, errs := unifyFunTParms(x, infer, fun, argTypes)
	if len(errs) > 0 {
		return nil, errs
	}

	var notes []string
	args := make([]TypeName, len(fun.TParms))
	for i := range fun.TParms {
		args[i] = sub[&fun.TParms[i]]
	}
	if len(notes) > 0 {
		err := x.err(argTypes.ast(), "cannot infer type parameters of %s", fun.Sig.Sel)
		note(err, "%s", fun)
		err.notes = append(err.notes, notes...)
		errs = append(errs, *err)
		return nil, errs
	}

	file := x.curFile()
	for _, inst := range file.funInsts {
		if inst.Def != fun.Def {
			continue
		}
		if (fun.Recv == nil || fun.Recv.Type == inst.Recv.Type) &&
			typeNamesEq(inst.TArgs, args) {
			return inst, errs
		}
	}

	inst := subFun(x, make(map[*Type]*Type), sub, fun)
	inst.Def = fun.Def
	inst.TArgs = args
	file.funInsts = append(file.funInsts, inst)
	fun.Def.Insts = append(fun.Def.Insts, inst)
	x.funTodo = append(x.funTodo, funFile{fun: inst, file: file})
	return inst, nil
}

// unifyFunTParms sets msg.Args for each arg passed to
// a fun param with a type variable in its type.
// The rest of msg.Args are left nil.
func unifyFunTParms(x *scope, infer *Type, fun *Fun, argTypes argTypes) (sub map[*TypeVar]TypeName, errs []checkError) {
	defer x.tr("unifyFunTParms(infer=%s, %s)", infer, fun)(&errs)
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
		inferName.AST = argTypes.ast()
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
		x.log("%s parm %d %s %s", fun.Sig.Sel, i, parm.Name, parm.TypeName)
		tname := subTypeName(x, seen, sub, parm.TypeName)
		x.log("subbed name: %s", tname)
		if !hasTParm(tparms, tname) {
			continue
		}
		argType, argAST, es := argTypes.arg(x, i)
		if len(es) > 0 {
			errs = append(errs, es...)
			continue
		}
		if argType == nil {
			continue
		}
		argTypeName := makeTypeName(argType)
		argTypeName.AST = argAST
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

// TODO: unify should handle the case that typ.AST is nil.
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

func instFunBodies(x *state) {
	for len(x.funTodo) > 0 {
		funFile := x.funTodo[len(x.funTodo)-1]
		x.funTodo = x.funTodo[:len(x.funTodo)-1]
		instFunBody(funFile.file.x, funFile.fun)
	}
}

func rmLiftedFunInsts(defs []Def) {
	for _, def := range defs {
		fun, ok := def.(*Fun)
		if !ok {
			continue
		}
		var insts []*Fun
		for _, inst := range fun.Insts {
			if isGroundFun(inst) {
				insts = append(insts, inst)
			}
		}
		fun.Insts = insts
	}
}

func instFunBody(x *scope, fun *Fun) {
	defer x.tr("instFunStmts(%s)", fun)()

	if x.defFiles[fun.Def] == nil {
		x.log("skipping built-in")
		return
	}
	if !isGroundFun(fun) {
		x.log("skipping lifted instance: %s", fun)
		return
	}
	if fun.Def.Stmts == nil {
		// This is as declaration; it's instance should be too.
		fun.Stmts = nil
		return
	}

	// Setup the scope, because subStmts will do scope lookups.
	x = x.new()
	x.def = fun
	if fun.Recv != nil {
		for i := range fun.Recv.Parms {
			x = x.new()
			x.typeVar = fun.Recv.Parms[i].Type
		}
	}
	for i := range fun.TParms {
		x = x.new()
		x.typeVar = fun.TParms[i].Type
	}
	x = x.new()
	x.fun = fun
	for i := range fun.Sig.Parms {
		x = x.new()
		x.variable = &fun.Sig.Parms[i]
	}
	for i := range fun.Locals {
		x = x.new()
		x.variable = fun.Locals[i]
	}

	sub := newSubMap(fun.Def.TParms, fun.TArgs)
	if fun.Def.Recv != nil {
		addSubMap(fun.Def.Recv.Parms, fun.Recv.Args, sub)
	}
	fun.Stmts = subStmts(x, sub, fun.Def.Stmts)
}

func newSubMap(parms []TypeVar, args []TypeName) map[*TypeVar]TypeName {
	sub := make(map[*TypeVar]TypeName)
	addSubMap(parms, args, sub)
	return sub
}

func addSubMap(parms []TypeVar, args []TypeName, sub map[*TypeVar]TypeName) {
	for i := range parms {
		sub[&parms[i]] = args[i]
	}
}

func isGroundFun(fun *Fun) bool {
	for _, a := range fun.TArgs {
		if !isGroundType(a.Type) {
			return false
		}
	}
	return len(fun.TParms) == len(fun.TArgs) &&
		(fun.Recv == nil || isGroundType(fun.Recv.Type))
}

func isGroundType(typ *Type) bool {
	if typ == nil || typ.Var != nil {
		return false
	}
	for _, a := range typ.Args {
		if !isGroundType(a.Type) {
			return false
		}
	}
	return true
}
