package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

type state struct {
	mod      *Mod
	mods     *module
	wordSize int
	importer func(string) (*Mod, error)

	aliasPath   []*Type
	onAliasPath map[*Type]bool
	aliasCycles map[*Type]*checkError

	typeVars  map[*Parm]*Type
	typeInsts map[string]interface{}
	funInsts  map[[2]string]*Fun // receiver + type parms reified

	next int

	trace bool
	dump  bool
	ident string
}

func newState(mod *Mod, opts ...Opt) *state {
	s := &state{
		mod:         mod,
		mods:        newModule(""),
		importer:    defaultImporter,
		onAliasPath: make(map[*Type]bool),
		aliasCycles: make(map[*Type]*checkError),
		typeVars:    make(map[*Parm]*Type),
		typeInsts:   make(map[string]interface{}),
		funInsts:    make(map[[2]string]*Fun),
		wordSize:    64,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func newScope(mod *Mod, opts ...Opt) *scope {
	s := newState(mod, opts...)
	x := &scope{state: s}
	builtin := builtinMod(x.wordSize)
	mod.Imports = append(mod.Imports, builtin)
	if es := collect(x, nil, nil, mod.Imports[0]); len(es) > 0 {
		panic("impossible")
	}
	for _, def := range builtin.Defs {
		if es := checkDef(x, def); len(es) > 0 {
			panic(fmt.Sprintf("impossible: %v", es))
		}
	}
	return x
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

func dump(m *module, ident string) {
	fmt.Printf("%s [%s]\n", ident, m.name)
	ident += "	"
	var defs []Def
	for _, d := range m.defs {
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name() < defs[j].Name()
	})
	for _, d := range defs {
		fmt.Printf("%s [%s]: %s\n", ident, d.Name(), d)
	}
	for _, k := range m.kids {
		dump(k, ident)
	}
}

// The return is an imported, a builtin, a Def, or nil.
func (t *module) find(mp ModPath, name string) interface{} {
	path := mp.Path
	if mp.Root != "" {
		path = append([]string{mp.Root}, path...)
	}
	return t._find(path, name)
}

func (t *module) _find(path []string, name string) interface{} {
	switch {
	case t == nil:
		return nil
	case len(path) == 0:
		return t.defs[name]
	case t.kids[path[0]] == nil:
		return nil
	default:
		def := t.kids[path[0]]._find(path[1:], name)
		if def == nil {
			return t.defs[name]
		}
		return def
	}
}

// add either adds the def and return nil or returns the previous def.
// The return is an imported, a builtin, a Def, or nil.
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
			x.log("adding %v %v", p, def.Name())
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
		err = x.err(def.imp, "imported definition %s is redefined", defName(def))
	case builtin:
		// Built-in defs are added first, so they can never be a redef.
		panic(fmt.Sprintf("impossible definition type %T", def))
	case Def:
		err = x.err(def, "%s is redefined", defName(def))
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
	defer x.tr("importMod(%s)", n.Path)(err)
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

	if es := checkMod(x, m); len(es) > 0 {
		panic(fmt.Sprintf("imported module contains errors: %v", es))
	}

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

func addDefNotes(err *checkError, x *scope, defOrImport interface{}) {
	switch d := defOrImport.(type) {
	case nil:
		break // shouldn't happen, but whatever.
	case imported:
		note(err, "%s is imported at %s", defName(d), x.loc(d.imp))
	case builtin:
		note(err, "%s is a built-in definition", defName(d))
	case Def:
		note(err, "%s defined at %s", defName(d), x.loc(d))
	}
}

type scope struct {
	*state
	parent *scope
	def    Def
	block  *Block
	node   Node   // If non-nil, name is the name identifier name for this node.
	name   string // only valid if node is non-nil
}

func (x *scope) find(s string) Node {
	switch {
	case x == nil:
		return nil
	case x.node != nil && x.name == s:
		return x.node
	default:
		return x.parent.find(s)
	}
}

func (x *scope) push(s string, n Node) *scope {
	return &scope{state: x.state, parent: x, name: s, node: n}
}

func (x *scope) root() *scope {
	if x.parent == nil {
		return x
	}
	return x.parent.root()
}

func (x *scope) modPath() *ModPath {
	switch {
	case x == nil:
		return nil
	case x.def != nil:
		mp := x.def.Mod()
		return &mp
	default:
		return x.parent.modPath()
	}
}

func (x *scope) fun() *Fun {
	if x == nil {
		return nil
	}
	if f, ok := x.def.(*Fun); ok {
		return f
	}
	return x.parent.fun()
}

func (x *scope) locals() *[]*Parm {
	if x == nil {
		return nil
	}
	if x.block != nil {
		return &x.block.Locals
	}
	if f, ok := x.def.(*Fun); ok {
		return &f.Locals
	}
	return x.parent.locals()
}

type checkError struct {
	loc   Loc
	msg   string
	notes []string
	cause []checkError
}

func (s *state) loc(n Node) Loc { return s.mod.Loc(n) }

func (s *state) newID() string {
	s.next++
	return fmt.Sprintf("$%d", s.next-1)
}

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

// The argument to the returned function,
// if non-empty, only the first element of vs is used.
// It must be a either pointer to a slice of types convertable to error,
// or a pointer to a type convertable to error.
func (x *scope) tr(f string, vs ...interface{}) func(...interface{}) {
	if !x.trace {
		return func(...interface{}) {}
	}
	x.log(f, vs...)
	olddent := x.ident
	x.ident += "---"
	return func(errs ...interface{}) {
		defer func() { x.ident = olddent }()
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

func (x *scope) log(f string, vs ...interface{}) {
	if !x.trace {
		return
	}
	fmt.Printf(x.ident)
	fmt.Printf(f, vs...)
	fmt.Println("")
}
