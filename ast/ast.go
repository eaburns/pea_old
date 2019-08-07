package ast

//go:generate peggy -o grammar.go -t grammar.peggy

// A Mod is a module: the unit of compilation.
type Mod struct {
	Name  string
	Files []File
}

// File is a single source code file.
type File struct {
	Path  string
	offs  int   // offset of the start of the file
	lines []int // offset of newlines
	Defs  []Def
}

// A Node is a node of the AST with location information.
type Node interface {
	Start() int
	End() int
}

// A Def is a module-level definition.
type Def interface {
	Node

	addMod(ModPath) Def
	setPriv(bool) Def
	setStart(int) Def
}

type location struct {
	start, end int
}

func (n location) Start() int { return n.start }
func (n location) End() int   { return n.end }

// Import is an import statement.
type Import struct {
	location
	Priv bool
	ModPath
	Path string
}

// A Fun is a function or method definition.
type Fun struct {
	location
	Priv bool
	ModPath
	Sel       string
	Recv      *TypeSig
	TypeParms []Parm // types may be nil
	Parms     []Parm // types cannot be nil
	Ret       *TypeName
	Stmts     []Stmt
}

func (n *Fun) Name() string {
	if n.Recv != nil {
		return n.Recv.Name + " " + n.Sel
	}
	return n.Sel
}

// A Parm is a name and a type.
// Parms are used in several AST nodes.
// In some cases, the type must be non-nil.
// In others, the type may be nil.
type Parm struct {
	location
	Name string
	Type *TypeName
}

// A Var is a module-level variable definition.
type Var struct {
	location
	Priv bool
	ModPath
	Var   *Parm
	Ident string
	Type  *TypeName
	Val   []Stmt
}

// A TypeSig is a type signature, a pattern defining a type or set o types.
type TypeSig struct {
	location
	Name  string
	Parms []Parm // types may be nil
}

// A TypeName is the name of a concrete type.
type TypeName struct {
	location
	Var  bool
	Mod  *ModPath
	Name string
	Args []TypeName
}

// A Type defines a type.
// There are several kinds of Types:
// 	1) Built-in types are primitive to the language.
// 	2) Aliases aren't types, but are alternative names for other types.
// 	3) And types are a composite of one or more other types
// 	   that are the fields of the And type.
// 	   This is akin to a struct or class in other languages.
// 	4) Or types are a conjunction of zero or more explicitly named types,
// 	   that are ethe cases of the Or type.
//	   Or types fill the role of enums, null-able pointers,
// 	   and tagged unions in other languages.
// 	5) Virtual types are defined by a set of method signatures.
// 	   Any type with the required methods is automatically
// 	   convertable to the virtual type.
type Type struct {
	location
	Priv bool
	ModPath
	Sig TypeSig

	// Alias, Fields, Cases, and Virts
	// are mutually exclusive.
	// If any one is non-nil, the others are nil.

	// Alias is non-nil for a type Alias.
	Alias *TypeName

	// Fields is non-nil for an And type.
	Fields []Parm // types cannot be nil

	// Cases is non-nil for an Or type.
	Cases []Parm // types can be nil

	// Virts is non-nil for a Virtual type.
	Virts []MethSig
}

// A MethSig is the signature of a method.
type MethSig struct {
	location
	Sel   string
	Parms []TypeName
	Ret   *TypeName
}

// A Stmt is a statement.
type Stmt interface {
	Node
}

// A Ret is a return statement.
type Ret struct {
	start int
	Val   Expr
}

func (n *Ret) Start() int { return n.start }
func (n *Ret) End() int   { return n.Val.End() }

// An Assign is an assignment statement.
type Assign struct {
	// Vars are the target of assignment.
	// After type checking, these refer to the defining Param,
	// either a local variable or Fun/Block parameter.
	Vars []*Parm // types may be nil before successful Check()
	Val  Expr
}

func (n *Assign) Start() int { return n.Vars[0].Start() }
func (n *Assign) End() int   { return n.Val.End() }

// An Expr is an expression
type Expr interface {
	Node
	isExpr()
}

// A Call is a method call or a cascade.
type Call struct {
	location
	Recv Node // Expr, ModPath, or nil
	Msgs []Msg
}

func (Call) isExpr() {}

// A Msg is a message, sent to a value.
type Msg struct {
	location
	Sel  string
	Args []Expr
}

// A Ctor type constructor literal.
type Ctor struct {
	location
	Type TypeName
	Sel  string
	Args []Expr
}

func (Ctor) isExpr() {}

// A Block is a block literal.
type Block struct {
	location
	Parms []Parm // if type is nil, it must be inferred
	Stmts []Stmt
}

func (Block) isExpr() {}

// A ModPath is a module name.
type ModPath struct {
	location
	Root string // current or imported module name
	Path []string
}

func (n ModPath) Mod() ModPath { return n }

// An Ident is a variable name as an expression.
type Ident struct {
	location
	Text string
}

func (Ident) isExpr() {}

// An Int is an integer literal.
type Int struct {
	location
	Text string
}

func (Int) isExpr() {}

// A Float is a floating point literal.
type Float struct {
	location
	Text string
}

func (Float) isExpr() {}

// A Rune is a rune literal.
type Rune struct {
	location
	Text string
	Rune rune
}

func (Rune) isExpr() {}

// A String is a string literal.
type String struct {
	location
	Text string
	Data string
}

func (String) isExpr() {}
