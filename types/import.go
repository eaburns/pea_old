package types

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eaburns/pea/ast"
)

// An Importer imports modules by path.
type Importer interface {
	Import(cfg Config, path string) ([]Def, error)
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

func (ir *importer) Import(cfg Config, path string) ([]Def, error) {
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
	defs, err := ir.importer.Import(cfg, path)
	ir.imports[path] = defs // add nil on error too
	if err != nil {
		return nil, err
	}
	return defs, nil
}

type dirImporter struct{}

func (ir *dirImporter) Import(cfg Config, path string) ([]Def, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %s", path, err)
	}
	finfos, err := f.Readdir(0) // all
	f.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %s", path, err)
	}
	p := ast.NewParser(path)
	for _, fi := range finfos {
		err := p.ParseFile(filepath.Join(path, fi.Name()))
		if err != nil {
			return nil, fmt.Errorf("error parsing import %s:\n%v", path, err)
		}
	}
	cfg.Trace = false // don't trace imports
	mod, errs := Check(p.Mod(), cfg)
	if len(errs) > 0 {
		return nil, fmt.Errorf("error checking import %s:\n%v", path, errs)
	}
	setMod(path, mod.Defs)
	return mod.Defs, nil
}

func setMod(path string, defs []Def) {
	for _, def := range defs {
		switch def := def.(type) {
		case *Val:
			def.Mod = path
		case *Fun:
			def.Mod = path
		case *Type:
			def.Sig.Mod = path
		default:
			panic(fmt.Sprintf("impossible type: %T", def))
		}
	}
}
