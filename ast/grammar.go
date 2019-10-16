package ast

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/eaburns/peggy/peg"
)

type parm struct {
	name Ident
	typ  TypeName
	key  Ident
}

type arg struct {
	name Ident
	val  Expr
}

type tname struct {
	mod  *Ident
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

const (
	_File         int = 0
	_Import       int = 1
	_Def          int = 2
	_Val          int = 3
	_Fun          int = 4
	_Meth         int = 5
	_Recv         int = 6
	_FunSig       int = 7
	_Parms        int = 8
	_Ret          int = 9
	_TypeSig      int = 10
	_TParms       int = 11
	_TParm        int = 12
	_TypeName     int = 13
	_TypeNameList int = 14
	_TName        int = 15
	_Type         int = 16
	_Alias        int = 17
	_And          int = 18
	_Field        int = 19
	_Or           int = 20
	_Case         int = 21
	_Virt         int = 22
	_MethSig      int = 23
	_Stmts        int = 24
	_Stmt         int = 25
	_Return       int = 26
	_Assign       int = 27
	_Lhs          int = 28
	_Expr         int = 29
	_Call         int = 30
	_Unary        int = 31
	_UnaryMsg     int = 32
	_Binary       int = 33
	_BinMsg       int = 34
	_Nary         int = 35
	_NaryMsg      int = 36
	_Primary      int = 37
	_Ctor         int = 38
	_Exprs        int = 39
	_Block        int = 40
	_Int          int = 41
	_Float        int = 42
	_Rune         int = 43
	_String       int = 44
	_Esc          int = 45
	_X            int = 46
	_Op           int = 47
	_TypeOp       int = 48
	_ModName      int = 49
	_IdentC       int = 50
	_CIdent       int = 51
	_Ident        int = 52
	_TypeVar      int = 53
	__            int = 54
	_Cmnt         int = 55
	_Space        int = 56
	_EOF          int = 57

	_N int = 58
)

type _Parser struct {
	text     string
	deltaPos [][_N]int32
	deltaErr [][_N]int32
	node     map[_key]*peg.Node
	fail     map[_key]*peg.Fail
	act      map[_key]interface{}
	lastFail int
	data     interface{}
}

type _key struct {
	start int
	rule  int
}

func _NewParser(text string) *_Parser {
	return &_Parser{
		text:     text,
		deltaPos: make([][_N]int32, len(text)+1),
		deltaErr: make([][_N]int32, len(text)+1),
		node:     make(map[_key]*peg.Node),
		fail:     make(map[_key]*peg.Fail),
		act:      make(map[_key]interface{}),
	}
}

func _max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func _memoize(parser *_Parser, rule, start, pos, perr int) (int, int) {
	parser.lastFail = perr
	derr := perr - start
	parser.deltaErr[start][rule] = int32(derr + 1)
	if pos >= 0 {
		dpos := pos - start
		parser.deltaPos[start][rule] = int32(dpos + 1)
		return dpos, derr
	}
	parser.deltaPos[start][rule] = -1
	return -1, derr
}

func _memo(parser *_Parser, rule, start int) (int, int, bool) {
	dp := parser.deltaPos[start][rule]
	if dp == 0 {
		return 0, 0, false
	}
	if dp > 0 {
		dp--
	}
	de := parser.deltaErr[start][rule] - 1
	return int(dp), int(de), true
}

func _failMemo(parser *_Parser, rule, start, errPos int) (int, *peg.Fail) {
	if start > parser.lastFail {
		return -1, &peg.Fail{}
	}
	dp := parser.deltaPos[start][rule]
	de := parser.deltaErr[start][rule]
	if start+int(de-1) < errPos {
		if dp > 0 {
			return start + int(dp-1), &peg.Fail{}
		}
		return -1, &peg.Fail{}
	}
	f := parser.fail[_key{start: start, rule: rule}]
	if dp < 0 && f != nil {
		return -1, f
	}
	if dp > 0 && f != nil {
		return start + int(dp-1), f
	}
	return start, nil
}

func _accept(parser *_Parser, f func(*_Parser, int) (int, int), pos, perr *int) bool {
	dp, de := f(parser, *pos)
	*perr = _max(*perr, *pos+de)
	if dp < 0 {
		return false
	}
	*pos += dp
	return true
}

func _node(parser *_Parser, f func(*_Parser, int) (int, *peg.Node), node *peg.Node, pos *int) bool {
	p, kid := f(parser, *pos)
	if kid == nil {
		return false
	}
	node.Kids = append(node.Kids, kid)
	*pos = p
	return true
}

func _fail(parser *_Parser, f func(*_Parser, int, int) (int, *peg.Fail), errPos int, node *peg.Fail, pos *int) bool {
	p, kid := f(parser, *pos, errPos)
	if kid.Want != "" || len(kid.Kids) > 0 {
		node.Kids = append(node.Kids, kid)
	}
	if p < 0 {
		return false
	}
	*pos = p
	return true
}

func _next(parser *_Parser, pos int) (rune, int) {
	r, w := peg.DecodeRuneInString(parser.text[pos:])
	return r, w
}

func _sub(parser *_Parser, start, end int, kids []*peg.Node) *peg.Node {
	node := &peg.Node{
		Text: parser.text[start:end],
		Kids: make([]*peg.Node, len(kids)),
	}
	copy(node.Kids, kids)
	return node
}

func _leaf(parser *_Parser, start, end int) *peg.Node {
	return &peg.Node{Text: parser.text[start:end]}
}

// A no-op function to mark a variable as used.
func use(interface{}) {}

func _FileAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _File, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// imports:Import* defs:Def* _ EOF
	// imports:Import*
	{
		pos1 := pos
		// Import*
		for {
			pos3 := pos
			// Import
			if !_accept(parser, _ImportAccepts, &pos, &perr) {
				goto fail5
			}
			continue
		fail5:
			pos = pos3
			break
		}
		labels[0] = parser.text[pos1:pos]
	}
	// defs:Def*
	{
		pos6 := pos
		// Def*
		for {
			pos8 := pos
			// Def
			if !_accept(parser, _DefAccepts, &pos, &perr) {
				goto fail10
			}
			continue
		fail10:
			pos = pos8
			break
		}
		labels[1] = parser.text[pos6:pos]
	}
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// EOF
	if !_accept(parser, _EOFAccepts, &pos, &perr) {
		goto fail
	}
	return _memoize(parser, _File, start, pos, perr)
fail:
	return _memoize(parser, _File, start, -1, perr)
}

func _FileFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _File, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "File",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _File}
	// action
	// imports:Import* defs:Def* _ EOF
	// imports:Import*
	{
		pos1 := pos
		// Import*
		for {
			pos3 := pos
			// Import
			if !_fail(parser, _ImportFail, errPos, failure, &pos) {
				goto fail5
			}
			continue
		fail5:
			pos = pos3
			break
		}
		labels[0] = parser.text[pos1:pos]
	}
	// defs:Def*
	{
		pos6 := pos
		// Def*
		for {
			pos8 := pos
			// Def
			if !_fail(parser, _DefFail, errPos, failure, &pos) {
				goto fail10
			}
			continue
		fail10:
			pos = pos8
			break
		}
		labels[1] = parser.text[pos6:pos]
	}
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// EOF
	if !_fail(parser, _EOFFail, errPos, failure, &pos) {
		goto fail
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _FileAction(parser *_Parser, start int) (int, *File) {
	var labels [2]string
	use(labels)
	var label0 []Import
	var label1 []Def
	dp := parser.deltaPos[start][_File]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _File}
	n := parser.act[key]
	if n != nil {
		n := n.(File)
		return start + int(dp-1), &n
	}
	var node File
	pos := start
	// action
	{
		start0 := pos
		// imports:Import* defs:Def* _ EOF
		// imports:Import*
		{
			pos2 := pos
			// Import*
			for {
				pos4 := pos
				var node5 Import
				// Import
				if p, n := _ImportAction(parser, pos); n == nil {
					goto fail6
				} else {
					node5 = *n
					pos = p
				}
				label0 = append(label0, node5)
				continue
			fail6:
				pos = pos4
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		// defs:Def*
		{
			pos7 := pos
			// Def*
			for {
				pos9 := pos
				var node10 Def
				// Def
				if p, n := _DefAction(parser, pos); n == nil {
					goto fail11
				} else {
					node10 = *n
					pos = p
				}
				label1 = append(label1, node10)
				continue
			fail11:
				pos = pos9
				break
			}
			labels[1] = parser.text[pos7:pos]
		}
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// EOF
		if p, n := _EOFAction(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		node = func(
			start, end int, defs []Def, imports []Import) File {
			return File{
				Imports: imports,
				Defs:    defs,
			}
		}(
			start0, pos, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ImportAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Import, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ i:("import" path:String {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// i:("import" path:String {…})
	{
		pos1 := pos
		// ("import" path:String {…})
		// action
		// "import" path:String
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			perr = _max(perr, pos)
			goto fail
		}
		pos += 6
		// path:String
		{
			pos3 := pos
			// String
			if !_accept(parser, _StringAccepts, &pos, &perr) {
				goto fail
			}
			labels[0] = parser.text[pos3:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Import, start, pos, perr)
fail:
	return _memoize(parser, _Import, start, -1, perr)
}

func _ImportFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Import, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Import",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Import}
	// action
	// _ i:("import" path:String {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// i:("import" path:String {…})
	{
		pos1 := pos
		// ("import" path:String {…})
		// action
		// "import" path:String
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"import\"",
				})
			}
			goto fail
		}
		pos += 6
		// path:String
		{
			pos3 := pos
			// String
			if !_fail(parser, _StringFail, errPos, failure, &pos) {
				goto fail
			}
			labels[0] = parser.text[pos3:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _ImportAction(parser *_Parser, start int) (int, *Import) {
	var labels [2]string
	use(labels)
	var label0 String
	var label1 Import
	dp := parser.deltaPos[start][_Import]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Import}
	n := parser.act[key]
	if n != nil {
		n := n.(Import)
		return start + int(dp-1), &n
	}
	var node Import
	pos := start
	// action
	{
		start0 := pos
		// _ i:("import" path:String {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// i:("import" path:String {…})
		{
			pos2 := pos
			// ("import" path:String {…})
			// action
			{
				start3 := pos
				// "import" path:String
				// "import"
				if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
					goto fail
				}
				pos += 6
				// path:String
				{
					pos5 := pos
					// String
					if p, n := _StringAction(parser, pos); n == nil {
						goto fail
					} else {
						label0 = *n
						pos = p
					}
					labels[0] = parser.text[pos5:pos]
				}
				label1 = func(
					start, end int, path String) Import {
					return Import{
						location: loc(parser, start, end),
						Path:     path.Text,
					}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, i Import, path String) Import {
			return Import(i)
		}(
			start0, pos, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _DefAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Def, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Val/Fun/Meth/Type
	{
		pos3 := pos
		// Val
		if !_accept(parser, _ValAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Fun
		if !_accept(parser, _FunAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Meth
		if !_accept(parser, _MethAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		// Type
		if !_accept(parser, _TypeAccepts, &pos, &perr) {
			goto fail7
		}
		goto ok0
	fail7:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Def, start, pos, perr)
fail:
	return _memoize(parser, _Def, start, -1, perr)
}

func _DefFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Def, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Def",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Def}
	// Val/Fun/Meth/Type
	{
		pos3 := pos
		// Val
		if !_fail(parser, _ValFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Fun
		if !_fail(parser, _FunFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Meth
		if !_fail(parser, _MethFail, errPos, failure, &pos) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		// Type
		if !_fail(parser, _TypeFail, errPos, failure, &pos) {
			goto fail7
		}
		goto ok0
	fail7:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _DefAction(parser *_Parser, start int) (int, *Def) {
	dp := parser.deltaPos[start][_Def]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Def}
	n := parser.act[key]
	if n != nil {
		n := n.(Def)
		return start + int(dp-1), &n
	}
	var node Def
	pos := start
	// Val/Fun/Meth/Type
	{
		pos3 := pos
		var node2 Def
		// Val
		if p, n := _ValAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Fun
		if p, n := _FunAction(parser, pos); n == nil {
			goto fail5
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail5:
		node = node2
		pos = pos3
		// Meth
		if p, n := _MethAction(parser, pos); n == nil {
			goto fail6
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail6:
		node = node2
		pos = pos3
		// Type
		if p, n := _TypeAction(parser, pos); n == nil {
			goto fail7
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail7:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ValAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [5]string
	use(labels)
	if dp, de, ok := _memo(parser, _Val, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ v:(key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// v:(key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
	{
		pos1 := pos
		// (key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
		// action
		// key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]"
		// key:("val"/"Val")
		{
			pos3 := pos
			// ("val"/"Val")
			// "val"/"Val"
			{
				pos7 := pos
				// "val"
				if len(parser.text[pos:]) < 3 || parser.text[pos:pos+3] != "val" {
					perr = _max(perr, pos)
					goto fail8
				}
				pos += 3
				goto ok4
			fail8:
				pos = pos7
				// "Val"
				if len(parser.text[pos:]) < 3 || parser.text[pos:pos+3] != "Val" {
					perr = _max(perr, pos)
					goto fail9
				}
				pos += 3
				goto ok4
			fail9:
				pos = pos7
				goto fail
			ok4:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// id:Ident
		{
			pos10 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail
			}
			labels[1] = parser.text[pos10:pos]
		}
		// typ:TypeName?
		{
			pos11 := pos
			// TypeName?
			{
				pos13 := pos
				// TypeName
				if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
					goto fail14
				}
				goto ok15
			fail14:
				pos = pos13
			ok15:
			}
			labels[2] = parser.text[pos11:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// ":="
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
			perr = _max(perr, pos)
			goto fail
		}
		pos += 2
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// stmts:Stmts
		{
			pos16 := pos
			// Stmts
			if !_accept(parser, _StmtsAccepts, &pos, &perr) {
				goto fail
			}
			labels[3] = parser.text[pos16:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		labels[4] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Val, start, pos, perr)
fail:
	return _memoize(parser, _Val, start, -1, perr)
}

func _ValFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [5]string
	use(labels)
	pos, failure := _failMemo(parser, _Val, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Val",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Val}
	// action
	// _ v:(key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// v:(key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
	{
		pos1 := pos
		// (key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
		// action
		// key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]"
		// key:("val"/"Val")
		{
			pos3 := pos
			// ("val"/"Val")
			// "val"/"Val"
			{
				pos7 := pos
				// "val"
				if len(parser.text[pos:]) < 3 || parser.text[pos:pos+3] != "val" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"val\"",
						})
					}
					goto fail8
				}
				pos += 3
				goto ok4
			fail8:
				pos = pos7
				// "Val"
				if len(parser.text[pos:]) < 3 || parser.text[pos:pos+3] != "Val" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"Val\"",
						})
					}
					goto fail9
				}
				pos += 3
				goto ok4
			fail9:
				pos = pos7
				goto fail
			ok4:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// id:Ident
		{
			pos10 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail
			}
			labels[1] = parser.text[pos10:pos]
		}
		// typ:TypeName?
		{
			pos11 := pos
			// TypeName?
			{
				pos13 := pos
				// TypeName
				if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
					goto fail14
				}
				goto ok15
			fail14:
				pos = pos13
			ok15:
			}
			labels[2] = parser.text[pos11:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// ":="
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\":=\"",
				})
			}
			goto fail
		}
		pos += 2
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"[\"",
				})
			}
			goto fail
		}
		pos++
		// stmts:Stmts
		{
			pos16 := pos
			// Stmts
			if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
				goto fail
			}
			labels[3] = parser.text[pos16:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"]\"",
				})
			}
			goto fail
		}
		pos++
		labels[4] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _ValAction(parser *_Parser, start int) (int, *Def) {
	var labels [5]string
	use(labels)
	var label0 string
	var label1 Ident
	var label2 *TypeName
	var label3 []Stmt
	var label4 *Val
	dp := parser.deltaPos[start][_Val]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Val}
	n := parser.act[key]
	if n != nil {
		n := n.(Def)
		return start + int(dp-1), &n
	}
	var node Def
	pos := start
	// action
	{
		start0 := pos
		// _ v:(key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// v:(key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
		{
			pos2 := pos
			// (key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]" {…})
			// action
			{
				start3 := pos
				// key:("val"/"Val") id:Ident typ:TypeName? _ ":=" _ "[" stmts:Stmts _ "]"
				// key:("val"/"Val")
				{
					pos5 := pos
					// ("val"/"Val")
					// "val"/"Val"
					{
						pos9 := pos
						var node8 string
						// "val"
						if len(parser.text[pos:]) < 3 || parser.text[pos:pos+3] != "val" {
							goto fail10
						}
						label0 = parser.text[pos : pos+3]
						pos += 3
						goto ok6
					fail10:
						label0 = node8
						pos = pos9
						// "Val"
						if len(parser.text[pos:]) < 3 || parser.text[pos:pos+3] != "Val" {
							goto fail11
						}
						label0 = parser.text[pos : pos+3]
						pos += 3
						goto ok6
					fail11:
						label0 = node8
						pos = pos9
						goto fail
					ok6:
					}
					labels[0] = parser.text[pos5:pos]
				}
				// id:Ident
				{
					pos12 := pos
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail
					} else {
						label1 = *n
						pos = p
					}
					labels[1] = parser.text[pos12:pos]
				}
				// typ:TypeName?
				{
					pos13 := pos
					// TypeName?
					{
						pos15 := pos
						label2 = new(TypeName)
						// TypeName
						if p, n := _TypeNameAction(parser, pos); n == nil {
							goto fail16
						} else {
							*label2 = *n
							pos = p
						}
						goto ok17
					fail16:
						label2 = nil
						pos = pos15
					ok17:
					}
					labels[2] = parser.text[pos13:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// ":="
				if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
					goto fail
				}
				pos += 2
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "["
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
					goto fail
				}
				pos++
				// stmts:Stmts
				{
					pos18 := pos
					// Stmts
					if p, n := _StmtsAction(parser, pos); n == nil {
						goto fail
					} else {
						label3 = *n
						pos = p
					}
					labels[3] = parser.text[pos18:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "]"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
					goto fail
				}
				pos++
				label4 = func(
					start, end int, id Ident, key string, stmts []Stmt, typ *TypeName) *Val {
					varEnd := id.end
					if typ != nil {
						varEnd = typ.end
					}
					return &Val{
						location: loc(parser, start, end),
						priv:     key == "val",
						Var: Var{
							location: location{start: id.start, end: varEnd},
							Name:     id.Text,
							Type:     typ,
						},
						Init: stmts,
					}
				}(
					start3, pos, label1, label0, label3, label2)
			}
			labels[4] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, id Ident, key string, stmts []Stmt, typ *TypeName, v *Val) Def {
			return Def(v)
		}(
			start0, pos, label1, label0, label3, label2, label4)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FunAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [5]string
	use(labels)
	if dp, de, ok := _memo(parser, _Fun, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ f:(key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// f:(key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
	{
		pos1 := pos
		// (key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
		// action
		// key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]"
		// key:("func"/"Func")
		{
			pos3 := pos
			// ("func"/"Func")
			// "func"/"Func"
			{
				pos7 := pos
				// "func"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "func" {
					perr = _max(perr, pos)
					goto fail8
				}
				pos += 4
				goto ok4
			fail8:
				pos = pos7
				// "Func"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Func" {
					perr = _max(perr, pos)
					goto fail9
				}
				pos += 4
				goto ok4
			fail9:
				pos = pos7
				goto fail
			ok4:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// tps:TParms
		{
			pos10 := pos
			// TParms
			if !_accept(parser, _TParmsAccepts, &pos, &perr) {
				goto fail
			}
			labels[1] = parser.text[pos10:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// sig:FunSig
		{
			pos11 := pos
			// FunSig
			if !_accept(parser, _FunSigAccepts, &pos, &perr) {
				goto fail
			}
			labels[2] = parser.text[pos11:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// stmts:Stmts
		{
			pos12 := pos
			// Stmts
			if !_accept(parser, _StmtsAccepts, &pos, &perr) {
				goto fail
			}
			labels[3] = parser.text[pos12:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		labels[4] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Fun, start, pos, perr)
fail:
	return _memoize(parser, _Fun, start, -1, perr)
}

func _FunFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [5]string
	use(labels)
	pos, failure := _failMemo(parser, _Fun, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Fun",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Fun}
	// action
	// _ f:(key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// f:(key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
	{
		pos1 := pos
		// (key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
		// action
		// key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]"
		// key:("func"/"Func")
		{
			pos3 := pos
			// ("func"/"Func")
			// "func"/"Func"
			{
				pos7 := pos
				// "func"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "func" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"func\"",
						})
					}
					goto fail8
				}
				pos += 4
				goto ok4
			fail8:
				pos = pos7
				// "Func"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Func" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"Func\"",
						})
					}
					goto fail9
				}
				pos += 4
				goto ok4
			fail9:
				pos = pos7
				goto fail
			ok4:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// tps:TParms
		{
			pos10 := pos
			// TParms
			if !_fail(parser, _TParmsFail, errPos, failure, &pos) {
				goto fail
			}
			labels[1] = parser.text[pos10:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"[\"",
				})
			}
			goto fail
		}
		pos++
		// sig:FunSig
		{
			pos11 := pos
			// FunSig
			if !_fail(parser, _FunSigFail, errPos, failure, &pos) {
				goto fail
			}
			labels[2] = parser.text[pos11:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"|\"",
				})
			}
			goto fail
		}
		pos++
		// stmts:Stmts
		{
			pos12 := pos
			// Stmts
			if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
				goto fail
			}
			labels[3] = parser.text[pos12:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"]\"",
				})
			}
			goto fail
		}
		pos++
		labels[4] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _FunAction(parser *_Parser, start int) (int, *Def) {
	var labels [5]string
	use(labels)
	var label0 string
	var label1 ([]Var)
	var label2 FunSig
	var label3 []Stmt
	var label4 *Fun
	dp := parser.deltaPos[start][_Fun]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Fun}
	n := parser.act[key]
	if n != nil {
		n := n.(Def)
		return start + int(dp-1), &n
	}
	var node Def
	pos := start
	// action
	{
		start0 := pos
		// _ f:(key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// f:(key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
		{
			pos2 := pos
			// (key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
			// action
			{
				start3 := pos
				// key:("func"/"Func") tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]"
				// key:("func"/"Func")
				{
					pos5 := pos
					// ("func"/"Func")
					// "func"/"Func"
					{
						pos9 := pos
						var node8 string
						// "func"
						if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "func" {
							goto fail10
						}
						label0 = parser.text[pos : pos+4]
						pos += 4
						goto ok6
					fail10:
						label0 = node8
						pos = pos9
						// "Func"
						if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Func" {
							goto fail11
						}
						label0 = parser.text[pos : pos+4]
						pos += 4
						goto ok6
					fail11:
						label0 = node8
						pos = pos9
						goto fail
					ok6:
					}
					labels[0] = parser.text[pos5:pos]
				}
				// tps:TParms
				{
					pos12 := pos
					// TParms
					if p, n := _TParmsAction(parser, pos); n == nil {
						goto fail
					} else {
						label1 = *n
						pos = p
					}
					labels[1] = parser.text[pos12:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "["
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
					goto fail
				}
				pos++
				// sig:FunSig
				{
					pos13 := pos
					// FunSig
					if p, n := _FunSigAction(parser, pos); n == nil {
						goto fail
					} else {
						label2 = *n
						pos = p
					}
					labels[2] = parser.text[pos13:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "|"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
					goto fail
				}
				pos++
				// stmts:Stmts
				{
					pos14 := pos
					// Stmts
					if p, n := _StmtsAction(parser, pos); n == nil {
						goto fail
					} else {
						label3 = *n
						pos = p
					}
					labels[3] = parser.text[pos14:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "]"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
					goto fail
				}
				pos++
				label4 = func(
					start, end int, key string, sig FunSig, stmts []Stmt, tps []Var) *Fun {
					return &Fun{
						location: loc(parser, start, end),
						priv:     key == "func",
						TParms:   tps,
						Sig:      sig,
						Stmts:    stmts,
					}
				}(
					start3, pos, label0, label2, label3, label1)
			}
			labels[4] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, f *Fun, key string, sig FunSig, stmts []Stmt, tps []Var) Def {
			return Def(f)
		}(
			start0, pos, label4, label0, label2, label3, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _MethAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [6]string
	use(labels)
	if dp, de, ok := _memo(parser, _Meth, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ m:(key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// m:(key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
	{
		pos1 := pos
		// (key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
		// action
		// key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]"
		// key:("meth"/"Meth")
		{
			pos3 := pos
			// ("meth"/"Meth")
			// "meth"/"Meth"
			{
				pos7 := pos
				// "meth"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "meth" {
					perr = _max(perr, pos)
					goto fail8
				}
				pos += 4
				goto ok4
			fail8:
				pos = pos7
				// "Meth"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Meth" {
					perr = _max(perr, pos)
					goto fail9
				}
				pos += 4
				goto ok4
			fail9:
				pos = pos7
				goto fail
			ok4:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// recv:Recv
		{
			pos10 := pos
			// Recv
			if !_accept(parser, _RecvAccepts, &pos, &perr) {
				goto fail
			}
			labels[1] = parser.text[pos10:pos]
		}
		// tps:TParms
		{
			pos11 := pos
			// TParms
			if !_accept(parser, _TParmsAccepts, &pos, &perr) {
				goto fail
			}
			labels[2] = parser.text[pos11:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// sig:FunSig
		{
			pos12 := pos
			// FunSig
			if !_accept(parser, _FunSigAccepts, &pos, &perr) {
				goto fail
			}
			labels[3] = parser.text[pos12:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// stmts:Stmts
		{
			pos13 := pos
			// Stmts
			if !_accept(parser, _StmtsAccepts, &pos, &perr) {
				goto fail
			}
			labels[4] = parser.text[pos13:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		labels[5] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Meth, start, pos, perr)
fail:
	return _memoize(parser, _Meth, start, -1, perr)
}

func _MethFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [6]string
	use(labels)
	pos, failure := _failMemo(parser, _Meth, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Meth",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Meth}
	// action
	// _ m:(key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// m:(key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
	{
		pos1 := pos
		// (key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
		// action
		// key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]"
		// key:("meth"/"Meth")
		{
			pos3 := pos
			// ("meth"/"Meth")
			// "meth"/"Meth"
			{
				pos7 := pos
				// "meth"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "meth" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"meth\"",
						})
					}
					goto fail8
				}
				pos += 4
				goto ok4
			fail8:
				pos = pos7
				// "Meth"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Meth" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"Meth\"",
						})
					}
					goto fail9
				}
				pos += 4
				goto ok4
			fail9:
				pos = pos7
				goto fail
			ok4:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// recv:Recv
		{
			pos10 := pos
			// Recv
			if !_fail(parser, _RecvFail, errPos, failure, &pos) {
				goto fail
			}
			labels[1] = parser.text[pos10:pos]
		}
		// tps:TParms
		{
			pos11 := pos
			// TParms
			if !_fail(parser, _TParmsFail, errPos, failure, &pos) {
				goto fail
			}
			labels[2] = parser.text[pos11:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"[\"",
				})
			}
			goto fail
		}
		pos++
		// sig:FunSig
		{
			pos12 := pos
			// FunSig
			if !_fail(parser, _FunSigFail, errPos, failure, &pos) {
				goto fail
			}
			labels[3] = parser.text[pos12:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"|\"",
				})
			}
			goto fail
		}
		pos++
		// stmts:Stmts
		{
			pos13 := pos
			// Stmts
			if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
				goto fail
			}
			labels[4] = parser.text[pos13:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"]\"",
				})
			}
			goto fail
		}
		pos++
		labels[5] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _MethAction(parser *_Parser, start int) (int, *Def) {
	var labels [6]string
	use(labels)
	var label0 string
	var label1 Recv
	var label2 ([]Var)
	var label3 FunSig
	var label4 []Stmt
	var label5 *Fun
	dp := parser.deltaPos[start][_Meth]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Meth}
	n := parser.act[key]
	if n != nil {
		n := n.(Def)
		return start + int(dp-1), &n
	}
	var node Def
	pos := start
	// action
	{
		start0 := pos
		// _ m:(key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// m:(key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
		{
			pos2 := pos
			// (key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]" {…})
			// action
			{
				start3 := pos
				// key:("meth"/"Meth") recv:Recv tps:TParms _ "[" sig:FunSig _ "|" stmts:Stmts _ "]"
				// key:("meth"/"Meth")
				{
					pos5 := pos
					// ("meth"/"Meth")
					// "meth"/"Meth"
					{
						pos9 := pos
						var node8 string
						// "meth"
						if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "meth" {
							goto fail10
						}
						label0 = parser.text[pos : pos+4]
						pos += 4
						goto ok6
					fail10:
						label0 = node8
						pos = pos9
						// "Meth"
						if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Meth" {
							goto fail11
						}
						label0 = parser.text[pos : pos+4]
						pos += 4
						goto ok6
					fail11:
						label0 = node8
						pos = pos9
						goto fail
					ok6:
					}
					labels[0] = parser.text[pos5:pos]
				}
				// recv:Recv
				{
					pos12 := pos
					// Recv
					if p, n := _RecvAction(parser, pos); n == nil {
						goto fail
					} else {
						label1 = *n
						pos = p
					}
					labels[1] = parser.text[pos12:pos]
				}
				// tps:TParms
				{
					pos13 := pos
					// TParms
					if p, n := _TParmsAction(parser, pos); n == nil {
						goto fail
					} else {
						label2 = *n
						pos = p
					}
					labels[2] = parser.text[pos13:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "["
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
					goto fail
				}
				pos++
				// sig:FunSig
				{
					pos14 := pos
					// FunSig
					if p, n := _FunSigAction(parser, pos); n == nil {
						goto fail
					} else {
						label3 = *n
						pos = p
					}
					labels[3] = parser.text[pos14:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "|"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
					goto fail
				}
				pos++
				// stmts:Stmts
				{
					pos15 := pos
					// Stmts
					if p, n := _StmtsAction(parser, pos); n == nil {
						goto fail
					} else {
						label4 = *n
						pos = p
					}
					labels[4] = parser.text[pos15:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "]"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
					goto fail
				}
				pos++
				label5 = func(
					start, end int, key string, recv Recv, sig FunSig, stmts []Stmt, tps []Var) *Fun {
					return &Fun{
						location: loc(parser, start, end),
						priv:     key == "meth",
						Recv:     &recv,
						TParms:   tps,
						Sig:      sig,
						Stmts:    stmts,
					}
				}(
					start3, pos, label0, label1, label3, label4, label2)
			}
			labels[5] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, key string, m *Fun, recv Recv, sig FunSig, stmts []Stmt, tps []Var) Def {
			return Def(m)
		}(
			start0, pos, label0, label5, label1, label3, label4, label2)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _RecvAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Recv, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// tps:TParms mod:ModName? n:(Ident/Op)
	// tps:TParms
	{
		pos1 := pos
		// TParms
		if !_accept(parser, _TParmsAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// mod:ModName?
	{
		pos2 := pos
		// ModName?
		{
			pos4 := pos
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail5
			}
			goto ok6
		fail5:
			pos = pos4
		ok6:
		}
		labels[1] = parser.text[pos2:pos]
	}
	// n:(Ident/Op)
	{
		pos7 := pos
		// (Ident/Op)
		// Ident/Op
		{
			pos11 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail12
			}
			goto ok8
		fail12:
			pos = pos11
			// Op
			if !_accept(parser, _OpAccepts, &pos, &perr) {
				goto fail13
			}
			goto ok8
		fail13:
			pos = pos11
			goto fail
		ok8:
		}
		labels[2] = parser.text[pos7:pos]
	}
	return _memoize(parser, _Recv, start, pos, perr)
fail:
	return _memoize(parser, _Recv, start, -1, perr)
}

func _RecvFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Recv, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Recv",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Recv}
	// action
	// tps:TParms mod:ModName? n:(Ident/Op)
	// tps:TParms
	{
		pos1 := pos
		// TParms
		if !_fail(parser, _TParmsFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// mod:ModName?
	{
		pos2 := pos
		// ModName?
		{
			pos4 := pos
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail5
			}
			goto ok6
		fail5:
			pos = pos4
		ok6:
		}
		labels[1] = parser.text[pos2:pos]
	}
	// n:(Ident/Op)
	{
		pos7 := pos
		// (Ident/Op)
		// Ident/Op
		{
			pos11 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail12
			}
			goto ok8
		fail12:
			pos = pos11
			// Op
			if !_fail(parser, _OpFail, errPos, failure, &pos) {
				goto fail13
			}
			goto ok8
		fail13:
			pos = pos11
			goto fail
		ok8:
		}
		labels[2] = parser.text[pos7:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _RecvAction(parser *_Parser, start int) (int, *Recv) {
	var labels [3]string
	use(labels)
	var label0 ([]Var)
	var label1 *Ident
	var label2 Ident
	dp := parser.deltaPos[start][_Recv]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Recv}
	n := parser.act[key]
	if n != nil {
		n := n.(Recv)
		return start + int(dp-1), &n
	}
	var node Recv
	pos := start
	// action
	{
		start0 := pos
		// tps:TParms mod:ModName? n:(Ident/Op)
		// tps:TParms
		{
			pos2 := pos
			// TParms
			if p, n := _TParmsAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// mod:ModName?
		{
			pos3 := pos
			// ModName?
			{
				pos5 := pos
				label1 = new(Ident)
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail6
				} else {
					*label1 = *n
					pos = p
				}
				goto ok7
			fail6:
				label1 = nil
				pos = pos5
			ok7:
			}
			labels[1] = parser.text[pos3:pos]
		}
		// n:(Ident/Op)
		{
			pos8 := pos
			// (Ident/Op)
			// Ident/Op
			{
				pos12 := pos
				var node11 Ident
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail13
				} else {
					label2 = *n
					pos = p
				}
				goto ok9
			fail13:
				label2 = node11
				pos = pos12
				// Op
				if p, n := _OpAction(parser, pos); n == nil {
					goto fail14
				} else {
					label2 = *n
					pos = p
				}
				goto ok9
			fail14:
				label2 = node11
				pos = pos12
				goto fail
			ok9:
			}
			labels[2] = parser.text[pos8:pos]
		}
		node = func(
			start, end int, mod *Ident, n Ident, tps []Var) Recv {
			l := loc(parser, start, end)
			if len(tps) > 0 {
				l.start = tps[0].start
			}
			return Recv{
				TypeSig: TypeSig{
					location: l,
					Name:     n.Text,
					Parms:    tps,
				},
				Mod: mod,
			}
		}(
			start0, pos, label1, label2, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FunSigAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _FunSig, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// ps:Parms r:Ret?
	// ps:Parms
	{
		pos1 := pos
		// Parms
		if !_accept(parser, _ParmsAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// r:Ret?
	{
		pos2 := pos
		// Ret?
		{
			pos4 := pos
			// Ret
			if !_accept(parser, _RetAccepts, &pos, &perr) {
				goto fail5
			}
			goto ok6
		fail5:
			pos = pos4
		ok6:
		}
		labels[1] = parser.text[pos2:pos]
	}
	return _memoize(parser, _FunSig, start, pos, perr)
fail:
	return _memoize(parser, _FunSig, start, -1, perr)
}

func _FunSigFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _FunSig, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "FunSig",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _FunSig}
	// action
	// ps:Parms r:Ret?
	// ps:Parms
	{
		pos1 := pos
		// Parms
		if !_fail(parser, _ParmsFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// r:Ret?
	{
		pos2 := pos
		// Ret?
		{
			pos4 := pos
			// Ret
			if !_fail(parser, _RetFail, errPos, failure, &pos) {
				goto fail5
			}
			goto ok6
		fail5:
			pos = pos4
		ok6:
		}
		labels[1] = parser.text[pos2:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _FunSigAction(parser *_Parser, start int) (int, *FunSig) {
	var labels [2]string
	use(labels)
	var label0 []parm
	var label1 *TypeName
	dp := parser.deltaPos[start][_FunSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _FunSig}
	n := parser.act[key]
	if n != nil {
		n := n.(FunSig)
		return start + int(dp-1), &n
	}
	var node FunSig
	pos := start
	// action
	{
		start0 := pos
		// ps:Parms r:Ret?
		// ps:Parms
		{
			pos2 := pos
			// Parms
			if p, n := _ParmsAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// r:Ret?
		{
			pos3 := pos
			// Ret?
			{
				pos5 := pos
				label1 = new(TypeName)
				// Ret
				if p, n := _RetAction(parser, pos); n == nil {
					goto fail6
				} else {
					*label1 = *n
					pos = p
				}
				goto ok7
			fail6:
				label1 = nil
				pos = pos5
			ok7:
			}
			labels[1] = parser.text[pos3:pos]
		}
		node = func(
			start, end int, ps []parm, r *TypeName) FunSig {
			if len(ps) == 1 && ps[0].name.Text == "" {
				p := ps[0]
				return FunSig{
					location: location{p.key.start, p.typ.end},
					Sel:      p.key.Text,
					Ret:      r,
				}
			}
			var sel string
			var parms []Var
			for i := range ps {
				p := &ps[i]
				sel += p.key.Text
				parms = append(parms, Var{
					location: location{p.key.start, p.typ.end},
					Name:     p.name.Text,
					Type:     &p.typ,
				})
			}
			return FunSig{
				location: loc(parser, start, end),
				Sel:      sel,
				Parms:    parms,
				Ret:      r,
			}
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ParmsAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [7]string
	use(labels)
	if dp, de, ok := _memo(parser, _Parms, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+
	{
		pos3 := pos
		// action
		// id0:Ident
		{
			pos5 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// o:Op id1:Ident t0:TypeName
		// o:Op
		{
			pos8 := pos
			// Op
			if !_accept(parser, _OpAccepts, &pos, &perr) {
				goto fail6
			}
			labels[1] = parser.text[pos8:pos]
		}
		// id1:Ident
		{
			pos9 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail6
			}
			labels[2] = parser.text[pos9:pos]
		}
		// t0:TypeName
		{
			pos10 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail6
			}
			labels[3] = parser.text[pos10:pos]
		}
		goto ok0
	fail6:
		pos = pos3
		// (c:IdentC id2:Ident t1:TypeName {…})+
		// (c:IdentC id2:Ident t1:TypeName {…})
		// action
		// c:IdentC id2:Ident t1:TypeName
		// c:IdentC
		{
			pos17 := pos
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail11
			}
			labels[4] = parser.text[pos17:pos]
		}
		// id2:Ident
		{
			pos18 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail11
			}
			labels[5] = parser.text[pos18:pos]
		}
		// t1:TypeName
		{
			pos19 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail11
			}
			labels[6] = parser.text[pos19:pos]
		}
		for {
			pos13 := pos
			// (c:IdentC id2:Ident t1:TypeName {…})
			// action
			// c:IdentC id2:Ident t1:TypeName
			// c:IdentC
			{
				pos21 := pos
				// IdentC
				if !_accept(parser, _IdentCAccepts, &pos, &perr) {
					goto fail15
				}
				labels[4] = parser.text[pos21:pos]
			}
			// id2:Ident
			{
				pos22 := pos
				// Ident
				if !_accept(parser, _IdentAccepts, &pos, &perr) {
					goto fail15
				}
				labels[5] = parser.text[pos22:pos]
			}
			// t1:TypeName
			{
				pos23 := pos
				// TypeName
				if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
					goto fail15
				}
				labels[6] = parser.text[pos23:pos]
			}
			continue
		fail15:
			pos = pos13
			break
		}
		goto ok0
	fail11:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Parms, start, pos, perr)
fail:
	return _memoize(parser, _Parms, start, -1, perr)
}

func _ParmsFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [7]string
	use(labels)
	pos, failure := _failMemo(parser, _Parms, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Parms",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Parms}
	// id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+
	{
		pos3 := pos
		// action
		// id0:Ident
		{
			pos5 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// o:Op id1:Ident t0:TypeName
		// o:Op
		{
			pos8 := pos
			// Op
			if !_fail(parser, _OpFail, errPos, failure, &pos) {
				goto fail6
			}
			labels[1] = parser.text[pos8:pos]
		}
		// id1:Ident
		{
			pos9 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail6
			}
			labels[2] = parser.text[pos9:pos]
		}
		// t0:TypeName
		{
			pos10 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail6
			}
			labels[3] = parser.text[pos10:pos]
		}
		goto ok0
	fail6:
		pos = pos3
		// (c:IdentC id2:Ident t1:TypeName {…})+
		// (c:IdentC id2:Ident t1:TypeName {…})
		// action
		// c:IdentC id2:Ident t1:TypeName
		// c:IdentC
		{
			pos17 := pos
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail11
			}
			labels[4] = parser.text[pos17:pos]
		}
		// id2:Ident
		{
			pos18 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail11
			}
			labels[5] = parser.text[pos18:pos]
		}
		// t1:TypeName
		{
			pos19 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail11
			}
			labels[6] = parser.text[pos19:pos]
		}
		for {
			pos13 := pos
			// (c:IdentC id2:Ident t1:TypeName {…})
			// action
			// c:IdentC id2:Ident t1:TypeName
			// c:IdentC
			{
				pos21 := pos
				// IdentC
				if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
					goto fail15
				}
				labels[4] = parser.text[pos21:pos]
			}
			// id2:Ident
			{
				pos22 := pos
				// Ident
				if !_fail(parser, _IdentFail, errPos, failure, &pos) {
					goto fail15
				}
				labels[5] = parser.text[pos22:pos]
			}
			// t1:TypeName
			{
				pos23 := pos
				// TypeName
				if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
					goto fail15
				}
				labels[6] = parser.text[pos23:pos]
			}
			continue
		fail15:
			pos = pos13
			break
		}
		goto ok0
	fail11:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _ParmsAction(parser *_Parser, start int) (int, *[]parm) {
	var labels [7]string
	use(labels)
	var label0 Ident
	var label1 Ident
	var label2 Ident
	var label3 TypeName
	var label4 Ident
	var label5 Ident
	var label6 TypeName
	dp := parser.deltaPos[start][_Parms]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Parms}
	n := parser.act[key]
	if n != nil {
		n := n.([]parm)
		return start + int(dp-1), &n
	}
	var node []parm
	pos := start
	// id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+
	{
		pos3 := pos
		var node2 []parm
		// action
		{
			start5 := pos
			// id0:Ident
			{
				pos6 := pos
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail4
				} else {
					label0 = *n
					pos = p
				}
				labels[0] = parser.text[pos6:pos]
			}
			node = func(
				start, end int, id0 Ident) []parm {
				return []parm{{key: id0}}
			}(
				start5, pos, label0)
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// action
		{
			start8 := pos
			// o:Op id1:Ident t0:TypeName
			// o:Op
			{
				pos10 := pos
				// Op
				if p, n := _OpAction(parser, pos); n == nil {
					goto fail7
				} else {
					label1 = *n
					pos = p
				}
				labels[1] = parser.text[pos10:pos]
			}
			// id1:Ident
			{
				pos11 := pos
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail7
				} else {
					label2 = *n
					pos = p
				}
				labels[2] = parser.text[pos11:pos]
			}
			// t0:TypeName
			{
				pos12 := pos
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail7
				} else {
					label3 = *n
					pos = p
				}
				labels[3] = parser.text[pos12:pos]
			}
			node = func(
				start, end int, id0 Ident, id1 Ident, o Ident, t0 TypeName) []parm {
				return []parm{{key: o, name: id1, typ: t0}}
			}(
				start8, pos, label0, label2, label1, label3)
		}
		goto ok0
	fail7:
		node = node2
		pos = pos3
		// (c:IdentC id2:Ident t1:TypeName {…})+
		{
			var node16 parm
			// (c:IdentC id2:Ident t1:TypeName {…})
			// action
			{
				start18 := pos
				// c:IdentC id2:Ident t1:TypeName
				// c:IdentC
				{
					pos20 := pos
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail13
					} else {
						label4 = *n
						pos = p
					}
					labels[4] = parser.text[pos20:pos]
				}
				// id2:Ident
				{
					pos21 := pos
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail13
					} else {
						label5 = *n
						pos = p
					}
					labels[5] = parser.text[pos21:pos]
				}
				// t1:TypeName
				{
					pos22 := pos
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail13
					} else {
						label6 = *n
						pos = p
					}
					labels[6] = parser.text[pos22:pos]
				}
				node16 = func(
					start, end int, c Ident, id0 Ident, id1 Ident, id2 Ident, o Ident, t0 TypeName, t1 TypeName) parm {
					return parm{key: c, name: id2, typ: t1}
				}(
					start18, pos, label4, label0, label2, label5, label1, label3, label6)
			}
			node = append(node, node16)
		}
		for {
			pos15 := pos
			var node16 parm
			// (c:IdentC id2:Ident t1:TypeName {…})
			// action
			{
				start23 := pos
				// c:IdentC id2:Ident t1:TypeName
				// c:IdentC
				{
					pos25 := pos
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail17
					} else {
						label4 = *n
						pos = p
					}
					labels[4] = parser.text[pos25:pos]
				}
				// id2:Ident
				{
					pos26 := pos
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail17
					} else {
						label5 = *n
						pos = p
					}
					labels[5] = parser.text[pos26:pos]
				}
				// t1:TypeName
				{
					pos27 := pos
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail17
					} else {
						label6 = *n
						pos = p
					}
					labels[6] = parser.text[pos27:pos]
				}
				node16 = func(
					start, end int, c Ident, id0 Ident, id1 Ident, id2 Ident, o Ident, t0 TypeName, t1 TypeName) parm {
					return parm{key: c, name: id2, typ: t1}
				}(
					start23, pos, label4, label0, label2, label5, label1, label3, label6)
			}
			node = append(node, node16)
			continue
		fail17:
			pos = pos15
			break
		}
		goto ok0
	fail13:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _RetAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [1]string
	use(labels)
	if dp, de, ok := _memo(parser, _Ret, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ "^" t:TypeName
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// "^"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	// t:TypeName
	{
		pos1 := pos
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Ret, start, pos, perr)
fail:
	return _memoize(parser, _Ret, start, -1, perr)
}

func _RetFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [1]string
	use(labels)
	pos, failure := _failMemo(parser, _Ret, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Ret",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Ret}
	// action
	// _ "^" t:TypeName
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// "^"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\"^\"",
			})
		}
		goto fail
	}
	pos++
	// t:TypeName
	{
		pos1 := pos
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _RetAction(parser *_Parser, start int) (int, *TypeName) {
	var labels [1]string
	use(labels)
	var label0 TypeName
	dp := parser.deltaPos[start][_Ret]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ret}
	n := parser.act[key]
	if n != nil {
		n := n.(TypeName)
		return start + int(dp-1), &n
	}
	var node TypeName
	pos := start
	// action
	{
		start0 := pos
		// _ "^" t:TypeName
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// "^"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
			goto fail
		}
		pos++
		// t:TypeName
		{
			pos2 := pos
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, t TypeName) TypeName {
			return TypeName(t)
		}(
			start0, pos, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeSigAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _TypeSig, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// tps:TParms n:(Ident/Op)
	// tps:TParms
	{
		pos1 := pos
		// TParms
		if !_accept(parser, _TParmsAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:(Ident/Op)
	{
		pos2 := pos
		// (Ident/Op)
		// Ident/Op
		{
			pos6 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail7
			}
			goto ok3
		fail7:
			pos = pos6
			// Op
			if !_accept(parser, _OpAccepts, &pos, &perr) {
				goto fail8
			}
			goto ok3
		fail8:
			pos = pos6
			goto fail
		ok3:
		}
		labels[1] = parser.text[pos2:pos]
	}
	return _memoize(parser, _TypeSig, start, pos, perr)
fail:
	return _memoize(parser, _TypeSig, start, -1, perr)
}

func _TypeSigFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _TypeSig, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeSig",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeSig}
	// action
	// tps:TParms n:(Ident/Op)
	// tps:TParms
	{
		pos1 := pos
		// TParms
		if !_fail(parser, _TParmsFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:(Ident/Op)
	{
		pos2 := pos
		// (Ident/Op)
		// Ident/Op
		{
			pos6 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail7
			}
			goto ok3
		fail7:
			pos = pos6
			// Op
			if !_fail(parser, _OpFail, errPos, failure, &pos) {
				goto fail8
			}
			goto ok3
		fail8:
			pos = pos6
			goto fail
		ok3:
		}
		labels[1] = parser.text[pos2:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _TypeSigAction(parser *_Parser, start int) (int, *TypeSig) {
	var labels [2]string
	use(labels)
	var label0 ([]Var)
	var label1 Ident
	dp := parser.deltaPos[start][_TypeSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeSig}
	n := parser.act[key]
	if n != nil {
		n := n.(TypeSig)
		return start + int(dp-1), &n
	}
	var node TypeSig
	pos := start
	// action
	{
		start0 := pos
		// tps:TParms n:(Ident/Op)
		// tps:TParms
		{
			pos2 := pos
			// TParms
			if p, n := _TParmsAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// n:(Ident/Op)
		{
			pos3 := pos
			// (Ident/Op)
			// Ident/Op
			{
				pos7 := pos
				var node6 Ident
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail8
				} else {
					label1 = *n
					pos = p
				}
				goto ok4
			fail8:
				label1 = node6
				pos = pos7
				// Op
				if p, n := _OpAction(parser, pos); n == nil {
					goto fail9
				} else {
					label1 = *n
					pos = p
				}
				goto ok4
			fail9:
				label1 = node6
				pos = pos7
				goto fail
			ok4:
			}
			labels[1] = parser.text[pos3:pos]
		}
		node = func(
			start, end int, n Ident, tps []Var) TypeSig {
			l := loc(parser, start, end)
			if len(tps) > 0 {
				l.start = tps[0].start
			}
			return TypeSig{
				location: l,
				Name:     n.Text,
				Parms:    tps,
			}
		}(
			start0, pos, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TParmsAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [5]string
	use(labels)
	if dp, de, ok := _memo(parser, _TParms, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// tps:(n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})?
	{
		pos0 := pos
		// (n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})?
		{
			pos2 := pos
			// (n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})
			// n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…}
			{
				pos7 := pos
				// action
				// n:TypeVar
				{
					pos9 := pos
					// TypeVar
					if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
						goto fail8
					}
					labels[0] = parser.text[pos9:pos]
				}
				goto ok4
			fail8:
				pos = pos7
				// action
				// _ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")"
				// _
				if !_accept(parser, __Accepts, &pos, &perr) {
					goto fail10
				}
				// "("
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
					perr = _max(perr, pos)
					goto fail10
				}
				pos++
				// p0:TParm
				{
					pos12 := pos
					// TParm
					if !_accept(parser, _TParmAccepts, &pos, &perr) {
						goto fail10
					}
					labels[1] = parser.text[pos12:pos]
				}
				// ps:(_ "," p1:TParm {…})*
				{
					pos13 := pos
					// (_ "," p1:TParm {…})*
					for {
						pos15 := pos
						// (_ "," p1:TParm {…})
						// action
						// _ "," p1:TParm
						// _
						if !_accept(parser, __Accepts, &pos, &perr) {
							goto fail17
						}
						// ","
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
							perr = _max(perr, pos)
							goto fail17
						}
						pos++
						// p1:TParm
						{
							pos19 := pos
							// TParm
							if !_accept(parser, _TParmAccepts, &pos, &perr) {
								goto fail17
							}
							labels[2] = parser.text[pos19:pos]
						}
						continue
					fail17:
						pos = pos15
						break
					}
					labels[3] = parser.text[pos13:pos]
				}
				// (_ ",")?
				{
					pos21 := pos
					// (_ ",")
					// _ ","
					// _
					if !_accept(parser, __Accepts, &pos, &perr) {
						goto fail22
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						perr = _max(perr, pos)
						goto fail22
					}
					pos++
					goto ok24
				fail22:
					pos = pos21
				ok24:
				}
				// _
				if !_accept(parser, __Accepts, &pos, &perr) {
					goto fail10
				}
				// ")"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
					perr = _max(perr, pos)
					goto fail10
				}
				pos++
				goto ok4
			fail10:
				pos = pos7
				goto fail3
			ok4:
			}
			goto ok25
		fail3:
			pos = pos2
		ok25:
		}
		labels[4] = parser.text[pos0:pos]
	}
	return _memoize(parser, _TParms, start, pos, perr)
}

func _TParmsFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [5]string
	use(labels)
	pos, failure := _failMemo(parser, _TParms, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TParms",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TParms}
	// action
	// tps:(n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})?
	{
		pos0 := pos
		// (n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})?
		{
			pos2 := pos
			// (n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})
			// n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…}
			{
				pos7 := pos
				// action
				// n:TypeVar
				{
					pos9 := pos
					// TypeVar
					if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
						goto fail8
					}
					labels[0] = parser.text[pos9:pos]
				}
				goto ok4
			fail8:
				pos = pos7
				// action
				// _ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")"
				// _
				if !_fail(parser, __Fail, errPos, failure, &pos) {
					goto fail10
				}
				// "("
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"(\"",
						})
					}
					goto fail10
				}
				pos++
				// p0:TParm
				{
					pos12 := pos
					// TParm
					if !_fail(parser, _TParmFail, errPos, failure, &pos) {
						goto fail10
					}
					labels[1] = parser.text[pos12:pos]
				}
				// ps:(_ "," p1:TParm {…})*
				{
					pos13 := pos
					// (_ "," p1:TParm {…})*
					for {
						pos15 := pos
						// (_ "," p1:TParm {…})
						// action
						// _ "," p1:TParm
						// _
						if !_fail(parser, __Fail, errPos, failure, &pos) {
							goto fail17
						}
						// ","
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
							if pos >= errPos {
								failure.Kids = append(failure.Kids, &peg.Fail{
									Pos:  int(pos),
									Want: "\",\"",
								})
							}
							goto fail17
						}
						pos++
						// p1:TParm
						{
							pos19 := pos
							// TParm
							if !_fail(parser, _TParmFail, errPos, failure, &pos) {
								goto fail17
							}
							labels[2] = parser.text[pos19:pos]
						}
						continue
					fail17:
						pos = pos15
						break
					}
					labels[3] = parser.text[pos13:pos]
				}
				// (_ ",")?
				{
					pos21 := pos
					// (_ ",")
					// _ ","
					// _
					if !_fail(parser, __Fail, errPos, failure, &pos) {
						goto fail22
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "\",\"",
							})
						}
						goto fail22
					}
					pos++
					goto ok24
				fail22:
					pos = pos21
				ok24:
				}
				// _
				if !_fail(parser, __Fail, errPos, failure, &pos) {
					goto fail10
				}
				// ")"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\")\"",
						})
					}
					goto fail10
				}
				pos++
				goto ok4
			fail10:
				pos = pos7
				goto fail3
			ok4:
			}
			goto ok25
		fail3:
			pos = pos2
		ok25:
		}
		labels[4] = parser.text[pos0:pos]
	}
	parser.fail[key] = failure
	return pos, failure
}

func _TParmsAction(parser *_Parser, start int) (int, *([]Var)) {
	var labels [5]string
	use(labels)
	var label0 Ident
	var label1 Var
	var label2 Var
	var label3 []Var
	var label4 *[]Var
	dp := parser.deltaPos[start][_TParms]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TParms}
	n := parser.act[key]
	if n != nil {
		n := n.(([]Var))
		return start + int(dp-1), &n
	}
	var node ([]Var)
	pos := start
	// action
	{
		start0 := pos
		// tps:(n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})?
		{
			pos1 := pos
			// (n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})?
			{
				pos3 := pos
				label4 = new([]Var)
				// (n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…})
				// n:TypeVar {…}/_ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")" {…}
				{
					pos8 := pos
					var node7 []Var
					// action
					{
						start10 := pos
						// n:TypeVar
						{
							pos11 := pos
							// TypeVar
							if p, n := _TypeVarAction(parser, pos); n == nil {
								goto fail9
							} else {
								label0 = *n
								pos = p
							}
							labels[0] = parser.text[pos11:pos]
						}
						*label4 = func(
							start, end int, n Ident) []Var {
							return []Var{{location: n.location, Name: n.Text}}
						}(
							start10, pos, label0)
					}
					goto ok5
				fail9:
					*label4 = node7
					pos = pos8
					// action
					{
						start13 := pos
						// _ "(" p0:TParm ps:(_ "," p1:TParm {…})* (_ ",")? _ ")"
						// _
						if p, n := __Action(parser, pos); n == nil {
							goto fail12
						} else {
							pos = p
						}
						// "("
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
							goto fail12
						}
						pos++
						// p0:TParm
						{
							pos15 := pos
							// TParm
							if p, n := _TParmAction(parser, pos); n == nil {
								goto fail12
							} else {
								label1 = *n
								pos = p
							}
							labels[1] = parser.text[pos15:pos]
						}
						// ps:(_ "," p1:TParm {…})*
						{
							pos16 := pos
							// (_ "," p1:TParm {…})*
							for {
								pos18 := pos
								var node19 Var
								// (_ "," p1:TParm {…})
								// action
								{
									start21 := pos
									// _ "," p1:TParm
									// _
									if p, n := __Action(parser, pos); n == nil {
										goto fail20
									} else {
										pos = p
									}
									// ","
									if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
										goto fail20
									}
									pos++
									// p1:TParm
									{
										pos23 := pos
										// TParm
										if p, n := _TParmAction(parser, pos); n == nil {
											goto fail20
										} else {
											label2 = *n
											pos = p
										}
										labels[2] = parser.text[pos23:pos]
									}
									node19 = func(
										start, end int, n Ident, p0 Var, p1 Var) Var {
										return Var(p1)
									}(
										start21, pos, label0, label1, label2)
								}
								label3 = append(label3, node19)
								continue
							fail20:
								pos = pos18
								break
							}
							labels[3] = parser.text[pos16:pos]
						}
						// (_ ",")?
						{
							pos25 := pos
							// (_ ",")
							// _ ","
							// _
							if p, n := __Action(parser, pos); n == nil {
								goto fail26
							} else {
								pos = p
							}
							// ","
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
								goto fail26
							}
							pos++
							goto ok28
						fail26:
							pos = pos25
						ok28:
						}
						// _
						if p, n := __Action(parser, pos); n == nil {
							goto fail12
						} else {
							pos = p
						}
						// ")"
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
							goto fail12
						}
						pos++
						*label4 = func(
							start, end int, n Ident, p0 Var, p1 Var, ps []Var) []Var {
							return []Var(append([]Var{p0}, ps...))
						}(
							start13, pos, label0, label1, label2, label3)
					}
					goto ok5
				fail12:
					*label4 = node7
					pos = pos8
					goto fail4
				ok5:
				}
				goto ok29
			fail4:
				label4 = nil
				pos = pos3
			ok29:
			}
			labels[4] = parser.text[pos1:pos]
		}
		node = func(
			start, end int, n Ident, p0 Var, p1 Var, ps []Var, tps *[]Var) []Var {
			if tps == nil {
				return ([]Var)(nil)
			}
			return ([]Var)(*tps)
		}(
			start0, pos, label0, label1, label2, label3, label4)
	}
	parser.act[key] = node
	return pos, &node
}

func _TParmAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _TParm, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// n:TypeVar t1:TypeName?
	// n:TypeVar
	{
		pos1 := pos
		// TypeVar
		if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// t1:TypeName?
	{
		pos2 := pos
		// TypeName?
		{
			pos4 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail5
			}
			goto ok6
		fail5:
			pos = pos4
		ok6:
		}
		labels[1] = parser.text[pos2:pos]
	}
	return _memoize(parser, _TParm, start, pos, perr)
fail:
	return _memoize(parser, _TParm, start, -1, perr)
}

func _TParmFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _TParm, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TParm",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TParm}
	// action
	// n:TypeVar t1:TypeName?
	// n:TypeVar
	{
		pos1 := pos
		// TypeVar
		if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// t1:TypeName?
	{
		pos2 := pos
		// TypeName?
		{
			pos4 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail5
			}
			goto ok6
		fail5:
			pos = pos4
		ok6:
		}
		labels[1] = parser.text[pos2:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _TParmAction(parser *_Parser, start int) (int, *Var) {
	var labels [2]string
	use(labels)
	var label0 Ident
	var label1 *TypeName
	dp := parser.deltaPos[start][_TParm]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TParm}
	n := parser.act[key]
	if n != nil {
		n := n.(Var)
		return start + int(dp-1), &n
	}
	var node Var
	pos := start
	// action
	{
		start0 := pos
		// n:TypeVar t1:TypeName?
		// n:TypeVar
		{
			pos2 := pos
			// TypeVar
			if p, n := _TypeVarAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// t1:TypeName?
		{
			pos3 := pos
			// TypeName?
			{
				pos5 := pos
				label1 = new(TypeName)
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail6
				} else {
					*label1 = *n
					pos = p
				}
				goto ok7
			fail6:
				label1 = nil
				pos = pos5
			ok7:
			}
			labels[1] = parser.text[pos3:pos]
		}
		node = func(
			start, end int, n Ident, t1 *TypeName) Var {
			e := n.end
			if t1 != nil {
				e = t1.end
			}
			return Var{
				location: location{n.start, e},
				Name:     n.Text,
				Type:     t1,
			}
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeNameAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [7]string
	use(labels)
	if dp, de, ok := _memo(parser, _TypeName, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// tv1:TypeVar? ns0:TName+ {…}/tv2:TypeVar {…}/_ tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…}) {…}/_ "(" n2:TypeName _ ")" {…}
	{
		pos3 := pos
		// action
		// tv1:TypeVar? ns0:TName+
		// tv1:TypeVar?
		{
			pos6 := pos
			// TypeVar?
			{
				pos8 := pos
				// TypeVar
				if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
					goto fail9
				}
				goto ok10
			fail9:
				pos = pos8
			ok10:
			}
			labels[0] = parser.text[pos6:pos]
		}
		// ns0:TName+
		{
			pos11 := pos
			// TName+
			// TName
			if !_accept(parser, _TNameAccepts, &pos, &perr) {
				goto fail4
			}
			for {
				pos13 := pos
				// TName
				if !_accept(parser, _TNameAccepts, &pos, &perr) {
					goto fail15
				}
				continue
			fail15:
				pos = pos13
				break
			}
			labels[1] = parser.text[pos11:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// tv2:TypeVar
		{
			pos17 := pos
			// TypeVar
			if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
				goto fail16
			}
			labels[2] = parser.text[pos17:pos]
		}
		goto ok0
	fail16:
		pos = pos3
		// action
		// _ tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail18
		}
		// tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
		{
			pos20 := pos
			// ("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
			// action
			// "(" ns1:TypeNameList _ ")" ns2:TName+
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				perr = _max(perr, pos)
				goto fail18
			}
			pos++
			// ns1:TypeNameList
			{
				pos22 := pos
				// TypeNameList
				if !_accept(parser, _TypeNameListAccepts, &pos, &perr) {
					goto fail18
				}
				labels[3] = parser.text[pos22:pos]
			}
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail18
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				perr = _max(perr, pos)
				goto fail18
			}
			pos++
			// ns2:TName+
			{
				pos23 := pos
				// TName+
				// TName
				if !_accept(parser, _TNameAccepts, &pos, &perr) {
					goto fail18
				}
				for {
					pos25 := pos
					// TName
					if !_accept(parser, _TNameAccepts, &pos, &perr) {
						goto fail27
					}
					continue
				fail27:
					pos = pos25
					break
				}
				labels[4] = parser.text[pos23:pos]
			}
			labels[5] = parser.text[pos20:pos]
		}
		goto ok0
	fail18:
		pos = pos3
		// action
		// _ "(" n2:TypeName _ ")"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail28
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail28
		}
		pos++
		// n2:TypeName
		{
			pos30 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail28
			}
			labels[6] = parser.text[pos30:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail28
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			perr = _max(perr, pos)
			goto fail28
		}
		pos++
		goto ok0
	fail28:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _TypeName, start, pos, perr)
fail:
	return _memoize(parser, _TypeName, start, -1, perr)
}

func _TypeNameFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [7]string
	use(labels)
	pos, failure := _failMemo(parser, _TypeName, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeName",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeName}
	// tv1:TypeVar? ns0:TName+ {…}/tv2:TypeVar {…}/_ tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…}) {…}/_ "(" n2:TypeName _ ")" {…}
	{
		pos3 := pos
		// action
		// tv1:TypeVar? ns0:TName+
		// tv1:TypeVar?
		{
			pos6 := pos
			// TypeVar?
			{
				pos8 := pos
				// TypeVar
				if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
					goto fail9
				}
				goto ok10
			fail9:
				pos = pos8
			ok10:
			}
			labels[0] = parser.text[pos6:pos]
		}
		// ns0:TName+
		{
			pos11 := pos
			// TName+
			// TName
			if !_fail(parser, _TNameFail, errPos, failure, &pos) {
				goto fail4
			}
			for {
				pos13 := pos
				// TName
				if !_fail(parser, _TNameFail, errPos, failure, &pos) {
					goto fail15
				}
				continue
			fail15:
				pos = pos13
				break
			}
			labels[1] = parser.text[pos11:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// tv2:TypeVar
		{
			pos17 := pos
			// TypeVar
			if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
				goto fail16
			}
			labels[2] = parser.text[pos17:pos]
		}
		goto ok0
	fail16:
		pos = pos3
		// action
		// _ tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail18
		}
		// tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
		{
			pos20 := pos
			// ("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
			// action
			// "(" ns1:TypeNameList _ ")" ns2:TName+
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"(\"",
					})
				}
				goto fail18
			}
			pos++
			// ns1:TypeNameList
			{
				pos22 := pos
				// TypeNameList
				if !_fail(parser, _TypeNameListFail, errPos, failure, &pos) {
					goto fail18
				}
				labels[3] = parser.text[pos22:pos]
			}
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail18
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\")\"",
					})
				}
				goto fail18
			}
			pos++
			// ns2:TName+
			{
				pos23 := pos
				// TName+
				// TName
				if !_fail(parser, _TNameFail, errPos, failure, &pos) {
					goto fail18
				}
				for {
					pos25 := pos
					// TName
					if !_fail(parser, _TNameFail, errPos, failure, &pos) {
						goto fail27
					}
					continue
				fail27:
					pos = pos25
					break
				}
				labels[4] = parser.text[pos23:pos]
			}
			labels[5] = parser.text[pos20:pos]
		}
		goto ok0
	fail18:
		pos = pos3
		// action
		// _ "(" n2:TypeName _ ")"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail28
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail28
		}
		pos++
		// n2:TypeName
		{
			pos30 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail28
			}
			labels[6] = parser.text[pos30:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail28
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
				})
			}
			goto fail28
		}
		pos++
		goto ok0
	fail28:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _TypeNameAction(parser *_Parser, start int) (int, *TypeName) {
	var labels [7]string
	use(labels)
	var label0 *Ident
	var label1 []tname
	var label2 Ident
	var label3 []TypeName
	var label4 []tname
	var label5 TypeName
	var label6 TypeName
	dp := parser.deltaPos[start][_TypeName]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeName}
	n := parser.act[key]
	if n != nil {
		n := n.(TypeName)
		return start + int(dp-1), &n
	}
	var node TypeName
	pos := start
	// tv1:TypeVar? ns0:TName+ {…}/tv2:TypeVar {…}/_ tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…}) {…}/_ "(" n2:TypeName _ ")" {…}
	{
		pos3 := pos
		var node2 TypeName
		// action
		{
			start5 := pos
			// tv1:TypeVar? ns0:TName+
			// tv1:TypeVar?
			{
				pos7 := pos
				// TypeVar?
				{
					pos9 := pos
					label0 = new(Ident)
					// TypeVar
					if p, n := _TypeVarAction(parser, pos); n == nil {
						goto fail10
					} else {
						*label0 = *n
						pos = p
					}
					goto ok11
				fail10:
					label0 = nil
					pos = pos9
				ok11:
				}
				labels[0] = parser.text[pos7:pos]
			}
			// ns0:TName+
			{
				pos12 := pos
				// TName+
				{
					var node15 tname
					// TName
					if p, n := _TNameAction(parser, pos); n == nil {
						goto fail4
					} else {
						node15 = *n
						pos = p
					}
					label1 = append(label1, node15)
				}
				for {
					pos14 := pos
					var node15 tname
					// TName
					if p, n := _TNameAction(parser, pos); n == nil {
						goto fail16
					} else {
						node15 = *n
						pos = p
					}
					label1 = append(label1, node15)
					continue
				fail16:
					pos = pos14
					break
				}
				labels[1] = parser.text[pos12:pos]
			}
			node = func(
				start, end int, ns0 []tname, tv1 *Ident) TypeName {
				s := ns0[0].name.start
				var a []TypeName
				if tv1 != nil {
					s = tv1.start
					a = []TypeName{{location: tv1.location, Name: tv1.Text, Var: true}}
				}
				for _, n := range ns0[:len(ns0)-1] {
					a = []TypeName{{
						location: location{s, n.name.end},
						Mod:      n.mod,
						Name:     n.name.Text,
						Args:     a,
					}}
				}
				n := ns0[len(ns0)-1]
				return TypeName{
					location: location{s, n.name.end},
					Mod:      n.mod,
					Name:     n.name.Text,
					Args:     a,
				}
			}(
				start5, pos, label1, label0)
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// action
		{
			start18 := pos
			// tv2:TypeVar
			{
				pos19 := pos
				// TypeVar
				if p, n := _TypeVarAction(parser, pos); n == nil {
					goto fail17
				} else {
					label2 = *n
					pos = p
				}
				labels[2] = parser.text[pos19:pos]
			}
			node = func(
				start, end int, ns0 []tname, tv1 *Ident, tv2 Ident) TypeName {
				return TypeName{location: tv2.location, Name: tv2.Text, Var: true}
			}(
				start18, pos, label1, label0, label2)
		}
		goto ok0
	fail17:
		node = node2
		pos = pos3
		// action
		{
			start21 := pos
			// _ tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail20
			} else {
				pos = p
			}
			// tn0:("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
			{
				pos23 := pos
				// ("(" ns1:TypeNameList _ ")" ns2:TName+ {…})
				// action
				{
					start24 := pos
					// "(" ns1:TypeNameList _ ")" ns2:TName+
					// "("
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
						goto fail20
					}
					pos++
					// ns1:TypeNameList
					{
						pos26 := pos
						// TypeNameList
						if p, n := _TypeNameListAction(parser, pos); n == nil {
							goto fail20
						} else {
							label3 = *n
							pos = p
						}
						labels[3] = parser.text[pos26:pos]
					}
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail20
					} else {
						pos = p
					}
					// ")"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
						goto fail20
					}
					pos++
					// ns2:TName+
					{
						pos27 := pos
						// TName+
						{
							var node30 tname
							// TName
							if p, n := _TNameAction(parser, pos); n == nil {
								goto fail20
							} else {
								node30 = *n
								pos = p
							}
							label4 = append(label4, node30)
						}
						for {
							pos29 := pos
							var node30 tname
							// TName
							if p, n := _TNameAction(parser, pos); n == nil {
								goto fail31
							} else {
								node30 = *n
								pos = p
							}
							label4 = append(label4, node30)
							continue
						fail31:
							pos = pos29
							break
						}
						labels[4] = parser.text[pos27:pos]
					}
					label5 = func(
						start, end int, ns0 []tname, ns1 []TypeName, ns2 []tname, tv1 *Ident, tv2 Ident) TypeName {
						s := loc1(parser, start)
						for _, n := range ns2[:len(ns2)-1] {
							ns1 = []TypeName{{
								location: location{s, n.name.end},
								Mod:      n.mod,
								Name:     n.name.Text,
								Args:     ns1,
							}}
						}
						return TypeName{
							location: loc(parser, start, end),
							Mod:      ns2[len(ns2)-1].mod,
							Name:     ns2[len(ns2)-1].name.Text,
							Args:     ns1,
						}
					}(
						start24, pos, label1, label3, label4, label0, label2)
				}
				labels[5] = parser.text[pos23:pos]
			}
			node = func(
				start, end int, ns0 []tname, ns1 []TypeName, ns2 []tname, tn0 TypeName, tv1 *Ident, tv2 Ident) TypeName {
				return TypeName(tn0)
			}(
				start21, pos, label1, label3, label4, label5, label0, label2)
		}
		goto ok0
	fail20:
		node = node2
		pos = pos3
		// action
		{
			start33 := pos
			// _ "(" n2:TypeName _ ")"
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail32
			} else {
				pos = p
			}
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				goto fail32
			}
			pos++
			// n2:TypeName
			{
				pos35 := pos
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail32
				} else {
					label6 = *n
					pos = p
				}
				labels[6] = parser.text[pos35:pos]
			}
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail32
			} else {
				pos = p
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				goto fail32
			}
			pos++
			node = func(
				start, end int, n2 TypeName, ns0 []tname, ns1 []TypeName, ns2 []tname, tn0 TypeName, tv1 *Ident, tv2 Ident) TypeName {
				return TypeName(n2)
			}(
				start33, pos, label6, label1, label3, label4, label5, label0, label2)
		}
		goto ok0
	fail32:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeNameListAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _TypeNameList, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")?
	// n0:TypeName
	{
		pos1 := pos
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// ns:(_ "," n1:TypeName {…})*
	{
		pos2 := pos
		// (_ "," n1:TypeName {…})*
		for {
			pos4 := pos
			// (_ "," n1:TypeName {…})
			// action
			// _ "," n1:TypeName
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail6
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				perr = _max(perr, pos)
				goto fail6
			}
			pos++
			// n1:TypeName
			{
				pos8 := pos
				// TypeName
				if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
					goto fail6
				}
				labels[1] = parser.text[pos8:pos]
			}
			continue
		fail6:
			pos = pos4
			break
		}
		labels[2] = parser.text[pos2:pos]
	}
	// (_ ",")?
	{
		pos10 := pos
		// (_ ",")
		// _ ","
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail11
		}
		// ","
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
			perr = _max(perr, pos)
			goto fail11
		}
		pos++
		goto ok13
	fail11:
		pos = pos10
	ok13:
	}
	return _memoize(parser, _TypeNameList, start, pos, perr)
fail:
	return _memoize(parser, _TypeNameList, start, -1, perr)
}

func _TypeNameListFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _TypeNameList, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeNameList",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeNameList}
	// action
	// n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")?
	// n0:TypeName
	{
		pos1 := pos
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// ns:(_ "," n1:TypeName {…})*
	{
		pos2 := pos
		// (_ "," n1:TypeName {…})*
		for {
			pos4 := pos
			// (_ "," n1:TypeName {…})
			// action
			// _ "," n1:TypeName
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail6
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\",\"",
					})
				}
				goto fail6
			}
			pos++
			// n1:TypeName
			{
				pos8 := pos
				// TypeName
				if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
					goto fail6
				}
				labels[1] = parser.text[pos8:pos]
			}
			continue
		fail6:
			pos = pos4
			break
		}
		labels[2] = parser.text[pos2:pos]
	}
	// (_ ",")?
	{
		pos10 := pos
		// (_ ",")
		// _ ","
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail11
		}
		// ","
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\",\"",
				})
			}
			goto fail11
		}
		pos++
		goto ok13
	fail11:
		pos = pos10
	ok13:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _TypeNameListAction(parser *_Parser, start int) (int, *[]TypeName) {
	var labels [3]string
	use(labels)
	var label0 TypeName
	var label1 TypeName
	var label2 []TypeName
	dp := parser.deltaPos[start][_TypeNameList]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeNameList}
	n := parser.act[key]
	if n != nil {
		n := n.([]TypeName)
		return start + int(dp-1), &n
	}
	var node []TypeName
	pos := start
	// action
	{
		start0 := pos
		// n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")?
		// n0:TypeName
		{
			pos2 := pos
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// ns:(_ "," n1:TypeName {…})*
		{
			pos3 := pos
			// (_ "," n1:TypeName {…})*
			for {
				pos5 := pos
				var node6 TypeName
				// (_ "," n1:TypeName {…})
				// action
				{
					start8 := pos
					// _ "," n1:TypeName
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail7
					} else {
						pos = p
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail7
					}
					pos++
					// n1:TypeName
					{
						pos10 := pos
						// TypeName
						if p, n := _TypeNameAction(parser, pos); n == nil {
							goto fail7
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos10:pos]
					}
					node6 = func(
						start, end int, n0 TypeName, n1 TypeName) TypeName {
						return TypeName(n1)
					}(
						start8, pos, label0, label1)
				}
				label2 = append(label2, node6)
				continue
			fail7:
				pos = pos5
				break
			}
			labels[2] = parser.text[pos3:pos]
		}
		// (_ ",")?
		{
			pos12 := pos
			// (_ ",")
			// _ ","
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail13
			} else {
				pos = p
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				goto fail13
			}
			pos++
			goto ok15
		fail13:
			pos = pos12
		ok15:
		}
		node = func(
			start, end int, n0 TypeName, n1 TypeName, ns []TypeName) []TypeName {
			return []TypeName(append([]TypeName{n0}, ns...))
		}(
			start0, pos, label0, label1, label2)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TNameAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _TName, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// mod:ModName? n:(TypeOp/Ident)
	// mod:ModName?
	{
		pos1 := pos
		// ModName?
		{
			pos3 := pos
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:(TypeOp/Ident)
	{
		pos6 := pos
		// (TypeOp/Ident)
		// TypeOp/Ident
		{
			pos10 := pos
			// TypeOp
			if !_accept(parser, _TypeOpAccepts, &pos, &perr) {
				goto fail11
			}
			goto ok7
		fail11:
			pos = pos10
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail12
			}
			goto ok7
		fail12:
			pos = pos10
			goto fail
		ok7:
		}
		labels[1] = parser.text[pos6:pos]
	}
	return _memoize(parser, _TName, start, pos, perr)
fail:
	return _memoize(parser, _TName, start, -1, perr)
}

func _TNameFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _TName, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TName",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TName}
	// action
	// mod:ModName? n:(TypeOp/Ident)
	// mod:ModName?
	{
		pos1 := pos
		// ModName?
		{
			pos3 := pos
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:(TypeOp/Ident)
	{
		pos6 := pos
		// (TypeOp/Ident)
		// TypeOp/Ident
		{
			pos10 := pos
			// TypeOp
			if !_fail(parser, _TypeOpFail, errPos, failure, &pos) {
				goto fail11
			}
			goto ok7
		fail11:
			pos = pos10
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail12
			}
			goto ok7
		fail12:
			pos = pos10
			goto fail
		ok7:
		}
		labels[1] = parser.text[pos6:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _TNameAction(parser *_Parser, start int) (int, *tname) {
	var labels [2]string
	use(labels)
	var label0 *Ident
	var label1 Ident
	dp := parser.deltaPos[start][_TName]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TName}
	n := parser.act[key]
	if n != nil {
		n := n.(tname)
		return start + int(dp-1), &n
	}
	var node tname
	pos := start
	// action
	{
		start0 := pos
		// mod:ModName? n:(TypeOp/Ident)
		// mod:ModName?
		{
			pos2 := pos
			// ModName?
			{
				pos4 := pos
				label0 = new(Ident)
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail5
				} else {
					*label0 = *n
					pos = p
				}
				goto ok6
			fail5:
				label0 = nil
				pos = pos4
			ok6:
			}
			labels[0] = parser.text[pos2:pos]
		}
		// n:(TypeOp/Ident)
		{
			pos7 := pos
			// (TypeOp/Ident)
			// TypeOp/Ident
			{
				pos11 := pos
				var node10 Ident
				// TypeOp
				if p, n := _TypeOpAction(parser, pos); n == nil {
					goto fail12
				} else {
					label1 = *n
					pos = p
				}
				goto ok8
			fail12:
				label1 = node10
				pos = pos11
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail13
				} else {
					label1 = *n
					pos = p
				}
				goto ok8
			fail13:
				label1 = node10
				pos = pos11
				goto fail
			ok8:
			}
			labels[1] = parser.text[pos7:pos]
		}
		node = func(
			start, end int, mod *Ident, n Ident) tname {
			return tname{mod: mod, name: n}
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Type, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ def:(key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// def:(key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
	{
		pos1 := pos
		// (key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
		// action
		// key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt)
		// key:("type"/"Type")
		{
			pos3 := pos
			// ("type"/"Type")
			// "type"/"Type"
			{
				pos7 := pos
				// "type"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "type" {
					perr = _max(perr, pos)
					goto fail8
				}
				pos += 4
				goto ok4
			fail8:
				pos = pos7
				// "Type"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Type" {
					perr = _max(perr, pos)
					goto fail9
				}
				pos += 4
				goto ok4
			fail9:
				pos = pos7
				goto fail
			ok4:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// sig:TypeSig
		{
			pos10 := pos
			// TypeSig
			if !_accept(parser, _TypeSigAccepts, &pos, &perr) {
				goto fail
			}
			labels[1] = parser.text[pos10:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// typ:(Alias/And/Or/Virt)
		{
			pos11 := pos
			// (Alias/And/Or/Virt)
			// Alias/And/Or/Virt
			{
				pos15 := pos
				// Alias
				if !_accept(parser, _AliasAccepts, &pos, &perr) {
					goto fail16
				}
				goto ok12
			fail16:
				pos = pos15
				// And
				if !_accept(parser, _AndAccepts, &pos, &perr) {
					goto fail17
				}
				goto ok12
			fail17:
				pos = pos15
				// Or
				if !_accept(parser, _OrAccepts, &pos, &perr) {
					goto fail18
				}
				goto ok12
			fail18:
				pos = pos15
				// Virt
				if !_accept(parser, _VirtAccepts, &pos, &perr) {
					goto fail19
				}
				goto ok12
			fail19:
				pos = pos15
				goto fail
			ok12:
			}
			labels[2] = parser.text[pos11:pos]
		}
		labels[3] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Type, start, pos, perr)
fail:
	return _memoize(parser, _Type, start, -1, perr)
}

func _TypeFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
	use(labels)
	pos, failure := _failMemo(parser, _Type, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Type",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Type}
	// action
	// _ def:(key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// def:(key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
	{
		pos1 := pos
		// (key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
		// action
		// key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt)
		// key:("type"/"Type")
		{
			pos3 := pos
			// ("type"/"Type")
			// "type"/"Type"
			{
				pos7 := pos
				// "type"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "type" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"type\"",
						})
					}
					goto fail8
				}
				pos += 4
				goto ok4
			fail8:
				pos = pos7
				// "Type"
				if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Type" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"Type\"",
						})
					}
					goto fail9
				}
				pos += 4
				goto ok4
			fail9:
				pos = pos7
				goto fail
			ok4:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// sig:TypeSig
		{
			pos10 := pos
			// TypeSig
			if !_fail(parser, _TypeSigFail, errPos, failure, &pos) {
				goto fail
			}
			labels[1] = parser.text[pos10:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// typ:(Alias/And/Or/Virt)
		{
			pos11 := pos
			// (Alias/And/Or/Virt)
			// Alias/And/Or/Virt
			{
				pos15 := pos
				// Alias
				if !_fail(parser, _AliasFail, errPos, failure, &pos) {
					goto fail16
				}
				goto ok12
			fail16:
				pos = pos15
				// And
				if !_fail(parser, _AndFail, errPos, failure, &pos) {
					goto fail17
				}
				goto ok12
			fail17:
				pos = pos15
				// Or
				if !_fail(parser, _OrFail, errPos, failure, &pos) {
					goto fail18
				}
				goto ok12
			fail18:
				pos = pos15
				// Virt
				if !_fail(parser, _VirtFail, errPos, failure, &pos) {
					goto fail19
				}
				goto ok12
			fail19:
				pos = pos15
				goto fail
			ok12:
			}
			labels[2] = parser.text[pos11:pos]
		}
		labels[3] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _TypeAction(parser *_Parser, start int) (int, *Def) {
	var labels [4]string
	use(labels)
	var label0 string
	var label1 TypeSig
	var label2 Type
	var label3 Def
	dp := parser.deltaPos[start][_Type]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Type}
	n := parser.act[key]
	if n != nil {
		n := n.(Def)
		return start + int(dp-1), &n
	}
	var node Def
	pos := start
	// action
	{
		start0 := pos
		// _ def:(key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// def:(key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
		{
			pos2 := pos
			// (key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt) {…})
			// action
			{
				start3 := pos
				// key:("type"/"Type") sig:TypeSig _ typ:(Alias/And/Or/Virt)
				// key:("type"/"Type")
				{
					pos5 := pos
					// ("type"/"Type")
					// "type"/"Type"
					{
						pos9 := pos
						var node8 string
						// "type"
						if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "type" {
							goto fail10
						}
						label0 = parser.text[pos : pos+4]
						pos += 4
						goto ok6
					fail10:
						label0 = node8
						pos = pos9
						// "Type"
						if len(parser.text[pos:]) < 4 || parser.text[pos:pos+4] != "Type" {
							goto fail11
						}
						label0 = parser.text[pos : pos+4]
						pos += 4
						goto ok6
					fail11:
						label0 = node8
						pos = pos9
						goto fail
					ok6:
					}
					labels[0] = parser.text[pos5:pos]
				}
				// sig:TypeSig
				{
					pos12 := pos
					// TypeSig
					if p, n := _TypeSigAction(parser, pos); n == nil {
						goto fail
					} else {
						label1 = *n
						pos = p
					}
					labels[1] = parser.text[pos12:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// typ:(Alias/And/Or/Virt)
				{
					pos13 := pos
					// (Alias/And/Or/Virt)
					// Alias/And/Or/Virt
					{
						pos17 := pos
						var node16 Type
						// Alias
						if p, n := _AliasAction(parser, pos); n == nil {
							goto fail18
						} else {
							label2 = *n
							pos = p
						}
						goto ok14
					fail18:
						label2 = node16
						pos = pos17
						// And
						if p, n := _AndAction(parser, pos); n == nil {
							goto fail19
						} else {
							label2 = *n
							pos = p
						}
						goto ok14
					fail19:
						label2 = node16
						pos = pos17
						// Or
						if p, n := _OrAction(parser, pos); n == nil {
							goto fail20
						} else {
							label2 = *n
							pos = p
						}
						goto ok14
					fail20:
						label2 = node16
						pos = pos17
						// Virt
						if p, n := _VirtAction(parser, pos); n == nil {
							goto fail21
						} else {
							label2 = *n
							pos = p
						}
						goto ok14
					fail21:
						label2 = node16
						pos = pos17
						goto fail
					ok14:
					}
					labels[2] = parser.text[pos13:pos]
				}
				label3 = func(
					start, end int, key string, sig TypeSig, typ Type) Def {
					typ.location = loc(parser, start, end)
					typ.priv = key == "type"
					typ.Sig = sig
					return Def(&typ)
				}(
					start3, pos, label0, label1, label2)
			}
			labels[3] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, def Def, key string, sig TypeSig, typ Type) Def {
			return Def(def)
		}(
			start0, pos, label3, label0, label1, label2)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _AliasAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [1]string
	use(labels)
	if dp, de, ok := _memo(parser, _Alias, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ ":=" n:TypeName _ "."
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// ":="
	if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
		perr = _max(perr, pos)
		goto fail
	}
	pos += 2
	// n:TypeName
	{
		pos1 := pos
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// "."
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	return _memoize(parser, _Alias, start, pos, perr)
fail:
	return _memoize(parser, _Alias, start, -1, perr)
}

func _AliasFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [1]string
	use(labels)
	pos, failure := _failMemo(parser, _Alias, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Alias",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Alias}
	// action
	// _ ":=" n:TypeName _ "."
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// ":="
	if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\":=\"",
			})
		}
		goto fail
	}
	pos += 2
	// n:TypeName
	{
		pos1 := pos
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// "."
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\".\"",
			})
		}
		goto fail
	}
	pos++
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _AliasAction(parser *_Parser, start int) (int, *Type) {
	var labels [1]string
	use(labels)
	var label0 TypeName
	dp := parser.deltaPos[start][_Alias]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Alias}
	n := parser.act[key]
	if n != nil {
		n := n.(Type)
		return start + int(dp-1), &n
	}
	var node Type
	pos := start
	// action
	{
		start0 := pos
		// _ ":=" n:TypeName _ "."
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// ":="
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
			goto fail
		}
		pos += 2
		// n:TypeName
		{
			pos2 := pos
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// "."
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
			goto fail
		}
		pos++
		node = func(
			start, end int, n TypeName) Type {
			return Type{Alias: &n}
		}(
			start0, pos, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _AndAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [1]string
	use(labels)
	if dp, de, ok := _memo(parser, _And, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ "{" fs:Field* _ "}"
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	// fs:Field*
	{
		pos1 := pos
		// Field*
		for {
			pos3 := pos
			// Field
			if !_accept(parser, _FieldAccepts, &pos, &perr) {
				goto fail5
			}
			continue
		fail5:
			pos = pos3
			break
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// "}"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	return _memoize(parser, _And, start, pos, perr)
fail:
	return _memoize(parser, _And, start, -1, perr)
}

func _AndFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [1]string
	use(labels)
	pos, failure := _failMemo(parser, _And, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "And",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _And}
	// action
	// _ "{" fs:Field* _ "}"
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\"{\"",
			})
		}
		goto fail
	}
	pos++
	// fs:Field*
	{
		pos1 := pos
		// Field*
		for {
			pos3 := pos
			// Field
			if !_fail(parser, _FieldFail, errPos, failure, &pos) {
				goto fail5
			}
			continue
		fail5:
			pos = pos3
			break
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// "}"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\"}\"",
			})
		}
		goto fail
	}
	pos++
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _AndAction(parser *_Parser, start int) (int, *Type) {
	var labels [1]string
	use(labels)
	var label0 []Var
	dp := parser.deltaPos[start][_And]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _And}
	n := parser.act[key]
	if n != nil {
		n := n.(Type)
		return start + int(dp-1), &n
	}
	var node Type
	pos := start
	// action
	{
		start0 := pos
		// _ "{" fs:Field* _ "}"
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			goto fail
		}
		pos++
		// fs:Field*
		{
			pos2 := pos
			// Field*
			for {
				pos4 := pos
				var node5 Var
				// Field
				if p, n := _FieldAction(parser, pos); n == nil {
					goto fail6
				} else {
					node5 = *n
					pos = p
				}
				label0 = append(label0, node5)
				continue
			fail6:
				pos = pos4
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// "}"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
			goto fail
		}
		pos++
		node = func(
			start, end int, fs []Var) Type {
			return Type{Fields: fs}
		}(
			start0, pos, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FieldAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Field, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// n:IdentC t:TypeName
	// n:IdentC
	{
		pos1 := pos
		// IdentC
		if !_accept(parser, _IdentCAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// t:TypeName
	{
		pos2 := pos
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail
		}
		labels[1] = parser.text[pos2:pos]
	}
	return _memoize(parser, _Field, start, pos, perr)
fail:
	return _memoize(parser, _Field, start, -1, perr)
}

func _FieldFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Field, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Field",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Field}
	// action
	// n:IdentC t:TypeName
	// n:IdentC
	{
		pos1 := pos
		// IdentC
		if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// t:TypeName
	{
		pos2 := pos
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail
		}
		labels[1] = parser.text[pos2:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _FieldAction(parser *_Parser, start int) (int, *Var) {
	var labels [2]string
	use(labels)
	var label0 Ident
	var label1 TypeName
	dp := parser.deltaPos[start][_Field]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Field}
	n := parser.act[key]
	if n != nil {
		n := n.(Var)
		return start + int(dp-1), &n
	}
	var node Var
	pos := start
	// action
	{
		start0 := pos
		// n:IdentC t:TypeName
		// n:IdentC
		{
			pos2 := pos
			// IdentC
			if p, n := _IdentCAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// t:TypeName
		{
			pos3 := pos
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail
			} else {
				label1 = *n
				pos = p
			}
			labels[1] = parser.text[pos3:pos]
		}
		node = func(
			start, end int, n Ident, t TypeName) Var {
			return Var{
				location: n.location,
				Name:     strings.TrimSuffix(n.Text, ":"),
				Type:     &t,
			}
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _OrAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Or, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ "{" (_ "|")? c:Case cs:(_ "|" c1:Case {…})* _ "}"
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	// (_ "|")?
	{
		pos2 := pos
		// (_ "|")
		// _ "|"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail3
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			perr = _max(perr, pos)
			goto fail3
		}
		pos++
		goto ok5
	fail3:
		pos = pos2
	ok5:
	}
	// c:Case
	{
		pos6 := pos
		// Case
		if !_accept(parser, _CaseAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos6:pos]
	}
	// cs:(_ "|" c1:Case {…})*
	{
		pos7 := pos
		// (_ "|" c1:Case {…})*
		for {
			pos9 := pos
			// (_ "|" c1:Case {…})
			// action
			// _ "|" c1:Case
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail11
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				perr = _max(perr, pos)
				goto fail11
			}
			pos++
			// c1:Case
			{
				pos13 := pos
				// Case
				if !_accept(parser, _CaseAccepts, &pos, &perr) {
					goto fail11
				}
				labels[1] = parser.text[pos13:pos]
			}
			continue
		fail11:
			pos = pos9
			break
		}
		labels[2] = parser.text[pos7:pos]
	}
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// "}"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	return _memoize(parser, _Or, start, pos, perr)
fail:
	return _memoize(parser, _Or, start, -1, perr)
}

func _OrFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Or, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Or",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Or}
	// action
	// _ "{" (_ "|")? c:Case cs:(_ "|" c1:Case {…})* _ "}"
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\"{\"",
			})
		}
		goto fail
	}
	pos++
	// (_ "|")?
	{
		pos2 := pos
		// (_ "|")
		// _ "|"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail3
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"|\"",
				})
			}
			goto fail3
		}
		pos++
		goto ok5
	fail3:
		pos = pos2
	ok5:
	}
	// c:Case
	{
		pos6 := pos
		// Case
		if !_fail(parser, _CaseFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos6:pos]
	}
	// cs:(_ "|" c1:Case {…})*
	{
		pos7 := pos
		// (_ "|" c1:Case {…})*
		for {
			pos9 := pos
			// (_ "|" c1:Case {…})
			// action
			// _ "|" c1:Case
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail11
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"|\"",
					})
				}
				goto fail11
			}
			pos++
			// c1:Case
			{
				pos13 := pos
				// Case
				if !_fail(parser, _CaseFail, errPos, failure, &pos) {
					goto fail11
				}
				labels[1] = parser.text[pos13:pos]
			}
			continue
		fail11:
			pos = pos9
			break
		}
		labels[2] = parser.text[pos7:pos]
	}
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// "}"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\"}\"",
			})
		}
		goto fail
	}
	pos++
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _OrAction(parser *_Parser, start int) (int, *Type) {
	var labels [3]string
	use(labels)
	var label0 Var
	var label1 Var
	var label2 []Var
	dp := parser.deltaPos[start][_Or]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Or}
	n := parser.act[key]
	if n != nil {
		n := n.(Type)
		return start + int(dp-1), &n
	}
	var node Type
	pos := start
	// action
	{
		start0 := pos
		// _ "{" (_ "|")? c:Case cs:(_ "|" c1:Case {…})* _ "}"
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			goto fail
		}
		pos++
		// (_ "|")?
		{
			pos3 := pos
			// (_ "|")
			// _ "|"
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail4
			} else {
				pos = p
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				goto fail4
			}
			pos++
			goto ok6
		fail4:
			pos = pos3
		ok6:
		}
		// c:Case
		{
			pos7 := pos
			// Case
			if p, n := _CaseAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos7:pos]
		}
		// cs:(_ "|" c1:Case {…})*
		{
			pos8 := pos
			// (_ "|" c1:Case {…})*
			for {
				pos10 := pos
				var node11 Var
				// (_ "|" c1:Case {…})
				// action
				{
					start13 := pos
					// _ "|" c1:Case
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail12
					} else {
						pos = p
					}
					// "|"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
						goto fail12
					}
					pos++
					// c1:Case
					{
						pos15 := pos
						// Case
						if p, n := _CaseAction(parser, pos); n == nil {
							goto fail12
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos15:pos]
					}
					node11 = func(
						start, end int, c Var, c1 Var) Var {
						return Var(c1)
					}(
						start13, pos, label0, label1)
				}
				label2 = append(label2, node11)
				continue
			fail12:
				pos = pos10
				break
			}
			labels[2] = parser.text[pos8:pos]
		}
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// "}"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
			goto fail
		}
		pos++
		node = func(
			start, end int, c Var, c1 Var, cs []Var) Type {
			return Type{Cases: append([]Var{c}, cs...)}
		}(
			start0, pos, label0, label1, label2)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _CaseAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Case, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// id0:Ident {…}/id1:IdentC t:TypeName {…}
	{
		pos3 := pos
		// action
		// id0:Ident
		{
			pos5 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// id1:IdentC t:TypeName
		// id1:IdentC
		{
			pos8 := pos
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail6
			}
			labels[1] = parser.text[pos8:pos]
		}
		// t:TypeName
		{
			pos9 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail6
			}
			labels[2] = parser.text[pos9:pos]
		}
		goto ok0
	fail6:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Case, start, pos, perr)
fail:
	return _memoize(parser, _Case, start, -1, perr)
}

func _CaseFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Case, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Case",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Case}
	// id0:Ident {…}/id1:IdentC t:TypeName {…}
	{
		pos3 := pos
		// action
		// id0:Ident
		{
			pos5 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// id1:IdentC t:TypeName
		// id1:IdentC
		{
			pos8 := pos
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail6
			}
			labels[1] = parser.text[pos8:pos]
		}
		// t:TypeName
		{
			pos9 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail6
			}
			labels[2] = parser.text[pos9:pos]
		}
		goto ok0
	fail6:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _CaseAction(parser *_Parser, start int) (int, *Var) {
	var labels [3]string
	use(labels)
	var label0 Ident
	var label1 Ident
	var label2 TypeName
	dp := parser.deltaPos[start][_Case]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Case}
	n := parser.act[key]
	if n != nil {
		n := n.(Var)
		return start + int(dp-1), &n
	}
	var node Var
	pos := start
	// id0:Ident {…}/id1:IdentC t:TypeName {…}
	{
		pos3 := pos
		var node2 Var
		// action
		{
			start5 := pos
			// id0:Ident
			{
				pos6 := pos
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail4
				} else {
					label0 = *n
					pos = p
				}
				labels[0] = parser.text[pos6:pos]
			}
			node = func(
				start, end int, id0 Ident) Var {
				return Var{
					location: id0.location,
					Name:     id0.Text,
				}
			}(
				start5, pos, label0)
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// action
		{
			start8 := pos
			// id1:IdentC t:TypeName
			// id1:IdentC
			{
				pos10 := pos
				// IdentC
				if p, n := _IdentCAction(parser, pos); n == nil {
					goto fail7
				} else {
					label1 = *n
					pos = p
				}
				labels[1] = parser.text[pos10:pos]
			}
			// t:TypeName
			{
				pos11 := pos
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail7
				} else {
					label2 = *n
					pos = p
				}
				labels[2] = parser.text[pos11:pos]
			}
			node = func(
				start, end int, id0 Ident, id1 Ident, t TypeName) Var {
				return Var{
					location: id1.location,
					Name:     id1.Text,
					Type:     &t,
				}
			}(
				start8, pos, label0, label1, label2)
		}
		goto ok0
	fail7:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _VirtAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [1]string
	use(labels)
	if dp, de, ok := _memo(parser, _Virt, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ "{" vs:MethSig+ _ "}"
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	// vs:MethSig+
	{
		pos1 := pos
		// MethSig+
		// MethSig
		if !_accept(parser, _MethSigAccepts, &pos, &perr) {
			goto fail
		}
		for {
			pos3 := pos
			// MethSig
			if !_accept(parser, _MethSigAccepts, &pos, &perr) {
				goto fail5
			}
			continue
		fail5:
			pos = pos3
			break
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// "}"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	return _memoize(parser, _Virt, start, pos, perr)
fail:
	return _memoize(parser, _Virt, start, -1, perr)
}

func _VirtFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [1]string
	use(labels)
	pos, failure := _failMemo(parser, _Virt, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Virt",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Virt}
	// action
	// _ "{" vs:MethSig+ _ "}"
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\"{\"",
			})
		}
		goto fail
	}
	pos++
	// vs:MethSig+
	{
		pos1 := pos
		// MethSig+
		// MethSig
		if !_fail(parser, _MethSigFail, errPos, failure, &pos) {
			goto fail
		}
		for {
			pos3 := pos
			// MethSig
			if !_fail(parser, _MethSigFail, errPos, failure, &pos) {
				goto fail5
			}
			continue
		fail5:
			pos = pos3
			break
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// "}"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\"}\"",
			})
		}
		goto fail
	}
	pos++
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _VirtAction(parser *_Parser, start int) (int, *Type) {
	var labels [1]string
	use(labels)
	var label0 []FunSig
	dp := parser.deltaPos[start][_Virt]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Virt}
	n := parser.act[key]
	if n != nil {
		n := n.(Type)
		return start + int(dp-1), &n
	}
	var node Type
	pos := start
	// action
	{
		start0 := pos
		// _ "{" vs:MethSig+ _ "}"
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			goto fail
		}
		pos++
		// vs:MethSig+
		{
			pos2 := pos
			// MethSig+
			{
				var node5 FunSig
				// MethSig
				if p, n := _MethSigAction(parser, pos); n == nil {
					goto fail
				} else {
					node5 = *n
					pos = p
				}
				label0 = append(label0, node5)
			}
			for {
				pos4 := pos
				var node5 FunSig
				// MethSig
				if p, n := _MethSigAction(parser, pos); n == nil {
					goto fail6
				} else {
					node5 = *n
					pos = p
				}
				label0 = append(label0, node5)
				continue
			fail6:
				pos = pos4
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// "}"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
			goto fail
		}
		pos++
		node = func(
			start, end int, vs []FunSig) Type {
			return Type{Virts: vs}
		}(
			start0, pos, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _MethSigAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [8]string
	use(labels)
	if dp, de, ok := _memo(parser, _MethSig, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ sig:("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// sig:("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
	{
		pos1 := pos
		// ("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
		// action
		// "[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]"
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+)
		{
			pos3 := pos
			// (id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+)
			// id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+
			{
				pos7 := pos
				// action
				// id0:Ident
				{
					pos9 := pos
					// Ident
					if !_accept(parser, _IdentAccepts, &pos, &perr) {
						goto fail8
					}
					labels[0] = parser.text[pos9:pos]
				}
				goto ok4
			fail8:
				pos = pos7
				// action
				// op:Op t0:TypeName
				// op:Op
				{
					pos12 := pos
					// Op
					if !_accept(parser, _OpAccepts, &pos, &perr) {
						goto fail10
					}
					labels[1] = parser.text[pos12:pos]
				}
				// t0:TypeName
				{
					pos13 := pos
					// TypeName
					if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
						goto fail10
					}
					labels[2] = parser.text[pos13:pos]
				}
				goto ok4
			fail10:
				pos = pos7
				// (id1:IdentC t1:TypeName {…})+
				// (id1:IdentC t1:TypeName {…})
				// action
				// id1:IdentC t1:TypeName
				// id1:IdentC
				{
					pos20 := pos
					// IdentC
					if !_accept(parser, _IdentCAccepts, &pos, &perr) {
						goto fail14
					}
					labels[3] = parser.text[pos20:pos]
				}
				// t1:TypeName
				{
					pos21 := pos
					// TypeName
					if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
						goto fail14
					}
					labels[4] = parser.text[pos21:pos]
				}
				for {
					pos16 := pos
					// (id1:IdentC t1:TypeName {…})
					// action
					// id1:IdentC t1:TypeName
					// id1:IdentC
					{
						pos23 := pos
						// IdentC
						if !_accept(parser, _IdentCAccepts, &pos, &perr) {
							goto fail18
						}
						labels[3] = parser.text[pos23:pos]
					}
					// t1:TypeName
					{
						pos24 := pos
						// TypeName
						if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
							goto fail18
						}
						labels[4] = parser.text[pos24:pos]
					}
					continue
				fail18:
					pos = pos16
					break
				}
				goto ok4
			fail14:
				pos = pos7
				goto fail
			ok4:
			}
			labels[5] = parser.text[pos3:pos]
		}
		// r:Ret?
		{
			pos25 := pos
			// Ret?
			{
				pos27 := pos
				// Ret
				if !_accept(parser, _RetAccepts, &pos, &perr) {
					goto fail28
				}
				goto ok29
			fail28:
				pos = pos27
			ok29:
			}
			labels[6] = parser.text[pos25:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		labels[7] = parser.text[pos1:pos]
	}
	return _memoize(parser, _MethSig, start, pos, perr)
fail:
	return _memoize(parser, _MethSig, start, -1, perr)
}

func _MethSigFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [8]string
	use(labels)
	pos, failure := _failMemo(parser, _MethSig, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "MethSig",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _MethSig}
	// action
	// _ sig:("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// sig:("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
	{
		pos1 := pos
		// ("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
		// action
		// "[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]"
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"[\"",
				})
			}
			goto fail
		}
		pos++
		// ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+)
		{
			pos3 := pos
			// (id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+)
			// id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+
			{
				pos7 := pos
				// action
				// id0:Ident
				{
					pos9 := pos
					// Ident
					if !_fail(parser, _IdentFail, errPos, failure, &pos) {
						goto fail8
					}
					labels[0] = parser.text[pos9:pos]
				}
				goto ok4
			fail8:
				pos = pos7
				// action
				// op:Op t0:TypeName
				// op:Op
				{
					pos12 := pos
					// Op
					if !_fail(parser, _OpFail, errPos, failure, &pos) {
						goto fail10
					}
					labels[1] = parser.text[pos12:pos]
				}
				// t0:TypeName
				{
					pos13 := pos
					// TypeName
					if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
						goto fail10
					}
					labels[2] = parser.text[pos13:pos]
				}
				goto ok4
			fail10:
				pos = pos7
				// (id1:IdentC t1:TypeName {…})+
				// (id1:IdentC t1:TypeName {…})
				// action
				// id1:IdentC t1:TypeName
				// id1:IdentC
				{
					pos20 := pos
					// IdentC
					if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
						goto fail14
					}
					labels[3] = parser.text[pos20:pos]
				}
				// t1:TypeName
				{
					pos21 := pos
					// TypeName
					if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
						goto fail14
					}
					labels[4] = parser.text[pos21:pos]
				}
				for {
					pos16 := pos
					// (id1:IdentC t1:TypeName {…})
					// action
					// id1:IdentC t1:TypeName
					// id1:IdentC
					{
						pos23 := pos
						// IdentC
						if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
							goto fail18
						}
						labels[3] = parser.text[pos23:pos]
					}
					// t1:TypeName
					{
						pos24 := pos
						// TypeName
						if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
							goto fail18
						}
						labels[4] = parser.text[pos24:pos]
					}
					continue
				fail18:
					pos = pos16
					break
				}
				goto ok4
			fail14:
				pos = pos7
				goto fail
			ok4:
			}
			labels[5] = parser.text[pos3:pos]
		}
		// r:Ret?
		{
			pos25 := pos
			// Ret?
			{
				pos27 := pos
				// Ret
				if !_fail(parser, _RetFail, errPos, failure, &pos) {
					goto fail28
				}
				goto ok29
			fail28:
				pos = pos27
			ok29:
			}
			labels[6] = parser.text[pos25:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"]\"",
				})
			}
			goto fail
		}
		pos++
		labels[7] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _MethSigAction(parser *_Parser, start int) (int, *FunSig) {
	var labels [8]string
	use(labels)
	var label0 Ident
	var label1 Ident
	var label2 TypeName
	var label3 Ident
	var label4 TypeName
	var label5 []parm
	var label6 *TypeName
	var label7 FunSig
	dp := parser.deltaPos[start][_MethSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _MethSig}
	n := parser.act[key]
	if n != nil {
		n := n.(FunSig)
		return start + int(dp-1), &n
	}
	var node FunSig
	pos := start
	// action
	{
		start0 := pos
		// _ sig:("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// sig:("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
		{
			pos2 := pos
			// ("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
			// action
			{
				start3 := pos
				// "[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]"
				// "["
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
					goto fail
				}
				pos++
				// ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+)
				{
					pos5 := pos
					// (id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+)
					// id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+
					{
						pos9 := pos
						var node8 []parm
						// action
						{
							start11 := pos
							// id0:Ident
							{
								pos12 := pos
								// Ident
								if p, n := _IdentAction(parser, pos); n == nil {
									goto fail10
								} else {
									label0 = *n
									pos = p
								}
								labels[0] = parser.text[pos12:pos]
							}
							label5 = func(
								start, end int, id0 Ident) []parm {
								return []parm{{name: id0}}
							}(
								start11, pos, label0)
						}
						goto ok6
					fail10:
						label5 = node8
						pos = pos9
						// action
						{
							start14 := pos
							// op:Op t0:TypeName
							// op:Op
							{
								pos16 := pos
								// Op
								if p, n := _OpAction(parser, pos); n == nil {
									goto fail13
								} else {
									label1 = *n
									pos = p
								}
								labels[1] = parser.text[pos16:pos]
							}
							// t0:TypeName
							{
								pos17 := pos
								// TypeName
								if p, n := _TypeNameAction(parser, pos); n == nil {
									goto fail13
								} else {
									label2 = *n
									pos = p
								}
								labels[2] = parser.text[pos17:pos]
							}
							label5 = func(
								start, end int, id0 Ident, op Ident, t0 TypeName) []parm {
								return []parm{{name: op, typ: t0}}
							}(
								start14, pos, label0, label1, label2)
						}
						goto ok6
					fail13:
						label5 = node8
						pos = pos9
						// (id1:IdentC t1:TypeName {…})+
						{
							var node21 parm
							// (id1:IdentC t1:TypeName {…})
							// action
							{
								start23 := pos
								// id1:IdentC t1:TypeName
								// id1:IdentC
								{
									pos25 := pos
									// IdentC
									if p, n := _IdentCAction(parser, pos); n == nil {
										goto fail18
									} else {
										label3 = *n
										pos = p
									}
									labels[3] = parser.text[pos25:pos]
								}
								// t1:TypeName
								{
									pos26 := pos
									// TypeName
									if p, n := _TypeNameAction(parser, pos); n == nil {
										goto fail18
									} else {
										label4 = *n
										pos = p
									}
									labels[4] = parser.text[pos26:pos]
								}
								node21 = func(
									start, end int, id0 Ident, id1 Ident, op Ident, t0 TypeName, t1 TypeName) parm {
									return parm{name: id1, typ: t1}
								}(
									start23, pos, label0, label3, label1, label2, label4)
							}
							label5 = append(label5, node21)
						}
						for {
							pos20 := pos
							var node21 parm
							// (id1:IdentC t1:TypeName {…})
							// action
							{
								start27 := pos
								// id1:IdentC t1:TypeName
								// id1:IdentC
								{
									pos29 := pos
									// IdentC
									if p, n := _IdentCAction(parser, pos); n == nil {
										goto fail22
									} else {
										label3 = *n
										pos = p
									}
									labels[3] = parser.text[pos29:pos]
								}
								// t1:TypeName
								{
									pos30 := pos
									// TypeName
									if p, n := _TypeNameAction(parser, pos); n == nil {
										goto fail22
									} else {
										label4 = *n
										pos = p
									}
									labels[4] = parser.text[pos30:pos]
								}
								node21 = func(
									start, end int, id0 Ident, id1 Ident, op Ident, t0 TypeName, t1 TypeName) parm {
									return parm{name: id1, typ: t1}
								}(
									start27, pos, label0, label3, label1, label2, label4)
							}
							label5 = append(label5, node21)
							continue
						fail22:
							pos = pos20
							break
						}
						goto ok6
					fail18:
						label5 = node8
						pos = pos9
						goto fail
					ok6:
					}
					labels[5] = parser.text[pos5:pos]
				}
				// r:Ret?
				{
					pos31 := pos
					// Ret?
					{
						pos33 := pos
						label6 = new(TypeName)
						// Ret
						if p, n := _RetAction(parser, pos); n == nil {
							goto fail34
						} else {
							*label6 = *n
							pos = p
						}
						goto ok35
					fail34:
						label6 = nil
						pos = pos33
					ok35:
					}
					labels[6] = parser.text[pos31:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "]"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
					goto fail
				}
				pos++
				label7 = func(
					start, end int, id0 Ident, id1 Ident, op Ident, ps []parm, r *TypeName, t0 TypeName, t1 TypeName) FunSig {
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
						Sel:      s,
						Parms:    parms,
						Ret:      r,
					}
				}(
					start3, pos, label0, label3, label1, label5, label6, label2, label4)
			}
			labels[7] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, id0 Ident, id1 Ident, op Ident, ps []parm, r *TypeName, sig FunSig, t0 TypeName, t1 TypeName) FunSig {
			return FunSig(sig)
		}(
			start0, pos, label0, label3, label1, label5, label6, label7, label2, label4)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _StmtsAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Stmts, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// ss:(s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})?
	{
		pos0 := pos
		// (s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})?
		{
			pos2 := pos
			// (s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})
			// action
			// s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")?
			// s0:Stmt
			{
				pos5 := pos
				// Stmt
				if !_accept(parser, _StmtAccepts, &pos, &perr) {
					goto fail3
				}
				labels[0] = parser.text[pos5:pos]
			}
			// s1s:(_ "." s1:Stmt {…})*
			{
				pos6 := pos
				// (_ "." s1:Stmt {…})*
				for {
					pos8 := pos
					// (_ "." s1:Stmt {…})
					// action
					// _ "." s1:Stmt
					// _
					if !_accept(parser, __Accepts, &pos, &perr) {
						goto fail10
					}
					// "."
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
						perr = _max(perr, pos)
						goto fail10
					}
					pos++
					// s1:Stmt
					{
						pos12 := pos
						// Stmt
						if !_accept(parser, _StmtAccepts, &pos, &perr) {
							goto fail10
						}
						labels[1] = parser.text[pos12:pos]
					}
					continue
				fail10:
					pos = pos8
					break
				}
				labels[2] = parser.text[pos6:pos]
			}
			// (_ ".")?
			{
				pos14 := pos
				// (_ ".")
				// _ "."
				// _
				if !_accept(parser, __Accepts, &pos, &perr) {
					goto fail15
				}
				// "."
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
					perr = _max(perr, pos)
					goto fail15
				}
				pos++
				goto ok17
			fail15:
				pos = pos14
			ok17:
			}
			goto ok18
		fail3:
			pos = pos2
		ok18:
		}
		labels[3] = parser.text[pos0:pos]
	}
	return _memoize(parser, _Stmts, start, pos, perr)
}

func _StmtsFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
	use(labels)
	pos, failure := _failMemo(parser, _Stmts, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Stmts",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Stmts}
	// action
	// ss:(s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})?
	{
		pos0 := pos
		// (s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})?
		{
			pos2 := pos
			// (s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})
			// action
			// s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")?
			// s0:Stmt
			{
				pos5 := pos
				// Stmt
				if !_fail(parser, _StmtFail, errPos, failure, &pos) {
					goto fail3
				}
				labels[0] = parser.text[pos5:pos]
			}
			// s1s:(_ "." s1:Stmt {…})*
			{
				pos6 := pos
				// (_ "." s1:Stmt {…})*
				for {
					pos8 := pos
					// (_ "." s1:Stmt {…})
					// action
					// _ "." s1:Stmt
					// _
					if !_fail(parser, __Fail, errPos, failure, &pos) {
						goto fail10
					}
					// "."
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "\".\"",
							})
						}
						goto fail10
					}
					pos++
					// s1:Stmt
					{
						pos12 := pos
						// Stmt
						if !_fail(parser, _StmtFail, errPos, failure, &pos) {
							goto fail10
						}
						labels[1] = parser.text[pos12:pos]
					}
					continue
				fail10:
					pos = pos8
					break
				}
				labels[2] = parser.text[pos6:pos]
			}
			// (_ ".")?
			{
				pos14 := pos
				// (_ ".")
				// _ "."
				// _
				if !_fail(parser, __Fail, errPos, failure, &pos) {
					goto fail15
				}
				// "."
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\".\"",
						})
					}
					goto fail15
				}
				pos++
				goto ok17
			fail15:
				pos = pos14
			ok17:
			}
			goto ok18
		fail3:
			pos = pos2
		ok18:
		}
		labels[3] = parser.text[pos0:pos]
	}
	parser.fail[key] = failure
	return pos, failure
}

func _StmtsAction(parser *_Parser, start int) (int, *[]Stmt) {
	var labels [4]string
	use(labels)
	var label0 Stmt
	var label1 Stmt
	var label2 []Stmt
	var label3 *[]Stmt
	dp := parser.deltaPos[start][_Stmts]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Stmts}
	n := parser.act[key]
	if n != nil {
		n := n.([]Stmt)
		return start + int(dp-1), &n
	}
	var node []Stmt
	pos := start
	// action
	{
		start0 := pos
		// ss:(s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})?
		{
			pos1 := pos
			// (s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})?
			{
				pos3 := pos
				label3 = new([]Stmt)
				// (s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})
				// action
				{
					start5 := pos
					// s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")?
					// s0:Stmt
					{
						pos7 := pos
						// Stmt
						if p, n := _StmtAction(parser, pos); n == nil {
							goto fail4
						} else {
							label0 = *n
							pos = p
						}
						labels[0] = parser.text[pos7:pos]
					}
					// s1s:(_ "." s1:Stmt {…})*
					{
						pos8 := pos
						// (_ "." s1:Stmt {…})*
						for {
							pos10 := pos
							var node11 Stmt
							// (_ "." s1:Stmt {…})
							// action
							{
								start13 := pos
								// _ "." s1:Stmt
								// _
								if p, n := __Action(parser, pos); n == nil {
									goto fail12
								} else {
									pos = p
								}
								// "."
								if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
									goto fail12
								}
								pos++
								// s1:Stmt
								{
									pos15 := pos
									// Stmt
									if p, n := _StmtAction(parser, pos); n == nil {
										goto fail12
									} else {
										label1 = *n
										pos = p
									}
									labels[1] = parser.text[pos15:pos]
								}
								node11 = func(
									start, end int, s0 Stmt, s1 Stmt) Stmt {
									return Stmt(s1)
								}(
									start13, pos, label0, label1)
							}
							label2 = append(label2, node11)
							continue
						fail12:
							pos = pos10
							break
						}
						labels[2] = parser.text[pos8:pos]
					}
					// (_ ".")?
					{
						pos17 := pos
						// (_ ".")
						// _ "."
						// _
						if p, n := __Action(parser, pos); n == nil {
							goto fail18
						} else {
							pos = p
						}
						// "."
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
							goto fail18
						}
						pos++
						goto ok20
					fail18:
						pos = pos17
					ok20:
					}
					*label3 = func(
						start, end int, s0 Stmt, s1 Stmt, s1s []Stmt) []Stmt {
						return []Stmt(append([]Stmt{s0}, s1s...))
					}(
						start5, pos, label0, label1, label2)
				}
				goto ok21
			fail4:
				label3 = nil
				pos = pos3
			ok21:
			}
			labels[3] = parser.text[pos1:pos]
		}
		node = func(
			start, end int, s0 Stmt, s1 Stmt, s1s []Stmt, ss *[]Stmt) []Stmt {
			if ss != nil {
				return *ss
			}
			return []Stmt{}
		}(
			start0, pos, label0, label1, label2, label3)
	}
	parser.act[key] = node
	return pos, &node
}

func _StmtAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [1]string
	use(labels)
	if dp, de, ok := _memo(parser, _Stmt, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Return/Assign/e:Expr {…}
	{
		pos3 := pos
		// Return
		if !_accept(parser, _ReturnAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Assign
		if !_accept(parser, _AssignAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// action
		// e:Expr
		{
			pos7 := pos
			// Expr
			if !_accept(parser, _ExprAccepts, &pos, &perr) {
				goto fail6
			}
			labels[0] = parser.text[pos7:pos]
		}
		goto ok0
	fail6:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Stmt, start, pos, perr)
fail:
	return _memoize(parser, _Stmt, start, -1, perr)
}

func _StmtFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [1]string
	use(labels)
	pos, failure := _failMemo(parser, _Stmt, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Stmt",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Stmt}
	// Return/Assign/e:Expr {…}
	{
		pos3 := pos
		// Return
		if !_fail(parser, _ReturnFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Assign
		if !_fail(parser, _AssignFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// action
		// e:Expr
		{
			pos7 := pos
			// Expr
			if !_fail(parser, _ExprFail, errPos, failure, &pos) {
				goto fail6
			}
			labels[0] = parser.text[pos7:pos]
		}
		goto ok0
	fail6:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _StmtAction(parser *_Parser, start int) (int, *Stmt) {
	var labels [1]string
	use(labels)
	var label0 Expr
	dp := parser.deltaPos[start][_Stmt]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Stmt}
	n := parser.act[key]
	if n != nil {
		n := n.(Stmt)
		return start + int(dp-1), &n
	}
	var node Stmt
	pos := start
	// Return/Assign/e:Expr {…}
	{
		pos3 := pos
		var node2 Stmt
		// Return
		if p, n := _ReturnAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Assign
		if p, n := _AssignAction(parser, pos); n == nil {
			goto fail5
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail5:
		node = node2
		pos = pos3
		// action
		{
			start7 := pos
			// e:Expr
			{
				pos8 := pos
				// Expr
				if p, n := _ExprAction(parser, pos); n == nil {
					goto fail6
				} else {
					label0 = *n
					pos = p
				}
				labels[0] = parser.text[pos8:pos]
			}
			node = func(
				start, end int, e Expr) Stmt {
				return Stmt(e)
			}(
				start7, pos, label0)
		}
		goto ok0
	fail6:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ReturnAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Return, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ r:("^" e:Expr {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// r:("^" e:Expr {…})
	{
		pos1 := pos
		// ("^" e:Expr {…})
		// action
		// "^" e:Expr
		// "^"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// e:Expr
		{
			pos3 := pos
			// Expr
			if !_accept(parser, _ExprAccepts, &pos, &perr) {
				goto fail
			}
			labels[0] = parser.text[pos3:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Return, start, pos, perr)
fail:
	return _memoize(parser, _Return, start, -1, perr)
}

func _ReturnFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Return, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Return",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Return}
	// action
	// _ r:("^" e:Expr {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// r:("^" e:Expr {…})
	{
		pos1 := pos
		// ("^" e:Expr {…})
		// action
		// "^" e:Expr
		// "^"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"^\"",
				})
			}
			goto fail
		}
		pos++
		// e:Expr
		{
			pos3 := pos
			// Expr
			if !_fail(parser, _ExprFail, errPos, failure, &pos) {
				goto fail
			}
			labels[0] = parser.text[pos3:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _ReturnAction(parser *_Parser, start int) (int, *Stmt) {
	var labels [2]string
	use(labels)
	var label0 Expr
	var label1 *Ret
	dp := parser.deltaPos[start][_Return]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Return}
	n := parser.act[key]
	if n != nil {
		n := n.(Stmt)
		return start + int(dp-1), &n
	}
	var node Stmt
	pos := start
	// action
	{
		start0 := pos
		// _ r:("^" e:Expr {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// r:("^" e:Expr {…})
		{
			pos2 := pos
			// ("^" e:Expr {…})
			// action
			{
				start3 := pos
				// "^" e:Expr
				// "^"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
					goto fail
				}
				pos++
				// e:Expr
				{
					pos5 := pos
					// Expr
					if p, n := _ExprAction(parser, pos); n == nil {
						goto fail
					} else {
						label0 = *n
						pos = p
					}
					labels[0] = parser.text[pos5:pos]
				}
				label1 = func(
					start, end int, e Expr) *Ret {
					return &Ret{start: loc1(parser, start), Val: e}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, e Expr, r *Ret) Stmt {
			return Stmt(r)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _AssignAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Assign, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// l:Lhs _ ":=" r:Expr
	// l:Lhs
	{
		pos1 := pos
		// Lhs
		if !_accept(parser, _LhsAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// ":="
	if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
		perr = _max(perr, pos)
		goto fail
	}
	pos += 2
	// r:Expr
	{
		pos2 := pos
		// Expr
		if !_accept(parser, _ExprAccepts, &pos, &perr) {
			goto fail
		}
		labels[1] = parser.text[pos2:pos]
	}
	return _memoize(parser, _Assign, start, pos, perr)
fail:
	return _memoize(parser, _Assign, start, -1, perr)
}

func _AssignFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Assign, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Assign",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Assign}
	// action
	// l:Lhs _ ":=" r:Expr
	// l:Lhs
	{
		pos1 := pos
		// Lhs
		if !_fail(parser, _LhsFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// ":="
	if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\":=\"",
			})
		}
		goto fail
	}
	pos += 2
	// r:Expr
	{
		pos2 := pos
		// Expr
		if !_fail(parser, _ExprFail, errPos, failure, &pos) {
			goto fail
		}
		labels[1] = parser.text[pos2:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _AssignAction(parser *_Parser, start int) (int, *Stmt) {
	var labels [2]string
	use(labels)
	var label0 []Var
	var label1 Expr
	dp := parser.deltaPos[start][_Assign]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Assign}
	n := parser.act[key]
	if n != nil {
		n := n.(Stmt)
		return start + int(dp-1), &n
	}
	var node Stmt
	pos := start
	// action
	{
		start0 := pos
		// l:Lhs _ ":=" r:Expr
		// l:Lhs
		{
			pos2 := pos
			// Lhs
			if p, n := _LhsAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// ":="
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
			goto fail
		}
		pos += 2
		// r:Expr
		{
			pos3 := pos
			// Expr
			if p, n := _ExprAction(parser, pos); n == nil {
				goto fail
			} else {
				label1 = *n
				pos = p
			}
			labels[1] = parser.text[pos3:pos]
		}
		node = func(
			start, end int, l []Var, r Expr) Stmt {
			return Stmt(&Assign{Vars: l, Expr: r})
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _LhsAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [6]string
	use(labels)
	if dp, de, ok := _memo(parser, _Lhs, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// id:(i0:Ident t0:TypeName? {…}) is:(_ "," i1:Ident t1:TypeName? {…})*
	// id:(i0:Ident t0:TypeName? {…})
	{
		pos1 := pos
		// (i0:Ident t0:TypeName? {…})
		// action
		// i0:Ident t0:TypeName?
		// i0:Ident
		{
			pos3 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail
			}
			labels[0] = parser.text[pos3:pos]
		}
		// t0:TypeName?
		{
			pos4 := pos
			// TypeName?
			{
				pos6 := pos
				// TypeName
				if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
					goto fail7
				}
				goto ok8
			fail7:
				pos = pos6
			ok8:
			}
			labels[1] = parser.text[pos4:pos]
		}
		labels[2] = parser.text[pos1:pos]
	}
	// is:(_ "," i1:Ident t1:TypeName? {…})*
	{
		pos9 := pos
		// (_ "," i1:Ident t1:TypeName? {…})*
		for {
			pos11 := pos
			// (_ "," i1:Ident t1:TypeName? {…})
			// action
			// _ "," i1:Ident t1:TypeName?
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail13
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				perr = _max(perr, pos)
				goto fail13
			}
			pos++
			// i1:Ident
			{
				pos15 := pos
				// Ident
				if !_accept(parser, _IdentAccepts, &pos, &perr) {
					goto fail13
				}
				labels[3] = parser.text[pos15:pos]
			}
			// t1:TypeName?
			{
				pos16 := pos
				// TypeName?
				{
					pos18 := pos
					// TypeName
					if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
						goto fail19
					}
					goto ok20
				fail19:
					pos = pos18
				ok20:
				}
				labels[4] = parser.text[pos16:pos]
			}
			continue
		fail13:
			pos = pos11
			break
		}
		labels[5] = parser.text[pos9:pos]
	}
	return _memoize(parser, _Lhs, start, pos, perr)
fail:
	return _memoize(parser, _Lhs, start, -1, perr)
}

func _LhsFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [6]string
	use(labels)
	pos, failure := _failMemo(parser, _Lhs, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Lhs",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Lhs}
	// action
	// id:(i0:Ident t0:TypeName? {…}) is:(_ "," i1:Ident t1:TypeName? {…})*
	// id:(i0:Ident t0:TypeName? {…})
	{
		pos1 := pos
		// (i0:Ident t0:TypeName? {…})
		// action
		// i0:Ident t0:TypeName?
		// i0:Ident
		{
			pos3 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail
			}
			labels[0] = parser.text[pos3:pos]
		}
		// t0:TypeName?
		{
			pos4 := pos
			// TypeName?
			{
				pos6 := pos
				// TypeName
				if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
					goto fail7
				}
				goto ok8
			fail7:
				pos = pos6
			ok8:
			}
			labels[1] = parser.text[pos4:pos]
		}
		labels[2] = parser.text[pos1:pos]
	}
	// is:(_ "," i1:Ident t1:TypeName? {…})*
	{
		pos9 := pos
		// (_ "," i1:Ident t1:TypeName? {…})*
		for {
			pos11 := pos
			// (_ "," i1:Ident t1:TypeName? {…})
			// action
			// _ "," i1:Ident t1:TypeName?
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail13
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\",\"",
					})
				}
				goto fail13
			}
			pos++
			// i1:Ident
			{
				pos15 := pos
				// Ident
				if !_fail(parser, _IdentFail, errPos, failure, &pos) {
					goto fail13
				}
				labels[3] = parser.text[pos15:pos]
			}
			// t1:TypeName?
			{
				pos16 := pos
				// TypeName?
				{
					pos18 := pos
					// TypeName
					if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
						goto fail19
					}
					goto ok20
				fail19:
					pos = pos18
				ok20:
				}
				labels[4] = parser.text[pos16:pos]
			}
			continue
		fail13:
			pos = pos11
			break
		}
		labels[5] = parser.text[pos9:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _LhsAction(parser *_Parser, start int) (int, *[]Var) {
	var labels [6]string
	use(labels)
	var label0 Ident
	var label1 *TypeName
	var label2 Var
	var label3 Ident
	var label4 *TypeName
	var label5 []Var
	dp := parser.deltaPos[start][_Lhs]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Lhs}
	n := parser.act[key]
	if n != nil {
		n := n.([]Var)
		return start + int(dp-1), &n
	}
	var node []Var
	pos := start
	// action
	{
		start0 := pos
		// id:(i0:Ident t0:TypeName? {…}) is:(_ "," i1:Ident t1:TypeName? {…})*
		// id:(i0:Ident t0:TypeName? {…})
		{
			pos2 := pos
			// (i0:Ident t0:TypeName? {…})
			// action
			{
				start3 := pos
				// i0:Ident t0:TypeName?
				// i0:Ident
				{
					pos5 := pos
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail
					} else {
						label0 = *n
						pos = p
					}
					labels[0] = parser.text[pos5:pos]
				}
				// t0:TypeName?
				{
					pos6 := pos
					// TypeName?
					{
						pos8 := pos
						label1 = new(TypeName)
						// TypeName
						if p, n := _TypeNameAction(parser, pos); n == nil {
							goto fail9
						} else {
							*label1 = *n
							pos = p
						}
						goto ok10
					fail9:
						label1 = nil
						pos = pos8
					ok10:
					}
					labels[1] = parser.text[pos6:pos]
				}
				label2 = func(
					start, end int, i0 Ident, t0 *TypeName) Var {
					e := i0.end
					if t0 != nil {
						e = t0.end
					}
					return Var{
						location: location{i0.start, e},
						Name:     i0.Text,
						Type:     t0,
					}
				}(
					start3, pos, label0, label1)
			}
			labels[2] = parser.text[pos2:pos]
		}
		// is:(_ "," i1:Ident t1:TypeName? {…})*
		{
			pos11 := pos
			// (_ "," i1:Ident t1:TypeName? {…})*
			for {
				pos13 := pos
				var node14 Var
				// (_ "," i1:Ident t1:TypeName? {…})
				// action
				{
					start16 := pos
					// _ "," i1:Ident t1:TypeName?
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail15
					} else {
						pos = p
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail15
					}
					pos++
					// i1:Ident
					{
						pos18 := pos
						// Ident
						if p, n := _IdentAction(parser, pos); n == nil {
							goto fail15
						} else {
							label3 = *n
							pos = p
						}
						labels[3] = parser.text[pos18:pos]
					}
					// t1:TypeName?
					{
						pos19 := pos
						// TypeName?
						{
							pos21 := pos
							label4 = new(TypeName)
							// TypeName
							if p, n := _TypeNameAction(parser, pos); n == nil {
								goto fail22
							} else {
								*label4 = *n
								pos = p
							}
							goto ok23
						fail22:
							label4 = nil
							pos = pos21
						ok23:
						}
						labels[4] = parser.text[pos19:pos]
					}
					node14 = func(
						start, end int, i0 Ident, i1 Ident, id Var, t0 *TypeName, t1 *TypeName) Var {
						e := i1.end
						if t1 != nil {
							e = t1.end
						}
						return Var{
							location: location{i1.start, e},
							Name:     i1.Text,
							Type:     t1,
						}
					}(
						start16, pos, label0, label3, label2, label1, label4)
				}
				label5 = append(label5, node14)
				continue
			fail15:
				pos = pos13
				break
			}
			labels[5] = parser.text[pos11:pos]
		}
		node = func(
			start, end int, i0 Ident, i1 Ident, id Var, is []Var, t0 *TypeName, t1 *TypeName) []Var {
			return []Var(append([]Var{id}, is...))
		}(
			start0, pos, label0, label3, label2, label5, label1, label4)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ExprAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Expr, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Call/Primary
	{
		pos3 := pos
		// Call
		if !_accept(parser, _CallAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Primary
		if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Expr, start, pos, perr)
fail:
	return _memoize(parser, _Expr, start, -1, perr)
}

func _ExprFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Expr, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Expr",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Expr}
	// Call/Primary
	{
		pos3 := pos
		// Call
		if !_fail(parser, _CallFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Primary
		if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _ExprAction(parser *_Parser, start int) (int, *Expr) {
	dp := parser.deltaPos[start][_Expr]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Expr}
	n := parser.act[key]
	if n != nil {
		n := n.(Expr)
		return start + int(dp-1), &n
	}
	var node Expr
	pos := start
	// Call/Primary
	{
		pos3 := pos
		var node2 Expr
		// Call
		if p, n := _CallAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Primary
		if p, n := _PrimaryAction(parser, pos); n == nil {
			goto fail5
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail5:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _CallAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Call, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// c:(Nary/Binary/Unary) cs:(_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
	// c:(Nary/Binary/Unary)
	{
		pos1 := pos
		// (Nary/Binary/Unary)
		// Nary/Binary/Unary
		{
			pos5 := pos
			// Nary
			if !_accept(parser, _NaryAccepts, &pos, &perr) {
				goto fail6
			}
			goto ok2
		fail6:
			pos = pos5
			// Binary
			if !_accept(parser, _BinaryAccepts, &pos, &perr) {
				goto fail7
			}
			goto ok2
		fail7:
			pos = pos5
			// Unary
			if !_accept(parser, _UnaryAccepts, &pos, &perr) {
				goto fail8
			}
			goto ok2
		fail8:
			pos = pos5
			goto fail
		ok2:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// cs:(_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
	{
		pos9 := pos
		// (_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		for {
			pos11 := pos
			// (_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})
			// action
			// _ "," m:(UnaryMsg/BinMsg/NaryMsg)
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail13
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				perr = _max(perr, pos)
				goto fail13
			}
			pos++
			// m:(UnaryMsg/BinMsg/NaryMsg)
			{
				pos15 := pos
				// (UnaryMsg/BinMsg/NaryMsg)
				// UnaryMsg/BinMsg/NaryMsg
				{
					pos19 := pos
					// UnaryMsg
					if !_accept(parser, _UnaryMsgAccepts, &pos, &perr) {
						goto fail20
					}
					goto ok16
				fail20:
					pos = pos19
					// BinMsg
					if !_accept(parser, _BinMsgAccepts, &pos, &perr) {
						goto fail21
					}
					goto ok16
				fail21:
					pos = pos19
					// NaryMsg
					if !_accept(parser, _NaryMsgAccepts, &pos, &perr) {
						goto fail22
					}
					goto ok16
				fail22:
					pos = pos19
					goto fail13
				ok16:
				}
				labels[1] = parser.text[pos15:pos]
			}
			continue
		fail13:
			pos = pos11
			break
		}
		labels[2] = parser.text[pos9:pos]
	}
	return _memoize(parser, _Call, start, pos, perr)
fail:
	return _memoize(parser, _Call, start, -1, perr)
}

func _CallFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Call, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Call",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Call}
	// action
	// c:(Nary/Binary/Unary) cs:(_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
	// c:(Nary/Binary/Unary)
	{
		pos1 := pos
		// (Nary/Binary/Unary)
		// Nary/Binary/Unary
		{
			pos5 := pos
			// Nary
			if !_fail(parser, _NaryFail, errPos, failure, &pos) {
				goto fail6
			}
			goto ok2
		fail6:
			pos = pos5
			// Binary
			if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
				goto fail7
			}
			goto ok2
		fail7:
			pos = pos5
			// Unary
			if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
				goto fail8
			}
			goto ok2
		fail8:
			pos = pos5
			goto fail
		ok2:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// cs:(_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
	{
		pos9 := pos
		// (_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		for {
			pos11 := pos
			// (_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})
			// action
			// _ "," m:(UnaryMsg/BinMsg/NaryMsg)
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail13
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\",\"",
					})
				}
				goto fail13
			}
			pos++
			// m:(UnaryMsg/BinMsg/NaryMsg)
			{
				pos15 := pos
				// (UnaryMsg/BinMsg/NaryMsg)
				// UnaryMsg/BinMsg/NaryMsg
				{
					pos19 := pos
					// UnaryMsg
					if !_fail(parser, _UnaryMsgFail, errPos, failure, &pos) {
						goto fail20
					}
					goto ok16
				fail20:
					pos = pos19
					// BinMsg
					if !_fail(parser, _BinMsgFail, errPos, failure, &pos) {
						goto fail21
					}
					goto ok16
				fail21:
					pos = pos19
					// NaryMsg
					if !_fail(parser, _NaryMsgFail, errPos, failure, &pos) {
						goto fail22
					}
					goto ok16
				fail22:
					pos = pos19
					goto fail13
				ok16:
				}
				labels[1] = parser.text[pos15:pos]
			}
			continue
		fail13:
			pos = pos11
			break
		}
		labels[2] = parser.text[pos9:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _CallAction(parser *_Parser, start int) (int, *Expr) {
	var labels [3]string
	use(labels)
	var label0 *Call
	var label1 Msg
	var label2 []Msg
	dp := parser.deltaPos[start][_Call]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Call}
	n := parser.act[key]
	if n != nil {
		n := n.(Expr)
		return start + int(dp-1), &n
	}
	var node Expr
	pos := start
	// action
	{
		start0 := pos
		// c:(Nary/Binary/Unary) cs:(_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		// c:(Nary/Binary/Unary)
		{
			pos2 := pos
			// (Nary/Binary/Unary)
			// Nary/Binary/Unary
			{
				pos6 := pos
				var node5 *Call
				// Nary
				if p, n := _NaryAction(parser, pos); n == nil {
					goto fail7
				} else {
					label0 = *n
					pos = p
				}
				goto ok3
			fail7:
				label0 = node5
				pos = pos6
				// Binary
				if p, n := _BinaryAction(parser, pos); n == nil {
					goto fail8
				} else {
					label0 = *n
					pos = p
				}
				goto ok3
			fail8:
				label0 = node5
				pos = pos6
				// Unary
				if p, n := _UnaryAction(parser, pos); n == nil {
					goto fail9
				} else {
					label0 = *n
					pos = p
				}
				goto ok3
			fail9:
				label0 = node5
				pos = pos6
				goto fail
			ok3:
			}
			labels[0] = parser.text[pos2:pos]
		}
		// cs:(_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		{
			pos10 := pos
			// (_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})*
			for {
				pos12 := pos
				var node13 Msg
				// (_ "," m:(UnaryMsg/BinMsg/NaryMsg) {…})
				// action
				{
					start15 := pos
					// _ "," m:(UnaryMsg/BinMsg/NaryMsg)
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail14
					} else {
						pos = p
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail14
					}
					pos++
					// m:(UnaryMsg/BinMsg/NaryMsg)
					{
						pos17 := pos
						// (UnaryMsg/BinMsg/NaryMsg)
						// UnaryMsg/BinMsg/NaryMsg
						{
							pos21 := pos
							var node20 Msg
							// UnaryMsg
							if p, n := _UnaryMsgAction(parser, pos); n == nil {
								goto fail22
							} else {
								label1 = *n
								pos = p
							}
							goto ok18
						fail22:
							label1 = node20
							pos = pos21
							// BinMsg
							if p, n := _BinMsgAction(parser, pos); n == nil {
								goto fail23
							} else {
								label1 = *n
								pos = p
							}
							goto ok18
						fail23:
							label1 = node20
							pos = pos21
							// NaryMsg
							if p, n := _NaryMsgAction(parser, pos); n == nil {
								goto fail24
							} else {
								label1 = *n
								pos = p
							}
							goto ok18
						fail24:
							label1 = node20
							pos = pos21
							goto fail14
						ok18:
						}
						labels[1] = parser.text[pos17:pos]
					}
					node13 = func(
						start, end int, c *Call, m Msg) Msg {
						return Msg(m)
					}(
						start15, pos, label0, label1)
				}
				label2 = append(label2, node13)
				continue
			fail14:
				pos = pos12
				break
			}
			labels[2] = parser.text[pos10:pos]
		}
		node = func(
			start, end int, c *Call, cs []Msg, m Msg) Expr {
			c.Msgs = append(c.Msgs, cs...)
			return Expr(c)
		}(
			start0, pos, label0, label2, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _UnaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Unary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// r:Primary? ms:UnaryMsg+
	// r:Primary?
	{
		pos1 := pos
		// Primary?
		{
			pos3 := pos
			// Primary
			if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// ms:UnaryMsg+
	{
		pos6 := pos
		// UnaryMsg+
		// UnaryMsg
		if !_accept(parser, _UnaryMsgAccepts, &pos, &perr) {
			goto fail
		}
		for {
			pos8 := pos
			// UnaryMsg
			if !_accept(parser, _UnaryMsgAccepts, &pos, &perr) {
				goto fail10
			}
			continue
		fail10:
			pos = pos8
			break
		}
		labels[1] = parser.text[pos6:pos]
	}
	return _memoize(parser, _Unary, start, pos, perr)
fail:
	return _memoize(parser, _Unary, start, -1, perr)
}

func _UnaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Unary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Unary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Unary}
	// action
	// r:Primary? ms:UnaryMsg+
	// r:Primary?
	{
		pos1 := pos
		// Primary?
		{
			pos3 := pos
			// Primary
			if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// ms:UnaryMsg+
	{
		pos6 := pos
		// UnaryMsg+
		// UnaryMsg
		if !_fail(parser, _UnaryMsgFail, errPos, failure, &pos) {
			goto fail
		}
		for {
			pos8 := pos
			// UnaryMsg
			if !_fail(parser, _UnaryMsgFail, errPos, failure, &pos) {
				goto fail10
			}
			continue
		fail10:
			pos = pos8
			break
		}
		labels[1] = parser.text[pos6:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _UnaryAction(parser *_Parser, start int) (int, **Call) {
	var labels [2]string
	use(labels)
	var label0 *Expr
	var label1 []Msg
	dp := parser.deltaPos[start][_Unary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Unary}
	n := parser.act[key]
	if n != nil {
		n := n.(*Call)
		return start + int(dp-1), &n
	}
	var node *Call
	pos := start
	// action
	{
		start0 := pos
		// r:Primary? ms:UnaryMsg+
		// r:Primary?
		{
			pos2 := pos
			// Primary?
			{
				pos4 := pos
				label0 = new(Expr)
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail5
				} else {
					*label0 = *n
					pos = p
				}
				goto ok6
			fail5:
				label0 = nil
				pos = pos4
			ok6:
			}
			labels[0] = parser.text[pos2:pos]
		}
		// ms:UnaryMsg+
		{
			pos7 := pos
			// UnaryMsg+
			{
				var node10 Msg
				// UnaryMsg
				if p, n := _UnaryMsgAction(parser, pos); n == nil {
					goto fail
				} else {
					node10 = *n
					pos = p
				}
				label1 = append(label1, node10)
			}
			for {
				pos9 := pos
				var node10 Msg
				// UnaryMsg
				if p, n := _UnaryMsgAction(parser, pos); n == nil {
					goto fail11
				} else {
					node10 = *n
					pos = p
				}
				label1 = append(label1, node10)
				continue
			fail11:
				pos = pos9
				break
			}
			labels[1] = parser.text[pos7:pos]
		}
		node = func(
			start, end int, ms []Msg, r *Expr) *Call {
			s := ms[0].start
			var recv Expr
			if r != nil {
				s, _ = (*r).loc()
				recv = *r
			}
			c := &Call{
				location: location{s, ms[0].end},
				Recv:     recv,
				Msgs:     []Msg{ms[0]},
			}
			for _, m := range ms[1:] {
				c = &Call{
					location: location{s, m.end},
					Recv:     c,
					Msgs:     []Msg{m},
				}
			}
			// TODO: fix the (*Call)(c) workaround for a Peggy bug.
			// Ideally we would just
			// 	return (*Call)(c)
			// However, there is a bugy in Peggy where it detects the type as
			// (*Call) instead of just *Call, and it gives a type mismatch error.
			if true {
				return (*Call)(c)
			}
			return &Call{}
		}(
			start0, pos, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _UnaryMsgAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _UnaryMsg, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// mod:ModName? i:Ident
	// mod:ModName?
	{
		pos1 := pos
		// ModName?
		{
			pos3 := pos
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// i:Ident
	{
		pos6 := pos
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail
		}
		labels[1] = parser.text[pos6:pos]
	}
	return _memoize(parser, _UnaryMsg, start, pos, perr)
fail:
	return _memoize(parser, _UnaryMsg, start, -1, perr)
}

func _UnaryMsgFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _UnaryMsg, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "UnaryMsg",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _UnaryMsg}
	// action
	// mod:ModName? i:Ident
	// mod:ModName?
	{
		pos1 := pos
		// ModName?
		{
			pos3 := pos
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// i:Ident
	{
		pos6 := pos
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail
		}
		labels[1] = parser.text[pos6:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _UnaryMsgAction(parser *_Parser, start int) (int, *Msg) {
	var labels [2]string
	use(labels)
	var label0 *Ident
	var label1 Ident
	dp := parser.deltaPos[start][_UnaryMsg]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _UnaryMsg}
	n := parser.act[key]
	if n != nil {
		n := n.(Msg)
		return start + int(dp-1), &n
	}
	var node Msg
	pos := start
	// action
	{
		start0 := pos
		// mod:ModName? i:Ident
		// mod:ModName?
		{
			pos2 := pos
			// ModName?
			{
				pos4 := pos
				label0 = new(Ident)
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail5
				} else {
					*label0 = *n
					pos = p
				}
				goto ok6
			fail5:
				label0 = nil
				pos = pos4
			ok6:
			}
			labels[0] = parser.text[pos2:pos]
		}
		// i:Ident
		{
			pos7 := pos
			// Ident
			if p, n := _IdentAction(parser, pos); n == nil {
				goto fail
			} else {
				label1 = *n
				pos = p
			}
			labels[1] = parser.text[pos7:pos]
		}
		node = func(
			start, end int, i Ident, mod *Ident) Msg {
			return Msg{location: i.location, Mod: mod, Sel: i.Text}
		}(
			start0, pos, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BinaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Binary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// r:(u:Unary {…}/Primary) m:BinMsg
	// r:(u:Unary {…}/Primary)
	{
		pos1 := pos
		// (u:Unary {…}/Primary)
		// u:Unary {…}/Primary
		{
			pos5 := pos
			// action
			// u:Unary
			{
				pos7 := pos
				// Unary
				if !_accept(parser, _UnaryAccepts, &pos, &perr) {
					goto fail6
				}
				labels[0] = parser.text[pos7:pos]
			}
			goto ok2
		fail6:
			pos = pos5
			// Primary
			if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
				goto fail8
			}
			goto ok2
		fail8:
			pos = pos5
			goto fail
		ok2:
		}
		labels[1] = parser.text[pos1:pos]
	}
	// m:BinMsg
	{
		pos9 := pos
		// BinMsg
		if !_accept(parser, _BinMsgAccepts, &pos, &perr) {
			goto fail
		}
		labels[2] = parser.text[pos9:pos]
	}
	return _memoize(parser, _Binary, start, pos, perr)
fail:
	return _memoize(parser, _Binary, start, -1, perr)
}

func _BinaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Binary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Binary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Binary}
	// action
	// r:(u:Unary {…}/Primary) m:BinMsg
	// r:(u:Unary {…}/Primary)
	{
		pos1 := pos
		// (u:Unary {…}/Primary)
		// u:Unary {…}/Primary
		{
			pos5 := pos
			// action
			// u:Unary
			{
				pos7 := pos
				// Unary
				if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
					goto fail6
				}
				labels[0] = parser.text[pos7:pos]
			}
			goto ok2
		fail6:
			pos = pos5
			// Primary
			if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
				goto fail8
			}
			goto ok2
		fail8:
			pos = pos5
			goto fail
		ok2:
		}
		labels[1] = parser.text[pos1:pos]
	}
	// m:BinMsg
	{
		pos9 := pos
		// BinMsg
		if !_fail(parser, _BinMsgFail, errPos, failure, &pos) {
			goto fail
		}
		labels[2] = parser.text[pos9:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _BinaryAction(parser *_Parser, start int) (int, **Call) {
	var labels [3]string
	use(labels)
	var label0 *Call
	var label1 Expr
	var label2 Msg
	dp := parser.deltaPos[start][_Binary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Binary}
	n := parser.act[key]
	if n != nil {
		n := n.(*Call)
		return start + int(dp-1), &n
	}
	var node *Call
	pos := start
	// action
	{
		start0 := pos
		// r:(u:Unary {…}/Primary) m:BinMsg
		// r:(u:Unary {…}/Primary)
		{
			pos2 := pos
			// (u:Unary {…}/Primary)
			// u:Unary {…}/Primary
			{
				pos6 := pos
				var node5 Expr
				// action
				{
					start8 := pos
					// u:Unary
					{
						pos9 := pos
						// Unary
						if p, n := _UnaryAction(parser, pos); n == nil {
							goto fail7
						} else {
							label0 = *n
							pos = p
						}
						labels[0] = parser.text[pos9:pos]
					}
					label1 = func(
						start, end int, u *Call) Expr {
						return Expr(u)
					}(
						start8, pos, label0)
				}
				goto ok3
			fail7:
				label1 = node5
				pos = pos6
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail10
				} else {
					label1 = *n
					pos = p
				}
				goto ok3
			fail10:
				label1 = node5
				pos = pos6
				goto fail
			ok3:
			}
			labels[1] = parser.text[pos2:pos]
		}
		// m:BinMsg
		{
			pos11 := pos
			// BinMsg
			if p, n := _BinMsgAction(parser, pos); n == nil {
				goto fail
			} else {
				label2 = *n
				pos = p
			}
			labels[2] = parser.text[pos11:pos]
		}
		node = func(
			start, end int, m Msg, r Expr, u *Call) *Call {
			s, _ := r.loc()
			return &Call{
				location: location{s, loc1(parser, end)},
				Recv:     r,
				Msgs:     []Msg{m},
			}
		}(
			start0, pos, label2, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BinMsgAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [5]string
	use(labels)
	if dp, de, ok := _memo(parser, _BinMsg, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// mod:ModName? n:Op a:(b:Binary {…}/u:Unary {…}/Primary)
	// mod:ModName?
	{
		pos1 := pos
		// ModName?
		{
			pos3 := pos
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:Op
	{
		pos6 := pos
		// Op
		if !_accept(parser, _OpAccepts, &pos, &perr) {
			goto fail
		}
		labels[1] = parser.text[pos6:pos]
	}
	// a:(b:Binary {…}/u:Unary {…}/Primary)
	{
		pos7 := pos
		// (b:Binary {…}/u:Unary {…}/Primary)
		// b:Binary {…}/u:Unary {…}/Primary
		{
			pos11 := pos
			// action
			// b:Binary
			{
				pos13 := pos
				// Binary
				if !_accept(parser, _BinaryAccepts, &pos, &perr) {
					goto fail12
				}
				labels[2] = parser.text[pos13:pos]
			}
			goto ok8
		fail12:
			pos = pos11
			// action
			// u:Unary
			{
				pos15 := pos
				// Unary
				if !_accept(parser, _UnaryAccepts, &pos, &perr) {
					goto fail14
				}
				labels[3] = parser.text[pos15:pos]
			}
			goto ok8
		fail14:
			pos = pos11
			// Primary
			if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
				goto fail16
			}
			goto ok8
		fail16:
			pos = pos11
			goto fail
		ok8:
		}
		labels[4] = parser.text[pos7:pos]
	}
	return _memoize(parser, _BinMsg, start, pos, perr)
fail:
	return _memoize(parser, _BinMsg, start, -1, perr)
}

func _BinMsgFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [5]string
	use(labels)
	pos, failure := _failMemo(parser, _BinMsg, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "BinMsg",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _BinMsg}
	// action
	// mod:ModName? n:Op a:(b:Binary {…}/u:Unary {…}/Primary)
	// mod:ModName?
	{
		pos1 := pos
		// ModName?
		{
			pos3 := pos
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:Op
	{
		pos6 := pos
		// Op
		if !_fail(parser, _OpFail, errPos, failure, &pos) {
			goto fail
		}
		labels[1] = parser.text[pos6:pos]
	}
	// a:(b:Binary {…}/u:Unary {…}/Primary)
	{
		pos7 := pos
		// (b:Binary {…}/u:Unary {…}/Primary)
		// b:Binary {…}/u:Unary {…}/Primary
		{
			pos11 := pos
			// action
			// b:Binary
			{
				pos13 := pos
				// Binary
				if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
					goto fail12
				}
				labels[2] = parser.text[pos13:pos]
			}
			goto ok8
		fail12:
			pos = pos11
			// action
			// u:Unary
			{
				pos15 := pos
				// Unary
				if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
					goto fail14
				}
				labels[3] = parser.text[pos15:pos]
			}
			goto ok8
		fail14:
			pos = pos11
			// Primary
			if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
				goto fail16
			}
			goto ok8
		fail16:
			pos = pos11
			goto fail
		ok8:
		}
		labels[4] = parser.text[pos7:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _BinMsgAction(parser *_Parser, start int) (int, *Msg) {
	var labels [5]string
	use(labels)
	var label0 *Ident
	var label1 Ident
	var label2 *Call
	var label3 *Call
	var label4 Expr
	dp := parser.deltaPos[start][_BinMsg]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _BinMsg}
	n := parser.act[key]
	if n != nil {
		n := n.(Msg)
		return start + int(dp-1), &n
	}
	var node Msg
	pos := start
	// action
	{
		start0 := pos
		// mod:ModName? n:Op a:(b:Binary {…}/u:Unary {…}/Primary)
		// mod:ModName?
		{
			pos2 := pos
			// ModName?
			{
				pos4 := pos
				label0 = new(Ident)
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail5
				} else {
					*label0 = *n
					pos = p
				}
				goto ok6
			fail5:
				label0 = nil
				pos = pos4
			ok6:
			}
			labels[0] = parser.text[pos2:pos]
		}
		// n:Op
		{
			pos7 := pos
			// Op
			if p, n := _OpAction(parser, pos); n == nil {
				goto fail
			} else {
				label1 = *n
				pos = p
			}
			labels[1] = parser.text[pos7:pos]
		}
		// a:(b:Binary {…}/u:Unary {…}/Primary)
		{
			pos8 := pos
			// (b:Binary {…}/u:Unary {…}/Primary)
			// b:Binary {…}/u:Unary {…}/Primary
			{
				pos12 := pos
				var node11 Expr
				// action
				{
					start14 := pos
					// b:Binary
					{
						pos15 := pos
						// Binary
						if p, n := _BinaryAction(parser, pos); n == nil {
							goto fail13
						} else {
							label2 = *n
							pos = p
						}
						labels[2] = parser.text[pos15:pos]
					}
					label4 = func(
						start, end int, b *Call, mod *Ident, n Ident) Expr {
						return Expr(b)
					}(
						start14, pos, label2, label0, label1)
				}
				goto ok9
			fail13:
				label4 = node11
				pos = pos12
				// action
				{
					start17 := pos
					// u:Unary
					{
						pos18 := pos
						// Unary
						if p, n := _UnaryAction(parser, pos); n == nil {
							goto fail16
						} else {
							label3 = *n
							pos = p
						}
						labels[3] = parser.text[pos18:pos]
					}
					label4 = func(
						start, end int, b *Call, mod *Ident, n Ident, u *Call) Expr {
						return Expr(u)
					}(
						start17, pos, label2, label0, label1, label3)
				}
				goto ok9
			fail16:
				label4 = node11
				pos = pos12
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail19
				} else {
					label4 = *n
					pos = p
				}
				goto ok9
			fail19:
				label4 = node11
				pos = pos12
				goto fail
			ok9:
			}
			labels[4] = parser.text[pos8:pos]
		}
		node = func(
			start, end int, a Expr, b *Call, mod *Ident, n Ident, u *Call) Msg {
			return Msg{
				location: location{n.start, loc1(parser, end)},
				Mod:      mod,
				Sel:      n.Text,
				Args:     []Expr{a},
			}
		}(
			start0, pos, label4, label2, label0, label1, label3)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _NaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Nary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// r:(b:Binary {…}/u:Unary {…}/Primary)? m:NaryMsg
	// r:(b:Binary {…}/u:Unary {…}/Primary)?
	{
		pos1 := pos
		// (b:Binary {…}/u:Unary {…}/Primary)?
		{
			pos3 := pos
			// (b:Binary {…}/u:Unary {…}/Primary)
			// b:Binary {…}/u:Unary {…}/Primary
			{
				pos8 := pos
				// action
				// b:Binary
				{
					pos10 := pos
					// Binary
					if !_accept(parser, _BinaryAccepts, &pos, &perr) {
						goto fail9
					}
					labels[0] = parser.text[pos10:pos]
				}
				goto ok5
			fail9:
				pos = pos8
				// action
				// u:Unary
				{
					pos12 := pos
					// Unary
					if !_accept(parser, _UnaryAccepts, &pos, &perr) {
						goto fail11
					}
					labels[1] = parser.text[pos12:pos]
				}
				goto ok5
			fail11:
				pos = pos8
				// Primary
				if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
					goto fail13
				}
				goto ok5
			fail13:
				pos = pos8
				goto fail4
			ok5:
			}
			goto ok14
		fail4:
			pos = pos3
		ok14:
		}
		labels[2] = parser.text[pos1:pos]
	}
	// m:NaryMsg
	{
		pos15 := pos
		// NaryMsg
		if !_accept(parser, _NaryMsgAccepts, &pos, &perr) {
			goto fail
		}
		labels[3] = parser.text[pos15:pos]
	}
	return _memoize(parser, _Nary, start, pos, perr)
fail:
	return _memoize(parser, _Nary, start, -1, perr)
}

func _NaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
	use(labels)
	pos, failure := _failMemo(parser, _Nary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Nary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Nary}
	// action
	// r:(b:Binary {…}/u:Unary {…}/Primary)? m:NaryMsg
	// r:(b:Binary {…}/u:Unary {…}/Primary)?
	{
		pos1 := pos
		// (b:Binary {…}/u:Unary {…}/Primary)?
		{
			pos3 := pos
			// (b:Binary {…}/u:Unary {…}/Primary)
			// b:Binary {…}/u:Unary {…}/Primary
			{
				pos8 := pos
				// action
				// b:Binary
				{
					pos10 := pos
					// Binary
					if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
						goto fail9
					}
					labels[0] = parser.text[pos10:pos]
				}
				goto ok5
			fail9:
				pos = pos8
				// action
				// u:Unary
				{
					pos12 := pos
					// Unary
					if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
						goto fail11
					}
					labels[1] = parser.text[pos12:pos]
				}
				goto ok5
			fail11:
				pos = pos8
				// Primary
				if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
					goto fail13
				}
				goto ok5
			fail13:
				pos = pos8
				goto fail4
			ok5:
			}
			goto ok14
		fail4:
			pos = pos3
		ok14:
		}
		labels[2] = parser.text[pos1:pos]
	}
	// m:NaryMsg
	{
		pos15 := pos
		// NaryMsg
		if !_fail(parser, _NaryMsgFail, errPos, failure, &pos) {
			goto fail
		}
		labels[3] = parser.text[pos15:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _NaryAction(parser *_Parser, start int) (int, **Call) {
	var labels [4]string
	use(labels)
	var label0 *Call
	var label1 *Call
	var label2 *Expr
	var label3 Msg
	dp := parser.deltaPos[start][_Nary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Nary}
	n := parser.act[key]
	if n != nil {
		n := n.(*Call)
		return start + int(dp-1), &n
	}
	var node *Call
	pos := start
	// action
	{
		start0 := pos
		// r:(b:Binary {…}/u:Unary {…}/Primary)? m:NaryMsg
		// r:(b:Binary {…}/u:Unary {…}/Primary)?
		{
			pos2 := pos
			// (b:Binary {…}/u:Unary {…}/Primary)?
			{
				pos4 := pos
				label2 = new(Expr)
				// (b:Binary {…}/u:Unary {…}/Primary)
				// b:Binary {…}/u:Unary {…}/Primary
				{
					pos9 := pos
					var node8 Expr
					// action
					{
						start11 := pos
						// b:Binary
						{
							pos12 := pos
							// Binary
							if p, n := _BinaryAction(parser, pos); n == nil {
								goto fail10
							} else {
								label0 = *n
								pos = p
							}
							labels[0] = parser.text[pos12:pos]
						}
						*label2 = func(
							start, end int, b *Call) Expr {
							return Expr(b)
						}(
							start11, pos, label0)
					}
					goto ok6
				fail10:
					*label2 = node8
					pos = pos9
					// action
					{
						start14 := pos
						// u:Unary
						{
							pos15 := pos
							// Unary
							if p, n := _UnaryAction(parser, pos); n == nil {
								goto fail13
							} else {
								label1 = *n
								pos = p
							}
							labels[1] = parser.text[pos15:pos]
						}
						*label2 = func(
							start, end int, b *Call, u *Call) Expr {
							return Expr(u)
						}(
							start14, pos, label0, label1)
					}
					goto ok6
				fail13:
					*label2 = node8
					pos = pos9
					// Primary
					if p, n := _PrimaryAction(parser, pos); n == nil {
						goto fail16
					} else {
						*label2 = *n
						pos = p
					}
					goto ok6
				fail16:
					*label2 = node8
					pos = pos9
					goto fail5
				ok6:
				}
				goto ok17
			fail5:
				label2 = nil
				pos = pos4
			ok17:
			}
			labels[2] = parser.text[pos2:pos]
		}
		// m:NaryMsg
		{
			pos18 := pos
			// NaryMsg
			if p, n := _NaryMsgAction(parser, pos); n == nil {
				goto fail
			} else {
				label3 = *n
				pos = p
			}
			labels[3] = parser.text[pos18:pos]
		}
		node = func(
			start, end int, b *Call, m Msg, r *Expr, u *Call) *Call {
			s := m.start
			var recv Expr
			if r != nil {
				s, _ = (*r).loc()
				recv = *r
			}
			return &Call{
				location: location{s, loc1(parser, end)},
				Recv:     recv,
				Msgs:     []Msg{m},
			}
		}(
			start0, pos, label0, label3, label2, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _NaryMsgAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [6]string
	use(labels)
	if dp, de, ok := _memo(parser, _NaryMsg, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// mod:ModName? as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
	// mod:ModName?
	{
		pos1 := pos
		// ModName?
		{
			pos3 := pos
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
	{
		pos6 := pos
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
		// action
		// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
		// n:IdentC
		{
			pos12 := pos
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail
			}
			labels[1] = parser.text[pos12:pos]
		}
		// v:(b:Binary {…}/u:Unary {…}/Primary)
		{
			pos13 := pos
			// (b:Binary {…}/u:Unary {…}/Primary)
			// b:Binary {…}/u:Unary {…}/Primary
			{
				pos17 := pos
				// action
				// b:Binary
				{
					pos19 := pos
					// Binary
					if !_accept(parser, _BinaryAccepts, &pos, &perr) {
						goto fail18
					}
					labels[2] = parser.text[pos19:pos]
				}
				goto ok14
			fail18:
				pos = pos17
				// action
				// u:Unary
				{
					pos21 := pos
					// Unary
					if !_accept(parser, _UnaryAccepts, &pos, &perr) {
						goto fail20
					}
					labels[3] = parser.text[pos21:pos]
				}
				goto ok14
			fail20:
				pos = pos17
				// Primary
				if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
					goto fail22
				}
				goto ok14
			fail22:
				pos = pos17
				goto fail
			ok14:
			}
			labels[4] = parser.text[pos13:pos]
		}
		for {
			pos8 := pos
			// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
			// action
			// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
			// n:IdentC
			{
				pos24 := pos
				// IdentC
				if !_accept(parser, _IdentCAccepts, &pos, &perr) {
					goto fail10
				}
				labels[1] = parser.text[pos24:pos]
			}
			// v:(b:Binary {…}/u:Unary {…}/Primary)
			{
				pos25 := pos
				// (b:Binary {…}/u:Unary {…}/Primary)
				// b:Binary {…}/u:Unary {…}/Primary
				{
					pos29 := pos
					// action
					// b:Binary
					{
						pos31 := pos
						// Binary
						if !_accept(parser, _BinaryAccepts, &pos, &perr) {
							goto fail30
						}
						labels[2] = parser.text[pos31:pos]
					}
					goto ok26
				fail30:
					pos = pos29
					// action
					// u:Unary
					{
						pos33 := pos
						// Unary
						if !_accept(parser, _UnaryAccepts, &pos, &perr) {
							goto fail32
						}
						labels[3] = parser.text[pos33:pos]
					}
					goto ok26
				fail32:
					pos = pos29
					// Primary
					if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
						goto fail34
					}
					goto ok26
				fail34:
					pos = pos29
					goto fail10
				ok26:
				}
				labels[4] = parser.text[pos25:pos]
			}
			continue
		fail10:
			pos = pos8
			break
		}
		labels[5] = parser.text[pos6:pos]
	}
	return _memoize(parser, _NaryMsg, start, pos, perr)
fail:
	return _memoize(parser, _NaryMsg, start, -1, perr)
}

func _NaryMsgFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [6]string
	use(labels)
	pos, failure := _failMemo(parser, _NaryMsg, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "NaryMsg",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _NaryMsg}
	// action
	// mod:ModName? as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
	// mod:ModName?
	{
		pos1 := pos
		// ModName?
		{
			pos3 := pos
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
	{
		pos6 := pos
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
		// action
		// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
		// n:IdentC
		{
			pos12 := pos
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail
			}
			labels[1] = parser.text[pos12:pos]
		}
		// v:(b:Binary {…}/u:Unary {…}/Primary)
		{
			pos13 := pos
			// (b:Binary {…}/u:Unary {…}/Primary)
			// b:Binary {…}/u:Unary {…}/Primary
			{
				pos17 := pos
				// action
				// b:Binary
				{
					pos19 := pos
					// Binary
					if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
						goto fail18
					}
					labels[2] = parser.text[pos19:pos]
				}
				goto ok14
			fail18:
				pos = pos17
				// action
				// u:Unary
				{
					pos21 := pos
					// Unary
					if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
						goto fail20
					}
					labels[3] = parser.text[pos21:pos]
				}
				goto ok14
			fail20:
				pos = pos17
				// Primary
				if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
					goto fail22
				}
				goto ok14
			fail22:
				pos = pos17
				goto fail
			ok14:
			}
			labels[4] = parser.text[pos13:pos]
		}
		for {
			pos8 := pos
			// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
			// action
			// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
			// n:IdentC
			{
				pos24 := pos
				// IdentC
				if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
					goto fail10
				}
				labels[1] = parser.text[pos24:pos]
			}
			// v:(b:Binary {…}/u:Unary {…}/Primary)
			{
				pos25 := pos
				// (b:Binary {…}/u:Unary {…}/Primary)
				// b:Binary {…}/u:Unary {…}/Primary
				{
					pos29 := pos
					// action
					// b:Binary
					{
						pos31 := pos
						// Binary
						if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
							goto fail30
						}
						labels[2] = parser.text[pos31:pos]
					}
					goto ok26
				fail30:
					pos = pos29
					// action
					// u:Unary
					{
						pos33 := pos
						// Unary
						if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
							goto fail32
						}
						labels[3] = parser.text[pos33:pos]
					}
					goto ok26
				fail32:
					pos = pos29
					// Primary
					if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
						goto fail34
					}
					goto ok26
				fail34:
					pos = pos29
					goto fail10
				ok26:
				}
				labels[4] = parser.text[pos25:pos]
			}
			continue
		fail10:
			pos = pos8
			break
		}
		labels[5] = parser.text[pos6:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _NaryMsgAction(parser *_Parser, start int) (int, *Msg) {
	var labels [6]string
	use(labels)
	var label0 *Ident
	var label1 Ident
	var label2 *Call
	var label3 *Call
	var label4 Expr
	var label5 []arg
	dp := parser.deltaPos[start][_NaryMsg]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _NaryMsg}
	n := parser.act[key]
	if n != nil {
		n := n.(Msg)
		return start + int(dp-1), &n
	}
	var node Msg
	pos := start
	// action
	{
		start0 := pos
		// mod:ModName? as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
		// mod:ModName?
		{
			pos2 := pos
			// ModName?
			{
				pos4 := pos
				label0 = new(Ident)
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail5
				} else {
					*label0 = *n
					pos = p
				}
				goto ok6
			fail5:
				label0 = nil
				pos = pos4
			ok6:
			}
			labels[0] = parser.text[pos2:pos]
		}
		// as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
		{
			pos7 := pos
			// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
			{
				var node10 arg
				// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
				// action
				{
					start12 := pos
					// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
					// n:IdentC
					{
						pos14 := pos
						// IdentC
						if p, n := _IdentCAction(parser, pos); n == nil {
							goto fail
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos14:pos]
					}
					// v:(b:Binary {…}/u:Unary {…}/Primary)
					{
						pos15 := pos
						// (b:Binary {…}/u:Unary {…}/Primary)
						// b:Binary {…}/u:Unary {…}/Primary
						{
							pos19 := pos
							var node18 Expr
							// action
							{
								start21 := pos
								// b:Binary
								{
									pos22 := pos
									// Binary
									if p, n := _BinaryAction(parser, pos); n == nil {
										goto fail20
									} else {
										label2 = *n
										pos = p
									}
									labels[2] = parser.text[pos22:pos]
								}
								label4 = func(
									start, end int, b *Call, mod *Ident, n Ident) Expr {
									return Expr(b)
								}(
									start21, pos, label2, label0, label1)
							}
							goto ok16
						fail20:
							label4 = node18
							pos = pos19
							// action
							{
								start24 := pos
								// u:Unary
								{
									pos25 := pos
									// Unary
									if p, n := _UnaryAction(parser, pos); n == nil {
										goto fail23
									} else {
										label3 = *n
										pos = p
									}
									labels[3] = parser.text[pos25:pos]
								}
								label4 = func(
									start, end int, b *Call, mod *Ident, n Ident, u *Call) Expr {
									return Expr(u)
								}(
									start24, pos, label2, label0, label1, label3)
							}
							goto ok16
						fail23:
							label4 = node18
							pos = pos19
							// Primary
							if p, n := _PrimaryAction(parser, pos); n == nil {
								goto fail26
							} else {
								label4 = *n
								pos = p
							}
							goto ok16
						fail26:
							label4 = node18
							pos = pos19
							goto fail
						ok16:
						}
						labels[4] = parser.text[pos15:pos]
					}
					node10 = func(
						start, end int, b *Call, mod *Ident, n Ident, u *Call, v Expr) arg {
						return arg{n, v}
					}(
						start12, pos, label2, label0, label1, label3, label4)
				}
				label5 = append(label5, node10)
			}
			for {
				pos9 := pos
				var node10 arg
				// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
				// action
				{
					start27 := pos
					// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
					// n:IdentC
					{
						pos29 := pos
						// IdentC
						if p, n := _IdentCAction(parser, pos); n == nil {
							goto fail11
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos29:pos]
					}
					// v:(b:Binary {…}/u:Unary {…}/Primary)
					{
						pos30 := pos
						// (b:Binary {…}/u:Unary {…}/Primary)
						// b:Binary {…}/u:Unary {…}/Primary
						{
							pos34 := pos
							var node33 Expr
							// action
							{
								start36 := pos
								// b:Binary
								{
									pos37 := pos
									// Binary
									if p, n := _BinaryAction(parser, pos); n == nil {
										goto fail35
									} else {
										label2 = *n
										pos = p
									}
									labels[2] = parser.text[pos37:pos]
								}
								label4 = func(
									start, end int, b *Call, mod *Ident, n Ident) Expr {
									return Expr(b)
								}(
									start36, pos, label2, label0, label1)
							}
							goto ok31
						fail35:
							label4 = node33
							pos = pos34
							// action
							{
								start39 := pos
								// u:Unary
								{
									pos40 := pos
									// Unary
									if p, n := _UnaryAction(parser, pos); n == nil {
										goto fail38
									} else {
										label3 = *n
										pos = p
									}
									labels[3] = parser.text[pos40:pos]
								}
								label4 = func(
									start, end int, b *Call, mod *Ident, n Ident, u *Call) Expr {
									return Expr(u)
								}(
									start39, pos, label2, label0, label1, label3)
							}
							goto ok31
						fail38:
							label4 = node33
							pos = pos34
							// Primary
							if p, n := _PrimaryAction(parser, pos); n == nil {
								goto fail41
							} else {
								label4 = *n
								pos = p
							}
							goto ok31
						fail41:
							label4 = node33
							pos = pos34
							goto fail11
						ok31:
						}
						labels[4] = parser.text[pos30:pos]
					}
					node10 = func(
						start, end int, b *Call, mod *Ident, n Ident, u *Call, v Expr) arg {
						return arg{n, v}
					}(
						start27, pos, label2, label0, label1, label3, label4)
				}
				label5 = append(label5, node10)
				continue
			fail11:
				pos = pos9
				break
			}
			labels[5] = parser.text[pos7:pos]
		}
		node = func(
			start, end int, as []arg, b *Call, mod *Ident, n Ident, u *Call, v Expr) Msg {
			var sel string
			var es []Expr
			for _, a := range as {
				sel += a.name.Text
				es = append(es, a.val)
			}
			return Msg{
				location: location{as[0].name.start, loc1(parser, end)},
				Mod:      mod,
				Sel:      sel,
				Args:     es,
			}
		}(
			start0, pos, label5, label2, label0, label1, label3, label4)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _PrimaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Primary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// i:Ident {…}/Float/Int/Rune/s:String {…}/Ctor/Block/_ "(" e:Expr _ ")" {…}
	{
		pos3 := pos
		// action
		// i:Ident
		{
			pos5 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// Float
		if !_accept(parser, _FloatAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		// Int
		if !_accept(parser, _IntAccepts, &pos, &perr) {
			goto fail7
		}
		goto ok0
	fail7:
		pos = pos3
		// Rune
		if !_accept(parser, _RuneAccepts, &pos, &perr) {
			goto fail8
		}
		goto ok0
	fail8:
		pos = pos3
		// action
		// s:String
		{
			pos10 := pos
			// String
			if !_accept(parser, _StringAccepts, &pos, &perr) {
				goto fail9
			}
			labels[1] = parser.text[pos10:pos]
		}
		goto ok0
	fail9:
		pos = pos3
		// Ctor
		if !_accept(parser, _CtorAccepts, &pos, &perr) {
			goto fail11
		}
		goto ok0
	fail11:
		pos = pos3
		// Block
		if !_accept(parser, _BlockAccepts, &pos, &perr) {
			goto fail12
		}
		goto ok0
	fail12:
		pos = pos3
		// action
		// _ "(" e:Expr _ ")"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail13
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail13
		}
		pos++
		// e:Expr
		{
			pos15 := pos
			// Expr
			if !_accept(parser, _ExprAccepts, &pos, &perr) {
				goto fail13
			}
			labels[2] = parser.text[pos15:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail13
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			perr = _max(perr, pos)
			goto fail13
		}
		pos++
		goto ok0
	fail13:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Primary, start, pos, perr)
fail:
	return _memoize(parser, _Primary, start, -1, perr)
}

func _PrimaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Primary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Primary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Primary}
	// i:Ident {…}/Float/Int/Rune/s:String {…}/Ctor/Block/_ "(" e:Expr _ ")" {…}
	{
		pos3 := pos
		// action
		// i:Ident
		{
			pos5 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// Float
		if !_fail(parser, _FloatFail, errPos, failure, &pos) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		// Int
		if !_fail(parser, _IntFail, errPos, failure, &pos) {
			goto fail7
		}
		goto ok0
	fail7:
		pos = pos3
		// Rune
		if !_fail(parser, _RuneFail, errPos, failure, &pos) {
			goto fail8
		}
		goto ok0
	fail8:
		pos = pos3
		// action
		// s:String
		{
			pos10 := pos
			// String
			if !_fail(parser, _StringFail, errPos, failure, &pos) {
				goto fail9
			}
			labels[1] = parser.text[pos10:pos]
		}
		goto ok0
	fail9:
		pos = pos3
		// Ctor
		if !_fail(parser, _CtorFail, errPos, failure, &pos) {
			goto fail11
		}
		goto ok0
	fail11:
		pos = pos3
		// Block
		if !_fail(parser, _BlockFail, errPos, failure, &pos) {
			goto fail12
		}
		goto ok0
	fail12:
		pos = pos3
		// action
		// _ "(" e:Expr _ ")"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail13
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail13
		}
		pos++
		// e:Expr
		{
			pos15 := pos
			// Expr
			if !_fail(parser, _ExprFail, errPos, failure, &pos) {
				goto fail13
			}
			labels[2] = parser.text[pos15:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail13
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
				})
			}
			goto fail13
		}
		pos++
		goto ok0
	fail13:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _PrimaryAction(parser *_Parser, start int) (int, *Expr) {
	var labels [3]string
	use(labels)
	var label0 Ident
	var label1 String
	var label2 Expr
	dp := parser.deltaPos[start][_Primary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Primary}
	n := parser.act[key]
	if n != nil {
		n := n.(Expr)
		return start + int(dp-1), &n
	}
	var node Expr
	pos := start
	// i:Ident {…}/Float/Int/Rune/s:String {…}/Ctor/Block/_ "(" e:Expr _ ")" {…}
	{
		pos3 := pos
		var node2 Expr
		// action
		{
			start5 := pos
			// i:Ident
			{
				pos6 := pos
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail4
				} else {
					label0 = *n
					pos = p
				}
				labels[0] = parser.text[pos6:pos]
			}
			node = func(
				start, end int, i Ident) Expr {
				return Expr(&i)
			}(
				start5, pos, label0)
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Float
		if p, n := _FloatAction(parser, pos); n == nil {
			goto fail7
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail7:
		node = node2
		pos = pos3
		// Int
		if p, n := _IntAction(parser, pos); n == nil {
			goto fail8
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail8:
		node = node2
		pos = pos3
		// Rune
		if p, n := _RuneAction(parser, pos); n == nil {
			goto fail9
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail9:
		node = node2
		pos = pos3
		// action
		{
			start11 := pos
			// s:String
			{
				pos12 := pos
				// String
				if p, n := _StringAction(parser, pos); n == nil {
					goto fail10
				} else {
					label1 = *n
					pos = p
				}
				labels[1] = parser.text[pos12:pos]
			}
			node = func(
				start, end int, i Ident, s String) Expr {
				return Expr(&s)
			}(
				start11, pos, label0, label1)
		}
		goto ok0
	fail10:
		node = node2
		pos = pos3
		// Ctor
		if p, n := _CtorAction(parser, pos); n == nil {
			goto fail13
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail13:
		node = node2
		pos = pos3
		// Block
		if p, n := _BlockAction(parser, pos); n == nil {
			goto fail14
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail14:
		node = node2
		pos = pos3
		// action
		{
			start16 := pos
			// _ "(" e:Expr _ ")"
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail15
			} else {
				pos = p
			}
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				goto fail15
			}
			pos++
			// e:Expr
			{
				pos18 := pos
				// Expr
				if p, n := _ExprAction(parser, pos); n == nil {
					goto fail15
				} else {
					label2 = *n
					pos = p
				}
				labels[2] = parser.text[pos18:pos]
			}
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail15
			} else {
				pos = p
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				goto fail15
			}
			pos++
			node = func(
				start, end int, e Expr, i Ident, s String) Expr {
				return Expr(e)
			}(
				start16, pos, label2, label0, label1)
		}
		goto ok0
	fail15:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _CtorAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Ctor, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ ctor:("{" es:Exprs? _ "}" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// ctor:("{" es:Exprs? _ "}" {…})
	{
		pos1 := pos
		// ("{" es:Exprs? _ "}" {…})
		// action
		// "{" es:Exprs? _ "}"
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// es:Exprs?
		{
			pos3 := pos
			// Exprs?
			{
				pos5 := pos
				// Exprs
				if !_accept(parser, _ExprsAccepts, &pos, &perr) {
					goto fail6
				}
				goto ok7
			fail6:
				pos = pos5
			ok7:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "}"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		labels[1] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Ctor, start, pos, perr)
fail:
	return _memoize(parser, _Ctor, start, -1, perr)
}

func _CtorFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Ctor, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Ctor",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Ctor}
	// action
	// _ ctor:("{" es:Exprs? _ "}" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// ctor:("{" es:Exprs? _ "}" {…})
	{
		pos1 := pos
		// ("{" es:Exprs? _ "}" {…})
		// action
		// "{" es:Exprs? _ "}"
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"{\"",
				})
			}
			goto fail
		}
		pos++
		// es:Exprs?
		{
			pos3 := pos
			// Exprs?
			{
				pos5 := pos
				// Exprs
				if !_fail(parser, _ExprsFail, errPos, failure, &pos) {
					goto fail6
				}
				goto ok7
			fail6:
				pos = pos5
			ok7:
			}
			labels[0] = parser.text[pos3:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "}"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"}\"",
				})
			}
			goto fail
		}
		pos++
		labels[1] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _CtorAction(parser *_Parser, start int) (int, *Expr) {
	var labels [2]string
	use(labels)
	var label0 *([]Expr)
	var label1 *Ctor
	dp := parser.deltaPos[start][_Ctor]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ctor}
	n := parser.act[key]
	if n != nil {
		n := n.(Expr)
		return start + int(dp-1), &n
	}
	var node Expr
	pos := start
	// action
	{
		start0 := pos
		// _ ctor:("{" es:Exprs? _ "}" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// ctor:("{" es:Exprs? _ "}" {…})
		{
			pos2 := pos
			// ("{" es:Exprs? _ "}" {…})
			// action
			{
				start3 := pos
				// "{" es:Exprs? _ "}"
				// "{"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
					goto fail
				}
				pos++
				// es:Exprs?
				{
					pos5 := pos
					// Exprs?
					{
						pos7 := pos
						label0 = new(([]Expr))
						// Exprs
						if p, n := _ExprsAction(parser, pos); n == nil {
							goto fail8
						} else {
							*label0 = *n
							pos = p
						}
						goto ok9
					fail8:
						label0 = nil
						pos = pos7
					ok9:
					}
					labels[0] = parser.text[pos5:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "}"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
					goto fail
				}
				pos++
				label1 = func(
					start, end int, es *([]Expr)) *Ctor {
					if es == nil {
						return &Ctor{location: loc(parser, start, end)}
					}
					return &Ctor{location: loc(parser, start, end), Args: *es}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, ctor *Ctor, es *([]Expr)) Expr {
			return Expr(ctor)
		}(
			start0, pos, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ExprsAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Exprs, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// e0:Expr es:(_ ";" e:Expr {…})* (_ ";")?
	// e0:Expr
	{
		pos1 := pos
		// Expr
		if !_accept(parser, _ExprAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// es:(_ ";" e:Expr {…})*
	{
		pos2 := pos
		// (_ ";" e:Expr {…})*
		for {
			pos4 := pos
			// (_ ";" e:Expr {…})
			// action
			// _ ";" e:Expr
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail6
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
				perr = _max(perr, pos)
				goto fail6
			}
			pos++
			// e:Expr
			{
				pos8 := pos
				// Expr
				if !_accept(parser, _ExprAccepts, &pos, &perr) {
					goto fail6
				}
				labels[1] = parser.text[pos8:pos]
			}
			continue
		fail6:
			pos = pos4
			break
		}
		labels[2] = parser.text[pos2:pos]
	}
	// (_ ";")?
	{
		pos10 := pos
		// (_ ";")
		// _ ";"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail11
		}
		// ";"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
			perr = _max(perr, pos)
			goto fail11
		}
		pos++
		goto ok13
	fail11:
		pos = pos10
	ok13:
	}
	return _memoize(parser, _Exprs, start, pos, perr)
fail:
	return _memoize(parser, _Exprs, start, -1, perr)
}

func _ExprsFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Exprs, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Exprs",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Exprs}
	// action
	// e0:Expr es:(_ ";" e:Expr {…})* (_ ";")?
	// e0:Expr
	{
		pos1 := pos
		// Expr
		if !_fail(parser, _ExprFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// es:(_ ";" e:Expr {…})*
	{
		pos2 := pos
		// (_ ";" e:Expr {…})*
		for {
			pos4 := pos
			// (_ ";" e:Expr {…})
			// action
			// _ ";" e:Expr
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail6
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\";\"",
					})
				}
				goto fail6
			}
			pos++
			// e:Expr
			{
				pos8 := pos
				// Expr
				if !_fail(parser, _ExprFail, errPos, failure, &pos) {
					goto fail6
				}
				labels[1] = parser.text[pos8:pos]
			}
			continue
		fail6:
			pos = pos4
			break
		}
		labels[2] = parser.text[pos2:pos]
	}
	// (_ ";")?
	{
		pos10 := pos
		// (_ ";")
		// _ ";"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail11
		}
		// ";"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\";\"",
				})
			}
			goto fail11
		}
		pos++
		goto ok13
	fail11:
		pos = pos10
	ok13:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _ExprsAction(parser *_Parser, start int) (int, *([]Expr)) {
	var labels [3]string
	use(labels)
	var label0 Expr
	var label1 Expr
	var label2 []Expr
	dp := parser.deltaPos[start][_Exprs]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Exprs}
	n := parser.act[key]
	if n != nil {
		n := n.(([]Expr))
		return start + int(dp-1), &n
	}
	var node ([]Expr)
	pos := start
	// action
	{
		start0 := pos
		// e0:Expr es:(_ ";" e:Expr {…})* (_ ";")?
		// e0:Expr
		{
			pos2 := pos
			// Expr
			if p, n := _ExprAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// es:(_ ";" e:Expr {…})*
		{
			pos3 := pos
			// (_ ";" e:Expr {…})*
			for {
				pos5 := pos
				var node6 Expr
				// (_ ";" e:Expr {…})
				// action
				{
					start8 := pos
					// _ ";" e:Expr
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail7
					} else {
						pos = p
					}
					// ";"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
						goto fail7
					}
					pos++
					// e:Expr
					{
						pos10 := pos
						// Expr
						if p, n := _ExprAction(parser, pos); n == nil {
							goto fail7
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos10:pos]
					}
					node6 = func(
						start, end int, e Expr, e0 Expr) Expr {
						return Expr(e)
					}(
						start8, pos, label1, label0)
				}
				label2 = append(label2, node6)
				continue
			fail7:
				pos = pos5
				break
			}
			labels[2] = parser.text[pos3:pos]
		}
		// (_ ";")?
		{
			pos12 := pos
			// (_ ";")
			// _ ";"
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail13
			} else {
				pos = p
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
				goto fail13
			}
			pos++
			goto ok15
		fail13:
			pos = pos12
		ok15:
		}
		node = func(
			start, end int, e Expr, e0 Expr, es []Expr) []Expr {
			return ([]Expr)(append([]Expr{e0}, es...))
		}(
			start0, pos, label1, label0, label2)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BlockAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [5]string
	use(labels)
	if dp, de, ok := _memo(parser, _Block, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ b:("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// b:("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
	{
		pos1 := pos
		// ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
		// action
		// "[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]"
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts)
		// (ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts
		// (ps:(n:CIdent t:TypeName? {…})+ _ "|")?
		{
			pos5 := pos
			// (ps:(n:CIdent t:TypeName? {…})+ _ "|")
			// ps:(n:CIdent t:TypeName? {…})+ _ "|"
			// ps:(n:CIdent t:TypeName? {…})+
			{
				pos8 := pos
				// (n:CIdent t:TypeName? {…})+
				// (n:CIdent t:TypeName? {…})
				// action
				// n:CIdent t:TypeName?
				// n:CIdent
				{
					pos14 := pos
					// CIdent
					if !_accept(parser, _CIdentAccepts, &pos, &perr) {
						goto fail6
					}
					labels[0] = parser.text[pos14:pos]
				}
				// t:TypeName?
				{
					pos15 := pos
					// TypeName?
					{
						pos17 := pos
						// TypeName
						if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
							goto fail18
						}
						goto ok19
					fail18:
						pos = pos17
					ok19:
					}
					labels[1] = parser.text[pos15:pos]
				}
				for {
					pos10 := pos
					// (n:CIdent t:TypeName? {…})
					// action
					// n:CIdent t:TypeName?
					// n:CIdent
					{
						pos21 := pos
						// CIdent
						if !_accept(parser, _CIdentAccepts, &pos, &perr) {
							goto fail12
						}
						labels[0] = parser.text[pos21:pos]
					}
					// t:TypeName?
					{
						pos22 := pos
						// TypeName?
						{
							pos24 := pos
							// TypeName
							if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
								goto fail25
							}
							goto ok26
						fail25:
							pos = pos24
						ok26:
						}
						labels[1] = parser.text[pos22:pos]
					}
					continue
				fail12:
					pos = pos10
					break
				}
				labels[2] = parser.text[pos8:pos]
			}
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail6
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				perr = _max(perr, pos)
				goto fail6
			}
			pos++
			goto ok27
		fail6:
			pos = pos5
		ok27:
		}
		// ss:Stmts
		{
			pos28 := pos
			// Stmts
			if !_accept(parser, _StmtsAccepts, &pos, &perr) {
				goto fail
			}
			labels[3] = parser.text[pos28:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		labels[4] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Block, start, pos, perr)
fail:
	return _memoize(parser, _Block, start, -1, perr)
}

func _BlockFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [5]string
	use(labels)
	pos, failure := _failMemo(parser, _Block, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Block",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Block}
	// action
	// _ b:("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// b:("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
	{
		pos1 := pos
		// ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
		// action
		// "[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]"
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"[\"",
				})
			}
			goto fail
		}
		pos++
		// ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts)
		// (ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts
		// (ps:(n:CIdent t:TypeName? {…})+ _ "|")?
		{
			pos5 := pos
			// (ps:(n:CIdent t:TypeName? {…})+ _ "|")
			// ps:(n:CIdent t:TypeName? {…})+ _ "|"
			// ps:(n:CIdent t:TypeName? {…})+
			{
				pos8 := pos
				// (n:CIdent t:TypeName? {…})+
				// (n:CIdent t:TypeName? {…})
				// action
				// n:CIdent t:TypeName?
				// n:CIdent
				{
					pos14 := pos
					// CIdent
					if !_fail(parser, _CIdentFail, errPos, failure, &pos) {
						goto fail6
					}
					labels[0] = parser.text[pos14:pos]
				}
				// t:TypeName?
				{
					pos15 := pos
					// TypeName?
					{
						pos17 := pos
						// TypeName
						if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
							goto fail18
						}
						goto ok19
					fail18:
						pos = pos17
					ok19:
					}
					labels[1] = parser.text[pos15:pos]
				}
				for {
					pos10 := pos
					// (n:CIdent t:TypeName? {…})
					// action
					// n:CIdent t:TypeName?
					// n:CIdent
					{
						pos21 := pos
						// CIdent
						if !_fail(parser, _CIdentFail, errPos, failure, &pos) {
							goto fail12
						}
						labels[0] = parser.text[pos21:pos]
					}
					// t:TypeName?
					{
						pos22 := pos
						// TypeName?
						{
							pos24 := pos
							// TypeName
							if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
								goto fail25
							}
							goto ok26
						fail25:
							pos = pos24
						ok26:
						}
						labels[1] = parser.text[pos22:pos]
					}
					continue
				fail12:
					pos = pos10
					break
				}
				labels[2] = parser.text[pos8:pos]
			}
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail6
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"|\"",
					})
				}
				goto fail6
			}
			pos++
			goto ok27
		fail6:
			pos = pos5
		ok27:
		}
		// ss:Stmts
		{
			pos28 := pos
			// Stmts
			if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
				goto fail
			}
			labels[3] = parser.text[pos28:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"]\"",
				})
			}
			goto fail
		}
		pos++
		labels[4] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _BlockAction(parser *_Parser, start int) (int, *Expr) {
	var labels [5]string
	use(labels)
	var label0 Ident
	var label1 *TypeName
	var label2 []Var
	var label3 []Stmt
	var label4 *Block
	dp := parser.deltaPos[start][_Block]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Block}
	n := parser.act[key]
	if n != nil {
		n := n.(Expr)
		return start + int(dp-1), &n
	}
	var node Expr
	pos := start
	// action
	{
		start0 := pos
		// _ b:("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// b:("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
		{
			pos2 := pos
			// ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]" {…})
			// action
			{
				start3 := pos
				// "[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]"
				// "["
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
					goto fail
				}
				pos++
				// ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts)
				// (ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts
				// (ps:(n:CIdent t:TypeName? {…})+ _ "|")?
				{
					pos7 := pos
					// (ps:(n:CIdent t:TypeName? {…})+ _ "|")
					// ps:(n:CIdent t:TypeName? {…})+ _ "|"
					// ps:(n:CIdent t:TypeName? {…})+
					{
						pos10 := pos
						// (n:CIdent t:TypeName? {…})+
						{
							var node13 Var
							// (n:CIdent t:TypeName? {…})
							// action
							{
								start15 := pos
								// n:CIdent t:TypeName?
								// n:CIdent
								{
									pos17 := pos
									// CIdent
									if p, n := _CIdentAction(parser, pos); n == nil {
										goto fail8
									} else {
										label0 = *n
										pos = p
									}
									labels[0] = parser.text[pos17:pos]
								}
								// t:TypeName?
								{
									pos18 := pos
									// TypeName?
									{
										pos20 := pos
										label1 = new(TypeName)
										// TypeName
										if p, n := _TypeNameAction(parser, pos); n == nil {
											goto fail21
										} else {
											*label1 = *n
											pos = p
										}
										goto ok22
									fail21:
										label1 = nil
										pos = pos20
									ok22:
									}
									labels[1] = parser.text[pos18:pos]
								}
								node13 = func(
									start, end int, n Ident, t *TypeName) Var {
									return Var{location: loc(parser, start, end), Name: n.Text, Type: t}
								}(
									start15, pos, label0, label1)
							}
							label2 = append(label2, node13)
						}
						for {
							pos12 := pos
							var node13 Var
							// (n:CIdent t:TypeName? {…})
							// action
							{
								start23 := pos
								// n:CIdent t:TypeName?
								// n:CIdent
								{
									pos25 := pos
									// CIdent
									if p, n := _CIdentAction(parser, pos); n == nil {
										goto fail14
									} else {
										label0 = *n
										pos = p
									}
									labels[0] = parser.text[pos25:pos]
								}
								// t:TypeName?
								{
									pos26 := pos
									// TypeName?
									{
										pos28 := pos
										label1 = new(TypeName)
										// TypeName
										if p, n := _TypeNameAction(parser, pos); n == nil {
											goto fail29
										} else {
											*label1 = *n
											pos = p
										}
										goto ok30
									fail29:
										label1 = nil
										pos = pos28
									ok30:
									}
									labels[1] = parser.text[pos26:pos]
								}
								node13 = func(
									start, end int, n Ident, t *TypeName) Var {
									return Var{location: loc(parser, start, end), Name: n.Text, Type: t}
								}(
									start23, pos, label0, label1)
							}
							label2 = append(label2, node13)
							continue
						fail14:
							pos = pos12
							break
						}
						labels[2] = parser.text[pos10:pos]
					}
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail8
					} else {
						pos = p
					}
					// "|"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
						goto fail8
					}
					pos++
					goto ok31
				fail8:
					pos = pos7
				ok31:
				}
				// ss:Stmts
				{
					pos32 := pos
					// Stmts
					if p, n := _StmtsAction(parser, pos); n == nil {
						goto fail
					} else {
						label3 = *n
						pos = p
					}
					labels[3] = parser.text[pos32:pos]
				}
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail
				} else {
					pos = p
				}
				// "]"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
					goto fail
				}
				pos++
				label4 = func(
					start, end int, n Ident, ps []Var, ss []Stmt, t *TypeName) *Block {
					return &Block{
						location: loc(parser, start, end),
						Parms:    ps,
						Stmts:    ss,
					}
				}(
					start3, pos, label0, label2, label3, label1)
			}
			labels[4] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, b *Block, n Ident, ps []Var, ss []Stmt, t *TypeName) Expr {
			return Expr(b)
		}(
			start0, pos, label4, label0, label2, label3, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _IntAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Int, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ tok:(text:([+\-]? [0-9]+) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// tok:(text:([+\-]? [0-9]+) {…})
	{
		pos1 := pos
		// (text:([+\-]? [0-9]+) {…})
		// action
		// text:([+\-]? [0-9]+)
		{
			pos2 := pos
			// ([+\-]? [0-9]+)
			// [+\-]? [0-9]+
			// [+\-]?
			{
				pos5 := pos
				// [+\-]
				if r, w := _next(parser, pos); r != '+' && r != '-' {
					perr = _max(perr, pos)
					goto fail6
				} else {
					pos += w
				}
				goto ok7
			fail6:
				pos = pos5
			ok7:
			}
			// [0-9]+
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			for {
				pos9 := pos
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					perr = _max(perr, pos)
					goto fail11
				} else {
					pos += w
				}
				continue
			fail11:
				pos = pos9
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	perr = start
	return _memoize(parser, _Int, start, pos, perr)
fail:
	return _memoize(parser, _Int, start, -1, perr)
}

func _IntFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Int, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Int",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Int}
	// action
	// _ tok:(text:([+\-]? [0-9]+) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// tok:(text:([+\-]? [0-9]+) {…})
	{
		pos1 := pos
		// (text:([+\-]? [0-9]+) {…})
		// action
		// text:([+\-]? [0-9]+)
		{
			pos2 := pos
			// ([+\-]? [0-9]+)
			// [+\-]? [0-9]+
			// [+\-]?
			{
				pos5 := pos
				// [+\-]
				if r, w := _next(parser, pos); r != '+' && r != '-' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[+\\-]",
						})
					}
					goto fail6
				} else {
					pos += w
				}
				goto ok7
			fail6:
				pos = pos5
			ok7:
			}
			// [0-9]+
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[0-9]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			for {
				pos9 := pos
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[0-9]",
						})
					}
					goto fail11
				} else {
					pos += w
				}
				continue
			fail11:
				pos = pos9
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "integer"
	parser.fail[key] = failure
	return -1, failure
}

func _IntAction(parser *_Parser, start int) (int, *Expr) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 *Int
	dp := parser.deltaPos[start][_Int]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Int}
	n := parser.act[key]
	if n != nil {
		n := n.(Expr)
		return start + int(dp-1), &n
	}
	var node Expr
	pos := start
	// action
	{
		start0 := pos
		// _ tok:(text:([+\-]? [0-9]+) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// tok:(text:([+\-]? [0-9]+) {…})
		{
			pos2 := pos
			// (text:([+\-]? [0-9]+) {…})
			// action
			{
				start3 := pos
				// text:([+\-]? [0-9]+)
				{
					pos4 := pos
					// ([+\-]? [0-9]+)
					// [+\-]? [0-9]+
					{
						var node5 string
						// [+\-]?
						{
							pos7 := pos
							// [+\-]
							if r, w := _next(parser, pos); r != '+' && r != '-' {
								goto fail8
							} else {
								node5 = parser.text[pos : pos+w]
								pos += w
							}
							goto ok9
						fail8:
							node5 = ""
							pos = pos7
						ok9:
						}
						label0, node5 = label0+node5, ""
						// [0-9]+
						{
							var node12 string
							// [0-9]
							if r, w := _next(parser, pos); r < '0' || r > '9' {
								goto fail
							} else {
								node12 = parser.text[pos : pos+w]
								pos += w
							}
							node5 += node12
						}
						for {
							pos11 := pos
							var node12 string
							// [0-9]
							if r, w := _next(parser, pos); r < '0' || r > '9' {
								goto fail13
							} else {
								node12 = parser.text[pos : pos+w]
								pos += w
							}
							node5 += node12
							continue
						fail13:
							pos = pos11
							break
						}
						label0, node5 = label0+node5, ""
					}
					labels[0] = parser.text[pos4:pos]
				}
				label1 = func(
					start, end int, text string) *Int {
					return &Int{location: loc(parser, start, end), Text: text}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, text string, tok *Int) Expr {
			return Expr(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FloatAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Float, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ tok:(text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// tok:(text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
	{
		pos1 := pos
		// (text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
		// action
		// text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?)
		{
			pos2 := pos
			// ([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?)
			// [+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?
			// [+\-]?
			{
				pos5 := pos
				// [+\-]
				if r, w := _next(parser, pos); r != '+' && r != '-' {
					perr = _max(perr, pos)
					goto fail6
				} else {
					pos += w
				}
				goto ok7
			fail6:
				pos = pos5
			ok7:
			}
			// [0-9]+
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			for {
				pos9 := pos
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					perr = _max(perr, pos)
					goto fail11
				} else {
					pos += w
				}
				continue
			fail11:
				pos = pos9
				break
			}
			// "."
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
				perr = _max(perr, pos)
				goto fail
			}
			pos++
			// [0-9]+
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			for {
				pos13 := pos
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					perr = _max(perr, pos)
					goto fail15
				} else {
					pos += w
				}
				continue
			fail15:
				pos = pos13
				break
			}
			// ([eE] [+\-]? [0-9]+)?
			{
				pos17 := pos
				// ([eE] [+\-]? [0-9]+)
				// [eE] [+\-]? [0-9]+
				// [eE]
				if r, w := _next(parser, pos); r != 'e' && r != 'E' {
					perr = _max(perr, pos)
					goto fail18
				} else {
					pos += w
				}
				// [+\-]?
				{
					pos21 := pos
					// [+\-]
					if r, w := _next(parser, pos); r != '+' && r != '-' {
						perr = _max(perr, pos)
						goto fail22
					} else {
						pos += w
					}
					goto ok23
				fail22:
					pos = pos21
				ok23:
				}
				// [0-9]+
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					perr = _max(perr, pos)
					goto fail18
				} else {
					pos += w
				}
				for {
					pos25 := pos
					// [0-9]
					if r, w := _next(parser, pos); r < '0' || r > '9' {
						perr = _max(perr, pos)
						goto fail27
					} else {
						pos += w
					}
					continue
				fail27:
					pos = pos25
					break
				}
				goto ok28
			fail18:
				pos = pos17
			ok28:
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	perr = start
	return _memoize(parser, _Float, start, pos, perr)
fail:
	return _memoize(parser, _Float, start, -1, perr)
}

func _FloatFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Float, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Float",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Float}
	// action
	// _ tok:(text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// tok:(text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
	{
		pos1 := pos
		// (text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
		// action
		// text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?)
		{
			pos2 := pos
			// ([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?)
			// [+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?
			// [+\-]?
			{
				pos5 := pos
				// [+\-]
				if r, w := _next(parser, pos); r != '+' && r != '-' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[+\\-]",
						})
					}
					goto fail6
				} else {
					pos += w
				}
				goto ok7
			fail6:
				pos = pos5
			ok7:
			}
			// [0-9]+
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[0-9]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			for {
				pos9 := pos
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[0-9]",
						})
					}
					goto fail11
				} else {
					pos += w
				}
				continue
			fail11:
				pos = pos9
				break
			}
			// "."
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\".\"",
					})
				}
				goto fail
			}
			pos++
			// [0-9]+
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[0-9]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			for {
				pos13 := pos
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[0-9]",
						})
					}
					goto fail15
				} else {
					pos += w
				}
				continue
			fail15:
				pos = pos13
				break
			}
			// ([eE] [+\-]? [0-9]+)?
			{
				pos17 := pos
				// ([eE] [+\-]? [0-9]+)
				// [eE] [+\-]? [0-9]+
				// [eE]
				if r, w := _next(parser, pos); r != 'e' && r != 'E' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[eE]",
						})
					}
					goto fail18
				} else {
					pos += w
				}
				// [+\-]?
				{
					pos21 := pos
					// [+\-]
					if r, w := _next(parser, pos); r != '+' && r != '-' {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "[+\\-]",
							})
						}
						goto fail22
					} else {
						pos += w
					}
					goto ok23
				fail22:
					pos = pos21
				ok23:
				}
				// [0-9]+
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[0-9]",
						})
					}
					goto fail18
				} else {
					pos += w
				}
				for {
					pos25 := pos
					// [0-9]
					if r, w := _next(parser, pos); r < '0' || r > '9' {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "[0-9]",
							})
						}
						goto fail27
					} else {
						pos += w
					}
					continue
				fail27:
					pos = pos25
					break
				}
				goto ok28
			fail18:
				pos = pos17
			ok28:
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "floating point"
	parser.fail[key] = failure
	return -1, failure
}

func _FloatAction(parser *_Parser, start int) (int, *Expr) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 *Float
	dp := parser.deltaPos[start][_Float]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Float}
	n := parser.act[key]
	if n != nil {
		n := n.(Expr)
		return start + int(dp-1), &n
	}
	var node Expr
	pos := start
	// action
	{
		start0 := pos
		// _ tok:(text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// tok:(text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
		{
			pos2 := pos
			// (text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
			// action
			{
				start3 := pos
				// text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?)
				{
					pos4 := pos
					// ([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?)
					// [+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?
					{
						var node5 string
						// [+\-]?
						{
							pos7 := pos
							// [+\-]
							if r, w := _next(parser, pos); r != '+' && r != '-' {
								goto fail8
							} else {
								node5 = parser.text[pos : pos+w]
								pos += w
							}
							goto ok9
						fail8:
							node5 = ""
							pos = pos7
						ok9:
						}
						label0, node5 = label0+node5, ""
						// [0-9]+
						{
							var node12 string
							// [0-9]
							if r, w := _next(parser, pos); r < '0' || r > '9' {
								goto fail
							} else {
								node12 = parser.text[pos : pos+w]
								pos += w
							}
							node5 += node12
						}
						for {
							pos11 := pos
							var node12 string
							// [0-9]
							if r, w := _next(parser, pos); r < '0' || r > '9' {
								goto fail13
							} else {
								node12 = parser.text[pos : pos+w]
								pos += w
							}
							node5 += node12
							continue
						fail13:
							pos = pos11
							break
						}
						label0, node5 = label0+node5, ""
						// "."
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
							goto fail
						}
						node5 = parser.text[pos : pos+1]
						pos++
						label0, node5 = label0+node5, ""
						// [0-9]+
						{
							var node16 string
							// [0-9]
							if r, w := _next(parser, pos); r < '0' || r > '9' {
								goto fail
							} else {
								node16 = parser.text[pos : pos+w]
								pos += w
							}
							node5 += node16
						}
						for {
							pos15 := pos
							var node16 string
							// [0-9]
							if r, w := _next(parser, pos); r < '0' || r > '9' {
								goto fail17
							} else {
								node16 = parser.text[pos : pos+w]
								pos += w
							}
							node5 += node16
							continue
						fail17:
							pos = pos15
							break
						}
						label0, node5 = label0+node5, ""
						// ([eE] [+\-]? [0-9]+)?
						{
							pos19 := pos
							// ([eE] [+\-]? [0-9]+)
							// [eE] [+\-]? [0-9]+
							{
								var node21 string
								// [eE]
								if r, w := _next(parser, pos); r != 'e' && r != 'E' {
									goto fail20
								} else {
									node21 = parser.text[pos : pos+w]
									pos += w
								}
								node5, node21 = node5+node21, ""
								// [+\-]?
								{
									pos23 := pos
									// [+\-]
									if r, w := _next(parser, pos); r != '+' && r != '-' {
										goto fail24
									} else {
										node21 = parser.text[pos : pos+w]
										pos += w
									}
									goto ok25
								fail24:
									node21 = ""
									pos = pos23
								ok25:
								}
								node5, node21 = node5+node21, ""
								// [0-9]+
								{
									var node28 string
									// [0-9]
									if r, w := _next(parser, pos); r < '0' || r > '9' {
										goto fail20
									} else {
										node28 = parser.text[pos : pos+w]
										pos += w
									}
									node21 += node28
								}
								for {
									pos27 := pos
									var node28 string
									// [0-9]
									if r, w := _next(parser, pos); r < '0' || r > '9' {
										goto fail29
									} else {
										node28 = parser.text[pos : pos+w]
										pos += w
									}
									node21 += node28
									continue
								fail29:
									pos = pos27
									break
								}
								node5, node21 = node5+node21, ""
							}
							goto ok30
						fail20:
							node5 = ""
							pos = pos19
						ok30:
						}
						label0, node5 = label0+node5, ""
					}
					labels[0] = parser.text[pos4:pos]
				}
				label1 = func(
					start, end int, text string) *Float {
					return &Float{location: loc(parser, start, end), Text: text}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, text string, tok *Float) Expr {
			return Expr(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _RuneAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Rune, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ tok:(text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// tok:(text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
	{
		pos1 := pos
		// (text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
		// action
		// text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\'])
		{
			pos2 := pos
			// ([\'] !"\n" data:(Esc/"\\'"/[^\']) [\'])
			// [\'] !"\n" data:(Esc/"\\'"/[^\']) [\']
			// [\']
			if r, w := _next(parser, pos); r != '\'' {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			// !"\n"
			{
				pos5 := pos
				perr7 := perr
				// "\n"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
					perr = _max(perr, pos)
					goto ok4
				}
				pos++
				pos = pos5
				perr = _max(perr7, pos)
				goto fail
			ok4:
				pos = pos5
				perr = perr7
			}
			// data:(Esc/"\\'"/[^\'])
			{
				pos8 := pos
				// (Esc/"\\'"/[^\'])
				// Esc/"\\'"/[^\']
				{
					pos12 := pos
					// Esc
					if !_accept(parser, _EscAccepts, &pos, &perr) {
						goto fail13
					}
					goto ok9
				fail13:
					pos = pos12
					// "\\'"
					if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\'" {
						perr = _max(perr, pos)
						goto fail14
					}
					pos += 2
					goto ok9
				fail14:
					pos = pos12
					// [^\']
					if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '\'' {
						perr = _max(perr, pos)
						goto fail15
					} else {
						pos += w
					}
					goto ok9
				fail15:
					pos = pos12
					goto fail
				ok9:
				}
				labels[0] = parser.text[pos8:pos]
			}
			// [\']
			if r, w := _next(parser, pos); r != '\'' {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			labels[1] = parser.text[pos2:pos]
		}
		labels[2] = parser.text[pos1:pos]
	}
	perr = start
	return _memoize(parser, _Rune, start, pos, perr)
fail:
	return _memoize(parser, _Rune, start, -1, perr)
}

func _RuneFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Rune, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Rune",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Rune}
	// action
	// _ tok:(text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// tok:(text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
	{
		pos1 := pos
		// (text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
		// action
		// text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\'])
		{
			pos2 := pos
			// ([\'] !"\n" data:(Esc/"\\'"/[^\']) [\'])
			// [\'] !"\n" data:(Esc/"\\'"/[^\']) [\']
			// [\']
			if r, w := _next(parser, pos); r != '\'' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[\\']",
					})
				}
				goto fail
			} else {
				pos += w
			}
			// !"\n"
			{
				pos5 := pos
				nkids6 := len(failure.Kids)
				// "\n"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"\\n\"",
						})
					}
					goto ok4
				}
				pos++
				pos = pos5
				failure.Kids = failure.Kids[:nkids6]
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "!\"\\n\"",
					})
				}
				goto fail
			ok4:
				pos = pos5
				failure.Kids = failure.Kids[:nkids6]
			}
			// data:(Esc/"\\'"/[^\'])
			{
				pos8 := pos
				// (Esc/"\\'"/[^\'])
				// Esc/"\\'"/[^\']
				{
					pos12 := pos
					// Esc
					if !_fail(parser, _EscFail, errPos, failure, &pos) {
						goto fail13
					}
					goto ok9
				fail13:
					pos = pos12
					// "\\'"
					if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\'" {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "\"\\\\'\"",
							})
						}
						goto fail14
					}
					pos += 2
					goto ok9
				fail14:
					pos = pos12
					// [^\']
					if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '\'' {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "[^\\']",
							})
						}
						goto fail15
					} else {
						pos += w
					}
					goto ok9
				fail15:
					pos = pos12
					goto fail
				ok9:
				}
				labels[0] = parser.text[pos8:pos]
			}
			// [\']
			if r, w := _next(parser, pos); r != '\'' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[\\']",
					})
				}
				goto fail
			} else {
				pos += w
			}
			labels[1] = parser.text[pos2:pos]
		}
		labels[2] = parser.text[pos1:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "rune"
	parser.fail[key] = failure
	return -1, failure
}

func _RuneAction(parser *_Parser, start int) (int, *Expr) {
	var labels [3]string
	use(labels)
	var label0 string
	var label1 string
	var label2 *Rune
	dp := parser.deltaPos[start][_Rune]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Rune}
	n := parser.act[key]
	if n != nil {
		n := n.(Expr)
		return start + int(dp-1), &n
	}
	var node Expr
	pos := start
	// action
	{
		start0 := pos
		// _ tok:(text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// tok:(text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
		{
			pos2 := pos
			// (text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
			// action
			{
				start3 := pos
				// text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\'])
				{
					pos4 := pos
					// ([\'] !"\n" data:(Esc/"\\'"/[^\']) [\'])
					// [\'] !"\n" data:(Esc/"\\'"/[^\']) [\']
					{
						var node5 string
						// [\']
						if r, w := _next(parser, pos); r != '\'' {
							goto fail
						} else {
							node5 = parser.text[pos : pos+w]
							pos += w
						}
						label1, node5 = label1+node5, ""
						// !"\n"
						{
							pos7 := pos
							// "\n"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
								goto ok6
							}
							pos++
							pos = pos7
							goto fail
						ok6:
							pos = pos7
							node5 = ""
						}
						label1, node5 = label1+node5, ""
						// data:(Esc/"\\'"/[^\'])
						{
							pos10 := pos
							// (Esc/"\\'"/[^\'])
							// Esc/"\\'"/[^\']
							{
								pos14 := pos
								var node13 string
								// Esc
								if p, n := _EscAction(parser, pos); n == nil {
									goto fail15
								} else {
									label0 = *n
									pos = p
								}
								goto ok11
							fail15:
								label0 = node13
								pos = pos14
								// "\\'"
								if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\'" {
									goto fail16
								}
								label0 = parser.text[pos : pos+2]
								pos += 2
								goto ok11
							fail16:
								label0 = node13
								pos = pos14
								// [^\']
								if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '\'' {
									goto fail17
								} else {
									label0 = parser.text[pos : pos+w]
									pos += w
								}
								goto ok11
							fail17:
								label0 = node13
								pos = pos14
								goto fail
							ok11:
							}
							node5 = label0
							labels[0] = parser.text[pos10:pos]
						}
						label1, node5 = label1+node5, ""
						// [\']
						if r, w := _next(parser, pos); r != '\'' {
							goto fail
						} else {
							node5 = parser.text[pos : pos+w]
							pos += w
						}
						label1, node5 = label1+node5, ""
					}
					labels[1] = parser.text[pos4:pos]
				}
				label2 = func(
					start, end int, data string, text string) *Rune {
					r, w := utf8.DecodeRuneInString(data)
					if w != len(data) {
						panic("impossible")
					}
					return &Rune{location: loc(parser, start, end), Text: text, Rune: r}
				}(
					start3, pos, label0, label1)
			}
			labels[2] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, data string, text string, tok *Rune) Expr {
			return Expr(tok)
		}(
			start0, pos, label0, label1, label2)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _StringAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [6]string
	use(labels)
	if dp, de, ok := _memo(parser, _String, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…}) {…}/_ tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…}) {…}
	{
		pos3 := pos
		// action
		// _ tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail4
		}
		// tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
		{
			pos6 := pos
			// (text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
			// action
			// text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["])
			{
				pos7 := pos
				// (["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["])
				// ["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]
				// ["]
				if r, w := _next(parser, pos); r != '"' {
					perr = _max(perr, pos)
					goto fail4
				} else {
					pos += w
				}
				// data0:(!"\n" (Esc/"\\\""/[^"]))*
				{
					pos9 := pos
					// (!"\n" (Esc/"\\\""/[^"]))*
					for {
						pos11 := pos
						// (!"\n" (Esc/"\\\""/[^"]))
						// !"\n" (Esc/"\\\""/[^"])
						// !"\n"
						{
							pos16 := pos
							perr18 := perr
							// "\n"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
								perr = _max(perr, pos)
								goto ok15
							}
							pos++
							pos = pos16
							perr = _max(perr18, pos)
							goto fail13
						ok15:
							pos = pos16
							perr = perr18
						}
						// (Esc/"\\\""/[^"])
						// Esc/"\\\""/[^"]
						{
							pos22 := pos
							// Esc
							if !_accept(parser, _EscAccepts, &pos, &perr) {
								goto fail23
							}
							goto ok19
						fail23:
							pos = pos22
							// "\\\""
							if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\"" {
								perr = _max(perr, pos)
								goto fail24
							}
							pos += 2
							goto ok19
						fail24:
							pos = pos22
							// [^"]
							if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '"' {
								perr = _max(perr, pos)
								goto fail25
							} else {
								pos += w
							}
							goto ok19
						fail25:
							pos = pos22
							goto fail13
						ok19:
						}
						continue
					fail13:
						pos = pos11
						break
					}
					labels[0] = parser.text[pos9:pos]
				}
				// ["]
				if r, w := _next(parser, pos); r != '"' {
					perr = _max(perr, pos)
					goto fail4
				} else {
					pos += w
				}
				labels[1] = parser.text[pos7:pos]
			}
			labels[2] = parser.text[pos6:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// _ tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…})
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail26
		}
		// tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…})
		{
			pos28 := pos
			// (text1:([`] data1:("\\`"/[^`])* [`]) {…})
			// action
			// text1:([`] data1:("\\`"/[^`])* [`])
			{
				pos29 := pos
				// ([`] data1:("\\`"/[^`])* [`])
				// [`] data1:("\\`"/[^`])* [`]
				// [`]
				if r, w := _next(parser, pos); r != '`' {
					perr = _max(perr, pos)
					goto fail26
				} else {
					pos += w
				}
				// data1:("\\`"/[^`])*
				{
					pos31 := pos
					// ("\\`"/[^`])*
					for {
						pos33 := pos
						// ("\\`"/[^`])
						// "\\`"/[^`]
						{
							pos39 := pos
							// "\\`"
							if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\`" {
								perr = _max(perr, pos)
								goto fail40
							}
							pos += 2
							goto ok36
						fail40:
							pos = pos39
							// [^`]
							if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '`' {
								perr = _max(perr, pos)
								goto fail41
							} else {
								pos += w
							}
							goto ok36
						fail41:
							pos = pos39
							goto fail35
						ok36:
						}
						continue
					fail35:
						pos = pos33
						break
					}
					labels[3] = parser.text[pos31:pos]
				}
				// [`]
				if r, w := _next(parser, pos); r != '`' {
					perr = _max(perr, pos)
					goto fail26
				} else {
					pos += w
				}
				labels[4] = parser.text[pos29:pos]
			}
			labels[5] = parser.text[pos28:pos]
		}
		goto ok0
	fail26:
		pos = pos3
		goto fail
	ok0:
	}
	perr = start
	return _memoize(parser, _String, start, pos, perr)
fail:
	return _memoize(parser, _String, start, -1, perr)
}

func _StringFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [6]string
	use(labels)
	pos, failure := _failMemo(parser, _String, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "String",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _String}
	// _ tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…}) {…}/_ tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…}) {…}
	{
		pos3 := pos
		// action
		// _ tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail4
		}
		// tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
		{
			pos6 := pos
			// (text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
			// action
			// text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["])
			{
				pos7 := pos
				// (["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["])
				// ["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]
				// ["]
				if r, w := _next(parser, pos); r != '"' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[\"]",
						})
					}
					goto fail4
				} else {
					pos += w
				}
				// data0:(!"\n" (Esc/"\\\""/[^"]))*
				{
					pos9 := pos
					// (!"\n" (Esc/"\\\""/[^"]))*
					for {
						pos11 := pos
						// (!"\n" (Esc/"\\\""/[^"]))
						// !"\n" (Esc/"\\\""/[^"])
						// !"\n"
						{
							pos16 := pos
							nkids17 := len(failure.Kids)
							// "\n"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
								if pos >= errPos {
									failure.Kids = append(failure.Kids, &peg.Fail{
										Pos:  int(pos),
										Want: "\"\\n\"",
									})
								}
								goto ok15
							}
							pos++
							pos = pos16
							failure.Kids = failure.Kids[:nkids17]
							if pos >= errPos {
								failure.Kids = append(failure.Kids, &peg.Fail{
									Pos:  int(pos),
									Want: "!\"\\n\"",
								})
							}
							goto fail13
						ok15:
							pos = pos16
							failure.Kids = failure.Kids[:nkids17]
						}
						// (Esc/"\\\""/[^"])
						// Esc/"\\\""/[^"]
						{
							pos22 := pos
							// Esc
							if !_fail(parser, _EscFail, errPos, failure, &pos) {
								goto fail23
							}
							goto ok19
						fail23:
							pos = pos22
							// "\\\""
							if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\"" {
								if pos >= errPos {
									failure.Kids = append(failure.Kids, &peg.Fail{
										Pos:  int(pos),
										Want: "\"\\\\\\\"\"",
									})
								}
								goto fail24
							}
							pos += 2
							goto ok19
						fail24:
							pos = pos22
							// [^"]
							if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '"' {
								if pos >= errPos {
									failure.Kids = append(failure.Kids, &peg.Fail{
										Pos:  int(pos),
										Want: "[^\"]",
									})
								}
								goto fail25
							} else {
								pos += w
							}
							goto ok19
						fail25:
							pos = pos22
							goto fail13
						ok19:
						}
						continue
					fail13:
						pos = pos11
						break
					}
					labels[0] = parser.text[pos9:pos]
				}
				// ["]
				if r, w := _next(parser, pos); r != '"' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[\"]",
						})
					}
					goto fail4
				} else {
					pos += w
				}
				labels[1] = parser.text[pos7:pos]
			}
			labels[2] = parser.text[pos6:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// _ tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…})
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail26
		}
		// tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…})
		{
			pos28 := pos
			// (text1:([`] data1:("\\`"/[^`])* [`]) {…})
			// action
			// text1:([`] data1:("\\`"/[^`])* [`])
			{
				pos29 := pos
				// ([`] data1:("\\`"/[^`])* [`])
				// [`] data1:("\\`"/[^`])* [`]
				// [`]
				if r, w := _next(parser, pos); r != '`' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[`]",
						})
					}
					goto fail26
				} else {
					pos += w
				}
				// data1:("\\`"/[^`])*
				{
					pos31 := pos
					// ("\\`"/[^`])*
					for {
						pos33 := pos
						// ("\\`"/[^`])
						// "\\`"/[^`]
						{
							pos39 := pos
							// "\\`"
							if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\`" {
								if pos >= errPos {
									failure.Kids = append(failure.Kids, &peg.Fail{
										Pos:  int(pos),
										Want: "\"\\\\`\"",
									})
								}
								goto fail40
							}
							pos += 2
							goto ok36
						fail40:
							pos = pos39
							// [^`]
							if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '`' {
								if pos >= errPos {
									failure.Kids = append(failure.Kids, &peg.Fail{
										Pos:  int(pos),
										Want: "[^`]",
									})
								}
								goto fail41
							} else {
								pos += w
							}
							goto ok36
						fail41:
							pos = pos39
							goto fail35
						ok36:
						}
						continue
					fail35:
						pos = pos33
						break
					}
					labels[3] = parser.text[pos31:pos]
				}
				// [`]
				if r, w := _next(parser, pos); r != '`' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[`]",
						})
					}
					goto fail26
				} else {
					pos += w
				}
				labels[4] = parser.text[pos29:pos]
			}
			labels[5] = parser.text[pos28:pos]
		}
		goto ok0
	fail26:
		pos = pos3
		goto fail
	ok0:
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "string"
	parser.fail[key] = failure
	return -1, failure
}

func _StringAction(parser *_Parser, start int) (int, *String) {
	var labels [6]string
	use(labels)
	var label0 string
	var label1 string
	var label2 String
	var label3 string
	var label4 string
	var label5 String
	dp := parser.deltaPos[start][_String]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _String}
	n := parser.act[key]
	if n != nil {
		n := n.(String)
		return start + int(dp-1), &n
	}
	var node String
	pos := start
	// _ tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…}) {…}/_ tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…}) {…}
	{
		pos3 := pos
		var node2 String
		// action
		{
			start5 := pos
			// _ tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail4
			} else {
				pos = p
			}
			// tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
			{
				pos7 := pos
				// (text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
				// action
				{
					start8 := pos
					// text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["])
					{
						pos9 := pos
						// (["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["])
						// ["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]
						{
							var node10 string
							// ["]
							if r, w := _next(parser, pos); r != '"' {
								goto fail4
							} else {
								node10 = parser.text[pos : pos+w]
								pos += w
							}
							label1, node10 = label1+node10, ""
							// data0:(!"\n" (Esc/"\\\""/[^"]))*
							{
								pos11 := pos
								// (!"\n" (Esc/"\\\""/[^"]))*
								for {
									pos13 := pos
									var node14 string
									// (!"\n" (Esc/"\\\""/[^"]))
									// !"\n" (Esc/"\\\""/[^"])
									{
										var node16 string
										// !"\n"
										{
											pos18 := pos
											// "\n"
											if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
												goto ok17
											}
											pos++
											pos = pos18
											goto fail15
										ok17:
											pos = pos18
											node16 = ""
										}
										node14, node16 = node14+node16, ""
										// (Esc/"\\\""/[^"])
										// Esc/"\\\""/[^"]
										{
											pos24 := pos
											var node23 string
											// Esc
											if p, n := _EscAction(parser, pos); n == nil {
												goto fail25
											} else {
												node16 = *n
												pos = p
											}
											goto ok21
										fail25:
											node16 = node23
											pos = pos24
											// "\\\""
											if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\"" {
												goto fail26
											}
											node16 = parser.text[pos : pos+2]
											pos += 2
											goto ok21
										fail26:
											node16 = node23
											pos = pos24
											// [^"]
											if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '"' {
												goto fail27
											} else {
												node16 = parser.text[pos : pos+w]
												pos += w
											}
											goto ok21
										fail27:
											node16 = node23
											pos = pos24
											goto fail15
										ok21:
										}
										node14, node16 = node14+node16, ""
									}
									label0 += node14
									continue
								fail15:
									pos = pos13
									break
								}
								node10 = label0
								labels[0] = parser.text[pos11:pos]
							}
							label1, node10 = label1+node10, ""
							// ["]
							if r, w := _next(parser, pos); r != '"' {
								goto fail4
							} else {
								node10 = parser.text[pos : pos+w]
								pos += w
							}
							label1, node10 = label1+node10, ""
						}
						labels[1] = parser.text[pos9:pos]
					}
					label2 = func(
						start, end int, data0 string, text0 string) String {
						return String{
							location: loc(parser, start, end),
							Text:     text0,
							Data:     data0,
						}
					}(
						start8, pos, label0, label1)
				}
				labels[2] = parser.text[pos7:pos]
			}
			node = func(
				start, end int, data0 string, text0 string, tok0 String) String {
				return String(tok0)
			}(
				start5, pos, label0, label1, label2)
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// action
		{
			start29 := pos
			// _ tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…})
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail28
			} else {
				pos = p
			}
			// tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…})
			{
				pos31 := pos
				// (text1:([`] data1:("\\`"/[^`])* [`]) {…})
				// action
				{
					start32 := pos
					// text1:([`] data1:("\\`"/[^`])* [`])
					{
						pos33 := pos
						// ([`] data1:("\\`"/[^`])* [`])
						// [`] data1:("\\`"/[^`])* [`]
						{
							var node34 string
							// [`]
							if r, w := _next(parser, pos); r != '`' {
								goto fail28
							} else {
								node34 = parser.text[pos : pos+w]
								pos += w
							}
							label4, node34 = label4+node34, ""
							// data1:("\\`"/[^`])*
							{
								pos35 := pos
								// ("\\`"/[^`])*
								for {
									pos37 := pos
									var node38 string
									// ("\\`"/[^`])
									// "\\`"/[^`]
									{
										pos43 := pos
										var node42 string
										// "\\`"
										if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\`" {
											goto fail44
										}
										node38 = parser.text[pos : pos+2]
										pos += 2
										goto ok40
									fail44:
										node38 = node42
										pos = pos43
										// [^`]
										if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '`' {
											goto fail45
										} else {
											node38 = parser.text[pos : pos+w]
											pos += w
										}
										goto ok40
									fail45:
										node38 = node42
										pos = pos43
										goto fail39
									ok40:
									}
									label3 += node38
									continue
								fail39:
									pos = pos37
									break
								}
								node34 = label3
								labels[3] = parser.text[pos35:pos]
							}
							label4, node34 = label4+node34, ""
							// [`]
							if r, w := _next(parser, pos); r != '`' {
								goto fail28
							} else {
								node34 = parser.text[pos : pos+w]
								pos += w
							}
							label4, node34 = label4+node34, ""
						}
						labels[4] = parser.text[pos33:pos]
					}
					label5 = func(
						start, end int, data0 string, data1 string, text0 string, text1 string, tok0 String) String {
						return String{
							location: loc(parser, start, end),
							Text:     text1,
							Data:     data1,
						}
					}(
						start32, pos, label0, label3, label1, label4, label2)
				}
				labels[5] = parser.text[pos31:pos]
			}
			node = func(
				start, end int, data0 string, data1 string, text0 string, text1 string, tok0 String, tok1 String) String {
				return String(tok1)
			}(
				start29, pos, label0, label3, label1, label4, label2, label5)
		}
		goto ok0
	fail28:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _EscAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Esc, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// "\\n" {…}/"\\t" {…}/"\\b" {…}/"\\\\" {…}/"\\" x0:(X X) {…}/"\\x" x1:(X X X X) {…}/"\\X" x2:(X X X X X X X X) {…}
	{
		pos3 := pos
		// action
		// "\\n"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\n" {
			perr = _max(perr, pos)
			goto fail4
		}
		pos += 2
		goto ok0
	fail4:
		pos = pos3
		// action
		// "\\t"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\t" {
			perr = _max(perr, pos)
			goto fail5
		}
		pos += 2
		goto ok0
	fail5:
		pos = pos3
		// action
		// "\\b"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\b" {
			perr = _max(perr, pos)
			goto fail6
		}
		pos += 2
		goto ok0
	fail6:
		pos = pos3
		// action
		// "\\\\"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\\" {
			perr = _max(perr, pos)
			goto fail7
		}
		pos += 2
		goto ok0
	fail7:
		pos = pos3
		// action
		// "\\" x0:(X X)
		// "\\"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\\" {
			perr = _max(perr, pos)
			goto fail8
		}
		pos++
		// x0:(X X)
		{
			pos10 := pos
			// (X X)
			// X X
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail8
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail8
			}
			labels[0] = parser.text[pos10:pos]
		}
		goto ok0
	fail8:
		pos = pos3
		// action
		// "\\x" x1:(X X X X)
		// "\\x"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\x" {
			perr = _max(perr, pos)
			goto fail12
		}
		pos += 2
		// x1:(X X X X)
		{
			pos14 := pos
			// (X X X X)
			// X X X X
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail12
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail12
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail12
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail12
			}
			labels[1] = parser.text[pos14:pos]
		}
		goto ok0
	fail12:
		pos = pos3
		// action
		// "\\X" x2:(X X X X X X X X)
		// "\\X"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\X" {
			perr = _max(perr, pos)
			goto fail16
		}
		pos += 2
		// x2:(X X X X X X X X)
		{
			pos18 := pos
			// (X X X X X X X X)
			// X X X X X X X X
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail16
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail16
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail16
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail16
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail16
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail16
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail16
			}
			// X
			if !_accept(parser, _XAccepts, &pos, &perr) {
				goto fail16
			}
			labels[2] = parser.text[pos18:pos]
		}
		goto ok0
	fail16:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Esc, start, pos, perr)
fail:
	return _memoize(parser, _Esc, start, -1, perr)
}

func _EscFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Esc, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Esc",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Esc}
	// "\\n" {…}/"\\t" {…}/"\\b" {…}/"\\\\" {…}/"\\" x0:(X X) {…}/"\\x" x1:(X X X X) {…}/"\\X" x2:(X X X X X X X X) {…}
	{
		pos3 := pos
		// action
		// "\\n"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\n" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\n\"",
				})
			}
			goto fail4
		}
		pos += 2
		goto ok0
	fail4:
		pos = pos3
		// action
		// "\\t"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\t" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\t\"",
				})
			}
			goto fail5
		}
		pos += 2
		goto ok0
	fail5:
		pos = pos3
		// action
		// "\\b"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\b" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\b\"",
				})
			}
			goto fail6
		}
		pos += 2
		goto ok0
	fail6:
		pos = pos3
		// action
		// "\\\\"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\\" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\\\\\\"",
				})
			}
			goto fail7
		}
		pos += 2
		goto ok0
	fail7:
		pos = pos3
		// action
		// "\\" x0:(X X)
		// "\\"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\\" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\\"",
				})
			}
			goto fail8
		}
		pos++
		// x0:(X X)
		{
			pos10 := pos
			// (X X)
			// X X
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail8
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail8
			}
			labels[0] = parser.text[pos10:pos]
		}
		goto ok0
	fail8:
		pos = pos3
		// action
		// "\\x" x1:(X X X X)
		// "\\x"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\x" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\x\"",
				})
			}
			goto fail12
		}
		pos += 2
		// x1:(X X X X)
		{
			pos14 := pos
			// (X X X X)
			// X X X X
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail12
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail12
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail12
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail12
			}
			labels[1] = parser.text[pos14:pos]
		}
		goto ok0
	fail12:
		pos = pos3
		// action
		// "\\X" x2:(X X X X X X X X)
		// "\\X"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\X" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\X\"",
				})
			}
			goto fail16
		}
		pos += 2
		// x2:(X X X X X X X X)
		{
			pos18 := pos
			// (X X X X X X X X)
			// X X X X X X X X
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail16
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail16
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail16
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail16
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail16
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail16
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail16
			}
			// X
			if !_fail(parser, _XFail, errPos, failure, &pos) {
				goto fail16
			}
			labels[2] = parser.text[pos18:pos]
		}
		goto ok0
	fail16:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _EscAction(parser *_Parser, start int) (int, *string) {
	var labels [3]string
	use(labels)
	var label0 string
	var label1 string
	var label2 string
	dp := parser.deltaPos[start][_Esc]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Esc}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// "\\n" {…}/"\\t" {…}/"\\b" {…}/"\\\\" {…}/"\\" x0:(X X) {…}/"\\x" x1:(X X X X) {…}/"\\X" x2:(X X X X X X X X) {…}
	{
		pos3 := pos
		var node2 string
		// action
		{
			start5 := pos
			// "\\n"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\n" {
				goto fail4
			}
			pos += 2
			node = func(
				start, end int) string {
				return "\n"
			}(
				start5, pos)
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// action
		{
			start7 := pos
			// "\\t"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\t" {
				goto fail6
			}
			pos += 2
			node = func(
				start, end int) string {
				return "\t"
			}(
				start7, pos)
		}
		goto ok0
	fail6:
		node = node2
		pos = pos3
		// action
		{
			start9 := pos
			// "\\b"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\b" {
				goto fail8
			}
			pos += 2
			node = func(
				start, end int) string {
				return "\b"
			}(
				start9, pos)
		}
		goto ok0
	fail8:
		node = node2
		pos = pos3
		// action
		{
			start11 := pos
			// "\\\\"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\\" {
				goto fail10
			}
			pos += 2
			node = func(
				start, end int) string {
				return "\\"
			}(
				start11, pos)
		}
		goto ok0
	fail10:
		node = node2
		pos = pos3
		// action
		{
			start13 := pos
			// "\\" x0:(X X)
			// "\\"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\\" {
				goto fail12
			}
			pos++
			// x0:(X X)
			{
				pos15 := pos
				// (X X)
				// X X
				{
					var node16 string
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail12
					} else {
						node16 = *n
						pos = p
					}
					label0, node16 = label0+node16, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail12
					} else {
						node16 = *n
						pos = p
					}
					label0, node16 = label0+node16, ""
				}
				labels[0] = parser.text[pos15:pos]
			}
			node = func(
				start, end int, x0 string) string {
				return string(hex(x0))
			}(
				start13, pos, label0)
		}
		goto ok0
	fail12:
		node = node2
		pos = pos3
		// action
		{
			start18 := pos
			// "\\x" x1:(X X X X)
			// "\\x"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\x" {
				goto fail17
			}
			pos += 2
			// x1:(X X X X)
			{
				pos20 := pos
				// (X X X X)
				// X X X X
				{
					var node21 string
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail17
					} else {
						node21 = *n
						pos = p
					}
					label1, node21 = label1+node21, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail17
					} else {
						node21 = *n
						pos = p
					}
					label1, node21 = label1+node21, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail17
					} else {
						node21 = *n
						pos = p
					}
					label1, node21 = label1+node21, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail17
					} else {
						node21 = *n
						pos = p
					}
					label1, node21 = label1+node21, ""
				}
				labels[1] = parser.text[pos20:pos]
			}
			node = func(
				start, end int, x0 string, x1 string) string {
				return string(hex(x1))
			}(
				start18, pos, label0, label1)
		}
		goto ok0
	fail17:
		node = node2
		pos = pos3
		// action
		{
			start23 := pos
			// "\\X" x2:(X X X X X X X X)
			// "\\X"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\X" {
				goto fail22
			}
			pos += 2
			// x2:(X X X X X X X X)
			{
				pos25 := pos
				// (X X X X X X X X)
				// X X X X X X X X
				{
					var node26 string
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail22
					} else {
						node26 = *n
						pos = p
					}
					label2, node26 = label2+node26, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail22
					} else {
						node26 = *n
						pos = p
					}
					label2, node26 = label2+node26, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail22
					} else {
						node26 = *n
						pos = p
					}
					label2, node26 = label2+node26, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail22
					} else {
						node26 = *n
						pos = p
					}
					label2, node26 = label2+node26, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail22
					} else {
						node26 = *n
						pos = p
					}
					label2, node26 = label2+node26, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail22
					} else {
						node26 = *n
						pos = p
					}
					label2, node26 = label2+node26, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail22
					} else {
						node26 = *n
						pos = p
					}
					label2, node26 = label2+node26, ""
					// X
					if p, n := _XAction(parser, pos); n == nil {
						goto fail22
					} else {
						node26 = *n
						pos = p
					}
					label2, node26 = label2+node26, ""
				}
				labels[2] = parser.text[pos25:pos]
			}
			node = func(
				start, end int, x0 string, x1 string, x2 string) string {
				return string(hex(x2))
			}(
				start23, pos, label0, label1, label2)
		}
		goto ok0
	fail22:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _XAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _X, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// [a-fA-F0-9]
	if r, w := _next(parser, pos); (r < 'a' || r > 'f') && (r < 'A' || r > 'F') && (r < '0' || r > '9') {
		perr = _max(perr, pos)
		goto fail
	} else {
		pos += w
	}
	return _memoize(parser, _X, start, pos, perr)
fail:
	return _memoize(parser, _X, start, -1, perr)
}

func _XFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _X, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "X",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _X}
	// [a-fA-F0-9]
	if r, w := _next(parser, pos); (r < 'a' || r > 'f') && (r < 'A' || r > 'F') && (r < '0' || r > '9') {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "[a-fA-F0-9]",
			})
		}
		goto fail
	} else {
		pos += w
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _XAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_X]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _X}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// [a-fA-F0-9]
	if r, w := _next(parser, pos); (r < 'a' || r > 'f') && (r < 'A' || r > 'F') && (r < '0' || r > '9') {
		goto fail
	} else {
		node = parser.text[pos : pos+w]
		pos += w
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _OpAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Op, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ !"//" !"/*" tok:(text:([!%&*+\-/<=>?@\\|~]+) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// !"//"
	{
		pos2 := pos
		perr4 := perr
		// "//"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
			perr = _max(perr, pos)
			goto ok1
		}
		pos += 2
		pos = pos2
		perr = _max(perr4, pos)
		goto fail
	ok1:
		pos = pos2
		perr = perr4
	}
	// !"/*"
	{
		pos6 := pos
		perr8 := perr
		// "/*"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
			perr = _max(perr, pos)
			goto ok5
		}
		pos += 2
		pos = pos6
		perr = _max(perr8, pos)
		goto fail
	ok5:
		pos = pos6
		perr = perr8
	}
	// tok:(text:([!%&*+\-/<=>?@\\|~]+) {…})
	{
		pos9 := pos
		// (text:([!%&*+\-/<=>?@\\|~]+) {…})
		// action
		// text:([!%&*+\-/<=>?@\\|~]+)
		{
			pos10 := pos
			// ([!%&*+\-/<=>?@\\|~]+)
			// [!%&*+\-/<=>?@\\|~]+
			// [!%&*+\-/<=>?@\\|~]
			if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			for {
				pos12 := pos
				// [!%&*+\-/<=>?@\\|~]
				if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
					perr = _max(perr, pos)
					goto fail14
				} else {
					pos += w
				}
				continue
			fail14:
				pos = pos12
				break
			}
			labels[0] = parser.text[pos10:pos]
		}
		labels[1] = parser.text[pos9:pos]
	}
	perr = start
	return _memoize(parser, _Op, start, pos, perr)
fail:
	return _memoize(parser, _Op, start, -1, perr)
}

func _OpFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Op, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Op",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Op}
	// action
	// _ !"//" !"/*" tok:(text:([!%&*+\-/<=>?@\\|~]+) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// !"//"
	{
		pos2 := pos
		nkids3 := len(failure.Kids)
		// "//"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"//\"",
				})
			}
			goto ok1
		}
		pos += 2
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!\"//\"",
			})
		}
		goto fail
	ok1:
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
	}
	// !"/*"
	{
		pos6 := pos
		nkids7 := len(failure.Kids)
		// "/*"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"/*\"",
				})
			}
			goto ok5
		}
		pos += 2
		pos = pos6
		failure.Kids = failure.Kids[:nkids7]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!\"/*\"",
			})
		}
		goto fail
	ok5:
		pos = pos6
		failure.Kids = failure.Kids[:nkids7]
	}
	// tok:(text:([!%&*+\-/<=>?@\\|~]+) {…})
	{
		pos9 := pos
		// (text:([!%&*+\-/<=>?@\\|~]+) {…})
		// action
		// text:([!%&*+\-/<=>?@\\|~]+)
		{
			pos10 := pos
			// ([!%&*+\-/<=>?@\\|~]+)
			// [!%&*+\-/<=>?@\\|~]+
			// [!%&*+\-/<=>?@\\|~]
			if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[!%&*+\\-/<=>?@\\\\|~]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			for {
				pos12 := pos
				// [!%&*+\-/<=>?@\\|~]
				if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[!%&*+\\-/<=>?@\\\\|~]",
						})
					}
					goto fail14
				} else {
					pos += w
				}
				continue
			fail14:
				pos = pos12
				break
			}
			labels[0] = parser.text[pos10:pos]
		}
		labels[1] = parser.text[pos9:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "operator"
	parser.fail[key] = failure
	return -1, failure
}

func _OpAction(parser *_Parser, start int) (int, *Ident) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 Ident
	dp := parser.deltaPos[start][_Op]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Op}
	n := parser.act[key]
	if n != nil {
		n := n.(Ident)
		return start + int(dp-1), &n
	}
	var node Ident
	pos := start
	// action
	{
		start0 := pos
		// _ !"//" !"/*" tok:(text:([!%&*+\-/<=>?@\\|~]+) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// !"//"
		{
			pos3 := pos
			// "//"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
				goto ok2
			}
			pos += 2
			pos = pos3
			goto fail
		ok2:
			pos = pos3
		}
		// !"/*"
		{
			pos7 := pos
			// "/*"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
				goto ok6
			}
			pos += 2
			pos = pos7
			goto fail
		ok6:
			pos = pos7
		}
		// tok:(text:([!%&*+\-/<=>?@\\|~]+) {…})
		{
			pos10 := pos
			// (text:([!%&*+\-/<=>?@\\|~]+) {…})
			// action
			{
				start11 := pos
				// text:([!%&*+\-/<=>?@\\|~]+)
				{
					pos12 := pos
					// ([!%&*+\-/<=>?@\\|~]+)
					// [!%&*+\-/<=>?@\\|~]+
					{
						var node15 string
						// [!%&*+\-/<=>?@\\|~]
						if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
							goto fail
						} else {
							node15 = parser.text[pos : pos+w]
							pos += w
						}
						label0 += node15
					}
					for {
						pos14 := pos
						var node15 string
						// [!%&*+\-/<=>?@\\|~]
						if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
							goto fail16
						} else {
							node15 = parser.text[pos : pos+w]
							pos += w
						}
						label0 += node15
						continue
					fail16:
						pos = pos14
						break
					}
					labels[0] = parser.text[pos12:pos]
				}
				label1 = func(
					start, end int, text string) Ident {
					return Ident{location: loc(parser, start, end), Text: text}
				}(
					start11, pos, label0)
			}
			labels[1] = parser.text[pos10:pos]
		}
		node = func(
			start, end int, text string, tok Ident) Ident {
			return Ident(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeOpAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _TypeOp, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ tok:(text:([!&?]+) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// tok:(text:([!&?]+) {…})
	{
		pos1 := pos
		// (text:([!&?]+) {…})
		// action
		// text:([!&?]+)
		{
			pos2 := pos
			// ([!&?]+)
			// [!&?]+
			// [!&?]
			if r, w := _next(parser, pos); r != '!' && r != '&' && r != '?' {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			for {
				pos4 := pos
				// [!&?]
				if r, w := _next(parser, pos); r != '!' && r != '&' && r != '?' {
					perr = _max(perr, pos)
					goto fail6
				} else {
					pos += w
				}
				continue
			fail6:
				pos = pos4
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	perr = start
	return _memoize(parser, _TypeOp, start, pos, perr)
fail:
	return _memoize(parser, _TypeOp, start, -1, perr)
}

func _TypeOpFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _TypeOp, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeOp",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeOp}
	// action
	// _ tok:(text:([!&?]+) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// tok:(text:([!&?]+) {…})
	{
		pos1 := pos
		// (text:([!&?]+) {…})
		// action
		// text:([!&?]+)
		{
			pos2 := pos
			// ([!&?]+)
			// [!&?]+
			// [!&?]
			if r, w := _next(parser, pos); r != '!' && r != '&' && r != '?' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[!&?]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			for {
				pos4 := pos
				// [!&?]
				if r, w := _next(parser, pos); r != '!' && r != '&' && r != '?' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[!&?]",
						})
					}
					goto fail6
				} else {
					pos += w
				}
				continue
			fail6:
				pos = pos4
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "type operator"
	parser.fail[key] = failure
	return -1, failure
}

func _TypeOpAction(parser *_Parser, start int) (int, *Ident) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 Ident
	dp := parser.deltaPos[start][_TypeOp]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeOp}
	n := parser.act[key]
	if n != nil {
		n := n.(Ident)
		return start + int(dp-1), &n
	}
	var node Ident
	pos := start
	// action
	{
		start0 := pos
		// _ tok:(text:([!&?]+) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// tok:(text:([!&?]+) {…})
		{
			pos2 := pos
			// (text:([!&?]+) {…})
			// action
			{
				start3 := pos
				// text:([!&?]+)
				{
					pos4 := pos
					// ([!&?]+)
					// [!&?]+
					{
						var node7 string
						// [!&?]
						if r, w := _next(parser, pos); r != '!' && r != '&' && r != '?' {
							goto fail
						} else {
							node7 = parser.text[pos : pos+w]
							pos += w
						}
						label0 += node7
					}
					for {
						pos6 := pos
						var node7 string
						// [!&?]
						if r, w := _next(parser, pos); r != '!' && r != '&' && r != '?' {
							goto fail8
						} else {
							node7 = parser.text[pos : pos+w]
							pos += w
						}
						label0 += node7
						continue
					fail8:
						pos = pos6
						break
					}
					labels[0] = parser.text[pos4:pos]
				}
				label1 = func(
					start, end int, text string) Ident {
					return Ident{location: loc(parser, start, end), Text: text}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, text string, tok Ident) Ident {
			return Ident(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ModNameAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _ModName, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ tok:(text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// tok:(text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
	{
		pos1 := pos
		// (text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
		// action
		// text:("#" [_a-zA-Z] [_a-zA-Z0-9]*)
		{
			pos2 := pos
			// ("#" [_a-zA-Z] [_a-zA-Z0-9]*)
			// "#" [_a-zA-Z] [_a-zA-Z0-9]*
			// "#"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "#" {
				perr = _max(perr, pos)
				goto fail
			}
			pos++
			// [_a-zA-Z]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			// [_a-zA-Z0-9]*
			for {
				pos5 := pos
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					perr = _max(perr, pos)
					goto fail7
				} else {
					pos += w
				}
				continue
			fail7:
				pos = pos5
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	perr = start
	return _memoize(parser, _ModName, start, pos, perr)
fail:
	return _memoize(parser, _ModName, start, -1, perr)
}

func _ModNameFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _ModName, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "ModName",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _ModName}
	// action
	// _ tok:(text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// tok:(text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
	{
		pos1 := pos
		// (text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
		// action
		// text:("#" [_a-zA-Z] [_a-zA-Z0-9]*)
		{
			pos2 := pos
			// ("#" [_a-zA-Z] [_a-zA-Z0-9]*)
			// "#" [_a-zA-Z] [_a-zA-Z0-9]*
			// "#"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "#" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"#\"",
					})
				}
				goto fail
			}
			pos++
			// [_a-zA-Z]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[_a-zA-Z]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			// [_a-zA-Z0-9]*
			for {
				pos5 := pos
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[_a-zA-Z0-9]",
						})
					}
					goto fail7
				} else {
					pos += w
				}
				continue
			fail7:
				pos = pos5
				break
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "module name"
	parser.fail[key] = failure
	return -1, failure
}

func _ModNameAction(parser *_Parser, start int) (int, *Ident) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 Ident
	dp := parser.deltaPos[start][_ModName]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _ModName}
	n := parser.act[key]
	if n != nil {
		n := n.(Ident)
		return start + int(dp-1), &n
	}
	var node Ident
	pos := start
	// action
	{
		start0 := pos
		// _ tok:(text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// tok:(text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
		{
			pos2 := pos
			// (text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
			// action
			{
				start3 := pos
				// text:("#" [_a-zA-Z] [_a-zA-Z0-9]*)
				{
					pos4 := pos
					// ("#" [_a-zA-Z] [_a-zA-Z0-9]*)
					// "#" [_a-zA-Z] [_a-zA-Z0-9]*
					{
						var node5 string
						// "#"
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "#" {
							goto fail
						}
						node5 = parser.text[pos : pos+1]
						pos++
						label0, node5 = label0+node5, ""
						// [_a-zA-Z]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
							goto fail
						} else {
							node5 = parser.text[pos : pos+w]
							pos += w
						}
						label0, node5 = label0+node5, ""
						// [_a-zA-Z0-9]*
						for {
							pos7 := pos
							var node8 string
							// [_a-zA-Z0-9]
							if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
								goto fail9
							} else {
								node8 = parser.text[pos : pos+w]
								pos += w
							}
							node5 += node8
							continue
						fail9:
							pos = pos7
							break
						}
						label0, node5 = label0+node5, ""
					}
					labels[0] = parser.text[pos4:pos]
				}
				label1 = func(
					start, end int, text string) Ident {
					return Ident{location: loc(parser, start, end), Text: text}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, text string, tok Ident) Ident {
			return Ident(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _IdentCAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _IdentC, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// !TypeVar
	{
		pos2 := pos
		perr4 := perr
		// TypeVar
		if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
			goto ok1
		}
		pos = pos2
		perr = _max(perr4, pos)
		goto fail
	ok1:
		pos = pos2
		perr = perr4
	}
	// !"import"
	{
		pos6 := pos
		perr8 := perr
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			perr = _max(perr, pos)
			goto ok5
		}
		pos += 6
		pos = pos6
		perr = _max(perr8, pos)
		goto fail
	ok5:
		pos = pos6
		perr = perr8
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
		// action
		// text:([_a-zA-Z] [_a-zA-Z0-9]* ":")
		{
			pos10 := pos
			// ([_a-zA-Z] [_a-zA-Z0-9]* ":")
			// [_a-zA-Z] [_a-zA-Z0-9]* ":"
			// [_a-zA-Z]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			// [_a-zA-Z0-9]*
			for {
				pos13 := pos
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					perr = _max(perr, pos)
					goto fail15
				} else {
					pos += w
				}
				continue
			fail15:
				pos = pos13
				break
			}
			// ":"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
				perr = _max(perr, pos)
				goto fail
			}
			pos++
			labels[0] = parser.text[pos10:pos]
		}
		labels[1] = parser.text[pos9:pos]
	}
	perr = start
	return _memoize(parser, _IdentC, start, pos, perr)
fail:
	return _memoize(parser, _IdentC, start, -1, perr)
}

func _IdentCFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _IdentC, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "IdentC",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _IdentC}
	// action
	// _ !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// !TypeVar
	{
		pos2 := pos
		nkids3 := len(failure.Kids)
		// TypeVar
		if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
			goto ok1
		}
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!TypeVar",
			})
		}
		goto fail
	ok1:
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
	}
	// !"import"
	{
		pos6 := pos
		nkids7 := len(failure.Kids)
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"import\"",
				})
			}
			goto ok5
		}
		pos += 6
		pos = pos6
		failure.Kids = failure.Kids[:nkids7]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!\"import\"",
			})
		}
		goto fail
	ok5:
		pos = pos6
		failure.Kids = failure.Kids[:nkids7]
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
		// action
		// text:([_a-zA-Z] [_a-zA-Z0-9]* ":")
		{
			pos10 := pos
			// ([_a-zA-Z] [_a-zA-Z0-9]* ":")
			// [_a-zA-Z] [_a-zA-Z0-9]* ":"
			// [_a-zA-Z]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[_a-zA-Z]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			// [_a-zA-Z0-9]*
			for {
				pos13 := pos
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[_a-zA-Z0-9]",
						})
					}
					goto fail15
				} else {
					pos += w
				}
				continue
			fail15:
				pos = pos13
				break
			}
			// ":"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\":\"",
					})
				}
				goto fail
			}
			pos++
			labels[0] = parser.text[pos10:pos]
		}
		labels[1] = parser.text[pos9:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "identifier:"
	parser.fail[key] = failure
	return -1, failure
}

func _IdentCAction(parser *_Parser, start int) (int, *Ident) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 Ident
	dp := parser.deltaPos[start][_IdentC]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _IdentC}
	n := parser.act[key]
	if n != nil {
		n := n.(Ident)
		return start + int(dp-1), &n
	}
	var node Ident
	pos := start
	// action
	{
		start0 := pos
		// _ !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// !TypeVar
		{
			pos3 := pos
			// TypeVar
			if p, n := _TypeVarAction(parser, pos); n == nil {
				goto ok2
			} else {
				pos = p
			}
			pos = pos3
			goto fail
		ok2:
			pos = pos3
		}
		// !"import"
		{
			pos7 := pos
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				goto ok6
			}
			pos += 6
			pos = pos7
			goto fail
		ok6:
			pos = pos7
		}
		// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
		{
			pos10 := pos
			// (text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
			// action
			{
				start11 := pos
				// text:([_a-zA-Z] [_a-zA-Z0-9]* ":")
				{
					pos12 := pos
					// ([_a-zA-Z] [_a-zA-Z0-9]* ":")
					// [_a-zA-Z] [_a-zA-Z0-9]* ":"
					{
						var node13 string
						// [_a-zA-Z]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
							goto fail
						} else {
							node13 = parser.text[pos : pos+w]
							pos += w
						}
						label0, node13 = label0+node13, ""
						// [_a-zA-Z0-9]*
						for {
							pos15 := pos
							var node16 string
							// [_a-zA-Z0-9]
							if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
								goto fail17
							} else {
								node16 = parser.text[pos : pos+w]
								pos += w
							}
							node13 += node16
							continue
						fail17:
							pos = pos15
							break
						}
						label0, node13 = label0+node13, ""
						// ":"
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
							goto fail
						}
						node13 = parser.text[pos : pos+1]
						pos++
						label0, node13 = label0+node13, ""
					}
					labels[0] = parser.text[pos12:pos]
				}
				label1 = func(
					start, end int, text string) Ident {
					return Ident{location: loc(parser, start, end), Text: text}
				}(
					start11, pos, label0)
			}
			labels[1] = parser.text[pos10:pos]
		}
		node = func(
			start, end int, text string, tok Ident) Ident {
			return Ident(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _CIdentAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _CIdent, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ ":" !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// ":"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	// !TypeVar
	{
		pos2 := pos
		perr4 := perr
		// TypeVar
		if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
			goto ok1
		}
		pos = pos2
		perr = _max(perr4, pos)
		goto fail
	ok1:
		pos = pos2
		perr = perr4
	}
	// !"import"
	{
		pos6 := pos
		perr8 := perr
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			perr = _max(perr, pos)
			goto ok5
		}
		pos += 6
		pos = pos6
		perr = _max(perr8, pos)
		goto fail
	ok5:
		pos = pos6
		perr = perr8
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
		// action
		// text:([_a-zA-Z] [_a-zA-Z0-9]*)
		{
			pos10 := pos
			// ([_a-zA-Z] [_a-zA-Z0-9]*)
			// [_a-zA-Z] [_a-zA-Z0-9]*
			// [_a-zA-Z]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			// [_a-zA-Z0-9]*
			for {
				pos13 := pos
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					perr = _max(perr, pos)
					goto fail15
				} else {
					pos += w
				}
				continue
			fail15:
				pos = pos13
				break
			}
			labels[0] = parser.text[pos10:pos]
		}
		labels[1] = parser.text[pos9:pos]
	}
	perr = start
	return _memoize(parser, _CIdent, start, pos, perr)
fail:
	return _memoize(parser, _CIdent, start, -1, perr)
}

func _CIdentFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _CIdent, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "CIdent",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _CIdent}
	// action
	// _ ":" !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// ":"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "\":\"",
			})
		}
		goto fail
	}
	pos++
	// !TypeVar
	{
		pos2 := pos
		nkids3 := len(failure.Kids)
		// TypeVar
		if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
			goto ok1
		}
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!TypeVar",
			})
		}
		goto fail
	ok1:
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
	}
	// !"import"
	{
		pos6 := pos
		nkids7 := len(failure.Kids)
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"import\"",
				})
			}
			goto ok5
		}
		pos += 6
		pos = pos6
		failure.Kids = failure.Kids[:nkids7]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!\"import\"",
			})
		}
		goto fail
	ok5:
		pos = pos6
		failure.Kids = failure.Kids[:nkids7]
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
		// action
		// text:([_a-zA-Z] [_a-zA-Z0-9]*)
		{
			pos10 := pos
			// ([_a-zA-Z] [_a-zA-Z0-9]*)
			// [_a-zA-Z] [_a-zA-Z0-9]*
			// [_a-zA-Z]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[_a-zA-Z]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			// [_a-zA-Z0-9]*
			for {
				pos13 := pos
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[_a-zA-Z0-9]",
						})
					}
					goto fail15
				} else {
					pos += w
				}
				continue
			fail15:
				pos = pos13
				break
			}
			labels[0] = parser.text[pos10:pos]
		}
		labels[1] = parser.text[pos9:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = ":identifier"
	parser.fail[key] = failure
	return -1, failure
}

func _CIdentAction(parser *_Parser, start int) (int, *Ident) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 Ident
	dp := parser.deltaPos[start][_CIdent]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _CIdent}
	n := parser.act[key]
	if n != nil {
		n := n.(Ident)
		return start + int(dp-1), &n
	}
	var node Ident
	pos := start
	// action
	{
		start0 := pos
		// _ ":" !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// ":"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
			goto fail
		}
		pos++
		// !TypeVar
		{
			pos3 := pos
			// TypeVar
			if p, n := _TypeVarAction(parser, pos); n == nil {
				goto ok2
			} else {
				pos = p
			}
			pos = pos3
			goto fail
		ok2:
			pos = pos3
		}
		// !"import"
		{
			pos7 := pos
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				goto ok6
			}
			pos += 6
			pos = pos7
			goto fail
		ok6:
			pos = pos7
		}
		// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
		{
			pos10 := pos
			// (text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
			// action
			{
				start11 := pos
				// text:([_a-zA-Z] [_a-zA-Z0-9]*)
				{
					pos12 := pos
					// ([_a-zA-Z] [_a-zA-Z0-9]*)
					// [_a-zA-Z] [_a-zA-Z0-9]*
					{
						var node13 string
						// [_a-zA-Z]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
							goto fail
						} else {
							node13 = parser.text[pos : pos+w]
							pos += w
						}
						label0, node13 = label0+node13, ""
						// [_a-zA-Z0-9]*
						for {
							pos15 := pos
							var node16 string
							// [_a-zA-Z0-9]
							if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
								goto fail17
							} else {
								node16 = parser.text[pos : pos+w]
								pos += w
							}
							node13 += node16
							continue
						fail17:
							pos = pos15
							break
						}
						label0, node13 = label0+node13, ""
					}
					labels[0] = parser.text[pos12:pos]
				}
				label1 = func(
					start, end int, text string) Ident {
					return Ident{location: loc(parser, start, end), Text: text}
				}(
					start11, pos, label0)
			}
			labels[1] = parser.text[pos10:pos]
		}
		node = func(
			start, end int, text string, tok Ident) Ident {
			return Ident(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _IdentAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Ident, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// !TypeVar
	{
		pos2 := pos
		perr4 := perr
		// TypeVar
		if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
			goto ok1
		}
		pos = pos2
		perr = _max(perr4, pos)
		goto fail
	ok1:
		pos = pos2
		perr = perr4
	}
	// !"import"
	{
		pos6 := pos
		perr8 := perr
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			perr = _max(perr, pos)
			goto ok5
		}
		pos += 6
		pos = pos6
		perr = _max(perr8, pos)
		goto fail
	ok5:
		pos = pos6
		perr = perr8
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
		// action
		// text:([_a-zA-Z] [_a-zA-Z0-9]*) !":"
		// text:([_a-zA-Z] [_a-zA-Z0-9]*)
		{
			pos11 := pos
			// ([_a-zA-Z] [_a-zA-Z0-9]*)
			// [_a-zA-Z] [_a-zA-Z0-9]*
			// [_a-zA-Z]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			// [_a-zA-Z0-9]*
			for {
				pos14 := pos
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					perr = _max(perr, pos)
					goto fail16
				} else {
					pos += w
				}
				continue
			fail16:
				pos = pos14
				break
			}
			labels[0] = parser.text[pos11:pos]
		}
		// !":"
		{
			pos18 := pos
			perr20 := perr
			// ":"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
				perr = _max(perr, pos)
				goto ok17
			}
			pos++
			pos = pos18
			perr = _max(perr20, pos)
			goto fail
		ok17:
			pos = pos18
			perr = perr20
		}
		labels[1] = parser.text[pos9:pos]
	}
	perr = start
	return _memoize(parser, _Ident, start, pos, perr)
fail:
	return _memoize(parser, _Ident, start, -1, perr)
}

func _IdentFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Ident, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Ident",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Ident}
	// action
	// _ !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// !TypeVar
	{
		pos2 := pos
		nkids3 := len(failure.Kids)
		// TypeVar
		if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
			goto ok1
		}
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!TypeVar",
			})
		}
		goto fail
	ok1:
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
	}
	// !"import"
	{
		pos6 := pos
		nkids7 := len(failure.Kids)
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"import\"",
				})
			}
			goto ok5
		}
		pos += 6
		pos = pos6
		failure.Kids = failure.Kids[:nkids7]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!\"import\"",
			})
		}
		goto fail
	ok5:
		pos = pos6
		failure.Kids = failure.Kids[:nkids7]
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
		// action
		// text:([_a-zA-Z] [_a-zA-Z0-9]*) !":"
		// text:([_a-zA-Z] [_a-zA-Z0-9]*)
		{
			pos11 := pos
			// ([_a-zA-Z] [_a-zA-Z0-9]*)
			// [_a-zA-Z] [_a-zA-Z0-9]*
			// [_a-zA-Z]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[_a-zA-Z]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			// [_a-zA-Z0-9]*
			for {
				pos14 := pos
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[_a-zA-Z0-9]",
						})
					}
					goto fail16
				} else {
					pos += w
				}
				continue
			fail16:
				pos = pos14
				break
			}
			labels[0] = parser.text[pos11:pos]
		}
		// !":"
		{
			pos18 := pos
			nkids19 := len(failure.Kids)
			// ":"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\":\"",
					})
				}
				goto ok17
			}
			pos++
			pos = pos18
			failure.Kids = failure.Kids[:nkids19]
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "!\":\"",
				})
			}
			goto fail
		ok17:
			pos = pos18
			failure.Kids = failure.Kids[:nkids19]
		}
		labels[1] = parser.text[pos9:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "identifier"
	parser.fail[key] = failure
	return -1, failure
}

func _IdentAction(parser *_Parser, start int) (int, *Ident) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 Ident
	dp := parser.deltaPos[start][_Ident]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ident}
	n := parser.act[key]
	if n != nil {
		n := n.(Ident)
		return start + int(dp-1), &n
	}
	var node Ident
	pos := start
	// action
	{
		start0 := pos
		// _ !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// !TypeVar
		{
			pos3 := pos
			// TypeVar
			if p, n := _TypeVarAction(parser, pos); n == nil {
				goto ok2
			} else {
				pos = p
			}
			pos = pos3
			goto fail
		ok2:
			pos = pos3
		}
		// !"import"
		{
			pos7 := pos
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				goto ok6
			}
			pos += 6
			pos = pos7
			goto fail
		ok6:
			pos = pos7
		}
		// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
		{
			pos10 := pos
			// (text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
			// action
			{
				start11 := pos
				// text:([_a-zA-Z] [_a-zA-Z0-9]*) !":"
				// text:([_a-zA-Z] [_a-zA-Z0-9]*)
				{
					pos13 := pos
					// ([_a-zA-Z] [_a-zA-Z0-9]*)
					// [_a-zA-Z] [_a-zA-Z0-9]*
					{
						var node14 string
						// [_a-zA-Z]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
							goto fail
						} else {
							node14 = parser.text[pos : pos+w]
							pos += w
						}
						label0, node14 = label0+node14, ""
						// [_a-zA-Z0-9]*
						for {
							pos16 := pos
							var node17 string
							// [_a-zA-Z0-9]
							if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
								goto fail18
							} else {
								node17 = parser.text[pos : pos+w]
								pos += w
							}
							node14 += node17
							continue
						fail18:
							pos = pos16
							break
						}
						label0, node14 = label0+node14, ""
					}
					labels[0] = parser.text[pos13:pos]
				}
				// !":"
				{
					pos20 := pos
					// ":"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
						goto ok19
					}
					pos++
					pos = pos20
					goto fail
				ok19:
					pos = pos20
				}
				label1 = func(
					start, end int, text string) Ident {
					return Ident{location: loc(parser, start, end), Text: text}
				}(
					start11, pos, label0)
			}
			labels[1] = parser.text[pos10:pos]
		}
		node = func(
			start, end int, text string, tok Ident) Ident {
			return Ident(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeVarAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _TypeVar, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ tok:(text:([A-Z] ![_a-zA-Z0-9]) {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// tok:(text:([A-Z] ![_a-zA-Z0-9]) {…})
	{
		pos1 := pos
		// (text:([A-Z] ![_a-zA-Z0-9]) {…})
		// action
		// text:([A-Z] ![_a-zA-Z0-9])
		{
			pos2 := pos
			// ([A-Z] ![_a-zA-Z0-9])
			// [A-Z] ![_a-zA-Z0-9]
			// [A-Z]
			if r, w := _next(parser, pos); r < 'A' || r > 'Z' {
				perr = _max(perr, pos)
				goto fail
			} else {
				pos += w
			}
			// ![_a-zA-Z0-9]
			{
				pos5 := pos
				perr7 := perr
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					perr = _max(perr, pos)
					goto ok4
				} else {
					pos += w
				}
				pos = pos5
				perr = _max(perr7, pos)
				goto fail
			ok4:
				pos = pos5
				perr = perr7
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	perr = start
	return _memoize(parser, _TypeVar, start, pos, perr)
fail:
	return _memoize(parser, _TypeVar, start, -1, perr)
}

func _TypeVarFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _TypeVar, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeVar",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeVar}
	// action
	// _ tok:(text:([A-Z] ![_a-zA-Z0-9]) {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// tok:(text:([A-Z] ![_a-zA-Z0-9]) {…})
	{
		pos1 := pos
		// (text:([A-Z] ![_a-zA-Z0-9]) {…})
		// action
		// text:([A-Z] ![_a-zA-Z0-9])
		{
			pos2 := pos
			// ([A-Z] ![_a-zA-Z0-9])
			// [A-Z] ![_a-zA-Z0-9]
			// [A-Z]
			if r, w := _next(parser, pos); r < 'A' || r > 'Z' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[A-Z]",
					})
				}
				goto fail
			} else {
				pos += w
			}
			// ![_a-zA-Z0-9]
			{
				pos5 := pos
				nkids6 := len(failure.Kids)
				// [_a-zA-Z0-9]
				if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[_a-zA-Z0-9]",
						})
					}
					goto ok4
				} else {
					pos += w
				}
				pos = pos5
				failure.Kids = failure.Kids[:nkids6]
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "![_a-zA-Z0-9]",
					})
				}
				goto fail
			ok4:
				pos = pos5
				failure.Kids = failure.Kids[:nkids6]
			}
			labels[0] = parser.text[pos2:pos]
		}
		labels[1] = parser.text[pos1:pos]
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "type variable"
	parser.fail[key] = failure
	return -1, failure
}

func _TypeVarAction(parser *_Parser, start int) (int, *Ident) {
	var labels [2]string
	use(labels)
	var label0 string
	var label1 Ident
	dp := parser.deltaPos[start][_TypeVar]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeVar}
	n := parser.act[key]
	if n != nil {
		n := n.(Ident)
		return start + int(dp-1), &n
	}
	var node Ident
	pos := start
	// action
	{
		start0 := pos
		// _ tok:(text:([A-Z] ![_a-zA-Z0-9]) {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// tok:(text:([A-Z] ![_a-zA-Z0-9]) {…})
		{
			pos2 := pos
			// (text:([A-Z] ![_a-zA-Z0-9]) {…})
			// action
			{
				start3 := pos
				// text:([A-Z] ![_a-zA-Z0-9])
				{
					pos4 := pos
					// ([A-Z] ![_a-zA-Z0-9])
					// [A-Z] ![_a-zA-Z0-9]
					{
						var node5 string
						// [A-Z]
						if r, w := _next(parser, pos); r < 'A' || r > 'Z' {
							goto fail
						} else {
							node5 = parser.text[pos : pos+w]
							pos += w
						}
						label0, node5 = label0+node5, ""
						// ![_a-zA-Z0-9]
						{
							pos7 := pos
							// [_a-zA-Z0-9]
							if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
								goto ok6
							} else {
								pos += w
							}
							pos = pos7
							goto fail
						ok6:
							pos = pos7
							node5 = ""
						}
						label0, node5 = label0+node5, ""
					}
					labels[0] = parser.text[pos4:pos]
				}
				label1 = func(
					start, end int, text string) Ident {
					return Ident{location: loc(parser, start, end), Text: text}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, text string, tok Ident) Ident {
			return Ident(tok)
		}(
			start0, pos, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func __Accepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, __, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// (Space/Cmnt)*
	for {
		pos1 := pos
		// (Space/Cmnt)
		// Space/Cmnt
		{
			pos7 := pos
			// Space
			if !_accept(parser, _SpaceAccepts, &pos, &perr) {
				goto fail8
			}
			goto ok4
		fail8:
			pos = pos7
			// Cmnt
			if !_accept(parser, _CmntAccepts, &pos, &perr) {
				goto fail9
			}
			goto ok4
		fail9:
			pos = pos7
			goto fail3
		ok4:
		}
		continue
	fail3:
		pos = pos1
		break
	}
	perr = start
	return _memoize(parser, __, start, pos, perr)
}

func __Fail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, __, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "_",
		Pos:  int(start),
	}
	key := _key{start: start, rule: __}
	// (Space/Cmnt)*
	for {
		pos1 := pos
		// (Space/Cmnt)
		// Space/Cmnt
		{
			pos7 := pos
			// Space
			if !_fail(parser, _SpaceFail, errPos, failure, &pos) {
				goto fail8
			}
			goto ok4
		fail8:
			pos = pos7
			// Cmnt
			if !_fail(parser, _CmntFail, errPos, failure, &pos) {
				goto fail9
			}
			goto ok4
		fail9:
			pos = pos7
			goto fail3
		ok4:
		}
		continue
	fail3:
		pos = pos1
		break
	}
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
}

func __Action(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][__]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: __}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// (Space/Cmnt)*
	for {
		pos1 := pos
		var node2 string
		// (Space/Cmnt)
		// Space/Cmnt
		{
			pos7 := pos
			var node6 string
			// Space
			if p, n := _SpaceAction(parser, pos); n == nil {
				goto fail8
			} else {
				node2 = *n
				pos = p
			}
			goto ok4
		fail8:
			node2 = node6
			pos = pos7
			// Cmnt
			if p, n := _CmntAction(parser, pos); n == nil {
				goto fail9
			} else {
				node2 = *n
				pos = p
			}
			goto ok4
		fail9:
			node2 = node6
			pos = pos7
			goto fail3
		ok4:
		}
		node += node2
		continue
	fail3:
		pos = pos1
		break
	}
	parser.act[key] = node
	return pos, &node
}

func _CmntAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Cmnt, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// "//" (!"\n" .)*/"/*" (!"*/" .)* "*/"
	{
		pos3 := pos
		// "//" (!"\n" .)*
		// "//"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
			perr = _max(perr, pos)
			goto fail4
		}
		pos += 2
		// (!"\n" .)*
		for {
			pos7 := pos
			// (!"\n" .)
			// !"\n" .
			// !"\n"
			{
				pos12 := pos
				perr14 := perr
				// "\n"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
					perr = _max(perr, pos)
					goto ok11
				}
				pos++
				pos = pos12
				perr = _max(perr14, pos)
				goto fail9
			ok11:
				pos = pos12
				perr = perr14
			}
			// .
			if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
				perr = _max(perr, pos)
				goto fail9
			} else {
				pos += w
			}
			continue
		fail9:
			pos = pos7
			break
		}
		goto ok0
	fail4:
		pos = pos3
		// "/*" (!"*/" .)* "*/"
		// "/*"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
			perr = _max(perr, pos)
			goto fail15
		}
		pos += 2
		// (!"*/" .)*
		for {
			pos18 := pos
			// (!"*/" .)
			// !"*/" .
			// !"*/"
			{
				pos23 := pos
				perr25 := perr
				// "*/"
				if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "*/" {
					perr = _max(perr, pos)
					goto ok22
				}
				pos += 2
				pos = pos23
				perr = _max(perr25, pos)
				goto fail20
			ok22:
				pos = pos23
				perr = perr25
			}
			// .
			if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
				perr = _max(perr, pos)
				goto fail20
			} else {
				pos += w
			}
			continue
		fail20:
			pos = pos18
			break
		}
		// "*/"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "*/" {
			perr = _max(perr, pos)
			goto fail15
		}
		pos += 2
		goto ok0
	fail15:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Cmnt, start, pos, perr)
fail:
	return _memoize(parser, _Cmnt, start, -1, perr)
}

func _CmntFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Cmnt, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Cmnt",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Cmnt}
	// "//" (!"\n" .)*/"/*" (!"*/" .)* "*/"
	{
		pos3 := pos
		// "//" (!"\n" .)*
		// "//"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"//\"",
				})
			}
			goto fail4
		}
		pos += 2
		// (!"\n" .)*
		for {
			pos7 := pos
			// (!"\n" .)
			// !"\n" .
			// !"\n"
			{
				pos12 := pos
				nkids13 := len(failure.Kids)
				// "\n"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"\\n\"",
						})
					}
					goto ok11
				}
				pos++
				pos = pos12
				failure.Kids = failure.Kids[:nkids13]
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "!\"\\n\"",
					})
				}
				goto fail9
			ok11:
				pos = pos12
				failure.Kids = failure.Kids[:nkids13]
			}
			// .
			if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: ".",
					})
				}
				goto fail9
			} else {
				pos += w
			}
			continue
		fail9:
			pos = pos7
			break
		}
		goto ok0
	fail4:
		pos = pos3
		// "/*" (!"*/" .)* "*/"
		// "/*"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"/*\"",
				})
			}
			goto fail15
		}
		pos += 2
		// (!"*/" .)*
		for {
			pos18 := pos
			// (!"*/" .)
			// !"*/" .
			// !"*/"
			{
				pos23 := pos
				nkids24 := len(failure.Kids)
				// "*/"
				if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "*/" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"*/\"",
						})
					}
					goto ok22
				}
				pos += 2
				pos = pos23
				failure.Kids = failure.Kids[:nkids24]
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "!\"*/\"",
					})
				}
				goto fail20
			ok22:
				pos = pos23
				failure.Kids = failure.Kids[:nkids24]
			}
			// .
			if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: ".",
					})
				}
				goto fail20
			} else {
				pos += w
			}
			continue
		fail20:
			pos = pos18
			break
		}
		// "*/"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "*/" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"*/\"",
				})
			}
			goto fail15
		}
		pos += 2
		goto ok0
	fail15:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _CmntAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Cmnt]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Cmnt}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// "//" (!"\n" .)*/"/*" (!"*/" .)* "*/"
	{
		pos3 := pos
		var node2 string
		// "//" (!"\n" .)*
		{
			var node5 string
			// "//"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
				goto fail4
			}
			node5 = parser.text[pos : pos+2]
			pos += 2
			node, node5 = node+node5, ""
			// (!"\n" .)*
			for {
				pos7 := pos
				var node8 string
				// (!"\n" .)
				// !"\n" .
				{
					var node10 string
					// !"\n"
					{
						pos12 := pos
						// "\n"
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
							goto ok11
						}
						pos++
						pos = pos12
						goto fail9
					ok11:
						pos = pos12
						node10 = ""
					}
					node8, node10 = node8+node10, ""
					// .
					if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
						goto fail9
					} else {
						node10 = parser.text[pos : pos+w]
						pos += w
					}
					node8, node10 = node8+node10, ""
				}
				node5 += node8
				continue
			fail9:
				pos = pos7
				break
			}
			node, node5 = node+node5, ""
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// "/*" (!"*/" .)* "*/"
		{
			var node16 string
			// "/*"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
				goto fail15
			}
			node16 = parser.text[pos : pos+2]
			pos += 2
			node, node16 = node+node16, ""
			// (!"*/" .)*
			for {
				pos18 := pos
				var node19 string
				// (!"*/" .)
				// !"*/" .
				{
					var node21 string
					// !"*/"
					{
						pos23 := pos
						// "*/"
						if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "*/" {
							goto ok22
						}
						pos += 2
						pos = pos23
						goto fail20
					ok22:
						pos = pos23
						node21 = ""
					}
					node19, node21 = node19+node21, ""
					// .
					if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
						goto fail20
					} else {
						node21 = parser.text[pos : pos+w]
						pos += w
					}
					node19, node21 = node19+node21, ""
				}
				node16 += node19
				continue
			fail20:
				pos = pos18
				break
			}
			node, node16 = node+node16, ""
			// "*/"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "*/" {
				goto fail15
			}
			node16 = parser.text[pos : pos+2]
			pos += 2
			node, node16 = node+node16, ""
		}
		goto ok0
	fail15:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _SpaceAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Space, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// " "/"\t"/"\n"
	{
		pos3 := pos
		// " "
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
			perr = _max(perr, pos)
			goto fail4
		}
		pos++
		goto ok0
	fail4:
		pos = pos3
		// "\t"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\t" {
			perr = _max(perr, pos)
			goto fail5
		}
		pos++
		goto ok0
	fail5:
		pos = pos3
		// "\n"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
			perr = _max(perr, pos)
			goto fail6
		}
		pos++
		goto ok0
	fail6:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Space, start, pos, perr)
fail:
	return _memoize(parser, _Space, start, -1, perr)
}

func _SpaceFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Space, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Space",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Space}
	// " "/"\t"/"\n"
	{
		pos3 := pos
		// " "
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\" \"",
				})
			}
			goto fail4
		}
		pos++
		goto ok0
	fail4:
		pos = pos3
		// "\t"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\t" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\t\"",
				})
			}
			goto fail5
		}
		pos++
		goto ok0
	fail5:
		pos = pos3
		// "\n"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\n\"",
				})
			}
			goto fail6
		}
		pos++
		goto ok0
	fail6:
		pos = pos3
		goto fail
	ok0:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _SpaceAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Space]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Space}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// " "/"\t"/"\n"
	{
		pos3 := pos
		var node2 string
		// " "
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
			goto fail4
		}
		node = parser.text[pos : pos+1]
		pos++
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// "\t"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\t" {
			goto fail5
		}
		node = parser.text[pos : pos+1]
		pos++
		goto ok0
	fail5:
		node = node2
		pos = pos3
		// "\n"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
			goto fail6
		}
		node = parser.text[pos : pos+1]
		pos++
		goto ok0
	fail6:
		node = node2
		pos = pos3
		goto fail
	ok0:
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _EOFAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _EOF, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// !.
	{
		pos1 := pos
		perr3 := perr
		// .
		if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
			perr = _max(perr, pos)
			goto ok0
		} else {
			pos += w
		}
		pos = pos1
		perr = _max(perr3, pos)
		goto fail
	ok0:
		pos = pos1
		perr = perr3
	}
	return _memoize(parser, _EOF, start, pos, perr)
fail:
	return _memoize(parser, _EOF, start, -1, perr)
}

func _EOFFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _EOF, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "EOF",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _EOF}
	// !.
	{
		pos1 := pos
		nkids2 := len(failure.Kids)
		// .
		if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: ".",
				})
			}
			goto ok0
		} else {
			pos += w
		}
		pos = pos1
		failure.Kids = failure.Kids[:nkids2]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!.",
			})
		}
		goto fail
	ok0:
		pos = pos1
		failure.Kids = failure.Kids[:nkids2]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _EOFAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_EOF]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _EOF}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// !.
	{
		pos1 := pos
		// .
		if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
			goto ok0
		} else {
			pos += w
		}
		pos = pos1
		goto fail
	ok0:
		pos = pos1
		node = ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}
