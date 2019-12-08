package basic

import (
	"fmt"
	"math/big"

	"github.com/eaburns/pea/types"
)

// Build builds a basic representation of a module.
func Build(typesMod *types.Mod) *Mod {
	mod := &Mod{Mod: typesMod}
	for _, def := range typesMod.Defs {
		fun, ok := def.(*types.Fun)
		if !ok {
			continue
		}
		for _, inst := range fun.Insts {
			buildFun(mod, inst)
		}
	}
	return mod
}

func addString(mod *Mod, str string) *String {
	for _, s := range mod.Strings {
		if s.Data == str {
			return s
		}
	}
	s := &String{N: mod.NDefs, Data: str}
	mod.Strings = append(mod.Strings, s)
	mod.NDefs++
	return s
}

func buildFun(mod *Mod, typesFun *types.Fun) {
	f := findFun(mod, typesFun)
	buildFunBody(f, f.Parms, typesFun.Locals, typesFun.Stmts)
}

func findFun(mod *Mod, fun *types.Fun) *Fun {
	for _, f := range mod.Funs {
		if f.Fun == fun && f.Block == nil {
			return f
		}
	}
	f := newFun(mod, fun.Sig.Parms, fun.Sig.Ret)
	f.Fun = fun
	return f
}

func buildBlockFun(mod *Mod, fun *types.Fun, block *types.Block) *Fun {
	ret := &block.Type().Args[len(block.Type().Args)-1]
	f := newFun(mod, block.Parms, ret)
	f.Fun = fun
	f.Block = block

	// Blocks always begin with a self parameter. Stick one on here.
	f.Parms = append([]*Parm{{Type: block.BlockType.Ref()}}, f.Parms...)
	for i, p := range f.Parms {
		p.N = i
	}
	if f.Ret != nil {
		f.Ret.N = len(f.Parms)
	}

	buildFunBody(f, f.Parms, block.Locals, block.Stmts)
	return f
}

func newFun(mod *Mod, parms []types.Var, ret *types.TypeName) *Fun {
	fun := &Fun{N: mod.NDefs, Mod: mod}
	mod.Funs = append(mod.Funs, fun)
	mod.NDefs++

	fun.Parms = make([]*Parm, 0, len(parms))
	for i := range parms {
		typ := parms[i].Type()
		if EmptyType(typ) {
			continue
		}
		parm := &Parm{N: i, Type: typ, Var: &parms[i]}
		if !SimpleType(typ) {
			parm.Value = true
			parm.Type = typ.Ref()
		}
		fun.Parms = append(fun.Parms, parm)
	}
	if ret != nil && !EmptyType(ret.Type) {
		fun.Ret = &Parm{
			N:    len(fun.Parms),
			Type: ret.Type.Ref(),
		}
	}
	return fun
}

func buildFunBody(f *Fun, parms []*Parm, locals []*types.Var, stmts []types.Stmt) {
	b0 := newBBlk(f)
	parmAllocs := make([]*Alloc, 0, len(parms))
	for _, parm := range parms {
		if parm.Value {
			continue
		}
		// For non-by-value parameters, the function may take its address.
		// We need to make sure there is a memory location for that address.
		a := addAlloc(f, b0, parm.Type)
		a.Var = parm.Var
		parmAllocs = append(parmAllocs, a)
	}
	for _, local := range locals {
		a := addAlloc(f, b0, local.Type())
		a.Var = local
	}
	for i, parm := range parms {
		if parm.Value {
			continue
		}
		addStore(b0, parmAllocs[i], addArg(f, b0, parm))
	}
	b1 := newBBlk(f)
	buildStmts(f, b1, stmts)
	addJmp(b0, b1)
}

func newBBlk(fun *Fun) *BBlk {
	b := &BBlk{N: len(fun.BBlks)}
	fun.BBlks = append(fun.BBlks, b)
	return b
}

func buildStmts(f *Fun, b *BBlk, stmts []types.Stmt) *BBlk {
	for i, stmt := range stmts {
		addComment(b, "%T", stmt)

		switch stmt := stmt.(type) {
		case *types.Ret:
			buildRet(f, b, stmt)
			if i < len(stmts)-1 {
				// Add a new block to collect the dead code after the return.
				b = newBBlk(f)
			}
		case *types.Assign:
			b = buildAssign(f, b, stmt)
		case types.Expr:
			var v Val
			v, b = buildExpr(f, b, stmt)
			if i == len(stmts)-1 && f.Block != nil {
				buildBlockFunRet(f, b, v)
			}
		default:
			panic(fmt.Sprintf("impossible: %T", stmt))
		}
	}
	if n := len(b.Stmts); n == 0 || !isTerm(b.Stmts[n-1]) {
		addRet(b)
	}
	return b
}

func isTerm(s Stmt) bool {
	_, ok := s.(Term)
	return ok
}

func buildRet(f *Fun, b *BBlk, typesRet *types.Ret) *BBlk {
	val, b := buildExpr(f, b, typesRet.Expr)
	if val == nil {
		ret := addRet(b)
		ret.Ret = typesRet
		return b
	}

	var dst Val
	if f.Block != nil {
		// This is a block literal far-return.
		// The return location is in a capture field.
		self := addLoad(f, b, selfParm(f, b))
		dstPtr := addField(f, b, self, len(f.Block.Captures))
		dst = addLoad(f, b, dstPtr)
	} else {
		dst = addArg(f, b, f.Ret)
	}

	if SimpleType(f.Fun.Sig.Ret.Type) {
		addStore(b, dst, val)
	} else {
		addCopy(b, dst, val)
	}
	ret := addRet(b)
	ret.Ret = typesRet
	ret.Far = f.Block != nil
	return b
}

func buildBlockFunRet(f *Fun, b *BBlk, v Val) {
	if f.Ret == nil {
		return
	}
	dst := addArg(f, b, f.Ret)
	if SimpleType(f.Ret.Type.Args[0].Type) {
		addStore(b, dst, v)
	} else {
		addCopy(b, dst, v)
	}
	addRet(b)
}

func buildAssign(f *Fun, b *BBlk, typesAssign *types.Assign) *BBlk {
	var val Val
	val, b = buildExpr(f, b, typesAssign.Expr)
	if val == nil {
		return b // empty type
	}
	if SimpleType(typesAssign.Expr.Type()) {
		dst := buildVar(f, b, typesAssign.Var)
		s := addStore(b, dst, val)
		s.Assign = typesAssign
		return b
	}
	dst := buildVar(f, b, typesAssign.Var)
	c := addCopy(b, dst, val)
	c.Assign = typesAssign
	return b
}

// buildExpr builds the expression and returns its value and the new current BBlk.
// The returned Val is nil if the expression resulted in an EmptyType value.
func buildExpr(f *Fun, b *BBlk, expr types.Expr) (Val, *BBlk) {
	switch expr := expr.(type) {
	case *types.Call:
		return buildCall(f, b, expr)
	case *types.Ctor:
		return buildCtor(f, b, expr)
	case *types.Convert:
		return buildConvert(f, b, expr), b
	case *types.Block:
		return buildBlockLit(f, b, expr), b
	case *types.Ident:
		if expr.Capture {
			return buildCapture(f, b, expr.Var), b
		}
		return buildVar(f, b, expr.Var), b
	case *types.Int:
		i := addIntLit(f, b, expr.Type(), expr.Val)
		i.Int = expr
		return i, b
	case *types.Float:
		f := addFloatLit(f, b, expr.Type(), expr.Val)
		f.Float = expr
		return f, b
	case *types.String:
		str := addString(f.Mod, expr.Data)
		dst := addAlloc(f, b, expr.Type())
		s := addMakeString(b, dst, str)
		s.String = expr
		return dst, b
	default:
		panic(fmt.Sprintf("impossible: %T", expr))
	}
}

var builtInMethOp = map[types.BuiltInMeth]OpCode{
	types.ArraySizeMeth:  ArraySizeOp,
	types.BitwiseAndMeth: BitwiseAndOp,
	types.BitwiseOrMeth:  BitwiseOrOp,
	types.BitwiseNotMeth: BitwiseNotOp,
	types.RightShiftMeth: RightShiftOp,
	types.LeftShiftMeth:  LeftShiftOp,
	types.NegMeth:        NegOp,
	types.PlusMeth:       PlusOp,
	types.MinusMeth:      MinusOp,
	types.TimesMeth:      TimesOp,
	types.DivideMeth:     DivideOp,
	types.ModMeth:        ModOp,
	types.EqMeth:         EqOp,
	types.NeqMeth:        NeqOp,
	types.LessMeth:       LessOp,
	types.LessEqMeth:     LessEqOp,
	types.GreaterMeth:    GreaterOp,
	types.GreaterEqMeth:  GreaterEqOp,
	types.NumConvertMeth: NumConvertOp,
}

func buildCall(f *Fun, b *BBlk, call *types.Call) (Val, *BBlk) {
	var recv Val
	if call.Recv != nil {
		recv, b = buildExpr(f, b, call.Recv)
	}
	var val Val
	for i := range call.Msgs {
		switch msg := &call.Msgs[i]; {
		case builtInMethOp[msg.Fun.BuiltIn] > 0:
			val, b = buildOp(f, b, recv, msg)
		case msg.Fun.BuiltIn == types.ArrayLoadMeth:
			val, b = buildArrayLoad(f, b, recv, msg)
		case msg.Fun.BuiltIn == types.ArrayStoreMeth:
			b = buildArrayStore(f, b, recv, msg)
		case msg.Fun.BuiltIn == types.ArraySliceMeth:
			val, b = buildArraySlice(f, b, recv, msg)
		case msg.Fun.BuiltIn == types.CaseMeth:
			val, b = buildCaseMeth(f, b, recv, msg)
		default:
			val, b = buildMsg(f, b, recv, msg)
		}
	}
	return val, b
}

func buildMsg(f *Fun, b *BBlk, recv Val, msg *types.Msg) (Val, *BBlk) {
	var i int
	var args []Val
	if recv != nil {
		args = append(args, recv)
		i++
	}
	for _, arg := range msg.Args {
		var val Val
		val, b = buildExpr(f, b, arg)
		if val == nil {
			continue // empty type
		}
		i++
		if SimpleType(arg.Type()) {
			args = append(args, val)
			continue
		}
		a := addAlloc(f, b, arg.Type())
		addCopy(b, a, val)
		args = append(args, a)
	}
	var retVal Val
	var retType *types.Type
	if msg.Fun.Sig.Ret != nil && !EmptyType(msg.Fun.Sig.Ret.Type) {
		retType = msg.Fun.Sig.Ret.Type
		retVal = addAlloc(f, b, retType)
		args = append(args, retVal)
	}
	switch {
	case msg.Fun.BuiltIn == types.VirtMeth:
		c := addVirtCallFun(b, msg.Fun, args)
		c.Msg = msg
	default:
		fun := findFun(f.Mod, msg.Fun)
		c := addCall(b, fun, args)
		c.Msg = msg
	}
	if retType != nil && SimpleType(retType) {
		retVal = addLoad(f, b, retVal)
	}
	return retVal, b
}

func buildOp(f *Fun, b *BBlk, recv Val, msg *types.Msg) (Val, *BBlk) {
	args := make([]Val, 0, len(msg.Args)+1)

	// The incoming receiver is a &, but ops always take values.
	// Add a load of the receiver as the 0th arg.
	args = append(args, addLoad(f, b, recv))

	for _, arg := range msg.Args {
		var val Val
		val, b = buildExpr(f, b, arg)
		args = append(args, val)
	}

	code := builtInMethOp[msg.Fun.BuiltIn]
	o := addOp(f, b, msg.Type(), code, args...)
	o.Msg = msg
	return o, b
}

func buildArrayLoad(f *Fun, b *BBlk, recv Val, msg *types.Msg) (Val, *BBlk) {
	i, b := buildExpr(f, b, msg.Args[0])
	elm := addIndex(f, b, recv, i)
	elm.Msg = msg
	return elm, b
}

func buildArrayStore(f *Fun, b *BBlk, recv Val, msg *types.Msg) *BBlk {
	i, b := buildExpr(f, b, msg.Args[0])
	val, b := buildExpr(f, b, msg.Args[1])
	elm := addIndex(f, b, recv, i)
	elm.Msg = msg
	if SimpleType(elm.Type().Args[0].Type) {
		addStore(b, elm, val)
	} else {
		addCopy(b, elm, val)
	}
	return b
}

func buildArraySlice(f *Fun, b *BBlk, recv Val, msg *types.Msg) (Val, *BBlk) {
	start, b := buildExpr(f, b, msg.Args[0])
	end, b := buildExpr(f, b, msg.Args[1])
	dst := addAlloc(f, b, recv.Type().Args[0].Type)
	s := addMakeSlice(b, dst, recv, start, end)
	s.Msg = msg
	return dst, b
}

func buildCaseMeth(f *Fun, b *BBlk, recv Val, msg *types.Msg) (Val, *BBlk) {
	if !isRefType(recv) || len(refElemType(recv).Cases) == 0 {
		panic(fmt.Sprintf("case method on non-or-type-reference type %T", recv.Type()))
	}

	var args []Val
	for _, msgArg := range msg.Args {
		var arg Val
		arg, b = buildExpr(f, b, msgArg)
		args = append(args, arg)
	}

	var ret Val
	if !EmptyType(msg.Type()) {
		ret = addAlloc(f, b, msg.Type())
	}

	var tag Val
	orType := recv.Type().Args[0].Type
	if enumType(orType) {
		tag = addLoad(f, b, recv)
	} else {
		tag = addOp(f, b, orType.Tag(), UnionTagOp, recv)
	}
	var cases []*BBlk
	for i, arg := range args {
		bb := newBBlk(f)
		cases = append(cases, bb)
		valueArgs := []Val{arg}
		if orType.Cases[i].TypeName != nil {
			f := addField(f, bb, recv, i)
			valueArgs = append(valueArgs, f)
		}
		if ret != nil {
			valueArgs = append(valueArgs, ret)
		}
		addVirtCallIndex(bb, 0, valueArgs)
	}
	s := addSwitch(b, tag, cases, orType)
	s.Msg = msg

	b = newBBlk(f)
	for _, c := range cases {
		addJmp(c, b)
	}
	if ret != nil && SimpleType(msg.Type()) {
		return addLoad(f, b, ret), b
	}
	return ret, b
}

func buildCtor(f *Fun, b *BBlk, ctor *types.Ctor) (Val, *BBlk) {
	switch typ := ctor.Type().Args[0].Type; {
	case EmptyType(typ):
		return nil, b
	case typ.BuiltIn == types.ArrayType:
		return buildArrayCtor(f, b, ctor)
	case ctor.Case != nil:
		return buildOrCtor(f, b, ctor)
	default:
		return buildAndCtor(f, b, ctor)
	}
}

func buildAndCtor(f *Fun, b *BBlk, ctor *types.Ctor) (Val, *BBlk) {
	var args []Val
	for _, arg := range ctor.Args {
		var val Val
		val, b = buildExpr(f, b, arg)
		args = append(args, val)
	}
	andType := ctor.Type().Args[0].Type
	a := addAlloc(f, b, andType)
	mk := addMakeAnd(b, a, args)
	mk.Ctor = ctor
	return a, b
}

func buildArrayCtor(f *Fun, b *BBlk, ctor *types.Ctor) (Val, *BBlk) {
	var args []Val
	for _, arg := range ctor.Args {
		var val Val
		if val, b = buildExpr(f, b, arg); val == nil {
			continue
		}
		args = append(args, val)
	}
	aryType := ctor.Type().Args[0].Type
	a := addAlloc(f, b, aryType)
	mk := addMakeArray(b, a, args)
	mk.Ctor = ctor
	return a, b
}

func buildOrCtor(f *Fun, b *BBlk, ctor *types.Ctor) (Val, *BBlk) {
	var val Val
	switch orType := ctor.Type().Args[0].Type; {
	case len(ctor.Args) > 1:
		panic("impossible")
	case enumType(orType):
		var val *big.Int
		switch {
		// Bool is currently defined as {true|false},
		// but it would be very confusing if true=0 and false=1,
		// so special case these and swap them: true=1 and false=0.
		case orType.BuiltIn == types.BoolType && *ctor.Case == 0:
			val = big.NewInt(1)
		case orType.BuiltIn == types.BoolType && *ctor.Case == 1:
			val = big.NewInt(0)
		default:
			val = big.NewInt(int64(*ctor.Case))
		}
		// Constructors must result in reference types.
		// The optimization pass should eliminate redundante allocs.
		a := addAlloc(f, b, orType)
		i := addIntLit(f, b, orType, val)
		addStore(b, a, i)
		return a, b
	default:
		val, b = buildExpr(f, b, ctor.Args[0])
		fallthrough
	case len(ctor.Args) == 0:
		a := addAlloc(f, b, orType)
		mk := addMakeOr(b, a, *ctor.Case, val)
		mk.Ctor = ctor
		return a, b
	}
}

func buildConvert(f *Fun, b *BBlk, convert *types.Convert) Val {
	switch val, b := buildExpr(f, b, convert.Expr); {
	case convert.Ref > 0:
		if !SimpleType(convert.Expr.Type()) {
			// Non-simple types are already a reference.
			return val
		}
		a := addAlloc(f, b, val.Type())
		if val != nil {
			addStore(b, a, val)
		}
		return a
	case convert.Ref < 0:
		if val == nil {
			return nil
		}
		if !SimpleType(convert.Type()) {
			// Ignore converts that dereference to a composite type.
			// Composite types cannot fit in a register,
			// so they are passed around by reference.
			return val
		}
		l := addLoad(f, b, val)
		l.Convert = convert
		return l
	case len(convert.Virts) != 0:
		if SimpleType(val.Type()) {
			tmp := addAlloc(f, b, val.Type())
			addStore(b, tmp, val)
			val = tmp
		}
		a := addAlloc(f, b, convert.Type())
		v := addMakeVirt(f, b, a, val, convert.Virts)
		v.Convert = convert
		return a
	default:
		panic("impossible")
	}
}

// buildBlockLit builds a block literal.
// This is similar to constructing an And-type,
// where there is a field for each of the block's captures.
// However, block literals have an additional, unnamed field
// that captures the far-return value.
func buildBlockLit(f *Fun, b *BBlk, block *types.Block) Val {
	var args []Val
	for _, cap := range block.Captures {
		switch {
		case EmptyType(cap.Type()):
			args = append(args, nil)
		case findCapture(f, cap) >= 0:
			// We are in a nested block and this capture of the inner block
			// is a capture of its containing block too.
			args = append(args, buildCapture(f, b, cap))
		case cap.Local != nil:
			args = append(args, findLocal(f, cap))
		case cap.Field != nil:
			self := addLoad(f, b, selfParm(f, b))
			args = append(args, addField(f, b, self, cap.Index))
		case cap.FunParm != nil:
			fallthrough
		case cap.BlkParm != nil:
			args = append(args, findParm(f, b, cap))
		default:
			panic("impossible")
		}
	}

	// Store the far-return location as the last field of the block literal.
	switch {
	case f.Ret != nil && f.Block == nil:
		args = append(args, addArg(f, b, f.Ret))
	case f.Ret != nil && f.Block != nil:
		// We are in a nested block.
		// The far return location  is a capture
		// of the containing block.
		self := addLoad(f, b, selfParm(f, b))
		retPtr := addField(f, b, self, len(f.Block.Captures))
		ret := addLoad(f, b, retPtr)
		args = append(args, ret)
	}

	blk := addAlloc(f, b, block.BlockType)
	addMakeAnd(b, blk, args)

	fun := buildBlockFun(f.Mod, f.Fun, block)
	virt := addAlloc(f, b, block.Type())
	addStmt(b, &MakeVirt{Dst: virt, Obj: blk, Virts: []*Fun{fun}})
	return virt
}

func findCapture(fun *Fun, vr *types.Var) int {
	if fun.Block == nil {
		return -1
	}
	for i, cap := range fun.Block.Captures {
		if cap == vr {
			return i
		}
	}
	return -1
}

func buildCapture(f *Fun, b *BBlk, vr *types.Var) Val {
	if EmptyType(vr.Type()) {
		return nil
	}
	self := addLoad(f, b, selfParm(f, b))
	capPtr := addField(f, b, self, findCapture(f, vr))
	return addLoad(f, b, capPtr)
}

func buildVar(f *Fun, b *BBlk, vr *types.Var) Val {
	if EmptyType(vr.Type()) {
		return nil
	}
	switch {
	case vr.Val != nil:
		return addGlobal(f, b, vr.Val)
	case vr.FunParm != nil:
		return findParm(f, b, vr)
	case vr.BlkParm != nil:
		return findParm(f, b, vr)
	case vr.Local != nil:
		return findLocal(f, vr)
	case vr.Field != nil:
		self := addLoad(f, b, selfParm(f, b))
		return addField(f, b, self, vr.Index)
	case vr.Case != nil:
		panic("impossible")
	default:
		panic("impossible")
	}
}

func findParm(f *Fun, b *BBlk, vr *types.Var) Val {
	for _, stmt := range f.BBlks[0].Stmts {
		if a, ok := stmt.(*Alloc); ok && a.Var == vr {
			return a
		}
	}
	for _, p := range f.Parms {
		if p.Var == vr {
			return addArg(f, b, p)
		}
	}
	// Note that vr cannot match the fun.Ret parm,
	// since that does not correspond to a types.Var.
	panic("imposible")
}

func selfParm(f *Fun, b *BBlk) Val {
	return findParm(f, b, f.Parms[0].Var)
}

func findLocal(fun *Fun, vr *types.Var) *Alloc {
	for _, stmt := range fun.BBlks[0].Stmts {
		if a, ok := stmt.(*Alloc); ok && a.Var == vr {
			return a
		}
	}
	panic("impossible")
}

func addStmt(b *BBlk, s Stmt) {
	if n := len(b.Stmts); n > 0 {
		if _, ok := b.Stmts[n-1].(Term); ok {
			panic("impossible")
		}
	}
	b.Stmts = append(b.Stmts, s)
	for _, v := range s.Uses() {
		v.value().addUser(s)
	}
	if term, ok := s.(Term); ok {
		for _, o := range term.Out() {
			o.addIn(b)
		}
	}
}

func addComment(b *BBlk, f string, vs ...interface{}) {
	addStmt(b, &Comment{Text: fmt.Sprintf(f, vs...)})
}

func addStore(b *BBlk, dst, val Val) *Store {
	s := &Store{Dst: dst, Val: val}
	addStmt(b, s)
	return s
}

func addCopy(b *BBlk, dst, src Val) *Copy {
	c := &Copy{Dst: dst, Src: src}
	addStmt(b, c)
	return c
}

func addMakeArray(b *BBlk, dst Val, args []Val) *MakeArray {
	s := &MakeArray{Dst: dst, Args: args}
	addStmt(b, s)
	return s
}

func addMakeSlice(b *BBlk, dst, ary, from, to Val) *MakeSlice {
	s := &MakeSlice{Dst: dst, Ary: ary, From: from, To: to}
	addStmt(b, s)
	return s
}

func addMakeString(b *BBlk, dst Val, str *String) *MakeString {
	s := &MakeString{Dst: dst, Data: str}
	addStmt(b, s)
	return s
}

func addMakeAnd(b *BBlk, dst Val, args []Val) *MakeAnd {
	s := &MakeAnd{Dst: dst, Fields: args}
	addStmt(b, s)
	return s
}

func addMakeOr(b *BBlk, dst Val, tag int, val Val) *MakeOr {
	s := &MakeOr{Dst: dst, Case: tag, Val: val}
	addStmt(b, s)
	return s
}

func addMakeVirt(f *Fun, b *BBlk, dst, obj Val, typesVirts []*types.Fun) *MakeVirt {
	var virts []*Fun
	for _, fun := range typesVirts {
		virts = append(virts, findFun(f.Mod, fun))
	}
	v := &MakeVirt{Dst: dst, Obj: obj, Virts: virts}
	addStmt(b, v)
	return v
}

func addCall(b *BBlk, fun *Fun, args []Val) *Call {
	c := &Call{Fun: fun, Args: args}
	addStmt(b, c)
	return c
}

func addVirtCallFun(b *BBlk, fun *types.Fun, args []Val) *VirtCall {
	index := -1
	if args[0].Type().BuiltIn != types.RefType {
		return addVirtCallIndex(b, -1, args)
	}
	virtType := args[0].Type().Args[0].Type
	for i, v := range virtType.Virts {
		if v.Sel == fun.Sig.Sel {
			index = i
			break
		}
	}
	return addVirtCallIndex(b, index, args)
}

func addVirtCallIndex(b *BBlk, index int, args []Val) *VirtCall {
	c := &VirtCall{Self: args[0], Index: index, Args: args}
	addStmt(b, c)
	return c
}

func addRet(b *BBlk) *Ret {
	r := &Ret{}
	addStmt(b, r)
	return r
}

func addJmp(b, dst *BBlk) { addStmt(b, &Jmp{Dst: dst}) }

func addSwitch(b *BBlk, val Val, dsts []*BBlk, typ *types.Type) *Switch {
	s := &Switch{Val: val, Dsts: dsts, OrType: typ}
	addStmt(b, s)
	return s
}

func addIntLit(f *Fun, b *BBlk, typ *types.Type, val *big.Int) *IntLit {
	i := &IntLit{val: newVal(f, typ), Val: val}
	addStmt(b, i)
	return i
}

func addFloatLit(f *Fun, b *BBlk, typ *types.Type, val *big.Float) *FloatLit {
	float := &FloatLit{val: newVal(f, typ), Val: val}
	addStmt(b, float)
	return float
}

func addOp(f *Fun, b *BBlk, typ *types.Type, code OpCode, args ...Val) *Op {
	o := &Op{val: newVal(f, typ), Code: code, Args: args}
	addStmt(b, o)
	return o
}

func addArg(f *Fun, b *BBlk, p *Parm) Val {
	a := &Arg{val: newVal(f, p.Type), Parm: p}
	addStmt(b, a)
	return a
}

func addLoad(f *Fun, b *BBlk, src Val) *Load {
	l := &Load{val: newVal(f, src.Type()), Src: src}
	if src.Type().BuiltIn == types.RefType {
		l.val.typ = src.Type().Args[0].Type
	}
	addStmt(b, l)
	return l
}

func addAlloc(f *Fun, b *BBlk, typ *types.Type) *Alloc {
	a := &Alloc{val: newVal(f, typ.Ref())}
	addStmt(f.BBlks[0], a)
	return a
}

func addGlobal(f *Fun, b *BBlk, val *types.Val) *Global {
	g := &Global{val: newVal(f, val.Var.Type().Ref()), Val: val}
	addStmt(b, g)
	return g
}

func addIndex(f *Fun, b *BBlk, ary, i Val) *Index {
	v := &Index{val: newVal(f, ary.Type()), Ary: ary, Index: i}
	if ary.Type().BuiltIn == types.RefType &&
		ary.Type().Args[0].Type.BuiltIn == types.ArrayType {
		// If ary is indeed an Array&, then this is the element type.
		v.val.typ = ary.Type().Args[0].Type.Args[0].Type.Ref()
	}
	addStmt(b, v)
	return v
}

func addField(f *Fun, b *BBlk, obj Val, i int) *Field {
	var typ *types.Type
	field := &Field{val: newVal(f, obj.Type()), Obj: obj, Index: i}
	if obj.Type().BuiltIn == types.RefType {
		switch objType := obj.Type().Args[0].Type; {
		case objType.BuiltIn == types.BlockType:
			if i >= len(objType.Fields) {
				// Block return value capture.
				typ = f.Fun.Sig.Ret.Type.Ref()
			} else {
				field.Field = &objType.Fields[i]
				typ = field.Field.Type()
			}
		case len(objType.Fields) > 0:
			field.Field = &objType.Fields[i]
			typ = field.Field.Type()
		case len(objType.Cases) > 0:
			field.Case = &objType.Cases[i]
			typ = field.Case.Type()
		default:
			typ = objType
		}
		field.val.typ = typ.Ref()
	}
	addStmt(b, field)
	return field
}
