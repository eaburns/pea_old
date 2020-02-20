// © 2020 the Pea Authors under the MIT license. See AUTHORS for the list of authors.

package basic

import (
	"fmt"
	"strings"

	"github.com/eaburns/pea/types"
)

func (n *Mod) String() string {
	return n.buildString(&strings.Builder{}, true).String()
}

func (n *Mod) buildString(s *strings.Builder, comments bool) *strings.Builder {
	for _, str := range n.Strings {
		if s.Len() > 0 {
			s.WriteRune('\n')
		}
		str.buildString(s)
	}
	for _, v := range n.Vars {
		if s.Len() > 0 {
			s.WriteRune('\n')
		}
		v.buildString(s)
	}
	for _, fun := range n.Funs {
		if s.Len() > 0 {
			s.WriteRune('\n')
		}
		fun.buildString(s, comments)
	}
	return s
}

func (n *String) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "string%d\n\t%q", n.N, n.Data)
	return s
}

func (n *Var) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "%s %s", n.Val.Var.Name, n.Val.Var.Type())
	return s
}

func (n *Fun) String() string {
	return n.buildString(&strings.Builder{}, true).String()
}

func (n *Fun) buildString(s *strings.Builder, comments bool) *strings.Builder {
	s.WriteString(n.name())
	if comments {
		fmt.Fprintf(s, " // %s\n\tcan inline: %v", n.Fun, n.CanInline)
	}
	s.WriteString("\n\tparms:")
	for _, p := range append(n.Parms, n.Ret) {
		if p == nil {
			continue // no ret
		}
		if p.Var != nil {
			fmt.Fprintf(s, "\n\t\t%d [%s] %s", p.N, p.Var.Name, p.Type)
		} else {
			fmt.Fprintf(s, "\n\t\t%d %s", p.N, p.Type)
		}
		if p.Value {
			s.WriteString(" (value)")
		}
	}
	for _, b := range n.BBlks {
		s.WriteRune('\n')
		b.buildString(s, comments)
	}
	return s
}

func (n *Fun) name() string {
	if n.Block != nil {
		return fmt.Sprintf("block%d", n.N)
	}
	return fmt.Sprintf("function%d", n.N)
}

type commenter interface {
	comment() string
}

func (n *BBlk) buildString(s *strings.Builder, comments bool) *strings.Builder {
	fmt.Fprintf(s, "\t%d:\n\t\t[in:", n.N)
	for _, in := range n.In {
		fmt.Fprintf(s, " %d", in.N)
	}
	s.WriteString("] [out:")
	for _, out := range n.Out() {
		fmt.Fprintf(s, " %d", out.N)
	}
	s.WriteRune(']')
	for i, t := range n.Stmts {
		if _, ok := t.(*Comment); ok {
			if !comments {
				continue
			}
			if i > 0 {
				s.WriteRune('\n')
			}
		}
		if bug := t.bugs(); !t.deleted() && bug != "" {
			s.WriteString("\n\t\t// BUG: ")
			s.WriteString(bug)
		}
		s.WriteString("\n\t\t")
		if t.deleted() {
			s.WriteString("ⓧ ")
		}
		start := s.Len()
		if v, ok := t.(Val); ok {
			fmt.Fprintf(s, "$%d := ", v.Num())
		}
		t.buildString(s)
		if c, ok := t.(commenter); comments && ok {
			const commentCol = 25
			if n := s.Len() - start; n < commentCol {
				s.WriteString(strings.Repeat(" ", commentCol-n))
			}
			fmt.Fprintf(s, " // %s", c.comment())
		}
	}
	return s
}

func (n *Store) comment() string {
	return fmt.Sprintf("*%s = %s", n.Dst.Type(), n.Val.Type())
}

func (n *Copy) comment() string {
	return fmt.Sprintf("*%s = *%s", n.Dst.Type(), n.Src.Type())
}

func (n *val) comment() string {
	if n.Type() == nil {
		return "<nil type>"
	}
	return n.Type().String()
}

func (n *Comment) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "// %s", n.Text)
	return s
}

func (n *Store) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "store($%d, $%d)", n.Dst.Num(), n.Val.Num())
	return s
}

func (n *Copy) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "copy($%d, $%d, %s)",
		n.Dst.Num(), n.Src.Num(), n.Dst.Type().Args[0].Type)
	return s
}

func (n *MakeArray) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "array($%d, {", n.Dst.Num())
	var deref string
	if n.Dst.Type().BuiltIn == types.RefType &&
		n.Dst.Type().Args[0].Type.BuiltIn == types.ArrayType &&
		!SimpleType(n.Dst.Type().Args[0].Type.Args[0].Type) {
		deref = "*"
	}
	for i, arg := range n.Args {
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "%s$%d", deref, arg.Num())
	}
	s.WriteString("})")
	return s
}

func (n *NewArray) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "array($%d, $%d)", n.Dst.Num(), n.Size.Num())
	return s
}

func (n *MakeSlice) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "slice($%d, $%d[$%d:$%d])",
		n.Dst.Num(), n.Ary.Num(), n.From.Num(), n.To.Num())
	return s
}

func (n *MakeString) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "string($%d, string%d)", n.Dst.Num(), n.Data.N)
	return s
}

func (n *NewString) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "string($%d, $%d)", n.Dst.Num(), n.Data.Num())
	return s
}

func (n *MakeAnd) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "and($%d, {", n.Dst.Num())
	andType := n.Dst.Type().Args[0].Type
	for i, arg := range n.Fields {
		if i > 0 {
			s.WriteRune(' ')
		}
		num := "{}"
		if arg != nil {
			num = fmt.Sprintf("$%d", arg.Num())
		}
		if i >= len(andType.Fields) {
			s.WriteString(num)
			continue
		}
		field := andType.Fields[i]
		var deref string
		// field.Type() should never be nil,
		// except in tests when we construct an And-type,
		// we cannot set it's unexported .typ field.
		// For now, we just ignore it to unblock the tests.
		if field.Type() != nil && !SimpleType(field.Type()) {
			deref = "*"
		}
		if field.Name == "" {
			fmt.Fprintf(s, "%s%s", deref, num)
		} else {
			fmt.Fprintf(s, "%s: %s%s", field.Name, deref, num)
		}
	}
	s.WriteString("})")
	return s
}

func (n *MakeOr) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "or($%d, {%d=", n.Dst.Num(), n.Case)
	typ := n.Dst.Type().Args[0].Type
	cas := typ.Cases[n.Case]
	s.WriteString(cas.Name)
	if n.Val != nil {
		var deref string
		// cas.Type() should never be nil,
		// except in tests when we construct an Or-type,
		// we cannot set it's unexported .typ field.
		// For now, we just ignore it to unblock the tests.
		if cas.Type() != nil && !SimpleType(cas.Type()) {
			deref = "*"
		}
		fmt.Fprintf(s, " %s$%d", deref, n.Val.Num())
	}
	s.WriteString("})")
	return s
}

func (n *MakeVirt) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "virt($%d, $%d, {", n.Dst.Num(), n.Obj.Num())
	for i, v := range n.Virts {
		if i > 0 {
			s.WriteString(", ")
		}
		s.WriteString(v.name())
	}
	s.WriteString("})")
	return s
}

func (n *Call) buildString(s *strings.Builder) *strings.Builder {
	s.WriteString("call ")
	s.WriteString(n.Fun.name())
	s.WriteRune('(')
	for i, a := range n.Args {
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "$%d", a.Num())
	}
	s.WriteRune(')')
	return s
}

func (n *VirtCall) buildString(s *strings.Builder) *strings.Builder {
	s.WriteString("virt call ")
	fmt.Fprintf(s, "$%d.%d", n.Self.Num(), n.Index)
	if n.Msg != nil {
		fmt.Fprintf(s, " [%s]", n.Msg.Sel)
	}
	s.WriteRune('(')
	for i, a := range n.Args {
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "$%d", a.Num())
	}
	s.WriteRune(')')
	return s
}

func (n *Ret) buildString(s *strings.Builder) *strings.Builder {
	if n.Far {
		s.WriteString("far ")
	}
	s.WriteString("return")
	return s
}

func (n *Panic) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "panic($%d)", n.Arg.Num())
	return s
}

func (n *Jmp) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "jmp %d", n.Dst.N)
	return s
}

func (n *Switch) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "switch $%d", n.Val.Num())
	for i, dst := range n.Dsts {
		if i >= len(n.OrType.Cases) {
			fmt.Fprintf(s, " [<bad case %s> %d]", n.OrType, dst.N)
			break
		}
		cas := &n.OrType.Cases[i]
		fmt.Fprintf(s, " [%s %d]", cas.Name, dst.N)
	}
	return s
}

func (n *IntLit) buildString(s *strings.Builder) *strings.Builder {
	s.WriteString(n.Val.String())
	if n.Case != nil {
		fmt.Fprintf(s, " [%s]", n.Case.Name)
	}
	return s
}

func (n *FloatLit) buildString(s *strings.Builder) *strings.Builder {
	s.WriteString(n.Val.String())
	return s
}

var opString = map[OpCode]string{
	BitwiseAndOp: "&",
	BitwiseOrOp:  "|",
	BitwiseNotOp: "!",
	RightShiftOp: ">>",
	LeftShiftOp:  "<<",
	NegOp:        "-",
	PlusOp:       "+",
	MinusOp:      "-",
	TimesOp:      "*",
	DivideOp:     "/",
	ModOp:        "%",
	EqOp:         "==",
	NeqOp:        "!=",
	LessOp:       "<",
	LessEqOp:     "<=",
	GreaterOp:    ">",
	GreaterEqOp:  ">=",
	// NumConvertOp
}

func (n *Op) buildString(s *strings.Builder) *strings.Builder {
	switch {
	case n.Code == ArraySizeOp:
		fmt.Fprintf(s, "size($%d)", n.Args[0].Num())
	case n.Code == UnionTagOp:
		fmt.Fprintf(s, "tag($%d)", n.Args[0].Num())
	case n.Code == NumConvertOp:
		fmt.Fprintf(s, "%s($%d)", n.Type(), n.Args[0].Num())

	case len(n.Args) == 1:
		fmt.Fprintf(s, "%s$%d", opString[n.Code], n.Args[0].Num())

	case len(n.Args) == 2:
		fmt.Fprintf(s, "$%d %s $%d", n.Args[0].Num(), opString[n.Code], n.Args[1].Num())
	default:
		panic("impossible")
	}
	return s
}

func (n *Load) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "load($%d)", n.Src.Num())
	return s
}

func (n *Alloc) buildString(s *strings.Builder) *strings.Builder {
	a := ""
	if n.Stack {
		a = "a"
	}
	fmt.Fprintf(s, "alloc%s(%s)", a, n.Type().Args[0].Type)
	return s
}

func (n *Arg) buildString(s *strings.Builder) *strings.Builder {
	if n.Parm.Var != nil {
		fmt.Fprintf(s, "arg(%d [%s])", n.Parm.N, n.Parm.Var.Name)
	} else {
		fmt.Fprintf(s, "arg(%d)", n.Parm.N)
	}
	return s
}

func (n *Global) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "global(%s)", n.Val.Var.Name)
	return s
}

func (n *Index) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "$%d[$%d]", n.Ary.Num(), n.Index.Num())
	return s
}

func (n *Field) buildString(s *strings.Builder) *strings.Builder {
	fmt.Fprintf(s, "$%d.%d", n.Obj.Num(), n.Index)
	switch {
	case n.Field != nil && n.Field.Name != "":
		fmt.Fprintf(s, " [%s]", n.Field.Name)
	case n.Case != nil:
		fmt.Fprintf(s, " [%s]", n.Case.Name)
	}
	return s
}

type stringBuilder interface {
	buildString(*strings.Builder) *strings.Builder
}

func buildString(v stringBuilder) string {
	return v.buildString(&strings.Builder{}).String()
}
