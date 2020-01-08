// Package types does type checking and semantic analysis, and
// builds a type-checked, linked representation of the source.
package types

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/eaburns/pea/ast"
)

// A Mod is a module: the unit of compilation.
type Mod struct {
	AST  *ast.Mod
	Path string
	Defs []Def

	// SortedVals contains all val defs, topologically sorted
	// such that values depending on other values
	// are later in the slice than their dependencies.
	SortedVals []*Val

	// IntType is a pointer to the Int type.
	IntType *Type
}

// A Node is a node of the AST with location information.
type Node interface {
	// ast returns the AST node corresponding to the type-checked node.
	// ast may return nil for nodes within built-in and imported definitions.
	ast() ast.Node
}

// A Def is a module-level definition.
type Def interface {
	Node

	// priv is whether the definition is private.
	priv() bool

	// kind is the name of the definition kind: type, method, function, value.
	kind() string

	// name returns the name of the definition.
	// The name is intended for error messages
	// and for debugging.
	name() string

	// String returns a human-readable string representation
	// of the definition's signature.
	String() string
}

// A Val is a module-level value definition.
type Val struct {
	AST     *ast.Val
	Priv    bool
	ModPath string
	Var     Var
	Init    []Stmt

	Locals []*Var
}

func (n *Val) ast() ast.Node { return n.AST }
func (n *Val) priv() bool    { return n.Priv }
func (n *Val) kind() string  { return "value" }

func (n *Val) name() string {
	if n.ModPath == "" {
		return n.Var.Name
	}
	return modName(n.ModPath) + " " + n.Var.Name
}

// A Fun is a function or method definition.
type Fun struct {
	// AST is one of:
	// 	*ast.Fun for a function or method defintion
	// 	*ast.FunSig for a virtual function definition
	// 	*ast.Type for a case-method definition
	AST ast.Node
	// Def is the original definition.
	// If the fun is not an instance, Def points to itself.
	// If the fun is an instance, Def points to the non-instantiated *Fun.
	Def *Fun
	// Insts are all instances of this fun.
	Insts   []*Fun
	Priv    bool
	ModPath string
	Recv    *Recv
	TParms  []TypeVar
	TArgs   []TypeName
	Sig     FunSig

	Locals []*Var

	// Stmts and BuiltIn are mutually exclusive.
	// They cannot both be non-zero at the same time.
	// If Stmts==nil and BuiltIn==0,
	// then this is a declaration.
	// Note that this differs from the case where
	// Stmts!=nil, len(Stmts)==0, and BuiltIn==0,
	// which represents a definition with no statements.

	// Stmts, if non-nil, will always end with a *Ret.
	// If the return type of the Fun is specified,
	// it is an error for the last statement to not be an explicit return.
	// If the return type of the Fun was not specified,
	// a return of the Nil literal, {}, is added.
	Stmts []Stmt

	BuiltIn BuiltInMeth
}

func (n *Fun) ast() ast.Node { return n.AST }

func (n *Fun) priv() bool { return n.Priv }

func (n *Fun) kind() string {
	if n.Recv != nil {
		return "method"
	}
	return "function"
}

func (n *Fun) name() string {
	switch {
	case n.ModPath == "" && n.Recv == nil:
		return n.Sig.Sel
	case n.ModPath == "" && n.Recv != nil:
		return fmt.Sprintf("(%d)%s %s", n.Recv.Arity, n.Recv.Name, n.Sig.Sel)
	case n.ModPath != "" && n.Recv == nil:
		return modName(n.ModPath) + " " + n.Sig.Sel
	default:
		return fmt.Sprintf("%s (%d)%s %s", modName(n.ModPath), n.Recv.Arity, n.Recv.Name, n.Sig.Sel)
	}
}

// Recv is a method receiver.
type Recv struct {
	AST   *ast.Recv
	Parms []TypeVar
	Args  []TypeName
	Mod   string
	Arity int
	Name  string

	Type *Type
}

func (n *Recv) ast() ast.Node { return n.AST }

// A FunSig is the signature of a function.
type FunSig struct {
	AST   *ast.FunSig
	Sel   string
	Parms []Var // types cannot be nil
	// Ret is the explicitly specified return type or nil.
	Ret *TypeName
	typ *Type
}

func (n *FunSig) ast() ast.Node { return n.AST }

// Type returns the return type.
// If the return type was not explicitly specified;
// the return type is the Nil type.
func (n *FunSig) Type() *Type { return n.typ }

// A Type defines a type.
type Type struct {
	AST ast.Node // *ast.Type, *ast.Var, or *ast.Block
	// Def is the original definition.
	// If the type is not an instance, Def points to itself.
	// If the type is an instance, Def points to the non-instantiated *Type.
	Def *Type
	// Insts are all instances of this type.
	Insts   []*Type
	Priv    bool
	ModPath string
	Arity   int
	Name    string

	Parms []TypeVar
	Args  []TypeName // what is subbed for Parms

	// Alias, Fields, Cases, and Virts
	// are mutually exclusive.
	// If any one is non-nil, the others are nil.

	// Var is non-nil for a type variable.
	Var *TypeVar

	// Alias is non-nil for a type Alias.
	Alias *TypeName

	// Fields is non-nil for an And type.
	Fields []Var // types cannot be nil

	// Cases is non-nil for an Or type.
	Cases []Var // types can be nil

	// Virts is non-nil for a Virtual type.
	Virts []FunSig

	// BuiltIn is non-zero for a built-in type.
	BuiltIn BuiltInType

	// refDef is the definition of the built-in & type.
	// It is used to implement the Ref() method,
	// since Ref() may create a new *Type,
	// and we need to make sure it's properly memoized.
	refDef *Type

	// tagType is the type of the or-type tag if this is an or-type,
	// otherwise it is nil.
	tagType *Type
}

func (n *Type) ast() ast.Node { return n.AST }

func (n *Type) priv() bool { return n.Priv }

func (n *Type) kind() string { return "type" }

func (n *Type) name() string {
	switch {
	case n.ModPath == "" && n.Arity == 0:
		return n.Name
	case n.ModPath == "" && n.Arity > 0:
		return fmt.Sprintf("(%d)%s", n.Arity, n.Name)
	case n.ModPath != "" && n.Arity == 0:
		return modName(n.ModPath) + " " + n.Name
	default:
		return fmt.Sprintf("%s (%d)%s", modName(n.ModPath), n.Arity, n.Name)
	}
}

// Ref returns the reference type for this type.
func (n *Type) Ref() *Type {
	for _, ref := range n.refDef.Insts {
		if ref.Args[0].Type == n {
			return ref
		}
	}
	ref := *n.refDef
	ref.Args = []TypeName{*makeTypeName(n)}
	ref.Insts = nil
	ref.Parms = []TypeVar{ref.Parms[0]}
	typ := &Type{
		Name:   ref.Parms[0].Name,
		Var:    &ref.Parms[0],
		refDef: n.refDef,
	}
	typ.Def = typ
	ref.Parms[0].Type = typ
	n.refDef.Insts = append(n.refDef.Insts, &ref)
	return &ref
}

// Tag returns an integer type big enough to hold the tag of an or-type
// or nil if this type is not an or-type.
func (n *Type) Tag() *Type { return n.tagType }

// A TypeName is the name of a concrete type.
type TypeName struct {
	// AST is one of:
	// *ast.TypeName if from an original type name.
	// *ast.Recv if the self variable type of a method.
	// *ast.Block if the inferred type of a block result.
	// *ast.Var if the inferred type of a block parameter.
	// *ast.TypeVar if a type variable definition.
	AST  ast.Node
	Mod  string
	Name string
	Args []TypeName

	Type *Type
}

func (n TypeName) ast() ast.Node { return n.AST }

// name returns a user-readable type identifier that includes
// the module if not the current module, name, and arity if non-zero.
func (n *TypeName) name() string {
	switch arity := len(n.Args); {
	case n.Mod == "" && arity == 0:
		return n.Name
	case n.Mod == "" && arity > 0:
		return fmt.Sprintf("(%d)%s", arity, n.Name)
	case n.Mod != "" && arity == 0:
		return modName(n.Mod) + " " + n.Name
	default:
		return fmt.Sprintf("%s (%d)%s", modName(n.Mod), arity, n.Name)
	}
}

// A Var is a name and a type.
// Var are used in several AST nodes.
// In some cases, the type must be non-nil.
// In others, the type may be nil.
type Var struct {
	AST  *ast.Var
	Name string

	// TypeName is non-nil if explicit.
	TypeName *TypeName

	// At most one of the following is non-nil.
	Val     *Val    // a module-level Val; Index is unused.
	FunParm *Fun    // a function parm; Index is the Parms index.
	BlkParm *Block  // a block parm; Index is the Parms index.
	Local   *[]*Var // a local variable; Index is the index.
	Field   *Type   // an And-type field; Index is the Fields index.
	Case    *Type   // an Or-type case; Index is the Case index.
	// Index is used as described above.
	Index int

	typ *Type
}

func (n *Var) ast() ast.Node { return n.AST }
func (n *Var) Type() *Type   { return n.typ }

func (n *Var) isSelf() bool {
	return n.FunParm != nil && n.FunParm.Recv != nil && n.Index == 0
}

// A TypeVar is a type variable.
type TypeVar struct {
	AST  *ast.Var
	Name string
	// ID is a unique identifier for this type variable definition.
	// Multiple type variables in a module may have the same name,
	// but each will have unique ID value.
	ID     int
	Ifaces []TypeName

	Type *Type
}

func (n TypeVar) ast() ast.Node { return n.AST }

// A Stmt is a statement.
type Stmt interface {
	Node
}

// A Ret is a return statement.
type Ret struct {
	AST  *ast.Ret
	Expr Expr
}

func (n *Ret) ast() ast.Node { return n.AST }

// An Assign is an assignment statement.
type Assign struct {
	AST  *ast.Assign
	Var  *Var
	Expr Expr
}

func (n *Assign) ast() ast.Node { return n.AST }

// An Expr is an expression
type Expr interface {
	Node
	Type() *Type
}

// A Convert is an expression node added by the type checker
// to represent a type conversion.
type Convert struct {
	// Expr is the converted expression; it is never nil.
	Expr Expr

	// One of the following is non-zero:

	// Ref is either -1 to remove a reference or 1 to add one.
	// 0 means this is a virtual conversion.
	Ref int

	// Virts is non-nil for a virtual conversion;
	// Virts[i] is the function implementing typ.Virts[i].
	Virts []*Fun

	typ *Type
}

func (n *Convert) ast() ast.Node { return n.Expr.ast() }
func (n *Convert) Type() *Type   { return n.typ }

// A Call is a method call or a cascade.
type Call struct {
	// AST is *ast.Call or *ast.Ident if in-module unary function call.
	AST ast.Node
	// Recv is nil for function calls.
	// For methods the receiver is always
	// a reference to some non-reference type.
	Recv Expr
	Msgs []Msg

	typ *Type
}

func (n *Call) ast() ast.Node { return n.AST }

// Type returns the return type of the last message in the Call.
// If the last message in the call has no return,
// then Type() is the Nil type.
func (n *Call) Type() *Type { return n.typ }

// A Msg is a message, sent to a value.
type Msg struct {
	// AST is *ast.Msg or *ast.Ident for unary function call.
	AST  ast.Node
	Mod  string
	Sel  string
	Args []Expr

	Fun *Fun
	typ *Type
}

func (n Msg) ast() ast.Node { return n.AST }

// Type returns the return type of the message.
// If the function has no return, then Type() is the Nil type.
func (n *Msg) Type() *Type { return n.typ }

func (n Msg) name() string {
	if n.Mod == "" {
		return n.Sel
	}
	return n.Mod + " " + n.Sel
}

// A Ctor type constructor literal.
// Ctor literals construct Arrays types, And types, and Or types.
//
// For Array types, Args correspond to the successive
// array elements starting from element 0.
//
// For And types, the Args correspond to the Fields,
// with Args[i] corresponding to Fields[i].
//
// For Or types, the case constructed is given by Cases[*Ctor.Case].
// If the case has a type, len(Args)==1 and the value is Args[0].
// If the case has no type, len(Args)==0.
type Ctor struct {
	AST  *ast.Ctor
	Args []Expr

	// Case is non-nil if this is an or-type constructor.
	// It is an index into the typ.Cases array.
	Case *int

	typ *Type
}

func (n *Ctor) ast() ast.Node { return n.AST }
func (n *Ctor) Type() *Type   { return n.typ }

// A Block is a block literal.
type Block struct {
	AST   *ast.Block
	Parms []Var // if type is nil, it must be inferred
	Stmts []Stmt

	// Captures are local variables or parameters
	// defined outside the block, used by the block.
	// For fields, the self parameter is captured.
	Captures []*Var
	Locals   []*Var

	// BlockType is a unique, underlying type for a block literal.
	// It is an and-type with a field for each of Captures.
	// This differs from Type() which returns the Fun type of the block.
	BlockType *Type

	typ *Type
}

func (n *Block) ast() ast.Node { return n.AST }
func (n *Block) Type() *Type   { return n.typ }

// An Ident is a variable name as an expression.
type Ident struct {
	AST  *ast.Ident
	Text string

	// The Var is the variable referenced by this identifier.
	// The Var will have one of the following fields non-nil:
	// 	Val if this is a module-level variable.
	// 	FunParm if this is a function or method parameter.
	// 	BlkParm if this is a block parameter.
	// 	Local if this is a local variable.
	// 	Field if this is a method receiver field.
	// The Case field will never be non-nil.
	Var *Var

	// Capture indicates that this identifier
	// is a capture of variable of a block literal.
	Capture bool

	typ *Type
}

func (n *Ident) ast() ast.Node { return n.AST }
func (n *Ident) Type() *Type   { return n.typ }

// An Int is an integer literal.
type Int struct {
	AST ast.Expr // *ast.Int, *ast.Float, or *ast.Rune
	Val *big.Int
	typ *Type
}

func (n *Int) ast() ast.Node { return n.AST }
func (n *Int) Type() *Type   { return n.typ }

func (n *Int) PrettyPrint() string {
	if _, ok := n.AST.(*ast.Rune); ok {
		return "Int{Val: " + strconv.QuoteRune(rune(n.Val.Int64())) + "}"
	}
	return "Int{Val: " + n.Val.String() + "}"
}

// A Float is a floating point literal.
type Float struct {
	AST ast.Expr // *ast.Float or *ast.Int
	Val *big.Float
	typ *Type
}

func (n *Float) ast() ast.Node       { return n.AST }
func (n *Float) PrettyPrint() string { return "Int{Val: " + n.Val.String() + "}" }
func (n *Float) Type() *Type         { return n.typ }

// A String is a string literal.
type String struct {
	AST  *ast.String
	Data string
	typ  *Type
}

func (n String) ast() ast.Node { return n.AST }
func (n String) Type() *Type   { return n.typ }
