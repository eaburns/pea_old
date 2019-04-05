package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/eaburns/peggy/peg"
	"github.com/eaburns/pretty"
)

func main() {
	pretty.Indent = "    "
	text := read()
	p := _NewParser(text)
	if pos, perr := _FileAccepts(p, 0); pos < 0 {
		_, t := _FileFail(p, 0, perr)
		if err := peg.PrettyWrite(os.Stdout, t); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("")
		err := peg.SimpleError(text, t)
		if len(os.Args) > 1 {
			err.FilePath = os.Args[1]
		}
		fmt.Println(err)
		os.Exit(1)
	}
	_, t := _FileAction(p, 0)
	for _, d := range (*t).Defs {
		pretty.Print(d)
		fmt.Println("")
		fmt.Println("")
	}
}

func read() string {
	var in io.Reader = os.Stdin
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer f.Close()
		in = f
	}
	data, err := ioutil.ReadAll(in)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return string(data)
}
