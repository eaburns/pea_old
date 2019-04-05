package main

import (
	"io"
	"io/ioutil"

	"github.com/eaburns/peggy/peg"
)

type parseError struct {
	path string
	loc  int
	text string
	fail *peg.Fail
}

func (err parseError) Error() string {
	e := peg.SimpleError(err.text, err.fail)
	e.FilePath = err.path
	return e.Error()
}

// Parse parses a *File from an io.Reader.
// The first argument is the file path or "" if unspecified.
func Parse(path string, r io.Reader) (*File, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p := _NewParser(string(data))
	if pos, perr := _FileAccepts(p, 0); pos < 0 {
		_, t := _FileFail(p, 0, perr)
		return nil, parseError{path: path, loc: perr, text: p.text, fail: t}
	}
	_, t := _FileAction(p, 0)
	file := *t
	file.Path = path
	file.Text = p.text
	return file, nil
}
