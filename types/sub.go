package types

import (
	"fmt"
	"sort"
	"strings"
)

func subTypeNames(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, names0 []TypeName) []TypeName {
	var names1 []TypeName
	for i := range names0 {
		n := subTypeName(x, seen, sub, &names0[i])
		names1 = append(names1, *n)
	}
	return names1
}

func subTypeName(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, name0 *TypeName) *TypeName {
	if name0 == nil || name0.Type == nil {
		return nil
	}
	defer x.tr("subTypeName(%s, %s [var=%p])", subDebugString(sub), name0.name(), name0.Type.Var)()

	if s, ok := sub[name0.Type.Var]; ok {
		x.log("%s→%s", name0.Type.Var.Name, s)
		return &s
	}

	name1 := *name0
	name1.Args = subTypeNames(x, seen, sub, name1.Args)
	name1.Type = subType(x, seen, sub, name1.Type)
	return &name1
}

func subType(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ0 *Type) *Type {
	if typ0 == nil {
		return nil
	}
	if typ1 := seen[typ0]; typ1 != nil {
		return typ1
	}

	defer x.tr("subType(%s, %p %s)", subDebugString(sub), typ0, typ0)()

	args := subTypeNames(x, seen, sub, typ0.Args)
	typ1, es := instType(x, typ0, args)
	if len(es) > 0 {
		panic("impossible?")
	}
	return typ1
}

func subTypeBody(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ *Type) {
	switch {
	case typ.Var != nil:
		subTypeVar(x, seen, sub, typ)
	case typ.Alias != nil:
		typ.Alias = subTypeName(x, seen, sub, typ.Alias)
	case typ.Fields != nil:
		subFields(x, seen, sub, typ)
	case typ.Cases != nil:
		subCases(x, seen, sub, typ)
	case typ.Virts != nil:
		subVirts(x, seen, sub, typ)
	}
}

func subTypeVar(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ *Type) {
	defer x.tr("subTypeVar(%s)", subDebugString(sub))()

	for i := range typ.Var.Ifaces {
		typ.Var.Ifaces[i] = *subTypeName(x, seen, sub, &typ.Var.Ifaces[i])
	}
	typ.Var.Type = subType(x, seen, sub, typ.Var.Type)
}

func subTypeParms(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, parms0 []TypeVar) []TypeVar {
	defer x.tr("subTypeParms(%s)", subDebugString(sub))()

	parms1 := make([]TypeVar, len(parms0))
	for i := range parms0 {
		parm0 := &parms0[i]
		parm1 := &parms1[i]
		parm1.AST = parm0.AST
		parm1.Name = parm0.Name
		for i := range parm0.Ifaces {
			parm1.Ifaces[i] = *subTypeName(x, seen, sub, &parm0.Ifaces[i])
		}
		parm1.Type = subType(x, seen, sub, parm0.Type)
	}
	return parms1
}

func subFields(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ *Type) {
	defer x.tr("subFields(%s)", subDebugString(sub))()

	fields0 := typ.Fields
	typ.Fields = make([]Var, len(fields0))
	for i := range fields0 {
		field0 := &fields0[i]
		field1 := &typ.Fields[i]
		field1.AST = field0.AST
		field1.Name = field0.Name
		field1.TypeName = subTypeName(x, seen, sub, field0.TypeName)
		field1.typ = subType(x, seen, sub, field0.typ)
		field1.Field = typ
		field1.Index = i
	}
}

func subCases(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ *Type) {
	defer x.tr("subCases(%s)", subDebugString(sub))()

	cases0 := typ.Cases
	typ.Cases = make([]Var, len(cases0))
	for i := range cases0 {
		case0 := &cases0[i]
		case1 := &typ.Cases[i]
		case1.AST = case0.AST
		case1.Name = case0.Name
		case1.TypeName = subTypeName(x, seen, sub, case0.TypeName)
		case1.typ = subType(x, seen, sub, case0.typ)
		case1.Index = i
	}
}

func subVirts(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ *Type) {
	defer x.tr("subVirts(%s)", subDebugString(sub))()
	sigs0 := typ.Virts
	typ.Virts = make([]FunSig, len(sigs0))
	for i := range sigs0 {
		sig0 := &sigs0[i]
		sig1 := &typ.Virts[i]
		sig1.AST = sig0.AST
		sig1.Sel = sig0.Sel
		sig1.Parms = make([]Var, len(sig0.Parms))
		for i := range sig0.Parms {
			parm0 := &sig0.Parms[i]
			parm1 := &sig1.Parms[i]
			parm1.AST = parm0.AST
			parm1.Name = parm0.Name
			parm1.TypeName = subTypeName(x, seen, sub, parm0.TypeName)
			parm1.typ = subType(x, seen, sub, parm0.typ)
		}
		sig1.Ret = subTypeName(x, seen, sub, sig0.Ret)
	}
}

func subRecv(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, recv0 *Recv) *Recv {
	defer x.tr("subRecv(%s, %s)", subDebugString(sub), recv0.name())()

	recv1 := *recv0
	recv1.Parms = subTypeParms(x, seen, sub, recv0.Parms)
	recv1.Type = subType(x, seen, sub, recv0.Type)
	return &recv1
}

func subFun(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, fun *Fun) *Fun {
	inst := &Fun{
		AST:    fun.AST,
		Priv:   fun.Priv,
		Mod:    fun.Mod,
		Recv:   subRecv(x, seen, sub, fun.Recv),
		TParms: subTypeParms(x, seen, sub, fun.TParms),
		Sig: FunSig{
			AST: fun.Sig.AST,
			Sel: fun.Sig.Sel,
			Ret: subTypeName(x, seen, sub, fun.Sig.Ret),
		},
	}

	inst.Sig.Parms = make([]Var, len(fun.Sig.Parms))
	for i := range fun.Sig.Parms {
		parm0 := &fun.Sig.Parms[i]
		parm1 := &inst.Sig.Parms[i]
		parm1.AST = parm0.AST
		parm1.Name = parm0.Name
		parm1.TypeName = subTypeName(x, seen, sub, parm0.TypeName)
		parm1.FunParm = fun
		parm1.Index = i
		parm1.typ = subType(x, seen, sub, parm1.typ)
	}

	inst.Locals = make([]*Var, len(fun.Locals))
	for i, loc0 := range fun.Locals {
		inst.Locals[i] = &Var{
			AST:      loc0.AST,
			Name:     loc0.Name,
			TypeName: subTypeName(x, seen, sub, loc0.TypeName),
			Local:    &inst.Locals,
			Index:    i,
			typ:      subType(x, seen, sub, loc0.typ),
		}
	}

	// Note that we don't substitute the statements here.
	// They are instead substituted after the check pass
	// if there were no check errors.

	return inst
}

func subDebugString(sub map[*TypeVar]TypeName) string {
	if sub == nil {
		return "[]"
	}
	var ss []string
	for k, v := range sub {
		s := fmt.Sprintf("%s[%p]=%s", k.Name, k, v)
		ss = append(ss, s)
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })
	return strings.Join(ss, ";")
}
