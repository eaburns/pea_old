// Package gengo generates Go code.
package gengo

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"path"
	"strings"

	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/types"
)

type writer struct {
	err error
	w   io.Writer
}

func (ww *writer) writeFmt(f string, vs ...interface{}) {
	if ww.err != nil {
		return
	}
	_, ww.err = fmt.Fprintf(ww.w, f, vs...)
}

func (ww *writer) writeString(s string) {
	if ww.err != nil {
		return
	}
	_, ww.err = io.WriteString(ww.w, s)
}

// WriteMod writes the module as formatted Go code.
func WriteMod(w io.Writer, mod *basic.Mod) error {
	return writeMod(w, mod)
}

func writeMod(w io.Writer, mod *basic.Mod) error {
	var s strings.Builder
	for _, str := range mod.Strings {
		if s.Len() > 0 {
			s.WriteRune('\n')
		}
		WriteString(&s, str)
	}
	for _, v := range mod.Vars {
		if s.Len() > 0 {
			s.WriteRune('\n')
		}
		WriteVar(&s, v)
	}
	for _, f := range mod.Funs {
		if f.BBlks == nil {
			continue
		}
		if s.Len() > 0 {
			s.WriteRune('\n')
		}
		WriteFun(&s, f)
	}

	src := fmt.Sprintf("package %s", mod.Mod.AST.Path)
	src += `
		import "fmt"
		func F1___0_String__main__print_3A__(x *[]byte) {
			fmt.Printf("%v", string(*x))
		}
		func F1___1__26____0_String__main__print_3A__(x *[]byte) {
			fmt.Printf("%v", string(*x))
		}
		func F1___0_Int__main__print_3A__(x int) {
			fmt.Printf("%v", x)
		}
		func F1___1__26____0_Int__main__print_3A__(x *int) {
			fmt.Printf("%v", *x)
		}
		func F1___0_UInt__main__print_3A__(x uint) {
			fmt.Printf("%v", x)
		}
		func F1___0_Float__main__print_3A__(x float64) {
			fmt.Printf("%v", x)
		}
		func F1___0_Bool__main__print_3A__(x uint8) {
			fmt.Printf("%v", x == 1)
		}
	`
	if path.Base(mod.Mod.AST.Path) == "main" {
		src += "func main() {\n"
		for _, f := range mod.Funs {
			if f.Fun == nil {
				src += mangleFun(f, new(strings.Builder)).String() + "()\n"
			}
		}
		src += "F0_main__main__()\n}\n"
	}
	src += s.String()

	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		io.WriteString(w, src)
		panic(err.Error())
	}
	return format.Node(w, fset, root)
}

// WriteString writes the Go code for a string definition.
func WriteString(w io.Writer, f *basic.String) error {
	_, err := fmt.Fprintf(w, "var string%d = []byte(%q)\n", f.N, f.Data)
	return err
}

// WriteVar writes the Go code for a module-level variable definition.
func WriteVar(w io.Writer, f *basic.Var) error {
	ww := &writer{w: w}
	ww.writeFmt("var _%s ", f.Val.Var.Name)
	writeType(ww, f.Val.Var.Type())
	return ww.err
}

// WriteFun writes the Go code for a function definition.
func WriteFun(w io.Writer, f *basic.Fun) error {
	ww := &writer{w: w}
	if f.Fun != nil && f.Block == nil {
		ww.writeFmt("// %s\n", f.Fun)
	}
	ww.writeFmt("func %s(", mangleFun(f, new(strings.Builder)).String())
	writeParms(ww, f)
	ww.writeString(") {\n")
	writeBody(ww, f)
	ww.writeString("}\n")
	return ww.err
}

func writeParms(ww *writer, f *basic.Fun) {
	if len(f.Parms) == 0 && f.Ret == nil {
		return
	}
	for _, parm := range f.Parms {
		ww.writeString("\n\t")
		writeParm(ww, parm)
		ww.writeString(" ")
		writeType(ww, parm.Type)
		ww.writeString(",")
	}
	if f.Ret != nil {
		ww.writeString("\n\t")
		writeParm(ww, f.Ret)
		ww.writeString(" ")
		writeType(ww, f.Ret.Type)
		ww.writeString(",")
	}
	ww.writeString("\n")
}

func writeParm(ww *writer, parm *basic.Parm) {
	if parm.Var != nil {
		ww.writeFmt("p%d_%s", parm.N, parm.Var.Name)
	} else {
		ww.writeFmt("p%d", parm.N)
	}
}

func writeType(ww *writer, typ *types.Type) {
	switch {
	case typ.BuiltIn == types.RefType:
		ww.writeString("*")
		writeType(ww, typ.Args[0].Type)

	case typ.BuiltIn == types.ArrayType:
		ww.writeString("[]")
		writeType(ww, typ.Args[0].Type)

	case len(typ.Virts) > 0:
		writeVirtType(ww, typ)

	case len(typ.Cases) > 0:
		writeOrType(ww, typ)

	case typ.BuiltIn == types.BlockType:
		fallthrough
	case typ.BuiltIn == types.NilType:
		fallthrough
	default:
		writeAndType(ww, typ)

	case typ.BuiltIn != 0:
		if t, ok := builtInTypes[typ.BuiltIn]; ok {
			ww.writeString(t)
			return
		}
		ww.writeFmt("<%s>", typ)
	}
}

var builtInTypes = map[types.BuiltInType]string{
	types.StringType: "[]byte",
	types.BoolType:   "uint8",
	// TODO: handle different size Int.
	types.IntType:   "int",
	types.Int8Type:  "int8",
	types.Int16Type: "int16",
	types.Int32Type: "int32",
	types.Int64Type: "int64",
	// TODO: handle different size UInt.
	types.UIntType:   "uint",
	types.UInt8Type:  "uint8",
	types.UInt16Type: "uint16",
	types.UInt32Type: "uint32",
	types.UInt64Type: "uint64",
	// TODO: handle different size Float.
	types.FloatType:   "float64",
	types.Float32Type: "float32",
	types.Float64Type: "float64",
}

func writeAndType(ww *writer, typ *types.Type) {
	ww.writeString("struct{")
	for i := range typ.Fields {
		field := &typ.Fields[i]
		if basic.EmptyType(field.Type()) {
			continue
		}
		ww.writeString(fieldName(typ, i))
		ww.writeString(" ")
		writeType(ww, field.Type())
		ww.writeString("; ")
	}
	ww.writeString("}")
}

func writeOrType(ww *writer, typ *types.Type) {
	if basic.SimpleType(typ) {
		// This case is an Or type converted to an int.
		writeType(ww, typ.Tag())
		return
	}

	ww.writeString("struct{tag ")
	writeType(ww, typ.Tag())
	ww.writeString("; ")
	for i := range typ.Cases {
		cas := &typ.Cases[i]
		if cas.Type() == nil || basic.EmptyType(cas.Type()) {
			continue
		}
		ww.writeString(caseName(typ, i))
		ww.writeString(" ")
		writeType(ww, cas.Type())
		ww.writeString("; ")
	}
	ww.writeString("}")
}

func writeVirtType(ww *writer, typ *types.Type) {
	ww.writeString("struct{")
	for i := range typ.Virts {
		ww.writeString(virtName(typ, i))
		ww.writeString(" ")
		writeVirtSig(ww, &typ.Virts[i])
		ww.writeString("; ")
	}
	ww.writeString("}")
}

func writeVirtSig(ww *writer, virt *types.FunSig) int {
	ww.writeString("func(")
	var i int
	for _, parm := range virt.Parms {
		if basic.EmptyType(parm.Type()) {
			continue
		}
		if i > 0 {
			ww.writeString(", ")
		}
		ww.writeFmt("p%d ", i)
		i++
		if !basic.SimpleType(parm.Type()) {
			ww.writeString("*")
		}
		writeType(ww, parm.Type())
	}
	if virt.Ret != nil && !basic.EmptyType(virt.Ret.Type) {
		if i > 0 {
			ww.writeString(", ")
		}
		ww.writeFmt("p%d *", i)
		writeType(ww, virt.Ret.Type)
		i++
	}
	ww.writeString(")")
	return i
}

func writeBody(ww *writer, f *basic.Fun) {
	for _, b := range f.BBlks {
		for _, s := range b.Stmts {
			v, ok := s.(basic.Val)
			if !ok {
				continue
			}
			ww.writeFmt("var x%d ", v.Num())
			writeType(ww, v.Type())
			ww.writeString("\n")
		}
	}
	for i, b := range f.BBlks {
		if i > 0 {
			ww.writeFmt("L%d:\n", b.N)
		}
		for _, s := range b.Stmts {
			writeStmt(ww, s)
		}
	}
}

func writeStmt(ww *writer, s basic.Stmt) {
	ww.writeString("\t")
	switch s := s.(type) {
	case *basic.Comment:
		ww.writeFmt("// %s", s.Text)
	case *basic.Store:
		ww.writeFmt("*x%d = x%d", s.Dst.Num(), s.Val.Num())
	case *basic.Copy:
		ww.writeFmt("*x%d = *x%d", s.Dst.Num(), s.Src.Num())
	case *basic.MakeArray:
		writeMakeArray(ww, s)
	case *basic.MakeSlice:
		writeMakeSlice(ww, s)
	case *basic.MakeString:
		writeMakeString(ww, s)
	case *basic.MakeAnd:
		writeMakeAnd(ww, s)
	case *basic.MakeOr:
		writeMakeOr(ww, s)
	case *basic.MakeVirt:
		writeMakeVirt(ww, s)
	case *basic.Call:
		writeCall(ww, s)
	case *basic.VirtCall:
		writeVirtCall(ww, s)
	case *basic.Ret:
		ww.writeFmt("return")
	case *basic.Jmp:
		ww.writeFmt("goto L%d", s.Dst.N)
	case *basic.Switch:
		writeSwitch(ww, s)
	case basic.Val:
		writeVal(ww, s)
	default:
		panic(fmt.Sprintf("impossible type %T", s))
	}
	ww.writeString("\n")
}

func writeMakeArray(ww *writer, s *basic.MakeArray) {
	typ := s.Dst.Type().Args[0].Type
	elmType := typ.Args[0].Type

	ww.writeFmt("*x%d = ", s.Dst.Num())
	writeType(ww, typ)
	ww.writeString("{")
	var deref string
	if !basic.SimpleType(elmType) {
		deref = "*"
	}
	for i, arg := range s.Args {
		if i > 0 {
			ww.writeString(", ")
		}
		ww.writeFmt("%sx%d", deref, arg.Num())
	}
	ww.writeString("}")
}

func writeMakeSlice(ww *writer, s *basic.MakeSlice) {
	ww.writeFmt("*x%d = (*x%d)[x%d:x%d]",
		s.Dst.Num(), s.Ary.Num(), s.From.Num(), s.To.Num())
}

func writeMakeString(ww *writer, s *basic.MakeString) {
	ww.writeFmt("*x%d = string%d[:]", s.Dst.Num(), s.Data.N)
}

func writeMakeAnd(ww *writer, s *basic.MakeAnd) {
	ww.writeFmt("*x%d = ", s.Dst.Num())
	typ := s.Dst.Type().Args[0].Type
	writeType(ww, typ)
	ww.writeString("{")

	for i, val := range s.Fields {
		if val == nil {
			continue
		}
		field := &typ.Fields[i]
		var deref string
		if !basic.SimpleType(field.Type()) {
			deref = "*"
		}
		ww.writeFmt("%s: %sx%d, ", fieldName(typ, i), deref, val.Num())
	}
	ww.writeString("}")
}

func writeMakeOr(ww *writer, s *basic.MakeOr) {
	ww.writeFmt("*x%d = ", s.Dst.Num())
	typ := s.Dst.Type().Args[0].Type
	writeType(ww, typ)
	ww.writeFmt("{tag: %d, ", s.Case)
	if s.Val != nil {
		cas := &typ.Cases[s.Case]
		var deref string
		if !basic.SimpleType(cas.Type()) {
			deref = "*"
		}
		ww.writeFmt("%s: %sx%d, ", caseName(typ, s.Case), deref, s.Val.Num())
	}
	ww.writeString("}")
}

func writeMakeVirt(ww *writer, s *basic.MakeVirt) {
	ww.writeFmt("*x%d = ", s.Dst.Num())
	typ := s.Dst.Type().Args[0].Type
	writeType(ww, typ)
	ww.writeString("{")
	for i, v := range s.Virts {
		ww.writeFmt("%s: ", virtName(typ, i))
		n := writeVirtSig(ww, &typ.Virts[i])
		ww.writeFmt("{%s(x%d", mangleFun(v, new(strings.Builder)).String(), s.Obj.Num())
		for i := 0; i < n; i++ {
			ww.writeFmt(", p%d", i)
		}
		ww.writeString(")}, ")
	}
	ww.writeString("}")
}

func writeCall(ww *writer, s *basic.Call) {
	ww.writeFmt("%s(", mangleFun(s.Fun, new(strings.Builder)).String())
	for i, arg := range s.Args {
		if i > 0 {
			ww.writeString(", ")
		}
		ww.writeFmt("x%d", arg.Num())
	}
	ww.writeString(")")
}

func writeVirtCall(ww *writer, s *basic.VirtCall) {
	typ := s.Self.Type().Args[0].Type
	ww.writeFmt("x%d.%s(", s.Self.Num(), virtName(typ, s.Index))
	// Strip off the self argument.
	// Go code gen handles that as a closure
	// at the time the Virt is created.
	for i, arg := range s.Args[1:] {
		if i > 0 {
			ww.writeString(", ")
		}
		ww.writeFmt("x%d", arg.Num())
	}
	ww.writeString(")")
}

func writeSwitch(ww *writer, s *basic.Switch) {
	ww.writeFmt("switch x%d {", s.Val.Num())
	if s.Val.Type().BuiltIn == types.BoolType {
		// TODO: remove the hack to reverse bool 0/1.
		for i, b := range s.Dsts {
			ww.writeFmt("case %d: goto L%d; ", 1-i, b.N)
		}
	} else {
		for i, b := range s.Dsts {
			ww.writeFmt("case %d: goto L%d; ", i, b.N)
		}
	}
	ww.writeString("}")
}

func writeVal(ww *writer, v basic.Val) {
	ww.writeFmt("x%d = ", v.Num())
	switch v := v.(type) {
	case *basic.IntLit:
		t := builtInTypes[v.Type().BuiltIn]
		ww.writeFmt("%s(%s)", t, v.Val.String())
	case *basic.FloatLit:
		t := builtInTypes[v.Type().BuiltIn]
		ww.writeFmt("%s(%s)", t, v.Val.String())
	case *basic.Op:
		writeOp(ww, v)
	case *basic.Load:
		ww.writeFmt("*x%d", v.Src.Num())
	case *basic.Alloc:
		ww.writeString("new(")
		writeType(ww, v.Type().Args[0].Type)
		ww.writeString(")")
	case *basic.Arg:
		writeParm(ww, v.Parm)
	case *basic.Global:
		ww.writeFmt("&_%s", v.Val.Var.Name)
	case *basic.Index:
		ww.writeFmt("&(*x%d)[x%d]", v.Ary.Num(), v.Index.Num())
	case *basic.Field:
		n := v.Obj.Num()
		i := v.Index
		typ := v.Obj.Type().Args[0].Type
		if len(typ.Cases) > 0 {
			ww.writeFmt("&x%d.%s", n, caseName(typ, i))
		} else {
			ww.writeFmt("&x%d.%s", n, fieldName(typ, i))
		}
	default:
		panic(fmt.Sprintf("impossible type %T", v))
	}
}

var numOps = map[basic.OpCode]string{
	basic.BitwiseAndOp: "&",
	basic.BitwiseOrOp:  "|",
	basic.BitwiseNotOp: "^",
	basic.RightShiftOp: ">>",
	basic.LeftShiftOp:  "<<",
	basic.NegOp:        "-",
	basic.PlusOp:       "+",
	basic.MinusOp:      "-",
	basic.TimesOp:      "*",
	basic.DivideOp:     "/",
	basic.ModOp:        "%",
}

var cmpOps = map[basic.OpCode]string{
	basic.EqOp:        "==",
	basic.NeqOp:       "!=",
	basic.LessOp:      "<",
	basic.LessEqOp:    "<=",
	basic.GreaterOp:   ">",
	basic.GreaterEqOp: ">=",
}

func writeOp(ww *writer, op *basic.Op) {
	switch {
	case op.Code == basic.ArraySizeOp:
		ww.writeFmt("len(*x%d)", op.Args[0].Num())

	case op.Code == basic.UnionTagOp:
		ww.writeFmt("(*x%d).tag", op.Args[0].Num())

	case op.Code == basic.NumConvertOp:
		ww.writeFmt("%s(x%d)",
			builtInTypes[op.Type().BuiltIn], op.Args[0].Num())

	case numOps[op.Code] != "":
		c := numOps[op.Code]
		l := op.Args[0].Num()
		if len(op.Args) == 1 {
			ww.writeFmt("%s x%d", c, l)
			break
		}
		r := op.Args[1].Num()
		ww.writeFmt("x%d %s x%d", l, c, r)

	case cmpOps[op.Code] != "":
		c := cmpOps[op.Code]
		l := op.Args[0].Num()
		r := op.Args[1].Num()
		ww.writeFmt("0; if x%d %s x%d { x%d = 1 }", l, c, r, op.Num())

	default:
		panic("impossible")
	}
}
