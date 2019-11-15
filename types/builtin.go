package types

import (
	"fmt"
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
	CaseMeth BuiltInMeth = iota + 1
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
)

var builtInMethTag = map[string]BuiltInMeth{
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
	tmp := x.newID()
	tparms := []TypeVar{{Name: tmp}}
	retType := &Type{
		Name:   tmp,
		Var:    &tparms[0],
		refDef: refTypeDef(x),
	}
	retType.Def = retType
	tparms[0].Type = retType
	retName := TypeName{Name: tmp, Type: retType}

	var sel strings.Builder
	selfType := builtInType(x, "&", *makeTypeName(typ))
	self := Var{
		Name:     "self",
		TypeName: makeTypeName(selfType),
		typ:      selfType,
	}
	parms := []Var{self}
	for _, c := range typ.Cases {
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
			Name:     "_",
			TypeName: makeTypeName(parmType),
			typ:      parmType,
		}
		parms = append(parms, parm)
	}
	fun := &Fun{
		AST:  typ.AST,
		Priv: typ.Priv,
		Mod:  typ.Mod,
		Recv: &Recv{
			Parms: typ.Parms,
			Mod:   typ.Mod,
			Arity: len(typ.Parms),
			Name:  typ.Name,
			Type:  typ,
		},
		TParms: tparms,
		Sig: FunSig{
			Sel:   sel.String(),
			Parms: parms,
			Ret:   &retName,
		},
		BuiltIn: CaseMeth,
	}
	fun.Def = fun
	return fun
}

func upperCase(s string) string {
	r, w := utf8.DecodeRuneInString(s)
	return string([]rune{unicode.ToUpper(r)}) + s[w:]
}

func makeVirtMeths(x *scope, typ *Type) []Def {
	var defs []Def
	for _, virt := range typ.Virts {
		defs = append(defs, makeVirtMeth(x, typ, virt))
	}
	return defs
}

func makeVirtMeth(x *scope, typ *Type, sig FunSig) *Fun {
	parms := make([]Var, len(sig.Parms)+1)
	selfType := builtInType(x, "&", *makeTypeName(typ))
	parms[0] = Var{
		Name:     "self",
		TypeName: makeTypeName(selfType),
		typ:      selfType,
	}
	for i, p := range sig.Parms {
		p.Name = "_"
		parms[i+1] = p
	}
	sig.Parms = parms
	fun := &Fun{
		AST:  sig.AST,
		Priv: typ.Priv,
		Mod:  typ.Mod,
		Recv: &Recv{
			Parms: typ.Parms,
			Mod:   typ.Mod,
			Arity: len(typ.Parms),
			Name:  typ.Name,
			Type:  typ,
		},
		Sig:     sig,
		BuiltIn: VirtMeth,
	}
	fun.Def = fun
	return fun
}

func makeTypeName(typ *Type) *TypeName {
	args := typ.Args
	if typ.Args == nil {
		for i := range typ.Parms {
			parm := &typ.Parms[i]
			args = append(args, TypeName{
				Mod:  "",
				Name: parm.Name,
				Type: parm.Type,
			})
		}
	}
	return &TypeName{
		Mod:  typ.Mod,
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
