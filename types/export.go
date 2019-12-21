package types

import (
	"fmt"
	"io"
	"math"
	"sort"
)

type tag int

const (
	valTag = iota + 1
	funTag
	typeTag
	varTag
)

// Write writes an exported module.
func Write(w io.Writer, m *Mod) (err error) {
	defer func() {
		x := recover()
		if x == nil {
			return
		}
		if ioErr, ok := x.(ioError); !ok {
			panic(x)
		} else {
			err = ioErr.err
		}
	}()
	writeString(w, m.Name)
	objs := &outObjs{
		num:     make(map[interface{}]int),
		written: make(map[interface{}]bool),
	}
	writeInt(w, len(m.Defs))
	for _, def := range m.Defs {
		writeObj(w, objs, def)
	}
	writeInt(w, getTypeNum(objs, m.IntType))

	for {
		var todo []interface{}
		for o := range objs.num {
			if !objs.written[o] {
				todo = append(todo, o)
			}
		}
		sort.Slice(todo, func(i, j int) bool {
			return objs.num[todo[i]] < objs.num[todo[j]]
		})
		writeInt(w, len(todo))
		if len(todo) == 0 {
			break
		}
		for _, o := range todo {
			writeObj(w, objs, o)
		}
	}

	return nil
}

// Read reads an exported module.
func Read(r io.Reader) (m *Mod, err error) {
	defer func() {
		x := recover()
		if x == nil {
			return
		}
		if ioErr, ok := x.(ioError); !ok {
			panic(x)
		} else {
			err = ioErr.err
		}
	}()
	m = &Mod{
		Name: readString(r),
	}
	var objs inObjs
	n := readInt(r)
	for i := 0; i < n; i++ {
		m.Defs = append(m.Defs, readObj(r, &objs).(Def))
	}
	patchType(&objs, readInt(r), &m.IntType)

	for {
		n := readInt(r)
		if n == 0 {
			break
		}
		for i := 0; i < n; i++ {
			readObj(r, &objs)
		}
	}
	applyPatches(&objs)
	return m, nil
}

// writeDef writes an object.
func writeObj(w io.Writer, objs *outObjs, obj interface{}) {
	switch obj := obj.(type) {
	case *Val:
		writeInt(w, valTag)
		writeVal(w, objs, obj)
	case *Fun:
		writeInt(w, funTag)
		writeFun(w, objs, obj)
	case *Type:
		writeInt(w, typeTag)
		writeType(w, objs, obj)
	case *Var:
		writeInt(w, varTag)
		writeVar(w, objs, obj)
	default:
		panic(fmt.Sprintf("impossible type: %T", obj))
	}
}

// readObj reads an object.
func readObj(r io.Reader, objs *inObjs) interface{} {
	switch tag := readInt(r); tag {
	case valTag:
		return readVal(r, objs)
	case funTag:
		return readFun(r, objs)
	case typeTag:
		return readType(r, objs)
	case varTag:
		var v Var
		readVar(r, objs, &v)
		return &v
	default:
		panic(fmt.Sprintf("impossible obj tag: %d", tag))
	}
}

// writeVal writes the Val number, then the following fields of Val:
// 	Priv
// 	Mod
// 	Var
// It does not write:
// 	Init, since the initialization is not needed on import.
// 	Locals, since they would only be needed by Init, which is not written.
func writeVal(w io.Writer, objs *outObjs, v *Val) {
	writeInt(w, getValNum(objs, v))
	writeBool(w, v.Priv)
	writeString(w, v.Mod)
	writeVar(w, objs, &v.Var)
	objs.written[v] = true
}

func readVal(r io.Reader, objs *inObjs) *Val {
	n := readInt(r)
	v := &Val{
		Priv: readBool(r),
		Mod:  readString(r),
	}
	readVar(r, objs, &v.Var)
	v.Var.Val = v
	addVal(objs, n, v)
	return v
}

// writeFun writes the Fun number, then the following fields of Fun:
// 	Def number
// 	Priv
// 	Mod
// 	a bool for whether Recv is set
// 	Recv if set
// 	the number of TParms
// 	TParms
// 	the number of Targs
// 	TArgs type numbers
// 	Sig
// 	the number of locals
// 	Locals
// 	the number of Stmts or -1 for a declaration (Stmts==nil)
// 	Stmts
// 	BuiltIn
// It does not write:
// 	Insts; they are used only private to this module.
func writeFun(w io.Writer, objs *outObjs, f *Fun) {
	writeInt(w, getFunNum(objs, f))
	writeInt(w, getFunNum(objs, f.Def))
	writeBool(w, f.Priv)
	writeString(w, f.Mod)
	writeBool(w, f.Recv != nil)
	if f.Recv != nil {
		writeRecv(w, objs, f.Recv)
	}
	writeInt(w, len(f.TParms))
	for i := range f.TParms {
		writeTypeVar(w, objs, &f.TParms[i])
	}
	writeInt(w, len(f.TArgs))
	for i := range f.TArgs {
		writeInt(w, getTypeNum(objs, f.TArgs[i].Type))
	}
	writeFunSig(w, objs, &f.Sig)
	writeInt(w, len(f.Locals))
	for _, l := range f.Locals {
		writeVar(w, objs, l)
	}
	if f.Stmts == nil {
		writeInt(w, -1)
	} else {
		writeInt(w, len(f.Stmts))
		for _, stmt := range f.Stmts {
			writeStmt(w, objs, stmt)
		}
	}
	writeInt(w, int(f.BuiltIn))
	objs.written[f] = true
}

func readFun(r io.Reader, objs *inObjs) *Fun {
	var f Fun
	n := readInt(r)
	patchFun(objs, readInt(r), &f.Def)
	f.Priv = readBool(r)
	f.Mod = readString(r)
	if readBool(r) {
		f.Recv = readRecv(r, objs)
	}
	if nparm := readInt(r); nparm > 0 {
		f.TParms = make([]TypeVar, nparm)
		for i := range f.TParms {
			readTypeVar(r, objs, &f.TParms[i])
		}
	}
	if nargs := readInt(r); nargs > 0 {
		f.TArgs = make([]TypeName, nargs)
		for i := range f.TArgs {
			patchTypeName(objs, readInt(r), &f.TArgs[i])
		}
	}
	readFunSig(r, objs, &f, &f.Sig)
	if nloc := readInt(r); nloc > 0 {
		f.Locals = make([]*Var, nloc)
		for i := range f.Locals {
			var l Var
			readVar(r, objs, &l)
			l.Local = &f.Locals
			l.Index = i
			f.Locals[i] = &l
		}
	}
	if nstmt := readInt(r); nstmt >= 0 {
		f.Stmts = make([]Stmt, nstmt)
		for i := range f.Stmts {
			f.Stmts[i] = readStmt(r, objs)
		}
	}
	f.BuiltIn = BuiltInMeth(readInt(r))
	addFun(objs, n, &f)
	return &f
}

// writeRecv writes the following fields of Recv:
// 	number of Parms
// 	Parms
// 	number of Args
// 	Args type numbers
// 	Mod
// 	Arity
// 	Name
// 	Type number
func writeRecv(w io.Writer, objs *outObjs, r *Recv) {
	writeInt(w, len(r.Parms))
	for i := range r.Parms {
		writeTypeVar(w, objs, &r.Parms[i])
	}
	writeInt(w, len(r.Args))
	for i := range r.Args {
		writeInt(w, getTypeNum(objs, r.Args[i].Type))
	}
	writeString(w, r.Mod)
	writeInt(w, r.Arity)
	writeString(w, r.Name)
	writeInt(w, getTypeNum(objs, r.Type))
}

func readRecv(r io.Reader, objs *inObjs) *Recv {
	var v Recv
	if nparms := readInt(r); nparms > 0 {
		v.Parms = make([]TypeVar, nparms)
		for i := range v.Parms {
			readTypeVar(r, objs, &v.Parms[i])
		}
	}
	if nargs := readInt(r); nargs > 0 {
		v.Args = make([]TypeName, nargs)
		for i := range v.Args {
			patchTypeName(objs, readInt(r), &v.Args[i])
		}
	}
	v.Mod = readString(r)
	v.Arity = readInt(r)
	v.Name = readString(r)
	patchType(objs, readInt(r), &v.Type)
	return &v
}

// writeTypeVar writes the following fields ofTypeVar:
// 	Name
// 	number of Ifaces
// 	Ifaces type numbers
// 	Type as the full type, not just its number
func writeTypeVar(w io.Writer, objs *outObjs, v *TypeVar) {
	writeString(w, v.Name)
	writeInt(w, len(v.Ifaces))
	for i := range v.Ifaces {
		writeInt(w, getTypeNum(objs, v.Ifaces[i].Type))
	}
	writeType(w, objs, v.Type)
}

func readTypeVar(r io.Reader, objs *inObjs, v *TypeVar) {
	v.Name = readString(r)
	if nifaces := readInt(r); nifaces > 0 {
		v.Ifaces = make([]TypeName, nifaces)
		for i := range v.Ifaces {
			patchTypeName(objs, readInt(r), &v.Ifaces[i])
		}
	}
	v.Type = readType(r, objs)
	v.Type.Var = v
}

// writeFunSig writes the following fields of FunSig:
// 	Sel
// 	number of Parms
// 	Parms
// 	a bool indicating whether Ret is nil
// 	Ret type number
// 	a bool indicating whether typ is nil
// 	typ type number
func writeFunSig(w io.Writer, objs *outObjs, s *FunSig) {
	writeString(w, s.Sel)
	writeInt(w, len(s.Parms))
	for i := range s.Parms {
		writeVar(w, objs, &s.Parms[i])
	}
	writeBool(w, s.Ret != nil)
	if s.Ret != nil {
		writeInt(w, getTypeNum(objs, s.Ret.Type))
	}
	writeBool(w, s.typ != nil)
	if s.typ != nil {
		writeInt(w, getTypeNum(objs, s.typ))
	}
}

func readFunSig(r io.Reader, objs *inObjs, f *Fun, s *FunSig) {
	s.Sel = readString(r)
	if nparms := readInt(r); nparms > 0 {
		s.Parms = make([]Var, nparms)
		for i := range s.Parms {
			readVar(r, objs, &s.Parms[i])
			if f != nil {
				s.Parms[i].FunParm = f
				s.Parms[i].Index = i
			}
		}
	}
	if readBool(r) {
		patchTypeNamePtr(objs, readInt(r), &s.Ret)
	}
	if readBool(r) {
		patchType(objs, readInt(r), &s.typ)
	}
}

func writeStmt(w io.Writer, objs *outObjs, s Stmt) {
	// TODO: implement writeStmt
}

func readStmt(r io.Reader, objs *inObjs) Stmt {
	// TODO: implement readStmt
	return nil
}

// writeType writes the Type number, then the following fields of Type:
// 	Def type number
// 	Priv
// 	Mod
// 	Arity
// 	Name
// 	number of Parms
// 	Parms
// 	number of args
// 	Args type numbers
// 	Alias type number or -1 if nil
// 	number of Fields
// 	Fields
// 	number of Cases
// 	Cases
// 	number of Virts
// 	Virts
// 	BuiltIn
// 	refDef type number
// 	tagType type number or -1 if nil
// It does not write:
// 	Insts; insts are only written if otherwise referenced.
// 	Var, since this can be reconstructed when reading a TypeVar.
func writeType(w io.Writer, objs *outObjs, t *Type) {
	writeInt(w, getTypeNum(objs, t))
	writeInt(w, getTypeNum(objs, t.Def))
	writeBool(w, t.Priv)
	writeString(w, t.Mod)
	writeInt(w, t.Arity)
	writeString(w, t.Name)
	writeInt(w, len(t.Parms))
	for i := range t.Parms {
		writeTypeVar(w, objs, &t.Parms[i])
	}
	writeInt(w, len(t.Args))
	for i := range t.Args {
		writeInt(w, getTypeNum(objs, t.Args[i].Type))
	}
	if t.Alias == nil {
		writeInt(w, -1)
	} else {
		writeInt(w, getTypeNum(objs, t.Alias.Type))
	}
	writeInt(w, len(t.Fields))
	for i := range t.Fields {
		writeVar(w, objs, &t.Fields[i])
	}
	writeInt(w, len(t.Cases))
	for i := range t.Cases {
		writeVar(w, objs, &t.Cases[i])
	}
	writeInt(w, len(t.Virts))
	for i := range t.Virts {
		writeFunSig(w, objs, &t.Virts[i])
	}
	writeInt(w, int(t.BuiltIn))
	writeInt(w, getTypeNum(objs, t.refDef))
	if t.tagType == nil {
		writeInt(w, -1)
	} else {
		writeInt(w, getTypeNum(objs, t.tagType))
	}
	objs.written[t] = true
}

func readType(r io.Reader, objs *inObjs) *Type {
	var t Type
	n := readInt(r)
	patchType(objs, readInt(r), &t.Def)
	t.Priv = readBool(r)
	t.Mod = readString(r)
	t.Arity = readInt(r)
	t.Name = readString(r)
	if nparms := readInt(r); nparms > 0 {
		t.Parms = make([]TypeVar, nparms)
		for i := range t.Parms {
			readTypeVar(r, objs, &t.Parms[i])
		}
	}
	if nargs := readInt(r); nargs > 0 {
		t.Args = make([]TypeName, nargs)
		for i := range t.Args {
			patchTypeName(objs, readInt(r), &t.Args[i])
		}
	}
	if aliasNum := readInt(r); aliasNum >= 0 {
		patchTypeNamePtr(objs, aliasNum, &t.Alias)
	}
	if nfields := readInt(r); nfields > 0 {
		t.Fields = make([]Var, nfields)
		for i := range t.Fields {
			readVar(r, objs, &t.Fields[i])
			t.Fields[i].Field = &t
			t.Fields[i].Index = i
		}
	}
	if ncases := readInt(r); ncases > 0 {
		t.Cases = make([]Var, ncases)
		for i := range t.Cases {
			readVar(r, objs, &t.Cases[i])
			t.Cases[i].Case = &t
			t.Cases[i].Index = i
		}
	}
	if nvirts := readInt(r); nvirts > 0 {
		t.Virts = make([]FunSig, nvirts)
		for i := range t.Virts {
			readFunSig(r, objs, nil, &t.Virts[i])
		}
	}
	t.BuiltIn = BuiltInType(readInt(r))
	patchType(objs, readInt(r), &t.refDef)
	if tagNum := readInt(r); tagNum >= 0 {
		patchType(objs, tagNum, &t.tagType)
	}
	addType(objs, n, &t)
	return &t
}

// writeVar writes the following fields of Var:
// 	the Var's object number
// 	Name
// 	Index
// 	a bool indicating whether typ is set, and if true:
// 		typ type num
// 		a bool indicating whether TypeName was explicitly set
// It does not write:
// 	TypeName, since that can be computed from the type when reading.
//	Val
// 	FunParm
// 	BlkParm
// 	Local
// 	Field
// 	Case, since these will all be know from the context when reading.
func writeVar(w io.Writer, objs *outObjs, v *Var) {
	writeInt(w, getVarNum(objs, v))
	writeString(w, v.Name)
	writeInt(w, v.Index)
	writeBool(w, v.typ != nil)
	if v.typ != nil {
		writeInt(w, getTypeNum(objs, v.typ))
		writeBool(w, v.TypeName != nil)
	}
	objs.written[v] = true
}

func readVar(r io.Reader, objs *inObjs, v *Var) {
	n := readInt(r)
	v.Name = readString(r)
	v.Index = readInt(r)
	if readBool(r) {
		tn := readInt(r)
		patchType(objs, tn, &v.typ)
		if readBool(r) {
			patchTypeNamePtr(objs, tn, &v.TypeName)
		}
	}
	addVar(objs, n, v)
}

type outObjs struct {
	num     map[interface{}]int
	written map[interface{}]bool
}

func (objs *outObjs) getNum(v interface{}) int {
	i, ok := objs.num[v]
	if !ok {
		i = len(objs.num)
		objs.num[v] = i
	}
	return i
}

func getValNum(objs *outObjs, v *Val) int   { return objs.getNum(v) }
func getFunNum(objs *outObjs, f *Fun) int   { return objs.getNum(f) }
func getTypeNum(objs *outObjs, t *Type) int { return objs.getNum(t) }
func getVarNum(objs *outObjs, v *Var) int   { return objs.getNum(v) }

type inObjs struct {
	nobjs int
	// patches has an entry for each object number
	// containing a slice of pointers to pointers to that object.
	// These pointers are patched with a pointer to the object
	// after all objecs are read.
	//
	// For types, the patch may be a **Type, *TypeName, or **TypeName
	patches [][]interface{}
	objs    []interface{}
}

func (objs *inObjs) add(n int, o interface{}) {
	objs.ensure(n)
	objs.objs[n] = o
}

func (objs *inObjs) patch(n int, pp interface{}) {
	switch pp.(type) {
	case **Var, **Fun, **Type, *TypeName, **TypeName:
		break
	default:
		panic(fmt.Sprintf("bad patch type: %T", pp))
	}
	objs.ensure(n)
	objs.patches[n] = append(objs.patches[n], pp)
}

func (objs *inObjs) ensure(n int) {
	l := len(objs.patches)
	if l == 0 {
		l = 1
	}
	for n >= l {
		l *= 2
	}
	if l > len(objs.patches) {
		ps := make([][]interface{}, l)
		copy(ps, objs.patches)
		objs.patches = ps
		os := make([]interface{}, l)
		copy(os, objs.objs)
		objs.objs = os
	}
	if n+1 > objs.nobjs {
		objs.nobjs = n + 1
	}
}

func patchVal(objs *inObjs, n int, v **Val)              { objs.patch(n, v) }
func patchFun(objs *inObjs, n int, f **Fun)              { objs.patch(n, f) }
func patchType(objs *inObjs, n int, t **Type)            { objs.patch(n, t) }
func patchTypeName(objs *inObjs, n int, t *TypeName)     { objs.patch(n, t) }
func patchTypeNamePtr(objs *inObjs, n int, t **TypeName) { objs.patch(n, t) }
func patchVar(objs *inObjs, n int, v **Var)              { objs.patch(n, v) }

func addVal(objs *inObjs, n int, v *Val)   { objs.add(n, v) }
func addFun(objs *inObjs, n int, f *Fun)   { objs.add(n, f) }
func addType(objs *inObjs, n int, t *Type) { objs.add(n, t) }
func addVar(objs *inObjs, n int, v *Var)   { objs.add(n, v) }

func applyPatches(objs *inObjs) {
	var types []*Type
	for i, obj := range objs.objs[:objs.nobjs] {
		switch obj := obj.(type) {
		case *Val:
			for _, p := range objs.patches[i] {
				*p.(**Val) = obj
			}
		case *Fun:
			for _, p := range objs.patches[i] {
				*p.(**Fun) = obj
			}
		case *Type:
			types = append(types, obj)
			for _, p := range objs.patches[i] {
				switch p := p.(type) {
				case **Type:
					*p = obj
				case *TypeName:
					*p = *makeTypeName(obj)
				case **TypeName:
					*p = makeTypeName(obj)
				default:
					panic(fmt.Sprintf("impossible type: %T", p))
				}
			}
		case *Var:
			for _, p := range objs.patches[i] {
				*p.(**Var) = obj
			}
		default:
			panic(fmt.Sprintf("impossible type: %T", obj))
		}
	}
	for _, t := range types {
		t.Def.Insts = append(t.Def.Insts, t)
	}
}

type ioError struct {
	err error
}

func writeBool(w io.Writer, b bool) {
	var bs [1]byte
	if b {
		bs[0] = 1
	}
	if _, err := w.Write(bs[:]); err != nil {
		panic(ioError{err})
	}
}

func readBool(r io.Reader) bool {
	var bs [1]byte
	if _, err := io.ReadFull(r, bs[:]); err != nil {
		panic(ioError{err})
	}
	return bs[0] == 1
}

func writeInt(w io.Writer, n int) {
	if n > math.MaxInt32 {
		panic("int too big")
	}
	if n < math.MinInt32 {
		panic("int too small")
	}
	if _, err := w.Write([]byte{
		byte(0xFF & n),
		byte(0xFF & (n >> 8)),
		byte(0xFF & (n >> 16)),
		byte(0xFF & (n >> 24)),
	}); err != nil {
		panic(ioError{err})
	}
}

func readInt(r io.Reader) int {
	var bs [4]byte
	if _, err := io.ReadFull(r, bs[:]); err != nil {
		panic(ioError{err})
	}
	var i int32
	i |= int32(bs[0]) << 0
	i |= int32(bs[1]) << 8
	i |= int32(bs[2]) << 16
	i |= int32(bs[3]) << 24
	return int(i)
}

func writeString(w io.Writer, s string) {
	writeInt(w, len(s))
	if _, err := io.WriteString(w, s); err != nil {
		panic(ioError{err})
	}
}

func readString(r io.Reader) string {
	n := readInt(r)
	bs := make([]byte, n)
	if _, err := io.ReadFull(r, bs); err != nil {
		panic(ioError{err})
	}
	return string(bs)
}
