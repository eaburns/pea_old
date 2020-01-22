package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/gengo"
	"github.com/eaburns/pea/mod"
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
	root, err := mod.NewMod(srcPath, *modPath, mod.Config{
		ImportRootDir: *modRoot,
	})
	if err != nil {
		die("failed to load module", err)
	}
	for _, m := range mod.TopologicalDeps(root) {
		compile(m)
	}
	if *modPath == "main" {
		link(root)
	}
}

func compile(m *mod.Mod) {
	objFile := objFile(m)
	if !*force && lastModTime(m.SrcFiles).Before(modTime(objFile)) {
		vprintf("ok %s\n", m.ModPath)
		return
	}
	vprintf("building %s\n", m.ModPath)
	astMod := parse(m)
	typesMod := check(astMod)
	basicMod := basic.Build(typesMod)
	basic.Optimize(basicMod)
	writeObj(basicMod, objFile)
}

func objFile(m *mod.Mod) string {
	return filepath.Join(m.SrcDir, m.ModName+".peago")
}

func parse(m *mod.Mod) *ast.Mod {
	p := ast.NewParser(m.ModPath)
	for _, srcFile := range m.SrcFiles {
		if err := p.ParseFile(srcFile); err != nil {
			die("", err)
		}
	}
	return p.Mod()
}

var testFileRegexp = regexp.MustCompile(".*_test.pea$")

func check(astMod *ast.Mod) *types.Mod {
	typesMod, errs := types.Check(astMod, types.Config{
		Importer: &types.SourceImporter{
			Root:   *modRoot,
			Ignore: testFileRegexp,
		},
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

func link(m *mod.Mod) {
	objFiles := []string{objFile(m)}
	for _, d := range m.Deps {
		objFiles = append(objFiles, objFile(d))
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
