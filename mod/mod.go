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
	srcFiles, srcDir, err := srcFiles(srcPath)
	if err != nil {
		return nil, err
	}
	m := &Mod{
		ModPath:  modPath,
		ModName:  filepath.Base(modPath),
		SrcPath:  srcPath,
		SrcDir:   srcDir,
		SrcFiles: srcFiles,
	}
	return m, err
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

func srcFiles(srcPath string) ([]string, string, error) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return nil, "", err
	}
	defer srcFile.Close()
	stat, err := srcFile.Stat()
	if err != nil {
		return nil, "", err
	}
	if !stat.IsDir() {
		return []string{srcPath}, filepath.Dir(srcPath), nil
	}
	finfos, err := srcFile.Readdir(-1)
	if err != nil {
		return nil, "", err
	}
	var paths []string
	for _, finfo := range finfos {
		if !strings.HasSuffix(finfo.Name(), ".pea") {
			continue
		}
		path := filepath.Join(srcPath, finfo.Name())
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, srcPath, nil
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
		ds, err := ast.ReadImports(bufio.NewReader(f))
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
func TopologicalDeps(root *Mod) []*Mod {
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
	add(root)
	return sorted
}
