package types

import (
	"fmt"
	"path"

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
	defer x.tr("check(%s)", astMod.Name)(errs)

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
	errs = append(errs, checkDefSigs(x, mod.Defs)...)

	return mod, errs
}

// checkDups returns redefinition errors for types, vals, and funs.
// It doesn't check duplicate methods.
func checkDups(x *scope, defs []Def) (errs []checkError) {
	defer x.tr("checkDups")(errs)

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
			if def.Recv != nil {
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
		var recv *TypeSig
		if astDef.Recv != nil {
			recv = &TypeSig{
				ast:   astDef.Recv,
				Arity: len(astDef.Recv.Parms),
				Name:  astDef.Recv.Name,
			}
		}
		return &Fun{
			ast:  astDef,
			priv: astDef.Priv(),
			Recv: recv,
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

func checkDefSigs(x *scope, defs []Def) (errs []checkError) {
	defer x.tr("checkDefSigs()")(errs)

	for _, def := range defs {
		errs = append(errs, checkDefSig(x, def)...)
	}
	return errs
}

func checkDefSig(x *scope, def Def) (errs []checkError) {
	file, ok := x.defFiles[def]
	if !ok {
		// It's not in this module. It must be already checked.
		return nil
	}
	x = file.x

	if typ, ok := def.(*Type); ok && typ.ast.Alias != nil {
		if err := aliasCycle(x, typ); err != nil {
			errs = append(errs, *err)
			return errs
		}
		x.aliasStack = append(x.aliasStack, typ)
		defer func() { x.aliasStack = x.aliasStack[:len(x.aliasStack)-1] }()
	}

	if x.checked[def] {
		return nil
	}
	x.checked[def] = true

	switch def := def.(type) {
	case *Val:
		return checkVal(x, def)
	case *Fun:
		return checkFun(x, def)
	case *Type:
		return checkType(x, def)
	default:
		panic(fmt.Sprintf("impossible type %T", def))
	}
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

func checkVal(x *scope, def *Val) (errs []checkError) {
	defer x.tr("checkVal(%s)", def.name())(errs)
	// TODO: implement checkVal.
	return errs
}

func checkFun(x *scope, def *Fun) (errs []checkError) {
	defer x.tr("checkFun(%s)", def.name())(errs)
	// TODO: implement checkFun.
	return errs
}

func checkType(x *scope, def *Type) (errs []checkError) {
	switch {
	case def.ast.Alias != nil:
		return checkAliasType(x, def)
	}

	// TODO: implement checkType.
	defer x.tr("checkType(%s)", def.name())(errs)

	return errs
}

func checkAliasType(x *scope, typ *Type) (errs []checkError) {
	defer x.tr("checkAliasType(%s)", typ.name())(errs)
	typ.Alias, errs = checkTypeName(x, typ.ast.Alias)
	return errs
}

func checkTypeName(x *scope, astName *ast.TypeName) (_ *TypeName, errs []checkError) {
	defer x.tr("checkTypeName(%s)", astName)(errs)

	n := &TypeName{
		ast:  astName,
		Name: astName.Name,
		Mod:  identString(astName.Mod),
	}
	for i := range astName.Args {
		arg, es := checkTypeName(x, &astName.Args[i])
		errs = append(errs, es...)
		n.Args = append(n.Args, *arg)
	}

	var imp *imp
	var typ *Type
	if n.Mod == "" {
		typ = x.findType(len(n.Args), n.Name)
	} else {
		imp = x.findImport(n.Mod)
		if imp == nil {
			err := x.err(astName.Mod, "module %s not found", n.Mod)
			errs = append(errs, *err)
		} else {
			typ = imp.findType(len(n.Args), n.Name)
		}
	}
	if typ == nil {
		var err *checkError
		if len(n.Args) == 0 {
			err = x.err(astName, "type %s not found", n.Name)
		} else {
			err = x.err(astName, "type (%d)%s not found", len(n.Args), n.Name)
		}
		// TODO: note candidate types of different arity if type is not found.
		errs = append(errs, *err)
		return n, errs
	}
	var es []checkError
	typ, es = instType(x, typ, n)
	errs = append(errs, es...)

	if typ != nil && typ.Alias != nil {
		typ = typ.Alias.Type
	}
	n.Type = typ
	return n, errs
}

func identString(id *ast.Ident) string {
	if id == nil {
		return ""
	}
	return id.Text
}
