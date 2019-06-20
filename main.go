package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

var (
	printParsed  = flag.Bool("parsed", false, "whether to print the parsed AST")
	printChecked = flag.Bool("checked", false, "whether to print checked AST")
	trace        = flag.Bool("trace", false, "whether to enable check tracing")
	dump         = flag.Bool("dump", false, "whether to dump the definition tree after checking")
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

	mod := parser.Mod()

	if *printParsed {
		fmt.Println("----- Parsed -----")
		pretty.Print(mod)
		fmt.Println("")
	}

	var opts []ast.Opt
	if *trace {
		opts = append(opts, ast.Trace)
	}
	if *dump {
		opts = append(opts, ast.Dump)
	}

	if errs := ast.Check(mod, opts...); len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		os.Exit(1)
	}

	if *printChecked {
		fmt.Println("----- Checked -----")
		// Don't print imports — too noisy.
		saved := mod.Imports
		mod.Imports = nil
		pretty.Print(mod)
		fmt.Println("")
		mod.Imports = saved
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
