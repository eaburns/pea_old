package gengo

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/types"
)

func stringName(str *basic.String) string {
	var s strings.Builder
	mangleMod(str.Mod.Mod.Path, &s)
	fmt.Fprintf(&s, "string%d", str.N)
	return s.String()
}

func valName(v *types.Val) string {
	var s strings.Builder
	mangleMod(v.ModPath, &s)
	writeStr(v.Var.Name, &s)
	return s.String()
}

func fieldName(typ *types.Type, i int) string {
	if i < 0 || i >= len(typ.Fields) || typ.Fields[i].Name == "" {
		return fmt.Sprintf("_field%d", i)
	}
	return escape(typ.Fields[i].Name, new(strings.Builder)).String()
}

func virtName(typ *types.Type, i int) string {
	return escape(typ.Virts[i].Sel, new(strings.Builder)).String()
}

func caseName(typ *types.Type, i int) string {
	return escape(typ.Cases[i].Name, new(strings.Builder)).String()
}

func mangleFun(f *basic.Fun, s *strings.Builder) *strings.Builder {
	switch {
	case f.Block != nil:
		if f.Fun != nil {
			mangleMod(f.Fun.ModPath, s)
			mangleMod(f.Fun.InstModPath, s)
		} else {
			mangleMod(f.Val.ModPath, s)
		}
		writeStr(f.Block.BlockType.Name, s)
	case f.Val != nil:
		fmt.Fprintf(s, "init__%s", valName(f.Val))
	case f.Fun != nil:
		mangleTypesFun(f.Mod.Mod.Path, f.Fun, s)
	default:
		mangleMod(f.Mod.Mod.Path, s)
		s.WriteString("init")
	}
	return s
}

// mangleTypesFun mangles the name of a types.Fun to the end of a strings.Builder.
// modPath is the mod path defining this fun instance,
// which may differ from fun.ModPath,
// which is the mod path of the Fun definition.
func mangleTypesFun(modPath string, fun *types.Fun, s *strings.Builder) *strings.Builder {
	typeConstraint := false
	switch {
	case fun.Test:
		s.WriteRune('T')
	case fun.Recv == nil:
		s.WriteRune('F')
	default:
		s.WriteRune('M')
		mangleType(fun.Recv.Type, s)
		for _, p := range fun.Recv.Parms {
			if len(p.Ifaces) > 0 {
				typeConstraint = true
				break
			}
		}
	}
	writeInt(len(fun.TArgs), s)
	for _, arg := range fun.TArgs {
		mangleType(arg.Type, s)
	}

	mangleMod(fun.ModPath, s)
	writeStr(fun.Sig.Sel, s)

	if !typeConstraint {
		for _, p := range fun.TParms {
			if len(p.Ifaces) > 0 {
				typeConstraint = true
				break
			}
		}
	}
	if typeConstraint {
		// If there is a type constraint on any of the type parameters,
		// this function instance must be unique to its defining file.
		//
		// TODO: relax the unique Fun inst constraint
		// by adding method sets to the mangled name.
		for i, f := range fun.Def.Insts {
			if f == fun {
				writeInt(i, s)
				mangleMod(fun.InstModPath, s)
				return s
			}
		}
		panic("impossible")
	}
	return s
}

func demangleTestName(s string) (mod, name string, err error) {
	rr := strings.NewReader(s)
	switch r, _, err := rr.ReadRune(); {
	case err == io.EOF:
		return "", "", errors.New("unexpected EOF")
	case err != nil:
		return "", "", err
	case r != 'T':
		return "", "", errors.New("expected 'T'")
	}
	switch nargs, err := readInt(rr); {
	case err != nil:
		return "", "", err
	case nargs != 0:
		return "", "", errors.New("expected 0 args")
	}
	if mod, err = demangleMod(rr); err != nil {
		return "", "", err
	}
	if name, err = readStr(rr); err != nil {
		return "", "", err
	}
	return mod, name, err
}

func demangleFun(rr io.RuneReader) (string, error) {
	var out strings.Builder
	switch r, _, err := rr.ReadRune(); {
	case err == io.EOF:
		return "", errors.New("unexpected EOF")
	case err != nil:
		return "", err
	case r == 'F':
		out.WriteString("Func ")
	case r == 'M':
		out.WriteString("Meth ")
		recvType, err := demangleType(rr)
		if err != nil {
			return "", err
		}
		out.WriteString(recvType)
		out.WriteRune(' ')
	case r == 'T':
		out.WriteString("test ")
	default:
		return "", fmt.Errorf("expected F or M, got %c", r)
	}

	switch nargs, err := readInt(rr); {
	case err != nil:
		return "", err
	case nargs == 1:
		arg, err := demangleType(rr)
		if err != nil {
			return "", err
		}
		out.WriteString(arg)
		out.WriteRune(' ')
	case nargs > 1:
		out.WriteRune('(')
		for i := 0; i < nargs; i++ {
			arg, err := demangleType(rr)
			if err != nil {
				return "", err
			}
			if i > 0 {
				out.WriteString(", ")
			}
			out.WriteString(arg)
		}
		out.WriteString(") ")
	}

	switch modPath, err := demangleMod(rr); {
	case err != nil:
		return "", err
	case modPath != "":
		out.WriteString(modPath)
		out.WriteRune(' ')
	}

	sel, err := readStr(rr)
	if err != nil {
		return "", err
	}
	out.WriteString(sel)

	switch n, err := readIntOrEOF(rr); {
	case err == io.EOF:
		break
	case err != nil:
		return "", err
	default:
		modPath, err := demangleMod(rr)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&out, " %d %s", n, modPath)
	}
	return out.String(), nil
}

func mangleType(typ *types.Type, s *strings.Builder) *strings.Builder {
	mangleMod(typ.ModPath, s)
	writeInt(typ.Arity, s)
	writeStr(typ.Name, s)
	for _, arg := range typ.Args {
		mangleType(arg.Type, s)
	}
	return s
}

func demangleType(rr io.RuneReader) (string, error) {
	modPath, err := demangleMod(rr)
	if err != nil {
		return "", err
	}
	arity, err := readInt(rr)
	if err != nil {
		return "", err
	}
	name, err := readStr(rr)
	if err != nil {
		return "", err
	}
	var args []string
	for i := 0; i < arity; i++ {
		arg, err := demangleType(rr)
		if err != nil {
			return "", err
		}
		args = append(args, arg)
	}
	var s strings.Builder

	switch {
	case len(args) == 1:
		s.WriteString(args[0])
		s.WriteRune(' ')
	case len(args) > 1:
		s.WriteRune('(')
		for i, arg := range args {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(arg)
		}
		s.WriteString(") ")
	}
	s.WriteString(modPath)
	if len(modPath) > 0 {
		s.WriteRune(' ')
	}
	s.WriteString(name)
	return s.String(), nil
}

func mangleMod(modPath string, s *strings.Builder) *strings.Builder {
	return writeStr(modPath, s)
}

func demangleMod(rr io.RuneReader) (string, error) {
	return readStr(rr)
}

func writeStr(str string, s *strings.Builder) *strings.Builder {
	escape(str, s)
	s.WriteString("__")
	return s
}

func readStr(rr io.RuneReader) (string, error) {
	var esc bool
	var out strings.Builder
	for {
		switch r, _, err := rr.ReadRune(); {
		case err == io.EOF:
			return "", errors.New("unexpected EOF")
		case err != nil:
			return "", err
		case esc:
			if r == '_' {
				return unescape(out.String())
			}
			out.WriteRune('_')
			out.WriteRune(r)
			esc = false
		case r == '_':
			esc = true
		default:
			out.WriteRune(r)
		}
	}
}

func writeInt(n int, s *strings.Builder) {
	fmt.Fprintf(s, "%d_", n)
}

func readInt(rr io.RuneReader) (int, error) {
	n, err := readIntOrEOF(rr)
	if err == io.EOF {
		return 0, errors.New("unexpected EOF")
	}
	return n, err
}

func readIntOrEOF(rr io.RuneReader) (int, error) {
	var n int
	for {
		switch r, _, err := rr.ReadRune(); {
		case err != nil:
			return 0, err
		case r == '_':
			return n, nil
		case r < '0' || r > '9':
			return 0, fmt.Errorf("bad digit: %c", r)
		default:
			if n > 4096 {
				return 0, fmt.Errorf("number too big")
			}
			n *= 10
			n += int(r - '0')
		}
	}
}
