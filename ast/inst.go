package ast

import (
	"fmt"
	"strings"
)

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
	x.mod.Insts = append(x.mod.Insts, &n)
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
