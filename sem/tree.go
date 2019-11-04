// Package sem contains a semantic checker and
// a type-checked, linked representation of the source.
package sem

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/eaburns/pea/ast"
)

// A Mod is a module: the unit of compilation.
type Mod struct {
	AST  *ast.Mod
	Name string
	Defs []Def
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
	AST  *ast.Val
	Priv bool
	Mod  string
	Var  Var
	Init []Stmt

	Locals []*Var
}

func (n *Val) ast() ast.Node { return n.AST }
func (n *Val) priv() bool    { return n.Priv }
func (n *Val) kind() string  { return "value" }

func (n *Val) name() string {
	if n.Mod == "" {
		return n.Var.Name
	}
	return n.Mod + " " + n.Var.Name
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
	Insts  []*Fun
	Priv   bool
	Mod    string
	Recv   *Recv
	TParms []TypeVar
	TArgs  []TypeName
	Sig    FunSig
	// Stmts are the body of the function or method.
	// If Stmts==nil, this is a declaration only;
	// for a function or method definition with no body
	// Stmts will be non-nil with length 0.
	Stmts []Stmt

	Locals []*Var
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
	case n.Mod == "" && n.Recv == nil:
		return n.Sig.Sel
	case n.Mod == "" && n.Recv != nil:
		return fmt.Sprintf("(%d)%s %s", n.Recv.Arity, n.Recv.Name, n.Sig.Sel)
	case n.Mod != "" && n.Recv == nil:
		return n.Mod + " " + n.Sig.Sel
	default:
		return fmt.Sprintf("%s (%d)%s %s", n.Mod, n.Recv.Arity, n.Recv.Name, n.Sig.Sel)
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

// ID returns a user-readable type identifier that includes
// the module if not the current module, name, and arity if non-zero.
func (n *Recv) name() string {
	switch {
	case n.Mod == "" && n.Arity == 0:
		return n.Name
	case n.Mod == "" && n.Arity > 0:
		return fmt.Sprintf("(%d)%s", n.Arity, n.Name)
	case n.Mod != "" && n.Arity == 0:
		return n.Mod + " " + n.Name
	default:
		return fmt.Sprintf("%s (%d)%s", n.Mod, n.Arity, n.Name)
	}
}

// A FunSig is the signature of a function.
type FunSig struct {
	AST   *ast.FunSig
	Sel   string
	Parms []Var // types cannot be nil
	Ret   *TypeName
}

func (n *FunSig) ast() ast.Node { return n.AST }

// A Type defines a type.
type Type struct {
	AST ast.Node // *ast.Type or *ast.Var
	// Def is the original definition.
	// If the type is not an instance, Def points to itself.
	// If the type is an instance, Def points to the non-instantiated *Type.
	Def *Type
	// Insts are all instances of this type.
	Insts []*Type
	Priv  bool
	Mod   string
	Arity int
	Name  string

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
}

func (n *Type) ast() ast.Node { return n.AST }

func (n *Type) priv() bool { return n.Priv }

func (n *Type) kind() string { return "type" }

func (n *Type) name() string {
	switch {
	case n.Mod == "" && n.Arity == 0:
		return n.Name
	case n.Mod == "" && n.Arity > 0:
		return fmt.Sprintf("(%d)%s", n.Arity, n.Name)
	case n.Mod != "" && n.Arity == 0:
		return n.Mod + " " + n.Name
	default:
		return fmt.Sprintf("%s (%d)%s", n.Mod, n.Arity, n.Name)
	}
}

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

// ID returns a user-readable type identifier that includes
// the module if not the current module, name, and arity if non-zero.
func (n *TypeName) name() string {
	switch arity := len(n.Args); {
	case n.Mod == "" && arity == 0:
		return n.Name
	case n.Mod == "" && arity > 0:
		return fmt.Sprintf("(%d)%s", arity, n.Name)
	case n.Mod != "" && arity == 0:
		return n.Mod + " " + n.Name
	default:
		return fmt.Sprintf("%s (%d)%s", n.Mod, arity, n.Name)
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
	AST    *ast.Var
	Name   string
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
	AST *ast.Ret
	Val Expr
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

	// Ref is non-0 for a reference conversion.
	// A negative value is the number of references to remove,
	// and a positive value is the number of references to add.
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
func (n *Call) Type() *Type   { return n.typ }

// A Msg is a message, sent to a value.
type Msg struct {
	// AST is *ast.Msg or *ast.Ident for unary function call.
	AST  ast.Node
	Mod  string
	Sel  string
	Args []Expr

	Fun *Fun
}

func (n Msg) ast() ast.Node { return n.AST }

func (n Msg) name() string {
	if n.Mod == "" {
		return n.Sel
	}
	return n.Mod + " " + n.Sel
}

// A Ctor type constructor literal.
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

	Locals []*Var
	typ    *Type
}

func (n *Block) ast() ast.Node { return n.AST }
func (n *Block) Type() *Type   { return n.typ }

// An Ident is a variable name as an expression.
type Ident struct {
	AST  *ast.Ident
	Text string
	Var  *Var
}

func (n *Ident) ast() ast.Node { return n.AST }

func (n *Ident) Type() *Type {
	if n.Var == nil {
		return nil
	}
	return n.Var.typ
}

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
