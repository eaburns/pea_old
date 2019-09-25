package types

import (
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
	tparms := []Var{{Name: tmp}}
	retType := &Type{
		Sig: TypeSig{Name: tmp},
		Var: &tparms[0],
	}
	tparms[0].TypeVar = retType
	tparms[0].typ = retType
	retName := TypeName{Name: tmp, Type: retType}

	var sel strings.Builder
	parms := []Var{
		{Name: "self", TypeName: makeTypeName(typ), typ: typ},
	}
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
			typ:      typ,
		})
	}
	return &Fun{
		ast:  typ.ast,
		Priv: typ.Priv,
		Mod:  typ.Sig.Mod,
		Recv: &Recv{
			Parms: typ.Sig.Parms,
			Mod:   typ.Sig.Mod,
			Arity: len(typ.Sig.Parms),
			Name:  typ.Sig.Name,
			Type:  typ,
		},
		TParms: tparms,
		Sig: FunSig{
			Sel:   sel.String(),
			Parms: parms,
			Ret:   &retName,
		},
	}
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
	parms[0] = Var{Name: "self", TypeName: makeTypeName(typ), typ: typ}
	for i, p := range sig.Parms {
		p.Name = "_"
		parms[i+1] = p
	}
	sig.Parms = parms
	return &Fun{
		ast:  sig.ast,
		Priv: typ.Priv,
		Mod:  typ.Sig.Mod,
		Recv: &Recv{
			Parms: typ.Sig.Parms,
			Mod:   typ.Sig.Mod,
			Arity: len(typ.Sig.Parms),
			Name:  typ.Sig.Name,
			Type:  typ,
		},
		Sig: sig,
	}
}

func makeTypeName(typ *Type) *TypeName {
	args := typ.Sig.Args
	if typ.Sig.Args == nil {
		for i := range typ.Sig.Parms {
			parm := &typ.Sig.Parms[i]
			args = append(args, TypeName{
				Mod:  "",
				Name: parm.Name,
				Type: parm.TypeVar,
			})
		}
	}
	return &TypeName{
		Mod:  typ.Sig.Mod,
		Name: typ.Sig.Name,
		Args: args,
		Type: typ,
	}
}
