package pea

import "fmt"

func builtinMod(wordSize int) *Mod { return &Mod{Defs: builtinDefs(wordSize)} }

func builtinDefs(wordSize int) []Def {
	defs := []Def{
		&Type{Sig: TypeSig{Name: "Nil"}},

		&Type{
			Sig: TypeSig{Name: "&", Parms: []Parm{{Name: "T"}}},
		},
		&Fun{
			Sel:  "deref",
			Recv: &TypeSig{Name: "&", Parms: []Parm{{Name: "T"}}},
			Ret:  &TypeName{Name: "T"},
		},

		&Type{
			Sig:   TypeSig{Name: "Bool"},
			Cases: []Parm{{Name: "true"}, {Name: "false"}},
		},
		&Fun{
			Sel:       "ifTrue:ifFalse:",
			Recv:      &TypeSig{Name: "Bool"},
			TypeParms: []Parm{{Name: "R"}},
			Parms: []Parm{
				{
					Type: &TypeName{
						Name: "[|]",
						Args: []TypeName{{Name: "R"}},
					},
				},
				{
					Type: &TypeName{
						Name: "[|]",
						Args: []TypeName{{Name: "R"}},
					},
				},
			},
			Ret: &TypeName{Name: "R"},
		},

		&Type{Sig: TypeSig{Name: "String"}},
		&Fun{
			Sel:  "byteSize",
			Recv: &TypeSig{Name: "String"},
			Ret:  &TypeName{Name: "Int"},
		},
		&Fun{
			Sel:   "atByte:",
			Recv:  &TypeSig{Name: "String"},
			Parms: []Parm{{Type: &TypeName{Name: "Int"}}},
			Ret:   &TypeName{Name: "Byte"},
		},
		&Fun{
			Sel:  "fromByte:toByte:",
			Recv: &TypeSig{Name: "String"},
			Parms: []Parm{
				{Type: &TypeName{Name: "Int"}},
				{Type: &TypeName{Name: "Int"}},
			},
			Ret: &TypeName{Name: "String"},
		},

		&Type{
			Sig: TypeSig{Name: "Array", Parms: []Parm{{Name: "T"}}},
		},
		&Fun{
			Sel: "size",
			Recv: &TypeSig{
				Name:  "Array",
				Parms: []Parm{{Name: "T"}},
			},
			Ret: &TypeName{Name: "Int"},
		},
		&Fun{
			Sel: "at:",
			Recv: &TypeSig{
				Name:  "Array",
				Parms: []Parm{{Name: "T"}},
			},
			Parms: []Parm{
				{Type: &TypeName{Name: "Int"}},
			},
			Ret: &TypeName{
				Name: "&",
				Args: []TypeName{{Name: "T"}},
			},
		},
		&Fun{
			Sel: "at:put:",
			Recv: &TypeSig{
				Name:  "Array",
				Parms: []Parm{{Name: "T"}},
			},
			Parms: []Parm{
				{Type: &TypeName{Name: "Int"}},
				{Type: &TypeName{Name: "T"}},
			},
			Ret: &TypeName{Name: "T"},
		},
		&Fun{
			Sel: "from:to:",
			Recv: &TypeSig{
				Name:  "Array",
				Parms: []Parm{{Name: "T"}},
			},
			Parms: []Parm{
				{Type: &TypeName{Name: "Int"}},
				{Type: &TypeName{Name: "Int"}},
			},
			Ret: &TypeName{
				Name: "Array",
				Args: []TypeName{{Name: "T"}},
			},
		},

		&Type{Sig: TypeSig{Name: "Byte"}, Alias: &TypeName{Name: "Uint8"}},
		&Type{Sig: TypeSig{Name: "Word"}, Alias: &TypeName{Name: fmt.Sprintf("Uint%d", wordSize)}},
		&Type{Sig: TypeSig{Name: "Uint"}, Alias: &TypeName{Name: fmt.Sprintf("Uint%d", wordSize)}},
		&Type{Sig: TypeSig{Name: "Int"}, Alias: &TypeName{Name: fmt.Sprintf("Int%d", wordSize)}},
		&Type{Sig: TypeSig{Name: "Rune"}, Alias: &TypeName{Name: "Int"}},
		&Type{Sig: TypeSig{Name: "Float"}, Alias: &TypeName{Name: "Float64"}},
	}
	for i := 0; i <= len(funTypeParms); i++ {
		defs = append(defs, funDefs(i)...)
		defs = append(defs, funAlias(i))
	}
	for _, prefix := range [...]string{"Uint", "Int"} {
		for _, size := range [...]int{8, 16, 32, 64} {
			defs = append(defs, intDefs(fmt.Sprintf("%s%d", prefix, size))...)
		}
	}
	for _, size := range [...]int{32, 64} {
		defs = append(defs, numDefs(fmt.Sprintf("Float%d", size))...)
	}
	return defs
}

var funTypeParms = [...]string{"T", "U", "V", "W", "X"}

func funDefs(arity int) []Def {
	ps := make([]Parm, 0, arity+1)
	for i := 0; i < arity; i++ {
		ps = append(ps, Parm{Type: &TypeName{Name: funTypeParms[i]}})
	}
	sel := "value"
	if arity > 0 {
		sel = ""
		for i := 0; i < arity; i++ {
			sel += "value:"
		}
	}
	return []Def{
		&Type{Sig: *funSig(arity)},
		&Fun{
			Sel:   sel,
			Recv:  funSig(arity),
			Parms: ps,
			Ret:   &TypeName{Name: "R"},
		},
	}
}

func funAlias(arity int) Def {
	as := make([]TypeName, 0, arity+1)
	for i := 0; i < arity; i++ {
		as = append(as, TypeName{Name: funTypeParms[i]})
	}
	as = append(as, TypeName{Name: "Nil"})

	aliasSig := funSig(arity)
	aliasSig.Name = fmt.Sprintf("[]%d", arity)
	aliasSig.Parms = aliasSig.Parms[:len(aliasSig.Parms)-1]
	return &Type{
		Sig:   *aliasSig,
		Alias: &TypeName{Name: "[|]", Args: as},
	}
}

func funSig(arity int) *TypeSig {
	sig := &TypeSig{Name: fmt.Sprintf("[|]%d", arity)}
	for i := 0; i < arity; i++ {
		sig.Parms = append(sig.Parms, Parm{Name: funTypeParms[i]})
	}
	sig.Parms = append(sig.Parms, Parm{Name: "R"})
	return sig
}

func intDefs(t string) []Def {
	return append(numDefs(t), []Def{
		builtinBinary(t, "&", t, t),
		builtinBinary(t, "|", t, t),
		builtinUnary(t, "not", t),
		builtinBinary(t, ">>", "Uint", t),
		builtinBinary(t, "<<", "Uint", t),
	}...)
}

func numDefs(t string) []Def {
	return []Def{
		&Type{Sig: TypeSig{Name: t}},
		builtinUnary(t, "neg", t),
		builtinBinary(t, "+", t, t),
		builtinBinary(t, "-", t, t),
		builtinBinary(t, "*", t, t),
		builtinBinary(t, "/", t, t),
		builtinBinary(t, "%", t, t),
		builtinBinary(t, "=", t, "Bool"),
		builtinBinary(t, "!=", t, "Bool"),
		builtinBinary(t, "<", t, "Bool"),
		builtinBinary(t, "<=", t, "Bool"),
		builtinBinary(t, ">", t, "Bool"),
		builtinBinary(t, ">=", t, "Bool"),
		builtinUnary(t, "asByte", "Byte"),
		builtinUnary(t, "asWord", "Word"),
		builtinUnary(t, "asUint", "Uint"),
		builtinUnary(t, "asUint8", "Uint8"),
		builtinUnary(t, "asUint16", "Uint16"),
		builtinUnary(t, "asUint32", "Uint32"),
		builtinUnary(t, "asUint64", "Uint64"),
		builtinUnary(t, "asInt", "Int"),
		builtinUnary(t, "asInt8", "Int8"),
		builtinUnary(t, "asInt16", "Int16"),
		builtinUnary(t, "asInt32", "Int32"),
		builtinUnary(t, "asInt64", "Int64"),
		builtinUnary(t, "asFloat", "asFloat"),
		builtinUnary(t, "asFloat32", "asFloat32"),
		builtinUnary(t, "asFloat64", "asFloat64"),
	}
}

func builtinUnary(recv, op, ret string) *Fun {
	return &Fun{
		Sel:  op,
		Recv: &TypeSig{Name: recv},
		Ret:  &TypeName{Name: ret},
	}
}

func builtinBinary(recv, op, parm, ret string) *Fun {
	return &Fun{
		Sel:   op,
		Recv:  &TypeSig{Name: recv},
		Parms: []Parm{{Name: "_", Type: &TypeName{Name: parm}}},
		Ret:   &TypeName{Name: ret},
	}
}
