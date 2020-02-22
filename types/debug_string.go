// Copyright © 2020 The Pea Authors under an MIT-style license.

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

func (n *Type) debugString(x *scope) string {
	var s strings.Builder
	buildTypeDebugString(x, n, &s)
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
			buildTypeSigDebugString(x, "\t\t", n.Recv.Type, s)
		}
		s.WriteRune('\n')
	}
	if len(n.Sig.Parms) > 0 {
		s.WriteString("\tparameters:")
		for i := range n.Sig.Parms {
			p := &n.Sig.Parms[i]
			s.WriteString("\n\t\t" + p.Name)
			if p.Type() != nil {
				buildTypeSigDebugString(x, "\t\t\t", p.Type(), s)
			}
		}
		s.WriteRune('\n')
	}
	if n.Sig.Ret != nil {
		s.WriteString("\treturn:")
		buildTypeNameDebugString(x, "\t\t", n.Sig.Ret, s)
		s.WriteRune('\n')
	}
}

func buildTypeDebugString(x *scope, n *Type, s *strings.Builder) {
	buildTypeSigDebugString(x, "", n, s)
	switch {
	case n.Var != nil:
		s.WriteString("\n\tvariable:")
		buildTypeVarDebugString(x, "\t", n.Var, s)
	case len(n.Fields) > 0:
		s.WriteString("\n\tfields:")
		for _, f := range n.Fields {
			s.WriteString("\n\t\t")
			s.WriteString(f.Name)
			buildTypeNameDebugString(x, "\t\t\t", f.TypeName, s)
		}
	case len(n.Cases) > 0:
		s.WriteString("\n\tcases:")
		for _, f := range n.Cases {
			s.WriteString("\n\t\t")
			s.WriteString(f.Name)
			if f.TypeName != nil {
				buildTypeNameDebugString(x, "\t\t\t", f.TypeName, s)
			}
		}
	case len(n.Virts) > 0:
		s.WriteString("\n\tvirts:")
		for _, v := range n.Virts {
			s.WriteString("\n\t\t")
			s.WriteString(v.Sel)
			if len(v.Parms) > 0 {
				s.WriteString("\n\t\t\tparameters:")
				for i := range v.Parms {
					p := &v.Parms[i]
					if p.Type() == nil {
						continue
					}
					buildTypeSigDebugString(x, "\t\t\t\t", p.Type(), s)
				}
			}
			if v.Ret != nil {
				s.WriteString("\n\t\t\treturn:")
				buildTypeNameDebugString(x, "\t\t\t\t", v.Ret, s)
			}
		}
	}
}

func buildTypeVarDebugString(x *scope, indent string, n *TypeVar, s *strings.Builder) {
	_buildTypeVarDebugString(x, map[*Type]bool{}, indent, n, s)
}

func _buildTypeVarDebugString(x *scope, seen map[*Type]bool, indent string, n *TypeVar, s *strings.Builder) {
	s.WriteString("\n" + indent)
	s.WriteString(fmt.Sprintf("%s (%p)", n.Name, n.Type))
	if len(n.Ifaces) > 0 {
		s.WriteString("\n" + indent + "\tinterfaces")
		for i := range n.Ifaces {
			_buildTypeNameDebugString(x, seen, indent+"\t\t", &n.Ifaces[i], s)
		}
	}
}

func buildTypeSigDebugString(x *scope, indent string, n *Type, s *strings.Builder) {
	_buildTypeSigDebugString(x, map[*Type]bool{}, indent, n, s)
}

func _buildTypeSigDebugString(x *scope, seen map[*Type]bool, indent string, n *Type, s *strings.Builder) {
	if seen[n] {
		s.WriteString("\n" + indent)
		s.WriteString(fmt.Sprintf("(%d)%s (%p) <cycle>", len(n.Parms), n.Name, n))
		return
	}
	seen[n] = true
	defer func() { seen[n] = false }()

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
			_buildTypeVarDebugString(x, seen, indent+"\t\t", &n.Parms[i], s)
		}
	}
	if len(n.Args) > 0 {
		s.WriteString("\n" + indent + "\targuments:")
		for i := range n.Args {
			_buildTypeNameDebugString(x, seen, indent+"\t\t", &n.Args[i], s)
		}
	}
}

func buildTypeNameDebugString(x *scope, indent string, n *TypeName, s *strings.Builder) {
	_buildTypeNameDebugString(x, map[*Type]bool{}, indent, n, s)
}

func _buildTypeNameDebugString(x *scope, seen map[*Type]bool, indent string, n *TypeName, s *strings.Builder) {
	if n.Type == nil {
		s.WriteString("\n" + indent)
		s.WriteString(fmt.Sprintf("(%d)%s — <error>", len(n.Args), n.Name))
		return
	}
	_buildTypeSigDebugString(x, seen, indent, n.Type, s)
}
