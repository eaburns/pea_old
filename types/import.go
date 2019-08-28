package types

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eaburns/pea/ast"
)

// An Importer imports modules by path.
type Importer interface {
	Import(cfg Config, path string) (*Import, error)
}

type importer struct {
	paths    []string
	imports  []*Import
	importer Importer
}

func newImporter(x *state, astMod string, base Importer) *importer {
	return &importer{
		paths:    []string{astMod},
		imports:  []*Import{newUniv(x)},
		importer: base,
	}
}

func (ir *importer) Import(cfg Config, path string) (*Import, error) {
	ir.paths = append(ir.paths, path)
	defer func() { ir.paths = ir.paths[:len(ir.paths)-1] }()
	for _, p := range ir.paths[:len(ir.paths)-1] {
		if p == path {
			return nil, fmt.Errorf("import cycle: %v", ir.paths)
		}
	}
	for _, imp := range ir.imports {
		if imp.Path == path {
			return imp, nil
		}
	}
	imp, err := ir.importer.Import(cfg, path)
	if err != nil {
		return nil, err
	}
	ir.imports = append(ir.imports, imp)
	return imp, nil
}

type dirImporter struct{}

func (ir *dirImporter) Import(cfg Config, path string) (*Import, error) {
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
	return &Import{Path: path, Defs: mod.Defs}, nil
}
