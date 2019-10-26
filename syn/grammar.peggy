{
package syn

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/eaburns/peggy/peg"
)

type parm struct {
	name Ident
	typ TypeName
	key Ident
}

type arg struct {
	name Ident
	val Expr
}

type tname struct {
	mod *ModTag
	name Ident
}

func hex(s string) rune {
	x, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		panic("impossible")
	}
	return rune(x)
}

func loc(p *_Parser, start, end int) location {
	offs := p.data.(*Parser).offs
	return location{start: start + offs, end: end + offs}
}

func loc1(p *_Parser, pos int) int { return pos + p.data.(*Parser).offs }
}

File <- imports:Import* defs:Def* _ EOF {
	return File{
		Imports: imports,
		Defs: defs,
	}
}

Import <- _ i:( kw:( "import" / "Import" ) path:String {
	return Import{
		location: loc(parser, start, end),
		All: kw == "Import",
		Path: path.Text,
	}
}) { return Import(i) }

Def <- Val / Fun / Meth / Type

Val <- _ v:(key:("val" / "Val") id:Ident typ:TypeName? _":=" _"[" stmts:Stmts _"]" {
	varEnd := id.end
	if typ != nil {
		varEnd = typ.end
	}
	return &Val{
		location: loc(parser, start, end),
		priv: key == "val",
		Var: Var{
			location: location{start: id.start, end: varEnd},
			Name: id.Text,
			Type: typ,
		},
		Init: stmts,
	}
}) { return Def(v) }

Fun <- _ f:(
	key:("func" / "Func") tps:TParms _ "[" sig:FunSig body:(_ "|" stmts:Stmts { return []Stmt(stmts) })? _"]" {
	if body != nil && stmts == nil {
		stmts = []Stmt{}
	}
	return &Fun{
		location: loc(parser, start, end),
		priv: key == "func",
		TParms: tps,
		Sig: sig,
		Stmts: stmts,
	}
})   { return Def(f) }

Meth <- _ m:(key:("meth" / "Meth") recv:Recv tps:TParms _ "[" sig:FunSig body: (_ "|" stmts:Stmts { return []Stmt(stmts)})? _"]" {
	if body != nil && stmts == nil {
		stmts = []Stmt{}
	}
	return &Fun{
		location: loc(parser, start, end),
		priv: key == "meth",
		Recv: &recv,
		TParms: tps,
		Sig: sig,
		Stmts: stmts,
	}
}) { return Def(m) }

Recv <- tps:TParms mod:ModName? n:( Ident / Op ) {
	l := loc(parser, start, end)
	if len(tps) > 0 {
		l.start = tps[0].start
	}
	return Recv{
		TypeSig: TypeSig{
			location: l,
			Name: n.Text,
			Parms: tps,
		},
		Mod: mod,
	}
}

FunSig <- ps:Parms r:Ret? {
	if len(ps) == 1 && ps[0].name.Text == "" {
		p := ps[0]
		return FunSig{
			location: location{p.key.start, p.typ.end},
			Sel: p.key.Text,
			Ret: r,
		}
	}
	var sel string
	var parms []Var
	for i := range ps {
		p := &ps[i]
		sel += p.key.Text
		parms = append(parms, Var{
			location: location{p.key.start, p.typ.end},
			Name: p.name.Text,
			Type: &p.typ,
		})
	}
	return FunSig{
		location: loc(parser, start, end),
		Sel: sel,
		Parms: parms,
		Ret: r,
	}
}

Parms <-
	id0:Ident {
		return []parm{{key: id0}}
	} /
	o:Op id1:Ident t0:TypeName {
		return []parm{{key: o, name: id1, typ: t0}}
	} /
	( c:IdentC id2:Ident t1:TypeName {
		return parm{key:c, name: id2, typ: t1}
	} )+

Ret <- _ "^" t:TypeName {
	return TypeName(t)
}

TypeSig <- tps:TParms n:( Ident / Op ) {
	l := loc(parser, start, end)
	if len(tps) > 0 {
		l.start = tps[0].start
	}
	return TypeSig{
		location: l,
		Name: n.Text,
		Parms: tps,
	}
}

TParms <- tps:(
	n:TypeVar {
		return []Var{{location: n.location, Name: n.Text}}
	} /
	_ v:( "_" {
		return []Var{{location: n.location, Name: "_"}}
	} ) { return []Var(v) } /
	_"(" p0:TParm ps:( _"," p1:TParm { return Var(p1) } )* ( _ "," )? _")" {
		return []Var(append([]Var{p0}, ps...))
	}
)? {
	if tps == nil {
		return ([]Var)(nil)
	}
	return ([]Var)(*tps)
}

TParm <- n:(
	TypeVar /
	_ empty:( "_" { return Ident{location:loc(parser, start, end), Text: "_"} } ) {
		 return Ident(empty)
	}
) t1:TypeName? {
	e := n.end
	if t1 != nil {
		e = t1.end
	}
	return Var{
		location: location{n.start, e},
		Name: n.Text,
		Type: t1,
	}
}

TypeName <-
	tv1:TypeVar? ns0:TName+ {
		s := ns0[0].name.start
		var a []TypeName
		if tv1 != nil {
			s = tv1.start
			a = []TypeName{{location: tv1.location, Name: tv1.Text, Var: true}}
		}
		for _, n := range ns0[:len(ns0)-1] {
			a = []TypeName{{
				location: location{s, n.name.end},
				Mod: n.mod,
				Name: n.name.Text,
				Args: a,
			}}
		}
		n := ns0[len(ns0)-1]
		return TypeName{
			location: location{s, n.name.end},
			Mod: n.mod,
			Name: n.name.Text,
			Args: a,
		}
	} /
	tv2:TypeVar {
		return TypeName{location: tv2.location, Name: tv2.Text, Var: true}
	} /
	_ tn0:(
		"(" ns1:TypeNameList _")" ns2:TName+ {
		s := loc1(parser, start)
		for _, n := range ns2[:len(ns2)-1] {
			ns1 = []TypeName{{
				location: location{s, n.name.end},
				Mod: n.mod,
				Name: n.name.Text,
				Args: ns1,
			}}
		}
		return TypeName{
			location: loc(parser, start, end),
			Mod: ns2[len(ns2)-1].mod,
			Name: ns2[len(ns2)-1].name.Text,
			Args: ns1,
		}
	} ) { return TypeName(tn0)} /
	_ "(" n2:TypeName _ ")" { return TypeName(n2) }

TypeNameList <- n0:TypeName ns:( _"," n1:TypeName { return TypeName(n1) } )* ( _ "," )? {
	return []TypeName(append([]TypeName{n0}, ns...))
}

TName <- mod:ModName? n:( TypeOp / Ident ) { return tname{mod: mod, name: n} }

Type <- _ def:(key:("type"/"Type") sig:TypeSig _ typ:( Alias / And / Or / Virt ) {
	typ.location = loc(parser, start, end)
	typ.priv = key == "type"
	typ.Sig = sig
	return Def(&typ)
}) { return Def(def) }

Alias <- _ ":=" n:TypeName _ "." { return Type{Alias: &n} }

And <- _ "{" fs:Field* _ "}" { return Type{Fields: fs} }

Field <- n:IdentC t:TypeName {
	return Var{
		location: n.location,
		Name: strings.TrimSuffix(n.Text, ":"),
		Type: &t,
	}
}

Or <- _ "{" ( _ "|" )? c:Case cs:( _ "|" c1:Case { return Var(c1) } )* _ "}" {
	return Type{Cases: append([]Var{c}, cs...)}
}

Case <-
	id0:Ident {
		return Var{
			location: id0.location,
			Name: id0.Text,
		}
	} /
	id1:IdentC t:TypeName {
		return Var{
			location: id1.location,
			Name: id1.Text,
			Type: &t,
		}
	}

Virt <- _ "{"  vs:MethSig+ _ "}" { return Type{Virts: vs} }

MethSig <- _ sig:( "[" ps:(
	id0:Ident { return []parm{{name: id0}} } /
	op:Op t0:TypeName { return []parm{{name: op, typ: t0}} } /
	( id1:IdentC t1:TypeName { return parm{name: id1, typ: t1} } )+
) r:Ret? _"]" {
	var s string
	var parms []Var
	for _, p := range ps {
		s += p.name.Text
		if p.typ.Name != "" { // unary
			tn := p.typ
			parms = append(parms, Var{Type: &tn})
		}
	}
	return FunSig{
		location: loc(parser, start, end),
		Sel: s,
		Parms: parms,
		Ret: r,
	}
} ) { return FunSig(sig) }

Stmts <- ss:(
	s0:Stmt s1s:( _"." s1:Stmt { return Stmt(s1) })* ( _ "." )? {
		return []Stmt(append([]Stmt{s0}, s1s...))
	}
)? {
	if ss != nil {
		return *ss
	}
	return []Stmt{}
}

Stmt <-
	Return /
	Assign /
	e:Expr { return Stmt(e) }

Return <- _ r:(
	"^" e:Expr { return &Ret{start: loc1(parser, start), Val: e} }
) {
	return Stmt(r)
}

Assign <- l:Lhs _ ":=" r:Expr { return Stmt(&Assign{Vars: l, Expr: r}) }

Lhs <- id:(
		i0:Ident t0:TypeName? {
			e := i0.end
			if t0 != nil {
				e = t0.end
			}
			return Var{
				location: location{i0.start, e},
				Name: i0.Text,
				Type: t0,
			}
		}
	) is:(
		_ "," i1:Ident t1:TypeName? {
			e := i1.end
			if t1 != nil {
				e = t1.end
			}
			return Var{
				location: location{i1.start, e},
				Name: i1.Text,
				Type: t1,
			}
		}
	)* {
	return []Var(append([]Var{id}, is...))
}

Expr <- Call / Primary

Call <- c:( Nary / Binary / Unary ) cs:(
	_"," m:( UnaryMsg / BinMsg / NaryMsg ) { return Msg(m) }
)* {
	c.Msgs = append(c.Msgs, cs...)
	return Expr(c)
}

Unary <- r:Primary? ms:UnaryMsg+ {
	s := ms[0].start
	var recv Expr
	if r != nil {
		s, _ = (*r).loc()
		recv = *r
	}
	c := &Call{
		location: location{s, ms[0].end},
		Recv: recv,
		Msgs: []Msg{ms[0]},
	}
	for _, m := range ms[1:] {
		c = &Call{
			location: location{s, m.end},
			Recv: c,
			Msgs: []Msg{m},
		}
	}
	// TODO: fix the (*Call)(c) workaround for a Peggy bug.
	// Ideally we would just
	// 	return (*Call)(c)
	// However, there is a bugy in Peggy where it detects the type as
	// (*Call) instead of just *Call, and it gives a type mismatch error.
	if true { return (*Call)(c) }
	return &Call{}
}

UnaryMsg <- mod:ModName? i:Ident {
	return Msg{location: i.location, Mod: mod, Sel: i.Text}
}

Binary <- r:( u:Unary { return Expr(u) } / Primary ) m:BinMsg {
	s, _ := r.loc()
	return &Call{
		location: location{s, loc1(parser, end)},
		Recv: r,
		Msgs: []Msg{m},
	}
}

BinMsg <-
	mod:ModName? n:Op a:(
		b:Binary { return Expr(b) } /
		u:Unary { return Expr(u) } /
		Primary
	) {
	return Msg{
		location: location{n.start, loc1(parser, end)},
		Mod: mod,
		Sel: n.Text,
		Args: []Expr{a},
	}
}

Nary <- r:(
	b:Binary { return Expr(b) } /
	u:Unary {return Expr(u) } /
	Primary
)? m:NaryMsg {
	s := m.start
	var recv Expr
	if r != nil {
		s, _ = (*r).loc()
		recv = *r
	}
	return &Call{
		location: location{s, loc1(parser, end)},
		Recv: recv,
		Msgs: []Msg{m},
	}
}

NaryMsg <- mod:ModName? as:(
	n:IdentC v:(
		b:Binary { return Expr(b) } /
		u:Unary { return Expr(u) } /
		Primary
	) { return arg{n, v} }
)+ {
	var sel string
	var es []Expr
	for _, a := range as {
		sel += a.name.Text
		es = append(es, a.val)
	}
	return Msg{
		location: location{as[0].name.start, loc1(parser, end)},
		Mod: mod,
		Sel: sel,
		Args: es,
	}
}

Primary <-
	mod:ModName? i:Ident { i.Mod = mod; return Expr(&i) }/
	Float /
	Int /
	Rune /
	s:String { return Expr(&s) } /
	Ctor /
	Block /
	_ "(" e:Expr _ ")" { return Expr(e) }

Ctor <- _ ctor:( "{" es:Exprs?  _"}"  {
	if es == nil {
		return &Ctor{location: loc(parser, start, end)}
	}
	return &Ctor{location: loc(parser, start, end), Args: *es}
} ) {
	return Expr(ctor)
}

Exprs <- e0:Expr  es:( _ ";" e:Expr { return Expr(e) } )* ( _";" )? {
	return ([]Expr)(append([]Expr{e0}, es...))
}

Block <- _ b:( "[" (
	( ps:( n:CIdent t:TypeName? {
		return Var{location: loc(parser, start, end), Name: n.Text, Type: t}
	} )+ _ "|" )? ss:Stmts
) _ "]"  {
	return &Block{
		location: loc(parser, start, end),
		Parms: ps,
		Stmts: ss,
	}
} ) { return Expr(b) }

# Tokens

Int "integer" <- _ tok:( text:( [+\-]? [0-9]+ ) {
	return &Int{location: loc(parser, start, end), Text: text}
} ) { return Expr(tok) }

Float "floating point" <- _ tok:( text:( [+\-]? [0-9]+ "." [0-9]+ ( [eE] [+\-]? [0-9]+ )? ) {
	return &Float{location: loc(parser, start, end), Text: text}
} ) { return Expr(tok) }

Rune "rune" <- _ tok:( text:( ['] !"\n" data:( Esc / "\\'" / [^'] ) ['] ) {
	r, w := utf8.DecodeRuneInString(data)
	if w != len(data) {
		panic("impossible")
	}
	return &Rune{location: loc(parser, start, end), Text: text, Rune: r}
} ) { return Expr(tok) }

String "string" <-
	_ tok0:( text0:( ["] data0:( !"\n" ( Esc / "\\\"" / [^"] ) )* ["] ) {
		return String{
			location: loc(parser, start, end),
			Text: text0,
			Data: data0,
		}
	} ) { return String(tok0) } /
	_ tok1:( text1: ( [`] data1:( "\\`" / [^`] )* [`] ) {
		return String{
			location: loc(parser, start, end),
			Text: text1,
			Data: data1,
		}
	} ) { return String(tok1) }

Esc <-
	"\\n" { return "\n" } /
	"\\t" { return "\t" }/
	"\\b" { return "\b" }/
	"\\\\" { return "\\" } /
	"\\" x0:( X X ) { return string(hex(x0)) } /
	"\\x" x1:( X X X X ) { return string(hex(x1)) } /
	"\\X" x2:( X X X X X X X X ) { return string(hex(x2)) }

X <- [a-fA-F0-9]

Op "operator" <- _ !"//" !"/*" tok:( text:( [!%&*+\-/<=>?@\\|~]+ ) {
	return Ident{location: loc(parser, start, end), Text: text}
} ) { return Ident(tok) }

TypeOp "type operator" <- _ tok:( text:( [!&?]+ ) {
	return Ident{location: loc(parser, start, end), Text: text}
} ) { return Ident(tok) }

ModName "module name" <- _ tok:( text:( "#" [_a-zA-Z][_a-zA-Z0-9]* ) {
	return ModTag{location: loc(parser, start, end), Text: text}
} ) { return ModTag(tok) }

IdentC "identifier:" <- _ !TypeVar !"import" tok:( text:( [_a-zA-Z][_a-zA-Z0-9]* ":" ) {
	return Ident{location: loc(parser, start, end), Text: text}
} ) { return Ident(tok) }

CIdent ":identifier"<- _ ":" !TypeVar !"import" tok:( text:( [_a-zA-Z][_a-zA-Z0-9]* ) {
	return Ident{location: loc(parser, start, end), Text: text}
} ) { return Ident(tok) }

Ident "identifier" <- _ !TypeVar !"import" tok:( text:( [_a-zA-Z][_a-zA-Z0-9]* ) !":" {
	return Ident{location: loc(parser, start, end), Text: text}
} ) { return Ident(tok) }

TypeVar "type variable" <- _ tok:( text:( [A-Z] ![_a-zA-Z0-9] ) {
	return Ident{location: loc(parser, start, end), Text: text}
} ) { return Ident(tok) }

_ "" <- ( Space / Cmnt )*

Cmnt <- "//" ( !"\n" . )* / "/*" ( !"*/" . )* "*/"

Space <- " " / "\t" / "\n"

EOF <- !.