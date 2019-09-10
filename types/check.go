package types

import (
	"fmt"
	"math/big"
	"path"
	"sort"
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
	x := newUnivScope(newState(cfg, astMod))
	mod, errs := check(x, astMod)
	if len(errs) > 0 {
		return nil, convertErrors(errs)
	}
	return mod, nil
}

type file struct {
	ast     *ast.File
	imports []imp
	x       *scope
}

type imp struct {
	path string
	name string
	defs []Def
}

func check(x *scope, astMod *ast.Mod) (_ *Mod, errs []checkError) {
	defer x.tr("check(%s)", astMod.Name)(&errs)

	mod := &Mod{AST: astMod}
	x = x.new()
	x.mod = mod

	var files []*file
	for i := range astMod.Files {
		file := &file{ast: &astMod.Files[i]}
		file.x = x.new()
		file.x.file = file
		errs = append(errs, imports(x.state, file)...)
		for _, astDef := range file.ast.Defs {
			def := makeDef(astDef)
			mod.Defs = append(mod.Defs, def)
			x.defFiles[def] = file
		}
		files = append(files, file)
	}

	errs = append(errs, checkDups(x, mod.Defs)...)
	errs = append(errs, gatherDefs(x, mod.Defs)...)
	errs = append(errs, checkDupMeths(x, mod.Defs)...)
	errs = append(errs, checkDefs(x, mod.Defs)...)

	return mod, errs
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
			id = def.Name
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
			if def.ast.Recv != nil {
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

func makeDef(astDef ast.Def) Def {
	switch astDef := astDef.(type) {
	case *ast.Val:
		return &Val{
			ast:  astDef,
			priv: astDef.Priv(),
			Name: astDef.Ident,
		}
	case *ast.Fun:
		return &Fun{
			ast:  astDef,
			priv: astDef.Priv(),
			Sig: FunSig{
				ast: &astDef.Sig,
				Sel: astDef.Sig.Sel,
			},
		}
	case *ast.Type:
		return &Type{
			ast:  astDef,
			priv: astDef.Priv(),
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

func identString(id *ast.Ident) string {
	if id == nil {
		return ""
	}
	return id.Text
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

func subTypeNames(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, names0 []TypeName) []TypeName {
	var names1 []TypeName
	for i := range names0 {
		n := subTypeName(x, seen, sub, &names0[i])
		names1 = append(names1, *n)
	}
	return names1
}

func subTypeName(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, name0 *TypeName) *TypeName {
	if name0 == nil {
		return nil
	}
	defer x.tr("subTypeName(%s, %s [var=%p])", subDebugString(sub), name0.ID(), name0.Var)()

	if s, ok := sub[name0.Var]; ok {
		x.log("%s→%s", name0.Var.Name, s)
		return &s
	}
	name1 := *name0
	name1.Args = subTypeNames(x, seen, sub, name1.Args)
	name1.Var = subVar(x, seen, sub, name1.Var)
	name1.Type = subType(x, seen, sub, name1.Type)
	return &name1
}

func subVars(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, vars0 []Var) []Var {
	var vars1 []Var
	for i := range vars0 {
		vars1 = append(vars1, *subVar(x, seen, sub, &vars0[i]))
	}
	return vars1
}

func subVar(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, var0 *Var) *Var {
	if var0 == nil {
		return nil
	}
	defer x.tr("subVar(%s, %s)", subDebugString(sub), var0.Name)()

	var1 := *var0
	var1.Type = subTypeName(x, seen, sub, var1.Type)
	return &var1
}

func subType(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, typ0 *Type) *Type {
	if typ0 == nil {
		return nil
	}
	if seen[typ0] {
		return typ0
	}
	seen[typ0] = true

	defer x.tr("subType(%s, %s)", subDebugString(sub), typ0.name())()

	typ1 := *typ0
	subTypeBody(x, seen, sub, &typ1)
	return &typ1
}

func subTypeBody(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, typ *Type) {
	typ.Sig.Parms = subVars(x, seen, sub, typ.Sig.Parms)
	typ.Sig.Args = subTypeNames(x, seen, sub, typ.Sig.Args)
	typ.Alias = subTypeName(x, seen, sub, typ.Alias)
	typ.Fields = subVars(x, seen, sub, typ.Fields)
	typ.Cases = subVars(x, seen, sub, typ.Cases)
	typ.Virts = subFunSigs(x, seen, sub, typ.Virts)
}

func subFunSigs(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, sigs0 []FunSig) []FunSig {
	var sigs1 []FunSig
	for i := range sigs0 {
		sigs1 = append(sigs1, *subFunSig(x, seen, sub, &sigs0[i]))
	}
	return sigs1
}

func subFunSig(x *scope, seen map[*Type]bool, sub map[*Var]TypeName, sig0 *FunSig) *FunSig {
	sig1 := *sig0
	sig1.Parms = subVars(x, seen, sub, sig1.Parms)
	sig1.Ret = subTypeName(x, seen, sub, sig1.Ret)
	return &sig1
}

func subDebugString(sub map[*Var]TypeName) string {
	var ss []string
	for k, v := range sub {
		s := fmt.Sprintf("%s[%p]=%s", k.Name, k, v)
		ss = append(ss, s)
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })
	return strings.Join(ss, ";")
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
		panic("impossible")
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
	if def.Type != nil {
		errs = append(errs, checkTypeName(x, def.Type)...)
	}
	var es []checkError
	if def.Init, es = gatherStmts(x, def.ast.Init); len(es) > 0 {
		return append(errs, es...)
	}
	errs = append(errs, checkStmts(x, def.Init)...)
	return errs
}

func checkFun(x *scope, def *Fun) (errs []checkError) {
	defer x.tr("checkFun(%s)", def.name())(&errs)
	if def.Recv != nil {
		for i := range def.Recv.Parms {
			x = x.new()
			x.typeVar = &def.Recv.Parms[i]
		}
	}
	for i := range def.TParms {
		x = x.new()
		x.typeVar = &def.TParms[i]
	}
	def.Stmts, errs = gatherStmts(x, def.ast.Stmts)
	return append(errs, checkStmts(x, def.Stmts)...)
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
		errs = append(errs, checkTypeName(x, field.Type)...)
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
		if cas.Type != nil {
			errs = append(errs, checkTypeName(x, cas.Type)...)
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
			errs = append(errs, checkTypeName(x, parm.Type)...)
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

func checkStmts(x *scope, stmts []Stmt) []checkError {
	// TODO: implement checkStmts.
	return nil
}

func gatherStmts(x *scope, astStmts []ast.Stmt) (_ []Stmt, errs []checkError) {
	var stmts []Stmt
	for _, astStmt := range astStmts {
		stmt, es := gatherStmt(x, astStmt)
		errs = append(errs, es...)
		stmts = append(stmts, stmt)
	}
	return stmts, errs
}

func gatherStmt(x *scope, astStmt ast.Stmt) (_ Stmt, errs []checkError) {
	switch astStmt := astStmt.(type) {
	case *ast.Ret:
		defer x.tr("gatherStmt(Ret)")(&errs)
		val, es := gatherExpr(x, astStmt.Val)
		errs = append(errs, es...)
		return &Ret{ast: astStmt, Val: val}, errs
	case *ast.Assign:
		defer x.tr("gatherStmt(Assign)")(&errs)
		vars, es := gatherVars(x, astStmt.Vars)
		errs = append(errs, es...)
		val, es := gatherExpr(x, astStmt.Val)
		errs = append(errs, es...)
		return &Assign{ast: astStmt, Vars: vars, Val: val}, errs
	case ast.Expr:
		return gatherExpr(x, astStmt)
	default:
		panic(fmt.Sprintf("impossible type: %T", astStmt))
	}
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
