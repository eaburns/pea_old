// Copyright © 2020 The Pea Authors under an MIT-style license.

// Package mod loads module source and object file lists
// along with dependency modules.
package mod

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eaburns/pea/ast"
)

// A Mod contains information about the source for a single module.
type Mod struct {
	// ModPath is the module path as it would appear in an import statement.
	ModPath string
	// ModName is the base file name of ModPath.
	ModName string
	// SrcPath is the source file path.
	// This is path to the source file or directory of the module.
	SrcPath string
	// SrcDir may differ from SrcPath for the root module
	// if the root module is given as a .peago file, not a directory.
	SrcDir string
	// SrcFiles contains the source file paths in alphabetical order.
	SrcFiles []string

	// GoSrcFiles are Go source files for the module.
	GoSrcFiles []string

	// Deps are the module dependencies
	// in alphabetical order on ModPath.
	//
	// Deps is nil until after a call to LoadDeps.
	Deps []*Mod
}

// Load returns a *Mod for the module modPath, loaded from srcPath.
// srcPath may be either a .pea source file or a directory of .pea source files.
func Load(srcPath, modPath string) (*Mod, error) {
	m, err := newMod(srcPath, modPath)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func newMod(srcPath, modPath string) (*Mod, error) {
	srcPath, err := realPath(srcPath)
	if err != nil {
		return nil, err
	}
	m := &Mod{
		ModPath: modPath,
		ModName: filepath.Base(modPath),
		SrcPath: srcPath,
	}
	if err := findSrcFiles(m); err != nil {
		return nil, err
	}
	return m, nil
}

func realPath(dir string) (string, error) {
	switch dir {
	case string([]rune{filepath.Separator}):
		return dir, nil
	case ".":
		return os.Getwd()
	default:
		base := filepath.Base(dir)
		dir, err := realPath(filepath.Dir(dir))
		if err != nil {
			return "", err
		}
		switch base {
		case ".":
			return dir, nil
		case "..":
			return filepath.Dir(dir), nil
		default:
			return filepath.Join(dir, base), nil
		}
	}
}

func findSrcFiles(m *Mod) error {
	srcFile, err := os.Open(m.SrcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	stat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		m.SrcFiles = []string{m.SrcPath}
		m.SrcDir = filepath.Dir(m.SrcPath)
		return nil
	}
	finfos, err := srcFile.Readdir(-1)
	if err != nil {
		return err
	}
	for _, finfo := range finfos {
		path := filepath.Join(m.SrcPath, finfo.Name())
		if strings.HasSuffix(finfo.Name(), ".pea") {
			m.SrcFiles = append(m.SrcFiles, path)
		}
		if strings.HasSuffix(finfo.Name(), ".go") {
			m.GoSrcFiles = append(m.GoSrcFiles, path)
		}
	}
	sort.Strings(m.SrcFiles)
	sort.Strings(m.GoSrcFiles)
	m.SrcDir = m.SrcPath
	return nil
}

// LoadDeps loads the modules's dependencies, setting the Deps field.
// Dependencies are loaded transitively, so all modules in Deps
// also have their Deps loaded.
func (m *Mod) LoadDeps(root string) error {
	return loadDeps(root, m)
}

func loadDeps(modRootDir string, root *Mod) error {
	seen := make(map[string]*Mod)
	seen[root.ModPath] = root
	var addDeps func(*Mod) error
	addDeps = func(m *Mod) error {
		depFiles, err := deps(m.SrcFiles)
		if err != nil {
			return err
		}
		for _, depFile := range depFiles {
			if d, ok := seen[depFile]; ok {
				m.Deps = append(m.Deps, d)
				continue
			}
			srcPath := filepath.Join(modRootDir, depFile)
			d, err := newMod(srcPath, depFile)
			if err != nil {
				return err
			}
			m.Deps = append(m.Deps, d)
			seen[depFile] = d
			if err := addDeps(d); err != nil {
				return err
			}
		}
		return nil
	}
	return addDeps(root)
}

func deps(srcFiles []string) ([]string, error) {
	var deps []string
	for _, file := range srcFiles {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		ds, err := ast.ReadImports(file, bufio.NewReader(f))
		if err != nil {
			return nil, err
		}
		f.Close()
		deps = append(deps, ds...)
	}

	sort.Strings(deps)

	var i int
	for _, d := range deps {
		if i == 0 || d != deps[i-1] {
			deps[i] = d
			i++
		}
	}
	return deps[:i], nil
}

// TopologicalDeps returns root and its dependencies
// in topologically sorted order, with dependencies
// before their dependants.
func TopologicalDeps(roots []*Mod) []*Mod {
	var sorted []*Mod
	seen := make(map[*Mod]bool)
	var add func(*Mod)
	add = func(m *Mod) {
		if seen[m] {
			return
		}
		seen[m] = true
		for _, d := range m.Deps {
			add(d)
		}
		sorted = append(sorted, m)
	}
	for _, root := range roots {
		add(root)
	}
	return sorted
}
