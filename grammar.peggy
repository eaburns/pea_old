{
package main
import "github.com/eaburns/peggy/peg"
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

Int "integer" <- _ [+\-]? [0-9]+

Float "floating point" <- _ [+\-]? [0-9]+ "." [0-9]+ ( [eE] [+\-]? [0-9]+ )?

Rune "rune" <- _ ['] !"\n" ( Esc / "\\'" / [^'] ) [']

String "string" <-
	_ ["] ( !"\n" ( Esc / "\\\"" / [^"] ) )* ["] /
	_ [`] ( "\\`" / [^`] )* [`]
	
Esc <-
	"\\n" /
	"\\t" /
	"\\b" /
	"\\\\" /
	"\\" X X /
	"\\x" X X X X /
	"\\X" X X X X X X X X
	
X <- [a-fA-F0-9]

Op "operator" <- _ !"//" !"/*" [!%&*+\-/<=>?@\\|~]+

ModName "module name" <- _ "#" [_a-zA-Z][_a-zA-Z0-9]*

IdentC "identifier:" <- _ !TypeVar !"import" [_a-zA-Z][_a-zA-Z0-9]* ":"

CIdent ":identifier"<- _ ":" !TypeVar !"import" [_a-zA-Z][_a-zA-Z0-9]*

Ident "identifier" <- _ !TypeVar !"import" [_a-zA-Z][_a-zA-Z0-9]* !":"

TypeVar "type variable" <- _ [A-Z] ![_a-zA-Z0-9]

_ "" <- ( Space / Cmnt )*

Cmnt <- "//" ( !"\n" . )* / "/*" ( !"*/" . )* "*/"

Space <- " " / "\t" / "\n"

EOF <- !.