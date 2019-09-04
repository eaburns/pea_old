package types

import (
	"fmt"
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
	// However, for alias types, we _do_ look at the Type.Alias field;
	// alias definitions must no be cyclic.
	// We break the recursion and emit an error for cycle aliases here.
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

	if def.ast.Recv != nil {
		def.Recv = &Recv{
			ast:   def.ast.Recv,
			Arity: len(def.ast.Recv.Parms),
			Name:  def.ast.Recv.Name,
		}
		var es []checkError
		x, def.Recv.Parms, es = gatherTypeParms(x, def.ast.Recv.Parms)
		errs = append(errs, es...)
	}

	var es []checkError
	x, def.TParms, es = gatherTypeParms(x, def.ast.TParms)
	errs = append(errs, es...)

	sig, es := gatherFunSig(x, &def.ast.Sig)
	errs = append(errs, es...)
	def.Sig = *sig

	return errs
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

func gatherRecv(x *scope, astRecv *ast.TypeSig) (_ *Recv, errs []checkError) {
	if astRecv == nil {
		return nil, nil
	}

	defer x.tr("gatherRecv(%s)", astRecv)(&errs)
	recv := &Recv{
		ast:   astRecv,
		Arity: len(astRecv.Parms),
		Name:  astRecv.Name,
	}
	recv.Parms, errs = gatherVars(x, astRecv.Parms)
	return recv, errs
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

	var imp *imp
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
		imp = x.findImport(name.Mod)
		if imp == nil {
			err := x.err(astName.Mod, "module %s not found", name.Mod)
			errs = append(errs, *err)
		} else {
			typ = imp.findType(len(name.Args), name.Name)
		}
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
	return argsKey{
		typ:  makeTypeKey(args[0].Name, args[0].Args),
		next: makeArgsKey(args[1:]),
	}
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
