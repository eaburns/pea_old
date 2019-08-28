package types

import (
	"fmt"

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
	// AST returns the AST node corresponding to the type-checked node.
	// AST may return nil for nodes within built-in and imported definitions.
	AST() ast.Node
}

// A Def is a module-level definition.
type Def interface {
	Node

	// type, method, function, value.
	kind() string

	// Priv returns whether the definition is private.
	Priv() bool
}

// A Val is a module-level value definition.
type Val struct {
	ast  *ast.Val
	priv bool
	Name string
	Type *TypeName
	Init []Stmt
}

func (n *Val) AST() ast.Node { return n.ast }
func (n *Val) kind() string  { return "value" }
func (n *Val) Priv() bool    { return n.priv }

// A Fun is a function or method definition.
type Fun struct {
	ast    *ast.Fun
	priv   bool
	Recv   *TypeSig
	TParms []Var // types may be nil
	Sig    FunSig
	Stmts  []Stmt
}

func (n *Fun) AST() ast.Node { return n.ast }

func (n *Fun) kind() string {
	if n.Recv != nil {
		return "method"
	}
	return "function"
}

func (n *Fun) Priv() bool { return n.priv }

// A FunSig is the signature of a function.
type FunSig struct {
	ast   *ast.FunSig
	Sel   string
	Parms []Var // types cannot be nil
	Ret   *TypeName
}

func (n *FunSig) AST() ast.Node { return n.ast }

// A Type defines a type.
type Type struct {
	ast  *ast.Type
	priv bool
	Sig  TypeSig

	// Alias, Fields, Cases, and Virts
	// are mutually exclusive.
	// If any one is non-nil, the others are nil.

	// Alias is non-nil for a type Alias.
	Alias *TypeName

	// Fields is non-nil for an And type.
	Fields []Var // types cannot be nil

	// Cases is non-nil for an Or type.
	Cases []Var // types can be nil

	// Virts is non-nil for a Virtual type.
	Virts []FunSig
}

func (n *Type) AST() ast.Node { return n.ast }
func (n *Type) kind() string  { return "type" }
func (n *Type) Priv() bool    { return n.priv }

// A TypeSig is a type signature, a pattern defining a type or set o types.
type TypeSig struct {
	ast   *ast.TypeSig
	Arity int
	Name  string
	Parms []Var // types may be nil
}

func (n *TypeSig) AST() ast.Node { return n.ast }

func (n *TypeSig) id() string {
	if n.Arity == 0 {
		return n.Name
	}
	return fmt.Sprintf("(%d)%s", n.Arity, n.Name)
}

// A TypeName is the name of a concrete type.
type TypeName struct {
	ast  *ast.TypeName
	Var  bool
	Mod  *Ident
	Name string
	Args []TypeName
}

func (n *TypeName) AST() ast.Node { return n.ast }

// A Var is a name and a type.
// Var are used in several AST nodes.
// In some cases, the type must be non-nil.
// In others, the type may be nil.
type Var struct {
	ast  *ast.Var
	Name string
	Type *TypeName
}

func (n *Var) AST() ast.Node { return n.ast }

// A Stmt is a statement.
type Stmt interface {
	Node
}

// A Ret is a return statement.
type Ret struct {
	ast *ast.Ret
	Val Expr
}

func (n *Ret) AST() ast.Node { return n.ast }

// An Assign is an assignment statement.
type Assign struct {
	ast *ast.Assign
	// Vars are the target of assignment.
	// After type checking, these refer to the defining Param,
	// either a local variable or Fun/Block parameter.
	Vars []*Var // types may be nil before successful Check()
	Val  Expr
}

func (n *Assign) AST() ast.Node { return n.ast }

// An Expr is an expression
type Expr interface {
	Node
}

// A Call is a method call or a cascade.
type Call struct {
	ast  *ast.Call
	Recv Node // Expr, ModName (Ident beginning with '#'), or nil
	Msgs []Msg
}

func (n *Call) AST() ast.Node { return n.ast }

// A Msg is a message, sent to a value.
type Msg struct {
	ast  *ast.Msg
	Sel  string
	Args []Expr
}

func (n *Msg) AST() ast.Node { return n.ast }

// A Ctor type constructor literal.
type Ctor struct {
	ast  *ast.Ctor
	Type TypeName
	Sel  string
	Args []Expr
}

func (n *Ctor) AST() ast.Node { return n.ast }

// A Block is a block literal.
type Block struct {
	ast   *ast.Block
	Parms []Var // if type is nil, it must be inferred
	Stmts []Stmt
}

func (n *Block) AST() ast.Node { return n.ast }

// An Ident is a variable name as an expression.
type Ident struct {
	ast  *ast.Ident
	Text string
}

func (n *Ident) AST() ast.Node { return n.ast }

// An Int is an integer literal.
type Int struct {
	ast  *ast.Int
	Text string
}

func (n *Int) AST() ast.Node { return n.ast }

// A Float is a floating point literal.
type Float struct {
	ast  *ast.Float
	Text string
}

func (n *Float) AST() ast.Node { return n.ast }

// A Rune is a rune literal.
type Rune struct {
	ast  *ast.Rune
	Text string
	Rune rune
}

func (n *Rune) AST() ast.Node { return n.ast }

// A String is a string literal.
type String struct {
	ast  *ast.String
	Text string
	Data string
}

func (n *String) AST() ast.Node { return n.ast }
