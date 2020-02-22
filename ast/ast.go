// Copyright © 2020 The Pea Authors under an MIT-style license.

// Package ast defines the syntax. It contains an AST and a parser.
package ast

import (
	"strings"

	"github.com/eaburns/pea/loc"
)

//go:generate peggy -t=false -o grammar.go grammar.peggy

// A Mod is a module: the unit of compilation.
type Mod struct {
	Path  string
	Files []File
	Locs  *loc.Files
}

// File is a single source code file.
type File struct {
	Path    string
	Imports []Import
	Defs    []Def
}

// An Import is an import statement.
type Import struct {
	loc.Range
	// All indicates whether the keyword was "Import",
	// which imports all the exported symbols
	// in addition to the module name.
	All  bool
	Path string
}

// A Node is a node of the AST with loc.Range information.
type Node interface {
	GetRange() loc.Range
}

// A Def is a module-level definition.
type Def interface {
	Node

	// Priv returns whether the definition is private.
	Priv() bool

	// String returns the string representation of the definition signature.
	// The signature is the definition excluding statements.
	String() string
}

// A Val is a module-level value definition.
type Val struct {
	loc.Range
	priv bool
	Var  Var
	Init []Stmt
}

func (n *Val) Priv() bool { return n.priv }

// A Fun is a function or method definition.
type Fun struct {
	loc.Range
	priv   bool
	Test   bool
	Recv   *Recv
	TParms []Var // types may be nil
	Sig    FunSig
	// Stmts are the body of the function or method.
	// If Stmts==nil, this is a declaration only;
	// for a function or method definition with no body
	// Stmts will be non-nil with length 0.
	Stmts []Stmt
}

func (n *Fun) Priv() bool { return n.priv }

// Recv is a method receiver.
type Recv struct {
	TypeSig
	Mod *ModTag
}

// A FunSig is the signature of a function.
type FunSig struct {
	loc.Range
	Sel   string
	Parms []Var // types cannot be nil
	Ret   *TypeName
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
	loc.Range
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

func (n Type) Priv() bool { return n.priv }

// A TypeName is the name of a concrete type.
type TypeName struct {
	loc.Range
	Var  bool
	Mod  *ModTag
	Name string
	Args []TypeName
}

// A TypeSig is a type signature, a pattern defining a type or set o types.
type TypeSig struct {
	loc.Range
	Name  string
	Parms []Var // types may be nil
}

// A Var is a name and a type.
// Var are used in several AST nodes.
// In some cases, the type must be non-nil.
// In others, the type may be nil.
type Var struct {
	loc.Range
	Name string
	Type *TypeName
}

// A Stmt is a statement.
type Stmt interface {
	Node
	buildString(string, *strings.Builder)
}

// A Ret is a return statement.
type Ret struct {
	start int
	Expr  Expr
}

// GetRange returns the source location Range.
func (n *Ret) GetRange() loc.Range {
	return loc.Range{n.start, n.Expr.GetRange()[1]}
}

// An Assign is an assignment statement.
type Assign struct {
	Vars []Var // types may be nil
	Expr Expr
}

// GetRange returns the source location Range.
func (n *Assign) GetRange() loc.Range {
	return loc.Range{
		n.Vars[0].Range[0],
		n.Expr.GetRange()[1],
	}
}

// An Expr is an expression
type Expr interface {
	Stmt
	isExpr()
}

// A Call is a method call or a cascade.
type Call struct {
	loc.Range
	Recv Expr // nil for function calls
	Msgs []Msg
}

func (*Call) isExpr() {}

// A Msg is a message, sent to a value.
type Msg struct {
	loc.Range
	Mod  *ModTag
	Sel  string
	Args []Expr
}

// A Ctor type constructor literal.
type Ctor struct {
	loc.Range
	Args []Expr
}

func (*Ctor) isExpr() {}

// A Block is a block literal.
type Block struct {
	loc.Range
	Parms []Var // if type is nil, it must be inferred
	Stmts []Stmt
}

func (*Block) isExpr() {}

// An Ident is a variable name as an expression.
type Ident struct {
	loc.Range
	Mod  *ModTag
	Text string
}

func (*Ident) isExpr() {}

// A ModTag is a module name preceeded by #.
type ModTag struct {
	loc.Range
	Text string
}

// An Int is an integer literal.
type Int struct {
	loc.Range
	Text string
}

func (*Int) isExpr() {}

// A Float is a floating point literal.
type Float struct {
	loc.Range
	Text string
}

func (*Float) isExpr() {}

// A Rune is a rune literal.
type Rune struct {
	loc.Range
	Text string
	Rune rune
}

func (*Rune) isExpr() {}

// A String is a string literal.
type String struct {
	loc.Range
	Text string
	Data string
}

func (*String) isExpr() {}
