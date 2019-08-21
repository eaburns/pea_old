package types

import (
	"fmt"

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
	x := newState(cfg, astMod)
	mod, errs := check(x, astMod)
	if len(errs) > 0 {
		return nil, convertErrors(errs)
	}
	return mod, nil
}

type file struct {
	ast     *ast.File
	imports []Import
	defs    []Def
}

func check(x *state, astMod *ast.Mod) (*Mod, []checkError) {
	mod := &Mod{AST: astMod}

	var files []file
	var errs []checkError
	seen := make(map[string]Def)
	for i := range astMod.Files {
		file := file{ast: &astMod.Files[i]}
		errs = append(errs, imports(x, &file)...)
		errs = append(errs, gather(x, seen, &file)...)
		mod.Defs = append(mod.Defs, file.defs...)
		files = append(files, file)
	}
	return mod, errs
}

func imports(x *state, file *file) []checkError {
	var errs []checkError
	for _, astImp := range file.ast.Imports {
		path := astImp.Path[1 : len(astImp.Path)-1] // trim "
		imp, err := x.cfg.Importer.Import(x.cfg, path)
		if err != nil {
			errs = append(errs, *x.err(astImp, err.Error()))
			continue
		}
		file.imports = append(file.imports, *imp)
	}
	return errs
}

func gather(x *state, seen map[string]Def, file *file) []checkError {
	var errs []checkError
	for _, astDef := range file.ast.Defs {
		def := makeDef(astDef)
		id := def.ID()
		if prev, ok := seen[id]; ok {
			err := x.err(astDef, "%s is redefined", id)
			note(err, "`previously defined at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[id] = def
		}
		file.defs = append(file.defs, def)
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
