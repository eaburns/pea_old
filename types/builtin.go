package types

import (
	"fmt"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// BuiltInType tags a built-in type.
type BuiltInType int

// The following are the built-in types.
const (
	RefType BuiltInType = iota + 1
	NilType
	BoolType
	StringType
	ArrayType
	FunType
	IntType
	Int8Type
	Int16Type
	Int32Type
	Int64Type
	UIntType
	UInt8Type
	UInt16Type
	UInt32Type
	UInt64Type
	FloatType
	Float32Type
	Float64Type
	// A BlockType is the type of a block literal.
	BlockType
)

var builtInTypeTag = map[string]BuiltInType{
	"&":       RefType,
	"Nil":     NilType,
	"Bool":    BoolType,
	"String":  StringType,
	"Array":   ArrayType,
	"Fun":     FunType,
	"Int":     IntType,
	"Int8":    Int8Type,
	"Int16":   Int16Type,
	"Int32":   Int32Type,
	"Int64":   Int64Type,
	"UInt":    UIntType,
	"UInt8":   UInt8Type,
	"UInt16":  UInt16Type,
	"UInt32":  UInt32Type,
	"UInt64":  UInt64Type,
	"Float":   FloatType,
	"Float32": Float32Type,
	"Float64": Float64Type,
}

// BuiltInMeth tags a built-in method.
type BuiltInMeth int

// The following are the built-in methods.
const (
	TrueFunc BuiltInMeth = iota + 1
	FalseFunc
	CaseMeth
	VirtMeth
	ArraySizeMeth
	ArrayLoadMeth
	ArrayStoreMeth
	ArraySliceMeth
	FunValueMeth
	BitwiseAndMeth
	BitwiseOrMeth
	BitwiseNotMeth
	RightShiftMeth
	LeftShiftMeth
	NegMeth
	PlusMeth
	MinusMeth
	TimesMeth
	DivideMeth
	ModMeth
	EqMeth
	NeqMeth
	LessMeth
	LessEqMeth
	GreaterMeth
	GreaterEqMeth
	NumConvertMeth
	PanicFunc
	NewStringFunc
	NewArrayFunc

	PrintFunc
)

var builtInFunTag = map[string]BuiltInMeth{
	"print:":                   PrintFunc,
	"newString:":               NewStringFunc,
	"newArray:init:":           NewArrayFunc,
	"panic:":                   PanicFunc,
	"true":                     TrueFunc,
	"false":                    FalseFunc,
	"size":                     ArraySizeMeth,
	"byteSize":                 ArraySizeMeth,
	"at:":                      ArrayLoadMeth,
	"atByte:":                  ArrayLoadMeth,
	"at:put:":                  ArrayStoreMeth,
	"from:to:":                 ArraySliceMeth,
	"fromByte:toByte:":         ArraySliceMeth,
	"value":                    FunValueMeth,
	"value:":                   FunValueMeth,
	"value:value:":             FunValueMeth,
	"value:value:value:":       FunValueMeth,
	"value:value:value:value:": FunValueMeth,
	"&":                        BitwiseAndMeth,
	"|":                        BitwiseOrMeth,
	"not":                      BitwiseNotMeth,
	">>":                       RightShiftMeth,
	"<<":                       LeftShiftMeth,
	"neg":                      NegMeth,
	"+":                        PlusMeth,
	"-":                        MinusMeth,
	"*":                        TimesMeth,
	"/":                        DivideMeth,
	"%":                        ModMeth,
	"=":                        EqMeth,
	"!=":                       NeqMeth,
	"<":                        LessMeth,
	"<=":                       LessEqMeth,
	">":                        GreaterMeth,
	">=":                       GreaterEqMeth,
	"asInt":                    NumConvertMeth,
	"asInt8":                   NumConvertMeth,
	"asInt16":                  NumConvertMeth,
	"asInt32":                  NumConvertMeth,
	"asInt64":                  NumConvertMeth,
	"asUInt":                   NumConvertMeth,
	"asUInt8":                  NumConvertMeth,
	"asUInt16":                 NumConvertMeth,
	"asUInt32":                 NumConvertMeth,
	"asUInt64":                 NumConvertMeth,
	"asFloat":                  NumConvertMeth,
	"asFloat32":                NumConvertMeth,
	"asFloat64":                NumConvertMeth,
}

func builtInMeths(x *scope, defs []Def) []Def {
	var out []Def
	for _, def := range defs {
		switch typ, ok := def.(*Type); {
		case !ok:
			continue
		case len(typ.Cases) > 0:
			out = append(out, makeCaseMeth(x, typ))
		case len(typ.Virts) > 0:
			out = append(out, makeVirtMeths(x, typ)...)
		}
	}
	return out
}

func makeCaseMeth(x *scope, typ *Type) *Fun {
	var fun Fun

	// We create a new instance of typ with its own, cloned params,
	// becaue every definition needs its own unique type parameters.
	recvParms, recvType := instWithClonedParms(x, typ)

	tmp := x.newID()
	tparms := []TypeVar{
		{Name: tmp, ID: x.nextTypeVar()},
	}
	retType := &Type{
		Name:   tmp,
		Var:    &tparms[0],
		refDef: refTypeDef(x),
	}
	retType.Def = retType
	tparms[0].Type = retType
	retName := TypeName{Name: tmp, Type: retType}

	var sel strings.Builder
	self := Var{
		Name:     "self",
		TypeName: makeTypeName(recvType),
		typ:      recvType,
		FunParm:  &fun,
		Index:    0,
	}
	parms := []Var{self}
	for i, c := range recvType.Cases {
		sel.WriteString("if")
		sel.WriteString(upperCase(c.Name))
		var parmType *Type
		if c.TypeName == nil {
			sel.WriteRune(':')
			parmType = builtInType(x, "Fun", retName)
		} else {
			ref := builtInType(x, "&", *c.TypeName)
			parmType = builtInType(x, "Fun", *makeTypeName(ref), retName)
		}
		parm := Var{
			Name:     fmt.Sprintf("x%d", i),
			TypeName: makeTypeName(parmType),
			typ:      parmType,
			FunParm:  &fun,
			Index:    i + 1,
		}
		parms = append(parms, parm)
	}
	fun = Fun{
		AST:     recvType.AST,
		Def:     &fun,
		Priv:    recvType.Priv,
		ModPath: typ.ModPath,
		Recv: &Recv{
			Parms: recvParms,
			Arity: len(recvType.Parms),
			Name:  recvType.Name,
			Type:  recvType,
		},
		TParms: tparms,
		Sig: FunSig{
			Sel:   sel.String(),
			Parms: parms,
			Ret:   &retName,
			typ:   retType,
		},
		BuiltIn: CaseMeth,
	}
	fun.Stmts = makeCaseBody(x, &fun)
	return &fun
}

func upperCase(s string) string {
	r, w := utf8.DecodeRuneInString(s)
	return string([]rune{unicode.ToUpper(r)}) + s[w:]
}

func makeCaseBody(x *scope, fun *Fun) []Stmt {
	var args []Expr
	for i := 1; i < len(fun.Sig.Parms); i++ {
		parm := &fun.Sig.Parms[i]
		args = append(args, &Ident{
			Text: parm.Name,
			Var:  parm,
			typ:  parm.Type(),
		})
	}
	msgs := []Msg{
		{
			Sel:  fun.Sig.Sel,
			Args: args,
			Fun:  fun,
			typ:  fun.Sig.Ret.Type,
		},
	}
	recv := &Convert{
		Expr: &Ident{
			Text: "self",
			Var:  &fun.Sig.Parms[0],
			typ:  fun.Sig.Parms[0].Type(),
		},
		Ref: 1,
		typ: fun.Sig.Parms[0].Type().Ref(),
	}
	call := &Call{Recv: recv, Msgs: msgs}
	ret := &Ret{Expr: call}
	return []Stmt{ret}
}

func makeVirtMeths(x *scope, typ *Type) []Def {
	var defs []Def
	for _, virt := range typ.Virts {
		defs = append(defs, makeVirtMeth(x, typ, virt.Sel))
	}
	return defs
}

func makeVirtMeth(x *scope, typ *Type, sel string) *Fun {
	var fun Fun

	// We create a new instance of typ with its own, cloned params,
	// becaue every definition needs its own unique type parameters.
	// We need to find the corresponding FunSig in the substituted type.
	recvParms, recvType := instWithClonedParms(x, typ)
	var sig FunSig
	for _, virt := range recvType.Virts {
		if virt.Sel == sel {
			sig = virt
			break
		}
	}
	if sig.Sel == "" {
		panic("impossible")
	}

	parms := make([]Var, len(sig.Parms)+1)
	parms[0] = Var{
		Name:     "self",
		TypeName: makeTypeName(recvType),
		FunParm:  &fun,
		Index:    0,
		typ:      recvType,
	}
	for i, p := range sig.Parms {
		p.Name = "_"
		p.FunParm = &fun
		p.Index = i + 1
		parms[i+1] = p
	}
	sig.Parms = parms
	fun = Fun{
		AST:     sig.AST,
		Def:     &fun,
		Priv:    recvType.Priv,
		ModPath: recvType.ModPath,
		Recv: &Recv{
			Parms: recvParms,
			Mod:   modName(typ.ModPath),
			Arity: len(typ.Parms),
			Name:  typ.Name,
			Type:  recvType,
		},
		Sig:     sig,
		BuiltIn: VirtMeth,
	}
	return &fun
}

func makeBlockType(x *scope, blk *Block) *Type {
	name := fmt.Sprintf("$Block%d", x.nextBlockType)
	x.nextBlockType++
	typ := &Type{
		AST:     blk.AST,
		Priv:    true,
		Name:    name,
		BuiltIn: BlockType,
	}
	typ.Def = typ
	typ.Insts = []*Type{typ}
	typ.Fields = make([]Var, 0, len(blk.Captures)+1)
	for i, cap := range blk.Captures {
		v := Var{
			AST:      cap.AST,
			Name:     cap.Name,
			TypeName: cap.TypeName,
			Field:    typ,
			Index:    i,
		}
		if cap.typ != nil {
			v.typ = cap.typ.Ref()
			if v.TypeName == nil {
				// Fields always need a type name.
				// Some captures may have one from the source,
				// if so we want that one, because it will have AST info.
				// Otherwise, we construct a new one here.
				v.TypeName = makeTypeName(cap.typ.Ref())
			}
		}
		typ.Fields = append(typ.Fields, v)
	}

	// Add a field to capture the containing function's return slot.
	if fun := x.function(); fun != nil && fun.Sig.Ret != nil {
		retType := fun.Sig.Ret.Type
		typ.Fields = append(typ.Fields, Var{
			Name:     "",
			TypeName: makeTypeName(retType.Ref()),
			Field:    blk.Type(),
			Index:    len(typ.Fields) - 1,
			typ:      retType.Ref(),
		})
	}
	typ.refDef = builtInType(x, "&", *makeTypeName(typ))
	return typ
}

func instWithClonedParms(x *scope, typ *Type) ([]TypeVar, *Type) {
	parms := cloneTypeParms(x, typ.Parms)
	args := make([]TypeName, 0, len(typ.Parms))
	for i := range parms {
		args = append(args, *makeTypeName(parms[i].Type))
	}
	var errs []checkError
	typ, errs = instType(x, typ, args)
	if len(errs) > 0 {
		panic(fmt.Sprintf("impossible: %v", errs))
	}
	return parms, typ
}

func cloneTypeParms(x *scope, parms0 []TypeVar) []TypeVar {
	if len(parms0) == 0 {
		return nil
	}
	parms1 := make([]TypeVar, len(parms0))
	for i := range parms0 {
		parm0 := &parms0[i]
		parm1 := &parms1[i]
		parm1.Name = parm0.Name
		parm1.ID = x.nextTypeVar()
		parm1.Ifaces = parm0.Ifaces
		parm1.Type = &Type{
			AST:    parm0.Type.AST,
			Name:   parm0.Type.Name,
			refDef: parm0.Type.refDef,
		}
		parm1.Type.Var = parm1
		parm1.Type.Def = parm1.Type
	}
	return parms1
}

func makeTypeName(typ *Type) *TypeName {
	args := typ.Args
	if typ.Args == nil {
		for i := range typ.Parms {
			parm := &typ.Parms[i]
			args = append(args, *makeTypeName(parm.Type))
		}
	}
	return &TypeName{
		Mod:  modName(typ.ModPath),
		Name: typ.Name,
		Args: args,
		Type: typ,
	}
}

func builtInType(x *scope, name string, args ...TypeName) *Type {
	// Silence tracing for looking up built-in types.
	savedTrace := x.cfg.Trace
	x.cfg.Trace = false
	defer func() { x.cfg.Trace = savedTrace }()

	for x.univ == nil {
		x = x.up
	}
	typ := findTypeInDefs(len(args), name, x.univ)
	if typ == nil {
		panic(fmt.Sprintf("built-in type (%d)%s not found", len(args), name))
	}
	typ, errs := instType(x, typ, args)
	if len(errs) > 0 {
		panic(fmt.Sprintf("failed to inst built-in type: %v", errs))
	}
	return typ
}

func isNil(typ *Type) bool {
	return typ != nil && typ.BuiltIn == NilType
}

func isAry(typ *Type) bool {
	return typ != nil && typ.BuiltIn == ArrayType
}

func isRef(typ *Type) bool {
	return typ != nil && typ.BuiltIn == RefType
}

func isFun(typ *Type) bool {
	return typ != nil && typ.BuiltIn == FunType
}

func isAnyInt(typ *Type) bool {
	return typ != nil && typ.BuiltIn >= IntType && typ.BuiltIn <= UInt64Type
}

func disectIntType(cfg Config, typ *Type) (bool, int) {
	switch typ.BuiltIn {
	case IntType:
		return true, cfg.IntSize
	case Int8Type:
		return true, 7
	case Int16Type:
		return true, 15
	case Int32Type:
		return true, 31
	case Int64Type:
		return true, 63
	case UIntType:
		return false, cfg.IntSize
	case UInt8Type:
		return false, 8
	case UInt16Type:
		return false, 16
	case UInt32Type:
		return false, 32
	case UInt64Type:
		return false, 64
	default:
		panic(fmt.Sprintf("impossible int type: %T", typ))
	}
}

func isAnyFloat(typ *Type) bool {
	return typ != nil && typ.BuiltIn >= FloatType && typ.BuiltIn <= Float64Type
}

func isBuiltInType(typ *Type) bool {
	return typ != nil && (typ.BuiltIn != 0 || typ.Alias != nil && typ.Alias.Type != nil && typ.Alias.Type.BuiltIn != 0)
}

func modName(p string) string {
	if p == "" {
		return ""
	}
	return "#" + path.Base(p)
}
