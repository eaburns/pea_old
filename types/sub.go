package types

import (
	"fmt"
	"sort"
	"strings"
)

func subTypeNames(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, names0 []TypeName) []TypeName {
	var names1 []TypeName
	for i := range names0 {
		n := subTypeName(x, seen, sub, &names0[i])
		names1 = append(names1, *n)
	}
	return names1
}

func subTypeName(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, name0 *TypeName) *TypeName {
	if name0 == nil {
		return nil
	}
	defer x.tr("subTypeName(%s, %s [var=%p])", subDebugString(sub), name0.ID(), name0.Var)()

	if s, ok := sub[name0.Var]; ok {
		x.log("%s→%s", name0.Var.Name, s)
		return &s
	}
	name1 := *name0
	name1.Args = subTypeNames(x, seen, sub, name1.Args)
	name1.Var = subVar(x, seen, sub, name1.Var)
	name1.Type = subType(x, seen, sub, name1.Type)
	return &name1
}

func subVars(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, vars0 []Var) []Var {
	var vars1 []Var
	for i := range vars0 {
		vars1 = append(vars1, *subVar(x, seen, sub, &vars0[i]))
	}
	return vars1
}

func subVar(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, var0 *Var) *Var {
	if var0 == nil {
		return nil
	}
	defer x.tr("subVar(%s, %s)", subDebugString(sub), var0.Name)()

	var1 := *var0
	var1.Type = subTypeName(x, seen, sub, var1.Type)
	return &var1
}

func subType(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, typ0 *Type) *Type {
	if typ0 == nil {
		return nil
	}
	if seen[typ0] {
		return typ0
	}
	seen[typ0] = true

	defer x.tr("subType(%s, %s)", subDebugString(sub), typ0.name())()

	typ1 := *typ0
	subTypeBody(x, seen, sub, &typ1)
	return &typ1
}

func subTypeBody(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, typ *Type) {
	typ.Sig.Parms = subVars(x, seen, sub, typ.Sig.Parms)
	typ.Sig.Args = subTypeNames(x, seen, sub, typ.Sig.Args)
	typ.Alias = subTypeName(x, seen, sub, typ.Alias)
	typ.Fields = subVars(x, seen, sub, typ.Fields)
	typ.Cases = subVars(x, seen, sub, typ.Cases)
	typ.Virts = subFunSigs(x, seen, sub, typ.Virts)
}

func subFunSigs(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, sigs0 []FunSig) []FunSig {
	var sigs1 []FunSig
	for i := range sigs0 {
		sigs1 = append(sigs1, *subFunSig(x, seen, sub, &sigs0[i]))
	}
	return sigs1
}

func subFunSig(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, sig0 *FunSig) *FunSig {
	sig1 := *sig0
	sig1.Parms = subVars(x, seen, sub, sig1.Parms)
	sig1.Ret = subTypeName(x, seen, sub, sig1.Ret)
	return &sig1
}

func subDebugString(sub map[*Var]TypeName) string {
	var ss []string
	for k, v := range sub {
		s := fmt.Sprintf("%s[%p]=%s", k.Name, k, v)
		ss = append(ss, s)
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })
	return strings.Join(ss, ";")
}
