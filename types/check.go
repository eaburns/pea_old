package types

import (
	"fmt"
	"math/big"
	"path"
	"strings"

	"github.com/eaburns/pea/ast"
)

// Config are configuration parameters for the type checker.
type Config struct {
	// IntSize is the bit size of the Int, UInt, and Word alias types.
	// It must be a valid int size: 8, 16, 32, or 64 (default=64).
	IntSize int
	// FloatSize is the bit size of the Float type alias.
	// It must be a valid float size: 32 or 64 (default=64).
	FloatSize int
	// Importer is used for importing modules.
	// The default importer reads packages from the local file system.
	Importer Importer
	// Trace is whether to enable debug tracing.
	Trace bool
}

// Check type-checks an AST and returns the type-checked tree or errors.
func Check(astMod *ast.Mod, cfg Config) (*Mod, []error) {
	x := newUnivScope(newDefaultState(cfg, astMod))
	mod, errs := check(x, astMod)
	if len(errs) > 0 {
		return nil, convertErrors(errs)
	}
	return mod, nil
}

func check(x *scope, astMod *ast.Mod) (_ *Mod, errs []checkError) {
	defer x.tr("check(%s)", astMod.Name)(&errs)

	isUniv := x.univ == nil

	mod := &Mod{AST: astMod}
	x = x.new()
	x.mod = mod

	mod.Defs, errs = makeDefs(x, astMod.Files)
	errs = append(errs, checkDups(x, mod.Defs)...)
	errs = append(errs, gatherDefs(x, mod.Defs)...)
	if isUniv {
		// In this case, we are checking the univ mod.
		// We've only now just gathered the defs, so set them in the state.
		x.up.univ = mod.Defs
	}
	mod.Defs = append(mod.Defs, builtInMeths(x, mod.Defs)...)
	if isUniv {
		// In this case, we are checking the univ mod.
		// Add the additional built-in defs to the state.
		x.up.univ = mod.Defs
	}
	errs = append(errs, checkDupMeths(x, mod.Defs)...)
	errs = append(errs, checkDefs(x, mod.Defs)...)

	return mod, errs
}

func makeDefs(x *scope, files []ast.File) ([]Def, []checkError) {
	var defs []Def
	var errs []checkError
	for i := range files {
		file := &file{ast: &files[i]}
		file.x = x.new()
		file.x.file = file
		errs = append(errs, imports(x.state, file)...)
		for _, astDef := range file.ast.Defs {
			def := makeDef(astDef)
			defs = append(defs, def)
			x.defFiles[def] = file
		}
	}
	return defs, errs
}

func imports(x *state, file *file) []checkError {
	var errs []checkError
	for _, astImp := range file.ast.Imports {
		p := astImp.Path[1 : len(astImp.Path)-1] // trim "
		x.log("importing %s", p)
		defs, err := x.cfg.Importer.Import(x.cfg, p)
		if err != nil {
			errs = append(errs, *x.err(astImp, err.Error()))
			continue
		}
		file.imports = append(file.imports, imp{
			path: p,
			name: path.Base(p),
			defs: defs,
		})
	}
	return errs
}

// checkDups returns redefinition errors for types, vals, and funs.
// It doesn't check duplicate methods.
func checkDups(x *scope, defs []Def) (errs []checkError) {
	defer x.tr("checkDups")(&errs)

	seen := make(map[string]Def)
	types := make(map[string]Def)
	for _, def := range defs {
		var id string
		switch def := def.(type) {
		case *Val:
			id = def.Var.Name
		case *Type:
			id = def.Sig.Name
			tid := fmt.Sprintf("(%d)%s", def.Sig.Arity, def.Sig.Name)
			if prev, ok := types[tid]; ok {
				err := x.err(def, "type %s redefined", tid)
				note(err, "previous definition is at %s", x.loc(prev))
				errs = append(errs, *err)
				continue
			}
			types[tid] = def
			if _, ok := seen[id].(*Type); ok {
				// Multiple defs of the same type name are OK
				// as long as their arity is different.
				continue
			}
		case *Fun:
			if astFun, ok := def.ast.(*ast.Fun); ok && astFun.Recv != nil {
				continue // check dup methods separately.
			}
			id = def.Sig.Sel
		default:
			panic(fmt.Sprintf("impossible type %T", def))
		}
		if prev, ok := seen[id]; ok {
			err := x.err(def, "%s redefined", id)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		}
		seen[id] = def
	}
	return errs
}

func makeDef(astDef ast.Def) Def {
	switch astDef := astDef.(type) {
	case *ast.Val:
		val := &Val{
			ast:  astDef,
			Priv: astDef.Priv(),
			Var: Var{
				ast:  &astDef.Var,
				Name: astDef.Var.Name,
			},
		}
		val.Var.Val = val
		return val
	case *ast.Fun:
		return &Fun{
			ast:  astDef,
			Priv: astDef.Priv(),
			Sig: FunSig{
				ast: &astDef.Sig,
				Sel: astDef.Sig.Sel,
			},
		}
	case *ast.Type:
		return &Type{
			ast:  astDef,
			Priv: astDef.Priv(),
			Sig: TypeSig{
				ast:   &astDef.Sig,
				Arity: len(astDef.Sig.Parms),
				Name:  astDef.Sig.Name,
			},
		}
	default:
		panic(fmt.Sprintf("impossible type %T", astDef))
	}
}

func checkDupMeths(x *scope, defs []Def) []checkError {
	var errs []checkError
	seen := make(map[string]Def)
	for _, def := range defs {
		fun, ok := def.(*Fun)
		if !ok || fun.Recv == nil || fun.Recv.Type == nil {
			continue
		}
		recv := fun.Recv.Type
		key := recv.name() + " " + fun.Sig.Sel
		if prev, ok := seen[key]; ok {
			err := x.err(def, "method %s redefined", key)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[key] = def
		}
	}
	return errs
}

func checkDefs(x *scope, defs []Def) []checkError {
	var errs []checkError
	for _, def := range defs {
		errs = append(errs, checkDef(x, def)...)
	}
	return errs
}

func checkDef(x *scope, def Def) []checkError {
	if !x.gathered[def] {
		// This is a built-in method, with no AST and nothing to check.
		return nil
	}
	file, ok := x.defFiles[def]
	if !ok {
		panic("impossible")
	}
	x = file.x

	switch def := def.(type) {
	case *Val:
		return checkVal(x, def)
	case *Fun:
		return checkFun(x, def)
	case *Type:
		return checkType(x, def)
	default:
		panic(fmt.Sprintf("impossible type: %T", def))
	}
}

func checkVal(x *scope, def *Val) (errs []checkError) {
	defer x.tr("checkVal(%s)", def.name())(&errs)
	if def.Var.TypeName != nil {
		errs = append(errs, checkTypeName(x, def.Var.TypeName)...)
		def.Var.typ = def.Var.TypeName.Type
	}

	x = x.new()
	x.val = def

	var es []checkError
	def.Init, es = checkStmts(x, def.Var.typ, def.ast.Init)
	return append(errs, es...)
}

func checkFun(x *scope, def *Fun) (errs []checkError) {
	defer x.tr("checkFun(%s)", def.name())(&errs)
	if def.Recv != nil {
		for i := range def.Recv.Parms {
			x = x.new()
			x.typeVar = def.Recv.Parms[i].TypeVar
		}
	}
	for i := range def.TParms {
		x = x.new()
		x.typeVar = def.TParms[i].TypeVar
	}

	x = x.new()
	x.fun = def
	for i := range def.Sig.Parms {
		parm := &def.Sig.Parms[i]
		errs = append(errs, checkTypeName(x, parm.TypeName)...)
		x = x.new()
		x.variable = parm
	}

	var es []checkError
	def.Stmts, es = checkStmts(x, nil, def.ast.(*ast.Fun).Stmts)
	return append(errs, es...)
}

func checkType(x *scope, def *Type) (errs []checkError) {
	defer x.tr("checkType(%s)", def.name())(&errs)
	switch {
	case def.Alias != nil:
		errs = checkTypeName(x, def.Alias)
	case def.Fields != nil:
		errs = checkFields(x, def.Fields)
	case def.Cases != nil:
		errs = checkCases(x, def.Cases)
	case def.Virts != nil:
		errs = checkVirts(x, def.Virts)
	}
	return errs
}

func checkFields(x *scope, fields []Var) []checkError {
	var errs []checkError
	seen := make(map[string]*Var)
	for i := range fields {
		field := &fields[i]
		if prev, ok := seen[field.Name]; ok {
			err := x.err(field, "field %s redefined", field.Name)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[field.Name] = field
		}
		errs = append(errs, checkTypeName(x, field.TypeName)...)
	}
	return errs
}

func checkCases(x *scope, cases []Var) []checkError {
	var errs []checkError
	seen := make(map[string]*Var)
	for i := range cases {
		cas := &cases[i]
		if prev, ok := seen[cas.Name]; ok {
			err := x.err(cas, "case %s redefined", cas.Name)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[cas.Name] = cas
		}
		if cas.TypeName != nil {
			errs = append(errs, checkTypeName(x, cas.TypeName)...)
		}
	}
	return errs
}

func checkVirts(x *scope, virts []FunSig) []checkError {
	var errs []checkError
	seen := make(map[string]*FunSig)
	for i := range virts {
		virt := &virts[i]
		if prev, ok := seen[virt.Sel]; ok {
			err := x.err(virt, "virtual method %s redefined", virt.Sel)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[virt.Sel] = virt
		}
		for i := range virt.Parms {
			parm := &virt.Parms[i]
			errs = append(errs, checkTypeName(x, parm.TypeName)...)
		}
		if virt.Ret != nil {
			errs = append(errs, checkTypeName(x, virt.Ret)...)
		}
	}
	return errs
}

func checkTypeName(x *scope, name *TypeName) (errs []checkError) {
	defer x.tr("checkTypeName(%s)", name.ID())(&errs)
	// TODO: implement checkTypeName.
	return errs
}

func checkStmts(x *scope, want *Type, astStmts []ast.Stmt) (_ []Stmt, errs []checkError) {
	defer x.tr("gatherStmts(want=%s)", want)(&errs)
	var stmts []Stmt
	for i, astStmt := range astStmts {
		switch astStmt := astStmt.(type) {
		case *ast.Ret:
			ret, es := checkRet(x, astStmt)
			errs = append(errs, es...)
			stmts = append(stmts, ret)
		case *ast.Assign:
			var ss []Stmt
			var es []checkError
			x, ss, es = checkAssign(x, astStmt)
			errs = append(errs, es...)
			stmts = append(stmts, ss...)
		case ast.Expr:
			var expr Expr
			var es []checkError
			if i == len(astStmts)-1 {
				expr, es = checkExpr(x, want, astStmt)
			} else {
				expr, es = checkExpr(x, nil, astStmt)
			}
			errs = append(errs, es...)
			stmts = append(stmts, expr)
		default:
			panic(fmt.Sprintf("impossible type: %T", astStmt))
		}
	}
	return stmts, errs
}

func checkRet(x *scope, astRet *ast.Ret) (_ *Ret, errs []checkError) {
	defer x.tr("checkRet(…)")(&errs)

	var want *Type
	if fun := x.function(); fun == nil {
		err := x.err(astRet, "return outside of a function or method")
		errs = append(errs, *err)
	} else if fun.Sig.Ret != nil {
		want = fun.Sig.Ret.Type
	}
	expr, es := checkExpr(x, want, astRet.Val)
	return &Ret{ast: astRet, Val: expr}, append(errs, es...)
}

func checkAssign(x *scope, astAss *ast.Assign) (_ *scope, _ []Stmt, errs []checkError) {
	defer x.tr("checkAssign(…)")(&errs)

	x, vars, newLocal, errs := checkAssignVars(x, astAss)

	if len(vars) == 1 {
		var es []checkError
		assign := &Assign{ast: astAss, Var: vars[0]}
		assign.Expr, es = checkExpr(x, vars[0].typ, astAss.Expr)
		if newLocal[0] && vars[0].TypeName == nil {
			vars[0].typ = assign.Expr.Type()
		}
		errs = append(errs, es...)
		return x, []Stmt{assign}, errs
	}

	var stmts []Stmt
	astCall, ok := astAss.Expr.(*ast.Call)
	if !ok || len(astCall.Msgs) != len(vars) {
		got := 1
		if ok {
			got = len(astCall.Msgs)
		}
		err := x.err(astAss, "assignment count mismatch: got %d, want %d", got, len(vars))
		errs = append(errs, *err)
		expr, es := checkExpr(x, nil, astAss.Expr)
		errs = append(errs, es...)
		stmts = append(stmts, &Assign{
			ast:  astAss,
			Var:  vars[0],
			Expr: expr,
		})
		for i := 1; i < len(vars); i++ {
			stmts = append(stmts, &Assign{
				ast:  astAss,
				Var:  vars[i],
				Expr: nil,
			})
		}
		return x, stmts, errs
	}

	recv, es := checkExpr(x, nil, astCall.Recv)
	recvType := recv.Type()
	errs = append(errs, es...)
	loc := x.locals()
	tmp := &Var{
		Name:  x.newID(),
		typ:   recvType,
		Local: loc,
		Index: len(*loc),
	}
	*loc = append(*loc, tmp)
	x = x.new()
	x.variable = tmp
	stmts = append(stmts, &Assign{Var: tmp, Expr: recv})
	for i := range vars {
		msg, es := checkMsg(x, recvType, &astCall.Msgs[i])
		errs = append(errs, es...)
		call := &Call{
			ast:  astCall,
			Recv: &Ident{Text: tmp.Name, Var: tmp},
			Msgs: []Msg{msg},
		}
		if newLocal[i] && vars[i].TypeName == nil {
			vars[i].typ = call.Type()
		}
		stmts = append(stmts, &Assign{ast: astAss, Var: vars[i], Expr: call})
	}
	return x, stmts, errs
}

func checkAssignVars(x *scope, astAss *ast.Assign) (*scope, []*Var, []bool, []checkError) {
	var errs []checkError
	vars := make([]*Var, len(astAss.Vars))
	newLocal := make([]bool, len(astAss.Vars))
	for i := range astAss.Vars {
		astVar := &astAss.Vars[i]

		var typ *Type
		var typName *TypeName
		if astVar.Type != nil {
			var es []checkError
			typName, es = gatherTypeName(x, astVar.Type)
			typ = typName.Type
			errs = append(errs, es...)
		}

		switch found := x.findIdent(astVar.Name).(type) {
		case nil:
			x.log("adding local %s", astVar.Name)
			loc := x.locals()
			vr := &Var{
				ast:      astVar,
				Name:     astVar.Name,
				TypeName: typName,
				typ:      typ,
				Local:    loc,
				Index:    len(*loc),
			}
			*loc = append(*loc, vr)
			x = x.new()
			x.variable = vr
			vars[i] = vr
			newLocal[i] = true
		case *Var:
			if !found.isSelf() {
				vars[i] = found
				break
			}
			err := x.err(astVar, "cannot assign to self")
			errs = append(errs, *err)
			vars[i] = &Var{
				ast:      astVar,
				Name:     astVar.Name,
				TypeName: typName,
				typ:      typ,
			}
		case *Fun:
			err := x.err(astVar, "assignment to a function")
			note(err, "%s is defined at %s", found.Sig.Sel, x.loc(found))
			errs = append(errs, *err)
			vars[i] = &Var{
				ast:      astVar,
				Name:     astVar.Name,
				TypeName: typName,
				typ:      typ,
			}
		default:
			panic(fmt.Sprintf("impossible type: %T", found))
		}
	}
	return x, vars, newLocal, errs
}

func checkExprs(x *scope, astExprs []ast.Expr) ([]Expr, []checkError) {
	var errs []checkError
	exprs := make([]Expr, len(astExprs))
	for i, expr := range astExprs {
		var es []checkError
		exprs[i], es = checkExpr(x, nil, expr)
		errs = append(errs, es...)
	}
	return exprs, errs
}

func checkExpr(x *scope, infer *Type, astExpr ast.Expr) (Expr, []checkError) {
	switch astExpr := astExpr.(type) {
	case *ast.Call:
		return checkCall(x, astExpr)
	case *ast.Ctor:
		return checkCtor(x, astExpr)
	case *ast.Block:
		return checkBlock(x, infer, astExpr)
	case *ast.Ident:
		return checkIdent(x, astExpr)
	case *ast.Int:
		return checkInt(x, infer, astExpr, astExpr.Text)
	case *ast.Float:
		return checkFloat(x, infer, astExpr, astExpr.Text)
	case *ast.Rune:
		return checkRune(x, astExpr)
	case *ast.String:
		return checkString(x, astExpr)
	default:
		panic(fmt.Sprintf("impossible type: %T", astExpr))
	}
}

func checkCall(x *scope, astCall *ast.Call) (_ *Call, errs []checkError) {
	defer x.tr("checkCall(…)")(&errs)

	call := &Call{
		ast:  astCall,
		Msgs: make([]Msg, len(astCall.Msgs)),
	}

	var recv Expr
	var recvType *Type
	if astCall.Recv != nil {
		recv, errs = checkExpr(x, nil, astCall.Recv)
		recvType = recv.Type()
		if recvType == nil {
			x.log("call receiver check error")
			// There was a receiver, but we don't know it's type.
			// That error was reported elsewhere, but we can't continue here.
			// Do best-effort checking of the message arguments.
			for i := range astCall.Msgs {
				astMsg := &astCall.Msgs[i]
				call.Msgs[i] = Msg{
					ast: astMsg,
					Mod: identString(astMsg.Mod),
					Sel: astMsg.Sel,
				}
				var es []checkError
				call.Msgs[i].Args, es = checkExprs(x, astMsg.Args)
				errs = append(errs, es...)
			}
			return call, errs
		}
	}
	for i := range astCall.Msgs {
		var es []checkError
		call.Msgs[i], es = checkMsg(x, recvType, &astCall.Msgs[i])
		errs = append(errs, es...)
	}
	return call, errs
}

func checkMsg(x *scope, recv *Type, astMsg *ast.Msg) (_ Msg, errs []checkError) {
	defer x.tr("checkMsg(%s, %s)", recv, astMsg.Sel)(&errs)

	msg := Msg{
		ast: astMsg,
		Mod: identString(astMsg.Mod),
		Sel: astMsg.Sel,
	}
	errs = findMsgFun(x, recv, &msg)
	if msg.Fun == nil {
		// findMsgFun failed; best-effort check the arguments.
		var es []checkError
		msg.Args, es = checkExprs(x, astMsg.Args)
		return msg, append(errs, es...)
	}
	parms := msg.Fun.Sig.Parms
	if msg.Fun.Recv != nil {
		parms = parms[1:]
	}
	msg.Args = make([]Expr, len(astMsg.Args))
	for i, astArg := range astMsg.Args {
		var es []checkError
		typ := parms[i].TypeName.Type
		msg.Args[i], es = checkExpr(x, typ, astArg)
		errs = append(errs, es...)
	}
	return msg, errs
}

func findMsgFun(x *scope, recv *Type, msg *Msg) (errs []checkError) {
	x.tr("findMsgFun(%s, %s)", recv, msg.name())(&errs)
	var fun *Fun
	var mod string

	switch {
	case recv != nil && recv.Var != nil:
		c := recv.Var.TypeName
		if c != nil && c.Type != nil {
			fun = x.findFun(c.Type, msg.Sel)
		}
	case msg.Mod != "":
		mod = msg.Mod + " "
		imp := x.findImport(msg.Mod)
		if imp == nil {
			// msg.ast must be an *ast.Msg,
			// since the only other case is ast.Ident,
			// which is only for in-module function calls,
			// and this is not in-module.
			err := x.err(msg.ast.(*ast.Msg).Mod, "module %s not found", msg.Mod)
			return append(errs, *err)
		}
		fun = imp.findFun(recv, msg.Sel)
	default:
		fun = x.findFun(recv, msg.Sel)
	}
	if fun == nil {
		if recv == nil {
			err := x.err(msg, "function %s%s not found", mod, msg.Sel)
			return append(errs, *err)
		}
		err := x.err(msg, "method %s %s%s not found", recv.Sig.ID(), mod, msg.Sel)
		return append(errs, *err)
	}

	if recv != nil && recv.Var == nil && recv != fun.Recv.Type {
		// TODO: implement findMsgFun for lifted receiver types.
		errs = append(errs, *x.err(msg, "calls on parameterized receivers unimplemented"))
		return errs
	}

	msg.Fun = fun
	return errs
}

func checkCtor(x *scope, astCtor *ast.Ctor) (_ *Ctor, errs []checkError) {
	defer x.tr("checkCtor(%s)", astCtor.Type)(&errs)

	name, es := gatherTypeName(x, &astCtor.Type)
	errs = append(errs, es...)

	ctor := &Ctor{ast: astCtor, TypeName: *name, Sel: astCtor.Sel}

	switch ctor.typ = name.Type; {
	case ctor.typ == nil:
		// There was an error in the type name; do best-effort arg checking.
		args, es := checkExprs(x, astCtor.Args)
		errs = append(errs, es...)
		ctor.Args = args
	case ctor.typ.Alias != nil:
		// This should have already been resolved by gatherTypeName.
		panic("impossible alias")
	case handleRefConvert(x, ctor):
		// if handleRefConvert returns true,
		// ctor.Args is the converted expr, and
		// ctor.Ref is the reference differenc.
		break
	case isAry(x, ctor.typ):
		errs = append(errs, checkAryCtor(x, ctor)...)
	case ctor.typ.Cases != nil:
		errs = append(errs, checkOrCtor(x, ctor)...)
	case ctor.typ.Virts != nil:
		errs = append(errs, checkVirtCtor(x, ctor)...)
	case isBuiltIn(x, ctor.typ):
		err := x.err(astCtor, "cannot construct built-in type %s", ctor.TypeName)
		errs = append(errs, *err)
		args, es := checkExprs(x, astCtor.Args)
		errs = append(errs, es...)
		ctor.Args = args
	default:
		errs = append(errs, checkAndCtor(x, ctor)...)
	}
	return ctor, errs
}

func handleRefConvert(x *scope, ctor *Ctor) bool {
	if ctor.Sel != "" || len(ctor.ast.Args) != 1 || ctor.TypeName.Type == nil {
		return false
	}

	expr, errs := checkExpr(x, nil, ctor.ast.Args[0])
	if len(errs) > 0 {
		// Ignore the errors, they will be reported elsewhere
		// as we try non-reference conversions.
		return false
	}

	gotI, got := refBaseType(x, expr.Type())
	wantI, want := refBaseType(x, ctor.TypeName.Type)
	if want != got || gotI == wantI {
		return false
	}
	ctor.Args = []Expr{expr}
	ctor.Ref = wantI - gotI
	return true
}

func refBaseType(x *scope, typ *Type) (int, *Type) {
	var i int
	for isRef(x, typ) {
		i++
		typ = typ.Sig.Args[0].Type
	}
	return i, typ
}

func checkAryCtor(x *scope, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkAryCtor(%s)", ctor.TypeName)(&errs)
	want := ctor.TypeName.Type.Sig.Args[0].Type
	ctor.Args = make([]Expr, len(ctor.ast.Args))
	for i, expr := range ctor.ast.Args {
		var es []checkError
		ctor.Args[i], es = checkExpr(x, want, expr)
		errs = append(errs, es...)
	}
	return errs
}

func checkOrCtor(x *scope, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkOrCtor(%s)", ctor.TypeName)(&errs)

	if len(ctor.ast.Args) > 1 || ctor.Sel == "" {
		err := x.err(ctor, "malformed or-type constructor")
		errs = append(errs, *err)
		var es []checkError
		ctor.Args, es = checkExprs(x, ctor.ast.Args)
		return append(errs, es...)
	}

	ctor.Case = findCase(ctor.TypeName.Type, ctor.Sel)
	if ctor.Case == nil {
		err := x.err(ctor, "case %s not found", ctor.Sel)
		errs = append(errs, *err)
		expr, es := checkExpr(x, nil, ctor.ast.Args[0])
		ctor.Args = []Expr{expr}
		return append(errs, es...)
	}
	c := &ctor.TypeName.Type.Cases[*ctor.Case]

	if c.TypeName == nil {
		// Or-type constructors have a bit of a grammar ambiguity:
		// Is the argument a no-type case constructor or an identifier?
		// So, it ends up looking like both:
		// There is a single argument that is an identifier
		// with the name equal to the selector.
		// If the argument isn't such, then this was an array-style constructor
		// and thus there are too many arguments.
		//
		// If it is not just a single identifier, the parser sets Sel=="",
		// which is handled in the mal-formed error returned above.
		if id, ok := ctor.ast.Args[0].(*ast.Ident); !ok || id.Text != c.Name {
			panic("impossible")
		}
		return errs
	}

	expr, es := checkExpr(x, c.typ, ctor.ast.Args[0])
	ctor.Args = []Expr{expr}
	return append(errs, es...)
}

func findCase(typ *Type, name string) *int {
	for i := range typ.Cases {
		if typ.Cases[i].Name == name {
			return &i
		}
	}
	return nil
}

func checkVirtCtor(x *scope, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkVirtCtor(%s)", ctor.TypeName)(&errs)

	var es []checkError
	ctor.Args, es = checkExprs(x, ctor.ast.Args)
	errs = append(errs, es...)

	if len(ctor.ast.Args) != 1 {
		err := x.err(ctor, "malformed virtual-type constructor")
		return append(errs, *err)
	}

	recv := ctor.Args[0].Type()
	if recv == nil {
		return errs
	}

	var notes []string
	ctor.Funs, notes = findVirts(x, recv, ctor.typ.Virts)
	if len(notes) > 0 {
		err := x.err(ctor, "type %s does not implement %s", recv.Sig.ID(), ctor.typ.Sig.ID())
		err.notes = notes
		errs = append(errs, *err)
	}
	return errs
}

func findVirts(x *scope, recv *Type, virts []FunSig) ([]*Fun, []string) {
	var funs []*Fun
	var notes []string

	funs = make([]*Fun, len(virts))
	for i, want := range virts {
		got := x.findFun(recv, want.Sel)
		if got == nil {
			notes = append(notes, fmt.Sprintf("no method %s", want.Sel))
			continue
		}

		// Make a copy and remove the self parameter.
		gotSig := got.Sig
		gotSig.Parms = gotSig.Parms[1:]

		if !funSigEq(&gotSig, &want) {
			// Clear the parameter names for printing the error note.
			for i := range gotSig.Parms {
				gotSig.Parms[i].Name = ""
			}
			var where string
			if got.ast != nil {
				where = fmt.Sprintf(", defined at %s", x.loc(got.ast))
			}
			notes = append(notes,
				fmt.Sprintf("wrong type for method %s", want.Sel),
				fmt.Sprintf("	have %s%s", gotSig, where),
				fmt.Sprintf("	want %s", want))
			continue
		}
		funs[i] = got
	}
	return funs, notes
}

func funSigEq(a, b *FunSig) bool {
	if a.Sel != b.Sel || len(a.Parms) != len(b.Parms) || (a.Ret == nil) != (b.Ret == nil) {
		return false
	}
	for i := range a.Parms {
		aParm := &a.Parms[i]
		bParm := &b.Parms[i]
		if aParm.typ != bParm.typ {
			return false
		}
	}
	return a.Ret == nil || a.Ret.Type == b.Ret.Type
}

func checkAndCtor(x *scope, ctor *Ctor) (errs []checkError) {
	defer x.tr("checkAndCtor(%s)", ctor.TypeName)(&errs)

	if ctor.Sel == "" && len(ctor.ast.Args) > 0 {
		err := x.err(ctor, "malformed and-type constructor")
		errs = append(errs, *err)
		var es []checkError
		ctor.Args, es = checkExprs(x, ctor.ast.Args)
		return append(errs, es...)
	}

	var s strings.Builder
	for _, v := range ctor.typ.Fields {
		s.WriteString(v.Name)
		s.WriteRune(':')
	}
	if want := s.String(); ctor.Sel != want {
		err := x.err(ctor, "bad and-type constructor: got %s, expected %s", ctor.Sel, want)
		errs = append(errs, *err)
		var es []checkError
		ctor.Args, es = checkExprs(x, ctor.ast.Args)
		return append(errs, es...)
	}

	ctor.Args = make([]Expr, len(ctor.ast.Args))
	for i, astArg := range ctor.ast.Args {
		field := &ctor.typ.Fields[i]
		var es []checkError
		ctor.Args[i], es = checkExpr(x, field.typ, astArg)
		errs = append(errs, es...)
	}
	return errs
}

func checkBlock(x *scope, infer *Type, astBlock *ast.Block) (_ *Block, errs []checkError) {
	defer x.tr("checkBlock(infer=%s)", infer)(&errs)

	var resInfer *Type
	parmInfer := make([]*Type, len(astBlock.Parms))
	if isFun(x, infer) {
		x.log("is a fun")
		resInfer = infer.Sig.Args[len(infer.Sig.Args)-1].Type
		n := len(infer.Sig.Args)
		if n > len(astBlock.Parms) {
			n = len(astBlock.Parms)
		}
		for i := 0; i < n; i++ {
			parmInfer[i] = infer.Sig.Args[i].Type
		}
	} else {
		x.log("is not a fun")
	}

	blk := &Block{
		ast:   astBlock,
		Parms: make([]Var, len(astBlock.Parms)),
	}

	for i := range astBlock.Parms {
		astParm := &astBlock.Parms[i]
		parm := &blk.Parms[i]
		parm.ast = astParm
		parm.Name = astParm.Name
		if astParm.Type == nil {
			if parmInfer[i] == nil {
				err := x.err(parm, "cannot infer block parameter type")
				errs = append(errs, *err)
			}
			parm.typ = parmInfer[i]
			continue
		}
		var es []checkError
		parm.TypeName, es = gatherTypeName(x, astParm.Type)
		parm.typ = parm.TypeName.Type
		errs = append(errs, es...)
	}

	x = x.new()
	x.block = blk
	for i := range blk.Parms {
		parm := &blk.Parms[i]
		parm.BlkParm = blk
		parm.Index = i
		x = x.new()
		x.variable = parm
	}

	var es []checkError
	blk.Stmts, es = checkStmts(x, resInfer, astBlock.Stmts)
	errs = append(errs, es...)

	if len(blk.Parms) >= MaxValueParms {
		err := x.err(astBlock, "too many block parameters: got %, max %d",
			len(astBlock.Parms), MaxValueParms)
		errs = append(errs, *err)
		return blk, errs
	}

	typeArgs := make([]TypeName, len(blk.Parms)+1)
	for i := range blk.Parms {
		parm := &blk.Parms[i]
		if parm.typ == nil {
			return blk, errs
		}
		if parm.TypeName != nil {
			typeArgs[i] = *parm.TypeName
			continue
		}
		typeArgs[i] = TypeName{
			ast:  &astBlock.Parms[i],
			Mod:  parm.typ.Sig.Mod,
			Name: parm.typ.Sig.Name,
			Args: parm.typ.Sig.Args,
			Type: parm.typ,
		}
	}

	resType := builtInType(x, "Nil")
	if n := len(blk.Stmts); n > 0 {
		if expr, ok := blk.Stmts[n-1].(Expr); ok {
			resType = expr.Type()
		}
	}
	if resType == nil {
		return blk, errs
	}
	typeArgs[len(typeArgs)-1] = TypeName{
		ast:  astBlock,
		Mod:  resType.Sig.Mod,
		Name: resType.Sig.Name,
		Args: resType.Sig.Args,
		Type: resType,
	}
	blk.typ = builtInType(x, "Fun", typeArgs...)
	return blk, errs
}

func checkIdent(x *scope, astIdent *ast.Ident) (_ Expr, errs []checkError) {
	defer x.tr("checkIdent(%s)", astIdent.Text)(&errs)

	ident := &Ident{ast: astIdent, Text: astIdent.Text}
	switch vr := x.findIdent(astIdent.Text).(type) {
	case nil:
		err := x.err(astIdent, "identifier %s not found", astIdent.Text)
		errs = append(errs, *err)
	case *Var:
		ident.Var = vr
	case *Fun:
		defer x.tr("checkMsg(%s, %s)", nil, astIdent.Text)(&errs)
		msg := Msg{ast: astIdent, Sel: astIdent.Text}
		es := findMsgFun(x, nil, &msg)
		errs = append(errs, es...)
		return &Call{ast: astIdent, Msgs: []Msg{msg}}, errs
	default:
		panic(fmt.Sprintf("impossible type: %T", vr))
	}
	return ident, errs
}

func checkInt(x *scope, infer *Type, AST ast.Expr, text string) (_ Expr, errs []checkError) {
	defer x.tr("checkInt(infer=%s, %s)", infer, text)(&errs)

	if isFloat(x, infer) {
		return checkFloat(x, infer, AST, text)
	}
	var i big.Int
	x.log("parsing int [%s]", text)
	if _, ok := i.SetString(text, 0); !ok {
		panic("malformed int")
	}
	typ := builtInType(x, "Int")
	if isInt(x, infer) {
		typ = infer
	}
	if err := checkIntBounds(x, AST, typ, &i); err != nil {
		errs = append(errs, *err)
	}
	return &Int{ast: AST, Val: &i, typ: typ}, errs
}

func checkIntBounds(x *scope, n interface{}, t *Type, i *big.Int) *checkError {
	signed, bits := disectIntType(x, t)
	x.log("signed=%v, bits=%v", signed, bits)
	if !signed && i.Cmp(&big.Int{}) < 0 {
		return x.err(n, "type %s cannot represent %s: negative unsigned", t, i)
	}
	min := big.NewInt(-(1 << uint(bits)))
	x.log("val=%v, val.BitLen()=%d, min=%v", i, i.BitLen(), min)
	if i.BitLen() > bits && (!signed || i.Cmp(min) != 0) {
		return x.err(n, "type %s cannot represent %s: overflow", t, i)
	}
	return nil
}

func disectIntType(x *scope, typ *Type) (bool, int) {
	switch typ {
	case builtInType(x, "Int8"):
		return true, 7
	case builtInType(x, "Int16"):
		return true, 15
	case builtInType(x, "Int32"):
		return true, 31
	case builtInType(x, "Int64"):
		return true, 63
	case builtInType(x, "UInt8"):
		return false, 8
	case builtInType(x, "UInt16"):
		return false, 16
	case builtInType(x, "UInt32"):
		return false, 32
	case builtInType(x, "UInt64"):
		return false, 64
	default:
		panic(fmt.Sprintf("impossible int type: %T", typ))
	}
}

func checkFloat(x *scope, infer *Type, AST ast.Expr, text string) (_ Expr, errs []checkError) {
	defer x.tr("checkFloat(infer=%s, %s)", infer, text)(&errs)

	var f big.Float
	if _, _, err := f.Parse(text, 10); err != nil {
		panic("malformed float")
	}
	if isInt(x, infer) {
		var i big.Int
		if _, acc := f.Int(&i); acc != big.Exact {
			err := x.err(AST, "type %s cannot represent %s: truncation", infer.Sig.ID(), text)
			errs = append(errs, *err)
		}
		expr, es := checkInt(x, infer, AST, i.String())
		return expr, append(errs, es...)
	}
	typ := builtInType(x, "Float")
	if isFloat(x, infer) {
		typ = infer
	}
	return &Float{ast: AST, Val: &f, typ: typ}, errs
}

func isInt(x *scope, typ *Type) bool {
	switch {
	case typ == nil:
		return false
	default:
		return false
	case typ == builtInType(x, "Int8") ||
		typ == builtInType(x, "Int16") ||
		typ == builtInType(x, "Int32") ||
		typ == builtInType(x, "Int64") ||
		typ == builtInType(x, "UInt8") ||
		typ == builtInType(x, "UInt16") ||
		typ == builtInType(x, "UInt32") ||
		typ == builtInType(x, "UInt64"):
		return true
	}
}

func isFloat(x *scope, typ *Type) bool {
	switch {
	case typ == nil:
		return false
	default:
		return false
	case typ == builtInType(x, "Float32") ||
		typ == builtInType(x, "Float64"):
		return true
	}
}

func checkRune(x *scope, astRune *ast.Rune) (*Int, []checkError) {
	defer x.tr("checkRune(%s)", astRune.Text)()
	return &Int{
		ast: astRune,
		Val: big.NewInt(int64(astRune.Rune)),
		typ: builtInType(x, "Int32"),
	}, nil
}

func checkString(x *scope, astString *ast.String) (*String, []checkError) {
	defer x.tr("checkString(%s)", astString.Text)()
	return &String{
		ast:  astString,
		Data: astString.Data,
		typ:  builtInType(x, "String"),
	}, nil
}

func builtInType(x *scope, name string, args ...TypeName) *Type {
	// Silence tracing for looking up built-in types.
	savedTrace := x.cfg.Trace
	x.cfg.Trace = false
	defer func() { x.cfg.Trace = savedTrace }()

	for x.univ == nil {
		x = x.up
	}
	typ := findType(len(args), name, x.univ)
	if typ == nil {
		panic(fmt.Sprintf("built-in type (%d)%s not found", len(args), name))
	}
	typ, errs := instType(x, typ, args)
	if len(errs) > 0 {
		panic(fmt.Sprintf("failed to inst built-in type: %v", errs))
	}
	return typ
}

func isAry(x *scope, typ *Type) bool {
	return isBuiltIn(x, typ) && typ.Sig.Name == "Array"
}

func isRef(x *scope, typ *Type) bool {
	return isBuiltIn(x, typ) && typ.Sig.Name == "&"
}

func isFun(x *scope, typ *Type) bool {
	return isBuiltIn(x, typ) && typ.Sig.Name == "Fun"
}

func isBuiltIn(x *scope, typ *Type) bool {
	return typ != nil && typ.Sig.Mod == "" && x.defFiles[typ] == nil
}

func identString(id *ast.Ident) string {
	if id == nil {
		return ""
	}
	return id.Text
}
