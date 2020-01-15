package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/gengo"
	"github.com/eaburns/pea/types"
)

var (
	modPath = flag.String("path", "main", "the current module's path")
	modRoot = flag.String("root", ".", "root directory for imported modules")
	force   = flag.Bool("force", false, "force compilation event if up-to-date")
	verbose = flag.Bool("v", false, "enable verbose output")
	output  = flag.String("o", "", "name of executable file or directory")
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if len(flag.Args()) != 1 {
		usage()
		os.Exit(1)
	}
	srcPath := flag.Args()[0]
	root := depGraph(srcPath, *modPath)
	for _, m := range sortedTransitiveDeps(root) {
		compile(m)
	}
	if *modPath == "main" {
		link(root)
	}
}

type mod struct {
	modPath string
	srcPath string
	// srcDir may differ from srcPath for the root module
	// if the root module is given as a .peago file, not a directory.
	srcDir   string
	srcFiles []string
	objFile  string

	deps []*mod
}

func newMod(srcPath, modPath string) *mod {
	srcPath = realPath(srcPath)
	srcFiles, srcDir := srcFiles(srcPath)
	objFile := filepath.Join(srcDir, filepath.Base(srcDir)+".peago")
	return &mod{
		modPath:  modPath,
		srcPath:  srcPath,
		srcDir:   srcDir,
		srcFiles: srcFiles,
		objFile:  objFile,
	}
}

func srcFiles(srcPath string) ([]string, string) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		die("failed to open the source path", err)
	}
	defer srcFile.Close()
	stat, err := srcFile.Stat()
	if err != nil {
		die("failed to stat", err)
	}
	if !stat.IsDir() {
		return []string{srcPath}, filepath.Dir(srcPath)
	}
	finfos, err := srcFile.Readdir(-1)
	if err != nil {
		die("failed to read the source directory", err)
	}
	var paths []string
	for _, finfo := range finfos {
		if !strings.HasSuffix(finfo.Name(), ".pea") {
			continue
		}
		path := filepath.Join(srcPath, finfo.Name())
		paths = append(paths, path)
	}
	return paths, srcPath
}

func sortedTransitiveDeps(root *mod) []*mod {
	var sorted []*mod
	seen := make(map[*mod]bool)
	var add func(*mod)
	add = func(m *mod) {
		if seen[m] {
			return
		}
		seen[m] = true
		for _, d := range m.deps {
			add(d)
		}
		sorted = append(sorted, m)
	}
	add(root)
	return sorted
}

func depGraph(srcPath, modPath string) *mod {
	seen := make(map[string]*mod)
	root := newMod(srcPath, modPath)
	seen[modPath] = root
	var addDeps func(*mod)
	addDeps = func(m *mod) {
		for _, depFile := range deps(m.srcFiles) {
			if d, ok := seen[depFile]; ok {
				m.deps = append(m.deps, d)
				continue
			}
			d := newMod(filepath.Join(*modRoot, depFile), depFile)
			m.deps = append(m.deps, d)
			seen[depFile] = d
			addDeps(d)
		}
	}
	addDeps(root)
	return root
}

func deps(srcFiles []string) []string {
	var deps []string
	for _, file := range srcFiles {
		f, err := os.Open(file)
		if err != nil {
			die("failed to open source file", err)
		}
		ds, err := ast.ReadImports(bufio.NewReader(f))
		if err != nil {
			die("failed to read imports", err)
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
	return deps[:i]
}

func compile(m *mod) {
	if !*force && lastModTime(m.srcFiles).Before(modTime(m.objFile)) {
		vprintf("ok %s\n", m.modPath)
		return
	}
	vprintf("building %s\n", m.modPath)
	astMod := parse(m)
	typesMod := check(astMod)
	basicMod := basic.Build(typesMod)
	basic.Optimize(basicMod)
	writeObj(basicMod, m.objFile)
}

func parse(m *mod) *ast.Mod {
	p := ast.NewParser(m.modPath)
	for _, srcFile := range m.srcFiles {
		if err := p.ParseFile(srcFile); err != nil {
			die("", err)
		}
	}
	return p.Mod()
}

func check(astMod *ast.Mod) *types.Mod {
	typesMod, errs := types.Check(astMod, types.Config{
		Importer: &types.SourceImporter{Root: *modRoot},
	})
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(flag.CommandLine.Output(), err)
		}
		os.Exit(1)
	}
	return typesMod
}

func writeObj(basicMod *basic.Mod, objFile string) {
	vprintf("writing %s\n", objFile)
	f, err := os.Create(objFile)
	if err != nil {
		die("failed to create object file", err)
	}
	w := bufio.NewWriter(f)
	if err := gengo.WriteMod(w, basicMod); err != nil {
		die("failed to write object file", err)
	}
	if err := w.Flush(); err != nil {
		die("failed to flush object file buffer", err)
	}
	if err := f.Close(); err != nil {
		die("failed to close object file", err)
	}
}

func link(m *mod) {
	objFiles := []string{m.objFile}
	for _, d := range m.deps {
		objFiles = append(objFiles, d.objFile)
	}

	dir := wd()
	goFile := filepath.Join(dir, filepath.Base(dir)+".go")
	merge(objFiles, goFile)

	var binFile string
	if *output == "" {
		binFile = filepath.Join(dir, filepath.Base(dir))
	} else {
		fi, err := os.Stat(*output)
		if os.IsNotExist(err) || !fi.IsDir() {
			binFile = *output
		} else if err != nil {
			die("failed to stat output file", err)
		} else {
			binFile = filepath.Join(*output, filepath.Base(dir))
		}
	}

	if !*force && modTime(goFile).Before(modTime(binFile)) {
		return
	}
	vprintf("linking %s\n", binFile)
	cmd := exec.Command("go", "build", "-o", binFile, goFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		die("failed to run go build", err)
	}
}

func merge(objFiles []string, goFile string) {
	if !*force && lastModTime(objFiles).Before(modTime(goFile)) {
		return
	}
	var rs []io.Reader
	for _, file := range objFiles {
		f, err := os.Open(file)
		if err != nil {
			die("failed to open object file", err)
		}
		defer f.Close()
		rs = append(rs, bufio.NewReader(f))
	}
	f, err := os.Create(goFile)
	if err != nil {
		die("failed to create temp file", err)
	}
	vprintf("merging %s:\n%v\n", goFile, objFiles)
	w := bufio.NewWriter(f)
	if err := gengo.MergeMods(w, rs); err != nil {
		die("failed to write Go file", err)
	}
	if err := w.Flush(); err != nil {
		die("failed to flush output", err)
	}
	if err := f.Close(); err != nil {
		die("failed to close", err)
	}
}

func realPath(dir string) string {
	switch dir {
	case string([]rune{filepath.Separator}):
		return dir
	case ".":
		return wd()
	default:
		base := filepath.Base(dir)
		dir = realPath(filepath.Dir(dir))
		switch base {
		case ".":
			return dir
		case "..":
			return filepath.Dir(dir)
		default:
			return filepath.Join(dir, base)
		}
	}
}

func lastModTime(files []string) time.Time {
	var t time.Time
	for _, file := range files {
		if mt := modTime(file); mt.After(t) {
			t = mt
		}
	}
	return t
}

func modTime(file string) time.Time {
	finfo, err := os.Stat(file)
	switch {
	case os.IsNotExist(err):
		return time.Time{}
	case err != nil:
		die("failed to get mod time", err)
	}
	return finfo.ModTime()
}

func wd() string {
	dir, err := os.Getwd()
	if err != nil {
		die("failed to get the current directory", err)
	}
	return dir
}

func vprintf(f string, vs ...interface{}) {
	if *verbose {
		fmt.Printf(f, vs...)
	}
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "Usage of %s:", os.Args[0])
	fmt.Fprintf(out, "%s [flags] <module dir or file>", os.Args[0])
	flag.PrintDefaults()
}

func die(s string, err error) {
	if s == "" {
		fmt.Fprintln(flag.CommandLine.Output(), err)
	} else {
		fmt.Fprintf(flag.CommandLine.Output(), "%s: %s\n", s, err)
	}
	os.Exit(1)
}
