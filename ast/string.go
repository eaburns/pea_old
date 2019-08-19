package ast

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
	s.WriteString(n.Ident)
	if n.Type != nil {
		s.WriteRune(' ')
		buildTypeNameString(n.Type, &s)
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
		buildTypeSigString(n.Recv, &s)
		s.WriteRune(' ')
	} else {
		if n.priv {
			s.WriteString("func ")
		} else {
			s.WriteString("Func ")
		}
	}
	buildFunSigString(&n.Sig, &s)
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
	buildFunSigString(&n, &s)
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

func buildFunSigString(n *FunSig, s *strings.Builder) {
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
	s.WriteRune(']')
}

func buildTypeSigString(n *TypeSig, s *strings.Builder) {
	switch {
	case len(n.Parms) == 0:
		s.WriteString(n.Name)
	case len(n.Parms) == 1 && n.Parms[0].Type == nil:
		s.WriteString(n.Parms[0].Name)
		if !isOpType(n.Name) {
			s.WriteRune(' ')
		}
		s.WriteString(n.Name)
	default:
		buildTypeParms(n.Parms, s)
		if !isOpType(n.Name) {
			s.WriteRune(' ')
		}
		s.WriteString(n.Name)
	}
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
		s.WriteString(n.Name)
	case len(n.Args) == 1:
		buildTypeNameString(&n.Args[0], s)
		if !isOpType(n.Name) || isOpType(n.Args[0].Name) {
			s.WriteRune(' ')
		}
		s.WriteString(n.Name)
	default:
		s.WriteRune('(')
		for i, arg := range n.Args {
			if i > 0 {
				s.WriteString(", ")
			}
			buildTypeNameString(&arg, s)
		}
		s.WriteRune(')')
		if !isOpType(n.Name) {
			s.WriteRune(' ')
		}
		s.WriteString(n.Name)
	}
}
