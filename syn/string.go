package syn

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func (n Val) String() string {
	var s strings.Builder
	if n.priv {
		s.WriteString("val ")
	} else {
		s.WriteString("Val ")
	}
	s.WriteString(n.Var.Name)
	if n.Var.Type != nil {
		s.WriteRune(' ')
		buildTypeNameString(n.Var.Type, &s)
	}
	return s.String()
}

func (n Fun) String() string {
	var s strings.Builder
	if n.Recv != nil {
		if n.priv {
			s.WriteString("meth ")
		} else {
			s.WriteString("Meth ")
		}
		buildRecvString(n.Recv, &s)
		s.WriteRune(' ')
	} else {
		if n.priv {
			s.WriteString("func ")
		} else {
			s.WriteString("Func ")
		}
	}
	if len(n.TParms) > 0 {
		if len(n.TParms) > 1 || n.TParms[0].Type != nil {
			s.WriteRune('(')
		}
		for i, parm := range n.TParms {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(parm.Name)
			if parm.Type != nil {
				s.WriteRune(' ')
				buildTypeNameString(parm.Type, &s)
			}
		}
		if len(n.TParms) > 1 || n.TParms[0].Type != nil {
			s.WriteRune(')')
		}
		s.WriteRune(' ')
	}
	buildFunSigString(&n.Sig, n.Stmts != nil, &s)
	return s.String()
}

func (n Type) String() string {
	var s strings.Builder
	if n.priv {
		s.WriteString("type ")
	} else {
		s.WriteString("Type ")
	}
	buildTypeSigString(&n.Sig, &s)
	return s.String()
}

func (n FunSig) String() string {
	var s strings.Builder
	buildFunSigString(&n, false, &s)
	return s.String()
}

func (n TypeSig) String() string {
	var s strings.Builder
	buildTypeSigString(&n, &s)
	return s.String()
}

func (n TypeName) String() string {
	var s strings.Builder
	buildTypeNameString(&n, &s)
	return s.String()
}

func buildFunSigString(n *FunSig, def bool, s *strings.Builder) {
	s.WriteRune('[')
	if len(n.Parms) == 0 { // unary
		s.WriteString(n.Sel)
	} else {
		keys := strings.SplitAfter(n.Sel, ":")
		for i, parm := range n.Parms {
			if i > 0 {
				s.WriteRune(' ')
			}
			s.WriteString(keys[i])
			s.WriteRune(' ')
			switch {
			case parm.Name != "" && parm.Type != nil:
				s.WriteString(parm.Name)
				s.WriteRune(' ')
				buildTypeNameString(parm.Type, s)
			case parm.Name != "":
				s.WriteString(parm.Name)
			case parm.Type != nil:
				buildTypeNameString(parm.Type, s)
			}
		}
	}
	if n.Ret != nil {
		s.WriteString(" ^")
		buildTypeNameString(n.Ret, s)
	}
	if def {
		s.WriteString(" |")
	}
	s.WriteRune(']')
}

func buildRecvString(n *Recv, s *strings.Builder) {
	switch {
	case len(n.Parms) == 0:
		break
	case len(n.Parms) == 1 && n.Parms[0].Type == nil:
		s.WriteString(n.Parms[0].Name)
		if n.Mod != nil || !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	default:
		buildTypeParms(n.Parms, s)
		if n.Mod != nil || !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	}
	if n.Mod != nil {
		s.WriteString(n.Mod.Text)
		s.WriteRune(' ')
	}
	s.WriteString(n.Name)
}

func buildTypeSigString(n *TypeSig, s *strings.Builder) {
	switch {
	case len(n.Parms) == 0:
		break
	case len(n.Parms) == 1 && n.Parms[0].Type == nil:
		s.WriteString(n.Parms[0].Name)
		if !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	default:
		buildTypeParms(n.Parms, s)
		if !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	}
	s.WriteString(n.Name)
}

func buildTypeParms(parms []Var, s *strings.Builder) {
	s.WriteRune('(')
	for i, parm := range parms {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(parm.Name)
		if parm.Type != nil {
			s.WriteRune(' ')
			buildTypeNameString(parm.Type, s)
		}
	}
	s.WriteRune(')')
}

func isOpType(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsPunct(r)
}

func buildTypeNameString(n *TypeName, s *strings.Builder) {
	switch {
	case len(n.Args) == 0:
		break
	case len(n.Args) == 1:
		buildTypeNameString(&n.Args[0], s)
		if n.Mod != nil || !isOpType(n.Name) || isOpType(n.Args[0].Name) {
			s.WriteRune(' ')
		}
	default:
		s.WriteRune('(')
		for i, arg := range n.Args {
			if i > 0 {
				s.WriteString(", ")
			}
			buildTypeNameString(&arg, s)
		}
		s.WriteRune(')')
		if n.Mod != nil || !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	}
	if n.Mod != nil {
		s.WriteString(n.Mod.Text)
		s.WriteRune(' ')
	}
	s.WriteString(n.Name)
}
