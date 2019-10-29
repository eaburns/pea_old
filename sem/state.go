package sem

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/eaburns/pea/syn"
)

type state struct {
	astMod     *syn.Mod
	cfg        Config
	files      []*file
	defFiles   map[Def]*file
	gathered   map[Def]bool
	checked    map[Def]bool
	aliasStack []*Type
	// initDeps tracks uses of *Fun and *Val for init cycle checking.
	initDeps map[Def][]witness
	localUse map[*Var]bool
	tvarUse  map[*TypeVar]bool

	nextID int
	indent string
}

type witness struct {
	def Def
	loc syn.Node
}

func newState(cfg Config, astMod *syn.Mod) *state {
	return &state{
		astMod:   astMod,
		cfg:      cfg,
		defFiles: make(map[Def]*file),
		gathered: make(map[Def]bool),
		checked:  make(map[Def]bool),
		initDeps: make(map[Def][]witness),
		localUse: make(map[*Var]bool),
		tvarUse:  make(map[*TypeVar]bool),
	}
}

func newDefaultState(cfg Config, astMod *syn.Mod) *state {
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

func (x *state) loc(n interface{}) syn.Loc {
	switch n := n.(type) {
	case syn.Node:
		return x.astMod.Loc(n)
	case Node:
		return x.astMod.Loc(n.ast())
	default:
		panic(fmt.Sprintf("bad type: %T", n))
	}
}

func (x *state) err(n interface{}, f string, vs ...interface{}) *checkError {
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