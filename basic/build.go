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
	fun := findFun(mod, typesFun)
	buildFunBody(fun, typesFun.Locals, typesFun.Stmts)
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

	buildFunBody(f, block.Locals, block.Stmts)
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

func buildFunBody(f *Fun, locals []*types.Var, stmts []types.Stmt) {
	b0 := newBBlk(f)
	for _, local := range locals {
		a := addAlloc(f, b0, local.Type())
		a.Var = local
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
		selfParm := addArg(f, b, findSelf(f))
		self := addLoad(f, b, selfParm)
		dstPtr := addField(f, b, self, len(f.Block.Captures))
		dst = addLoad(f, b, dstPtr)
	} else {
		arg := addArg(f, b, f.Ret)
		dst = addLoad(f, b, arg)
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
	arg := addArg(f, b, f.Ret)
	dst := addLoad(f, b, arg)
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

	for i, arg := range msg.Args {
		var val Val
		val, b = buildExpr(f, b, arg)
		panicIf(val == nil || EmptyType(val.Type()), "op arg %d is an empty type", i)
		panicIf(!SimpleType(val.Type()), "op arg %d is a composite type", i)
		panicIf(val.Type().BuiltIn == types.RefType, "op arg %d is a reference type", i)
		args = append(args, val)
	}

	code := builtInMethOp[msg.Fun.BuiltIn]
	o := addOp(f, b, msg.Type(), code, args...)
	o.Msg = msg
	return o, b
}

func buildArrayLoad(f *Fun, b *BBlk, recv Val, msg *types.Msg) (Val, *BBlk) {
	panicIf(len(msg.Args) != 1, "array load got %d args, wanted 1", len(msg.Args))
	i, b := buildExpr(f, b, msg.Args[0])
	elm := addIndex(f, b, recv, i)
	elm.Msg = msg
	return elm, b
}

func buildArrayStore(f *Fun, b *BBlk, recv Val, msg *types.Msg) *BBlk {
	panicIf(len(msg.Args) != 2, "array store got %d args, wanted 2", len(msg.Args))
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
	panicIf(len(msg.Args) != 2, "array slice got %d args, wanted 2", len(msg.Args))
	start, b := buildExpr(f, b, msg.Args[0])
	end, b := buildExpr(f, b, msg.Args[1])
	dst := addAlloc(f, b, recv.Type().Args[0].Type)
	s := addMakeSlice(b, dst, recv, start, end)
	s.Msg = msg
	return dst, b
}

func buildCaseMeth(f *Fun, b *BBlk, recv Val, msg *types.Msg) (Val, *BBlk) {
	panicIf(recv.Type().BuiltIn != types.RefType,
		"case method on non-reference receiver type %T", recv.Type())
	orType := recv.Type().Args[0].Type
	panicIf(len(orType.Cases) == 0,
		"case method on non-or-type-reference type %T", recv.Type())
	panicIf(len(orType.Cases) != len(msg.Args),
		"case method argument count mismatch: got %d, want %d",
		len(msg.Args), len(orType.Cases))

	var args []Val
	for i, msgArg := range msg.Args {
		var arg Val
		arg, b = buildExpr(f, b, msgArg)
		panicIf(arg.Type().BuiltIn != types.RefType,
			"case method argument %d is non-reference type %s",
			i, arg.Type())
		panicIf(arg.Type().Args[0].Type.BuiltIn != types.FunType,
			"case method argument %d is non-Fun-reference type %s",
			i, arg.Type().Args[0].Type)
		args = append(args, arg)
	}

	var ret Val
	if !EmptyType(msg.Type()) {
		ret = addAlloc(f, b, msg.Type())
	}

	tag := addOp(f, b, orType.Tag().Ref(), UnionTagOp, recv)
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
	switch {
	case len(ctor.Args) > 1:
		panic("impossible")
	default:
		val, b = buildExpr(f, b, ctor.Args[0])
		fallthrough
	case len(ctor.Args) == 0:
		orType := ctor.Type().Args[0].Type
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
			selfParm := addArg(f, b, findSelf(f))
			self := addLoad(f, b, selfParm)
			args = append(args, addField(f, b, self, cap.Index))
		case cap.FunParm != nil:
			fallthrough
		case cap.BlkParm != nil:
			args = append(args, addArg(f, b, findParm(f, cap)))
		default:
			panic("impossible")
		}
	}

	// Store the far-return location as the last field of the block literal.
	switch {
	case f.Ret != nil && f.Block == nil:
		v := addArg(f, b, f.Ret)
		args = append(args, addLoad(f, b, v))
	case f.Ret != nil && f.Block != nil:
		// We are in a nested block.
		// The far return location  is a capture
		// of the containing block.
		selfParm := addArg(f, b, findSelf(f))
		self := addLoad(f, b, selfParm)
		retPtr := addField(f, b, self, len(f.Block.Captures))
		ret := addLoad(f, b, retPtr)
		args = append(args, ret)
	}

	blk := addAlloc(f, b, block.BlockType)
	addMakeAnd(b, blk, args)

	fun := buildBlockFun(f.Mod, f.Fun, block)
	virt := addAlloc(f, b, block.Type())
	addStmt(b, newMakeVirt(virt, blk, []*Fun{fun}))
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

	i := findCapture(f, vr)
	panicIf(i < 0, "block %s has no capture %s", f.Block.BlockType, vr.Name)

	selfParm := addArg(f, b, findSelf(f))
	self := addLoad(f, b, selfParm)
	capPtr := addField(f, b, self, i)
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
		return addArg(f, b, findParm(f, vr))
	case vr.BlkParm != nil:
		return addArg(f, b, findParm(f, vr))
	case vr.Local != nil:
		return findLocal(f, vr)
	case vr.Field != nil:
		selfParm := addArg(f, b, findSelf(f))
		self := addLoad(f, b, selfParm)
		return addField(f, b, self, vr.Index)
	case vr.Case != nil:
		panic("impossible")
	default:
		panic("impossible")
	}
}

func findSelf(fun *Fun) *Parm {
	return fun.Parms[0]
}

func findParm(fun *Fun, vr *types.Var) *Parm {
	for _, p := range fun.Parms {
		if p.Var == vr {
			return p
		}
	}
	// Note that vr cannot match the fun.Ret parm,
	// since that does not correspond to a types.Var.
	panic("imposible")
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

// TODO: change the newXyz/panicIf convention to addXyz/Xyz.check().
// Instead of having addXyz, which calls newXyz, which may panic,
// give each Xyz a check() method of some kind.
// One option would be to have it behave like comment(),
// changing the output string, which will cause diffs in tests.

func addStore(b *BBlk, dst, val Val) *Store {
	s := newStore(dst, val)
	addStmt(b, s)
	return s
}

func newStore(dst, val Val) *Store {
	panicIf(dst.Type().BuiltIn != types.RefType,
		"store to non-reference type %s", dst.Type())
	panicIf(dst.Type().Args[0].Type != val.Type(),
		"store type mismatch: %s != %s",
		dst.Type().Args[0].Type, val.Type())
	return &Store{Dst: dst, Val: val}
}

func addCopy(b *BBlk, dst, src Val) *Copy {
	c := newCopy(dst, src)
	addStmt(b, c)
	return c
}

func newCopy(dst, src Val) *Copy {
	panicIf(dst.Type().BuiltIn != types.RefType,
		"copy to non-reference type %s", dst.Type())
	panicIf(src.Type().BuiltIn != types.RefType,
		"copy from non-reference type %s", src.Type())
	panicIf(dst.Type() != src.Type(),
		"copy type mismatch: %s != %s", dst.Type(), src.Type())
	return &Copy{Dst: dst, Src: src}
}

func addMakeSlice(b *BBlk, dst, ary, from, to Val) *MakeSlice {
	s := newMakeSlice(dst, ary, from, to)
	addStmt(b, s)
	return s
}

func newMakeSlice(dst, ary, from, to Val) *MakeSlice {
	panicIf(dst.Type().BuiltIn != types.RefType,
		"make slice to non-reference type %s", dst.Type())
	panicIf(dst.Type().Args[0].Type.BuiltIn != types.ArrayType,
		"make slice to non-array-reference type %s",
		dst.Type().Args[0].Type)
	panicIf(ary.Type().BuiltIn != types.RefType,
		"make slice from non-reference type %s", ary.Type())
	panicIf(ary.Type().Args[0].Type.BuiltIn != types.ArrayType,
		"make slice from non-array-reference type %s",
		ary.Type().Args[0].Type)
	panicIf(from.Type().BuiltIn != types.IntType,
		"make slice non-Int start type %s", from.Type())
	panicIf(to.Type().BuiltIn != types.IntType,
		"make slice non-Int end type %s", to.Type())
	return &MakeSlice{Dst: dst, Ary: ary, From: from, To: to}
}

func addMakeArray(b *BBlk, dst Val, args []Val) *MakeArray {
	s := newMakeArray(dst, args)
	addStmt(b, s)
	return s
}

func newMakeArray(dst Val, args []Val) *MakeArray {
	panicIf(dst.Type().BuiltIn != types.RefType,
		"make array of non-reference type %s", dst.Type())
	panicIf(dst.Type().Args[0].Type.BuiltIn != types.ArrayType,
		"make string of non-array-reference type %s",
		dst.Type().Args[0].Type)
	return &MakeArray{Dst: dst, Args: args}
}

func addMakeString(b *BBlk, dst Val, str *String) *MakeString {
	s := newMakeString(dst, str)
	addStmt(b, s)
	return s
}

func newMakeString(dst Val, str *String) *MakeString {
	panicIf(dst.Type().BuiltIn != types.RefType,
		"make string of non-reference type %s", dst.Type())
	panicIf(dst.Type().Args[0].Type.BuiltIn != types.StringType,
		"make string of non-string-reference type %s",
		dst.Type().Args[0].Type)
	return &MakeString{Dst: dst, Data: str}
}

func addMakeAnd(b *BBlk, dst Val, args []Val) *MakeAnd {
	s := newMakeAnd(dst, args)
	addStmt(b, s)
	return s
}

func newMakeAnd(dst Val, args []Val) *MakeAnd {
	panicIf(dst.Type().BuiltIn != types.RefType,
		"make and of non-reference type %s", dst.Type())
	andType := dst.Type().Args[0].Type

	for i := range andType.Fields {
		panicIf(i >= len(args), "make and too few args")
		arg := args[i]
		field := &andType.Fields[i]
		if arg == nil {
			panicIf(!EmptyType(field.Type()) &&
				// For block literals, we elide empty-type captures.
				// But captures always have one extra level of &,
				// so we have to account for that in this check.
				(andType.BuiltIn != types.BlockType ||
					field.Type().BuiltIn != types.RefType ||
					!EmptyType(field.Type().Args[0].Type)),
				"make and field %d type mismatch: got nil, want %s",
				i, field.Type())
		} else {
			panicIf(EmptyType(field.Type()) && arg != nil,
				"make and field %d type mismatch: got %s, want nil",
				i, arg.Type())
			panicIf(field.Type() != arg.Type(),
				"make and field %d type mismatch: got %s, want %s",
				i, arg.Type(), field.Type())
		}
	}
	return &MakeAnd{Dst: dst, Fields: args}
}

func addMakeOr(b *BBlk, dst Val, tag int, val Val) *MakeOr {
	s := newMakeOr(dst, tag, val)
	addStmt(b, s)
	return s
}

func newMakeOr(dst Val, tag int, val Val) *MakeOr {
	panicIf(dst.Type().BuiltIn != types.RefType,
		"make or of non-reference type %s", dst.Type())
	orType := dst.Type().Args[0].Type
	panicIf(len(orType.Cases) <= tag,
		"make or bad tag: %d, but only %d cases", tag, len(orType.Cases))
	c := &orType.Cases[tag]
	panicIf(c.TypeName != nil && !EmptyType(c.Type()) && val == nil,
		"make or type mismatch: got nil, want %s", c.Type())
	if val != nil {
		panicIf(c.TypeName == nil,
			"make or type mismatch: got %s, want nil", val.Type())
		panicIf(c.TypeName != nil && c.Type() != val.Type(),
			"make or type mismatch: got %s, want %s",
			val.Type(), c.Type())
	}
	return &MakeOr{Dst: dst, Case: tag, Val: val}
}

func addMakeVirt(f *Fun, b *BBlk, dst, obj Val, typesVirts []*types.Fun) *MakeVirt {
	var virts []*Fun
	for _, fun := range typesVirts {
		virts = append(virts, findFun(f.Mod, fun))
	}
	v := newMakeVirt(dst, obj, virts)
	addStmt(b, v)
	return v
}

func newMakeVirt(dst, obj Val, virts []*Fun) *MakeVirt {
	panicIf(dst.Type().BuiltIn != types.RefType,
		"make virt with non-reference dest %s", dst.Type())
	virtType := dst.Type().Args[0].Type
	panicIf(len(virts) != len(virtType.Virts),
		"make virt count mismatch: got %d, want %d",
		len(virts), len(virtType.Virts))
	panicIf(obj.Type().BuiltIn != types.RefType,
		"make virt with non-reference obj %s", obj.Type())
	return &MakeVirt{Dst: dst, Obj: obj, Virts: virts}
}

func addCall(b *BBlk, fun *Fun, args []Val) *Call {
	c := newCall(fun, args)
	addStmt(b, c)
	return c
}

func newCall(fun *Fun, args []Val) *Call {
	parms := fun.Parms
	if fun.Ret != nil {
		parms = append(parms, fun.Ret)
	}
	panicIf(len(args) != len(parms),
		"call argument count mismatch: got %d, want %d",
		len(args), len(parms))
	for i, a := range args {
		panicIf(a.Type() != parms[i].Type,
			"argument %d type mismatch: got %s, want %s",
			i, a.Type(), parms[i].Type)
	}
	return &Call{Fun: fun, Args: args}
}

func addVirtCallFun(b *BBlk, fun *types.Fun, args []Val) *VirtCall {
	c := newVirtCallFun(fun, args)
	addStmt(b, c)
	return c
}

func newVirtCallFun(fun *types.Fun, args []Val) *VirtCall {
	recv := args[0]
	panicIf(recv.Type().BuiltIn != types.RefType,
		"virtual call to non-reference type %s", recv.Type())
	virtType := recv.Type().Args[0].Type
	panicIf(len(virtType.Virts) == 0,
		"virtual call to non-virt-reference type %s", virtType)
	index := -1
	for i, v := range virtType.Virts {
		if v.Sel == fun.Sig.Sel {
			index = i
			break
		}
	}
	panicIf(index < 0, "virtual call to non-existent method %s of %s",
		virtType, fun.Sig.Sel)
	return newVirtCallIndex(index, args)
}

func addVirtCallIndex(b *BBlk, index int, args []Val) *VirtCall {
	c := newVirtCallIndex(index, args)
	addStmt(b, c)
	return c
}

func newVirtCallIndex(index int, args []Val) *VirtCall {
	recv, checkArgs := args[0], args[1:]
	panicIf(recv.Type().BuiltIn != types.RefType,
		"virtual call to non-reference type %s", recv.Type())
	virtType := recv.Type().Args[0].Type
	panicIf(index >= len(virtType.Virts),
		"virtual call to non-existent method index=%d of %s",
		index, virtType)
	virt := virtType.Virts[index]
	if virt.Ret != nil && !EmptyType(virt.Ret.Type) {
		// strip return value location
		checkArgs = checkArgs[:len(checkArgs)-1]
	}
	panicIf(len(checkArgs) != len(virt.Parms),
		"virtual call argument count mismatch: got %d, want %d",
		len(checkArgs), len(virt.Parms))
	for i, a := range checkArgs {
		panicIf(a.Type() != virt.Parms[i].Type(),
			"argument %d type mismatch: got %s, want %s",
			i, a.Type(), virt.Parms[i].Type())
	}
	return &VirtCall{Self: recv, Index: index, Args: args}
}

func addRet(b *BBlk) *Ret {
	r := &Ret{}
	addStmt(b, r)
	return r
}

func addSwitch(b *BBlk, val Val, dsts []*BBlk, typ *types.Type) *Switch {
	panicIf(len(typ.Cases) != len(dsts),
		"switch case count mismatch: got %d, want %d",
		len(dsts), len(typ.Cases))
	s := &Switch{Val: val, Dsts: dsts, OrType: typ}
	addStmt(b, s)
	return s
}

func addJmp(b, dst *BBlk) { addStmt(b, &Jmp{Dst: dst}) }

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
	a := &Arg{val: newVal(f, p.Type.Ref()), Parm: p}
	addStmt(b, a)
	if p.Value {
		// By-value parameters have an extra level of reference
		// that is not accounted for by the deref types.Converts.
		// Strip it.
		return addLoad(f, b, a)
	}
	return a
}

func addLoad(f *Fun, b *BBlk, src Val) *Load {
	l := newLoad(src)
	l.val = newVal(f, src.Type().Args[0].Type)
	addStmt(b, l)
	return l
}

func newLoad(src Val) *Load {
	panicIf(src.Type().BuiltIn != types.RefType,
		"load from non-reference type %s", src.Type())
	panicIf(!SimpleType(src.Type().Args[0].Type),
		"load a composite type %s",
		src.Type().Args[0].Type)
	return &Load{Src: src}
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

func addIndex(f *Fun, b *BBlk, obj, i Val) *Index {
	v := newIndex(obj, i)
	aryType := obj.Type().Args[0].Type
	elmType := aryType.Args[0].Type
	v.val = newVal(f, elmType.Ref())
	addStmt(b, v)
	return v
}

func newIndex(ary, i Val) *Index {
	typ := ary.Type()
	panicIf(typ.BuiltIn != types.RefType,
		"index of non-reference type %s", typ)
	aryType := typ.Args[0].Type
	panicIf(aryType.BuiltIn != types.ArrayType,
		"index of non-array reference type %s", typ)
	panicIf(i.Type().BuiltIn != types.IntType,
		"index with non-Int index type %s", i.Type())
	return &Index{Ary: ary, Index: i}
}

func addField(f *Fun, b *BBlk, obj Val, i int) *Field {
	panicIf(obj.Type().BuiltIn != types.RefType,
		"field of non-reference type %s", obj.Type())
	var typ *types.Type
	field := &Field{Obj: obj, Index: i}
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
		panic(fmt.Sprintf("type %s has no field or case %d", typ, i))
	}
	field.val = newVal(f, typ.Ref())
	addStmt(b, field)
	return field
}

func panicIf(c bool, f string, vs ...interface{}) {
	if c {
		panic(fmt.Sprintf(f, vs...))
	}
}
