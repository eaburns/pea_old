package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/types"
	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

var (
	printAST  = flag.Bool("ast", false, "print the AST to standard output")
	printType = flag.Bool("type", false, "print the type tree to standard output")
	trace     = flag.Bool("trace", false, "enable tracing in the type checker")
)

func main() {
	flag.Parse()

	pretty.Indent = "    "
	parser := ast.NewParser("#main")
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

	var opts []types.Opt
	if *trace {
		opts = append(opts, types.Trace)
	}
	typeMod, errs := types.Check(astMod, opts...)
	if len(errs) > 0 {
		fmt.Println(errs)
		os.Exit(1)
	}
	if *printType {
		// Clear out some noisy fields before printing.
		trimmedTypeMod := *typeMod
		trimmedTypeMod.AST = nil
		trimmedTypeMod.Imports = nil
		pretty.Print(trimmedTypeMod)
		fmt.Println("")
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
