{
package pea

import (
	"strconv"
	"unicode/utf8"

	"github.com/eaburns/peggy/peg"
)

func flatten(dss [][]Def) []Def {
	var defs []Def
	for _, ds := range dss {
		defs = append(defs, ds...)
	}
	return defs
}

type setSiger interface {
	setSig(TypeSig) Def
}

func (n Fun) setSig(s TypeSig) Def { n.Recv = &s; return &n }
func (n Struct) setSig(s TypeSig) Def { n.Sig = s; return &n }
func (n Enum) setSig(s TypeSig) Def { n.Sig = s; return &n }
func (n Virt) setSig(s TypeSig) Def { n.Sig = s; return &n }

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

func distMod(p *_Parser, m *ModPath, in []Def) []Def {
	var out []Def
	var mp []Ident
	if m == nil {
		mp = []Ident{{Text: p.data.(*Parser).mod}}
	} else {
		mp = *m
	}
	for i := range in {
		out = append(out, in[i].setMod(mp))
	}
	if m != nil && len(out) == 1 {
		out[0] = out[0].setStart((*m)[1].start)
	}
	return []Def(out)
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

File <- dss:Def* _ EOF { return &File{ Defs: flatten(dss) } }

Def <- m:ModPath? ds0:(
	_ ( "import" p:String ) {
		return []Def{
			&Import{location: loc(parser, start, end), Path: p.Data},
		}
	} /
	_ ( "(" dss:Def+ _ ")" ) {
		return []Def{
			&SubMod{location: loc(parser, start, end), Defs: flatten(dss)},
		}
	} /
	f:Fun {
		return []Def{f}
	} /
	v:Var {
		return []Def{v}
	} /
	sig:TypeSig ds1:(
		t0:Type { return []Def{t0} } /
		m0:Fun { return []Def{m0} } /
		_"(" ds2:( Type / Fun )+ _")" { return []Def(ds2) }
	) {
		return []Def(distSig(sig, ds1))
	}
) {
	return []Def(distMod(parser, m, ds0))
}

ModPath <- n:ModName+ {
	n = append([]Ident{{
		Text: parser.data.(*Parser).mod,
	}}, n...)
	return ModPath(n)
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
	var parms []Parm
	for i := range ps {
		p := &ps[i]
		sel += p.key.Text
		parms = append(parms, Parm{
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

Var <- n:Ident _":=" _"[" ss:Stmts _"]" {
	return &Var{
		location: location{n.start, loc1(parser, end)},
		Name: n.Text,
		Val: ss,
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
		return []Parm{{location: n.location, Name: n.Text}}
	} /
	_"(" p0:TypeParm ps:( _"," p1:TypeParm { return Parm(p1) } )* ( _ "," )? _")" {
		return []Parm(append([]Parm{p0}, ps...))
	}

TypeParm <- n:TypeVar t1:TypeName? {
	e := n.end
	if t1 != nil {
		e = t1.end
	}
	return Parm{
		location: location{n.start, e},
		Name: n.Text,
		Type: t1,
	}
}

TypeName <-
	tv0:TypeVar? ids0:Ident* op:TypeOp {
		s := op.start
		var a []TypeName
		if tv0 != nil {
			s = tv0.start
			a = []TypeName{{location: tv0.location, Name: tv0.Text}}
		}
		if len(ids0) > 0 {
			if tv0 == nil {
				s = ids0[0].start
			}
			for _, id := range ids0[:len(ids0)-1] {
				a = []TypeName{{
					location: location{s, id.end},
					Name: id.Text,
					Args: a,
				}}
			}
		}
		return TypeName{
			location: location{s, op.end},
			Name: op.Text,
			Args: a,
		}
	} /
	tv1:TypeVar? ids1:Ident+ {
		s := ids1[0].start
		var a []TypeName
		if tv1 != nil {
			s = tv1.start
			a = []TypeName{{location: tv1.location, Name: tv1.Text}}
		}
		for _, id := range ids1[:len(ids1)-1] {
			a = []TypeName{{
				location: location{s, id.end},
				Name: id.Text,
				Args: a,
			}}
		}
		n := ids1[len(ids1)-1]
		return TypeName{
			location: location{s, n.end},
			Name: n.Text,
			Args: a,
		}
	} /
	tv2:TypeVar {
		return TypeName{location: tv2.location, Name: tv2.Text}
	} /
	_ blk:(
		"[" psp:TypeNameList? r:( _"|" r1:TypeName { return TypeName(r1) } )? _"]" {
		name := "[]"
		var ps []TypeName
		if psp != nil {
			ps = *psp
		}
		if r != nil {
			name = "[|]"
			ps = append(ps, *r)
		}
		return TypeName{
			location: loc(parser, start, end),
			Name: name,
			Args: ps,
		}
	} ) { return TypeName(blk) } /
	_ tn0:(
		"(" ns:TypeNameList _")" ids2:Ident+ {
		s := loc1(parser, start)
		for _, id := range ids2[:len(ids2)-1] {
			ns = []TypeName{{
				location: location{s, id.end},
				Name: id.Text,
				Args: ns,
			}}
		}
		return TypeName{
			location: loc(parser, start, end),
			Name: ids2[len(ids2)-1].Text,
			Args: ns,
		}
	} ) { return TypeName(tn0)} /
	_ "(" n2:TypeName _ ")" { return TypeName(n2) }

TypeNameList <- n0:TypeName ns:( _"," n1:TypeName { return TypeName(n1) } )* ( _ "," )? {
	return []TypeName(append([]TypeName{n0}, ns...))
}

Type <- Struct / Enum / Virt

Struct <- _ s:(
	"{" fs:(
		n:IdentC t:TypeName {
			return Parm{location: n.location, Name: n.Text, Type: &t}
		}
	)* _ "}" {
	return &Struct{
		location: loc(parser, start, end),
		Fields: fs,
	}
} ) { return Def(s) }

Enum <- _ e: (
	"{" c:Case cs:( _ "," c1:Case { return Parm(c1) } )* ( _ "," )? _ "}" {
	return &Enum{
		location: loc(parser, start, end),
		Cases: append([]Parm{c}, cs...),
	}
} ) { return Def(e) }

Case <-
	id0:Ident {
		return Parm{
			location: id0.location,
			Name: id0.Text,
		}
	} /
	id1:IdentC t:TypeName {
		return Parm{
			location: id1.location,
			Name: id1.Text,
			Type: &t,
		}
	}

Virt <- _ v:( "{"  ss:MethSig+ _ "}" {
	return &Virt{
		location: loc(parser, start, end),
		Meths: ss,
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
		ts = append(ts, p.typ)
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
	"^" e:Expr { return Ret{start: loc1(parser, start), Val: e} }
) {
	return Stmt(r)
}

Assign <- l:Lhs _ ":=" r:Expr { return Stmt(Assign{Var: l, Val: r}) }

Lhs <- id:(
		i0:Ident t0:TypeName? {
			e := i0.end
			if t0 != nil {
				e = t0.end
			}
			return Parm{
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
			return Parm{
				location: location{i1.start, e},
				Name: i1.Text,
				Type: t1,
			}
		}
	)* {
	return []Parm(append([]Parm{id}, is...))
}

Expr <- Call / Primary

Call <- c:( Nary / Binary / Unary ) cs:(
	_";" m:( UnaryMsg / BinMsg / NaryMsg ) { return Msg(m) }
)* {
	c.Msgs = append(c.Msgs, cs...)
	return Expr(c)
}

Unary <- r:( Primary / n:ModPath { return Expr(n) } ) ms:UnaryMsg+ {
	c := Call{
		location: location{r.Start(), ms[0].end},
		Recv: r,
		Msgs: []Msg{ms[0]},
	}
	for _, m := range ms[1:] {
		c = Call{
			location: location{r.Start(), m.end},
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
	u:Unary { return Expr(u) } /
	Primary /
	n:ModPath { return Expr(n) }
) m:BinMsg {
	return Call{
		location: location{r.Start(), loc1(parser, end)},
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
	b:Binary { return Expr(b) } /
	u:Unary { return Expr(u) } /
	Primary /
	n:ModPath { return Expr(n) }
)? m:NaryMsg {
	s := m.start
	var recv Expr
	if r != nil {
		s = (*r).Start()
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
	Int /
	Float /
	Rune /
	s:String { return Expr(s) } /
	Ctor /
	Block /
	_ "(" e:Expr _ ")" { return Expr(e) }

Ctor <- _ ( "{" t:TypeName _"|" as:(
	( IdentC e0:Expr { return Expr(e0) } )+ /
	a:Ary? { if a != nil { return *a }; return []Expr{} }
) _"}" ) {
	return Expr(Ctor{
		location: loc(parser, start, end),
		Type: t,
		Args: as,
	})
}

Ary <- e0:Expr  es:( _ ";" e:Expr { return Expr(e) } )* ( _";" )? {
	return []Expr(append([]Expr{e0}, es...))
}

Block <- _ ( "[" (
	( ps:( n:CIdent t:TypeName? { return Parm{Name: n.Text, Type: t} } )+ _ "|" )? ss:Stmts
) _ "]" ) {
	return Expr(Block{
		location: loc(parser, start, end),
		Parms: ps,
		Stmts: ss,
	})
}

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