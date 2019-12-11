// Package basic has an intermediate representation
// that tries to strike a balance between
// being high-level enough to easily convertable
// to a high-level language like Go,
// and being low-level enough to be easily convertable to LLVM.
//
// The representation of function bodies is a naive SSA form
// which uses explicit allocations, loads, and stores for local variables.
// A light optimization pass can conservatively lift locals to registers,
// but only in cases where φ-nodes need not be inserted.
//
// Types
//
// Empty types are not allocated, copied, or stored in any way.
// Empty types are types with 0 size, which includes the built-in Nil type
// and And-types that have no fields or only empty type fields.
// All other types are non-empty.
//
// Only simple types are ever loaded into registers.
// All non-simple types are referred by reference, not value,
// even if they were values in the original source.
// Explicit copies are inserted to implement
// the value semantics of non-simple types.
//
// Simple types are any reference type, &, and the  built-in int and float types.
//
// Or-types that consist only of non-typed cases may be converted to ints.
// In that case are considered simple types too.
//
// Or-types that consist of exactly two cases where one is non-typed
// and the other is a reference type may be converted to a nil-able & type.
// In that case it would be considered a simple type too.
//
// Arrays, Strings, and Virtual types are treated at a somewhat high-level,
// since these types are expected to have first-class counterparts
// in high-level target languages, and they are simple enough to implement
// in lower-level targets like LLVM.
//
// Array and String types
//
// Arrays are assumed to be implemented as an and-type with two fields.
// The first field is an Int that holds the array size.
// The second is the address of size number of elements of the element type.
// Strings are assumed to be implemented as byte arrays.
//
// The two array fields are only used indirectly.
// The first is a read-only field that can be read with Op.Code==ArraySize,
// and used by the implementation of MakeArray, MakeString, and MakeSlice.
// The second field is never accessed, but only used by the implementation
// of MakeArray, MakeSring, MakeSlice, and Index.
// MakeArray allocates the array data addressed by the second field.
// MakeString sets the data address to that of a read-only string constant.
// MakeSlice sets the data address to that of an existing array (plus offset).
// Index computes an address using the data address as the base.
//
// The reason these are treated somewhat specially
// is because high-level languages have first-class Arrays and Strings.
// This implementation should make it easy to implement
// these types using the first-class types of the target language,
// but should still be simple enough for a lower-level,
// LLVM-like target too.
//
// Virtual types
//
// Virtual types are assumed to be implemented as an and-type
// with one field that is a pointer to the underlying, virtualized object
// and a subsequent function-pointer field for each of the virtual methods.
//
// These fields cannot be accessed directly;
// they can be se with MakeVirt and called with VirtCall.
//
// Block literal types
//
// Block literals are Fun virtual types.
// However, each block literal also has its own underlying and-type
// which holds pointers to all variables captured by the block literal.
// If the function containing the block definition has a non-empty return type,
// then there is a field after all of the capture fields
// that has a pointer to the return value location of the containing function.
// This is used to implement far returns.
package basic

import (
	"math/big"
	"strings"

	"github.com/eaburns/pea/types"
)

// A Mod is a module.
type Mod struct {
	Strings []*String
	Funs    []*Fun
	NDefs   int

	Mod *types.Mod
}

// A String is the data of a string constant.
type String struct {
	// N is unique among Mod-level defs.
	N    int
	Data string
}

// A Fun is a code block.
type Fun struct {
	// N is unique among Mod-level defs.
	N     int
	Mod   *Mod
	NVals int
	Parms []*Parm
	Ret   *Parm // return parameter
	BBlks []*BBlk

	Fun   *types.Fun
	Block *types.Block
}

// A Parm is a function/block parameter.
type Parm struct {
	// N is the index into the Fun's Parms.
	N int
	// Type is always a SimpleType.
	Type *types.Type
	// Value indicates whether this parameter is "pass by value".
	// If true, Type will be a &, but the caller is intended to make a copy
	// and pass the address of the copy.
	Value bool

	// Var is nil for the Ret parm or block literal self parm.
	Var *types.Var
}

// A BBlk is a basic block.
type BBlk struct {
	// N is unique within the containing Fun.
	N     int
	Stmts []Stmt
	In    []*BBlk
}

func (b *BBlk) Out() []*BBlk {
	if len(b.Stmts) == 0 {
		return nil
	}
	term, ok := b.Stmts[len(b.Stmts)-1].(Term)
	if !ok {
		return nil
	}
	return term.Out()
}

func (b *BBlk) addIn(in *BBlk) {
	for _, i := range b.In {
		if i == in {
			return
		}
	}
	b.In = append(b.In, in)
}

// A Stmt is an instruction that does not produce a value.
type Stmt interface {
	Uses() []Val
	buildString(*strings.Builder) *strings.Builder

	// sub substitutes values of the statement
	// that are keys of the map for their values.
	sub(valMap)

	// bugs returns a strings describing errors in the statement.
	// An empty return indicates no errors.
	// These are indicative of bugs like type mismatches
	// or storing to a non-address, and so forth.
	bugs() string
}

type valMap []Val

func makeValMap(n int) valMap {
	return valMap(make([]Val, n))
}

func (s valMap) add(key, val Val) {
	s[key.Num()] = val
}

func (s valMap) get(v Val) Val {
	u := s[v.Num()]
	if u == nil {
		return v
	}
	u = s.get(u)
	s[v.Num()] = u
	return u
}

// A Comment is a no-op statement that adds a note to the output.
type Comment struct {
	Text string
}

func (n *Comment) Uses() []Val    { return nil }
func (n *Comment) sub(sub valMap) {}

// Store is a Stmt stores a value to a location specified by address.
type Store struct {
	// Dst is the address to which the value is stored.
	Dst Val
	Val Val

	Assign *types.Assign
}

func (n *Store) Uses() []Val { return []Val{n.Dst, n.Val} }

func (n *Store) sub(sub valMap) {
	n.Dst = sub.get(n.Dst)
	n.Val = sub.get(n.Val)
}

// Copy is a Stmt that copies a composite value
// to a location specified by address.
//
// Copy is like Store, but both arguments are given by addresses.
// The amount of data copied is given by the size of
// Dst.Type().Args[0].Type.
//
// For Array and String types, Copy is a shallow copy that copies
// the size and data address, but not the data itself.
type Copy struct {
	// Dst is the address to which the value is copied.
	Dst Val
	Src Val

	Assign *types.Assign
}

func (n *Copy) Uses() []Val { return []Val{n.Dst, n.Src} }

func (n *Copy) sub(sub valMap) {
	n.Dst = sub.get(n.Dst)
	n.Src = sub.get(n.Src)
}

// MakeArray initializes an array.
// It assumes that Dst holds the size and data address,
// and that the data address is set by MakeArray
// to a newly allocated object of len(Args) elements.
type MakeArray struct {
	Dst  Val
	Args []Val

	Ctor *types.Ctor
}

func (n *MakeArray) Uses() []Val { return append(n.Args, n.Dst) }

func (n *MakeArray) sub(sub valMap) {
	n.Dst = sub.get(n.Dst)
	for i := range n.Args {
		n.Args[i] = sub.get(n.Args[i])
	}
}

// MakeSlice initializes an array by slicing another array.
type MakeSlice struct {
	Dst  Val
	Ary  Val
	From Val
	To   Val

	Msg *types.Msg
}

func (n *MakeSlice) Uses() []Val { return []Val{n.Ary, n.From, n.To, n.Dst} }

func (n *MakeSlice) sub(sub valMap) {
	n.Dst = sub.get(n.Dst)
	n.Ary = sub.get(n.Ary)
	n.From = sub.get(n.From)
	n.To = sub.get(n.To)
}

// MakeString initializes a string literal.
//
// A String is a pair <size (int), address>,
// where the address is the address of size bytes.
type MakeString struct {
	Dst  Val
	Data *String

	String *types.String
}

func (n *MakeString) Uses() []Val { return []Val{n.Dst} }

func (n *MakeString) sub(sub valMap) {
	n.Dst = sub.get(n.Dst)
}

// MakeAnd initializes an and-type.
type MakeAnd struct {
	Dst Val
	// Fields are the values for each field.
	// If field index i has an EmptyType, Fields[i]==nil.
	Fields []Val

	// Ctor is non-nil if this originated from a constructor.
	Ctor *types.Ctor
	// Block is non-nil if this originated from a block literal.
	Block *types.Block
}

func (n *MakeAnd) Uses() []Val {
	uses := make([]Val, 0, len(n.Fields))
	for _, f := range n.Fields {
		if f != nil {
			uses = append(uses, f)
		}
	}
	return uses
}

func (n *MakeAnd) sub(sub valMap) {
	n.Dst = sub.get(n.Dst)
	for i := range n.Fields {
		n.Fields[i] = sub.get(n.Fields[i])
	}
}

// MakeOr initializes an or-type.
type MakeOr struct {
	Dst  Val
	Case int
	Val  Val

	Ctor *types.Ctor
}

func (n *MakeOr) Uses() []Val {
	if n.Val == nil {
		return []Val{n.Dst}
	}
	return []Val{n.Dst, n.Val}
}

func (n *MakeOr) sub(sub valMap) {
	n.Dst = sub.get(n.Dst)
	if n.Val != nil {
		n.Val = sub.get(n.Val)
	}
}

// MakeVirt initializes a virtual type.
type MakeVirt struct {
	Dst   Val
	Obj   Val
	Virts []*Fun

	Convert *types.Convert
}

func (n *MakeVirt) Uses() []Val { return []Val{n.Dst, n.Obj} }

func (n *MakeVirt) sub(sub valMap) {
	n.Dst = sub.get(n.Dst)
	n.Obj = sub.get(n.Obj)
}

// Call is a static function call.
type Call struct {
	Fun  *Fun
	Args []Val

	Msg *types.Msg
}

func (n *Call) Uses() []Val { return n.Args }

func (n *Call) sub(sub valMap) {
	for i := range n.Args {
		n.Args[i] = sub.get(n.Args[i])
	}
}

// VirtCall is a virtual function call.
type VirtCall struct {
	// Self is the receiver of the call.
	Self Val
	// Index is the index of the virtual method, 0-indexed.
	Index int
	Args  []Val

	Msg *types.Msg
}

func (n *VirtCall) Uses() []Val { return n.Args }

func (n *VirtCall) sub(sub valMap) {
	n.Self = sub.get(n.Self)
	for i := range n.Args {
		n.Args[i] = sub.get(n.Args[i])
	}
}

// A Term is a terminal statement.
type Term interface {
	Stmt
	Out() []*BBlk
}

// Ret is a Term statement that returns from the current Fun.
type Ret struct {
	Ret *types.Ret
	// Far indicates whether this is a far return.
	Far bool
}

func (*Ret) Uses() []Val      { return nil }
func (n *Ret) sub(sub valMap) {}
func (*Ret) Out() []*BBlk     { return nil }

// Jmp is a Term that changes control to another BBlk.
type Jmp struct {
	Dst *BBlk
}

func (*Jmp) Uses() []Val      { return nil }
func (n *Jmp) sub(sub valMap) {}
func (n *Jmp) Out() []*BBlk   { return []*BBlk{n.Dst} }

// Switch is a Term that transfers control to one of several BBlks.
// The Val is either an address or an integer type.
// If it is an Addr, there are 2 Dsts.
// Dsts[0] corresponds to the 0 address, and Dsts[1] to non-zero.
// If it is an integer type, there are N Dsts.
// Dsts[i], 0 ≤ i < N corresponds to Val=i.
type Switch struct {
	Val    Val
	Dsts   []*BBlk
	OrType *types.Type

	Msg *types.Msg
}

func (n *Switch) Uses() []Val { return []Val{n.Val} }

func (n *Switch) sub(sub valMap) {
	n.Val = sub.get(n.Val)
}

func (n *Switch) Out() []*BBlk { return n.Dsts }

// Val is a value
type Val interface {
	Stmt
	// Num is the Val's unique number.
	Num() int
	// Type returns the Val's type.
	Type() *types.Type
	// Users returns the Stmts that use this Val.
	Users() []Stmt

	value() *val
}

type val struct {
	n     int
	typ   *types.Type
	users []Stmt
}

func newVal(f *Fun, typ *types.Type) val {
	v := val{
		n:   f.NVals,
		typ: typ,
	}
	f.NVals++
	return v
}

func (v *val) Num() int          { return v.n }
func (v *val) Type() *types.Type { return v.typ }
func (v *val) Users() []Stmt     { return v.users }
func (v *val) value() *val       { return v }

func (v *val) addUser(s Stmt) {
	for _, u := range v.users {
		if u == s {
			return
		}
	}
	v.users = append(v.users, s)
}

// IntLit is an integer literal.
type IntLit struct {
	val
	Val *big.Int

	// Int is non-nil if this was an integer literal in the source.
	Int *types.Int
	// Case is non-nil if this is a case tag integer.
	Case *types.Var
}

func (n *IntLit) Uses() []Val    { return nil }
func (n *IntLit) sub(sub valMap) {}

// FloatLit is an floating-point literal.
type FloatLit struct {
	val
	Val *big.Float

	Float *types.Float
}

func (FloatLit) Uses() []Val       { return nil }
func (n *FloatLit) sub(sub valMap) {}

// OpCode are the names of the built-in Ops.
type OpCode int

// The names of built-in operations.
const (
	ArraySizeOp OpCode = iota + 1
	BitwiseAndOp
	BitwiseOrOp
	BitwiseNotOp
	RightShiftOp
	LeftShiftOp
	NegOp
	PlusOp
	MinusOp
	TimesOp
	DivideOp
	ModOp
	EqOp
	NeqOp
	LessOp
	LessEqOp
	GreaterOp
	GreaterEqOp
	NumConvertOp
	UnionTagOp
)

// Op is the result of a built-in operation.
// The arguments and result of an Op are simple types.
type Op struct {
	val
	Code OpCode
	Args []Val

	Msg *types.Msg
}

func (n *Op) Uses() []Val { return n.Args }

func (n *Op) sub(sub valMap) {
	for i := range n.Args {
		n.Args[i] = sub.get(n.Args[i])
	}
}

// Load loads the value at an address.
// The type of a load is always a simple type.
type Load struct {
	val
	Src Val

	Convert *types.Convert
}

func (n *Load) Uses() []Val { return []Val{n.Src} }

func (n *Load) sub(sub valMap) {
	n.Src = sub.get(n.Src)
}

// Alloc is an address of a newly allocated location of a given type.
type Alloc struct {
	val
	// Stack is true if the allocation can be on the stack.
	Stack bool

	Var *types.Var
}

func (*Alloc) Uses() []Val      { return nil }
func (n *Alloc) sub(sub valMap) {}

// Arg is an argument to the current function.
type Arg struct {
	val
	Parm *Parm

	Ident *types.Ident
}

func (*Arg) Uses() []Val      { return nil }
func (n *Arg) sub(sub valMap) {}

// Global is the address of a module-level variable.
type Global struct {
	val
	Val *types.Val // non-nil
}

func (*Global) Uses() []Val      { return nil }
func (n *Global) sub(sub valMap) {}

// Index is the address of an element of an array.
type Index struct {
	val
	// Ary is the address of the array.
	Ary   Val
	Index Val

	Msg *types.Msg
}

func (n *Index) Uses() []Val { return []Val{n.Ary, n.Index} }

func (n *Index) sub(sub valMap) {
	n.Ary = sub.get(n.Ary)
	n.Index = sub.get(n.Index)
}

// Field is the address of an and-type field, an or-type case, or an or-type tag.
type Field struct {
	val
	// Obj is the base address of the object.
	Obj Val

	// Index is the field index.
	Index int

	// Field is non-nil if this is an and-type field.
	Field *types.Var
	// Case is non-nil if this is an or-type case.
	Case *types.Var

	Ident *types.Ident
}

func (n *Field) Uses() []Val { return []Val{n.Obj} }

func (n *Field) sub(sub valMap) {
	n.Obj = sub.get(n.Obj)
}

// EmptyType returns whether the type has zero-size.
func EmptyType(typ *types.Type) bool {
	if len(typ.Fields) > 0 {
		for _, f := range typ.Fields {
			if !EmptyType(f.Type()) {
				return false
			}
		}
		return true
	}
	return len(typ.Fields) == 0 &&
		len(typ.Cases) == 0 &&
		len(typ.Virts) == 0 &&
		(typ.BuiltIn == 0 || typ.BuiltIn == types.NilType)
}

// SimpleType returns whether a type is "simple",
// indicating that it can be held as a Val (register).
func SimpleType(typ *types.Type) bool {
	/* pointerType(typ) */
	return enumType(typ) ||
		typ.BuiltIn != 0 &&
			typ.BuiltIn != types.BoolType &&
			typ.BuiltIn != types.StringType &&
			typ.BuiltIn != types.ArrayType &&
			typ.BuiltIn != types.FunType
}

func pointerType(typ *types.Type) bool {
	if len(typ.Cases) != 2 {
		return false
	}
	return typ.Cases[0].Type() == nil && typ.Cases[1].Type() != nil ||
		typ.Cases[0].Type() != nil && typ.Cases[1].Type() == nil
}

func enumType(typ *types.Type) bool {
	for _, c := range typ.Cases {
		if c.Type() != nil {
			return false
		}
	}
	return len(typ.Cases) > 0
}

func enumTag(cas *types.Var) *big.Int {
	orTyp := cas.Case
	for i := range orTyp.Cases {
		if cas == &orTyp.Cases[i] {
			return big.NewInt(int64(i))
		}
	}
	panic("impossible")
}
