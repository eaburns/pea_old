package ast

import "fmt"

// A Loc describes the location in the source code.
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

// Loc returns the Loc for a node in the module AST.
func (m *Mod) Loc(n Node) Loc {
	if len(m.files) == 0 {
		return Loc{}
	}
	var l Loc
	var spath, epath string
	spath, l.Line[0], l.Col[0] = m.loc1(n.Start())
	epath, l.Line[1], l.Col[1] = m.loc1(n.End())
	if spath != epath {
		panic("impossible")
	}
	l.Path = spath
	return l
}

func (m *Mod) loc1(p int) (string, int, int) {
	file := m.files[0]
	for _, f := range m.files {
		if f.offs > p {
			break
		}
		file = f
	}
	line, col1 := 1, file.offs-1
	for _, nl := range file.lines {
		if nl >= p {
			break
		}
		col1 = nl
		line++
	}
	return file.path, line, p - col1
}
