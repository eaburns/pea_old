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

	// name returns the name of the definition for use in debug and tracing.
	name() string

	// Priv returns whether the definition is private.
	Priv() bool

	// Mod returns the module name of the definition if imported.
	// Mod returns the empty string for definitions in the current module.
	Mod() string

	// String returns a human-readable string representation
	// of the definition's signature.
	String() string
}

// A Val is a module-level value definition.
type Val struct {
	ast  *ast.Val
	priv bool
	mod  string
	Name string
	Type *TypeName
	Init []Stmt
}

func (n *Val) AST() ast.Node { return n.ast }
func (n *Val) kind() string  { return "value" }

func (n *Val) name() string {
	if n.mod == "" {
		return n.Name
	}
	return n.mod + " " + n.Name
}

func (n *Val) Priv() bool  { return n.priv }
func (n *Val) Mod() string { return n.mod }

// A Fun is a function or method definition.
type Fun struct {
	ast    *ast.Fun
	priv   bool
	mod    string
	Recv   *Recv
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

func (n *Fun) name() string {
	switch {
	case n.mod == "" && n.Recv == nil:
		return n.Sig.Sel
	case n.mod == "" && n.Recv != nil:
		return fmt.Sprintf("(%d)%s %s", n.Recv.Arity, n.Recv.Name, n.Sig.Sel)
	case n.mod != "" && n.Recv == nil:
		return n.mod + " " + n.Sig.Sel
	default:
		return fmt.Sprintf("%s (%d)%s %s", n.mod, n.Recv.Arity, n.Recv.Name, n.Sig.Sel)
	}
}

func (n *Fun) Priv() bool  { return n.priv }
func (n *Fun) Mod() string { return n.mod }

// Recv is a method receiver.
type Recv struct {
	ast   *ast.Recv
	Parms []Var
	Mod   string
	Arity int
	Name  string

	Type *Type
}

// ID returns a user-readable type identifier that includes
// the module if not the current module, name, and arity if non-zero.
func (n *Recv) ID() string {
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
func (n *Type) name() string  { return n.Sig.ID() }
func (n *Type) Priv() bool    { return n.priv }
func (n *Type) Mod() string   { return n.Sig.mod }

// A TypeSig is a type signature, a pattern defining a type or set o types.
type TypeSig struct {
	ast   *ast.TypeSig
	mod   string
	Arity int
	Name  string

	Parms []Var      // types may be nil
	Args  []TypeName // what is subbed for Parms
}

func (n *TypeSig) AST() ast.Node { return n.ast }

// ID returns a user-readable type identifier that includes the name
// and the arity if non-zero.
func (n *TypeSig) ID() string {
	switch {
	case n.mod == "" && n.Arity == 0:
		return n.Name
	case n.mod == "" && n.Arity > 0:
		return fmt.Sprintf("(%d)%s", n.Arity, n.Name)
	case n.mod != "" && n.Arity == 0:
		return n.mod + " " + n.Name
	default:
		return fmt.Sprintf("%s (%d)%s", n.mod, n.Arity, n.Name)
	}
}

// A TypeName is the name of a concrete type.
type TypeName struct {
	ast  *ast.TypeName
	Mod  string
	Name string
	Args []TypeName

	Var  *Var  // non-nil if a type variable
	Type *Type // nil if error or a type variable (in which case Var is non-nil)
}

func (n *TypeName) AST() ast.Node { return n.ast }

// ID returns a user-readable type identifier that includes
// the module if not the current module, name, and arity if non-zero.
func (n *TypeName) ID() string {
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
	Var Var
	Val Expr
}

func (n *Assign) AST() ast.Node { return n.ast }

// An Expr is an expression
type Expr interface {
	Node

	check(*scope, *TypeName) (Expr, []checkError)
}

// A Call is a method call or a cascade.
type Call struct {
	ast  *ast.Call
	Recv Expr // nil for function calls
	Msgs []Msg
}

func (n *Call) AST() ast.Node { return n.ast }

// A Msg is a message, sent to a value.
type Msg struct {
	ast  *ast.Msg
	Mod  string
	Sel  string
	Args []Expr
}

func (n Msg) AST() ast.Node { return n.ast }

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
	ast ast.Expr // *ast.Int, *ast.Float, or *ast.Rune
	Val *big.Int
	typ *Type
}

func (n *Int) AST() ast.Node { return n.ast }
func (n *Int) Type() *Type   { return n.typ }

func (n *Int) PrettyPrint() string {
	if _, ok := n.ast.(*ast.Rune); ok {
		return "Int{Val: " + strconv.QuoteRune(rune(n.Val.Int64())) + "}"
	}
	return "Int{Val: " + n.Val.String() + "}"
}

// A Float is a floating point literal.
type Float struct {
	ast ast.Expr // *ast.Float or *ast.Int
	Val *big.Float
	typ *Type
}

func (n *Float) AST() ast.Node       { return n.ast }
func (n *Float) PrettyPrint() string { return "Int{Val: " + n.Val.String() + "}" }
func (n *Float) Type() *Type         { return n.typ }

// A String is a string literal.
type String struct {
	ast  *ast.String
	Data string
	typ  *Type
}

func (n String) AST() ast.Node { return n.ast }
func (n String) Type() *Type   { return n.typ }
