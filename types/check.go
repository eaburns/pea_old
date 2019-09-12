package types

import (
	"fmt"
	"math/big"
	"path"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pretty"
)

// Config are configuration parameters for the type checker.
type Config struct {
	// IntSize is the bit size of the Int, UInt, and Word alias types.
	// It must be a valid int size: 8, 16, 32, or 64 (default=64).
	IntSize int
	// FloatSize is the bit size of the Float type alias.
	// It must be a valid float size: 32 or 64 (default=64).
	FloatSize int
	// Importer is used for importing modules.
	// The default importer reads packages from the local file system.
	Importer Importer
	// Trace is whether to enable debug tracing.
	Trace bool
}

// Check type-checks an AST and returns the type-checked tree or errors.
func Check(astMod *ast.Mod, cfg Config) (*Mod, []error) {
	x := newUnivScope(newState(cfg, astMod))
	mod, errs := check(x, astMod)
	if len(errs) > 0 {
		return nil, convertErrors(errs)
	}
	return mod, nil
}

func check(x *scope, astMod *ast.Mod) (_ *Mod, errs []checkError) {
	defer x.tr("check(%s)", astMod.Name)(&errs)

	mod := &Mod{AST: astMod}
	x = x.new()
	x.mod = mod

	mod.Defs, errs = makeDefs(x, astMod.Files)
	errs = append(errs, checkDups(x, mod.Defs)...)
	errs = append(errs, gatherDefs(x, mod.Defs)...)
	errs = append(errs, checkDupMeths(x, mod.Defs)...)
	errs = append(errs, checkDefs(x, mod.Defs)...)

	return mod, errs
}

func makeDefs(x *scope, files []ast.File) ([]Def, []checkError) {
	var defs []Def
	var errs []checkError
	for i := range files {
		file := &file{ast: &files[i]}
		file.x = x.new()
		file.x.file = file
		errs = append(errs, imports(x.state, file)...)
		for _, astDef := range file.ast.Defs {
			def := makeDef(astDef)
			defs = append(defs, def)
			x.defFiles[def] = file
		}
	}
	return defs, errs
}

func imports(x *state, file *file) []checkError {
	var errs []checkError
	for _, astImp := range file.ast.Imports {
		p := astImp.Path[1 : len(astImp.Path)-1] // trim "
		x.log("importing %s", p)
		defs, err := x.cfg.Importer.Import(x.cfg, p)
		if err != nil {
			errs = append(errs, *x.err(astImp, err.Error()))
			continue
		}
		file.imports = append(file.imports, imp{
			path: p,
			name: path.Base(p),
			defs: defs,
		})
	}
	return errs
}

// checkDups returns redefinition errors for types, vals, and funs.
// It doesn't check duplicate methods.
func checkDups(x *scope, defs []Def) (errs []checkError) {
	defer x.tr("checkDups")(&errs)

	seen := make(map[string]Def)
	types := make(map[string]Def)
	for _, def := range defs {
		var id string
		switch def := def.(type) {
		case *Val:
			id = def.Name
		case *Type:
			id = def.Sig.Name
			tid := fmt.Sprintf("(%d)%s", def.Sig.Arity, def.Sig.Name)
			if prev, ok := types[tid]; ok {
				err := x.err(def, "type %s redefined", tid)
				note(err, "previous definition is at %s", x.loc(prev))
				errs = append(errs, *err)
				continue
			}
			types[tid] = def
			if _, ok := seen[id].(*Type); ok {
				// Multiple defs of the same type name are OK
				// as long as their arity is different.
				continue
			}
		case *Fun:
			if def.ast.Recv != nil {
				continue // check dup methods separately.
			}
			id = def.Sig.Sel
		default:
			panic(fmt.Sprintf("impossible type %T", def))
		}
		if prev, ok := seen[id]; ok {
			err := x.err(def, "%s redefined", id)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		}
		seen[id] = def
	}
	return errs
}

func makeDef(astDef ast.Def) Def {
	switch astDef := astDef.(type) {
	case *ast.Val:
		return &Val{
			ast:  astDef,
			priv: astDef.Priv(),
			Name: astDef.Ident,
		}
	case *ast.Fun:
		return &Fun{
			ast:  astDef,
			priv: astDef.Priv(),
			Sig: FunSig{
				ast: &astDef.Sig,
				Sel: astDef.Sig.Sel,
			},
		}
	case *ast.Type:
		return &Type{
			ast:  astDef,
			priv: astDef.Priv(),
			Sig: TypeSig{
				ast:   &astDef.Sig,
				Arity: len(astDef.Sig.Parms),
				Name:  astDef.Sig.Name,
			},
		}
	default:
		panic(fmt.Sprintf("impossible type %T", astDef))
	}
}

func checkDupMeths(x *scope, defs []Def) []checkError {
	var errs []checkError
	seen := make(map[string]Def)
	for _, def := range defs {
		fun, ok := def.(*Fun)
		if !ok || fun.Recv == nil || fun.Recv.Type == nil {
			continue
		}
		recv := fun.Recv.Type
		key := recv.name() + " " + fun.Sig.Sel
		if prev, ok := seen[key]; ok {
			err := x.err(def, "method %s redefined", key)
			note(err, "previous definition is at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[key] = def
		}
	}
	return errs
}

func checkDefs(x *scope, defs []Def) []checkError {
	var errs []checkError
	for _, def := range defs {
		errs = append(errs, checkDef(x, def)...)
	}
	return errs
}

func checkDef(x *scope, def Def) []checkError {
	if !x.gathered[def] {
		panic("impossible")
	}
	file, ok := x.defFiles[def]
	if !ok {
		panic("impossible")
	}
	x = file.x

	switch def := def.(type) {
	case *Val:
		return checkVal(x, def)
	case *Fun:
		return checkFun(x, def)
	case *Type:
		return checkType(x, def)
	default:
		panic(fmt.Sprintf("impossible type: %T", def))
	}
}

func checkVal(x *scope, def *Val) (errs []checkError) {
	defer x.tr("checkVal(%s)", def.name())(&errs)
	if def.Type != nil {
		errs = append(errs, checkTypeName(x, def.Type)...)
	}
	var es []checkError
	if def.Init, es = gatherStmts(x, def.ast.Init); len(es) > 0 {
		errs = append(errs, es...)
	}
	errs = append(errs, checkStmts(x, def.Type, def.Init)...)
	return errs
}

func checkFun(x *scope, def *Fun) (errs []checkError) {
	defer x.tr("checkFun(%s)", def.name())(&errs)
	if def.Recv != nil {
		for i := range def.Recv.Parms {
			x = x.new()
			x.typeVar = &def.Recv.Parms[i]
		}
	}
	for i := range def.TParms {
		x = x.new()
		x.typeVar = &def.TParms[i]
	}

	x = x.new()
	x.fun = def
	def.Stmts, errs = gatherStmts(x, def.ast.Stmts)
	return append(errs, checkStmts(x, nil, def.Stmts)...)
}

func checkType(x *scope, def *Type) (errs []checkError) {
	defer x.tr("checkType(%s)", def.name())(&errs)
	switch {
	case def.Alias != nil:
		errs = checkTypeName(x, def.Alias)
	case def.Fields != nil:
		errs = checkFields(x, def.Fields)
	case def.Cases != nil:
		errs = checkCases(x, def.Cases)
	case def.Virts != nil:
		errs = checkVirts(x, def.Virts)
	}
	return errs
}

func checkFields(x *scope, fields []Var) []checkError {
	var errs []checkError
	seen := make(map[string]*Var)
	for i := range fields {
		field := &fields[i]
		if prev, ok := seen[field.Name]; ok {
			err := x.err(field, "field %s redefined", field.Name)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[field.Name] = field
		}
		errs = append(errs, checkTypeName(x, field.Type)...)
	}
	return errs
}

func checkCases(x *scope, cases []Var) []checkError {
	var errs []checkError
	seen := make(map[string]*Var)
	for i := range cases {
		cas := &cases[i]
		if prev, ok := seen[cas.Name]; ok {
			err := x.err(cas, "case %s redefined", cas.Name)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[cas.Name] = cas
		}
		if cas.Type != nil {
			errs = append(errs, checkTypeName(x, cas.Type)...)
		}
	}
	return errs
}

func checkVirts(x *scope, virts []FunSig) []checkError {
	var errs []checkError
	seen := make(map[string]*FunSig)
	for i := range virts {
		virt := &virts[i]
		if prev, ok := seen[virt.Sel]; ok {
			err := x.err(virt, "virtual method %s redefined", virt.Sel)
			note(err, "previous definition at %s", x.loc(prev))
			errs = append(errs, *err)
		} else {
			seen[virt.Sel] = virt
		}
		for i := range virt.Parms {
			parm := &virt.Parms[i]
			errs = append(errs, checkTypeName(x, parm.Type)...)
		}
		if virt.Ret != nil {
			errs = append(errs, checkTypeName(x, virt.Ret)...)
		}
	}
	return errs
}

func checkTypeName(x *scope, name *TypeName) (errs []checkError) {
	defer x.tr("checkTypeName(%s)", name.ID())(&errs)
	// TODO: implement checkTypeName.
	return errs
}

// want is the type of the result of the last statement in the case that it's an expression.
func checkStmts(x *scope, want *TypeName, stmts []Stmt) []checkError {
	var errs []checkError
	for i, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *Ret:
			errs = append(errs, checkRet(x, stmt)...)
		case *Assign:
			errs = append(errs, checkAssign(x, stmt)...)
			x = x.new()
			x.local = stmt
		case Expr:
			var es []checkError
			if i == len(stmts)-1 {
				stmts[i], es = checkExprWant(x, stmt, want)
			} else {
				stmts[i], es = checkExpr(x, stmt, nil)
			}
			errs = append(errs, es...)
		default:
			panic(fmt.Sprintf("impossible type: %T", stmt))
		}
	}
	return errs
}

func checkRet(x *scope, ret *Ret) (errs []checkError) {
	defer x.tr("checkRet(…)")(&errs)
	fun := x.function()
	if fun == nil {
		err := x.err(ret, "return outside of a function or method")
		ret.Val, errs = checkExpr(x, ret.Val, nil)
		return append(errs, *err)
	}
	ret.Val, errs = checkExprWant(x, ret.Val, fun.Sig.Ret)
	return errs
}

func checkAssign(x *scope, ass *Assign) (errs []checkError) {
	defer x.tr("checkAssign(%s)", ass.Var.Name)(&errs)
	if ass.Var.Type != nil {
		errs = checkTypeName(x, ass.Var.Type)
	}
	x.log(pretty.String(ass))
	if ass.Val == nil {
		// ass.Val can be nil in the case of assignment count mismatch.
		// We still want to check the type above, but then we are done.
		return errs
	}
	var es []checkError
	if ass.Var.Type == nil {
		ass.Val, es = checkExpr(x, ass.Val, nil)
	} else {
		ass.Val, es = checkExprWant(x, ass.Val, ass.Var.Type)
	}
	return append(errs, es...)
}

func checkExprWant(x *scope, expr Expr, want *TypeName) (Expr, []checkError) {
	expr, errs := checkExpr(x, expr, want)
	// TODO: implement checkExprWant
	return expr, errs
}

func checkExpr(x *scope, expr Expr, infer *TypeName) (_ Expr, errs []checkError) {
	return expr.check(x, infer)
}

func (expr *Call) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Call.check(infer=%s)", infer)(&errs)
	// TODO: implement Call.check.
	return expr, nil
}

func (expr *Ctor) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Ctor.check(infer=%s)", infer)(&errs)
	// TODO: implement Ctor.check.
	return expr, nil
}

func (expr *Block) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Block.check(infer=%s)", infer)(&errs)
	// TODO: implement Block.check.
	return expr, nil
}

func (expr *Ident) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Ident.check(infer=%s)", infer)(&errs)
	// TODO: implement Ident.check.
	return expr, nil
}

func (expr *Int) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Int.check(infer=%s)", infer)(&errs)
	switch {
	case isFloat(x, infer):
		var f big.Float
		f.Int(expr.Val)
		return checkExpr(x, &Float{ast: expr.ast, Val: &f}, infer)
	case isInt(x, infer):
		expr.typ = infer.Type
	default:
		expr.typ = builtInType(x, "Int")
	}
	signed, bits := disectInt(x, expr.typ)
	x.log("signed=%v, bits=%v", signed, bits)
	if !signed && expr.Val.Cmp(&big.Int{}) < 0 {
		err := x.err(expr, "type %s cannot represent %s: negative unsigned",
			expr.typ, expr.Val)
		return expr, append(errs, *err)
	}
	min := big.NewInt(-(1 << uint(bits)))
	x.log("val=%v, val.BitLen()=%d, min=%v",
		expr.Val, expr.Val.BitLen(), min)
	if expr.Val.BitLen() > bits && (!signed || expr.Val.Cmp(min) != 0) {
		err := x.err(expr, "type %s cannot represent %s: overflow",
			expr.typ, expr.Val)
		return expr, append(errs, *err)
	}
	return expr, errs
}

func disectInt(x *scope, typ *Type) (bool, int) {
	switch typ {
	case builtInType(x, "Int8"):
		return true, 7
	case builtInType(x, "Int16"):
		return true, 15
	case builtInType(x, "Int32"):
		return true, 31
	case builtInType(x, "Int64"):
		return true, 63
	case builtInType(x, "UInt8"):
		return false, 8
	case builtInType(x, "UInt16"):
		return false, 16
	case builtInType(x, "UInt32"):
		return false, 32
	case builtInType(x, "UInt64"):
		return false, 64
	default:
		panic(fmt.Sprintf("impossible int type: %T", typ))
	}
}

func (expr *Float) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("Float.check(infer=%s)", infer)(&errs)
	switch {
	case isInt(x, infer):
		var i big.Int
		if _, acc := expr.Val.Int(&i); acc != big.Exact {
			err := x.err(expr, "type %s cannot represent %s: truncation", infer, expr.Val.String())
			return expr, append(errs, *err)
		}
		return checkExpr(x, &Int{ast: expr.ast, Val: &i}, infer)
	case isFloat(x, infer):
		expr.typ = infer.Type
	default:
		expr.typ = builtInType(x, "Float")
	}
	return expr, nil
}

func isInt(x *scope, name *TypeName) bool {
	switch {
	case name == nil || name.Type == nil:
		return false
	default:
		return false
	case name.Type == builtInType(x, "Int8") ||
		name.Type == builtInType(x, "Int16") ||
		name.Type == builtInType(x, "Int32") ||
		name.Type == builtInType(x, "Int64") ||
		name.Type == builtInType(x, "UInt8") ||
		name.Type == builtInType(x, "UInt16") ||
		name.Type == builtInType(x, "UInt32") ||
		name.Type == builtInType(x, "UInt64"):
		return true
	}
}

func isFloat(x *scope, name *TypeName) bool {
	switch {
	case name == nil || name.Type == nil:
		return false
	default:
		return false
	case name.Type == builtInType(x, "Float32") ||
		name.Type == builtInType(x, "Float64"):
		return true
	}
}

func (expr *String) check(x *scope, infer *TypeName) (_ Expr, errs []checkError) {
	defer x.tr("String.check(infer=%s)", infer)(&errs)
	expr.typ = builtInType(x, "String")
	return expr, nil
}

func builtInType(x *scope, name string, args ...TypeName) *Type {
	// Silence tracing for looking up built-in types.
	savedTrace := x.cfg.Trace
	x.cfg.Trace = false
	defer func() { x.cfg.Trace = savedTrace }()

	for x.univ == nil {
		x = x.up
	}
	typ := findType(len(args), name, x.univ)
	if typ == nil {
		panic(fmt.Sprintf("built-in type (%d)%s not found", len(args), name))
	}
	typ, errs := instType(x, typ, args)
	if len(errs) > 0 {
		panic(fmt.Sprintf("failed to inst built-in type: %v", errs))
	}
	return typ
}
