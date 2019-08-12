{
package ast

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/eaburns/peggy/peg"
)

func (n *ModPath) prepend(m ModPath) {
	if m.start != m.end {
		n.start = m.start
	}
	n.Root = m.Root
	n.Path = append(m.Path[:len(m.Path):len(m.Path)], n.Path...)
}

func (n Sub) setStart(s int) Def { n.start = s; return &n }
func (n Import) setStart(s int) Def { n.start = s; return &n }
func (n Fun) setStart(s int) Def { n.start = s; return &n }
func (n Val) setStart(s int) Def { n.start = s; return &n }
func (n Type) setStart(s int) Def { n.start = s; return &n }

type setSiger interface {
	setSig(TypeSig) Def
}

func (n Fun) setSig(s TypeSig) Def { n.Recv = &s; return &n }
func (n Type) setSig(s TypeSig) Def { n.Sig = s; return &n }

func distSig(s TypeSig, in []Def) []Def {
	var out []Def
	for i := range in {
		out = append(out, in[i].(setSiger).setSig(s))
	}
	if len(out) == 1 {
		out[0] = out[0].setStart(s.start)
	}
	return out
}

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
	mod *ModPath
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

File <- defss:Defs* _ EOF {
	var out []Def
	for _, defs := range defss {
		for _, def := range defs {
			out = append(out, def)
		}
	}
	return []Def(out)
}

Defs <-
	i:Import { return []Def{i} } /
	f:Fun { return []Def{f} } /
	v:Val { return []Def{v} } /
	a:Alias { return []Def{a} } /
	s0:TypeSig t:Type { t0 := *t.(*Type); t0.Sig = s0; return []Def{&t0} } /
	s1:TypeSig m:Fun { m0 := *m.(*Fun); m0.Recv = &s1; return []Def{&m0} } /
	sig:TypeSig _"(" ds3:( Type / Fun )+ _")"  { return []Def(distSig(sig, ds3)) } /
	n0:ModName? _ "(" ds1:Defs+ _ ")" {
		sub := &Sub{location: loc(parser, start, end)}
		if n0 != nil {
			sub.start = n0.start
			sub.Name = n0.Text
		}
		for _, defs := range ds1 {
			for _, def := range defs {
				sub.Defs = append(sub.Defs, def)
			}
		}
		return []Def{sub}
	} /
	n1:ModName ds2:Defs {
		return []Def{&Sub{
			location: loc(parser, n1.start, end),
			Name: n1.Text,
			Defs: ds2,
		}}
	}

Import <- "import" p:String  {
	return &Import{location: loc(parser, start, end), Path: p.Data}
}

Fun <- tps:TypeParms? _ f:( "[" sig:FunSig _ "|" ss:Stmts _"]" {
	copy := *sig
	copy.location = loc(parser, start, end)
	copy.Stmts = ss
	return (*Fun)(&copy)
} ) {
	if tps != nil {
		copy := *f
		copy.TypeParms = *tps
		return Def(&copy)
	}
	return Def(f)
}

FunSig <- ps:(
	id0:Ident {
		return []parm{{key: id0}}
	} /
	o:Op id1:Ident t0:TypeName {
		return []parm{{key: o, name: id1, typ: t0}}
	} /
	( c:IdentC id2:Ident t1:TypeName {
		return parm{key:c, name: id2, typ: t1}
	} )+
) r:Ret? {
	if len(ps) == 1 && ps[0].name.Text == "" {
		p := ps[0]
		return &Fun{
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
	return &Fun{Sel: sel, Parms: parms, Ret: r}
}

Ret <- _ "^" t:TypeName {
	return TypeName(t)
}

Val <- n:Ident tn:TypeName? _":=" _"[" ss:Stmts _"]" {
	return &Val{
		location: location{n.start, loc1(parser, end)},
		Ident: n.Text,
		Type: tn,
		Init: ss,
	}
}

TypeSig <- ps:TypeParms? n:( Ident / Op ) {
	if ps == nil {
		return TypeSig{location: n.location, Name: n.Text}
	}
	return TypeSig{
		location: location{(*ps)[0].start, n.end},
		Name: n.Text,
		Parms: *ps,
	}
}

TypeParms <-
	n:TypeVar {
		return []Var{{location: n.location, Name: n.Text}}
	} /
	_"(" p0:TypeParm ps:( _"," p1:TypeParm { return Var(p1) } )* ( _ "," )? _")" {
		return []Var(append([]Var{p0}, ps...))
	}

TypeParm <- n:TypeVar t1:TypeName? {
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

TName <- mp:ModPath? n:( TypeOp / Ident ) { return tname{mod: mp, name: n} }

ModPath <- ns:ModName+ {
	s, _ := ns[0].loc()
	_, e := ns[len(ns)-1].loc()
	mp := ModPath{
		location: location{start: s, end: e},
		Root: parser.data.(*Parser).mod,
	}
	for _, n := range ns {
		mp.Path = append(mp.Path, n.Text)
	}
	return ModPath(mp)
}

Alias <- sig:TypeSig _ ":=" n:TypeName _ "." {
	return Def(&Type{
		location: location{sig.start, n.end},
		Sig: sig,
		Alias: &n,
	})
}

Type <- And / Or / Virt

And <- _ s:(
	"{" fs:(
		n:IdentC t:TypeName {
			return Var{
				location: n.location,
				Name: strings.TrimSuffix(n.Text, ":"),
				Type: &t,
			}
		}
	)* _ "}" {
	return &Type{
		location: loc(parser, start, end),
		Fields: fs,
	}
} ) { return Def(s) }

Or <- _ e: (
	"{" c:Case cs:( _ "," c1:Case { return Var(c1) } )* ( _ "," )? _ "}" {
	return &Type{
		location: loc(parser, start, end),
		Cases: append([]Var{c}, cs...),
	}
} ) { return Def(e) }

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
			Name: strings.TrimSuffix(id1.Text, ":"),
			Type: &t,
		}
	}

Virt <- _ v:( "{"  vs:MethSig+ _ "}" {
	return &Type{
		location: loc(parser, start, end),
		Virts: vs,
	}
} ) { return Def(v) }

MethSig <- _ sig:( "[" ps:(
	id0:Ident { return []parm{{name: id0}} } /
	op:Op t0:TypeName { return []parm{{name: op, typ: t0}} } /
	( id1:IdentC t1:TypeName { return parm{name: id1, typ: t1} } )+
) r:Ret? _"]" {
	var s string
	var ts []TypeName
	for _, p := range ps {
		s += p.name.Text
		if p.typ.Name != "" { // p.typ.Name=="" means unary method
			ts = append(ts, p.typ)
		}
	}
	return MethSig{
		location: loc(parser, start, end),
		Sel: s,
		Parms: ts,
		Ret: r,
	}
} ) { return MethSig(sig) }

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

Assign <- l:Lhs _ ":=" r:Expr { return Stmt(&Assign{Vars: l, Val: r}) }

Lhs <- id:(
		i0:Ident t0:TypeName? {
			e := i0.end
			if t0 != nil {
				e = t0.end
			}
			return &Var{
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
			return &Var{
				location: location{i1.start, e},
				Name: i1.Text,
				Type: t1,
			}
		}
	)* {
	return []*Var(append([]*Var{id}, is...))
}

Expr <- Call / Primary

Call <- c:( Nary / Binary / Unary ) cs:(
	_"," m:( UnaryMsg / BinMsg / NaryMsg ) { return Msg(m) }
)* {
	c.Msgs = append(c.Msgs, cs...)
	return Expr(c)
}

Unary <- r:(
	p:Primary { return Node(p) } /
	n:ModPath { return Node(n) }
) ms:UnaryMsg+ {
	s, _ := r.loc()
	c := Call{
		location: location{s, ms[0].end},
		Recv: r,
		Msgs: []Msg{ms[0]},
	}
	for _, m := range ms[1:] {
		c = Call{
			location: location{s, m.end},
			Recv: c,
			Msgs: []Msg{m},
		}
	}
	return Call(c)
}

UnaryMsg <- i:Ident {
	return Msg{location: i.location, Sel: i.Text}
}

Binary <- r:(
	u:Unary { return Node(u) } /
	p:Primary { return Node(p) } /
	n:ModPath { return Node(n) }
) m:BinMsg {
	s, _ := r.loc()
	return Call{
		location: location{s, loc1(parser, end)},
		Recv: r,
		Msgs: []Msg{m},
	}
}

BinMsg <-
	n:Op a:(
		b:Binary { return Expr(b) } /
		u:Unary { return Expr(u) } /
		Primary
	) {
	return Msg{
		location: location{n.start, loc1(parser, end)},
		Sel: n.Text,
		Args: []Expr{a},
	}
}

Nary <- r:(
	b:Binary { return Node(b) } /
	u:Unary { return Node(u) } /
	p:Primary { return Node(p) } /
	n:ModPath { return Node(n) }
)? m:NaryMsg {
	s := m.start
	var recv Node
	if r != nil {
		s, _ = (*r).loc()
		recv = *r
	}
	return Call{
		location: location{s, loc1(parser, end)},
		Recv: recv,
		Msgs: []Msg{m},
	}
}

NaryMsg <- as:(
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
		Sel: sel,
		Args: es,
	}
}

Primary <-
	i:Ident { return Expr(i) }/
	Float /
	Int /
	Rune /
	s:String { return Expr(s) } /
	Ctor /
	Block /
	_ "(" e:Expr _ ")" { return Expr(e) }

Ctor <- _ ( "{" t:TypeName _"|" as:(
	Struct /
	id:Ident { return []arg{{name: id, val: id}} } /
	a:Ary? { if a != nil { return *a }; return []arg{} }
) _"}" ) {
	ctor := Ctor{location: loc(parser, start, end), Type: t}
	for _, a := range as {
		ctor.Sel += a.name.Text
		if a.val != nil {
			ctor.Args = append(ctor.Args, a.val)
		}
	}
	return Expr(ctor)
}

Struct <- i0:IdentC e0:Expr args:(
	_ ";" i1:IdentC e1:Expr { return arg{name: i1, val: e1} }
)* ( _";" )? {
	return []arg(append([]arg{{name: i0, val:e0}}, args...))
}

Ary <- e0:Expr  es:( _ ";" e:Expr { return arg{val: e} } )* ( _";" )? {
	return []arg(append([]arg{{val: e0}}, es...))
}

Block <- _ b:( "[" (
	( ps:( n:CIdent t:TypeName? {
		return Var{location: loc(parser, start, end), Name: n.Text, Type: t}
	} )+ _ "|" )? ss:Stmts
) _ "]"  {
	return Block{
		location: loc(parser, start, end),
		Parms: ps,
		Stmts: ss,
	}
} ) { return Expr(b) }

# Tokens

Int "integer" <- _ tok:( text:( [+\-]? [0-9]+ ) {
	return Int{location: loc(parser, start, end), Text: text}
} ) { return Expr(tok) }

Float "floating point" <- _ tok:( text:( [+\-]? [0-9]+ "." [0-9]+ ( [eE] [+\-]? [0-9]+ )? ) {
	return Float{location: loc(parser, start, end), Text: text}
} ) { return Expr(tok) }

Rune "rune" <- _ tok:( text:( ['] !"\n" data:( Esc / "\\'" / [^'] ) ['] ) {
	r, w := utf8.DecodeRuneInString(data)
	if w != len(data) {
		panic("impossible")
	}
	return Rune{location: loc(parser, start, end), Text: text, Rune: r}
} ) { return Expr(tok) }

String "string" <-
	_ tok0:( text0:( ["] data0:( !"\n" ( Esc / "\\\"" / [^"] ) )* ["] ) {
		return String{location: loc(parser, start, end), Text: text0, Data: data0}
	} ) { return String(tok0) } /
	_ tok1:( text1: ( [`] data1:( "\\`" / [^`] )* [`] ) {
		return String{location: loc(parser, start, end), Text: text1, Data: data1}
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
	return Ident{location: loc(parser, start, end), Text: text}
} ) { return Ident(tok) }

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