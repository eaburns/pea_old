package types

import (
	"fmt"
	"math/big"
	"path"

	"github.com/eaburns/pea/ast"
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
			id = def.Var.Name
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
		val := &Val{
			ast:  astDef,
			Priv: astDef.Priv(),
			Var: Var{
				ast:  &astDef.Var,
				Name: astDef.Var.Name,
			},
		}
		val.Var.Val = val
		return val
	case *ast.Fun:
		return &Fun{
			ast:  astDef,
			Priv: astDef.Priv(),
			Sig: FunSig{
				ast: &astDef.Sig,
				Sel: astDef.Sig.Sel,
			},
		}
	case *ast.Type:
		return &Type{
			ast:  astDef,
			Priv: astDef.Priv(),
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
	if def.Var.TypeName != nil {
		errs = append(errs, checkTypeName(x, def.Var.TypeName)...)
		def.Var.typ = def.Var.TypeName.Type
	}

	x = x.new()
	x.val = def

	var es []checkError
	if def.Init, es = gatherStmts(x, def.Var.typ, def.ast.Init); len(es) > 0 {
		errs = append(errs, es...)
	}
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
	for i := range def.Sig.Parms {
		parm := &def.Sig.Parms[i]
		errs = append(errs, checkTypeName(x, parm.TypeName)...)
		x = x.new()
		x.variable = parm
	}

	def.Stmts, errs = gatherStmts(x, nil, def.ast.Stmts)
	return errs
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
		errs = append(errs, checkTypeName(x, field.TypeName)...)
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
		if cas.TypeName != nil {
			errs = append(errs, checkTypeName(x, cas.TypeName)...)
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
			errs = append(errs, checkTypeName(x, parm.TypeName)...)
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

func checkBlock(x *scope, infer *Type, astBlock *ast.Block) (_ *Block, errs []checkError) {
	defer x.tr("checkBlock(…)")(&errs)
	blk := &Block{ast: astBlock}
	blk.Parms, errs = gatherVars(x, astBlock.Parms)

	x = x.new()
	x.block = blk
	for i := range blk.Parms {
		parm := &blk.Parms[i]
		parm.BlkParm = blk
		parm.Index = i
		x = x.new()
		x.variable = parm
	}

	var es []checkError
	blk.Stmts, es = gatherStmts(x, infer, astBlock.Stmts)
	errs = append(errs, es...)

	// TODO: set Block.typ once Expr.Type is added

	return blk, errs
}

func checkIdent(x *scope, astIdent *ast.Ident) (_ Expr, errs []checkError) {
	defer x.tr("checkIdent(%s)", astIdent.Text)(&errs)

	ident := &Ident{ast: astIdent, Text: astIdent.Text}
	switch vr := x.findIdent(astIdent.Text).(type) {
	case nil:
		err := x.err(astIdent, "%s not found", astIdent.Text)
		errs = append(errs, *err)
	case *Var:
		ident.Var = vr
	case *Fun:
		// TODO: recursively check the call.
		return &Call{
			ast:  astIdent,
			Msgs: []Msg{{ast: astIdent, Sel: astIdent.Text}},
		}, errs
	default:
		panic(fmt.Sprintf("impossible type: %T", vr))
	}
	return ident, errs
}

func checkInt(x *scope, infer *Type, AST ast.Expr, text string) (_ Expr, errs []checkError) {
	defer x.tr("checkInt(infer=%s, %s)", infer, text)(&errs)

	if isFloat(x, infer) {
		return checkFloat(x, infer, AST, text)
	}
	var i big.Int
	x.log("parsing int [%s]", text)
	if _, ok := i.SetString(text, 0); !ok {
		panic("malformed int")
	}
	typ := builtInType(x, "Int")
	if isInt(x, infer) {
		typ = infer
	}
	if err := checkIntBounds(x, AST, typ, &i); err != nil {
		errs = append(errs, *err)
	}
	return &Int{ast: AST, Val: &i, typ: typ}, errs
}

func checkIntBounds(x *scope, n interface{}, t *Type, i *big.Int) *checkError {
	signed, bits := disectInt(x, t)
	x.log("signed=%v, bits=%v", signed, bits)
	if !signed && i.Cmp(&big.Int{}) < 0 {
		return x.err(n, "type %s cannot represent %s: negative unsigned", t, i)
	}
	min := big.NewInt(-(1 << uint(bits)))
	x.log("val=%v, val.BitLen()=%d, min=%v", i, i.BitLen(), min)
	if i.BitLen() > bits && (!signed || i.Cmp(min) != 0) {
		return x.err(n, "type %s cannot represent %s: overflow", t, i)
	}
	return nil
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

func checkFloat(x *scope, infer *Type, AST ast.Expr, text string) (_ Expr, errs []checkError) {
	defer x.tr("checkFloat(infer=%s, %s)", infer, text)(&errs)

	var f big.Float
	if _, _, err := f.Parse(text, 10); err != nil {
		panic("malformed float")
	}
	if isInt(x, infer) {
		var i big.Int
		if _, acc := f.Int(&i); acc != big.Exact {
			err := x.err(AST, "type %s cannot represent %s: truncation", infer.Sig.ID(), text)
			errs = append(errs, *err)
		}
		expr, es := checkInt(x, infer, AST, i.String())
		return expr, append(errs, es...)
	}
	typ := builtInType(x, "Float")
	if isFloat(x, infer) {
		typ = infer
	}
	return &Float{ast: AST, Val: &f, typ: typ}, errs
}

func isInt(x *scope, typ *Type) bool {
	switch {
	case typ == nil:
		return false
	default:
		return false
	case typ == builtInType(x, "Int8") ||
		typ == builtInType(x, "Int16") ||
		typ == builtInType(x, "Int32") ||
		typ == builtInType(x, "Int64") ||
		typ == builtInType(x, "UInt8") ||
		typ == builtInType(x, "UInt16") ||
		typ == builtInType(x, "UInt32") ||
		typ == builtInType(x, "UInt64"):
		return true
	}
}

func isFloat(x *scope, typ *Type) bool {
	switch {
	case typ == nil:
		return false
	default:
		return false
	case typ == builtInType(x, "Float32") ||
		typ == builtInType(x, "Float64"):
		return true
	}
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

func checkRune(x *scope, astRune *ast.Rune) (*Int, []checkError) {
	defer x.tr("checkRune(%s)", astRune.Text)()
	return &Int{
		ast: astRune,
		Val: big.NewInt(int64(astRune.Rune)),
		typ: builtInType(x, "Int32"),
	}, nil
}

func checkString(x *scope, astString *ast.String) (*String, []checkError) {
	defer x.tr("checkString(%s)", astString.Text)()
	return &String{
		ast:  astString,
		Data: astString.Data,
		typ:  builtInType(x, "String"),
	}, nil
}

func identString(id *ast.Ident) string {
	if id == nil {
		return ""
	}
	return id.Text
}
