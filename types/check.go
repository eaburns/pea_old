package types

import (
	"fmt"

	"github.com/eaburns/pea/ast"
)

// Check type-checks an AST and returns the type-checked tree or errors.
func Check(astMod *ast.Mod, opts ...Opt) (*Mod, []error) {
	x := newState(astMod, opts...)
	mod := &Mod{AST: astMod}
	mod.Imports = append(mod.Imports, newUniv(x))

	var errs []checkError
	var files [][]Def
	seen := make(map[string]Def)
	for _, astFile := range astMod.Files {
		ds, es := gather(x, seen, astFile.Defs)
		errs = append(errs, es...)
		files = append(files, ds)
		mod.Defs = append(mod.Defs, ds...)
	}

	if len(errs) > 0 {
		return nil, convertErrors(errs)
	}
	return mod, nil
}

// gather creates a type node for each AST definition
// containing its AST node, whether it's private,
// and minimum information to compute its ID()s,
// For each new ID, if it is in the seen map, a redefinition error is returned,
// otherwise the new definition is added to the map.
func gather(x *state, seen map[string]Def, astDefs []ast.Def) (defs []Def, errs []checkError) {
	for _, astDef := range astDefs {
		var def Def
		switch astDef := astDef.(type) {
		case *ast.Val:
			def = &Val{
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
			def = &Fun{
				ast:  astDef,
				priv: astDef.Priv(),
				Recv: recv,
				Sig: FunSig{
					ast: &astDef.Sig,
					Sel: astDef.Sig.Sel,
				},
			}
		case *ast.Type:
			def = &Type{
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

		id := def.ID()
		if prev, ok := seen[id]; ok {
			err := x.err(astDef, "%s is redefined", id)
			note(err, "`previously defined at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[id] = def
		}
		defs = append(defs, def)
	}
	return defs, errs
}
