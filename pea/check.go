package pea

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// An Opt is an option to the type checker.
type Opt func(*state)

var (
	// Trace enables tracing of the type checker.
	Trace Opt = func(x *state) { x.trace = true }
)

// Check type-checks the module.
// Check modifies its arugment, performing some simplifications on the AST
// and populating several fields not set by parsing.
func Check(mod *Mod, opts ...Opt) []error {
	s := &state{mod: mod, defs: make(map[[2]string]Def)}
	for _, opt := range opts {
		opt(s)
	}
	errs := checkMod(&ctx{state: s}, mod)
	return convertErrors(errs)
}

func checkMod(x *ctx, mod *Mod) (errs []checkError) {
	defer x.tr("checkMod(…)")(errs)
	for _, file := range mod.Files {
		if es := checkFile(x, file); len(es) > 0 {
			errs = append(errs, es...)
		}
	}
	return errs
}

func checkFile(x *ctx, file *File) (errs []checkError) {
	defer x.tr("checkFile(%s)", file.Path)(errs)
	return checkDefs(x, file.Defs)
}

func checkDefs(x *ctx, defs []Def) (errs []checkError) {
	defer x.tr("checkDefs(…)")(errs)
	for _, def := range defs {
		// TODO: resolve imports before checking defs.
		if _, ok := def.(*Import); ok {
			continue
		}
		k := key(def)
		if prev, ok := x.defs[k]; ok {
			err := x.err(def, "%s redefined", k[0]+" "+k[1])
			note(err, "previous definition %s", x.loc(prev))
			errs = append(errs, *err)
			continue
		}
		x.defs[k] = def
	}
	return errs
}

type state struct {
	mod *Mod
	// Defs maps <ModPath, Name> → Def for fun, var, and type definitions.
	defs map[[2]string]Def

	trace bool
	ident string
}

// key returns a state.defs map key from a Def.
func key(d Def) [2]string { return [2]string{d.Mod().String(), d.Name()} }

type ctx struct {
	*state
	parent *ctx
	finder finder
}

type finder interface {
	find(string) Node
}

type checkError struct {
	loc   Loc
	msg   string
	notes []string
	cause []checkError
}

func (s *state) loc(n Node) Loc { return s.mod.Loc(n) }

func (x *ctx) err(n Node, f string, vs ...interface{}) *checkError {
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
func (x *ctx) tr(f string, vs ...interface{}) func(...interface{}) {
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

func (x *ctx) log(f string, vs ...interface{}) {
	if !x.trace {
		return
	}
	fmt.Printf(x.ident)
	fmt.Printf(f, vs...)
	fmt.Println("")
}
