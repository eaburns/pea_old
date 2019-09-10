package types

import (
	"fmt"
	"math/big"

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
	def.Type, errs = gatherTypeName(x, def.ast.Type)
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
		vars[i].Type, es = gatherTypeName(x, astVars[i].Type)
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
		vr.Type, es = gatherTypeName(x, astVars[i].Type)
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

	name.Type, es = instType(x, typ, name)
	errs = append(errs, es...)
	return name, errs
}

func instType(x *scope, typ *Type, name *TypeName) (res *Type, errs []checkError) {
	defer func() { x.log("inst=%p", res) }()
	defer x.tr("instType(%s, %v)", typ.name(), name)(&errs)

	// We access typ.Alias and typ.Sig.Parms.
	// Both of these must be cycle free to guarantee
	// that they are populated by this call.
	// TODO: check typ.Sig.Parms cycle.
	if es := gatherDef(x, typ); es != nil {
		return nil, append(errs, es...)
	}

	args := name.Args
	if typ.Alias != nil {
		if typ.Alias.Type == nil {
			return nil, errs // error reported elsewhere
		}
		sub := make(map[*Var]TypeName)
		for i := range typ.Sig.Parms {
			sub[&typ.Sig.Parms[i]] = name.Args[i]
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

func gatherStmts(x *scope, astStmts []ast.Stmt) (_ []Stmt, errs []checkError) {
	var stmts []Stmt
	for _, astStmt := range astStmts {
		ss, es := gatherStmt(x, astStmt)
		errs = append(errs, es...)
		stmts = append(stmts, ss...)
	}
	return stmts, errs
}

func gatherStmt(x *scope, astStmt ast.Stmt) (_ []Stmt, errs []checkError) {
	switch astStmt := astStmt.(type) {
	case *ast.Ret:
		var ret *Ret
		ret, errs = gatherRet(x, astStmt)
		return []Stmt{ret}, errs
	case *ast.Assign:
		return gatherAssign(x, astStmt)
	case ast.Expr:
		var expr Expr
		expr, errs = gatherExpr(x, astStmt)
		return []Stmt{expr}, errs
	default:
		panic(fmt.Sprintf("impossible type: %T", astStmt))
	}
}

func gatherRet(x *scope, astRet *ast.Ret) (_ *Ret, errs []checkError) {
	defer x.tr("gatherRet(…)")(&errs)
	var expr Expr
	expr, errs = gatherExpr(x, astRet.Val)
	return &Ret{ast: astRet, Val: expr}, errs
}

func gatherAssign(x *scope, astAss *ast.Assign) (_ []Stmt, errs []checkError) {
	defer x.tr("gatherAssign(…)")(&errs)
	vars, es := gatherVars(x, astAss.Vars)
	errs = append(errs, es...)
	val, es := gatherExpr(x, astAss.Val)
	errs = append(errs, es...)

	if len(vars) == 1 {
		return []Stmt{&Assign{ast: astAss, Var: vars[0], Val: val}}, errs
	}

	var stmts []Stmt
	call, ok := val.(*Call)
	if !ok || len(call.Msgs) != len(vars) {
		got := 1
		if ok {
			got = len(call.Msgs)
		}
		err := x.err(astAss, "assignment count mismatch: got %d, want %d", got, len(vars))
		errs = append(errs, *err)
		stmts = append(stmts, &Assign{ast: astAss, Var: vars[0], Val: val})
		for _, v := range vars[1:] {
			stmts = append(stmts, &Assign{ast: astAss, Var: v, Val: nil})
		}
		return stmts, errs
	}

	tmp := x.newID()
	stmts = append(stmts, &Assign{
		Var: Var{Name: tmp},
		Val: call.Recv,
	})
	for i := range vars {
		stmts = append(stmts, &Assign{
			ast: astAss,
			Var: vars[i],
			Val: &Call{
				ast:  call.ast,
				Recv: &Ident{Text: tmp},
				Msgs: []Msg{call.Msgs[i]},
			},
		})
	}
	return stmts, errs
}

func gatherExprs(x *scope, astExprs []ast.Expr) ([]Expr, []checkError) {
	var errs []checkError
	exprs := make([]Expr, len(astExprs))
	for i, expr := range astExprs {
		var es []checkError
		exprs[i], es = gatherExpr(x, expr)
		errs = append(errs, es...)
	}
	return exprs, errs
}

func gatherExpr(x *scope, astExpr ast.Expr) (Expr, []checkError) {
	switch astExpr := astExpr.(type) {
	case *ast.Call:
		return gatherCall(x, astExpr)
	case *ast.Ctor:
		return gatherCtor(x, astExpr)
	case *ast.Block:
		return gatherBlock(x, astExpr)
	case *ast.Ident:
		return gatherIdent(x, astExpr)
	case *ast.Int:
		return gatherInt(x, astExpr)
	case *ast.Float:
		return gatherFloat(x, astExpr)
	case *ast.Rune:
		return gatherRune(x, astExpr)
	case *ast.String:
		return gatherString(x, astExpr)
	default:
		panic(fmt.Sprintf("impossible type: %T", astExpr))
	}
}

func gatherCall(x *scope, astCall *ast.Call) (_ *Call, errs []checkError) {
	defer x.tr("gatherCall(…)")(&errs)
	var recv Expr
	if astCall.Recv != nil {
		recv, errs = gatherExpr(x, astCall.Recv)
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
	blk.Stmts, es = gatherStmts(x, astBlock.Stmts)
	errs = append(errs, es...)
	return blk, errs
}

func gatherIdent(x *scope, astIdent *ast.Ident) (*Ident, []checkError) {
	defer x.tr("gatherIdent(%s)", astIdent.Text)()
	return &Ident{ast: astIdent, Text: astIdent.Text}, nil
}

func gatherInt(x *scope, astInt *ast.Int) (*Int, []checkError) {
	defer x.tr("gatherInt(%s)", astInt.Text)()
	var z big.Int
	if _, ok := z.SetString(astInt.Text, 0); !ok {
		panic("malformed int")
	}
	return &Int{ast: astInt, Val: &z}, nil
}

func gatherFloat(x *scope, astFloat *ast.Float) (*Float, []checkError) {
	defer x.tr("gatherFloat(%s)", astFloat.Text)()
	var z big.Float
	if _, _, err := z.Parse(astFloat.Text, 10); err != nil {
		panic("malformed float")
	}
	return &Float{ast: astFloat, Val: &z}, nil
}

func gatherRune(x *scope, astRune *ast.Rune) (*Int, []checkError) {
	defer x.tr("gatherRune(%s)", astRune.Text)()
	return &Int{ast: astRune, Val: big.NewInt(int64(astRune.Rune))}, nil
}

func gatherString(x *scope, astString *ast.String) (*String, []checkError) {
	defer x.tr("gatherString(%s)", astString.Text)()
	return &String{ast: astString, Data: astString.Data}, nil
}

func identString(id *ast.Ident) string {
	if id == nil {
		return ""
	}
	return id.Text
}
