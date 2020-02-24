// Copyright © 2020 The Pea Authors under an MIT-style license.

// Package gengo generates Go code.
package gengo

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/eaburns/pea/basic"
	"github.com/eaburns/pea/types"
)

type typeSet map[*types.Type]bool

var builtInTypes = map[types.BuiltInType]string{
	types.NilType:    "struct{}",
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

var numOps = map[basic.OpCode]string{
	basic.BitwiseAndOp: "&",
	basic.BitwiseOrOp:  "|",
	basic.BitwiseXOrOp: "^",
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

// WriteMod writes the generated Go definitions of a module.
// The output has a section for each definition.
// A section begins with single line (\n delimited) header is:
// the number of bytes, N, following the header
// a single space (\20),
// the globally unique name of the defintion.
// The following N bytes are the Go source of the definition,
// always ending in a newline.
func WriteMod(w io.Writer, mod *basic.Mod) error {
	ts := make(typeSet)
	for _, str := range mod.Strings {
		var s strings.Builder
		genStringDef(str, &s)
		_, err := fmt.Fprintf(w, "%d %s\n%s", s.Len(), stringName(str), s.String())
		if err != nil {
			return err
		}
	}
	for _, v := range mod.Vars {
		var s strings.Builder
		genVarDef(v, ts, &s)
		_, err := fmt.Fprintf(w, "%d %s\n%s", s.Len(), valName(v.Val), s.String())
		if err != nil {
			return err
		}
	}
	for _, f := range mod.Funs {
		if f.BBlks == nil {
			continue
		}
		var s strings.Builder
		genFunDef(f, ts, &s)
		name := mangleFun(f, new(strings.Builder)).String()
		_, err := fmt.Fprintf(w, "%d %s\n%s", s.Len(), name, s.String())
		if err != nil {
			return err
		}
	}
	done := make(typeSet)
	for {
		var sorted []*types.Type
		for typ := range ts {
			if done[typ] {
				continue
			}
			sorted = append(sorted, typ)
		}
		if len(sorted) == 0 {
			break
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].String() < sorted[j].String()
		})
		ts = make(typeSet)
		for _, typ := range sorted {
			done[typ] = true
			var s strings.Builder
			genTypeDef(typ, ts, &s)
			name := mangleType(typ, new(strings.Builder))
			_, err := fmt.Fprintf(w, "%d %s\n%s", s.Len(), name, s.String())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func genStringDef(str *basic.String, s *strings.Builder) {
	fmt.Fprintf(s, "var %s = []byte(%q)\n", stringName(str), str.Data)
}

func genVarDef(v *basic.Var, ts typeSet, s *strings.Builder) {
	fmt.Fprintf(s, "var %s ", valName(v.Val))
	genTypeName(v.Val.Var.Type(), ts, s)
	s.WriteRune('\n')
}

func genTypeName(typ *types.Type, ts typeSet, s *strings.Builder) {
	switch {
	case typ.BuiltIn == types.RefType:
		s.WriteString("*")
		genTypeName(typ.Args[0].Type, ts, s)
	case typ.BuiltIn == types.ArrayType:
		s.WriteString("[]")
		genTypeName(typ.Args[0].Type, ts, s)
	case len(typ.Cases) > 0 && basic.SimpleType(typ):
		// This is actually an or-type reduced to an integer.
		s.WriteString(builtInTypes[typ.Tag().BuiltIn])
	case typ.BuiltIn == types.BlockType ||
		typ.BuiltIn == types.FunType ||
		typ.BuiltIn == 0:
		ts[typ] = true
		mangleType(typ, s)
	case builtInTypes[typ.BuiltIn] != "":
		s.WriteString(builtInTypes[typ.BuiltIn])
	default:
		panic(fmt.Sprintf("impossible type %s", typ))
	}
}

func genTypeDef(typ *types.Type, ts typeSet, s *strings.Builder) {
	switch {
	case len(typ.Virts) > 0:
		genVirtTypeDef(typ, ts, s)
	case len(typ.Cases) > 0:
		genOrTypeDef(typ, ts, s)
	case typ.BuiltIn == types.BlockType ||
		typ.BuiltIn == 0:
		genAndTypeDef(typ, ts, s)
	default:
		// This is a built-in type that uses built-in Go types;
		// we should never emit a definition for it.
		panic("impossible")
	}
}

func genAndTypeDef(typ *types.Type, ts typeSet, s *strings.Builder) {
	s.WriteString("type ")
	mangleType(typ, s)
	s.WriteString(" struct{")
	for i := range typ.Fields {
		field := &typ.Fields[i]
		if basic.EmptyType(field.Type()) {
			continue
		}
		s.WriteString("\n\t")
		s.WriteString(fieldName(typ, i))
		s.WriteRune(' ')
		genTypeName(field.Type(), ts, s)
	}
	if typ.BuiltIn == types.BlockType {
		// Add a field to store the far-return token
		// used to match a possible far return with
		// the function that must catch it.
		s.WriteString("\ntoken retToken\n")
	}
	s.WriteString("\n}\n")
}

func genOrTypeDef(typ *types.Type, ts typeSet, s *strings.Builder) {
	if basic.SimpleType(typ) {
		// This case is an Or type converted to an int.
		// We never generate a definition in this case,
		// becaues it just uses a built-in Go type.
		panic("impossible")
	}

	s.WriteString("type ")
	mangleType(typ, s)
	s.WriteString(" struct{\n\ttag ")
	genTypeName(typ.Tag(), ts, s)
	for i := range typ.Cases {
		cas := &typ.Cases[i]
		if cas.Type() == nil || basic.EmptyType(cas.Type()) {
			continue
		}
		s.WriteString("\n\t")
		s.WriteString(caseName(typ, i))
		s.WriteRune(' ')
		genTypeName(cas.Type(), ts, s)
	}
	s.WriteString("\n}\n")
}

func genVirtTypeDef(typ *types.Type, ts typeSet, s *strings.Builder) {
	s.WriteString("type ")
	mangleType(typ, s)
	s.WriteString(" struct{")
	for i := range typ.Virts {
		s.WriteString("\n\t")
		s.WriteString(virtName(typ, i))
		s.WriteRune(' ')
		genVirtSig(&typ.Virts[i], ts, s)
	}
	s.WriteString("\n}\n")
}

func genVirtSig(virt *types.FunSig, ts typeSet, s *strings.Builder) int {
	var i int
	s.WriteString("func(")
	for _, parm := range virt.Parms {
		if basic.EmptyType(parm.Type()) {
			continue
		}
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "p%d ", i)
		i++
		if !basic.SimpleType(parm.Type()) {
			s.WriteRune('*')
		}
		genTypeName(parm.Type(), ts, s)
	}
	if virt.Ret != nil && !basic.EmptyType(virt.Ret.Type) {
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "p%d *", i)
		i++
		genTypeName(virt.Ret.Type, ts, s)
	}
	s.WriteRune(')')
	return i
}

const catchFarRet = `	token := nextToken()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		tok, ok := r.(retToken)
		if !ok || tok != token {
			panic(r)
		}
	}()
`

func genFunDef(f *basic.Fun, ts typeSet, s *strings.Builder) {
	if f.Fun != nil && f.Block == nil {
		fmt.Fprintf(s, "// %s\n", f.Fun)
	}
	s.WriteString("func ")
	mangleFun(f, s)
	s.WriteRune('(')
	genFunParms(f, ts, s)
	s.WriteString(") {\n")
	switch {
	case f.CanFarRet:
		s.WriteString(catchFarRet)
	case f.Block != nil:
		s.WriteString("token := p0.token\n\tuse(token)\n")
	default:
		s.WriteString("var token retToken\n\tuse(token)\n")
	}
	genFunBody(f, ts, s)
	s.WriteString("}\n")
}

func genFunParms(f *basic.Fun, ts typeSet, s *strings.Builder) {
	if len(f.Parms) == 0 && f.Ret == nil {
		return
	}
	var i int
	for _, parm := range f.Parms {
		fmt.Fprintf(s, "\n\tp%d ", i)
		i++
		s.WriteRune(' ')
		genTypeName(parm.Type, ts, s)
		s.WriteRune(',')
	}
	if f.Ret != nil {
		fmt.Fprintf(s, "\n\tp%d ", i)
		genTypeName(f.Ret.Type, ts, s)
		s.WriteRune(',')
	}
	s.WriteRune('\n')
}

func genFunBody(f *basic.Fun, ts typeSet, s *strings.Builder) {
	for _, b := range f.BBlks {
		for _, stmt := range b.Stmts {
			v, ok := stmt.(basic.Val)
			if !ok {
				continue
			}
			fmt.Fprintf(s, "\tvar x%d ", v.Num())
			genTypeName(v.Type(), ts, s)
			s.WriteRune('\n')
		}
	}
	for i, b := range f.BBlks {
		if i > 0 {
			fmt.Fprintf(s, "L%d:\n", b.N)
		}
		for _, stmt := range b.Stmts {
			genStmt(f, stmt, ts, s)
		}
	}
}

func genStmt(f *basic.Fun, stmt basic.Stmt, ts typeSet, s *strings.Builder) {
	s.WriteRune('\t')
	switch stmt := stmt.(type) {
	case *basic.Comment:
		fmt.Fprintf(s, "// %s", stmt.Text)
	case *basic.Store:
		fmt.Fprintf(s, "*x%d = x%d", stmt.Dst.Num(), stmt.Val.Num())
	case *basic.Copy:
		fmt.Fprintf(s, "*x%d = *x%d", stmt.Dst.Num(), stmt.Src.Num())
	case *basic.MakeArray:
		genMakeArray(stmt, ts, s)
	case *basic.NewArray:
		genNewArray(stmt, ts, s)
	case *basic.MakeSlice:
		genMakeSlice(stmt, s)
	case *basic.MakeString:
		genMakeString(stmt, s)
	case *basic.NewString:
		genNewString(stmt, s)
	case *basic.MakeAnd:
		genMakeAnd(stmt, ts, s)
	case *basic.MakeOr:
		genMakeOr(stmt, ts, s)
	case *basic.MakeVirt:
		genMakeVirt(stmt, ts, s)
	case *basic.Panic:
		genPanic(f, stmt, s)
	case *basic.Call:
		genCall(f, stmt, s)
	case *basic.VirtCall:
		genVirtCall(stmt, s)
	case *basic.Ret:
		genRet(stmt, s)
	case *basic.Jmp:
		fmt.Fprintf(s, "goto L%d", stmt.Dst.N)
	case *basic.Switch:
		genSwitch(stmt, s)
	case basic.Val:
		genVal(stmt, ts, s)
	default:
		panic(fmt.Sprintf("impossible type %T", stmt))
	}
	s.WriteRune('\n')
}

func genMakeArray(stmt *basic.MakeArray, ts typeSet, s *strings.Builder) {
	typ := stmt.Dst.Type().Args[0].Type
	elmType := typ.Args[0].Type

	fmt.Fprintf(s, "*x%d = ", stmt.Dst.Num())
	genTypeName(typ, ts, s)
	s.WriteRune('{')
	var deref string
	if !basic.SimpleType(elmType) {
		deref = "*"
	}
	for i, arg := range stmt.Args {
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "%sx%d", deref, arg.Num())
	}
	s.WriteRune('}')
}

func genNewArray(stmt *basic.NewArray, ts typeSet, s *strings.Builder) {
	fmt.Fprintf(s, "*x%d = make(", stmt.Dst.Num())
	genTypeName(stmt.Dst.Type().Args[0].Type, ts, s)
	fmt.Fprintf(s, ", x%d)", stmt.Size.Num())
}

func genMakeSlice(stmt *basic.MakeSlice, s *strings.Builder) {
	fmt.Fprintf(s, "*x%d = (*x%d)[x%d:x%d]",
		stmt.Dst.Num(), stmt.Ary.Num(), stmt.From.Num(), stmt.To.Num())
}

func genMakeString(stmt *basic.MakeString, s *strings.Builder) {
	fmt.Fprintf(s, "*x%d = %s[:]", stmt.Dst.Num(), stringName(stmt.Data))
}

func genNewString(stmt *basic.NewString, s *strings.Builder) {
	fmt.Fprintf(s, "*x%d = append([]byte{}, *x%d...)", stmt.Dst.Num(), stmt.Data.Num())
}

func genMakeAnd(stmt *basic.MakeAnd, ts typeSet, s *strings.Builder) {
	fmt.Fprintf(s, "*x%d = ", stmt.Dst.Num())
	typ := stmt.Dst.Type().Args[0].Type
	genTypeName(typ, ts, s)
	s.WriteRune('{')
	for i, val := range stmt.Fields {
		if val == nil {
			continue
		}
		field := &typ.Fields[i]
		var deref string
		if !basic.SimpleType(field.Type()) {
			deref = "*"
		}
		fmt.Fprintf(s, "%s: %sx%d, ", fieldName(typ, i), deref, val.Num())
	}
	if stmt.BlockFun != nil {
		s.WriteString("token: token, ")
	}
	s.WriteRune('}')
}

func genMakeOr(stmt *basic.MakeOr, ts typeSet, s *strings.Builder) {
	fmt.Fprintf(s, "*x%d = ", stmt.Dst.Num())
	typ := stmt.Dst.Type().Args[0].Type
	genTypeName(typ, ts, s)
	i := stmt.Case
	fmt.Fprintf(s, "{tag: %d, ", i)
	if stmt.Val != nil {
		cas := &typ.Cases[i]
		var deref string
		if !basic.SimpleType(cas.Type()) {
			deref = "*"
		}
		fmt.Fprintf(s, "%s: %sx%d, ", caseName(typ, i), deref, stmt.Val.Num())
	}
	s.WriteRune('}')
}

func genMakeVirt(stmt *basic.MakeVirt, ts typeSet, s *strings.Builder) {
	fmt.Fprintf(s, "*x%d = ", stmt.Dst.Num())
	typ := stmt.Dst.Type().Args[0].Type
	genTypeName(typ, ts, s)
	s.WriteRune('{')
	for i, v := range stmt.Virts {
		fmt.Fprintf(s, "%s: ", virtName(typ, i))
		n := genVirtSig(&typ.Virts[i], ts, s)
		s.WriteRune('{')
		mangleFun(v, s)
		fmt.Fprintf(s, "(x%d", stmt.Obj.Num())
		for i := 0; i < n; i++ {
			fmt.Fprintf(s, ", p%d", i)
		}
		s.WriteString(")}, ")
	}
	s.WriteRune('}')
}

func genPanic(f *basic.Fun, stmt *basic.Panic, s *strings.Builder) {
	loc := f.Mod.Mod.AST.Locs.Loc(stmt.Msg.AST.GetRange())
	fmt.Fprintf(s, "panic(panicVal{msg: string(*x%d), file: %q, line: %d})",
		stmt.Arg.Num(), loc.Path, loc.Line[0])
}

func genCall(f *basic.Fun, stmt *basic.Call, s *strings.Builder) {
	if f.Block == nil && f.Fun != nil && f.Fun.Test {
		// This is a call made from a test.
		// Wrap the call in a function containing a defer
		// to catch a panicVal panic and
		// set the testFile and testLine.
		loc := f.Mod.Mod.AST.Locs.Loc(stmt.Msg.AST.GetRange())
		fmt.Fprintf(s, "func() {defer recoverTestLoc(%q, %d); ",
			loc.Path, loc.Line[0])
		defer s.WriteString("}()")
	}
	mangleFun(stmt.Fun, s)
	s.WriteRune('(')
	for i, arg := range stmt.Args {
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "x%d", arg.Num())
	}
	s.WriteRune(')')
}

func genVirtCall(stmt *basic.VirtCall, s *strings.Builder) {
	typ := stmt.Self.Type().Args[0].Type
	fmt.Fprintf(s, "x%d.%s(", stmt.Self.Num(), virtName(typ, stmt.Index))
	// Strip off the self argument.
	// Go code gen handles that as a closure
	// at the time the Virt is created.
	for i, arg := range stmt.Args[1:] {
		if i > 0 {
			s.WriteString(", ")
		}
		fmt.Fprintf(s, "x%d", arg.Num())
	}
	s.WriteRune(')')
}

func genRet(stmt *basic.Ret, s *strings.Builder) {
	if stmt.Far {
		s.WriteString("panic(p0.token)")
	} else {
		s.WriteString("return")
	}
}

func genSwitch(stmt *basic.Switch, s *strings.Builder) {
	fmt.Fprintf(s, "switch x%d {", stmt.Val.Num())
	if stmt.Val.Type().BuiltIn == types.BoolType {
		// TODO: remove the hack to reverse bool 0/1.
		for i, b := range stmt.Dsts {
			fmt.Fprintf(s, "case %d: goto L%d; ", 1-i, b.N)
		}
	} else {
		for i, b := range stmt.Dsts {
			fmt.Fprintf(s, "case %d: goto L%d; ", i, b.N)
		}
	}
	s.WriteRune('}')
}

func genVal(v basic.Val, ts typeSet, s *strings.Builder) {
	fmt.Fprintf(s, "x%d = ", v.Num())
	switch v := v.(type) {
	case *basic.IntLit:
		t := builtInTypes[v.Type().BuiltIn]
		fmt.Fprintf(s, "%s(%s)", t, v.Val.String())
	case *basic.FloatLit:
		t := builtInTypes[v.Type().BuiltIn]
		// 39 digits of precision are needed to output the values of
		// math.MaxFloat64 and math.SmallestNonzeroFloat64.
		fmt.Fprintf(s, "%s(%.39e)", t, v.Val)
	case *basic.Op:
		genOp(v, s)
	case *basic.Load:
		fmt.Fprintf(s, "*x%d", v.Src.Num())
	case *basic.Alloc:
		s.WriteString("new(")
		genTypeName(v.Type().Args[0].Type, ts, s)
		s.WriteRune(')')
	case *basic.Arg:
		fmt.Fprintf(s, "p%d", v.Parm.N)
	case *basic.Global:
		fmt.Fprintf(s, "&%s", valName(v.Val))
	case *basic.Index:
		if v.Type().BuiltIn == types.RefType {
			s.WriteRune('&')
		}
		fmt.Fprintf(s, "(*x%d)[x%d]", v.Ary.Num(), v.Index.Num())
	case *basic.Field:
		n := v.Obj.Num()
		i := v.Index
		typ := v.Obj.Type().Args[0].Type
		if len(typ.Cases) > 0 {
			fmt.Fprintf(s, "&x%d.%s", n, caseName(typ, i))
		} else {
			fmt.Fprintf(s, "&x%d.%s", n, fieldName(typ, i))
		}
	default:
		panic(fmt.Sprintf("impossible type %T", v))
	}
}

func genOp(op *basic.Op, s *strings.Builder) {
	switch {
	case op.Code == basic.ArraySizeOp:
		fmt.Fprintf(s, "len(*x%d)", op.Args[0].Num())

	case op.Code == basic.UnionTagOp:
		fmt.Fprintf(s, "(*x%d).tag", op.Args[0].Num())

	case op.Code == basic.NumConvertOp:
		fmt.Fprintf(s, "%s(x%d)",
			builtInTypes[op.Type().BuiltIn], op.Args[0].Num())

	case numOps[op.Code] != "":
		c := numOps[op.Code]
		l := op.Args[0].Num()
		if len(op.Args) == 1 {
			fmt.Fprintf(s, "%s x%d", c, l)
			break
		}
		r := op.Args[1].Num()
		fmt.Fprintf(s, "x%d %s x%d", l, c, r)

	case cmpOps[op.Code] != "":
		c := cmpOps[op.Code]
		l := op.Args[0].Num()
		r := op.Args[1].Num()
		fmt.Fprintf(s, "0; if x%d %s x%d { x%d = 1 }", l, c, r, op.Num())

	default:
		panic("impossible")
	}
}
