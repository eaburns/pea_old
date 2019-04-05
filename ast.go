package main

//go:generate peggy -o grammar.go -t grammar.peggy

// A Mod is a module: the unit of compilation.
type Mod struct {
	Name  string
	Files []*File
}

// A File is a source code file.
type File struct {
	Path string
	Defs []Def

	offs  int   // offset of the start of the file
	lines []int // offset of newlines
}

// A Def is a module-level definition.
type Def interface {
	Start() int
	End() int
}

// A SubMod is a module definition.
type SubMod struct {
	start, end int
	Mod        ModPath
	Defs       []Def
}

func (n *SubMod) Start() int { return n.start }
func (n *SubMod) End() int   { return n.end }

// A ModPath is a module name.
type ModPath []Ident

func (n ModPath) Start() int { return n[0].Start() }
func (n ModPath) End() int   { return n[len(n)-1].End() }

// Import is an import statement.
type Import struct {
	start, end int
	Path       string
}

func (n *Import) Start() int { return n.start }
func (n *Import) End() int   { return n.end }

// A Fun is a function or method definition.
type Fun struct {
	start, end int
	Mod        ModPath
	Sel        string
	Recv       *TypeSig
	TypeParms  []Parm // types may be nil
	Parms      []Parm // types cannot be nil
	Ret        *TypeName
	Stmts      []Stmt
}

func (n *Fun) Start() int { return n.start }
func (n *Fun) End() int   { return n.end }

// A Parm is a name and a type.
// Parms are used in several AST nodes.
// In some cases, the type must be non-nil.
// In others, the type may be nil.
type Parm struct {
	Name string
	Type *TypeName
}

// A Var is a module-level variable definition.
type Var struct {
	start, end int
	Mod        ModPath
	Name       string
	Val        []Stmt
}

func (n *Var) Start() int { return n.start }
func (n *Var) End() int   { return n.end }

// A TypeSig is a type signature, a pattern defining a type or set o types.
type TypeSig struct {
	Name  string
	Parms []Parm // types may be nil
}

// A TypeName is the name of a concrete type.
type TypeName struct {
	Name string
	Args []TypeName
}

// A Struct defines a struct type.
type Struct struct {
	start, end int
	Mod        ModPath
	Sig        TypeSig
	Fields     []Parm // types cannot be nil
}

func (n *Struct) Start() int { return n.start }
func (n *Struct) End() int   { return n.end }

// A Enum defines an enum type.
type Enum struct {
	start, end int
	Mod        ModPath
	Sig        TypeSig
	Cases      []Parm // types may be nil
}

func (n *Enum) Start() int { return n.start }
func (n *Enum) End() int   { return n.end }

// A Virt defines a virtual type.
type Virt struct {
	start, end int
	Mod        ModPath
	Sig        TypeSig
	Meths      []MethSig
}

func (n *Virt) Start() int { return n.start }
func (n *Virt) End() int   { return n.end }

// A MethSig is the signature of a method.
type MethSig struct {
	Sel   string
	Parms []TypeName
	Ret   *TypeName
}

// A Stmt is a statement.
type Stmt interface{}

// A Ret is a return statement.
type Ret struct {
	Val Expr
}

// An Assign is an assignment statement.
type Assign struct {
	Var []Parm // types may be nil
	Val Expr
}

// An Expr is an expression
type Expr interface {
	Start() int
	End() int
}

// A Call is a method call or a cascade.
type Call struct {
	start, end int
	Recv       Expr // nil for module-less function calls
	Msgs       []Msg
}

func (n Call) Start() int { return n.start }
func (n Call) End() int   { return n.end }

// A Msg is a message, sent to a value.
type Msg struct {
	start, end int
	Sel        string
	Args       []Expr
}

// An Ident is a variable name as an expression.
type Ident struct {
	start, end int
	Text       string
}

func (n Ident) Start() int { return n.start }
func (n Ident) End() int   { return n.end }

// An Int is an integer literal.
type Int struct {
	start, end int
	Text       string
}

func (n Int) Start() int { return n.start }
func (n Int) End() int   { return n.end }

// A Float is a floating point literal.
type Float struct {
	start, end int
	Text       string
}

func (n Float) Start() int { return n.start }
func (n Float) End() int   { return n.end }

// A Rune is a rune literal.
type Rune struct {
	start, end int
	Text       string
	Rune       rune
}

func (n Rune) Start() int { return n.start }
func (n Rune) End() int   { return n.end }

// A String is a string literal.
type String struct {
	start, end int
	Text       string
	Data       string
}

func (n String) Start() int { return n.start }
func (n String) End() int   { return n.end }

// A Ctor type constructor literal.
type Ctor struct {
	start, end int
	Type       TypeName
	Args       []Expr
}

func (n Ctor) Start() int { return n.start }
func (n Ctor) End() int   { return n.end }

// A Block is a block literal.
type Block struct {
	start, end int
	Parms      []Parm // if type is nil, it must be inferred
	Stmts      []Stmt
}

func (n Block) Start() int { return n.start }
func (n Block) End() int   { return n.end }
