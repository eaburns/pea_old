package main

import (
	"fmt"
	"os"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

func main() {
	pretty.Indent = "    "

	parser := ast.NewParser("#main")

	if len(os.Args) == 1 {
		if err := parser.Parse("", os.Stdin); err != nil {
			die(err)
		}
	} else {
		for _, file := range os.Args[1:] {
			if err := parser.ParseFile(file); err != nil {
				die(err)
			}
		}
	}

	mod := parser.Mod()
	pretty.Print(mod)
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
