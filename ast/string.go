package ast

import (
	"strings"
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
	case len(n.Parms) == 1 && n.Parms[0].Type == nil:
		s.WriteString(n.Parms[0].Name)
		s.WriteRune(' ')
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

func (n TypeName) String() string {
	var s strings.Builder
	buildTypeNameString(n, &s)
	return s.String()
}

func buildTypeNameString(n TypeName, s *strings.Builder) {
	switch {
	case len(n.Args) == 1:
		buildTypeNameString(n.Args[0], s)
		s.WriteRune(' ')
		fallthrough
	case len(n.Args) == 0:
		if n.Mod != nil && (n.Mod.Root != "" || len(n.Mod.Path) > 0) {
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

func (n MethSig) String() string {
	var s strings.Builder
	buildMethSigString(n, &s)
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

func methSigStringForUser(n MethSig) string {
	var s strings.Builder
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
			buildTypeStringForUser(&n.Parms[i], &s)
		}
	}
	if n.Ret != nil {
		s.WriteString(" ^")
		buildTypeStringForUser(n.Ret, &s)
	}
	s.WriteRune(']')
	return s.String()
}

func typeStringForUser(n *TypeName) string {
	var s strings.Builder
	buildTypeStringForUser(n, &s)
	return s.String()
}

// TODO: typeStringForUser should go away; all the .String methods should be for the user.
func buildTypeStringForUser(n *TypeName, s *strings.Builder) {
	if n.Type == nil {
		buildTypeNameString(*n, s)
		return
	}
	if n.Var {
		s.WriteString(n.Name)
		return
	}
	sig := n.Type.Sig
	if len(sig.Parms) > 1 {
		s.WriteRune('(')
	}
	for i := range sig.Parms {
		if i > 0 {
			s.WriteString(", ")
		}
		p := &sig.Parms[i]
		if a, ok := sig.Args[p]; ok {
			buildTypeStringForUser(&a, s)
		} else {
			s.WriteString(p.Name)
		}
	}
	switch len(sig.Parms) {
	case 0:
		break
	case 1:
		s.WriteRune(' ')
	default:
		s.WriteString(") ")
	}
	s.WriteString(sig.Name)
}