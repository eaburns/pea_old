package sem

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
	if name0 == nil {
		return nil
	}
	defer x.tr("subTypeName(%s, %s)", subDebugString(sub), name0)()

	if name0.Type != nil {
		if s, ok := sub[name0.Type.Var]; ok {
			return &s
		}
	}

	name1 := *name0
	name1.Args = subTypeNames(x, seen, sub, name0.Args)
	name1.Type = subType(x, seen, sub, name0.Type)
	return &name1
}

func subType(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ0 *Type) *Type {
	if typ0 == nil {
		return nil
	}
	if s, ok := sub[typ0.Var]; ok {
		return s.Type
	}
	if typ1 := seen[typ0]; typ1 != nil {
		return typ1
	}

	defer x.tr("subType(%s, %s %p)", subDebugString(sub), typ0, typ0)()

	args := subTypeNames(x, seen, sub, typ0.Args)
	typ1, es := instType(x, typ0, args)
	if len(es) > 0 {
		panic("impossible?")
	}
	if typ0.Var != nil && typ0 != typ1 {
		panic("impossible")
	}
	return typ1
}

func subTypeBody(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ *Type) {
	defer x.tr("subTypeBody(%s, %s", subDebugString(sub), typ)()

	typ.Parms = subTypeParms(x, seen, sub, typ.Parms)
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

	var0 := typ.Var
	var1 := *var0
	var1.Ifaces = make([]TypeName, len(var0.Ifaces))
	for i := range var0.Ifaces {
		var1.Ifaces[i] = *subTypeName(x, seen, sub, &var0.Ifaces[i])
	}
	var1.Type = &Type{
		AST:  var1.AST,
		Name: var1.Name,
		Var:  &var1,
	}
	var1.Type.Def = var1.Type
	seen[var0.Type] = var1.Type
	typ.Var = &var1
}

func subTypeParms(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, parms0 []TypeVar) []TypeVar {
	defer x.tr("subTypeParms(%s)", subDebugString(sub))()

	parms1 := make([]TypeVar, len(parms0))
	for i := range parms0 {
		parm0 := &parms0[i]
		parm1 := &parms1[i]
		parm1.AST = parm0.AST
		parm1.Name = parm0.Name
		parm1.Ifaces = make([]TypeName, len(parm0.Ifaces))
		for i := range parm0.Ifaces {
			parm1.Ifaces[i] = *subTypeName(x, seen, sub, &parm0.Ifaces[i])
		}
		parm1.Type = &Type{
			AST:  parm1.AST,
			Name: parm1.Name,
			Var:  parm1,
		}
		seen[parm0.Type] = parm1.Type
	}
	return parms1
}

func subFields(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ *Type) {
	defer x.tr("subFields(%s)", subDebugString(sub))()

	fields0 := typ.Fields
	typ.Fields = make([]Var, len(fields0))
	for i := range fields0 {
		typ.Fields[i] = subVar(x, seen, sub, &fields0[i])
		typ.Fields[i].Field = typ
		typ.Fields[i].Index = i
	}
}

func subCases(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, typ *Type) {
	defer x.tr("subCases(%s)", subDebugString(sub))()
	cases0 := typ.Cases
	typ.Cases = make([]Var, len(cases0))
	for i := range cases0 {
		typ.Cases[i] = subVar(x, seen, sub, &cases0[i])
		typ.Cases[i].Index = i
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
			sig1.Parms[i] = subVar(x, seen, sub, &sig0.Parms[i])
		}
		sig1.Ret = subTypeName(x, seen, sub, sig0.Ret)
	}
}

func subVar(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, vr0 *Var) Var {
	var vr1 Var
	vr1.AST = vr0.AST
	vr1.Name = vr0.Name
	if vr0.TypeName != nil {
		vr1.TypeName = subTypeName(x, seen, sub, vr0.TypeName)
		vr1.typ = vr1.TypeName.Type
	}
	return vr1
}

func subRecv(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, recv0 *Recv) *Recv {
	if recv0 == nil {
		return nil
	}
	defer x.tr("subRecv(%s, %s)", subDebugString(sub), recv0.name())()
	if recv0.Type != nil {
		x.log("recv type: %s", recv0.Type.fullString())
	}

	recv1 := *recv0
	recv1.Parms = subTypeParms(x, seen, sub, recv0.Parms)
	recv1.Type = subType(x, seen, sub, recv0.Type)
	return &recv1
}

func subFun(x *scope, seen map[*Type]*Type, sub map[*TypeVar]TypeName, fun *Fun) *Fun {
	defer x.tr("subFun(%s)", fun)()

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
		inst.Sig.Parms[i] = subVar(x, seen, sub, &fun.Sig.Parms[i])
		inst.Sig.Parms[i].FunParm = fun
		inst.Sig.Parms[i].Index = i
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
