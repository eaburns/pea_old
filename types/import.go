package types

import (
	"fmt"
	"path/filepath"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/loc"
	"github.com/eaburns/pea/mod"
)

// An Importer imports modules by path.
type Importer interface {
	Import(cfg Config, locs *loc.Files, path string) ([]Def, error)
}

type importer struct {
	paths    []string
	imports  map[string][]Def
	importer Importer
}

func newImporter(x *state, astMod string, base Importer) *importer {
	imports := make(map[string][]Def)
	imports[""] = newUniv(x)
	return &importer{
		paths:    []string{astMod},
		imports:  imports,
		importer: base,
	}
}

func (ir *importer) Import(cfg Config, files *loc.Files, path string) ([]Def, error) {
	ir.paths = append(ir.paths, path)
	defer func() { ir.paths = ir.paths[:len(ir.paths)-1] }()
	for _, p := range ir.paths[:len(ir.paths)-1] {
		if p == path {
			return nil, fmt.Errorf("import cycle: %v", ir.paths)
		}
	}
	if defs, ok := ir.imports[path]; ok {
		return defs, nil
	}
	defs, err := ir.importer.Import(cfg, files, path)
	ir.imports[path] = defs // add nil on error too
	if err != nil {
		return nil, err
	}
	return defs, nil
}

// SourceImporter imports modules from their source code.
type SourceImporter struct {
	// Root is the root directory prepended to module paths.
	Root string
}

// Import implemements the Importer interface.
func (ir *SourceImporter) Import(cfg Config, locs *loc.Files, modPath string) ([]Def, error) {
	path := filepath.Join(ir.Root, modPath)
	mod, err := mod.Load(path, modPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %s", path, err)
	}
	p := ast.NewParserWithLocs(modPath, locs)
	for _, f := range mod.SrcFiles {
		if err := p.ParseFile(f); err != nil {
			return nil, fmt.Errorf("error parsing import %s:\n%v", path, err)
		}
	}
	cfg.Trace = false // don't trace imports
	checkedMod, errs := Check(p.Mod(), cfg)
	if len(errs) > 0 {
		return nil, fmt.Errorf("error checking import %s:\n%v", path, errs)
	}
	setMod(modPath, checkedMod.Defs)
	return checkedMod.Defs, nil
}

func setMod(path string, defs []Def) {
	for _, def := range defs {
		switch def := def.(type) {
		case *Val:
			def.ModPath = path
		case *Fun:
			def.ModPath = path
		case *Type:
			def.ModPath = path
		default:
			panic(fmt.Sprintf("impossible type: %T", def))
		}
	}
}
