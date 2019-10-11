package types

import (
	"fmt"
	"strings"
)

func (n Fun) debugString(x *scope) string {
	var s strings.Builder
	buildFunDebugString(x, &n, &s)
	return s.String()
}

func buildFunDebugString(x *scope, n *Fun, s *strings.Builder) {
	if n.Recv == nil {
		if n.Priv {
			s.WriteString("func ")
		} else {
			s.WriteString("Func ")
		}
	} else {
		if n.Priv {
			s.WriteString("meth ")
		} else {
			s.WriteString("Meth ")
		}
	}
	s.WriteString(n.Sig.Sel)
	if n.AST != nil {
		s.WriteString(" [")
		s.WriteString(x.loc(n).String())
		s.WriteRune(']')
	}
	s.WriteRune('\n')
	if n.Recv != nil {
		s.WriteString("\treceiver:")
		s.WriteString(fmt.Sprintf("\n\t\t(%d)%s", n.Recv.Arity, n.Recv.Name))
		if len(n.Recv.Parms) > 0 {
			s.WriteString("\n\t\ttype parameters")
			for i := range n.Recv.Parms {
				buildTypeVarDebugString(x, "\t\t\t\t", &n.Recv.Parms[i], s)
			}
		}
		s.WriteString("\n\treceiver type:")
		if n.Recv.Type == nil {
			s.WriteString("\n\t\t<error>")
		} else {
			buildTypeDebugString(x, "\t\t", n.Recv.Type, s)
		}
		s.WriteRune('\n')
	}
	if len(n.Sig.Parms) > 0 {
		s.WriteString("\tparameters:")
		for i := range n.Sig.Parms {
			p := &n.Sig.Parms[i]
			s.WriteString("\n\t\t" + p.Name)
			buildTypeDebugString(x, "\t\t\t", p.typ, s)
		}
		s.WriteRune('\n')
	}
	if n.Sig.Ret != nil {
		s.WriteString("\treturn:")
		buildTypeNameDebugString(x, "\t\t", n.Sig.Ret, s)
		s.WriteRune('\n')
	}
}

func buildTypeVarDebugString(x *scope, indent string, n *TypeVar, s *strings.Builder) {
	s.WriteString("\n" + indent)
	s.WriteString(fmt.Sprintf("%s (%p)", n.Name, n.Type))
	if len(n.Ifaces) > 0 {
		s.WriteString("\n" + indent + "\tinterfaces")
		for i := range n.Ifaces {
			buildTypeNameDebugString(x, indent+"\t\t", &n.Ifaces[i], s)
		}
	}
}

func buildTypeDebugString(x *scope, indent string, n *Type, s *strings.Builder) {
	s.WriteString("\n" + indent)
	if n.Priv {
		s.WriteString("type ")
	} else {
		s.WriteString("Type ")
	}
	s.WriteString(fmt.Sprintf("%s (%p)", n.Name, n))
	if n.AST != nil {
		s.WriteString(" [")
		s.WriteString(x.loc(n).String())
		s.WriteRune(']')
	}
	if len(n.Parms) > 0 {
		s.WriteString("\n" + indent + "\tparameters:")
		for i := range n.Parms {
			buildTypeVarDebugString(x, indent+"\t\t", &n.Parms[i], s)
		}
	}
	if len(n.Args) > 0 {
		s.WriteString("\n" + indent + "\targuments:")
		for i := range n.Args {
			buildTypeNameDebugString(x, indent+"\t\t", &n.Args[i], s)
		}
	}
}

func buildTypeNameDebugString(x *scope, indent string, n *TypeName, s *strings.Builder) {
	if n.Type == nil {
		s.WriteString("\n" + indent)
		s.WriteString(fmt.Sprintf("(%d)%s — <error>", len(n.Args), n.Name))
		return
	}
	buildTypeDebugString(x, indent, n.Type, s)
}
