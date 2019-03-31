{
package main

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

type setModer interface {
	setMod(ModName) Def
}

func (n Mod) setMod(m ModName) Def { n.Mod = m; return &n }
func (n Import) setMod(m ModName) Def { return &n }
func (n Fun) setMod(m ModName) Def { n.Mod = m; return &n }
func (n Var) setMod(m ModName) Def { n.Mod = m; return &n }
func (n Struct) setMod(m ModName) Def { n.Mod = m; return &n }
func (n Enum) setMod(m ModName) Def { n.Mod = m; return &n }
func (n Virt) setMod(m ModName) Def { n.Mod = m; return &n }

type setSiger interface {
	setSig(TypeSig) Def
}

func (n Fun) setSig(s TypeSig) Def { n.Recv = &s; return &n }
func (n Struct) setSig(s TypeSig) Def { n.Sig = s; return &n }
func (n Enum) setSig(s TypeSig) Def { n.Sig = s; return &n }
func (n Virt) setSig(s TypeSig) Def { n.Sig = s; return &n }

type setLocer interface {
	setLoc(int, int) Def
}

func (n Struct) setLoc(s, e int) Def { n.start = s; n.end = e; return &n }
func (n Enum) setLoc(s, e int) Def { n.start = s; n.end = e; return &n }
func (n Virt) setLoc(s, e int) Def { n.start = s; n.end = e; return &n }

type parm struct {
	name Ident
	typ TypeName
	key string
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

}

File <- dss:Def* _ EOF { return []Def(flatten(dss)) }

Def <- m:( m1:ModName* { return ModName(m1) } ) ds0:(
	_ ( "import" p:String ) {
		return []Def{&Import{start: start, end: end, Path: p.Data}}
	} /
	_ ( "(" dss:Def+ _ ")" ) {
		return []Def{&Mod{start: start, end: end, Defs: flatten(dss)}}
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
		var ds []Def
		for i := range ds1 {
			ds = append(ds, ds1[i].(setSiger).setSig(sig))
		}
		return []Def(ds)
	}
) {
	var ds []Def
	for i := range ds0 {
		ds = append(ds, ds0[i].(setModer).setMod(m))
	}
	return []Def(ds)
}

Fun <- _ ( "[" f:FunSig _ "|" ss:Stmts _"]" ) {
	copy := *f
	copy.start = start
	copy.end = end
	copy.Stmts = ss
	return Def(&copy)
}

FunSig <- ps:(
	id0:Ident {
		return []parm{{key: id0.Text}}
	} /
	o:Op id1:Ident t0:TypeName {
		return []parm{{key: o.Text, name: id1, typ: t0}}
	} /
	( c:IdentC id2:Ident t1:TypeName {
		return parm{key:c.Text, name: id2, typ: t1}
	} )+
) r:Ret? {
	if len(ps) == 1 && ps[0].name.Text == "" {
		return &Fun{Sel: ps[0].key, Ret: r}
	}
	var s string
	var parms []Parm
	for _, p := range ps {
		s += p.key
		parms = append(parms, Parm{Name: p.name.Text, Type: &p.typ})
	}
	return &Fun{Sel: s, Parms: parms, Ret: r}
}

Ret <- _"^" t:TypeName {
	return TypeName(t)
}

Var <- n:Ident _":=" _"[" ss:Stmts _"]" {
	return &Var{start: n.start, end: end, Name: n.Text, Val: ss}
}

TypeSig <- ps:TypeParms? n:( Ident / Op ) {
	if ps != nil {
		return TypeSig{Name: n.Text, Parms: *ps}
	}
	return TypeSig{Name: n.Text}
}

TypeParms <-
	n:TypeVar {
		return []Parm{{Name: n.Text}}
	} /
	_"(" p0:TypeParm ps:( _"," p1:TypeParm { return Parm(p1) } )* ( _ "," )? _")" {
		return []Parm(append([]Parm{p0}, ps...))
	}

TypeParm <-
	t0:TypeName { return Parm{Type: &t0} }/
	n:TypeVar t1:TypeName? { return Parm{Name: n.Text, Type: t1} }

TypeName <-
	tv0:TypeVar? ids0:Ident* op:[?&] {
		var a []TypeName
		if tv0 != nil {
			a = []TypeName{{Name: tv0.Text}}
		}
		if len(ids0) > 0 {
			for _, id := range ids0[:len(ids0)-1] {
				a = []TypeName{{Name: id.Text, Args: a}}
			}
		}
		return TypeName{Name: op, Args: a}
	} /
	tv1:TypeVar? ids1:Ident+ {
		var a []TypeName
		if tv1 != nil {
			a = []TypeName{{Name: tv1.Text}}
		}
		for _, id := range ids1[:len(ids1)-1] {
			a = []TypeName{{Name: id.Text, Args: a}}
		}
		return TypeName{Name: ids1[len(ids1)-1].Text, Args: a}
	} /
	tv2:TypeVar {
		return TypeName{Name: tv2.Text}
	} /
	_ "[" ps:TypeName* r:( _"|" r1:TypeName { return TypeName(r1) } )? _"]" {
		name := "[]"
		if r != nil {
			name = "[|]"
			ps = append(ps, *r)
		}
		return TypeName{Name: name, Args: ps}
	} /
	_ "(" n0:TypeName ns:( _"," n1:TypeName { return TypeName(n1) } )* ( _ "," )? _")" ids2:Ident+ {
		as := append([]TypeName{n0}, ns...)
		for _, id := range ids1[:len(ids1)-1] {
			as = []TypeName{{Name: id.Text, Args: as}}
		}
		return TypeName{Name: ids1[len(ids1)-1].Text, Args: as}
	} /
	_ "(" n2:TypeName _ ")" { return TypeName(n2) }

Type <- _ ( "{" d:( Struct / Enum / Virt ) _ "}" ) {
	return Def(d.(setLocer).setLoc(start, end))
}

Struct <- fs:( n:IdentC t:TypeName { return Parm{Name: n.Text, Type: &t} } )+ {
	return Def(&Struct{Fields: fs})
}

Enum <- c:Case cs:( _ "," c1:Case { return Parm(c1) } )* ( _ "," )? {
	return Def(&Enum{Cases: append([]Parm{c}, cs...)})
}

Case <-
	id0:Ident { return Parm{Name: id0.Text} } /
	id1:IdentC t:TypeName { return Parm{Name: id1.Text, Type: &t} }

Virt <- ss:MethSig+ { return Def(&Virt{Meths: ss}) }

MethSig <- _ "[" ps:(
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
	return MethSig{Sel: s, Parms: ts, Ret: r}
}

Stmts <- ss:(
	s0:Stmt s1s:( _"." s1:Stmt { return Stmt(s1) })* ( _ "." )? {
		return []Stmt(append([]Stmt{s0}, s1s...))
	}
)? { if ss != nil { return *ss }; return []Stmt{} }

Stmt <- Return / Assign / e:Expr { return Stmt(e) }

Return <- _ "^" e:Expr { return Stmt(Ret{Val: e}) }

Assign <- l:Lhs _ ":=" r:Expr { return Stmt(Assign{Var: l, Val: r}) }

Lhs <- id:(
	i0:Ident t0:TypeName? { return Parm{Name: i0.Text, Type: t0} }
) is:(
	_ "," i1:Ident t1:TypeName? { return Parm{Name: i1.Text, Type: t1} }
)* { return []Parm(append([]Parm{id}, is...)) }

Expr <- Call / Primary

Call <- c:( Nary / Binary / Unary ) cs:(
	_";" m:( UnaryMsg / BinMsg / NaryMsg ) { return Msg(m) }
)* {
	c.Msgs = append(c.Msgs, cs...)
	return Expr(c)
}

Unary <- r:( Primary / n:ModName+ { return Expr(ModName(n)) } ) ms:UnaryMsg+ {
	c := Call{start: r.Start(), end: ms[0].end, Recv: r, Msgs: []Msg{ms[0]}}
	for _, m := range ms[1:] {
		c = Call{start: r.Start(), end: m.end, Recv: c, Msgs: []Msg{m}}
	}
	return Call(c)
}

UnaryMsg <- i:Ident { return Msg{start: i.start, end: i.end, Sel: i.Text} }

Binary <- r:(
	u:Unary { return Expr(u) } /
	Primary /
	n:ModName+ { return Expr(ModName(n)) }
) m:BinMsg {
	return Call{start: r.Start(), end: end, Recv: r, Msgs: []Msg{m}}}

BinMsg <-
	n:Op a:(
		b:Binary { return Expr(b) } /
		u:Unary { return Expr(u) } /
		Primary
	) {
	return Msg{start: n.start, end: end, Sel: n.Text, Args: []Expr{a}}
}

Nary <- r:(
	b:Binary { return Expr(b) } /
	u:Unary { return Expr(u) } /
	Primary /
	n:ModName+ { return Expr(ModName(n)) }
)? m:NaryMsg {
	s := m.start
	var recv Expr
	if r != nil {
		s = (*r).Start()
		recv = *r
	}
	return Call{start: s, end: end, Recv: recv, Msgs: []Msg{m}}
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
	return Msg{start: as[0].name.start, end: end, Sel: sel, Args: es}
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
) _"}" ) { return Expr(Ctor{start: start, end: end, Type: t, Args: as}) }

Ary <- e0:Expr  es:( _ ";" e:Expr { return Expr(e) } )* ( _";" )? {
	return []Expr(append([]Expr{e0}, es...))
}

Block <- _ ( "[" (
	( ps:( n:CIdent t:TypeName? { return Parm{Name: n.Text, Type: t} } )+ _ "|" )? ss:Stmts
) _ "]" ) {
	return Expr(Block{start: start, end: end, Parms: ps, Stmts: ss})
}

# Tokens

Int "integer" <- _ tok:( text:( [+\-]? [0-9]+ ) {
	return Int{start: start, end: end, Text: text}
} ) { return Expr(tok) }

Float "floating point" <- _ tok:( text:( [+\-]? [0-9]+ "." [0-9]+ ( [eE] [+\-]? [0-9]+ )? ) {
	return Float{start: start, end: end, Text: text}
} ) { return Expr(tok) }

Rune "rune" <- _ tok:( text:( ['] !"\n" data:( Esc / "\\'" / [^'] ) ['] ) {
	r, w := utf8.DecodeRuneInString(data)
	if w != len(data) {
		panic("impossible")
	}
	return Rune{start: start, end: end, Text: text, Rune: r}
} ) { return Expr(tok) }

String "string" <-
	_ tok0:( text0:( ["] data0:( !"\n" ( Esc / "\\\"" / [^"] ) )* ["] ) {
		return String{start: start, end: end, Text: text0, Data: data0}
	} ) { return String(tok0) } /
	_ tok1:( text1: ( [`] data1:( "\\`" / [^`] )* [`] ) {
		return String{start: start, end: end, Text: text1, Data: data1}
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
	return Ident{start: start, end: end, Text: text}
} ) { return Ident(tok) }

ModName "module name" <- _ tok:( text:( "#" [_a-zA-Z][_a-zA-Z0-9]* ) {
	return Ident{start: start, end: end, Text: text}
} ) { return Ident(tok) }

IdentC "identifier:" <- _ !TypeVar !"import" tok:( text:( [_a-zA-Z][_a-zA-Z0-9]* ":" ) {
	return Ident{start: start, end: end, Text: text}
} ) { return Ident(tok) }

CIdent ":identifier"<- _ ":" !TypeVar !"import" tok:( text:( [_a-zA-Z][_a-zA-Z0-9]* ) {
	return Ident{start: start, end: end, Text: text}
} ) { return Ident(tok) }

Ident "identifier" <- _ !TypeVar !"import" tok:( text:( [_a-zA-Z][_a-zA-Z0-9]* ) !":" {
	return Ident{start: start, end: end, Text: text}
} ) { return Ident(tok) }

TypeVar "type variable" <- _ tok:( text:( [A-Z] ![_a-zA-Z0-9] ) {
	return Ident{start: start, end: end, Text: text }
} ) { return Ident(tok) }

_ "" <- ( Space / Cmnt )*

Cmnt <- "//" ( !"\n" . )* / "/*" ( !"*/" . )* "*/"

Space <- " " / "\t" / "\n"

EOF <- !.