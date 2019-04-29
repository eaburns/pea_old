package pea

import "fmt"

func (*Import) inst(*scope, []TypeName) (Def, []checkError) {
	panic("impossible")
}

func (n Fun) instRecv(x *scope, typ TypeName) (_ *Fun, errs []checkError) {
	defer x.tr("Fun.instRecv(%s, %s)", n.Name(), typ)(errs)
	if n.Recv == nil {
		return &n, nil
	}
	sig, errs := instTypeSig(x, *n.Recv, typ)
	if len(errs) > 0 {
		return &n, errs
	}
	n.Recv = &sig
	n.TypeParms = subParms(n.Recv.x, n.Recv.Args, n.TypeParms)
	n.Parms = subParms(n.Recv.x, n.Recv.Args, n.Parms)
	if n.Ret != nil {
		n.Ret = subTypeName(n.Recv.x, n.Recv.Args, *n.Ret)
	}
	n.Stmts = subStmts(n.Recv.x, n.Recv.Args, n.Stmts)
	return &n, nil
}

func (n Type) inst(x *scope, typ TypeName) (_ *Type, errs []checkError) {
	defer x.tr("Type.inst(%s, %s)", n.Name(), typ)(errs)
	n.Sig, errs = instTypeSig(x, n.Sig, typ)
	if len(errs) != 0 {
		return &n, errs
	}
	switch {
	case n.Alias != nil:
		n.Alias = subTypeName(x, n.Sig.Args, *n.Alias)
	case n.Fields != nil:
		n.Fields = subParms(n.Sig.x, n.Sig.Args, n.Fields)
	case n.Cases != nil:
		n.Cases = subParms(n.Sig.x, n.Sig.Args, n.Cases)
	case n.Virts != nil:
		n.Virts = subMethSigs(n.Sig.x, n.Sig.Args, n.Virts)
	}
	return &n, nil
}

func instTypeSig(x *scope, sig TypeSig, typ TypeName) (_ TypeSig, errs []checkError) {
	defer x.tr("instTypeSig(%s, %s)", sig, typ)(errs)
	ps := sig.Parms
	if len(ps) != len(typ.Args) {
		err := x.err(sig, "expected %d arguments, got %d", len(ps), len(typ.Args))
		errs = append(errs, *err)
		return sig, errs
	}

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
		sig.Args[p] = typ.Args[i]
	}
	return sig, nil
}

func (n Call) sub(x *scope, sub map[*Parm]TypeName) Expr {
	defer x.tr("Call.sub(…)")()
	if n.Recv != nil {
		n.Recv = n.Recv.sub(x, sub)
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

func (n ModPath) sub(*scope, map[*Parm]TypeName) Expr { return n }

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
		case Ret:
			up := x.tr("sub ret")
			s.Val = s.Val.sub(x, sub)
			out[i] = s
			up()
		case Assign:
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
