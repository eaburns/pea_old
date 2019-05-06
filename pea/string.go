package pea

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func (n ModPath) String() string {
	var s strings.Builder
	buildModPathString(n, &s)
	return s.String()
}

func buildModPathString(n ModPath, s *strings.Builder) {
	s.WriteString(n.Root)
	for _, p := range n.Path {
		s.WriteRune(' ')
		s.WriteString(p)
	}
}

func (n *Import) String() string { return "import " + n.Path }

func (n *Fun) String() string {
	var s strings.Builder
	if n.ModPath.Root != "" || len(n.ModPath.Path) > 0 {
		buildModPathString(n.ModPath, &s)
		s.WriteRune(' ')
	}
	if n.Recv != nil {
		buildTypeSigString(*n.Recv, &s)
		s.WriteRune(' ')
	}
	if len(n.TypeParms) == 1 && n.TypeParms[0].Type == nil {
		s.WriteString(n.TypeParms[0].Name)
		s.WriteRune(' ')
	} else if len(n.TypeParms) > 0 {
		s.WriteRune('(')
		for i, t := range n.TypeParms {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(t.Name)
			if t.Type != nil {
				s.WriteRune(' ')
				buildTypeNameString(*t.Type, &s)
			}
		}
		s.WriteString(") ")
	}
	s.WriteRune('[')
	if len(n.Parms) == 0 {
		s.WriteString(n.Sel)
	} else {
		for i, sel := range strings.SplitAfter(n.Sel, ":") {
			if sel == "" {
				break
			}
			if i > 0 {
				s.WriteRune(' ')
			}
			s.WriteString(sel)
			s.WriteRune(' ')
			buildTypeNameString(*n.Parms[i].Type, &s)
		}
	}
	if n.Ret != nil {
		s.WriteString(" ^")
		buildTypeNameString(*n.Ret, &s)
	}
	s.WriteRune(']')

	return s.String()
}

func (n *Var) String() string {
	var s strings.Builder
	buildModPathString(n.ModPath, &s)
	s.WriteRune(' ')
	s.WriteString(n.Ident)
	return s.String()
}

func (n TypeSig) String() string {
	var s strings.Builder
	buildTypeSigString(n, &s)
	return s.String()
}

func buildTypeSigString(n TypeSig, s *strings.Builder) {
	switch {
	case isFunName(n.Name):
		s.WriteRune('[')
		for i, p := range n.Parms {
			if i == len(n.Parms)-1 && isRetFunName(n.Name) {
				s.WriteString(" | ")
			} else if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(p.Name)
			if p.Type != nil {
				panic("impossible")
			}
		}
		s.WriteRune(']')
		return
	case len(n.Parms) == 1 && n.Parms[0].Type == nil:
		s.WriteString(n.Parms[0].Name)
		if r, _ := utf8.DecodeRuneInString(n.Name); !unicode.IsPunct(r) {
			s.WriteRune(' ')
		}
		fallthrough
	case len(n.Parms) == 0:
		s.WriteString(n.Name)
		return
	}
	s.WriteRune('(')
	for i, p := range n.Parms {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(p.Name)
		if p.Type != nil {
			s.WriteRune(' ')
			buildTypeNameString(*p.Type, s)
		}
	}
	s.WriteString(") ")
	s.WriteString(n.Name)
}

func isFunName(n string) bool    { return len(n) > 0 && n[0] == '[' }
func isRetFunName(n string) bool { return len(n) > 2 && n[1] == '|' }

func (n TypeName) String() string {
	var s strings.Builder
	buildTypeNameString(n, &s)
	return s.String()
}

func buildTypeNameString(n TypeName, s *strings.Builder) {
	switch {
	case isFunName(n.Name):
		if n.Mod != nil {
			buildModPathString(*n.Mod, s)
			s.WriteRune(' ')
		}
		s.WriteRune('[')
		for i, a := range n.Args {
			if i == len(n.Args)-1 && isRetFunName(n.Name) {
				s.WriteString(" | ")
			} else if i > 0 {
				s.WriteString(", ")
			}
			buildTypeNameString(a, s)
		}
		s.WriteRune(']')
		return
	case len(n.Args) == 1:
		buildTypeNameString(n.Args[0], s)
		s.WriteRune(' ')
		fallthrough
	case len(n.Args) == 0:
		if n.Mod != nil {
			buildModPathString(*n.Mod, s)
			s.WriteRune(' ')
		}
		s.WriteString(n.Name)
		return
	}
	s.WriteRune('(')
	for i, a := range n.Args {
		if i > 0 {
			s.WriteString(", ")
		}
		buildTypeNameString(a, s)
	}
	s.WriteString(") ")
	if n.Mod != nil {
		buildModPathString(*n.Mod, s)
		s.WriteRune(' ')
	}
	s.WriteString(n.Name)
}

func (n *Type) String() string {
	var s strings.Builder
	if n.ModPath.Root != "" || len(n.ModPath.Path) > 0 {
		buildModPathString(n.ModPath, &s)
		s.WriteRune(' ')
	}
	buildTypeSigString(n.Sig, &s)

	switch {
	case n.Alias != nil:
		s.WriteString(" := ")
		buildTypeNameString(*n.Alias, &s)
		s.WriteRune('.')
	case n.Fields != nil:
		s.WriteString(" {")
		for i, f := range n.Fields {
			if i > 0 {
				s.WriteRune(' ')
			}
			s.WriteString(f.Name)
			s.WriteString(": ")
			buildTypeNameString(*f.Type, &s)
		}
		s.WriteRune('}')
	case n.Cases != nil:
		s.WriteString(" {")
		for i, c := range n.Cases {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(c.Name)
			if c.Type != nil {
				s.WriteString(": ")
				buildTypeNameString(*c.Type, &s)
			}
		}
		s.WriteRune('}')
	case n.Virts != nil:
		s.WriteString(" {")
		for i, v := range n.Virts {
			if i > 0 {
				s.WriteRune(' ')
			}
			buildMethSigString(v, &s)
		}
		s.WriteRune('}')
	default:
		s.WriteString(" {}")
	}
	return s.String()
}

func buildMethSigString(n MethSig, s *strings.Builder) {
	s.WriteRune('[')
	if len(n.Parms) == 0 {
		s.WriteString(n.Sel)
	} else {
		for i, sel := range strings.SplitAfter(n.Sel, ":") {
			if sel == "" {
				break
			}
			if i > 0 {
				s.WriteRune(' ')
			}
			s.WriteString(sel)
			s.WriteRune(' ')
			buildTypeNameString(n.Parms[i], s)
		}
	}
	if n.Ret != nil {
		s.WriteString(" ^")
		buildTypeNameString(*n.Ret, s)
	}
	s.WriteRune(']')
}
