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

func subStmts(x *scope, sub map[*TypeVar]TypeName, stmts0 []Stmt) []Stmt {
	defer x.tr("subStmts()")()

	stmts1 := make([]Stmt, len(stmts0))
	for i, stmt := range stmts0 {
		stmts1[i] = subStmt(x, sub, stmt)
	}
	return stmts1
}

func subStmt(x *scope, sub map[*TypeVar]TypeName, stmt0 Stmt) Stmt {
	defer x.tr("subStmt()")()

	switch stmt0 := stmt0.(type) {
	case *Ret:
		return subRet(x, sub, stmt0)
	case *Assign:
		return subAssign(x, sub, stmt0)
	case Expr:
		return subExpr(x, sub, stmt0)
	default:
		panic(fmt.Sprintf("impossible type: %T", stmt0))
	}
}

func subRet(x *scope, sub map[*TypeVar]TypeName, ret0 *Ret) *Ret {
	defer x.tr("subRet()")()

	return &Ret{AST: ret0.AST, Val: subExpr(x, sub, ret0.Val)}
}

func subAssign(x *scope, sub map[*TypeVar]TypeName, assign0 *Assign) *Assign {
	defer x.tr("subAssign(%s)", assign0.Var.Name)()

	return &Assign{
		AST:  assign0.AST,
		Var:  lookUpVar(x, assign0.Var),
		Expr: subExpr(x, sub, assign0.Expr),
	}
}

func subExprs(x *scope, sub map[*TypeVar]TypeName, exprs0 []Expr) []Expr {
	defer x.tr("subExprs()")()

	exprs1 := make([]Expr, len(exprs0))
	for i, expr0 := range exprs0 {
		exprs1[i] = subExpr(x, sub, expr0)
	}
	return exprs1
}

func subExpr(x *scope, sub map[*TypeVar]TypeName, expr0 Expr) Expr {
	defer x.tr("subExpr()")()

	switch expr0 := expr0.(type) {
	case nil:
		return nil
	case *Call:
		return subCall(x, sub, expr0)
	case *Ctor:
		return subCtor(x, sub, expr0)
	case *Block:
		return subBlock(x, sub, expr0)
	case *Ident:
		return subIdent(x, sub, expr0)
	case *Int:
		defer x.tr("subInt()")()
		return expr0
	case *Float:
		defer x.tr("subFloat()")()
		return expr0
	case *String:
		defer x.tr("subString()")()
		return expr0
	case *Convert:
		return subConvert(x, sub, expr0)
	default:
		panic(fmt.Sprintf("impossible type: %T", expr0))
	}
}

func subConvert(x *scope, sub map[*TypeVar]TypeName, cvt0 *Convert) *Convert {
	defer x.tr("subConvert()")()

	cvt1 := &Convert{
		Expr: subExpr(x, sub, cvt0.Expr),
		Ref:  cvt0.Ref,
		typ:  subType(x, map[*Type]*Type{}, sub, cvt0.typ),
	}
	if len(cvt0.Virts) > 0 {
		if len(cvt1.typ.Virts) == 0 {
			panic("impossible")
		}
		virts, errs := findVirts(x, cvt1.ast(), cvt1.Expr.Type(), cvt1.typ.Virts)
		if len(errs) > 0 {
			panic(fmt.Sprintf("impossible: %v", errs))
		}
		cvt1.Virts = virts
	}
	return cvt1
}

func subCall(x *scope, sub map[*TypeVar]TypeName, call0 *Call) *Call {
	defer x.tr("subCall()")()

	call1 := &Call{
		AST:  call0.AST,
		Recv: subExpr(x, sub, call0.Recv),
		typ:  subType(x, map[*Type]*Type{}, sub, call0.typ),
	}
	var recv *Type
	if call1.Recv != nil {
		recv = call1.Recv.Type()
		if !isRef(x, recv) {
			// The receiver is always converted to a ref by the check pass.
			panic("impossible")
		}
		recv = recv.Args[0].Type
	}
	call1.Msgs = subMsgs(x, sub, call1.typ, recv, call0.Msgs)
	return call1
}

func subMsgs(x *scope, sub map[*TypeVar]TypeName, ret1, recv1 *Type, msgs0 []Msg) []Msg {
	defer x.tr("subMsgs(ret1=%s, recv1=%s)", ret1, recv1)()

	msgs1 := make([]Msg, len(msgs0))
	for i := range msgs0 {
		var ret *Type
		if i == len(msgs0)-1 {
			ret = ret1
		}
		msgs1[i] = subMsg(x, sub, ret, recv1, &msgs0[i])
	}
	return msgs1
}

func subMsg(x *scope, sub map[*TypeVar]TypeName, ret1, recv1 *Type, msg0 *Msg) Msg {
	defer x.tr("subMsg(ret1=%s, recv1=%s, %s)", ret1, recv1, msg0.Sel)()

	msg1 := Msg{
		AST:  msg0.AST,
		Mod:  msg0.Mod,
		Sel:  msg0.Sel,
		Args: subExprs(x, sub, msg0.Args),
	}
	if errs := findMsgFun(x, ret1, recv1, &msg1); len(errs) > 0 {
		panic(fmt.Sprintf("impossible: %v", errs))
	}
	return msg1
}

func subCtor(x *scope, sub map[*TypeVar]TypeName, ctor0 *Ctor) *Ctor {
	defer x.tr("subCtor()")()

	return &Ctor{
		AST:  ctor0.AST,
		Args: subExprs(x, sub, ctor0.Args),
		Case: ctor0.Case,
		typ:  subType(x, map[*Type]*Type{}, sub, ctor0.typ),
	}
}

func subBlock(x *scope, sub map[*TypeVar]TypeName, block0 *Block) *Block {
	defer x.tr("subBlock()")()

	seen := map[*Type]*Type{}
	block1 := &Block{
		AST:    block0.AST,
		Parms:  make([]Var, len(block0.Parms)),
		Locals: make([]*Var, len(block0.Parms)),
		typ:    subType(x, seen, sub, block0.typ),
	}
	x = x.new()
	x.block = block1

	for i := range block0.Parms {
		parm0 := &block0.Parms[i]
		parm1 := &block1.Parms[i]
		parm1.AST = parm0.AST
		parm1.Name = parm0.Name
		parm1.TypeName = subTypeName(x, seen, sub, parm0.TypeName)
		parm1.typ = subType(x, seen, sub, parm0.typ)
		parm1.BlkParm = block1
		parm1.Index = i
		x = x.new()
		x.variable = parm1
	}
	for i, local0 := range block0.Locals {
		local1 := new(Var)
		block1.Locals[i] = local1
		local1.AST = local0.AST
		local1.Name = local0.Name
		local1.TypeName = subTypeName(x, seen, sub, local0.TypeName)
		local1.typ = subType(x, seen, sub, local0.typ)
		local1.Local = &block1.Locals
		local1.Index = i
		x = x.new()
		x.variable = local1
	}

	block1.Stmts = subStmts(x, sub, block0.Stmts)

	return block1
}

func subIdent(x *scope, sub map[*TypeVar]TypeName, ident0 *Ident) *Ident {
	defer x.tr("subIdent(%s)", ident0.Text)()

	return &Ident{
		AST:  ident0.AST,
		Text: ident0.Text,
		Var:  lookUpVar(x, ident0.Var),
	}
}

func lookUpVar(x *scope, var0 *Var) *Var {
	defer x.tr("lookUpVar(%s)", var0.Name)()

	var1, err := x.findIdent(var0.AST, var0.Name)
	if err != nil {
		panic("impossible: " + err.Error())
	}
	switch var1, ok := var1.(*Var); {
	case !ok:
		panic("impossible")
	case var1 == nil:
		panic("impossible")
	case var1.typ == nil:
		panic("impossible")
	case (var1.Val == nil) != (var0.Val == nil):
		panic("impossible")
	case (var1.FunParm == nil) != (var0.FunParm == nil):
		panic("impossible")
	case (var1.BlkParm == nil) != (var0.BlkParm == nil):
		panic("impossible")
	case (var1.Local == nil) != (var0.Local == nil):
		panic("impossible")
	case (var1.Field == nil) != (var0.Field == nil):
		panic("impossible")
	case var1.Index != var0.Index:
		panic("impossible")
	default:
		return var1
	}
}
