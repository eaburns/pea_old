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

	// String returns a human-readable string representation
	// of the definition's signature.
	String() string
}

// A Val is a module-level value definition.
type Val struct {
	ast  *ast.Val
	Priv bool
	Mod  string
	Var  Var
	Init []Stmt

	Locals []*Var
}

func (n *Val) AST() ast.Node { return n.ast }
func (n *Val) kind() string  { return "value" }

func (n *Val) name() string {
	if n.Mod == "" {
		return n.Var.Name
	}
	return n.Mod + " " + n.Var.Name
}

// A Fun is a function or method definition.
type Fun struct {
	// ast is one of:
	// 	*ast.Fun for a function or method defintion
	// 	*ast.FunSig for a virtual function definition
	// 	*ast.Type for a case-method definition
	ast    ast.Node
	Priv   bool
	Mod    string
	Recv   *Recv
	TParms []Var // types may be nil
	Sig    FunSig
	Stmts  []Stmt

	Locals []*Var
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
	ast  ast.Node // *ast.Type or *ast.Var
	Priv bool
	Sig  TypeSig

	// Alias, Fields, Cases, and Virts
	// are mutually exclusive.
	// If any one is non-nil, the others are nil.

	// Var is non-nil for a type variable.
	Var *Var

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

// A TypeSig is a type signature, a pattern defining a type or set o types.
type TypeSig struct {
	ast   *ast.TypeSig
	Mod   string
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
	// *ast.TypeName if from an original type name.
	// *ast.Recv if the self variable type of a method.
	// *ast.Block if the inferred type of a block result.
	// *ast.Var if the inferred type of a block parameter.
	ast  ast.Node
	Mod  string
	Name string
	Args []TypeName

	Type *Type
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
	// TypeName is non-nil if explicit.
	TypeName *TypeName

	// At most one of the following is non-nil.
	TypeVar *Type   // a type variable; Index is unused.
	Val     *Val    // a module-level Val; Index is unused.
	FunParm *Fun    // a function parm; Index is the Parms index.
	BlkParm *Block  // a block parm; Index is the Parms index.
	Local   *[]*Var // a local variable; Index is the index.
	Field   *Type   // an And-type field; Index is the Fields index.
	// Index is used as described above.
	Index int

	typ *Type
}

func (n *Var) AST() ast.Node { return n.ast }
func (n *Var) Type() *Type   { return n.typ }

func (n *Var) isSelf() bool {
	return n.FunParm != nil && n.FunParm.Recv != nil && n.Index == 0
}

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
	ast  *ast.Assign
	Var  *Var
	Expr Expr
}

func (n *Assign) AST() ast.Node { return n.ast }

// An Expr is an expression
type Expr interface {
	Node
	Type() *Type
}

// A Call is a method call or a cascade.
type Call struct {
	ast  ast.Node // *ast.Call or *ast.Ident if in-module unary function call.
	Recv Expr     // nil for function calls
	Msgs []Msg

	typ *Type
}

func (n *Call) AST() ast.Node { return n.ast }
func (n *Call) Type() *Type   { return n.typ }

// A Msg is a message, sent to a value.
type Msg struct {
	ast  ast.Node // *ast.Msg or *ast.Ident if in-module unary function call.
	Mod  string
	Sel  string
	Args []Expr
}

func (n Msg) AST() ast.Node { return n.ast }

// A Ctor type constructor literal.
type Ctor struct {
	ast      *ast.Ctor
	TypeName TypeName
	Sel      string
	Args     []Expr

	// Case is non-nil if this is an or-type constructor.
	// It is an index into the typ.Cases array.
	Case *int

	// Ref is non-0 for a reference conversion.
	// A negative value is the number of references to remove,
	// and a positive value is the number of references to add.
	Ref int

	typ *Type
}

func (n *Ctor) AST() ast.Node { return n.ast }
func (n *Ctor) Type() *Type   { return n.typ }

// A Block is a block literal.
type Block struct {
	ast   *ast.Block
	Parms []Var // if type is nil, it must be inferred
	Stmts []Stmt

	Locals []*Var
	typ    *Type
}

func (n *Block) AST() ast.Node { return n.ast }
func (n *Block) Type() *Type   { return n.typ }

// An Ident is a variable name as an expression.
type Ident struct {
	ast  *ast.Ident
	Text string
	Var  *Var
}

func (n *Ident) AST() ast.Node { return n.ast }

func (n *Ident) Type() *Type {
	if n.Var == nil {
		return nil
	}
	return n.Var.typ
}

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
