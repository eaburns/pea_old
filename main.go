package main

import (
	"fmt"
	"io"
	"os"

	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

func main() {
	pretty.Indent = "    "

	var path string
	var in io.Reader = os.Stdin
	if len(os.Args) > 1 {
		path = os.Args[1]
		f, err := os.Open(path)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()
		in = f
	}
	file, err := Parse(path, in)
	if err != nil {
		peg.PrettyWrite(os.Stdout, err.(parseError).fail)
		fmt.Println("")
		fmt.Println(err)
		os.Exit(1)
	}
	for _, d := range file.Defs {
		pretty.Print(d)
		fmt.Println("")
		fmt.Println("")
	}
}
