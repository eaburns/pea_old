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
	// Vars are the module-level variables,
	// in topological order, dependencies first..
	Vars []*Var
	// Init is the initialization function that initializes Vars.
	Init  *Fun
	Funs  []*Fun
	NDefs int

	Mod *types.Mod
}

// A String is the data of a string constant.
type String struct {
	Mod *Mod
	// N is unique among Mod-level defs.
	N    int
	Data string
}

// A Var is a module-level variable.
type Var struct {
	Mod *Mod
	// N is unique among Mod-level defs.
	N    int
	Init *Fun

	Val *types.Val
}

// A Fun is a code block.
type Fun struct {
	Mod *Mod
	// N is unique among Mod-level defs.
	N         int
	NVals     int
	Parms     []*Parm
	Ret       *Parm // return parameter
	BBlks     []*BBlk
	CanInline bool
	CanFarRet bool

	Fun   *types.Fun
	Block *types.Block
	Val   *types.Val
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
	// Self indicates that this is the self parameter of a method.
	Self bool

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

func (n *BBlk) Out() []*BBlk {
	if len(n.Stmts) == 0 {
		return nil
	}
	term, ok := n.Stmts[len(n.Stmts)-1].(Term)
	if !ok {
		return nil
	}
	return term.Out()
}

func (n *BBlk) addIn(in *BBlk) {
	for _, i := range n.In {
		if i == in {
			return
		}
	}
	n.In = append(n.In, in)
}

func (n *BBlk) rmIn(in *BBlk) {
	var i int
	for _, b := range n.In {
		if b != in {
			n.In[i] = b
			i++
		}
	}
	n.In = n.In[:i]
}

// A Stmt is an instruction that does not produce a value.
type Stmt interface {
	Uses() []Val
	buildString(*strings.Builder) *strings.Builder

	// delete marks the statement as deleted.
	delete()

	// deleted returns whether this statement is deleted.
	deleted() bool

	// storesTo returns whether this is an initialization
	// of the variable addressed by the Val.
	//
	// This function only considers stores
	// that can have no other side effect besides the storing.
	// For example, Calls and VirtCalls can store to their arguments,
	// but those are not considered stores in this sense.
	storesTo(Val) bool

	// subVals substitutes values of the statement
	// that are keys of the map for their values.
	// subVals properly maintains the users slices.
	subVals(valMap)

	// shallowCopy returns a shallow copy of the Stmt.
	// The Stmt is copied along with any slices in it,
	// but the Vals referenced by it still point to the original Vals.
	// The val.users is not copied for Vals;
	// the slice is shared with that of the receiver
	// and should be reset by the caller.
	shallowCopy() Stmt

	// bugs returns a strings describing errors in the statement.
	// An empty return indicates no errors.
	// These are indicative of bugs like type mismatches
	// or storing to a non-address, and so forth.
	bugs() string
}

type stmt struct {
	del bool
}

func (*stmt) Uses() []Val     { return nil }
func (s *stmt) delete()       { s.del = true }
func (s *stmt) deleted() bool { return s.del }
func (*stmt) bugs() string    { return "" }

// A Comment is a no-op statement that adds a note to the output.
type Comment struct {
	stmt
	Text string
}

// Store is a Stmt stores a value to a location specified by address.
type Store struct {
	stmt
	// Dst is the address to which the value is stored.
	Dst Val
	Val Val

	Assign *types.Assign
}

func (n *Store) Uses() []Val { return []Val{n.Dst, n.Val} }

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
	stmt
	// Dst is the address to which the value is copied.
	Dst Val
	Src Val

	Assign *types.Assign
}

func (n *Copy) Uses() []Val { return []Val{n.Dst, n.Src} }

// MakeArray initializes an array.
// It assumes that Dst holds the size and data address,
// and that the data address is set by MakeArray
// to a newly allocated object of len(Args) elements.
type MakeArray struct {
	stmt
	Dst Val
	// Args are the array elements.
	// If the array element type is not a SimpleType,
	// Args will be references to the values,
	// which MakeArray must copy into the array.
	Args []Val

	Ctor *types.Ctor
}

func (n *MakeArray) Uses() []Val { return append(n.Args, n.Dst) }

// MakeSlice initializes an array by slicing another array.
type MakeSlice struct {
	stmt
	Dst  Val
	Ary  Val
	From Val
	To   Val

	Msg *types.Msg
}

func (n *MakeSlice) Uses() []Val { return []Val{n.Ary, n.From, n.To, n.Dst} }

// MakeString initializes a string literal.
//
// A String is a pair <size (int), address>,
// where the address is the address of size bytes.
type MakeString struct {
	stmt
	Dst  Val
	Data *String

	String *types.String
}

func (n *MakeString) Uses() []Val { return []Val{n.Dst} }

// MakeAnd initializes an and-type.
type MakeAnd struct {
	stmt
	Dst Val
	// Fields are the values for each field.
	// If field index i has an EmptyType, Fields[i]==nil.
	// If the field type is not a SimpleType,
	// the value in Fields is a reference to the value
	// which must be copied by MakeAnd.
	//
	// Any fields not present in the types.Type.Field slice
	// are assumed to be references, and needn't be copied.
	Fields []Val

	// Ctor is non-nil if this originated from a constructor.
	Ctor *types.Ctor
	// Block is non-nil if this originated from a block literal.
	Block *types.Block
	// BlockFun is the *Fun of the block.
	BlockFun *Fun
}

func (n *MakeAnd) Uses() []Val {
	uses := make([]Val, 0, len(n.Fields)+1)
	uses = append(uses, n.Dst)
	for _, f := range n.Fields {
		if f != nil {
			uses = append(uses, f)
		}
	}
	return uses
}

// MakeOr initializes an or-type.
type MakeOr struct {
	stmt
	Dst  Val
	Case int
	// Val is the value or nil for a value-less case.
	// If the case type is not a simple type,
	// Val is a reference to the value,
	// which must be copied by MakeOr.
	Val Val

	Ctor *types.Ctor
}

func (n *MakeOr) Uses() []Val {
	if n.Val == nil {
		return []Val{n.Dst}
	}
	return []Val{n.Dst, n.Val}
}

// MakeVirt initializes a virtual type.
type MakeVirt struct {
	stmt
	Dst   Val
	Obj   Val
	Virts []*Fun

	Convert *types.Convert
}

func (n *MakeVirt) Uses() []Val { return []Val{n.Dst, n.Obj} }

// Call is a static function call.
type Call struct {
	stmt
	Fun  *Fun
	Args []Val

	Msg *types.Msg
}

func (n *Call) Uses() []Val { return n.Args }

func (n Call) shallowCopy() Stmt {
	n.Args = append([]Val{}, n.Args...)
	return &n
}

// VirtCall is a virtual function call.
type VirtCall struct {
	stmt
	// Self is the receiver of the call.
	Self Val
	// Index is the index of the virtual method, 0-indexed.
	Index int
	Args  []Val

	Msg *types.Msg
}

func (n *VirtCall) Uses() []Val { return n.Args }

// A Term is a terminal statement.
type Term interface {
	Stmt
	Out() []*BBlk

	// subBBlk does not update the BBlk.In slice.
	// Use the subBblks([]*BBlk, bblkMap) function
	// to ensure that the BBlk.In slices are propertyl updated.
	subBBlk(bblkMap)
}

// Ret is a Term statement that returns from the current Fun.
type Ret struct {
	stmt
	Ret *types.Ret
	// Far indicates whether this is a far return.
	Far bool
}

func (*Ret) Out() []*BBlk { return nil }

// Jmp is a Term that changes control to another BBlk.
type Jmp struct {
	stmt
	Dst *BBlk
}

func (n *Jmp) Out() []*BBlk { return []*BBlk{n.Dst} }

// Switch is a Term that transfers control to one of several BBlks.
// The Val is either an address or an integer type.
// If it is an Addr, there are 2 Dsts.
// Dsts[0] corresponds to the 0 address, and Dsts[1] to non-zero.
// If it is an integer type, there are N Dsts.
// Dsts[i], 0 ≤ i < N corresponds to Val=i.
type Switch struct {
	stmt
	Val    Val
	Dsts   []*BBlk
	OrType *types.Type

	Msg *types.Msg
}

func (n *Switch) Uses() []Val  { return []Val{n.Val} }
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
	stmt
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

func (v *val) rmUser(s Stmt) {
	var i int
	for _, u := range v.users {
		if u != s {
			v.users[i] = u
			i++
		}
	}
	v.users = v.users[:i]
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

// FloatLit is an floating-point literal.
type FloatLit struct {
	val
	Val *big.Float

	Float *types.Float
}

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

// Load loads the value at an address.
// The type of a load is always a simple type.
type Load struct {
	val
	Src Val

	Convert *types.Convert
}

func (n *Load) Uses() []Val { return []Val{n.Src} }

// Alloc is an address of a newly allocated location of a given type.
type Alloc struct {
	val
	// Stack is true if the allocation can be on the stack.
	Stack bool

	Var *types.Var
}

// Arg is an argument to the current function.
type Arg struct {
	val
	Parm *Parm

	Ident *types.Ident
}

// Global is the address of a module-level variable.
type Global struct {
	val
	Val *types.Val // non-nil
}

// Index is the address of an element of an array.
type Index struct {
	val
	// Ary is the address of the array.
	Ary   Val
	Index Val

	Msg *types.Msg
}

func (n *Index) Uses() []Val { return []Val{n.Ary, n.Index} }

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
