package types

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/eaburns/pea/ast"
)

type state struct {
	astMod     *ast.Mod
	cfg        Config
	files      []*file
	defFiles   map[Def]*file
	gathered   map[Def]bool
	checked    map[Def]bool
	insted     map[*Type]bool
	aliasStack []*Type
	// initDeps tracks uses of *Fun and *Val for init cycle checking.
	initDeps map[Def][]witness
	localUse map[*Var]bool
	tvarUse  map[*TypeVar]bool

	// Fun instances needing instFunStmts.
	//
	// Each file that calls a parameterized func or meth
	// gets its own unique instantiation.
	// This is because each file can have different methods
	// for the same type due to different Import statements.
	// To ensure that the instantatiation uses the correct methods,
	// we make a different instance for each calling file.
	//
	// A future improvement would be to find constraint methods
	// for all type arguments, and then dedup on the method sets.
	// This would allow the possibility of sharing instances
	// across multiple files if they all use the same methods.
	funTodo []funFile

	nextID        int
	nextVar       int
	nextBlockType int
	indent        string
}

type witness struct {
	def Def
	loc ast.Node
}

type funFile struct {
	file *file
	fun  *Fun
}

func newState(cfg Config, astMod *ast.Mod) *state {
	return &state{
		astMod:   astMod,
		cfg:      cfg,
		defFiles: make(map[Def]*file),
		gathered: make(map[Def]bool),
		checked:  make(map[Def]bool),
		insted:   make(map[*Type]bool),
		initDeps: make(map[Def][]witness),
		localUse: make(map[*Var]bool),
		tvarUse:  make(map[*TypeVar]bool),
	}
}

func newDefaultState(cfg Config, astMod *ast.Mod) *state {
	x := newState(cfg, astMod)
	setConfigDefaults(x)
	return x
}

func setConfigDefaults(x *state) {
	switch x.cfg.IntSize {
	case 0:
		x.cfg.IntSize = 64
	case 8, 16, 32, 64:
		break
	default:
		panic("bad IntSize " + strconv.Itoa(x.cfg.IntSize))
	}
	if x.cfg.Importer == nil {
		x.cfg.Importer = &dirImporter{}
	}
	if _, ok := x.cfg.Importer.(*importer); !ok {
		x.cfg.Importer = newImporter(x, x.astMod.Name, x.cfg.Importer)
	}
}

func (x *state) newID() string {
	x.nextID++
	return fmt.Sprintf("$%d", x.nextID-1)
}

func (x *state) nextTypeVar() int {
	n := x.nextVar
	x.nextVar++
	return n
}

func (x *state) loc(n interface{}) ast.Loc {
	switch n := n.(type) {
	case ast.Node:
		return x.astMod.Loc(n)
	case Node:
		return x.astMod.Loc(n.ast())
	default:
		panic(fmt.Sprintf("bad type: %T", n))
	}
}

func (x *state) err(n interface{}, f string, vs ...interface{}) *checkError {
	for i, v := range vs {
		switch v := v.(type) {
		case *Val:
			if v.ModPath == x.astMod.Path {
				copy := *v
				copy.ModPath = ""
				vs[i] = &copy
			}
		case *Fun:
			if v.ModPath == x.astMod.Path {
				copy := *v
				copy.ModPath = ""
				vs[i] = &copy
			}
		case *Type:
			if v.ModPath == x.astMod.Path {
				copy := *v
				copy.ModPath = ""
				vs[i] = &copy
			}
		}
	}
	return &checkError{loc: x.loc(n), msg: fmt.Sprintf(f, vs...)}
}

const indent = "-"

// The argument to the returned function,
// if non-empty, only the first element of vs is used.
// It must be a either pointer to a slice of types convertable to error,
// or a pointer to a type convertable to error.
func (x *state) tr(f string, vs ...interface{}) func(...interface{}) {
	if !x.cfg.Trace {
		return func(...interface{}) {}
	}
	x.log(f, vs...)
	olddent := x.indent
	x.indent += indent
	return func(errs ...interface{}) {
		defer func() { x.indent = olddent }()
		if len(errs) == 0 {
			return
		}
		v := reflect.ValueOf(errs[0])
		if v.IsNil() || v.Elem().Kind() == reflect.Slice && v.Elem().Len() == 0 {
			return
		}
		x.log("%v", v.Elem().Interface())
	}
}

func (x *state) log(f string, vs ...interface{}) {
	if !x.cfg.Trace {
		return
	}
	fmt.Fprintf(os.Stderr, "%02d%s", len(x.indent)/len(indent), x.indent)
	fmt.Fprintf(os.Stderr, f, vs...)
	fmt.Fprintf(os.Stderr, "\n")
}
