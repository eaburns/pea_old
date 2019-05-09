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

	for _, def := range mod.Defs {
		errs = append(errs, checkDef(x, def)...)
	}

	return convertErrors(errs)
}

type state struct {
	mod      *Mod
	mods     *module
	wordSize int
	importer func(string) (*Mod, error)

	aliasPath   []*Type
	onAliasPath map[*Type]bool
	aliasCycles map[*Type]*checkError

	methInsts map[string]interface{}
	typeInsts map[string]interface{}

	trace bool
	ident string
}

func newState(mod *Mod) *state {
	return &state{
		mod:         mod,
		mods:        newModule(""),
		importer:    defaultImporter,
		onAliasPath: make(map[*Type]bool),
		aliasCycles: make(map[*Type]*checkError),
		methInsts:   make(map[string]interface{}),
		typeInsts:   make(map[string]interface{}),
		wordSize:    64,
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

// The return is an imported, a builtin, a Def, or nil.
func (t *module) find(path []string, name string) interface{} {
	switch {
	case t == nil:
		return nil
	case len(path) == 0:
		return t.defs[name]
	case t.kids[path[0]] == nil:
		return nil
	default:
		def := t.kids[path[0]].find(path[1:], name)
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

	if es := collect(x, nil, []string{m.Name}, m); len(es) > 0 {
		panic(fmt.Sprintf("error collecting imported module: %v", es))
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

func checkDef(x *scope, def Def) (errs []checkError) {
	defer x.tr("checkDef(%s)", defName(def))(&errs)
	x = &scope{state: x.state, parent: x, def: def}
	switch def := def.(type) {
	case *Import:
		return nil // handled elsewhere
	case *Fun:
		return checkFun(x, def)
	case *Var:
		// TODO: checkDef(*Var) is unimplemented.
	case *Type:
		return checkType(x, def)
	default:
		panic(fmt.Sprintf("impossible Def type %T", def))
	}
	return errs
}

func checkFun(x *scope, fun *Fun) (errs []checkError) {
	defer x.tr("checkFun(%s)", fun)(&errs)

	if fun.Recv != nil {
		errs = append(errs, checkRecvSig(x, fun)...)

		if len(fun.Recv.Parms) > 0 {
			// TODO: checkFun for param receivers is only partially implemented.
			// We should create stub type arguments, instantiate the fun,
			// then check the instance.
			return errs
		}
	}
	if len(fun.TypeParms) > 0 {
		// TODO: checkFun for parameterized funs is unimplemented.
		// We should create stub type arguments, instantiate the fun,
		// then check the instance.
		return errs
	}

	seen := make(map[string]*Parm)
	for i := range fun.Parms {
		p := &fun.Parms[i]
		switch prev := seen[p.Name]; {
		case prev != nil:
			err := x.err(p, "parameter %s is redefined", p.Name)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		case p.Name == "_":
			break
		default:
			seen[p.Name] = p
		}
		if err := checkTypeName(x, p.Type); err != nil {
			errs = append(errs, *err)
		}
	}
	if fun.Ret != nil {
		if err := checkTypeName(x, fun.Ret); err != nil {
			errs = append(errs, *err)
		}
	}

	// TODO: checkFun checking statements is unimplemented.

	return errs
}

func checkRecvSig(x *scope, fun *Fun) (errs []checkError) {
	defer x.tr("checkTypeSig(%s)", fun)(&errs)

	var def Def
	path := append([]string{fun.Mod().Root}, fun.Mod().Path...)
	defOrImport := x.mods.find(path, fun.Recv.Name)
	switch d := defOrImport.(type) {
	case builtin:
		def = d.Def
	case imported:
		def = d.Def
	case Def:
		def = d
	case nil:
		err := x.err(fun, "type %s is undefined", fun.Recv.Name)
		errs = append(errs, *err)
	}

	var typ *Type
	if def != nil {
		var ok bool
		if typ, ok = def.(*Type); !ok {
			err := x.err(fun, "got %s, expected a type", def.kind())
			addDefNotes(err, x, defOrImport)
			errs = append(errs, *err)
		}
	}

	if typ != nil {
		if len(fun.Recv.Parms) != len(typ.Sig.Parms) {
			err := x.err(fun, "parameter count mismatch: got %d, expected %d",
				len(fun.Recv.Parms), len(typ.Sig.Parms))
			addDefNotes(err, x, defOrImport)
			errs = append(errs, *err)
		}
		for i := range typ.Virts {
			v := &typ.Virts[i]
			if v.Sel == fun.Sel {
				err := x.err(fun, "method %s is redefined", fun.Sel)
				note(err, "previous definition is a virtual method")
				addDefNotes(err, x, defOrImport)
				errs = append(errs, *err)
			}
		}
		fun.RecvType = typ
	}

	for i := range fun.Recv.Parms {
		p := &fun.Recv.Parms[i]
		if p.Type == nil {
			continue
		}
		if err := checkTypeName(x, p.Type); err != nil {
			errs = append(errs, *err)
		}
	}
	return errs
}

func checkType(x *scope, typ *Type) (errs []checkError) {
	defer x.tr("checkType(%s)", typ)(&errs)

	if len(typ.Sig.Parms) > 0 {
		for i := range typ.Sig.Parms {
			p := &typ.Sig.Parms[i]
			if p.Type == nil {
				continue
			}
			if err := checkTypeName(x, p.Type); err != nil {
				errs = append(errs, *err)
			}
		}
		// TODO: checkType for param types is only partially implemented.
		// We should create stub type arguments, instantiate the type,
		// then check the instance.
		return errs
	}

	switch {
	case typ.Alias != nil:
		if err := checkAliasType(x, typ); err != nil {
			errs = append(errs, *err)
		}
	case typ.Fields != nil:
		errs = append(errs, checkFields(x, typ.Fields)...)
	case typ.Cases != nil:
		errs = append(errs, checkCases(x, typ.Cases)...)
	case typ.Virts != nil:
		errs = append(errs, checkVirts(x, typ.Virts)...)
	}
	return errs
}

func checkAliasType(x *scope, typ *Type) (err *checkError) {
	defer x.tr("checkAliasType(%s)", typ)(&err)
	if _, ok := x.aliasCycles[typ]; ok {
		// This alias is already found to be on a cycle.
		// The error is returned at the root of the cycle.
		return err
	}
	if x.onAliasPath[typ] {
		markAliasCycle(x, x.aliasPath, typ)
		// The error is returned at the root of the cycle.
		return err
	}

	x.onAliasPath[typ] = true
	x.aliasPath = append(x.aliasPath, typ)
	defer func() {
		delete(x.onAliasPath, typ)
		x.aliasPath = x.aliasPath[:len(x.aliasPath)-1]
	}()

	if err = checkTypeName(x, typ.Alias); err != nil {
		return err
	}
	// checkTypeName can make a recursive call to checkAliasType.
	// In the case of an alias cycle, that call would have added the error
	// to the aliasCycles map for the root alias definition.
	// We return any such error here.
	return x.aliasCycles[typ]
}

func markAliasCycle(x *scope, path []*Type, alias *Type) {
	err := x.err(alias, "type alias cycle")
	i := len(x.aliasPath) - 1
	for x.aliasPath[i] != alias {
		i--
	}
	for ; i < len(x.aliasPath); i++ {
		note(err, "%s at %s", defName(x.aliasPath[i]), x.loc(x.aliasPath[i]))
		// We only want to report the cycle error for one definition.
		// Mark all other nodes on the cycle with a nil error.
		x.aliasCycles[x.aliasPath[i]] = nil
	}
	note(err, "%s at %s", defName(alias), x.loc(alias))
	x.aliasCycles[alias] = err
}

func checkFields(x *scope, ps []Parm) (errs []checkError) {
	defer x.tr("checkFields(…)")(&errs)
	seen := make(map[string]*Parm)
	for i := range ps {
		p := &ps[i]
		if prev := seen[p.Name]; prev != nil {
			err := x.err(p, "field %s is redefined", p.Name)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[p.Name] = p
		}
		if err := checkTypeName(x, p.Type); err != nil {
			errs = append(errs, *err)
		}
	}
	return errs
}

func checkCases(x *scope, ps []Parm) (errs []checkError) {
	defer x.tr("checkCases(…)")(&errs)
	seen := make(map[string]*Parm)
	for i := range ps {
		p := &ps[i]
		lower := strings.ToLower(p.Name)
		if prev := seen[lower]; prev != nil {
			err := x.err(p, "case %s is redefined", lower)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[lower] = p
		}
		if p.Type != nil {
			if err := checkTypeName(x, p.Type); err != nil {
				errs = append(errs, *err)
			}
		}
	}
	return errs
}

func checkVirts(x *scope, sigs []MethSig) (errs []checkError) {
	defer x.tr("checkVirts(…)")(&errs)
	seen := make(map[string]*MethSig)
	for i := range sigs {
		sig := &sigs[i]
		if prev, ok := seen[sig.Sel]; ok {
			err := x.err(sig, "virtual method %s is redefined", sig.Sel)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[sig.Sel] = sig
		}
		for j := range sig.Parms {
			if err := checkTypeName(x, &sig.Parms[j]); err != nil {
				errs = append(errs, *err)
			}
		}
		if sig.Ret != nil {
			if err := checkTypeName(x, sig.Ret); err != nil {
				errs = append(errs, *err)
			}
		}
	}
	return errs
}

func checkTypeName(x *scope, name *TypeName) (err *checkError) {
	defer x.tr("checkTypeName(%s)", name)(err)

	if name.Mod == nil {
		name.Mod = x.modPath()
	}

	var def Def
	path := append([]string{name.Mod.Root}, name.Mod.Path...)
	defOrImport := x.mods.find(path, name.Name)
	switch d := defOrImport.(type) {
	case builtin:
		def = d.Def
	case imported:
		def = d.Def
	case Def:
		def = d
	case nil:
		return x.err(name, "type %s is undefined", name)
	}

	typ, ok := def.(*Type)
	if !ok {
		err = x.err(name, "got %s, expected a type", def.kind())
		addDefNotes(err, x, defOrImport)
		return err
	}

	if len(typ.Sig.Parms) > 0 {
		var es []checkError
		if typ, es = typ.inst(x, *name); len(es) > 0 {
			err = x.err(name, "%s cannot be instantiated", name)
			err.cause = es
			return err
		}
	}

	if typ.Alias != nil {
		if checkAliasType(x, typ) != nil {
			// Return nil, because the error is reported
			// by the call to checkAliasType from its definition.
			return nil
		}
		name.Type = typ.Alias.Type
	} else {
		name.Type = typ
	}
	return nil
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
