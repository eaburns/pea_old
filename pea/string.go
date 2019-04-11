package pea

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func (n *SubMod) String() string { return "submodule: " + n.Mod.String() }

func (n ModPath) String() string {
	var s strings.Builder
	buildModPathString(n, &s)
	return s.String()
}

func buildModPathString(n ModPath, s *strings.Builder) {
	for i, m := range n {
		if i > 0 {
			s.WriteRune(' ')
		}
		s.WriteString(m.Text)
	}
}

func (n *Import) String() string { return "import: " + n.Path }

func (n *Fun) String() string {
	var s strings.Builder
	s.WriteString("function: ")
	buildFunString(n, &s)
	return s.String()
}

// Key returns a unique identifier for the function.
// It is an error if two functions have the same key.
// The exact format of the key is unspecified, but it includes
// the module path, the Key of the receiver type if any, and the selector.
func (n *Fun) Key() string {
	var s strings.Builder
	buildModPathString(n.Mod, &s)
	s.WriteRune(' ')
	if n.Recv != nil {
		s.WriteString(n.Recv.Name)
		s.WriteRune(' ')
	}
	s.WriteString(n.Sel)
	return s.String()
}

func buildFunString(n *Fun, s *strings.Builder) {
	buildModPathString(n.Mod, s)
	s.WriteRune(' ')
	if n.Recv != nil {
		buildTypeSigString(*n.Recv, s)
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
				buildTypeNameString(*t.Type, s)
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
			buildTypeNameString(*n.Parms[i].Type, s)
		}
	}
	if n.Ret != nil {
		s.WriteString(" ^")
		buildTypeNameString(*n.Ret, s)
	}
	s.WriteRune(']')
}

func (n *Var) String() string {
	var s strings.Builder
	s.WriteString("variable: ")
	buildModPathString(n.Mod, &s)
	s.WriteRune(' ')
	s.WriteString(n.Name)
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

func (n TypeName) String() string {
	var s strings.Builder
	buildTypeNameString(n, &s)
	return s.String()
}

// Key returns a unique identifier for the type.
// The key is the module path and the name of the type.
// The key will match the Key() of the type's definition.
func (n TypeName) Key() string { return n.Name }

func buildTypeNameString(n TypeName, s *strings.Builder) {
	switch {
	case len(n.Name) > 0 && n.Name[0] == '[':
		s.WriteRune('[')
		for i, a := range n.Args {
			if i == len(n.Args)-1 && len(n.Name) > 2 && n.Name[1] == '|' {
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
		if r, _ := utf8.DecodeRuneInString(n.Name); !unicode.IsPunct(r) {
			s.WriteRune(' ')
		}
		fallthrough
	case len(n.Args) == 0:
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
	s.WriteString(n.Name)
}

func (n *Struct) String() string {
	var s strings.Builder
	s.WriteString("struct: ")
	buildModPathString(n.Mod, &s)
	s.WriteRune(' ')
	buildTypeSigString(n.Sig, &s)
	s.WriteString(" {")
	for i, field := range n.Fields {
		if i > 0 {
			s.WriteRune(' ')
		}
		s.WriteString(field.Name)
		s.WriteRune(' ')
		buildTypeNameString(*field.Type, &s)
	}
	s.WriteRune('}')
	return s.String()
}

// Key returns a unique identifier for the type.
// The exact format is unspecified,
// but it includes the module path and type name.
func (n *Struct) Key() string { return n.Mod.String() + " " + n.Sig.Name }

func (n *Enum) String() string {
	var s strings.Builder
	s.WriteString("enum: ")
	buildModPathString(n.Mod, &s)
	s.WriteRune(' ')
	buildTypeSigString(n.Sig, &s)
	s.WriteString(" {")
	for i, cas := range n.Cases {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(cas.Name)
		if cas.Type != nil {
			s.WriteRune(' ')
			buildTypeNameString(*cas.Type, &s)
		}
	}
	s.WriteRune('}')
	return s.String()
}

// Key returns a unique identifier for the type.
// The exact format is unspecified,
// but it includes the module path and type name.
func (n *Enum) Key() string { return n.Mod.String() + " " + n.Sig.Name }

func (n *Virt) String() string {
	var s strings.Builder
	s.WriteString("virtual: ")
	buildModPathString(n.Mod, &s)
	s.WriteRune(' ')
	buildTypeSigString(n.Sig, &s)
	s.WriteString(" {")
	for i, meth := range n.Meths {
		if i > 0 {
			s.WriteRune(' ')
		}
		buildMethSigString(meth, &s)
	}
	s.WriteRune('}')
	return s.String()
}

// Key returns a unique identifier for the type.
// The exact format is unspecified,
// but it includes the module path and type name.
func (n *Virt) Key() string { return n.Mod.String() + " " + n.Sig.Name }

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
