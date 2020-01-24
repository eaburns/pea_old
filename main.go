package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/gengo"
	"github.com/eaburns/pea/types"
	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

var (
	printAST   = flag.Bool("ast", false, "print the AST")
	printTypes = flag.Bool("types", false, "print the semantic tree")
	printBasic = flag.Bool("basic", false, "print the basic representation")
	printGo    = flag.Bool("go", false, "print go code")
	runGo      = flag.Bool("rungo", false, "compiles to Go and runs")
	opt        = flag.Bool("opt", false, "optimize the basic representation")
	trace      = flag.Bool("trace", false, "enable tracing in the type checker")
)

func main() {
	flag.Parse()

	pretty.Indent = "    "
	parser := ast.NewParser("main")
	if len(flag.Args()) == 0 {
		if err := parser.Parse("", os.Stdin); err != nil {
			die(err)
		}
	} else {
		for _, file := range flag.Args() {
			if err := parser.ParseFile(file); err != nil {
				die(err)
			}
		}
	}

	astMod := parser.Mod()
	if *printAST {
		pretty.Print(astMod)
		fmt.Println("")
	}

	typesMod, errs := types.Check(astMod, types.Config{Trace: *trace})
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		os.Exit(1)
	}
	if *printTypes {
		// Clear out some noisy fields before printing.
		trimmedTypeMod := *typesMod
		trimmedTypeMod.AST = nil
		pretty.Print(trimmedTypeMod)
		fmt.Println("")
	}

	basicMod := basic.Build(typesMod)
	if *opt {
		basic.Optimize(basicMod)
	}
	if *printBasic {
		fmt.Println(basicMod.String())
	}

	if *printGo {
		writeGo(os.Stdout, basicMod)
	}
	if *runGo {
		run(basicMod)
	}
}

func writeGo(w io.Writer, mod *basic.Mod) {
	var b bytes.Buffer
	if err := gengo.WriteMod(&b, mod); err != nil {
		die(err)
	}
	merger, err := gengo.NewMerger(w)
	if err != nil {
		die(err)
	}
	if err := merger.Add(&b); err != nil {
		die(err)
	}
	if err := merger.Done(); err != nil {
		die(err)
	}
}

func run(mod *basic.Mod) {
	f, err := ioutil.TempFile("", "peac-run-*.go")
	if err != nil {
		die(err)
	}
	writeGo(f, mod)
	path := f.Name()
	if err := f.Close(); err != nil {
		die(err)
	}
	cmd := exec.Command("go", "run", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	runErr := cmd.Run()
	if runErr != nil {
		die(err)
	}
	if err := os.Remove(path); err != nil {
		die(err)
	}
}

func die(err error) {
	if pe, ok := err.(interface{ Tree() *peg.Fail }); ok {
		peg.PrettyWrite(os.Stdout, pe.Tree())
		fmt.Println("")
	}
	fmt.Println(err)
	os.Exit(1)
}
