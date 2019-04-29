package pea

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// An Opt is an option to the type checker.
type Opt func(*state)

var (
	// Trace enables tracing of the type checker.
	Trace Opt = func(x *state) { x.trace = true }
	// Word64 makes Word and Uint aliases to Uint64, and Int an alias to Int64.
	Word64 = func(x *state) { x.wordSize = 64 }
	// Word32 makes Word and Uint aliases to Uint32, and Int an alias to Int32.
	Word32 = func(x *state) { x.wordSize = 32 }
)

// Check type-checks the module.
// Check modifies its arugment, performing some simplifications on the AST
// and populating several fields not set by parsing.
func Check(mod *Mod, opts ...Opt) []error {
	s := newState(mod)
	for _, opt := range opts {
		opt(s)
	}
	x := &scope{state: s}
	mod.Imports = append(mod.Imports, builtinMod(x.wordSize))
	if es := collect(x, nil, []string{mod.Name}, mod.Imports[0]); len(es) > 0 {
		panic("impossible")
	}
	errs := collect(x, nil, []string{mod.Name}, mod)
	return convertErrors(errs)
}

type state struct {
	mod      *Mod
	mods     *module
	wordSize int
	importer func(string) (*Mod, error)

	trace bool
	ident string
}

func newState(mod *Mod) *state {
	return &state{
		mod:      mod,
		mods:     newModule(""),
		importer: defaultImporter,
		wordSize: 64,
	}
}

type module struct {
	name string
	defs map[string]Def
	kids map[string]*module
}

type imported struct {
	imp *Import
	Def
}

type builtin struct {
	Def
}

func newModule(name string) *module {
	return &module{
		name: name,
		defs: make(map[string]Def),
		kids: make(map[string]*module),
	}
}

// add either adds the def and return nil
// or a def with the same name already exists
// and nothing is added and the previous def is returned.
// The return is an imported, a builtin, or a Def.
func (t *module) add(path []string, def Def) interface{} {
	if len(path) == 0 {
		switch prev := t.defs[def.Name()].(type) {
		case nil:
			break
		case imported:
			if _, ok := def.(imported); ok {
				return prev
			}
			break // overridden
		case builtin:
			break // overridden
		default:
			return prev
		}
		t.defs[def.Name()] = def
		return nil
	}
	k := t.kids[path[0]]
	if k == nil {
		k = newModule(path[0])
		t.kids[path[0]] = k
	}
	return k.add(path[1:], def)
}

func collect(x *scope, imp *Import, path []string, mod *Mod) (errs []checkError) {
	defer x.tr("collect(%v, %v, %s)", imp, path, mod.Name)(errs)
	for _, def := range mod.Defs {
		if d, ok := def.(*Import); ok {
			m, err := importMod(x, d)
			if err != nil {
				errs = append(errs, *err)
				continue
			}
			kidImp := imp
			if kidImp == nil {
				kidImp = d
			}
			es := collect(x, kidImp, append(path, d.Mod().Path...), m)
			if len(es) > 0 {
				errs = append(errs, es...)
			}
		} else {
			switch {
			case def.Mod().Root == "":
				def = builtin{def}
			case imp != nil:
				def = imported{imp, def}
			}
			p := append(path, def.Mod().Path...)
			x.log("adding %v %v", p, def)
			prev := x.mods.add(p, def)
			if prev != nil {
				errs = append(errs, *redefError(x, def, prev))
			}
		}
	}
	return errs
}

func redefError(x *scope, def, prev interface{}) *checkError {
	var err *checkError
	switch def := def.(type) {
	case imported:
		err = x.err(def.imp, "imported definition %s redefined", defName(def))
	case builtin:
		// Built-in defs are added first, so they can never be a redef.
		panic(fmt.Sprintf("impossible definition type %T", def))
	case Def:
		err = x.err(def, "%s redefined", defName(def))
	default:
		panic(fmt.Sprintf("impossible definition type %T", def))
	}
	switch prev := prev.(type) {
	case imported:
		note(err, "previous definition imported from %s", x.loc(prev.imp))
	case builtin:
		note(err, "%s is a built-in definition", defName(prev))
	case Def:
		note(err, "previous definition %s", x.loc(prev))
	default:
		panic(fmt.Sprintf("impossible previous definition type %T", prev))
	}
	return err
}

func defName(def Def) string {
	if m := def.Mod().String(); m != "" {
		return m + " " + def.Name()
	}
	return def.Name()
}

func importMod(x *scope, n *Import) (m *Mod, err *checkError) {
	x.tr("importMod(%s)", n.Path)(err)
	for _, m = range x.mod.Imports {
		if m.Name == n.Path {
			x.log("returning previously imported module")
			return m, nil
		}
	}
	var e error
	if m, e = x.importer(n.Path); e != nil {
		return nil, x.err(n, "error importing %s: %s", n.Path, e)
	}
	x.log("returning new module: %s, %s", n.Path, m.Name)
	x.mod.Imports = append(x.mod.Imports, m)
	return m, nil
}

func defaultImporter(path string) (*Mod, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	finfos, err := f.Readdir(0) // all
	f.Close()
	if err != nil {
		return nil, err
	}
	p := NewParser(path)
	for _, fi := range finfos {
		err := p.ParseFile(filepath.Join(path, fi.Name()))
		if err != nil {
			return nil, err
		}
	}
	return p.Mod(), nil
}

type scope struct {
	*state
	parent *scope
	name   string
	node   Node
}

func (x *scope) find(s string) Node {
	if x.name == s {
		return x.node
	}
	if x.parent == nil {
		return nil
	}
	return x.parent.find(s)
}

func (x *scope) push(s string, n Node) *scope {
	return &scope{state: x.state, parent: x, name: s, node: n}
}

type checkError struct {
	loc   Loc
	msg   string
	notes []string
	cause []checkError
}

func (s *state) loc(n Node) Loc { return s.mod.Loc(n) }

func (x *scope) err(n Node, f string, vs ...interface{}) *checkError {
	return &checkError{loc: x.mod.Loc(n), msg: fmt.Sprintf(f, vs...)}
}

func note(err *checkError, f string, vs ...interface{}) {
	err.notes = append(err.notes, fmt.Sprintf(f, vs...))
}

func (err *checkError) Error() string {
	var s strings.Builder
	buildError(&s, "", err)
	return s.String()
}

func buildError(s *strings.Builder, ident string, err *checkError) {
	s.WriteString(ident)
	s.WriteString(err.loc.String())
	s.WriteString(": ")
	s.WriteString(err.msg)
	ident2 := ident + "	"
	for _, n := range err.notes {
		s.WriteRune('\n')
		s.WriteString(ident2)
		s.WriteString(n)
	}
	for i := range err.cause {
		s.WriteRune('\n')
		buildError(s, ident2, &err.cause[i])
	}
}

func convertErrors(cerrs []checkError) []error {
	var errs []error
	for i := range sortErrors(cerrs) {
		errs = append(errs, &cerrs[i])
	}
	return errs
}

func sortErrors(errs []checkError) []checkError {
	if len(errs) == 0 {
		return errs
	}
	sort.Slice(errs, func(i, j int) bool {
		switch ei, ej := errs[i].loc, &errs[j].loc; {
		case ei.Path == ej.Path && ei.Line[0] == ej.Line[0]:
			return ei.Col[0] < ej.Col[0]
		case ei.Path == ej.Path:
			return ei.Line[0] < ej.Line[0]
		default:
			return ei.Path < ej.Path
		}
	})
	dedup := []checkError{errs[0]}
	for _, e := range errs[1:] {
		d := &dedup[len(dedup)-1]
		if e.loc != d.loc || e.msg != d.msg {
			dedup = append(dedup, e)
		}
	}
	for i := range dedup {
		dedup[i].cause = sortErrors(dedup[i].cause)
	}
	return dedup
}

// If non-empty, only the first element of vs is used.
// It must be either a slice of types convertable to error,
// or a pointer to a type convertable to error.
func (x *scope) tr(f string, vs ...interface{}) func(...interface{}) {
	if !x.trace {
		return func(...interface{}) {}
	}
	x.log(f, vs...)
	olddent := x.ident
	x.ident += "---"
	return func(errs ...interface{}) {
		if len(errs) == 0 {
			x.ident = olddent
			return
		}
		switch v := reflect.ValueOf(errs[0]); v.Kind() {
		case reflect.Slice:
			if v.Len() > 0 {
				x.log(v.Index(0).Interface().(error).Error())
			}
		case reflect.Ptr:
			if !v.IsNil() {
				x.log(v.Elem().Interface().(error).Error())
			}
		}
		x.ident = olddent
	}
}

func (x *scope) log(f string, vs ...interface{}) {
	if !x.trace {
		return
	}
	fmt.Printf(x.ident)
	fmt.Printf(f, vs...)
	fmt.Println("")
}
