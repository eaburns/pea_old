package gengo

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/types"
)

const sep = '_'

// mangleFun returns a Go identifier
// that uniquely names a function
// within a module.
func mangleFun(f *basic.Fun) string {
	switch {
	case f.Block != nil:
		return fmt.Sprintf("block%d", f.N)
	case f.Val != nil:
		return fmt.Sprintf("init%s", f.Val.Var.Name)
	case f.Fun == nil:
		return "init"
	}
	var s strings.Builder
	if f.Fun.Recv != nil {
		if isOp(f.Fun.Sig.Sel) {
			s.WriteString("Op")
		} else {
			s.WriteString("Meth")
		}
		writeInstNum(&s, f)
		s.WriteRune(sep)
		mangleTypeName(&s, f.Fun.Recv.Type)
	} else {
		s.WriteString("Func")
		writeInstNum(&s, f)
	}
	if n := len(f.Fun.TArgs); n > 0 {
		s.WriteRune(sep)
		s.WriteString(strconv.Itoa(n))
		for _, arg := range f.Fun.TArgs {
			s.WriteRune(sep)
			mangleTypeName(&s, arg.Type)
		}
	}
	s.WriteRune(sep)
	mangleSelector(&s, &f.Fun.Sig)
	return s.String()
}

func writeInstNum(s *strings.Builder, f *basic.Fun) {
	if f.BBlks == nil {
		// Don't write an instance number if calling a declared function
		// for which we don't have a definition.
		// There won't be different implementations of this
		// for each calling file.
		// The type arguments are enough to distinguish the instances.
		return
	}
	for n, inst := range f.Fun.Def.Insts {
		if inst == f.Fun {
			if n == 0 {
				return
			}
			s.WriteString(strconv.Itoa(n))
			return
		}
	}
	panic(fmt.Sprintf("no inst for %s", f.Fun))
}

func mangleTypeName(s *strings.Builder, t *types.Type) {
	if n := len(t.Args); n > 0 {
		s.WriteString(strconv.Itoa(n))
	}
	if isOp(t.Name) {
		s.WriteRune(sep)
		mangleOp(s, t.Name)
	} else {
		s.WriteString(t.Name)
	}
	for _, arg := range t.Args {
		s.WriteRune(sep)
		mangleTypeName(s, arg.Type)
	}
}

func mangleSelector(s *strings.Builder, sig *types.FunSig) {
	if isOp(sig.Sel) {
		mangleOp(s, sig.Sel)
		return
	}
	s.WriteString(strings.Replace(sig.Sel, ":", "_", -1))
}

func mangleOp(s *strings.Builder, op string) {
	for i, r := range op {
		if i > 0 {
			s.WriteRune(sep)
		}
		n, ok := opNames[r]
		if !ok {
			panic("impossible")
		}
		s.WriteString(n)
	}
}

var opNames = map[rune]string{
	'!':  "exclamation",
	'%':  "percent",
	'&':  "and",
	'*':  "star",
	'+':  "plus",
	'-':  "minus",
	'/':  "slash",
	'<':  "less",
	'=':  "equal",
	'>':  "greater",
	'?':  "question",
	'@':  "at",
	'\\': "backslash",
	'|':  "or",
	'~':  "tilde",
}

func isOp(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return !unicode.IsLetter(r)
}

func fieldName(typ *types.Type, i int) string {
	if i < 0 || i >= len(typ.Fields) || typ.Fields[i].Name == "" {
		return fmt.Sprintf("_field%d", i)
	}
	return typ.Fields[i].Name
}

func virtName(typ *types.Type, i int) string {
	var s strings.Builder
	s.WriteRune('_')
	mangleSelector(&s, &typ.Virts[i])
	return s.String()
}

func caseName(typ *types.Type, i int) string {
	return "_" + strings.Replace(typ.Cases[i].Name, ":", "_", -1)
}
