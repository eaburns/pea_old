package main

import (
	"fmt"
	"os"

	"github.com/eaburns/pea/pea"
	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

func main() {
	pretty.Indent = "    "

	p := pea.NewParser("#Main")

	if len(os.Args) == 1 {
		if err := p.Parse("", os.Stdin); err != nil {
			die(err)
		}
	} else {
		for _, file := range os.Args[1:] {
			if err := p.ParseFile(file); err != nil {
				die(err)
			}
		}
	}

	mod := p.Mod()
	for _, d := range mod.Defs {
		fmt.Printf("%s: %s\n", mod.Loc(d), d.String())
		pretty.Print(d)
		fmt.Println("")
	}
	fmt.Println("")

	if errs := pea.Check(mod, pea.Trace); len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		os.Exit(1)
	}

	fmt.Println("Imports:")
	for _, m := range mod.Imports {
		if m.Name != "" {
			fmt.Println("---- module", m.Name)
		}
		for _, d := range m.Defs {
			fmt.Println(d.String())
		}
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
