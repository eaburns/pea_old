{
package main

import (
	"strconv"
	"unicode/utf8"

	"github.com/eaburns/peggy/peg"
)

type token struct {
	start, end int
	text string
	string string
	rune rune
}

func hex(s string) rune {
	x, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		panic("impossible")
	}
	return rune(x)
}

}

File <- Def* _ EOF

Def <- ModName* (
	_ "import" String /
	_ "(" Def+ _ ")" /
	Fun /
	Var /
	TypeSig (
		Type /
		Fun /
		_"(" ( Type / Fun )+ _")" 
	)
)

Fun <- _ "[" FunSig _ "|" Stmts _"]"

FunSig <- (
	Ident /
	Op Ident TypeName /
	( IdentC Ident TypeName )+
) Ret?

Ret <- _"^" TypeName

Var <- Ident _":=" _"[" Stmts _"]"

TypeSig <-
	!TypeVar TypeName /
	TypeParms? ( Ident / Op )

TypeParms <-
	TypeVar /
	_"(" TypeParm ( _"," TypeParm )* ( _ "," )? _")" 

TypeParm <-
	TypeName /
	TypeVar TypeName?

TypeName <-
	TypeVar? Ident* [?&] /
	TypeVar? Ident+ /
	TypeVar /
	_ "[" TypeName* ( _"|" TypeName )? _"]" /
	_"(" TypeName ( _"," TypeName )* ( _ "," )? _")" Ident+ /
	_ "(" TypeName _ ")"

Type <- _ "{" (
	( IdentC TypeName )+ /
	Case ( _ "," Case ) ( _ "," )? /
	MethSig+
) _ "}"

Case <- Ident / IdentC TypeName

MethSig <- _"[" ( Ident / Op TypeName / ( IdentC TypeName )+ ) Ret? _"]" 

Stmts <- ( Stmt ( _"." Stmt )* ( _ "." )? )?

Stmt <- Return / Assign / Expr

Return <- _ "^" Expr 

Assign <- Ident TypeName? ( _ "," Ident TypeName? )* _ ":=" Expr

Expr <- Cascade / Call / Primary 

Cascade <- Call ( _";" Ident / BinMsg / NaryMsg )+

Call <- Nary / Binary / Unary

Unary <- ( Primary / ModName+ ) Ident+

Binary <- ( Unary / Primary / ModName+ ) BinMsg

BinMsg <- Op ( Binary / Unary / Primary )

Nary <- ( Binary / Unary / Primary / ModName+ )? NaryMsg

NaryMsg <- ( IdentC ( Binary / Unary / Primary ) )+

Primary <- Ident / Int / Float / Rune / String / Ctor / Block / _ "(" Expr _ ")"

Ctor <- _"{" TypeName _"|" (
	( IdentC Expr )+ /
	( Expr ( _ ";" Expr )* ( _";" )? )?
) _"}"

Block <- _ "[" ( ( CIdent TypeName? )+ _ "|" )? Stmts _ "]"

# Tokens

Int "integer" <- _ ( text:( [+\-]? [0-9]+ ) {
	return token{start: start, end: end, text: text}
} ) { return "" }

Float "floating point" <- _ ( text:( [+\-]? [0-9]+ "." [0-9]+ ( [eE] [+\-]? [0-9]+ )? ) {
	return token{start: start, end: end, text: text}
} ) { return "" }

Rune "rune" <- _ ( text:( ['] !"\n" data:( Esc / "\\'" / [^'] ) ['] ) {
	r, w := utf8.DecodeRuneInString(data)
	if w != len(data) {
		panic("impossible")
	}
	return token{start: start, end: end, text: text, rune: r}
} ) { return "" }

String "string" <-
	_ ( text0:( ["] data0:( !"\n" ( Esc / "\\\"" / [^"] ) )* ["] ) {
		return token{start: start, end: end, text: text0, string: data0}
	} )  { return "" } /
	_ ( text1: ( [`] data1:( "\\`" / [^`] )* [`] ) {
		return token{start: start, end: end, text: text1, string: data1}
	} ) { return "" }
	
Esc <-
	"\\n" { return "\n" } /
	"\\t" { return "\t" }/
	"\\b" { return "\b" }/
	"\\\\" { return "\\" } /
	"\\" x0:( X X ) { return string(hex(x0)) } /
	"\\x" x1:( X X X X ) { return string(hex(x1)) } /
	"\\X" x2:( X X X X X X X X ) { return string(hex(x2)) }
	
X <- [a-fA-F0-9]

Op "operator" <- _ !"//" !"/*" ( text:( [!%&*+\-/<=>?@\\|~]+ ) {
	return token{start: start, end: end, text: text}
} ) { return "" }

ModName "module name" <- _ ( text:( "#" [_a-zA-Z][_a-zA-Z0-9]* ) {
	return token{start: start, end: end, text: text}
} ) { return "" }

IdentC "identifier:" <- _ !TypeVar !"import" ( text:( [_a-zA-Z][_a-zA-Z0-9]* ":" ) {
	return token{start: start, end: end, text: text}
} ) { return "" }

CIdent ":identifier"<- _ ":" !TypeVar !"import" ( text:( [_a-zA-Z][_a-zA-Z0-9]* ) {
	return token{start: start, end: end, text: text}
} ) { return "" }

Ident "identifier" <- _ !TypeVar !"import" ( text:( [_a-zA-Z][_a-zA-Z0-9]* ) !":" {
	return token{start: start, end: end, text: text}
} ) { return "" }

TypeVar "type variable" <- _ ( text:( [A-Z] ![_a-zA-Z0-9] ) {
	return token{start: start, end: end, text: text }
} ) { return "" }

_ "" <- ( Space / Cmnt )*

Cmnt <- "//" ( !"\n" . )* / "/*" ( !"*/" . )* "*/"

Space <- " " / "\t" / "\n"

EOF <- !.