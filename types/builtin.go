package types

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func builtInMeths(x *scope, defs []Def) []Def {
	var out []Def
	for _, def := range defs {
		switch typ, ok := def.(*Type); {
		case !ok:
			continue
		case len(typ.Cases) > 0:
			out = append(out, makeCaseMeth(x, typ))
		case len(typ.Virts) > 0:
			out = append(out, makeVirtMeths(x, typ)...)
		}
	}
	return out
}

func makeCaseMeth(x *scope, typ *Type) *Fun {
	tmp := x.newID()
	tparms := []TypeVar{{Name: tmp}}
	retType := &Type{
		Name: tmp,
		Var:  &tparms[0],
	}
	retType.Def = retType
	tparms[0].Type = retType
	retName := TypeName{Name: tmp, Type: retType}

	var sel strings.Builder
	selfType := builtInType(x, "&", *makeTypeName(typ))
	parms := []Var{{
		Name:     "self",
		TypeName: makeTypeName(selfType),
		typ:      selfType,
	}}
	for _, c := range typ.Cases {
		sel.WriteString("if")
		sel.WriteString(upperCase(c.Name))
		var parmType *Type
		if c.TypeName == nil {
			sel.WriteRune(':')
			parmType = builtInType(x, "Fun", retName)
		} else {
			ref := builtInType(x, "&", *c.TypeName)
			parmType = builtInType(x, "Fun", *makeTypeName(ref), retName)
		}
		parms = append(parms, Var{
			Name:     "_",
			TypeName: makeTypeName(parmType),
			typ:      parmType,
		})
	}
	fun := &Fun{
		AST:  typ.AST,
		Priv: typ.Priv,
		Mod:  typ.Mod,
		Recv: &Recv{
			Parms: typ.Parms,
			Mod:   typ.Mod,
			Arity: len(typ.Parms),
			Name:  typ.Name,
			Type:  typ,
		},
		TParms: tparms,
		Sig: FunSig{
			Sel:   sel.String(),
			Parms: parms,
			Ret:   &retName,
		},
	}
	fun.Def = fun
	return fun
}

func upperCase(s string) string {
	r, w := utf8.DecodeRuneInString(s)
	return string([]rune{unicode.ToUpper(r)}) + s[w:]
}

func makeVirtMeths(x *scope, typ *Type) []Def {
	var defs []Def
	for _, virt := range typ.Virts {
		defs = append(defs, makeVirtMeth(x, typ, virt))
	}
	return defs
}

func makeVirtMeth(x *scope, typ *Type, sig FunSig) *Fun {
	parms := make([]Var, len(sig.Parms)+1)
	selfType := builtInType(x, "&", *makeTypeName(typ))
	parms[0] = Var{
		Name:     "self",
		TypeName: makeTypeName(selfType),
		typ:      selfType,
	}
	for i, p := range sig.Parms {
		p.Name = "_"
		parms[i+1] = p
	}
	sig.Parms = parms
	fun := &Fun{
		AST:  sig.AST,
		Priv: typ.Priv,
		Mod:  typ.Mod,
		Recv: &Recv{
			Parms: typ.Parms,
			Mod:   typ.Mod,
			Arity: len(typ.Parms),
			Name:  typ.Name,
			Type:  typ,
		},
		Sig: sig,
	}
	fun.Def = fun
	return fun
}

func makeTypeName(typ *Type) *TypeName {
	args := typ.Args
	if typ.Args == nil {
		for i := range typ.Parms {
			parm := &typ.Parms[i]
			args = append(args, TypeName{
				Mod:  "",
				Name: parm.Name,
				Type: parm.Type,
			})
		}
	}
	return &TypeName{
		Mod:  typ.Mod,
		Name: typ.Name,
		Args: args,
		Type: typ,
	}
}

func builtInType(x *scope, name string, args ...TypeName) *Type {
	// Silence tracing for looking up built-in types.
	savedTrace := x.cfg.Trace
	x.cfg.Trace = false
	defer func() { x.cfg.Trace = savedTrace }()

	for x.univ == nil {
		x = x.up
	}
	typ := findTypeInDefs(len(args), name, x.univ)
	if typ == nil {
		panic(fmt.Sprintf("built-in type (%d)%s not found", len(args), name))
	}
	typ, errs := instType(x, typ, args)
	if len(errs) > 0 {
		panic(fmt.Sprintf("failed to inst built-in type: %v", errs))
	}
	return typ
}

func isNil(x *scope, typ *Type) bool {
	return isBuiltIn(x, typ) && typ.Name == "Nil"
}

func isAry(x *scope, typ *Type) bool {
	return isBuiltIn(x, typ) && typ.Name == "Array"
}

func isRef(x *scope, typ *Type) bool {
	return isBuiltIn(x, typ) && typ.Name == "&"
}

func isFun(x *scope, typ *Type) bool {
	return isBuiltIn(x, typ) && typ.Name == "Fun"
}

func isBuiltIn(x *scope, typ *Type) bool {
	return typ != nil && typ.Mod == "" && x.defFiles[typ] == nil
}
