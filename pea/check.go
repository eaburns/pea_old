package pea

import (
	"fmt"
	"math/big"
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
	x := newScope(mod, opts...)
	errs := checkMod(x, mod)
	if x.state.trace {
		dump(x.state.mods, "")
	}
	return convertErrors(errs)
}

func checkMod(x *scope, mod *Mod) (errs []checkError) {
	defer x.tr("checkMod(%s)", mod.Name)(errs)
	errs = collect(x, nil, []string{mod.Name}, mod)

	// First check types, since expression checking assumes
	// that type definitions have been fully checked
	// and have non-nil TypeName.Type values.
	for _, def := range mod.Defs {
		errs = append(errs, checkDef(x, def)...)
	}
	for _, def := range mod.Defs {
		errs = append(errs, checkDefStmts(x, def)...)
	}
	return errs
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

	next int

	trace bool
	ident string
}

func newState(mod *Mod, opts ...Opt) *state {
	s := &state{
		mod:         mod,
		mods:        newModule(""),
		importer:    defaultImporter,
		onAliasPath: make(map[*Type]bool),
		aliasCycles: make(map[*Type]*checkError),
		methInsts:   make(map[string]interface{}),
		typeInsts:   make(map[string]interface{}),
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
		fmt.Printf("%s %s\n", ident, d)
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

func checkDef(x *scope, def Def) (errs []checkError) {
	defer x.tr("checkDef(%s)", defName(def))(&errs)
	x = &scope{state: x.state, parent: x, def: def}
	switch def := def.(type) {
	case *Import:
		return nil // handled elsewhere
	case *Fun:
		return checkFun(x, def)
	case *Var:
		return checkVar(x, def)
	case *Type:
		return checkType(x, def)
	default:
		panic(fmt.Sprintf("impossible Def type %T", def))
	}
}

func checkDefStmts(x *scope, def Def) (errs []checkError) {
	defer x.tr("checkDefStmts(%s)", def.Name())(&errs)

	x = &scope{state: x.state, parent: x, def: def}
	switch def := def.(type) {
	case *Fun:
		errs = append(errs, checkFunStmts(x, def)...)
	case *Var:
		errs = append(errs, checkVarStmts(x, def)...)
	}
	return errs
}

func checkVar(x *scope, vr *Var) (errs []checkError) {
	defer x.tr("checkVar(%s)", vr.Ident)(&errs)
	if vr.Type != nil {
		errs = append(errs, checkTypeName(x, vr.Type)...)
	}
	// TODO: checkVar only checks the type if the name is explicit.
	// There should be a pass after all checkDefs
	// to check the var def statements in topological order,
	// setting the inferred types.
	return errs
}

func checkVarStmts(x *scope, vr *Var) (errs []checkError) {
	defer x.tr("checkVarStmts(%s)", vr.Ident)(&errs)
	if ss, es := checkStmts(x, vr.Val, vr.Type); len(es) > 0 {
		errs = append(errs, es...)
	} else {
		vr.Val = ss
	}
	if vr.Type == nil {
		typ := builtInType(x, "Nil")
		if len(vr.Val) > 0 {
			if expr, ok := vr.Val[len(vr.Val)-1].(Expr); ok && expr.ExprType() != nil {
				typ = expr.ExprType()
			}
		}
		vr.Type = typeName(typ)
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
	if fun.Recv != nil && fun.RecvType != nil {
		fun.Self = &Parm{
			location: fun.Recv.location,
			Name:     "self",
			Type:     typeName(fun.RecvType),
		}
		seen[fun.Self.Name] = fun.Self
		x = x.push(fun.Self.Name, fun.Self)
	}

	for i := range fun.Parms {
		p := &fun.Parms[i]
		switch prev := seen[p.Name]; {
		case prev != nil:
			err := x.err(p, "parameter %s is redefined", p.Name)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		case p.Name == "_" || p.Name == "":
			break
		default:
			seen[p.Name] = p
			x = x.push(p.Name, p)
		}
		errs = append(errs, checkTypeName(x, p.Type)...)
	}
	if fun.Ret != nil {
		errs = append(errs, checkTypeName(x, fun.Ret)...)
	}

	return errs
}

func checkFunStmts(x *scope, fun *Fun) (errs []checkError) {
	defer x.tr("checkFunStmts(%s)", fun)(&errs)

	if fun.Recv != nil && len(fun.Recv.Parms) > 0 {
		// TODO: checkFunStmts for param receivers is only partially implemented.
		// We should create stub type arguments, instantiate the fun,
		// then check the instance.
		return errs
	}
	if len(fun.TypeParms) > 0 {
		// TODO: checkFunStmts for parameterized funs is unimplemented.
		// We should create stub type arguments, instantiate the fun,
		// then check the instance.
		return errs
	}

	seen := make(map[string]*Parm)
	if fun.Self != nil {
		seen[fun.Self.Name] = fun.Self
		x = x.push(fun.Self.Name, fun.Self)
	}
	for i := range fun.Parms {
		p := &fun.Parms[i]
		if seen[p.Name] == nil && p.Name != "_" {
			seen[p.Name] = p
			x = x.push(p.Name, p)
		}
	}
	if ss, es := checkStmts(x, fun.Stmts, nil); len(es) > 0 {
		errs = append(errs, es...)
	} else {
		fun.Stmts = ss
	}
	return errs
}

func checkRecvSig(x *scope, fun *Fun) (errs []checkError) {
	defer x.tr("checkRecvSig(%s)", fun)(&errs)

	var def Def
	defOrImport := x.mods.find(fun.Mod(), fun.Recv.Name)
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
		errs = append(errs, checkTypeName(x, p.Type)...)
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
			errs = append(errs, checkTypeName(x, p.Type)...)
		}
		// TODO: checkType for param types is only partially implemented.
		// We should create stub type arguments, instantiate the type,
		// then check the instance.
		return errs
	}

	switch {
	case typ.Alias != nil:
		errs = append(errs, checkAliasType(x, typ)...)
	case typ.Fields != nil:
		errs = append(errs, checkFields(x, typ.Fields)...)
	case typ.Cases != nil:
		errs = append(errs, checkCases(x, typ.Cases)...)
	case typ.Virts != nil:
		errs = append(errs, checkVirts(x, typ.Virts)...)
	}
	return errs
}

func checkAliasType(x *scope, typ *Type) (errs []checkError) {
	defer x.tr("checkAliasType(%s)", typ)(&errs)
	if _, ok := x.aliasCycles[typ]; ok {
		// This alias is already found to be on a cycle.
		// The error is returned at the root of the cycle.
		return errs
	}
	if x.onAliasPath[typ] {
		markAliasCycle(x, x.aliasPath, typ)
		// The error is returned at the root of the cycle.
		return errs
	}

	x.onAliasPath[typ] = true
	x.aliasPath = append(x.aliasPath, typ)
	defer func() {
		delete(x.onAliasPath, typ)
		x.aliasPath = x.aliasPath[:len(x.aliasPath)-1]
	}()

	if errs = checkTypeName(x, typ.Alias); len(errs) > 0 {
		return errs
	}
	// checkTypeName can make a recursive call to checkAliasType.
	// In the case of an alias cycle, that call would have added the error
	// to the aliasCycles map for the root alias definition.
	// We return any such error here.
	if err := x.aliasCycles[typ]; err != nil {
		errs = append(errs, *err)
	}
	return errs
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
		errs = append(errs, checkTypeName(x, p.Type)...)
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
			errs = append(errs, checkTypeName(x, p.Type)...)
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
			errs = append(errs, checkTypeName(x, &sig.Parms[j])...)
		}
		if sig.Ret != nil {
			errs = append(errs, checkTypeName(x, sig.Ret)...)
		}
	}
	return errs
}

func checkTypeName(x *scope, name *TypeName) (errs []checkError) {
	defer x.tr("checkTypeName(%s)", name)(&errs)

	if name.Var {
		x.log("type variable")
		// TODO: checkTypeName on var should go away,
		// once we fully instantiate Type and Fun before checking,
		// it should be impossible to hit this case.
		return nil
	}

	for i := range name.Args {
		errs = append(errs, checkTypeName(x, &name.Args[i])...)
	}

	if name.Mod == nil {
		name.Mod = x.modPath()
	}

	var def Def
	defOrImport := x.mods.find(*name.Mod, name.Name)
	switch d := defOrImport.(type) {
	case builtin:
		def = d.Def
	case imported:
		def = d.Def
	case Def:
		def = d
	case nil:
		err := x.err(name, "type %s is undefined", name)
		errs = append(errs, *err)
		return errs
	}

	typ, ok := def.(*Type)
	if !ok {
		err := x.err(name, "got %s, expected a type", def.kind())
		addDefNotes(err, x, defOrImport)
		return append(errs, *err)
	}

	if len(typ.Sig.Parms) > 0 {
		var es []checkError
		if typ, es = typ.inst(x, *name); len(es) > 0 {
			err := x.err(name, "%s cannot be instantiated", name)
			err.cause = es
			return append(errs, *err)
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
	return errs
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

func checkStmts(x *scope, stmts []Stmt, blockRes *TypeName) (_ []Stmt, errs []checkError) {
	defer x.tr("checkStmts(…)")(&errs)

	var out []Stmt
	for i := range stmts {
		switch stmt := stmts[i].(type) {
		case *Ret:
			if es := checkRet(x, stmt); len(es) > 0 {
				errs = append(errs, es...)
			}
			out = append(out, stmt)
		case *Assign:
			var ss []Stmt
			var es []checkError
			if x, ss, es = checkAssign(x, stmt); len(es) > 0 {
				out = append(out, stmt)
				errs = append(errs, es...)
			} else {
				out = append(out, ss...)
			}
		case Expr:
			var infer *TypeName
			if i == len(stmts)-1 {
				infer = blockRes
			}
			if expr, es := checkExpr(x, stmt, infer); len(es) > 0 {
				out = append(out, stmt)
				errs = append(errs, es...)
			} else {
				out = append(out, expr)
			}
		}
	}

	return out, errs
}

func checkRet(x *scope, ret *Ret) (errs []checkError) {
	defer x.tr("checkRet(…)")(&errs)

	var infer *TypeName
	if fun := x.fun(); fun == nil {
		err := x.err(ret, "return outside of a method")
		errs = append(errs, *err)
	} else {
		infer = fun.Ret
	}
	if expr, es := checkExpr(x, ret.Val, infer); len(es) > 0 {
		errs = append(errs, es...)
	} else {
		ret.Val = expr
	}
	return errs
}

func checkAssign(x *scope, as *Assign) (_ *scope, _ []Stmt, errs []checkError) {
	defer x.tr("checkAssign(…)")(&errs)

	if es := checkAssignCount(x, as); len(es) > 0 {
		errs = append(errs, es...)
		return x, []Stmt{as}, errs
	}
	x, ss, ass, es := splitAssign(x, as)
	if len(es) > 0 {
		errs = append(errs, es...)
	}
	x1 := x
	for _, as := range ass {
		if x1, es = checkAssign1(x, x1, as); len(es) > 0 {
			errs = append(errs, es...)
		}
	}
	return x1, ss, errs
}

func checkAssignCount(x *scope, as *Assign) (errs []checkError) {
	defer x.tr("checkAssignCount(…)")(&errs)
	if len(as.Vars) == 1 {
		return nil
	}
	c, ok := as.Val.(Call)
	if ok && len(c.Msgs) == len(as.Vars) {
		return nil
	}
	got := 1
	if ok {
		got = len(c.Msgs)
	}
	// This is best-effort to report any errors,
	// but since the assignment mismatches
	// the infer type must always be nil.
	if _, es := checkExpr(x, as.Val, nil); len(es) > 0 {
		errs = append(errs, es...)
	}
	for _, p := range as.Vars {
		if p.Type != nil {
			errs = append(errs, checkTypeName(x, p.Type)...)
		}
	}
	err := x.err(as, "assignment count mismatch: got %d, expected %d",
		got, len(as.Vars))
	errs = append(errs, *err)
	return errs
}

func splitAssign(x *scope, as *Assign) (_ *scope, ss []Stmt, ass []*Assign, errs []checkError) {
	defer x.tr("splitAssign(…)")(&errs)

	if len(as.Vars) == 1 {
		return x, []Stmt{as}, []*Assign{as}, nil
	}

	call := as.Val.(Call) // must, because checkAssignCount was OK
	recv := call.Recv
	if expr, ok := recv.(Expr); ok {
		tmp := x.newID()
		loc := location{start: expr.Start(), end: expr.End()}
		p := &Parm{location: loc, Name: tmp}
		a := &Assign{Vars: []*Parm{p}, Val: expr}
		if e, es := checkExpr(x, expr, nil); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			a.Val = e
		}
		ss = append(ss, a)
		locals := x.locals() // cannot be nil
		*locals = append(*locals, p)
		x = x.push(tmp, p)
		recv = Ident{location: loc, Text: tmp}
	}
	for i, p := range as.Vars {
		as := &Assign{
			Vars: []*Parm{p},
			Val: Call{
				location: call.Msgs[i].location,
				Recv:     recv,
				Msgs:     []Msg{call.Msgs[i]},
			},
		}
		ss = append(ss, as)
		ass = append(ass, as)
	}
	return x, ss, ass, errs
}

func checkAssign1(x, x1 *scope, as *Assign) (_ *scope, errs []checkError) {
	defer x.tr("checkAssign1(%s)", as.Vars[0].Name)(&errs)

	vr := as.Vars[0]
	def, _ := x.find(vr.Name).(*Parm)
	if def == nil {
		locals := x.locals() // cannot be nil
		*locals = append(*locals, vr)
		x1 = x1.push(vr.Name, vr)
		def = vr
	}
	if vr.Type != nil {
		errs = append(errs, checkTypeName(x, vr.Type)...)
		if vr != def {
			err := x.err(vr, "%s is redefined", vr.Name)
			note(err, "previous definition is at %s", x.loc(def))
			errs = append(errs, *err)
		}
	}
	as.Vars[0] = def

	var infer *TypeName
	if vr.Type != nil {
		infer = vr.Type
	}

	if expr, es := checkExpr(x, as.Val, infer); len(es) > 0 {
		errs = append(errs, es...)
	} else {
		as.Val = expr
	}
	if vr.Type == nil && as.Val.ExprType() != nil {
		vr.Type = typeName(as.Val.ExprType())
	}
	return x1, errs
}

func typeName(typ *Type) *TypeName {
	mp := typ.Mod()
	var args []TypeName
	if typ.Sig.Args != nil {
		for i := range typ.Sig.Parms {
			args = append(args, typ.Sig.Args[&typ.Sig.Parms[i]])
		}
	}
	return &TypeName{
		location: typ.location,
		Mod:      &mp,
		Name:     typ.Sig.Name,
		Args:     args,
		Type:     typ,
	}
}

// checkExpr checks the expression.
// If infer is non-nil, and the expression type is convertable to infer,
// a conversion node is added.
// If infer is non-nil, and the experssion type is not convertable to infer,
// a type-mismatch error is returned.
func checkExpr(x *scope, expr Expr, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("checkExpr(infer=%s)", infer)(&errs)
	// TODO: implement type conversion in checkExpr.
	// TODO: implement type-mismatch checking in checkExpr.
	return expr.check(x, infer)
}

func (n Call) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Call.check(…)")(&errs)
	// TODO: Call.check is unimplemented.
	return n, nil
}

func (n Ctor) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Ctor.check(infer=%s)", infer)(&errs)

	errs = append(errs, checkTypeName(x, &n.Type)...)
	switch typ := n.Type.Type; {
	case typ == nil:
		break
	case typ.Alias != nil:
		panic("impossible") // aliases should already be forwarded
	case isAry(typ):
		errs = append(errs, checkAryCtor(x, &n)...)
	case len(typ.Cases) > 0:
		errs = append(errs, checkOrCtor(x, &n)...)
	case len(typ.Virts) > 0:
		errs = append(errs, checkVirtCtor(x, &n)...)
	case isBuiltInType(typ):
		err := x.err(n, "built-in type %s cannot be constructed", typ.Name())
		errs = append(errs, *err)
	default:
		errs = append(errs, checkAndCtor(x, &n)...)
	}

	return n, errs
}

func isAry(t *Type) bool {
	return isBuiltInType(t) && t.Sig.Name == "Array"
}

func isBuiltInType(t *Type) bool {
	return t.ModPath.Root == ""
}

func checkAryCtor(x *scope, n *Ctor) (errs []checkError) {
	defer x.tr("checkAryCtor(%s)", n.Type.Type.Sig.Name)(&errs)

	typ := n.Type.Type
	t := typ.Sig.Args[&typ.Sig.Parms[0]]
	elmType := &t
	x.log("elmType=%s", elmType)

	for i := range n.Args {
		if expr, es := checkExpr(x, n.Args[i], elmType); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Args[i] = expr
		}
	}
	return errs
}

func checkAndCtor(x *scope, n *Ctor) (errs []checkError) {
	defer x.tr("checkAndCtor(%s)", n.Type.Type.Sig.Name)(&errs)

	typ := n.Type.Type
	var sel string
	var inferTypes []*TypeName
	for _, p := range typ.Fields {
		sel += p.Name + ":"
		inferTypes = append(inferTypes, p.Type)
	}
	if sel != n.Sel {
		err := x.err(n, "bad and-type constructor: got %s, expected %s", n.Sel, sel)
		errs = append(errs, *err)
	}
	if n.Sel == "" && len(n.Args) > 0 {
		err := x.err(n, "bad and-type constructor: Nil with non-nil expression")
		errs = append(errs, *err)
	}
	if len(inferTypes) != len(n.Args) {
		inferTypes = make([]*TypeName, len(n.Args))
	}
	for i := range n.Args {
		if expr, es := checkExpr(x, n.Args[i], inferTypes[i]); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Args[i] = expr
		}
	}
	return errs
}

func checkOrCtor(x *scope, n *Ctor) (errs []checkError) {
	defer x.tr("checkOrCtor(%s)", n.Type.Type.Sig.Name)(&errs)
	typ := n.Type.Type
	var cas *Parm
	for i := range typ.Cases {
		c := &typ.Cases[i]
		name := c.Name
		if c.Type != nil {
			name += ":"
		}
		if name == n.Sel {
			cas = c
		}
	}
	if cas == nil {
		err := x.err(n, "bad or-type constructor: no case %s", n.Sel)
		errs = append(errs, *err)
	}
	// Currently n.Args can never be >1 if cas != nil,
	// (cas.Name can contain no more than one :, and n.Sel has one : per-arg).
	// but we still want to check all args to at least report their errors.
	for i := range n.Args {
		var inferType *TypeName
		if i == 0 && cas != nil && cas.Type != nil {
			inferType = cas.Type
		}
		if expr, es := checkExpr(x, n.Args[i], inferType); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Args[i] = expr
		}
	}
	return errs
}

func checkVirtCtor(x *scope, n *Ctor) (errs []checkError) {
	defer x.tr("checkVirtCtor(%s)", n.Type.Type.Sig.Name)(&errs)

	infer := &n.Type
	if n.Sel != "" {
		infer = nil
		err := x.err(n, "a virtual conversion cannot have a selector")
		errs = append(errs, *err)
	}
	if len(n.Args) != 1 {
		infer = nil
		err := x.err(n, "a virtual conversion must have exactly one argument")
		errs = append(errs, *err)
	}

	for i := range n.Args {
		if expr, es := checkExpr(x, n.Args[i], infer); len(es) > 0 {
			errs = append(errs, es...)
		} else {
			n.Args[i] = expr
		}
	}

	// TODO: checkVirtCtor should verify that the type is convertable.

	return errs
}

func (n Block) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Block.check(…)")(&errs)

	for i := range n.Parms {
		p := &n.Parms[i]
		if p.Type != nil {
			errs = append(errs, checkTypeName(x, p.Type)...)
		}
		x = x.push(p.Name, p)
	}

	resInfer := inferBlock(x, &n, infer)

	x = &scope{state: x.state, parent: x, block: &n}
	resultType := builtInType(x, "Nil")
	if ss, es := checkStmts(x, n.Stmts, resInfer); es != nil {
		errs = append(errs, es...)
	} else {
		n.Stmts = ss
		if t := lastStmtExprType(n.Stmts); t != nil {
			resultType = t
		}
	}

	var typeArgs []TypeName
	for _, p := range n.Parms {
		if p.Type == nil {
			err := x.err(p, "cannot infer block parameter type")
			errs = append(errs, *err)
			continue
		}
		typeArgs = append(typeArgs, *p.Type)
	}
	typeArgs = append(typeArgs, *typeName(resultType))

	if max := len(funTypeParms); len(n.Parms) > max {
		err := x.err(n, "too many block parameters (max %d)", max)
		errs = append(errs, *err)
		return n, errs
	}

	// The following condition can only be true
	// if there were un-inferable parameter types,
	// thus len(errs)>0, so it's OK for n.Type to be nil;
	// the type check will not be error-free.
	if len(typeArgs) == len(n.Parms)+1 {
		n.Type = builtInType(x, fmt.Sprintf("Fun%d", len(n.Parms)), typeArgs...)
	}
	return n, errs
}

func inferBlock(x *scope, n *Block, infer *TypeName) *TypeName {
	if infer == nil || infer.Type == nil || infer.Type.ModPath.Root != "" ||
		!strings.HasPrefix(infer.Type.Sig.Name, "Fun") ||
		len(n.Parms) != len(infer.Type.Sig.Parms)-1 {
		return nil
	}
	sig := infer.Type.Sig
	for i := range n.Parms {
		if n.Parms[i].Type == nil {
			p := &sig.Parms[i]
			t := sig.Args[p]
			n.Parms[i].Type = &t
		}
	}
	p := &sig.Parms[len(sig.Parms)-1]
	t := sig.Args[p]
	return &t
}

func lastStmtExprType(ss []Stmt) *Type {
	if len(ss) == 0 {
		return nil
	}
	if e, ok := ss[len(ss)-1].(Expr); ok {
		return e.ExprType()
	}
	return nil
}

func (n Ident) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Ident.check(%s, infer=%s)", n.Text, infer)(&errs)
	if parm, ok := x.find(n.Text).(*Parm); ok {
		n.Parm = parm
		return n, errs
	}

	if fun := x.fun(); fun != nil && fun.RecvType != nil {
		if f := findField(fun.RecvType, n.Text); f != nil {
			n.Parm = f
			n.RecvType = fun.RecvType
			return n, errs
		}
	}

	def, defOrImport := findDef(x, n.Text)
	if def == nil {
		err := x.err(n, "undefined: %s", n.Text)
		errs = append(errs, *err)
		return n, errs
	}

	if v, ok := def.(*Var); ok {
		n.Var = v
		return n, errs
	}

	if fun, ok := def.(*Fun); ok {
		if len(fun.Parms) > 0 || fun.Recv != nil {
			// The name is just an ident,
			// it cannot have a receiver or params.
			panic("impossible")
		}
		loc := n.location
		call := &Call{
			location: loc,
			Msgs:     []Msg{{location: loc, Sel: n.Text}},
		}
		expr, es := call.check(x, infer)
		return expr, append(errs, es...)
	}

	err := x.err(n, "got %s, expected a variable or 0-ary function", def.kind())
	addDefNotes(err, x, defOrImport)
	errs = append(errs, *err)
	return n, errs
}

func findDef(x *scope, name string) (def Def, defOrImport interface{}) {
	defOrImport = x.mods.find(*x.modPath(), name)
	switch d := defOrImport.(type) {
	case nil:
		return nil, nil
	case builtin:
		return d.Def, defOrImport
	case imported:
		return d.Def, defOrImport
	case Def:
		return d, defOrImport
	default:
		panic(fmt.Sprintf("impossible definition type %T", def))
	}
}

func findField(typ *Type, name string) *Parm {
	for i := range typ.Fields {
		f := &typ.Fields[i]
		if f.Name == name {
			return f
		}
	}
	return nil
}

func (n Int) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Int.check(%s, infer=%s)", n.Text, infer)(&errs)

	n.Val = big.NewInt(0)
	if _, ok := n.Val.SetString(n.Text, 10); !ok {
		// If the int is syntactiacally valid,
		// it should be a valid Go int too.
		panic("impossible")
	}

	n.Type = builtInType(x, "Int")
	if ok, signed, bits := isInt(infer); ok {
		x.log("isInt(%s): signed=%v, bits=%v", infer.Type.Sig.Name, signed, bits)
		n.Type = infer.Type
		n.BitLen = bits
		n.Signed = signed

		var zero big.Int
		if !signed && n.Val.Cmp(&zero) < 0 {
			err := x.err(n, "%s cannot represent %s: negative unsigned", infer.Name, n.Text)
			errs = append(errs, *err)
			return n, errs
		}
		if signed {
			bits-- // sign bit
		}
		min := big.NewInt(-(1 << uint(bits)))
		x.log("bits=%d, val.BitLen()=%d, val=%v, min=%v",
			bits, n.Val.BitLen(), n.Val, min)
		if n.Val.BitLen() > bits && (!signed || n.Val.Cmp(min) != 0) {
			err := x.err(n, "%s cannot represent %s: overflow", infer.Name, n.Text)
			errs = append(errs, *err)
			return n, errs
		}
		return n, nil
	}
	if isFloat(infer) {
		return Float{location: n.location, Text: n.Text}.check(x, infer)
	}
	return n, nil
}

func (n Float) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Float.check(…)")(&errs)

	n.Val = big.NewFloat(0)
	if _, _, err := n.Val.Parse(n.Text, 10); err != nil {
		// If the float is syntactiacally valid,
		// it should be a valid Go float too.
		panic("impossible: " + err.Error())
	}

	n.Type = builtInType(x, "Float")
	if isFloat(infer) {
		x.log("isFloat(%s)", infer.Type.Sig.Name)
		n.Type = infer.Type
		return n, nil
	}
	if ok, _, _ := isInt(infer); ok {
		x.log("isInt(%s)", infer.Type.Sig.Name)
		var i big.Int
		if _, acc := n.Val.Int(&i); acc != big.Exact {
			err := x.err(n, "%s cannot represent %s: truncation", infer.Name, n.Text)
			errs = append(errs, *err)
			return n, errs
		}
		return Int{location: n.location, Text: i.String()}.check(x, infer)
	}
	return n, nil
}

func isFloat(t *TypeName) bool {
	if t == nil || t.Type == nil || t.Type.ModPath.Root != "" {
		return false
	}
	return t.Type.Sig.Name == "Float32" || t.Type.Sig.Name == "Float64"
}

func isInt(t *TypeName) (ok bool, signed bool, bits int) {
	if t == nil || t.Type == nil || t.Type.ModPath.Root != "" {
		return false, false, 0
	}
	if n, _ := fmt.Sscanf(t.Type.Sig.Name, "Uint%d", &bits); n == 1 {
		return true, false, bits
	}
	if n, _ := fmt.Sscanf(t.Type.Sig.Name, "Int%d", &bits); n == 1 {
		return true, true, bits
	}
	return false, false, 0
}

func (n Rune) check(x *scope, _ *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Rune.check(…)")(&errs)
	// TODO: Rune.check should error on invalid unicode codepoints.
	n.Type = builtInType(x, "Rune")
	return n, nil
}

func (n String) check(x *scope, _ *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("String.check(…)")(&errs)
	n.Type = builtInType(x, "String")
	return n, nil
}

func builtInType(x *scope, name string, args ...TypeName) *Type {
	tn := TypeName{
		Mod:  &ModPath{},
		Name: name,
		Args: args,
	}
	if errs := checkTypeName(x, &tn); len(errs) > 0 {
		panic(fmt.Sprintf("impossible error: %v", errs))
	}
	return tn.Type
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
