package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eaburns/pea/sem"
	"github.com/eaburns/pea/syn"
	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

var (
	printSyn = flag.Bool("syn", false, "print the syntax tree to standard output")
	printSem = flag.Bool("sem", false, "print the semantic tree to standard output")
	trace    = flag.Bool("trace", false, "enable tracing in the type checker")
)

func main() {
	flag.Parse()

	pretty.Indent = "    "
	parser := syn.NewParser("#main")
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
	if *printSyn {
		pretty.Print(astMod)
		fmt.Println("")
	}

	typeMod, errs := sem.Check(astMod, sem.Config{Trace: *trace})
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		os.Exit(1)
	}
	if *printSem {
		// Clear out some noisy fields before printing.
		trimmedTypeMod := *typeMod
		trimmedTypeMod.AST = nil
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
