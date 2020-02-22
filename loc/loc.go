// Copyright © 2020 The Pea Authors under an MIT-style license.

// Package loc has routines for tracking file locations.
package loc

import "fmt"

// A Range is a start and end byte offset.
type Range [2]int

// GetRange returns itself.
// This is useful so than Range can be embedded in a struct
// and that struct can implement interface{GetRange() Range}.
func (r Range) GetRange() Range { return r }

// A Loc describes a file location.
type Loc struct {
	Path string
	Line [2]int
	Col  [2]int
}

func (l Loc) String() string {
	switch {
	case l.Line[0] == l.Line[1] && l.Col[0] == l.Col[1]:
		return fmt.Sprintf("%s:%d.%d", l.Path, l.Line[0], l.Col[0])
	default:
		return fmt.Sprintf("%s:%d.%d-%d.%d", l.Path, l.Line[0], l.Col[0], l.Line[1], l.Col[1])
	}
}

// Files tracks locations within a set of files.
type Files []File

// A File is a single file in a Files.
type File struct {
	Path  string
	Offs  int
	Len   int
	Lines []int
}

// Len returns the total length of all files.
func (fs Files) Len() int {
	if len(fs) == 0 {
		return 0
	}
	last := fs[len(fs)-1]
	return last.Offs + last.Len
}

// Add adds a new file to the set given its path and text.
func (fs *Files) Add(path, text string) {
	var lines []int
	offs := fs.Len()
	for i, r := range text {
		if r == '\n' {
			lines = append(lines, offs+i)
		}
	}
	*fs = append(*fs, File{
		Path:  path,
		Offs:  offs,
		Len:   len(text),
		Lines: lines,
	})
}

// Loc returns the Loc for a node in the module AST.
func (fs Files) Loc(r Range) *Loc {
	if fs == nil || r[0] < 0 || r[1] > fs.Len() {
		return nil
	}
	var l Loc
	var spath, epath string
	spath, l.Line[0], l.Col[0] = fs.loc1(r[0])
	epath, l.Line[1], l.Col[1] = fs.loc1(r[1])
	if spath != epath {
		panic("impossible")
	}
	l.Path = spath
	return &l
}

func (fs Files) loc1(p int) (string, int, int) {
	file := fs[0]
	for _, f := range fs {
		if f.Offs > p {
			break
		}
		file = f
	}
	line, col1 := 1, file.Offs-1
	for _, nl := range file.Lines {
		if nl >= p {
			break
		}
		col1 = nl
		line++
	}
	return file.Path, line, p - col1
}
