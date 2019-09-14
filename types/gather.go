package types

import (
	"fmt"

	"github.com/eaburns/pea/ast"
)

func gatherDefs(x *scope, defs []Def) (errs []checkError) {
	for _, def := range defs {
		errs = append(errs, gatherDef(x, def)...)
	}
	return errs
}

func gatherDef(x *scope, def Def) (errs []checkError) {
	file, ok := x.defFiles[def]
	if !ok {
		defer x.tr("gatherDef(%s) from other module", def.name())(&errs)
		return nil
	}
	x = file.x
	if def.AST() == nil {
		panic("impossible")
	}

	// Gathering defs is recrursive for Types, which can be self-referential.
	// For all recurrences, we only want a pointer to the target definition,
	// so it is OK if the definition is not yet fully gathered.
	// This can happen if a type definition is cyclic
	// and we are still in the process of gathering some of its fields.
	// We break the recursion below by checking x.gathered[def].
	// However, for alias types, we look at the Type.Alias field;
	// alias definitions must no be cyclic.
	// We break the recursion and emit an error for cycle aliases here.
	// We also look at type parameter constraints, which are types,
	// and must also be acyclic.
	if typ, ok := def.(*Type); ok && typ.ast.Alias != nil {
		if err := aliasCycle(x, typ); err != nil {
			return append(errs, *err)
		}
		x.aliasStack = append(x.aliasStack, typ)
		defer func() { x.aliasStack = x.aliasStack[:len(x.aliasStack)-1] }()
	}
	if x.gathered[def] {
		return nil
	}
	x.gathered[def] = true

	switch def := def.(type) {
	case *Val:
		errs = append(errs, gatherVal(x, def)...)
	case *Fun:
		errs = append(errs, gatherFun(x, def)...)
	case *Type:
		errs = append(errs, gatherType(x, def)...)
	default:
		panic(fmt.Sprintf("impossible type: %T", def))
	}
	return errs
}

func aliasCycle(x *scope, typ *Type) *checkError {
	for i, t := range x.aliasStack {
		if typ != t {
			continue
		}
		err := x.err(t, "type alias cycle")
		for ; i < len(x.aliasStack); i++ {
			alias := x.aliasStack[i]
			// alias loops can only occur in the current package,
			// so alias.AST() is guaranteed to be non-nil,
			// and x.loc(alias) is OK.
			note(err, "%s at %s", alias.ast, x.loc(alias))
		}
		note(err, "%s at %s", typ.ast, x.loc(typ))
		return err
	}
	return nil
}

func gatherVal(x *scope, def *Val) (errs []checkError) {
	defer x.tr("gatherVal(%s)", def.name())(&errs)
	if def.ast.Var.Type != nil {
		def.Var.TypeName, errs = gatherTypeName(x, def.ast.Var.Type)
		def.Var.typ = def.Var.TypeName.Type
	}
	return errs
}

func gatherFun(x *scope, def *Fun) (errs []checkError) {
	defer x.tr("gatherFun(%s)", def.name())(&errs)

	x, def.Recv, errs = gatherRecv(x, def.ast.Recv)

	var es []checkError
	x, def.TParms, es = gatherTypeParms(x, def.ast.TParms)
	errs = append(errs, es...)

	sig, es := gatherFunSig(x, &def.ast.Sig)
	errs = append(errs, es...)
	def.Sig = *sig

	for i := range def.Sig.Parms {
		def.Sig.Parms[i].Parm = def
		def.Sig.Parms[i].Index = i
	}

	return errs
}

func gatherRecv(x *scope, astRecv *ast.Recv) (_ *scope, _ *Recv, errs []checkError) {
	if astRecv == nil {
		return x, nil, nil
	}
	defer x.tr("gatherRecv(%s)", astRecv)(&errs)

	recv := &Recv{
		ast:   astRecv,
		Arity: len(astRecv.Parms),
		Name:  astRecv.Name,
		Mod:   identString(astRecv.Mod),
	}
	var es []checkError
	x, recv.Parms, es = gatherTypeParms(x, astRecv.Parms)
	errs = append(errs, es...)

	var typ *Type
	if recv.Mod == "" {
		switch t := x.findType(recv.Arity, recv.Name).(type) {
		case nil:
			break
		case *Type:
			typ = t
		case *Var:
			panic("impossible")
		}
	} else {
		imp := x.findImport(recv.Mod)
		if imp == nil {
			err := x.err(astRecv.Mod, "module %s not found", recv.Mod)
			errs = append(errs, *err)
			return x, recv, errs
		}
		typ = imp.findType(recv.Arity, recv.Name)
	}
	if typ == nil {
		var err *checkError
		err = x.err(astRecv, "type %s not found", recv.ID())
		// TODO: note candidate types of different arity if a type is not found.
		errs = append(errs, *err)
		return x, recv, errs
	}

	// We access typ.Alias; it must be cycle free to guarantee
	// that they are populated by this call.
	if es := gatherDef(x, typ); es != nil {
		return x, recv, append(errs, es...)
	}
	if typ.Alias != nil {
		typ = typ.Alias.Type
	}
	recv.Type = typ
	return x, recv, errs
}

func gatherTypeParms(x *scope, astVars []ast.Var) (_ *scope, _ []Var, errs []checkError) {
	if astVars == nil {
		return x, nil, nil
	}

	defer x.tr("gatherTypeParms(…)")(&errs)
	vars := make([]Var, len(astVars))
	for i := range astVars {
		vars[i] = Var{ast: &astVars[i], Name: astVars[i].Name}
		x = x.new()
		x.typeVar = &vars[i]

		var es []checkError
		if astVars[i].Type != nil {
			vars[i].TypeName, es = gatherTypeName(x, astVars[i].Type)
			vars[i].typ = vars[i].TypeName.Type
		}
		errs = append(errs, es...)
	}
	return x, vars, errs
}

func gatherFunSigs(x *scope, astSigs []ast.FunSig) (_ []FunSig, errs []checkError) {
	var sigs []FunSig
	for i := range astSigs {
		sig, es := gatherFunSig(x, &astSigs[i])
		errs = append(errs, es...)
		sigs = append(sigs, *sig)
	}
	return sigs, errs
}

func gatherFunSig(x *scope, astSig *ast.FunSig) (_ *FunSig, errs []checkError) {
	defer x.tr("gatherFunSig(%s)", astSig)(&errs)

	sig := &FunSig{
		ast: astSig,
		Sel: astSig.Sel,
	}
	var es []checkError
	sig.Parms, es = gatherVars(x, astSig.Parms)
	errs = append(errs, es...)

	sig.Ret, es = gatherTypeName(x, astSig.Ret)
	errs = append(errs, es...)

	return sig, errs
}

func gatherType(x *scope, def *Type) (errs []checkError) {
	defer x.tr("gatherType(%s [%p])", def.name(), def)(&errs)

	var es []checkError
	x, def.Sig.Parms, es = gatherTypeParms(x, def.ast.Sig.Parms)
	errs = append(errs, es...)

	switch {
	case def.ast.Alias != nil:
		def.Alias, es = gatherTypeName(x, def.ast.Alias)
		errs = append(errs, es...)
		if def.Sig.Parms != nil {
			// TODO: error on unused type parameters.
			// The following comment is only true if the type params
			// are all referenced by the alias target type.

			// If Parms is non-nil, def.Alias.Type
			// must be a new type instance,
			// because it was created
			// with freshly gathered type arguments
			// from this type name.
			def.Alias.Type.Sig.Parms = def.Sig.Parms
		}
	case def.ast.Fields != nil:
		def.Fields, es = gatherVars(x, def.ast.Fields)
		errs = append(errs, es...)
		for i := range def.Fields {
			def.Fields[i].Field = def
			def.Fields[i].Index = i
		}
	case def.ast.Cases != nil:
		def.Cases, es = gatherVars(x, def.ast.Cases)
		errs = append(errs, es...)
	case def.ast.Virts != nil:
		def.Virts, es = gatherFunSigs(x, def.ast.Virts)
		errs = append(errs, es...)
	}
	return errs
}

func gatherVars(x *scope, astVars []ast.Var) (_ []Var, errs []checkError) {
	defer x.tr("gatherVars(…)")(&errs)
	var vars []Var
	for i := range astVars {
		var es []checkError
		vr := Var{ast: &astVars[i], Name: astVars[i].Name}
		if astVars[i].Type != nil {
			vr.TypeName, es = gatherTypeName(x, astVars[i].Type)
			vr.typ = vr.TypeName.Type
		}
		errs = append(errs, es...)
		vars = append(vars, vr)
	}
	return vars, errs
}

func gatherTypeNames(x *scope, astNames []ast.TypeName) ([]TypeName, []checkError) {
	var errs []checkError
	var names []TypeName
	for i := range astNames {
		arg, es := gatherTypeName(x, &astNames[i])
		errs = append(errs, es...)
		names = append(names, *arg)
	}
	return names, errs
}

func gatherTypeName(x *scope, astName *ast.TypeName) (_ *TypeName, errs []checkError) {
	if astName == nil {
		return nil, nil
	}
	defer x.tr("gatherTypeName(%s)", astName)(&errs)

	name := &TypeName{
		ast:  astName,
		Name: astName.Name,
		Mod:  identString(astName.Mod),
	}
	var es []checkError
	name.Args, es = gatherTypeNames(x, astName.Args)
	errs = append(errs, es...)

	var typ *Type
	if name.Mod == "" {
		switch t := x.findType(len(name.Args), name.Name).(type) {
		case nil:
			break
		case *Type:
			typ = t
		case *Var:
			name.Var = t
			return name, errs
		}
	} else {
		imp := x.findImport(name.Mod)
		if imp == nil {
			err := x.err(astName.Mod, "module %s not found", name.Mod)
			errs = append(errs, *err)
			return name, errs
		}
		typ = imp.findType(len(name.Args), name.Name)
	}
	if typ == nil {
		var err *checkError
		err = x.err(astName, "type %s not found", name.ID())
		// TODO: note candidate types of different arity if a type is not found.
		errs = append(errs, *err)
		return name, errs
	}

	name.Type, es = instType(x, typ, name.Args)
	errs = append(errs, es...)
	return name, errs
}

func instType(x *scope, typ *Type, args []TypeName) (res *Type, errs []checkError) {
	defer func() { x.log("inst=%p", res) }()
	defer x.tr("instType(%s, %v)", typ.name(), args)(&errs)

	// We access typ.Alias and typ.Sig.Parms.
	// Both of these must be cycle free to guarantee
	// that they are populated by this call.
	// TODO: check typ.Sig.Parms cycle.
	if es := gatherDef(x, typ); es != nil {
		return nil, append(errs, es...)
	}

	if typ.Alias != nil {
		if typ.Alias.Type == nil {
			return nil, errs // error reported elsewhere
		}
		sub := make(map[*Var]TypeName)
		for i := range typ.Sig.Parms {
			sub[&typ.Sig.Parms[i]] = args[i]
		}
		args = subTypeNames(x, make(map[*Type]bool), sub, typ.Alias.Args)
		typ = typ.Alias.Type
	}
	if len(args) == 0 {
		x.log("nothing to instantiate")
		return typ, nil
	}

	key := makeTypeKey(typ.Sig.Name, args)
	if inst, ok := x.typeInsts[key]; ok {
		return inst, nil
	}

	inst := *typ
	x.typeInsts[key] = &inst
	x.insts = append(x.insts, &inst)

	if file, ok := x.defFiles[typ]; ok {
		x.defFiles[&inst] = file
		// The type was defined within this module.
		// It may not be fully gathered; we need to gather our new instance.
		//
		// Further, this call to gatherDef must make a complete *Type.
		// The only way an incomplete *Type would be made
		// is if we are currently gathering &inst previously on the call stack
		// and gatherDef returns true because x.gathered[&inst]=true.
		// However, if this were the case, x.typeInsts[key] above
		// would have had an entry, and we would have never gotten here.
		//
		// Lastly, call gatherDef, not gatherType, because gatherDef
		// fixes the scope to file-scope and does alias cycle checking.
		es := gatherDef(x, &inst)
		errs = append(errs, es...)
	}

	sub := make(map[*Var]TypeName)
	for i := range inst.Sig.Parms {
		sub[&inst.Sig.Parms[i]] = args[i]
	}
	subTypeBody(x, make(map[*Type]bool), sub, &inst)
	inst.Sig.Parms = nil
	inst.Sig.Args = args
	return &inst, errs
}

type typeKey struct {
	name string
	args interface{}
}

type argsKey struct {
	typ  typeKey
	next interface{}
}

func makeTypeKey(name string, args []TypeName) typeKey {
	return typeKey{name: name, args: makeArgsKey(args)}
}

func makeArgsKey(args []TypeName) interface{} {
	if len(args) == 0 {
		return nil
	}
	var tkey typeKey
	switch a := args[0]; {
	case a.Type == nil && a.Var == nil:
		// This case indicates an error somwhere in the args.
		// The error was reported elsewhere; just use the empty key.
		break
	case a.Type == nil:
		tkey = makeTypeKey(a.Var.Name, nil)
	default:
		tkey = makeTypeKey(a.Type.Sig.Name, args[0].Args)
	}
	return argsKey{typ: tkey, next: makeArgsKey(args[1:])}
}

func gatherStmts(x *scope, want *Type, astStmts []ast.Stmt) (_ []Stmt, errs []checkError) {
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
			x, ss, es = gatherAssign(x, astStmt)
			errs = append(errs, es...)
			stmts = append(stmts, ss...)
		case ast.Expr:
			var expr Expr
			var es []checkError
			if i == len(astStmts)-1 {
				expr, es = gatherExpr(x, want, astStmt)
			} else {
				expr, es = gatherExpr(x, nil, astStmt)
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
	expr, es := gatherExpr(x, want, astRet.Val)
	return &Ret{ast: astRet, Val: expr}, append(errs, es...)
}

func gatherAssign(x *scope, astAss *ast.Assign) (_ *scope, _ []Stmt, errs []checkError) {
	defer x.tr("gatherAssign(…)")(&errs)

	vars := make([]*Var, len(astAss.Vars))
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
		case *Var:
			vars[i] = found
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

	if len(vars) == 1 {
		var es []checkError
		assign := &Assign{ast: astAss, Var: vars[0]}
		assign.Expr, es = gatherExpr(x, vars[0].typ, astAss.Expr)
		errs = append(errs, es...)
		return x, []Stmt{assign}, errs
	}

	// TODO: actually check the assignment left-hand-side expression.

	var stmts []Stmt
	astCall, ok := astAss.Expr.(*ast.Call)
	if !ok || len(astCall.Msgs) != len(vars) {
		got := 1
		if ok {
			got = len(astCall.Msgs)
		}
		err := x.err(astAss, "assignment count mismatch: got %d, want %d", got, len(vars))
		errs = append(errs, *err)
		expr, es := gatherExpr(x, nil, astAss.Expr)
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

	recv, es := gatherExpr(x, nil, astCall.Recv)
	var recvType *Type // TODO: recv.Type()
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
		stmts = append(stmts, &Assign{
			ast: astAss,
			Var: vars[i],
			Expr: &Call{
				ast:  astCall,
				Recv: &Ident{Text: tmp.Name, Var: tmp},
				Msgs: []Msg{msg},
			},
		})
	}
	return x, stmts, errs
}

func checkMsg(x *scope, typ *Type, astMsg *ast.Msg) (_ Msg, errs []checkError) {
	x.tr("checkMsg(%s, %s)", typ, astMsg.Sel)(&errs)

	return Msg{
		ast: astMsg,
		Mod: identString(astMsg.Mod),
		Sel: astMsg.Sel,
		// TODO: check Msg.Args
		Args: nil,
		// TODO: lookup Msg's Fun
	}, nil
}

func gatherExprs(x *scope, astExprs []ast.Expr) ([]Expr, []checkError) {
	var errs []checkError
	exprs := make([]Expr, len(astExprs))
	for i, expr := range astExprs {
		var es []checkError
		exprs[i], es = gatherExpr(x, nil /* TODO */, expr)
		errs = append(errs, es...)
	}
	return exprs, errs
}

func gatherExpr(x *scope, infer *Type, astExpr ast.Expr) (Expr, []checkError) {
	switch astExpr := astExpr.(type) {
	case *ast.Call:
		return gatherCall(x, astExpr)
	case *ast.Ctor:
		return gatherCtor(x, astExpr)
	case *ast.Block:
		return gatherBlock(x, astExpr)
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

func gatherCall(x *scope, astCall *ast.Call) (_ *Call, errs []checkError) {
	defer x.tr("gatherCall(…)")(&errs)
	var recv Expr
	if astCall.Recv != nil {
		recv, errs = gatherExpr(x, nil /* TODO */, astCall.Recv)
	}
	msgs, es := gatherMsgs(x, astCall.Msgs)
	errs = append(errs, es...)
	return &Call{ast: astCall, Recv: recv, Msgs: msgs}, errs
}

func gatherMsgs(x *scope, astMsgs []ast.Msg) ([]Msg, []checkError) {
	var errs []checkError
	msgs := make([]Msg, len(astMsgs))
	for i := range astMsgs {
		var es []checkError
		msgs[i], es = gatherMsg(x, &astMsgs[i])
		errs = append(errs, es...)
	}
	return msgs, errs
}

func gatherMsg(x *scope, astMsg *ast.Msg) (_ Msg, errs []checkError) {
	defer x.tr("gatherMsg(%s)", astMsg.Sel)(&errs)
	msg := Msg{
		ast: astMsg,
		Mod: identString(astMsg.Mod),
		Sel: astMsg.Sel,
	}
	msg.Args, errs = gatherExprs(x, astMsg.Args)
	return msg, errs
}

func gatherCtor(x *scope, astCtor *ast.Ctor) (_ *Ctor, errs []checkError) {
	defer x.tr("gatherCtor(%s)", astCtor.Type)(&errs)
	typ, es := gatherTypeName(x, &astCtor.Type)
	errs = append(errs, es...)
	args, es := gatherExprs(x, astCtor.Args)
	errs = append(errs, es...)
	return &Ctor{ast: astCtor, Type: *typ, Sel: astCtor.Sel, Args: args}, nil
}

func gatherBlock(x *scope, astBlock *ast.Block) (_ *Block, errs []checkError) {
	defer x.tr("gatherBlock(…)")(&errs)
	blk := &Block{ast: astBlock}
	blk.Parms, errs = gatherVars(x, astBlock.Parms)
	var es []checkError
	blk.Stmts, es = gatherStmts(x, nil, astBlock.Stmts)
	errs = append(errs, es...)
	return blk, errs
}
