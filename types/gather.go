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
	if def.ast() == nil {
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
	if typ, ok := def.(*Type); ok {
		if astType, ok := typ.AST.(*ast.Type); ok && astType.Alias != nil {
			if err := aliasCycle(x, typ); err != nil {
				return append(errs, *err)
			}
			x.aliasStack = append(x.aliasStack, typ)
			defer func() { x.aliasStack = x.aliasStack[:len(x.aliasStack)-1] }()
		}
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
			note(err, "%s at %s", alias.AST, x.loc(alias))
		}
		note(err, "%s at %s", typ.AST, x.loc(typ))
		return err
	}
	return nil
}

func gatherVal(x *scope, def *Val) (errs []checkError) {
	defer x.tr("gatherVal(%s)", def.name())(&errs)
	if def.AST.Var.Type != nil {
		def.Var.TypeName, errs = gatherTypeName(x, def.AST.Var.Type)
		def.Var.typ = def.Var.TypeName.Type
	}
	return errs
}

func gatherFun(x *scope, def *Fun) (errs []checkError) {
	defer x.tr("gatherFun(%s)", def.name())(&errs)

	x, def.Recv, errs = gatherRecv(x, def.AST.(*ast.Fun).Recv)

	var es []checkError
	x, def.TParms, es = gatherTypeParms(x, def.AST.(*ast.Fun).TParms)
	errs = append(errs, es...)

	sig, es := gatherFunSig(x, &def.AST.(*ast.Fun).Sig)
	errs = append(errs, es...)
	def.Sig = *sig

	if def.Recv != nil {
		self := Var{
			Name: "self",
			TypeName: &TypeName{
				AST:  def.Recv.AST,
				Mod:  def.Recv.Mod,
				Name: def.Recv.Name,
				Type: def.Recv.Type,
			},
			typ: def.Recv.Type,
		}
		if def.Recv.Type != nil {
			self.TypeName.Args = def.Recv.Type.Sig.Args
		}
		def.Sig.Parms = append([]Var{self}, def.Sig.Parms...)
	}

	for i := range def.Sig.Parms {
		def.Sig.Parms[i].FunParm = def
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
		AST:   astRecv,
		Arity: len(astRecv.Parms),
		Name:  astRecv.Name,
		Mod:   identString(astRecv.Mod),
	}
	var es []checkError
	x, recv.Parms, es = gatherTypeParms(x, astRecv.Parms)
	errs = append(errs, es...)

	var typ *Type
	if recv.Mod == "" {
		typ = x.findType(recv.Arity, recv.Name)
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

func gatherTypeParms(x *scope, astVars []ast.Var) (_ *scope, _ []TypeVar, errs []checkError) {
	if astVars == nil {
		return x, nil, nil
	}

	defer x.tr("gatherTypeParms(%v)", astVars)(&errs)
	vars := make([]TypeVar, len(astVars))
	for i := range astVars {
		astVar := &astVars[i]
		typ := &Type{
			AST: astVar,
			Sig: TypeSig{Name: astVar.Name},
			Var: &vars[i],
		}
		vars[i] = TypeVar{
			AST:  astVar,
			Name: astVar.Name,
			Type: typ,
		}
		x = x.new()
		x.typeVar = vars[i].Type

		if astVars[i].Type != nil {
			n, es := gatherTypeName(x, astVar.Type)
			errs = append(errs, es...)
			if n != nil {
				vars[i].Ifaces = append(vars[i].Ifaces, *n)
			}
		}
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
		AST: astSig,
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
	defer x.tr("gatherType(%p %s)", def, def.ast)(&errs)

	astType := def.AST.(*ast.Type)

	var es []checkError
	x, def.Sig.Parms, es = gatherTypeParms(x, astType.Sig.Parms)
	errs = append(errs, es...)

	x.typeInsts[makeTypeSigKey(&def.Sig)] = def

	switch {
	case astType.Alias != nil:
		def.Alias, es = gatherTypeName(x, astType.Alias)
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
	case astType.Fields != nil:
		def.Fields, es = gatherVars(x, astType.Fields)
		errs = append(errs, es...)
		for i := range def.Fields {
			def.Fields[i].Field = def
			def.Fields[i].Index = i
		}
	case astType.Cases != nil:
		def.Cases, es = gatherVars(x, astType.Cases)
		errs = append(errs, es...)
	case astType.Virts != nil:
		def.Virts, es = gatherFunSigs(x, astType.Virts)
		errs = append(errs, es...)
	}
	return errs
}

func gatherVars(x *scope, astVars []ast.Var) (_ []Var, errs []checkError) {
	defer x.tr("gatherVars(…)")(&errs)
	var vars []Var
	for i := range astVars {
		var es []checkError
		vr := Var{AST: &astVars[i], Name: astVars[i].Name}
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
		AST:  astName,
		Name: astName.Name,
		Mod:  identString(astName.Mod),
	}
	var es []checkError
	name.Args, es = gatherTypeNames(x, astName.Args)
	errs = append(errs, es...)

	var typ *Type
	if name.Mod == "" {
		typ = x.findType(len(name.Args), name.Name)
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
	defer x.tr("instType(%p %s, %v)", typ, typ, args)(&errs)

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
		sub := make(map[*TypeVar]TypeName)
		for i := range typ.Sig.Parms {
			sub[&typ.Sig.Parms[i]] = args[i]
		}
		seen := make(map[*Type]*Type)
		args = subTypeNames(x, seen, sub, typ.Alias.Args)
		typ = typ.Alias.Type
	}
	if len(args) == 0 {
		return typ, nil
	}

	key := makeTypeKey(typ.Sig.Mod, typ.Sig.Name, args)
	if inst, ok := x.typeInsts[key]; ok {
		return inst, nil
	}

	var inst Type
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
		es := gatherDef(x, typ)
		errs = append(errs, es...)
	}

	inst = *typ
	x.typeInsts[key] = &inst
	x.insts = append(x.insts, &inst)

	sub := make(map[*TypeVar]TypeName)
	for i := range inst.Sig.Parms {
		sub[&inst.Sig.Parms[i]] = args[i]
	}
	seen := make(map[*Type]*Type)
	seen[typ] = &inst
	subTypeBody(x, seen, sub, &inst)
	inst.Sig.Parms = nil
	inst.Sig.Args = args
	return &inst, errs
}

type typeKey struct {
	// Either mod+name+args or Var.

	mod  string
	name string
	args interface{}

	Var *TypeVar
}

type argsKey struct {
	typ  typeKey
	next interface{}
}

func makeTypeSigKey(sig *TypeSig) typeKey {
	k := typeKey{mod: sig.Mod, name: sig.Name}
	for i := len(sig.Parms) - 1; i >= 0; i-- {
		k.args = argsKey{
			typ:  typeKey{Var: &sig.Parms[i]},
			next: k.args,
		}
	}
	return k
}

func makeTypeKey(mod, name string, args []TypeName) typeKey {
	return typeKey{mod: mod, name: name, args: makeArgsKey(args)}
}

func makeArgsKey(args []TypeName) interface{} {
	if len(args) == 0 {
		return nil
	}
	var tkey typeKey
	switch a := args[0]; {
	case a.Type == nil:
		// This case indicates an error somwhere in the args.
		// The error was reported elsewhere; just use the empty key.
		break
	case a.Type.Var != nil:
		tkey = typeKey{Var: a.Type.Var}
	default:
		sig := &a.Type.Sig
		tkey = makeTypeKey(sig.Mod, sig.Name, a.Args)
	}
	return argsKey{typ: tkey, next: makeArgsKey(args[1:])}
}
