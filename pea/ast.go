package pea

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

// A Node is a node of the AST with location information.
type Node interface {
	Start() int
	End() int
}

// A Def is a module-level definition.
type Def interface {
	Node
	// String returns a 1-line summary of the definition.
	String() string
	setMod(ModPath) Def
	setStart(int) Def
}

type location struct {
	start, end int
}

func (n location) Start() int { return n.start }
func (n location) End() int   { return n.end }

// A SubMod is a module definition.
type SubMod struct {
	location
	Mod  ModPath
	Defs []Def
}

// A ModPath is a module name.
type ModPath []Ident

func (n ModPath) Start() int { return n[0].Start() }
func (n ModPath) End() int   { return n[len(n)-1].End() }

// Import is an import statement.
type Import struct {
	location
	Path string
}

// A Fun is a function or method definition.
type Fun struct {
	location
	Mod       ModPath
	Sel       string
	Recv      *TypeSig
	TypeParms []Parm // types may be nil
	Parms     []Parm // types cannot be nil
	Ret       *TypeName
	Stmts     []Stmt
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
	Mod  ModPath
	Name string
	Val  []Stmt
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
	Name string
	Args []TypeName
}

// A Struct defines a struct type.
type Struct struct {
	location
	Mod    ModPath
	Sig    TypeSig
	Fields []Parm // types cannot be nil
}

// A Enum defines an enum type.
type Enum struct {
	location
	Mod   ModPath
	Sig   TypeSig
	Cases []Parm // types may be nil
}

// A Virt defines a virtual type.
type Virt struct {
	location
	Mod   ModPath
	Sig   TypeSig
	Meths []MethSig
}

// A MethSig is the signature of a method.
type MethSig struct {
	location
	Sel   string
	Parms []TypeName
	Ret   *TypeName
}

// A Stmt is a statement.
type Stmt interface{}

// A Ret is a return statement.
type Ret struct {
	start int
	Val   Expr
}

func (n *Ret) Start() int { return n.start }
func (n *Ret) End() int   { return n.Val.End() }

// An Assign is an assignment statement.
type Assign struct {
	Var []Parm // types may be nil
	Val Expr
}

func (n *Assign) Start() int { return n.Var[0].Start() }
func (n *Assign) End() int   { return n.Val.End() }

// An Expr is an expression
type Expr interface {
	Start() int
	End() int
}

// A Call is a method call or a cascade.
type Call struct {
	location
	Recv Expr // nil for module-less function calls
	Msgs []Msg
}

// A Msg is a message, sent to a value.
type Msg struct {
	location
	Sel  string
	Args []Expr
}

// An Ident is a variable name as an expression.
type Ident struct {
	location
	Text string
}

// An Int is an integer literal.
type Int struct {
	location
	Text string
}

// A Float is a floating point literal.
type Float struct {
	location
	Text string
}

// A Rune is a rune literal.
type Rune struct {
	location
	Text string
	Rune rune
}

// A String is a string literal.
type String struct {
	location
	Text string
	Data string
}

// A Ctor type constructor literal.
type Ctor struct {
	location
	Type TypeName
	Args []Expr
}

// A Block is a block literal.
type Block struct {
	location
	Parms []Parm // if type is nil, it must be inferred
	Stmts []Stmt
}
