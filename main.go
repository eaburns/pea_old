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

	p := pea.NewParser("main")

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

	m := p.Mod()
	for _, f := range m.Files {
		for _, d := range f.Defs {
			fmt.Println(m.Loc(d))
			pretty.Print(d)
			fmt.Println("")
		}
	}
	fmt.Println("")
}

func die(err error) {
	if pe, ok := err.(interface{ Tree() *peg.Fail }); ok {
		peg.PrettyWrite(os.Stdout, pe.Tree())
		fmt.Println("")
	}
	fmt.Println(err)
	os.Exit(1)
}
