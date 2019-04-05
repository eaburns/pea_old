package main

import (
	"fmt"
	"os"

	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

func main() {
	pretty.Indent = "    "

	p := NewParser("main")

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
	fmt.Println("=== " + m.Name)
	for _, f := range m.Files {
		fmt.Println("--- " + f.Path)
		for _, d := range f.Defs {
			pretty.Print(d)
			fmt.Println("")
		}
	}
	fmt.Println("")
}

func die(err error) {
	if pe, ok := err.(parseError); ok {
		peg.PrettyWrite(os.Stdout, pe.fail)
		fmt.Println("")
	}
	fmt.Println(err)
	os.Exit(1)
}
