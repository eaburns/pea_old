// Copyright © 2020 The Pea Authors under an MIT-style license.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/gengo"
	"github.com/eaburns/pea/mod"
	"github.com/eaburns/pea/types"
)

var (
	modPath       = flag.String("path", "main", "the current module's path")
	modRoot       = flag.String("root", ".", "root directory for imported modules")
	force         = flag.Bool("force", false, "force compilation event if up-to-date")
	test          = flag.Bool("test", false, "build a test executable")
	verbose       = flag.Bool("v", false, "enable verbose output")
	output        = flag.String("o", "", "name of executable file or directory")
	cleanUp       = flag.Bool("cleanup", true, "remove temporary files")
	cpuProfile    = flag.String("cpuprofile", "", "write cpu profile to file for the compiler")
	profileBinary = flag.Bool("profile_binary", false, "whether the generated binary should emit profiler output")
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if len(flag.Args()) != 1 {
		usage()
		os.Exit(1)
	}

	flag.Parse()
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			die("failed to create cpu profile file", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	srcPath := flag.Args()[0]
	root, err := mod.Load(srcPath, *modPath)
	if err != nil {
		die("failed to load module", err)
	}
	if err := root.LoadDeps(*modRoot); err != nil {
		die("failed to load dependencies", err)
	}
	for _, m := range mod.TopologicalDeps([]*mod.Mod{root}) {
		compile(m)
	}
	if *modPath == "main" || *test {
		link(root)
	}
}

func compile(m *mod.Mod) {
	objFile := objFile(m)
	srcMod := lastModTime(m.SrcFiles)
	// TODO: checking timestamps is insufficient to determine whether an object file is stale.
	// We could delete a source file; it will have no timestamp change, but the object file is now stale.
	// Instead, we should add the source file list to the object file and check that.
	if !*force && srcMod.Before(modTime(objFile)) {
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

func link(m *mod.Mod) {
	objFiles := objFiles(m)
	binFile := binFile()
	if !*force && lastModTime(objFiles).Before(modTime(binFile)) {
		return
	}

	goFile := merge(objFiles)
	objFile := binFile + ".o"

	vprintf("compiling %s\n", objFile)
	args := []string{"tool", "compile", "-o", objFile, goFile}
	args = append(args, goFiles(m)...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		die("failed to run go build", err)
	}

	if *cleanUp {
		os.Remove(goFile)
	}

	vprintf("linking %s\n", binFile)
	cmd = exec.Command("go", "tool", "link", "-o", binFile, objFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		die("failed to run go build", err)
	}

	if *cleanUp {
		os.Remove(objFile)
	}
}

func objFiles(m *mod.Mod) []string {
	var objFiles []string
	seen := make(map[*mod.Mod]bool)
	var addObjFiles func(*mod.Mod)
	addObjFiles = func(m *mod.Mod) {
		if seen[m] {
			return
		}
		seen[m] = true
		objFiles = append(objFiles, objFile(m))
		for _, d := range m.Deps {
			addObjFiles(d)
		}
	}
	addObjFiles(m)
	return objFiles
}

func binFile() string {
	dir := wd()
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
	if *test && *output == "" {
		binFile += "_test"
	}
	return binFile
}

func goFiles(m *mod.Mod) []string {
	var goFiles []string
	seen := make(map[*mod.Mod]bool)
	var addGoFiles func(*mod.Mod)
	addGoFiles = func(m *mod.Mod) {
		if seen[m] {
			return
		}
		seen[m] = true
		goFiles = append(goFiles, m.GoSrcFiles...)
		for _, d := range m.Deps {
			addGoFiles(d)
		}
	}
	addGoFiles(m)
	return goFiles
}

func merge(objFiles []string) string {
	f, err := ioutil.TempFile(wd(), "*.go")
	if err != nil {
		die("failed to make temp .go file", err)
	}
	goFile := f.Name()
	w := bufio.NewWriter(f)
	merger, err := gengo.NewMerger(w)
	if err != nil {
		die("failed to write Go header", err)
	}
	if *test {
		merger.TestMod = *modPath
	}
	merger.Profile = *profileBinary
	for _, file := range objFiles {
		f, err := os.Open(file)
		if err != nil {
			die("failed to open object file", err)
		}
		if err := merger.Add(bufio.NewReader(f)); err != nil {
			die("failed to read peago", err)
		}
		if err := f.Close(); err != nil {
			die("failed to close peago", err)
		}
	}
	vprintf("merging %s\n", goFile)
	if err := merger.Done(); err != nil {
		die("failed to write Go footer", err)
	}
	if err := w.Flush(); err != nil {
		die("failed to flush output", err)
	}
	if err := f.Close(); err != nil {
		die("failed to close", err)
	}
	return goFile
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
	fmt.Fprintf(out, "%s [flags] <module dir or file>\n", os.Args[0])
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
