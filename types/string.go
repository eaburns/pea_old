package types

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func (n Val) String() string {
	var s strings.Builder
	if n.Priv {
		s.WriteString("val ")
	} else {
		s.WriteString("Val ")
	}
	s.WriteString(n.Var.Name)
	if n.Var.TypeName != nil {
		s.WriteRune(' ')
		buildTypeNameString(n.Var.TypeName, &s)
	}
	return s.String()
}

func (n Fun) String() string {
	var s strings.Builder
	if n.Recv != nil {
		if n.Priv {
			s.WriteString("meth ")
		} else {
			s.WriteString("Meth ")
		}
		buildRecvString(n.Recv, &s)
		s.WriteRune(' ')
	} else {
		if n.Priv {
			s.WriteString("func ")
		} else {
			s.WriteString("Func ")
		}
	}
	buildFunSigString(&n.Sig, n.Recv != nil, &s)
	return s.String()
}

func (n Type) String() string {
	var s strings.Builder
	if n.Priv {
		s.WriteString("type ")
	} else {
		s.WriteString("Type ")
	}
	buildTypeSigString(&n.Sig, &s)
	return s.String()
}

func (n Type) fullString() string {
	var s strings.Builder
	if n.Priv {
		s.WriteString("type ")
	} else {
		s.WriteString("Type ")
	}
	buildTypeSigString(&n.Sig, &s)
	if n.Alias != nil {
		s.WriteString(" := ")
		s.WriteString(n.Alias.String())
		s.WriteRune('.')
		return s.String()
	}
	s.WriteString(" {")
	l := s.Len()
	for _, v := range n.Fields {
		s.WriteRune(' ')
		s.WriteString(v.Name)
		s.WriteString(": ")
		buildTypeNameString(v.TypeName, &s)
	}
	for i, v := range n.Cases {
		if i > 0 {
			s.WriteString(", ")
		} else {
			s.WriteRune(' ')
		}
		s.WriteString(v.Name)
		if v.TypeName == nil {
			continue
		}
		s.WriteRune(' ')
		buildTypeNameString(v.TypeName, &s)
	}
	for _, v := range n.Virts {
		s.WriteRune(' ')
		buildFunSigString(&v, false, &s)
	}
	if s.Len() > l {
		s.WriteRune(' ')
	}
	s.WriteRune('}')
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

func buildFunSigString(n *FunSig, stripSelf bool, s *strings.Builder) {
	s.WriteRune('[')
	parms := n.Parms
	if stripSelf {
		parms = parms[1:]
	}
	if len(parms) == 0 { // unary
		s.WriteString(n.Sel)
	} else {
		keys := strings.SplitAfter(n.Sel, ":")
		for i, parm := range parms {
			if i > 0 {
				s.WriteRune(' ')
			}
			s.WriteString(keys[i])
			s.WriteRune(' ')
			switch {
			case parm.Name != "" && parm.TypeName != nil:
				s.WriteString(parm.Name)
				s.WriteRune(' ')
				buildTypeNameString(parm.TypeName, s)
			case parm.Name != "":
				s.WriteString(parm.Name)
			case parm.TypeName != nil:
				buildTypeNameString(parm.TypeName, s)
			}
		}
	}
	if n.Ret != nil {
		s.WriteString(" ^")
		buildTypeNameString(n.Ret, s)
	}
	s.WriteRune(']')
}

func buildRecvString(n *Recv, s *strings.Builder) {
	switch {
	case len(n.Parms) == 0:
		break
	case len(n.Parms) == 1 && n.Parms[0].TypeName == nil:
		s.WriteString(n.Parms[0].Name)
		if n.Mod != "" || !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	default:
		buildTypeParms(n.Parms, s)
		if n.Mod != "" || !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	}
	if n.Mod != "" {
		s.WriteString(n.Mod)
		s.WriteRune(' ')
	}
	s.WriteString(n.Name)
}

func buildTypeSigString(n *TypeSig, s *strings.Builder) {
	switch {
	case len(n.Args) == 1:
		buildTypeNameString(&n.Args[0], s)
		if !isOpType(n.Name) || isOpType(n.Args[0].Name) {
			s.WriteRune(' ')
		}
	case len(n.Args) > 1:
		s.WriteRune('(')
		for i := range n.Args {
			if i > 0 {
				s.WriteString(", ")
			}
			buildTypeNameString(&n.Args[i], s)
		}
		s.WriteRune(')')
		if !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	case len(n.Parms) == 0:
		break
	case len(n.Parms) == 1 && n.Parms[0].TypeName == nil:
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
		if parm.TypeName != nil {
			s.WriteRune(' ')
			buildTypeNameString(parm.TypeName, s)
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
		if n.Mod != "" || !isOpType(n.Name) || isOpType(n.Args[0].Name) {
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
		if n.Mod != "" || !isOpType(n.Name) {
			s.WriteRune(' ')
		}
	}
	if n.Mod != "" {
		s.WriteString(n.Mod)
		s.WriteRune(' ')
	}
	s.WriteString(n.Name)
}
