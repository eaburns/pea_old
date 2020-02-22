// Copyright © 2020 The Pea Authors under an MIT-style license.

package ast

import "io"

// ReadImports returns all imports from the source given by an io.Reader.
func ReadImports(path string, r io.Reader) ([]string, error) {
	// TODO: implement a more efficient ReadImports.
	// The current implementation reads and fully parses
	// the entire source file. However, imports always come first,
	// so it should be easy to implement a variant of this
	// that only reads and parses the beginning of the input.
	p := NewParser("")
	if err := p.Parse(path, r); err != nil {
		return nil, err
	}

	var paths []string
	for _, f := range p.Mod().Files {
		for _, imp := range f.Imports {
			path := imp.Path[1 : len(imp.Path)-1] // trim "
			paths = append(paths, path)
		}
	}
	return paths, nil
}
