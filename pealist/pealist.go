// The pealist command lists all pea modules in the given directory
// in topological order, dependencies first.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eaburns/pea/mod"
)

func main() {
	if len(os.Args) < 2 {
		die(errors.New("Usage: pealist <mod root>"))
	}
	var mods []*mod.Mod
	seen := make(map[string]*mod.Mod)
	listed := make(map[string]bool)
	root, err := realPath(os.Args[1])
	if err != nil {
		die(err)
	}
	for _, dir := range peaDirs(root) {
		path, err := filepath.Rel(root, dir)
		if err != nil {
			die(err)
		}
		listed[path] = true
		if seen[path] != nil {
			continue
		}

		m, err := mod.Load(dir, path)
		if err != nil {
			die(err)
		}
		seen[path] = m
		mods = append(mods, m)
		if err := m.LoadDeps(root); err != nil {
			die(err)
		}
		seeDeps(m, seen)
	}
	// Pre-sort to make make tie-breaking alphabetical.
	sort.Slice(mods, func(i, j int) bool {
		return mods[i].SrcPath < mods[j].SrcPath
	})
	for _, mod := range mod.TopologicalDeps(mods) {
		if listed[mod.ModPath] {
			fmt.Println(mod.ModPath)
		}
	}
}

func seeDeps(root *mod.Mod, seen map[string]*mod.Mod) {
	for i, d := range root.Deps {
		s, ok := seen[d.ModPath]
		if ok {
			if s != d {
				root.Deps[i] = s
			}
			continue
		}
		seen[d.ModPath] = d
		seeDeps(d, seen)
	}
}

func peaDirs(root string) []string {
	f, err := os.Open(root)
	if err != nil {
		die(err)
	}
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		die(err)
	}
	if !finfo.IsDir() {
		return nil
	}

	finfos, err := f.Readdir(-1)
	if err != nil {
		die(err)
	}
	var dirs []string
	hasPeaSrc := false
	for _, finfo := range finfos {
		path := filepath.Join(root, finfo.Name())
		if strings.HasSuffix(path, ".pea") {
			hasPeaSrc = true
		}
		dirs = append(dirs, peaDirs(path)...)
	}
	if hasPeaSrc {
		dirs = append(dirs, root)
	}
	return dirs
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "Usage of %s:", os.Args[0])
	fmt.Fprintf(out, "%s [flags] <directory>", os.Args[0])
	flag.PrintDefaults()
}

func die(err error) {
	fmt.Fprintln(flag.CommandLine.Output(), err)
	os.Exit(1)
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
