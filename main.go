package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/types"
	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

var (
	printAST   = flag.Bool("ast", false, "print the AST")
	printTypes = flag.Bool("types", false, "print the semantic tree")
	printBasic = flag.Bool("basic", false, "print the basic representation")
	opt        = flag.Bool("opt", false, "optimize the basic representation")
	trace      = flag.Bool("trace", false, "enable tracing in the type checker")
)

func main() {
	flag.Parse()

	pretty.Indent = "    "
	parser := ast.NewParser("/test/main")
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
}

func die(err error) {
	if pe, ok := err.(interface{ Tree() *peg.Fail }); ok {
		peg.PrettyWrite(os.Stdout, pe.Tree())
		fmt.Println("")
	}
	fmt.Println(err)
	os.Exit(1)
}
