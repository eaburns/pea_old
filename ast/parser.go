package ast

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/eaburns/peggy/peg"
)

// A Parser parses source code files.
type Parser struct {
	files []file
	defs  []Def
	offs  int
	mod   string
}

// NewParser returns a new parser for the named module.
func NewParser(mod string) *Parser {
	return &Parser{mod: mod}
}

// Mod returns the module built from the parsed files.
func (p *Parser) Mod() *Mod {
	return &Mod{Name: p.mod, files: p.files, Defs: p.defs}
}

// Parse parses a *File from an io.Reader.
// The first argument is the file path or "" if unspecified.
func (p *Parser) Parse(path string, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	_p := _NewParser(string(data))
	_p.data = p
	if pos, perr := _FileAccepts(_p, 0); pos < 0 {
		_, t := _FileFail(_p, 0, perr)
		return parseError{path: path, loc: perr, text: _p.text, fail: t}
	}
	_, defs := _FileAction(_p, 0)

	var lines []int
	for i, r := range _p.text {
		if r == '\n' {
			lines = append(lines, p.offs+i)
		}
	}
	p.files = append(p.files, file{path: path, offs: p.offs, lines: lines})
	p.offs += len(_p.text)
	p.defs = append(p.defs, *defs...)
	return nil
}

// ParseFile parses the source in the file specified by a path.
func (p *Parser) ParseFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return p.Parse(path, f)
}

type parseError struct {
	path string
	loc  int
	text string
	fail *peg.Fail
}

func (err parseError) Tree() *peg.Fail { return err.fail }

func (err parseError) Error() string {
	e := peg.SimpleError(err.text, err.fail)
	e.FilePath = err.path
	return e.Error()
}
