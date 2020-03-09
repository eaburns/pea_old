// Copyright © 2020 The Pea Authors under an MIT-style license.

package basic

/*
The bugs() methods perform checking for bugs in the instructions,
returning a string describing a bug found.
A bug string indicates a bug in the basic package, not the user input.

The general scheme is to defer recoverBug(&b) on the return variable b.
Then for each item needing to be checked, use bugIf.
The bugIf function panics if the condition is false.
This means that subsequent bugIfs can assume
that the condition of all preceeding bugIfs was true.

The reason to use bugIf instead of if-statements
is that the condition is always expected to be false.
The bugIf calls, are executed regardless of whether its true or false,
so it will not show up as test coverage losses in the expected, false, case.
*/

import (
	"fmt"

	"github.com/eaburns/pea/types"
)

func (n *Store) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"store to non-reference type %s", n.Dst.Type())
	bugIf(refElemType(n.Dst) != n.Val.Type(),
		"store type mismatch: dst %s != val %s",
		refElemType(n.Dst), n.Val.Type())
	return ""
}

func (n *Copy) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"copy to non-reference type %s", n.Dst.Type())
	bugIf(!isRefType(n.Src),
		"copy from non-reference type %s", n.Src.Type())
	bugIf(n.Dst.Type() != n.Src.Type(),
		"copy type mismatch: dst %s != src %s",
		refElemType(n.Dst), refElemType(n.Src))
	return ""
}

func (n *MakeArray) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"make array of non-reference type %s", n.Dst.Type())
	bugIf(refElemType(n.Dst).BuiltIn != types.ArrayType,
		"make string of non-array-reference type %s",
		refElemType(n.Dst))

	elmType := refElemType(n.Dst).Args[0].Type
	for i, arg := range n.Args {
		bugIf(SimpleType(elmType) && arg.Type() != elmType,
			"make and field %d type mismatch: got %s, want %s",
			i, arg.Type(), elmType)
		bugIf(!SimpleType(elmType) && arg.Type() != elmType.Ref(),
			"make and field %d type mismatch: got %s, want %s",
			i, arg.Type(), elmType.Ref())
	}
	return ""
}

func (n *NewArray) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"make array of non-reference type %s", n.Dst.Type())
	bugIf(refElemType(n.Dst).BuiltIn != types.ArrayType,
		"make string of non-array-reference type %s",
		refElemType(n.Dst))
	bugIf(n.Size.Type().BuiltIn != types.IntType,
		"new array size is not Int: %s", n.Dst.Type())
	return ""
}

func (n *MakeSlice) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"make slice to non-reference type %s", n.Dst.Type())
	bugIf(refElemType(n.Dst).BuiltIn != types.ArrayType,
		"make slice to non-array-reference type %s", refElemType(n.Dst))
	bugIf(!isRefType(n.Ary),
		"make slice from non-reference type %s", n.Ary.Type())
	bugIf(refElemType(n.Ary).BuiltIn != types.ArrayType,
		"make slice from non-array-reference type %s", refElemType(n.Ary))
	bugIf(n.From.Type().BuiltIn != types.IntType,
		"make slice non-Int start type %s", n.From.Type())
	bugIf(n.To.Type().BuiltIn != types.IntType,
		"make slice non-Int end type %s", n.To.Type())
	return ""
}

func (n *MakeString) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"make string of non-reference type %s", n.Dst.Type())
	bugIf(refElemType(n.Dst).BuiltIn != types.StringType,
		"make string of non-string-reference type %s", refElemType(n.Dst))
	return ""
}

func (n *MakeAnd) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"make and of non-reference type %s", n.Dst.Type())
	andType := refElemType(n.Dst)
	for i := range andType.Fields {
		field := &andType.Fields[i]
		bugIf(i >= len(n.Fields), "make and too few args")
		arg := n.Fields[i]
		if arg == nil {
			bugIf(!EmptyType(field.Type()) &&
				// For block literals, we elide empty-type captures.
				// But captures always have one extra level of &,
				// so we have to account for that in this check.
				(andType.BuiltIn != types.BlockType ||
					field.Type().BuiltIn != types.RefType ||
					!EmptyType(field.Type().Args[0].Type)),
				"make and field %d type mismatch: got nil, want %s",
				i, field.Type())
			continue
		}
		bugIf(EmptyType(field.Type()) && arg != nil,
			"make and field %d type mismatch: got %s, want nil",
			i, arg.Type())
		bugIf(SimpleType(field.Type()) && field.Type() != arg.Type(),
			"make and field %d type mismatch: got %s, want %s",
			i, arg.Type(), field.Type())
		bugIf(!SimpleType(field.Type()) && field.Type().Ref() != arg.Type(),
			"make and field %d type mismatch: got %s, want %s",
			i, arg.Type(), field.Type().Ref())
	}
	return ""
}

func (n *MakeOr) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"make or of non-reference type %s", n.Dst.Type())
	orType := refElemType(n.Dst)
	bugIf(len(orType.Cases) <= n.Case,
		"make or tag: %d, but only %d cases", n.Case, len(orType.Cases))
	c := &orType.Cases[n.Case]
	bugIf(c.TypeName != nil && !EmptyType(c.Type()) && n.Val == nil,
		"make or type mismatch: got nil, want %s", c.Type())
	if n.Val == nil {
		return ""
	}
	bugIf(c.TypeName == nil,
		"make or type mismatch: got %s, want nil", n.Val.Type())
	bugIf(c.TypeName != nil &&
		SimpleType(c.Type()) &&
		c.Type() != n.Val.Type(),
		"make or type mismatch: got %s, want %s", n.Val.Type(), c.Type())
	bugIf(c.TypeName != nil &&
		!SimpleType(c.Type()) &&
		c.Type().Ref() != n.Val.Type(),
		"make or type mismatch: got %s, want %s", n.Val.Type().Ref(), c.Type())
	return ""
}

func (n *MakeVirt) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Dst),
		"make virt with non-reference dest %s", n.Dst.Type())
	virtType := refElemType(n.Dst)
	bugIf(len(n.Virts) != len(virtType.Virts),
		"make virt count mismatch: got %d, want %d",
		len(n.Virts), len(virtType.Virts))
	bugIf(!isRefType(n.Obj),
		"make virt with non-reference obj %s", n.Obj.Type())
	return ""
}

func (n *Call) bugs() (b string) {
	defer recoverBug(&b)
	parms := n.Fun.Parms
	if n.Fun.Ret != nil {
		parms = append(parms, n.Fun.Ret)
	}
	bugIf(len(n.Args) != len(parms),
		"call argument count mismatch: got %d, want %d",
		len(n.Args), len(parms))
	for i, a := range n.Args {
		bugIf(a.Type() != parms[i].Type,
			"argument %d type mismatch: got %s, want %s",
			i, a.Type(), parms[i].Type)
	}
	return ""
}

func (n *VirtCall) bugs() (b string) {
	defer recoverBug(&b)
	recv := n.Args[0]
	bugIf(!isRefType(recv),
		"virtual call to non-reference type %s", recv.Type())
	virtType := refElemType(recv)
	bugIf(len(virtType.Virts) == 0,
		"virtual call to non-virt-reference type %s", virtType)
	bugIf(n.Index < 0 || n.Index >= len(virtType.Virts),
		"virtual call index out of bounds: %d (%s max=%d)",
		n.Index, virtType, len(virtType.Virts)-1)
	virt := virtType.Virts[n.Index]
	checkArgs := n.Args[1:]
	if virt.Ret != nil && !EmptyType(virt.Ret.Type) {
		// strip return value location
		checkArgs = checkArgs[:len(checkArgs)-1]
	}
	bugIf(len(checkArgs) != len(virt.Parms),
		"virtual call argument count mismatch: got %d, want %d",
		len(checkArgs), len(virt.Parms))
	for i, a := range checkArgs {
		wantType := virt.Parms[i].Type()
		if !SimpleType(wantType) {
			wantType = wantType.Ref()
		}
		bugIf(a.Type() != wantType,
			"argument %d type mismatch: got %s, want %s",
			i, a.Type(), wantType)
	}
	return ""
}

func (n *Switch) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(len(n.OrType.Cases) != len(n.Dsts),
		"switch case count mismatch: got %d, want %d",
		len(n.Dsts), len(n.OrType.Cases))
	bugIf(n.Val.Type().BuiltIn != types.UInt8Type &&
		n.Val.Type().BuiltIn != types.UInt16Type &&
		!enumType(n.Val.Type()),
		"switch value type mismatch: got %s, "+
			"want UInt8, UInt16, or an enum-style or-type",
		n.Val.Type())
	return ""
}

func (n *Op) bugs() (b string) {
	defer recoverBug(&b)
	for i, arg := range n.Args {
		bugIf(arg == nil || EmptyType(arg.Type()),
			"op arg %d is an empty type", i)
		bugIf(!SimpleType(arg.Type()),
			"op arg %d is a composite type %s", i, arg.Type())
		bugIf(n.Code != ArraySizeOp &&
			n.Code != UnionTagOp &&
			isRefType(arg),
			"op arg %d is a reference type %s", i, arg.Type())
	}
	return ""
}

func (n *Load) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Src),
		"load from non-reference type %s", n.Src.Type())
	bugIf(!SimpleType(refElemType(n.Src)),
		"load a composite type %s", refElemType(n.Src))
	return ""
}

func (n *Index) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Ary),
		"index of non-reference type %s", n.Ary.Type())
	aryType := refElemType(n.Ary)
	bugIf(aryType.BuiltIn != types.ArrayType &&
		aryType.BuiltIn != types.StringType,
		"index of non-array reference type %s", n.Ary.Type())
	bugIf(n.Index.Type().BuiltIn != types.IntType,
		"index with non-Int index type %s", n.Index.Type())
	return ""
}

func (n *Field) bugs() (b string) {
	defer recoverBug(&b)
	bugIf(!isRefType(n.Obj),
		"field of non-reference type %s", n.Obj.Type())
	objType := refElemType(n.Obj)
	switch {
	case objType.BuiltIn == types.BlockType:
		bugIf(n.Index < 0 ||
			n.Index >= len(objType.Fields),
			"field %d does not exist on type %s", n.Index, objType)
	case len(objType.Cases) > 0:
		bugIf(n.Index < 0 ||
			n.Index >= len(objType.Cases),
			"field %d does not exist on type %s", n.Index, objType)
	case len(objType.Fields) > 0:
		bugIf(n.Index < 0 ||
			n.Index >= len(objType.Fields),
			"field %d does not exist on type %s", n.Index, objType)
	}
	return ""
}

func isRefType(v Val) bool {
	return v.Type().BuiltIn == types.RefType
}

func refElemType(v Val) *types.Type {
	return v.Type().Args[0].Type
}

type bug string

func recoverBug(ret *string) {
	if b, ok := recover().(bug); ok {
		*ret = string(b)
	}
}

func bugIf(c bool, f string, vs ...interface{}) {
	if c {
		panic(bug(fmt.Sprintf(f, vs...)))
	}
}
