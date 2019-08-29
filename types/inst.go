package types

func instType(x *scope, typ *Type, name *TypeName) (_ *Type, errs []checkError) {
	defer x.tr("instType(%s, %s)", typ.name(), name.name())(&errs)

	if errs = append(errs, checkDefSig(x, typ)...); len(errs) > 0 {
		return nil, errs
	} else if len(typ.Sig.Parms) != typ.Sig.Arity {
		// There was an error on a previous call to check.
		return nil, nil
	}
	if len(name.Args) != typ.Sig.Arity {
		panic("impossible")
	}
	if len(typ.Sig.Parms) == 0 {
		return typ, errs
	}

	// TODO: instantiate parameterized types.
	panic("unimplemented")
}
