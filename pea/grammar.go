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

func (n Fun) setSig(s TypeSig) Def    { n.Recv = &s; return &n }
func (n Struct) setSig(s TypeSig) Def { n.Sig = s; return &n }
func (n Enum) setSig(s TypeSig) Def   { n.Sig = s; return &n }
func (n Virt) setSig(s TypeSig) Def   { n.Sig = s; return &n }

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
	typ  TypeName
	key  Ident
}

type arg struct {
	name Ident
	val  Expr
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
	_File      int = 0
	_Def       int = 1
	_ModPath   int = 2
	_Fun       int = 3
	_FunSig    int = 4
	_Ret       int = 5
	_Var       int = 6
	_TypeSig   int = 7
	_TypeParms int = 8
	_TypeParm  int = 9
	_TypeName  int = 10
	_Type      int = 11
	_Struct    int = 12
	_Enum      int = 13
	_Case      int = 14
	_Virt      int = 15
	_MethSig   int = 16
	_Stmts     int = 17
	_Stmt      int = 18
	_Return    int = 19
	_Assign    int = 20
	_Lhs       int = 21
	_Expr      int = 22
	_Call      int = 23
	_Unary     int = 24
	_UnaryMsg  int = 25
	_Binary    int = 26
	_BinMsg    int = 27
	_Nary      int = 28
	_NaryMsg   int = 29
	_Primary   int = 30
	_Ctor      int = 31
	_Ary       int = 32
	_Block     int = 33
	_Int       int = 34
	_Float     int = 35
	_Rune      int = 36
	_String    int = 37
	_Esc       int = 38
	_X         int = 39
	_Op        int = 40
	_TypeOp    int = 41
	_ModName   int = 42
	_IdentC    int = 43
	_CIdent    int = 44
	_Ident     int = 45
	_TypeVar   int = 46
	__         int = 47
	_Cmnt      int = 48
	_Space     int = 49
	_EOF       int = 50

	_N int = 51
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
	var labels [1]string
	use(labels)
	if dp, de, ok := _memo(parser, _File, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// dss:Def* _ EOF
	// dss:Def*
	{
		pos1 := pos
		// Def*
		for {
			pos3 := pos
			// Def
			if !_accept(parser, _DefAccepts, &pos, &perr) {
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
	// EOF
	if !_accept(parser, _EOFAccepts, &pos, &perr) {
		goto fail
	}
	return _memoize(parser, _File, start, pos, perr)
fail:
	return _memoize(parser, _File, start, -1, perr)
}

func _FileNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [1]string
	use(labels)
	dp := parser.deltaPos[start][_File]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _File}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "File"}
	// action
	// dss:Def* _ EOF
	// dss:Def*
	{
		pos1 := pos
		// Def*
		for {
			nkids2 := len(node.Kids)
			pos3 := pos
			// Def
			if !_node(parser, _DefNode, node, &pos) {
				goto fail5
			}
			continue
		fail5:
			node.Kids = node.Kids[:nkids2]
			pos = pos3
			break
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// EOF
	if !_node(parser, _EOFNode, node, &pos) {
		goto fail
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _FileFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [1]string
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
	// dss:Def* _ EOF
	// dss:Def*
	{
		pos1 := pos
		// Def*
		for {
			pos3 := pos
			// Def
			if !_fail(parser, _DefFail, errPos, failure, &pos) {
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

func _FileAction(parser *_Parser, start int) (int, **File) {
	var labels [1]string
	use(labels)
	var label0 [][]Def
	dp := parser.deltaPos[start][_File]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _File}
	n := parser.act[key]
	if n != nil {
		n := n.(*File)
		return start + int(dp-1), &n
	}
	var node *File
	pos := start
	// action
	{
		start0 := pos
		// dss:Def* _ EOF
		// dss:Def*
		{
			pos2 := pos
			// Def*
			for {
				pos4 := pos
				var node5 []Def
				// Def
				if p, n := _DefAction(parser, pos); n == nil {
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
		// EOF
		if p, n := _EOFAction(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		node = func(
			start, end int, dss [][]Def) *File {
			return &File{Defs: flatten(dss)}
		}(
			start0, pos, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _DefAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [11]string
	use(labels)
	if dp, de, ok := _memo(parser, _Def, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// m:ModPath? ds0:(_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
	// m:ModPath?
	{
		pos1 := pos
		// ModPath?
		{
			pos3 := pos
			// ModPath
			if !_accept(parser, _ModPathAccepts, &pos, &perr) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// ds0:(_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
	{
		pos6 := pos
		// (_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
		// _ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…}
		{
			pos10 := pos
			// action
			// _ ("import" p:String)
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail11
			}
			// ("import" p:String)
			// "import" p:String
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				perr = _max(perr, pos)
				goto fail11
			}
			pos += 6
			// p:String
			{
				pos14 := pos
				// String
				if !_accept(parser, _StringAccepts, &pos, &perr) {
					goto fail11
				}
				labels[1] = parser.text[pos14:pos]
			}
			goto ok7
		fail11:
			pos = pos10
			// action
			// _ ("(" dss:Def+ _ ")")
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail15
			}
			// ("(" dss:Def+ _ ")")
			// "(" dss:Def+ _ ")"
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				perr = _max(perr, pos)
				goto fail15
			}
			pos++
			// dss:Def+
			{
				pos18 := pos
				// Def+
				// Def
				if !_accept(parser, _DefAccepts, &pos, &perr) {
					goto fail15
				}
				for {
					pos20 := pos
					// Def
					if !_accept(parser, _DefAccepts, &pos, &perr) {
						goto fail22
					}
					continue
				fail22:
					pos = pos20
					break
				}
				labels[2] = parser.text[pos18:pos]
			}
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail15
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				perr = _max(perr, pos)
				goto fail15
			}
			pos++
			goto ok7
		fail15:
			pos = pos10
			// action
			// f:Fun
			{
				pos24 := pos
				// Fun
				if !_accept(parser, _FunAccepts, &pos, &perr) {
					goto fail23
				}
				labels[3] = parser.text[pos24:pos]
			}
			goto ok7
		fail23:
			pos = pos10
			// action
			// v:Var
			{
				pos26 := pos
				// Var
				if !_accept(parser, _VarAccepts, &pos, &perr) {
					goto fail25
				}
				labels[4] = parser.text[pos26:pos]
			}
			goto ok7
		fail25:
			pos = pos10
			// action
			// sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
			// sig:TypeSig
			{
				pos29 := pos
				// TypeSig
				if !_accept(parser, _TypeSigAccepts, &pos, &perr) {
					goto fail27
				}
				labels[5] = parser.text[pos29:pos]
			}
			// ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
			{
				pos30 := pos
				// (t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
				// t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}
				{
					pos34 := pos
					// action
					// t0:Type
					{
						pos36 := pos
						// Type
						if !_accept(parser, _TypeAccepts, &pos, &perr) {
							goto fail35
						}
						labels[6] = parser.text[pos36:pos]
					}
					goto ok31
				fail35:
					pos = pos34
					// action
					// m0:Fun
					{
						pos38 := pos
						// Fun
						if !_accept(parser, _FunAccepts, &pos, &perr) {
							goto fail37
						}
						labels[7] = parser.text[pos38:pos]
					}
					goto ok31
				fail37:
					pos = pos34
					// action
					// _ "(" ds2:(Type/Fun)+ _ ")"
					// _
					if !_accept(parser, __Accepts, &pos, &perr) {
						goto fail39
					}
					// "("
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
						perr = _max(perr, pos)
						goto fail39
					}
					pos++
					// ds2:(Type/Fun)+
					{
						pos41 := pos
						// (Type/Fun)+
						// (Type/Fun)
						// Type/Fun
						{
							pos49 := pos
							// Type
							if !_accept(parser, _TypeAccepts, &pos, &perr) {
								goto fail50
							}
							goto ok46
						fail50:
							pos = pos49
							// Fun
							if !_accept(parser, _FunAccepts, &pos, &perr) {
								goto fail51
							}
							goto ok46
						fail51:
							pos = pos49
							goto fail39
						ok46:
						}
						for {
							pos43 := pos
							// (Type/Fun)
							// Type/Fun
							{
								pos55 := pos
								// Type
								if !_accept(parser, _TypeAccepts, &pos, &perr) {
									goto fail56
								}
								goto ok52
							fail56:
								pos = pos55
								// Fun
								if !_accept(parser, _FunAccepts, &pos, &perr) {
									goto fail57
								}
								goto ok52
							fail57:
								pos = pos55
								goto fail45
							ok52:
							}
							continue
						fail45:
							pos = pos43
							break
						}
						labels[8] = parser.text[pos41:pos]
					}
					// _
					if !_accept(parser, __Accepts, &pos, &perr) {
						goto fail39
					}
					// ")"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
						perr = _max(perr, pos)
						goto fail39
					}
					pos++
					goto ok31
				fail39:
					pos = pos34
					goto fail27
				ok31:
				}
				labels[9] = parser.text[pos30:pos]
			}
			goto ok7
		fail27:
			pos = pos10
			goto fail
		ok7:
		}
		labels[10] = parser.text[pos6:pos]
	}
	return _memoize(parser, _Def, start, pos, perr)
fail:
	return _memoize(parser, _Def, start, -1, perr)
}

func _DefNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [11]string
	use(labels)
	dp := parser.deltaPos[start][_Def]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Def}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Def"}
	// action
	// m:ModPath? ds0:(_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
	// m:ModPath?
	{
		pos1 := pos
		// ModPath?
		{
			nkids2 := len(node.Kids)
			pos3 := pos
			// ModPath
			if !_node(parser, _ModPathNode, node, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			node.Kids = node.Kids[:nkids2]
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// ds0:(_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
	{
		pos6 := pos
		// (_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
		{
			nkids7 := len(node.Kids)
			pos08 := pos
			// _ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…}
			{
				pos12 := pos
				nkids10 := len(node.Kids)
				// action
				// _ ("import" p:String)
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail13
				}
				// ("import" p:String)
				{
					nkids15 := len(node.Kids)
					pos016 := pos
					// "import" p:String
					// "import"
					if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
						goto fail13
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+6))
					pos += 6
					// p:String
					{
						pos18 := pos
						// String
						if !_node(parser, _StringNode, node, &pos) {
							goto fail13
						}
						labels[1] = parser.text[pos18:pos]
					}
					sub := _sub(parser, pos016, pos, node.Kids[nkids15:])
					node.Kids = append(node.Kids[:nkids15], sub)
				}
				goto ok9
			fail13:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				// action
				// _ ("(" dss:Def+ _ ")")
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail19
				}
				// ("(" dss:Def+ _ ")")
				{
					nkids21 := len(node.Kids)
					pos022 := pos
					// "(" dss:Def+ _ ")"
					// "("
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
						goto fail19
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					// dss:Def+
					{
						pos24 := pos
						// Def+
						// Def
						if !_node(parser, _DefNode, node, &pos) {
							goto fail19
						}
						for {
							nkids25 := len(node.Kids)
							pos26 := pos
							// Def
							if !_node(parser, _DefNode, node, &pos) {
								goto fail28
							}
							continue
						fail28:
							node.Kids = node.Kids[:nkids25]
							pos = pos26
							break
						}
						labels[2] = parser.text[pos24:pos]
					}
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail19
					}
					// ")"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
						goto fail19
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					sub := _sub(parser, pos022, pos, node.Kids[nkids21:])
					node.Kids = append(node.Kids[:nkids21], sub)
				}
				goto ok9
			fail19:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				// action
				// f:Fun
				{
					pos30 := pos
					// Fun
					if !_node(parser, _FunNode, node, &pos) {
						goto fail29
					}
					labels[3] = parser.text[pos30:pos]
				}
				goto ok9
			fail29:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				// action
				// v:Var
				{
					pos32 := pos
					// Var
					if !_node(parser, _VarNode, node, &pos) {
						goto fail31
					}
					labels[4] = parser.text[pos32:pos]
				}
				goto ok9
			fail31:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				// action
				// sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
				// sig:TypeSig
				{
					pos35 := pos
					// TypeSig
					if !_node(parser, _TypeSigNode, node, &pos) {
						goto fail33
					}
					labels[5] = parser.text[pos35:pos]
				}
				// ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
				{
					pos36 := pos
					// (t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
					{
						nkids37 := len(node.Kids)
						pos038 := pos
						// t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}
						{
							pos42 := pos
							nkids40 := len(node.Kids)
							// action
							// t0:Type
							{
								pos44 := pos
								// Type
								if !_node(parser, _TypeNode, node, &pos) {
									goto fail43
								}
								labels[6] = parser.text[pos44:pos]
							}
							goto ok39
						fail43:
							node.Kids = node.Kids[:nkids40]
							pos = pos42
							// action
							// m0:Fun
							{
								pos46 := pos
								// Fun
								if !_node(parser, _FunNode, node, &pos) {
									goto fail45
								}
								labels[7] = parser.text[pos46:pos]
							}
							goto ok39
						fail45:
							node.Kids = node.Kids[:nkids40]
							pos = pos42
							// action
							// _ "(" ds2:(Type/Fun)+ _ ")"
							// _
							if !_node(parser, __Node, node, &pos) {
								goto fail47
							}
							// "("
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
								goto fail47
							}
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
							pos++
							// ds2:(Type/Fun)+
							{
								pos49 := pos
								// (Type/Fun)+
								// (Type/Fun)
								{
									nkids54 := len(node.Kids)
									pos055 := pos
									// Type/Fun
									{
										pos59 := pos
										nkids57 := len(node.Kids)
										// Type
										if !_node(parser, _TypeNode, node, &pos) {
											goto fail60
										}
										goto ok56
									fail60:
										node.Kids = node.Kids[:nkids57]
										pos = pos59
										// Fun
										if !_node(parser, _FunNode, node, &pos) {
											goto fail61
										}
										goto ok56
									fail61:
										node.Kids = node.Kids[:nkids57]
										pos = pos59
										goto fail47
									ok56:
									}
									sub := _sub(parser, pos055, pos, node.Kids[nkids54:])
									node.Kids = append(node.Kids[:nkids54], sub)
								}
								for {
									nkids50 := len(node.Kids)
									pos51 := pos
									// (Type/Fun)
									{
										nkids62 := len(node.Kids)
										pos063 := pos
										// Type/Fun
										{
											pos67 := pos
											nkids65 := len(node.Kids)
											// Type
											if !_node(parser, _TypeNode, node, &pos) {
												goto fail68
											}
											goto ok64
										fail68:
											node.Kids = node.Kids[:nkids65]
											pos = pos67
											// Fun
											if !_node(parser, _FunNode, node, &pos) {
												goto fail69
											}
											goto ok64
										fail69:
											node.Kids = node.Kids[:nkids65]
											pos = pos67
											goto fail53
										ok64:
										}
										sub := _sub(parser, pos063, pos, node.Kids[nkids62:])
										node.Kids = append(node.Kids[:nkids62], sub)
									}
									continue
								fail53:
									node.Kids = node.Kids[:nkids50]
									pos = pos51
									break
								}
								labels[8] = parser.text[pos49:pos]
							}
							// _
							if !_node(parser, __Node, node, &pos) {
								goto fail47
							}
							// ")"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
								goto fail47
							}
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
							pos++
							goto ok39
						fail47:
							node.Kids = node.Kids[:nkids40]
							pos = pos42
							goto fail33
						ok39:
						}
						sub := _sub(parser, pos038, pos, node.Kids[nkids37:])
						node.Kids = append(node.Kids[:nkids37], sub)
					}
					labels[9] = parser.text[pos36:pos]
				}
				goto ok9
			fail33:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				goto fail
			ok9:
			}
			sub := _sub(parser, pos08, pos, node.Kids[nkids7:])
			node.Kids = append(node.Kids[:nkids7], sub)
		}
		labels[10] = parser.text[pos6:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _DefFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [11]string
	use(labels)
	pos, failure := _failMemo(parser, _Def, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Def",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Def}
	// action
	// m:ModPath? ds0:(_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
	// m:ModPath?
	{
		pos1 := pos
		// ModPath?
		{
			pos3 := pos
			// ModPath
			if !_fail(parser, _ModPathFail, errPos, failure, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// ds0:(_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
	{
		pos6 := pos
		// (_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
		// _ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…}
		{
			pos10 := pos
			// action
			// _ ("import" p:String)
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail11
			}
			// ("import" p:String)
			// "import" p:String
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"import\"",
					})
				}
				goto fail11
			}
			pos += 6
			// p:String
			{
				pos14 := pos
				// String
				if !_fail(parser, _StringFail, errPos, failure, &pos) {
					goto fail11
				}
				labels[1] = parser.text[pos14:pos]
			}
			goto ok7
		fail11:
			pos = pos10
			// action
			// _ ("(" dss:Def+ _ ")")
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail15
			}
			// ("(" dss:Def+ _ ")")
			// "(" dss:Def+ _ ")"
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"(\"",
					})
				}
				goto fail15
			}
			pos++
			// dss:Def+
			{
				pos18 := pos
				// Def+
				// Def
				if !_fail(parser, _DefFail, errPos, failure, &pos) {
					goto fail15
				}
				for {
					pos20 := pos
					// Def
					if !_fail(parser, _DefFail, errPos, failure, &pos) {
						goto fail22
					}
					continue
				fail22:
					pos = pos20
					break
				}
				labels[2] = parser.text[pos18:pos]
			}
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail15
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\")\"",
					})
				}
				goto fail15
			}
			pos++
			goto ok7
		fail15:
			pos = pos10
			// action
			// f:Fun
			{
				pos24 := pos
				// Fun
				if !_fail(parser, _FunFail, errPos, failure, &pos) {
					goto fail23
				}
				labels[3] = parser.text[pos24:pos]
			}
			goto ok7
		fail23:
			pos = pos10
			// action
			// v:Var
			{
				pos26 := pos
				// Var
				if !_fail(parser, _VarFail, errPos, failure, &pos) {
					goto fail25
				}
				labels[4] = parser.text[pos26:pos]
			}
			goto ok7
		fail25:
			pos = pos10
			// action
			// sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
			// sig:TypeSig
			{
				pos29 := pos
				// TypeSig
				if !_fail(parser, _TypeSigFail, errPos, failure, &pos) {
					goto fail27
				}
				labels[5] = parser.text[pos29:pos]
			}
			// ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
			{
				pos30 := pos
				// (t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
				// t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}
				{
					pos34 := pos
					// action
					// t0:Type
					{
						pos36 := pos
						// Type
						if !_fail(parser, _TypeFail, errPos, failure, &pos) {
							goto fail35
						}
						labels[6] = parser.text[pos36:pos]
					}
					goto ok31
				fail35:
					pos = pos34
					// action
					// m0:Fun
					{
						pos38 := pos
						// Fun
						if !_fail(parser, _FunFail, errPos, failure, &pos) {
							goto fail37
						}
						labels[7] = parser.text[pos38:pos]
					}
					goto ok31
				fail37:
					pos = pos34
					// action
					// _ "(" ds2:(Type/Fun)+ _ ")"
					// _
					if !_fail(parser, __Fail, errPos, failure, &pos) {
						goto fail39
					}
					// "("
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "\"(\"",
							})
						}
						goto fail39
					}
					pos++
					// ds2:(Type/Fun)+
					{
						pos41 := pos
						// (Type/Fun)+
						// (Type/Fun)
						// Type/Fun
						{
							pos49 := pos
							// Type
							if !_fail(parser, _TypeFail, errPos, failure, &pos) {
								goto fail50
							}
							goto ok46
						fail50:
							pos = pos49
							// Fun
							if !_fail(parser, _FunFail, errPos, failure, &pos) {
								goto fail51
							}
							goto ok46
						fail51:
							pos = pos49
							goto fail39
						ok46:
						}
						for {
							pos43 := pos
							// (Type/Fun)
							// Type/Fun
							{
								pos55 := pos
								// Type
								if !_fail(parser, _TypeFail, errPos, failure, &pos) {
									goto fail56
								}
								goto ok52
							fail56:
								pos = pos55
								// Fun
								if !_fail(parser, _FunFail, errPos, failure, &pos) {
									goto fail57
								}
								goto ok52
							fail57:
								pos = pos55
								goto fail45
							ok52:
							}
							continue
						fail45:
							pos = pos43
							break
						}
						labels[8] = parser.text[pos41:pos]
					}
					// _
					if !_fail(parser, __Fail, errPos, failure, &pos) {
						goto fail39
					}
					// ")"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "\")\"",
							})
						}
						goto fail39
					}
					pos++
					goto ok31
				fail39:
					pos = pos34
					goto fail27
				ok31:
				}
				labels[9] = parser.text[pos30:pos]
			}
			goto ok7
		fail27:
			pos = pos10
			goto fail
		ok7:
		}
		labels[10] = parser.text[pos6:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _DefAction(parser *_Parser, start int) (int, *[]Def) {
	var labels [11]string
	use(labels)
	var label2 [][]Def
	var label3 Def
	var label4 *Var
	var label7 Def
	var label9 []Def
	var label0 *ModPath
	var label1 String
	var label8 []Def
	var label10 []Def
	var label5 TypeSig
	var label6 Def
	dp := parser.deltaPos[start][_Def]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Def}
	n := parser.act[key]
	if n != nil {
		n := n.([]Def)
		return start + int(dp-1), &n
	}
	var node []Def
	pos := start
	// action
	{
		start0 := pos
		// m:ModPath? ds0:(_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
		// m:ModPath?
		{
			pos2 := pos
			// ModPath?
			{
				pos4 := pos
				label0 = new(ModPath)
				// ModPath
				if p, n := _ModPathAction(parser, pos); n == nil {
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
		// ds0:(_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
		{
			pos7 := pos
			// (_ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…})
			// _ ("import" p:String) {…}/_ ("(" dss:Def+ _ ")") {…}/f:Fun {…}/v:Var {…}/sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}) {…}
			{
				pos11 := pos
				var node10 []Def
				// action
				{
					start13 := pos
					// _ ("import" p:String)
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail12
					} else {
						pos = p
					}
					// ("import" p:String)
					// "import" p:String
					// "import"
					if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
						goto fail12
					}
					pos += 6
					// p:String
					{
						pos16 := pos
						// String
						if p, n := _StringAction(parser, pos); n == nil {
							goto fail12
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos16:pos]
					}
					label10 = func(
						start, end int, m *ModPath, p String) []Def {
						return []Def{
							&Import{location: loc(parser, start, end), Path: p.Data},
						}
					}(
						start13, pos, label0, label1)
				}
				goto ok8
			fail12:
				label10 = node10
				pos = pos11
				// action
				{
					start18 := pos
					// _ ("(" dss:Def+ _ ")")
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail17
					} else {
						pos = p
					}
					// ("(" dss:Def+ _ ")")
					// "(" dss:Def+ _ ")"
					// "("
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
						goto fail17
					}
					pos++
					// dss:Def+
					{
						pos21 := pos
						// Def+
						{
							var node24 []Def
							// Def
							if p, n := _DefAction(parser, pos); n == nil {
								goto fail17
							} else {
								node24 = *n
								pos = p
							}
							label2 = append(label2, node24)
						}
						for {
							pos23 := pos
							var node24 []Def
							// Def
							if p, n := _DefAction(parser, pos); n == nil {
								goto fail25
							} else {
								node24 = *n
								pos = p
							}
							label2 = append(label2, node24)
							continue
						fail25:
							pos = pos23
							break
						}
						labels[2] = parser.text[pos21:pos]
					}
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail17
					} else {
						pos = p
					}
					// ")"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
						goto fail17
					}
					pos++
					label10 = func(
						start, end int, dss [][]Def, m *ModPath, p String) []Def {
						return []Def{
							&SubMod{location: loc(parser, start, end), Defs: flatten(dss)},
						}
					}(
						start18, pos, label2, label0, label1)
				}
				goto ok8
			fail17:
				label10 = node10
				pos = pos11
				// action
				{
					start27 := pos
					// f:Fun
					{
						pos28 := pos
						// Fun
						if p, n := _FunAction(parser, pos); n == nil {
							goto fail26
						} else {
							label3 = *n
							pos = p
						}
						labels[3] = parser.text[pos28:pos]
					}
					label10 = func(
						start, end int, dss [][]Def, f Def, m *ModPath, p String) []Def {
						return []Def{f}
					}(
						start27, pos, label2, label3, label0, label1)
				}
				goto ok8
			fail26:
				label10 = node10
				pos = pos11
				// action
				{
					start30 := pos
					// v:Var
					{
						pos31 := pos
						// Var
						if p, n := _VarAction(parser, pos); n == nil {
							goto fail29
						} else {
							label4 = *n
							pos = p
						}
						labels[4] = parser.text[pos31:pos]
					}
					label10 = func(
						start, end int, dss [][]Def, f Def, m *ModPath, p String, v *Var) []Def {
						return []Def{v}
					}(
						start30, pos, label2, label3, label0, label1, label4)
				}
				goto ok8
			fail29:
				label10 = node10
				pos = pos11
				// action
				{
					start33 := pos
					// sig:TypeSig ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
					// sig:TypeSig
					{
						pos35 := pos
						// TypeSig
						if p, n := _TypeSigAction(parser, pos); n == nil {
							goto fail32
						} else {
							label5 = *n
							pos = p
						}
						labels[5] = parser.text[pos35:pos]
					}
					// ds1:(t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
					{
						pos36 := pos
						// (t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…})
						// t0:Type {…}/m0:Fun {…}/_ "(" ds2:(Type/Fun)+ _ ")" {…}
						{
							pos40 := pos
							var node39 []Def
							// action
							{
								start42 := pos
								// t0:Type
								{
									pos43 := pos
									// Type
									if p, n := _TypeAction(parser, pos); n == nil {
										goto fail41
									} else {
										label6 = *n
										pos = p
									}
									labels[6] = parser.text[pos43:pos]
								}
								label9 = func(
									start, end int, dss [][]Def, f Def, m *ModPath, p String, sig TypeSig, t0 Def, v *Var) []Def {
									return []Def{t0}
								}(
									start42, pos, label2, label3, label0, label1, label5, label6, label4)
							}
							goto ok37
						fail41:
							label9 = node39
							pos = pos40
							// action
							{
								start45 := pos
								// m0:Fun
								{
									pos46 := pos
									// Fun
									if p, n := _FunAction(parser, pos); n == nil {
										goto fail44
									} else {
										label7 = *n
										pos = p
									}
									labels[7] = parser.text[pos46:pos]
								}
								label9 = func(
									start, end int, dss [][]Def, f Def, m *ModPath, m0 Def, p String, sig TypeSig, t0 Def, v *Var) []Def {
									return []Def{m0}
								}(
									start45, pos, label2, label3, label0, label7, label1, label5, label6, label4)
							}
							goto ok37
						fail44:
							label9 = node39
							pos = pos40
							// action
							{
								start48 := pos
								// _ "(" ds2:(Type/Fun)+ _ ")"
								// _
								if p, n := __Action(parser, pos); n == nil {
									goto fail47
								} else {
									pos = p
								}
								// "("
								if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
									goto fail47
								}
								pos++
								// ds2:(Type/Fun)+
								{
									pos50 := pos
									// (Type/Fun)+
									{
										var node53 Def
										// (Type/Fun)
										// Type/Fun
										{
											pos58 := pos
											var node57 Def
											// Type
											if p, n := _TypeAction(parser, pos); n == nil {
												goto fail59
											} else {
												node53 = *n
												pos = p
											}
											goto ok55
										fail59:
											node53 = node57
											pos = pos58
											// Fun
											if p, n := _FunAction(parser, pos); n == nil {
												goto fail60
											} else {
												node53 = *n
												pos = p
											}
											goto ok55
										fail60:
											node53 = node57
											pos = pos58
											goto fail47
										ok55:
										}
										label8 = append(label8, node53)
									}
									for {
										pos52 := pos
										var node53 Def
										// (Type/Fun)
										// Type/Fun
										{
											pos64 := pos
											var node63 Def
											// Type
											if p, n := _TypeAction(parser, pos); n == nil {
												goto fail65
											} else {
												node53 = *n
												pos = p
											}
											goto ok61
										fail65:
											node53 = node63
											pos = pos64
											// Fun
											if p, n := _FunAction(parser, pos); n == nil {
												goto fail66
											} else {
												node53 = *n
												pos = p
											}
											goto ok61
										fail66:
											node53 = node63
											pos = pos64
											goto fail54
										ok61:
										}
										label8 = append(label8, node53)
										continue
									fail54:
										pos = pos52
										break
									}
									labels[8] = parser.text[pos50:pos]
								}
								// _
								if p, n := __Action(parser, pos); n == nil {
									goto fail47
								} else {
									pos = p
								}
								// ")"
								if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
									goto fail47
								}
								pos++
								label9 = func(
									start, end int, ds2 []Def, dss [][]Def, f Def, m *ModPath, m0 Def, p String, sig TypeSig, t0 Def, v *Var) []Def {
									return []Def(ds2)
								}(
									start48, pos, label8, label2, label3, label0, label7, label1, label5, label6, label4)
							}
							goto ok37
						fail47:
							label9 = node39
							pos = pos40
							goto fail32
						ok37:
						}
						labels[9] = parser.text[pos36:pos]
					}
					label10 = func(
						start, end int, ds1 []Def, ds2 []Def, dss [][]Def, f Def, m *ModPath, m0 Def, p String, sig TypeSig, t0 Def, v *Var) []Def {
						return []Def(distSig(sig, ds1))
					}(
						start33, pos, label9, label8, label2, label3, label0, label7, label1, label5, label6, label4)
				}
				goto ok8
			fail32:
				label10 = node10
				pos = pos11
				goto fail
			ok8:
			}
			labels[10] = parser.text[pos7:pos]
		}
		node = func(
			start, end int, ds0 []Def, ds1 []Def, ds2 []Def, dss [][]Def, f Def, m *ModPath, m0 Def, p String, sig TypeSig, t0 Def, v *Var) []Def {
			return []Def(distMod(parser, m, ds0))
		}(
			start0, pos, label10, label9, label8, label2, label3, label0, label7, label1, label5, label6, label4)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ModPathAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [1]string
	use(labels)
	if dp, de, ok := _memo(parser, _ModPath, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// n:ModName+
	{
		pos0 := pos
		// ModName+
		// ModName
		if !_accept(parser, _ModNameAccepts, &pos, &perr) {
			goto fail
		}
		for {
			pos2 := pos
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail4
			}
			continue
		fail4:
			pos = pos2
			break
		}
		labels[0] = parser.text[pos0:pos]
	}
	return _memoize(parser, _ModPath, start, pos, perr)
fail:
	return _memoize(parser, _ModPath, start, -1, perr)
}

func _ModPathNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [1]string
	use(labels)
	dp := parser.deltaPos[start][_ModPath]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _ModPath}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "ModPath"}
	// action
	// n:ModName+
	{
		pos0 := pos
		// ModName+
		// ModName
		if !_node(parser, _ModNameNode, node, &pos) {
			goto fail
		}
		for {
			nkids1 := len(node.Kids)
			pos2 := pos
			// ModName
			if !_node(parser, _ModNameNode, node, &pos) {
				goto fail4
			}
			continue
		fail4:
			node.Kids = node.Kids[:nkids1]
			pos = pos2
			break
		}
		labels[0] = parser.text[pos0:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _ModPathFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [1]string
	use(labels)
	pos, failure := _failMemo(parser, _ModPath, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "ModPath",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _ModPath}
	// action
	// n:ModName+
	{
		pos0 := pos
		// ModName+
		// ModName
		if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
			goto fail
		}
		for {
			pos2 := pos
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail4
			}
			continue
		fail4:
			pos = pos2
			break
		}
		labels[0] = parser.text[pos0:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _ModPathAction(parser *_Parser, start int) (int, *ModPath) {
	var labels [1]string
	use(labels)
	var label0 []Ident
	dp := parser.deltaPos[start][_ModPath]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _ModPath}
	n := parser.act[key]
	if n != nil {
		n := n.(ModPath)
		return start + int(dp-1), &n
	}
	var node ModPath
	pos := start
	// action
	{
		start0 := pos
		// n:ModName+
		{
			pos1 := pos
			// ModName+
			{
				var node4 Ident
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail
				} else {
					node4 = *n
					pos = p
				}
				label0 = append(label0, node4)
			}
			for {
				pos3 := pos
				var node4 Ident
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail5
				} else {
					node4 = *n
					pos = p
				}
				label0 = append(label0, node4)
				continue
			fail5:
				pos = pos3
				break
			}
			labels[0] = parser.text[pos1:pos]
		}
		node = func(
			start, end int, n []Ident) ModPath {
			n = append([]Ident{{
				Text: parser.data.(*Parser).mod,
			}}, n...)
			return ModPath(n)
		}(
			start0, pos, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FunAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Fun, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// tps:TypeParms? _ f:("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
	// tps:TypeParms?
	{
		pos1 := pos
		// TypeParms?
		{
			pos3 := pos
			// TypeParms
			if !_accept(parser, _TypeParmsAccepts, &pos, &perr) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// f:("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
	{
		pos6 := pos
		// ("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
		// action
		// "[" sig:FunSig _ "|" ss:Stmts _ "]"
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// sig:FunSig
		{
			pos8 := pos
			// FunSig
			if !_accept(parser, _FunSigAccepts, &pos, &perr) {
				goto fail
			}
			labels[1] = parser.text[pos8:pos]
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
		// ss:Stmts
		{
			pos9 := pos
			// Stmts
			if !_accept(parser, _StmtsAccepts, &pos, &perr) {
				goto fail
			}
			labels[2] = parser.text[pos9:pos]
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
		labels[3] = parser.text[pos6:pos]
	}
	return _memoize(parser, _Fun, start, pos, perr)
fail:
	return _memoize(parser, _Fun, start, -1, perr)
}

func _FunNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_Fun]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Fun}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Fun"}
	// action
	// tps:TypeParms? _ f:("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
	// tps:TypeParms?
	{
		pos1 := pos
		// TypeParms?
		{
			nkids2 := len(node.Kids)
			pos3 := pos
			// TypeParms
			if !_node(parser, _TypeParmsNode, node, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			node.Kids = node.Kids[:nkids2]
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// f:("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
	{
		pos6 := pos
		// ("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
		{
			nkids7 := len(node.Kids)
			pos08 := pos
			// action
			// "[" sig:FunSig _ "|" ss:Stmts _ "]"
			// "["
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// sig:FunSig
			{
				pos10 := pos
				// FunSig
				if !_node(parser, _FunSigNode, node, &pos) {
					goto fail
				}
				labels[1] = parser.text[pos10:pos]
			}
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// ss:Stmts
			{
				pos11 := pos
				// Stmts
				if !_node(parser, _StmtsNode, node, &pos) {
					goto fail
				}
				labels[2] = parser.text[pos11:pos]
			}
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail
			}
			// "]"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			sub := _sub(parser, pos08, pos, node.Kids[nkids7:])
			node.Kids = append(node.Kids[:nkids7], sub)
		}
		labels[3] = parser.text[pos6:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _FunFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
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
	// tps:TypeParms? _ f:("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
	// tps:TypeParms?
	{
		pos1 := pos
		// TypeParms?
		{
			pos3 := pos
			// TypeParms
			if !_fail(parser, _TypeParmsFail, errPos, failure, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// f:("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
	{
		pos6 := pos
		// ("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
		// action
		// "[" sig:FunSig _ "|" ss:Stmts _ "]"
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
			pos8 := pos
			// FunSig
			if !_fail(parser, _FunSigFail, errPos, failure, &pos) {
				goto fail
			}
			labels[1] = parser.text[pos8:pos]
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
		// ss:Stmts
		{
			pos9 := pos
			// Stmts
			if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
				goto fail
			}
			labels[2] = parser.text[pos9:pos]
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
		labels[3] = parser.text[pos6:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _FunAction(parser *_Parser, start int) (int, *Def) {
	var labels [4]string
	use(labels)
	var label0 *[]Parm
	var label1 *Fun
	var label2 []Stmt
	var label3 (*Fun)
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
		// tps:TypeParms? _ f:("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
		// tps:TypeParms?
		{
			pos2 := pos
			// TypeParms?
			{
				pos4 := pos
				label0 = new([]Parm)
				// TypeParms
				if p, n := _TypeParmsAction(parser, pos); n == nil {
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
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// f:("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
		{
			pos7 := pos
			// ("[" sig:FunSig _ "|" ss:Stmts _ "]" {…})
			// action
			{
				start8 := pos
				// "[" sig:FunSig _ "|" ss:Stmts _ "]"
				// "["
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
					goto fail
				}
				pos++
				// sig:FunSig
				{
					pos10 := pos
					// FunSig
					if p, n := _FunSigAction(parser, pos); n == nil {
						goto fail
					} else {
						label1 = *n
						pos = p
					}
					labels[1] = parser.text[pos10:pos]
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
				// ss:Stmts
				{
					pos11 := pos
					// Stmts
					if p, n := _StmtsAction(parser, pos); n == nil {
						goto fail
					} else {
						label2 = *n
						pos = p
					}
					labels[2] = parser.text[pos11:pos]
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
				label3 = func(
					start, end int, sig *Fun, ss []Stmt, tps *[]Parm) *Fun {
					copy := *sig
					copy.location = loc(parser, start, end)
					copy.Stmts = ss
					return (*Fun)(&copy)
				}(
					start8, pos, label1, label2, label0)
			}
			labels[3] = parser.text[pos7:pos]
		}
		node = func(
			start, end int, f *Fun, sig *Fun, ss []Stmt, tps *[]Parm) Def {
			if tps != nil {
				copy := *f
				copy.TypeParms = *tps
				return Def(&copy)
			}
			return Def(f)
		}(
			start0, pos, label3, label1, label2, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FunSigAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [9]string
	use(labels)
	if dp, de, ok := _memo(parser, _FunSig, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// ps:(id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+) r:Ret?
	// ps:(id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+)
	{
		pos1 := pos
		// (id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+)
		// id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+
		{
			pos5 := pos
			// action
			// id0:Ident
			{
				pos7 := pos
				// Ident
				if !_accept(parser, _IdentAccepts, &pos, &perr) {
					goto fail6
				}
				labels[0] = parser.text[pos7:pos]
			}
			goto ok2
		fail6:
			pos = pos5
			// action
			// o:Op id1:Ident t0:TypeName
			// o:Op
			{
				pos10 := pos
				// Op
				if !_accept(parser, _OpAccepts, &pos, &perr) {
					goto fail8
				}
				labels[1] = parser.text[pos10:pos]
			}
			// id1:Ident
			{
				pos11 := pos
				// Ident
				if !_accept(parser, _IdentAccepts, &pos, &perr) {
					goto fail8
				}
				labels[2] = parser.text[pos11:pos]
			}
			// t0:TypeName
			{
				pos12 := pos
				// TypeName
				if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
					goto fail8
				}
				labels[3] = parser.text[pos12:pos]
			}
			goto ok2
		fail8:
			pos = pos5
			// (c:IdentC id2:Ident t1:TypeName {…})+
			// (c:IdentC id2:Ident t1:TypeName {…})
			// action
			// c:IdentC id2:Ident t1:TypeName
			// c:IdentC
			{
				pos19 := pos
				// IdentC
				if !_accept(parser, _IdentCAccepts, &pos, &perr) {
					goto fail13
				}
				labels[4] = parser.text[pos19:pos]
			}
			// id2:Ident
			{
				pos20 := pos
				// Ident
				if !_accept(parser, _IdentAccepts, &pos, &perr) {
					goto fail13
				}
				labels[5] = parser.text[pos20:pos]
			}
			// t1:TypeName
			{
				pos21 := pos
				// TypeName
				if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
					goto fail13
				}
				labels[6] = parser.text[pos21:pos]
			}
			for {
				pos15 := pos
				// (c:IdentC id2:Ident t1:TypeName {…})
				// action
				// c:IdentC id2:Ident t1:TypeName
				// c:IdentC
				{
					pos23 := pos
					// IdentC
					if !_accept(parser, _IdentCAccepts, &pos, &perr) {
						goto fail17
					}
					labels[4] = parser.text[pos23:pos]
				}
				// id2:Ident
				{
					pos24 := pos
					// Ident
					if !_accept(parser, _IdentAccepts, &pos, &perr) {
						goto fail17
					}
					labels[5] = parser.text[pos24:pos]
				}
				// t1:TypeName
				{
					pos25 := pos
					// TypeName
					if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
						goto fail17
					}
					labels[6] = parser.text[pos25:pos]
				}
				continue
			fail17:
				pos = pos15
				break
			}
			goto ok2
		fail13:
			pos = pos5
			goto fail
		ok2:
		}
		labels[7] = parser.text[pos1:pos]
	}
	// r:Ret?
	{
		pos26 := pos
		// Ret?
		{
			pos28 := pos
			// Ret
			if !_accept(parser, _RetAccepts, &pos, &perr) {
				goto fail29
			}
			goto ok30
		fail29:
			pos = pos28
		ok30:
		}
		labels[8] = parser.text[pos26:pos]
	}
	return _memoize(parser, _FunSig, start, pos, perr)
fail:
	return _memoize(parser, _FunSig, start, -1, perr)
}

func _FunSigNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [9]string
	use(labels)
	dp := parser.deltaPos[start][_FunSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _FunSig}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "FunSig"}
	// action
	// ps:(id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+) r:Ret?
	// ps:(id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+)
	{
		pos1 := pos
		// (id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+)
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+
			{
				pos7 := pos
				nkids5 := len(node.Kids)
				// action
				// id0:Ident
				{
					pos9 := pos
					// Ident
					if !_node(parser, _IdentNode, node, &pos) {
						goto fail8
					}
					labels[0] = parser.text[pos9:pos]
				}
				goto ok4
			fail8:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				// action
				// o:Op id1:Ident t0:TypeName
				// o:Op
				{
					pos12 := pos
					// Op
					if !_node(parser, _OpNode, node, &pos) {
						goto fail10
					}
					labels[1] = parser.text[pos12:pos]
				}
				// id1:Ident
				{
					pos13 := pos
					// Ident
					if !_node(parser, _IdentNode, node, &pos) {
						goto fail10
					}
					labels[2] = parser.text[pos13:pos]
				}
				// t0:TypeName
				{
					pos14 := pos
					// TypeName
					if !_node(parser, _TypeNameNode, node, &pos) {
						goto fail10
					}
					labels[3] = parser.text[pos14:pos]
				}
				goto ok4
			fail10:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				// (c:IdentC id2:Ident t1:TypeName {…})+
				// (c:IdentC id2:Ident t1:TypeName {…})
				{
					nkids20 := len(node.Kids)
					pos021 := pos
					// action
					// c:IdentC id2:Ident t1:TypeName
					// c:IdentC
					{
						pos23 := pos
						// IdentC
						if !_node(parser, _IdentCNode, node, &pos) {
							goto fail15
						}
						labels[4] = parser.text[pos23:pos]
					}
					// id2:Ident
					{
						pos24 := pos
						// Ident
						if !_node(parser, _IdentNode, node, &pos) {
							goto fail15
						}
						labels[5] = parser.text[pos24:pos]
					}
					// t1:TypeName
					{
						pos25 := pos
						// TypeName
						if !_node(parser, _TypeNameNode, node, &pos) {
							goto fail15
						}
						labels[6] = parser.text[pos25:pos]
					}
					sub := _sub(parser, pos021, pos, node.Kids[nkids20:])
					node.Kids = append(node.Kids[:nkids20], sub)
				}
				for {
					nkids16 := len(node.Kids)
					pos17 := pos
					// (c:IdentC id2:Ident t1:TypeName {…})
					{
						nkids26 := len(node.Kids)
						pos027 := pos
						// action
						// c:IdentC id2:Ident t1:TypeName
						// c:IdentC
						{
							pos29 := pos
							// IdentC
							if !_node(parser, _IdentCNode, node, &pos) {
								goto fail19
							}
							labels[4] = parser.text[pos29:pos]
						}
						// id2:Ident
						{
							pos30 := pos
							// Ident
							if !_node(parser, _IdentNode, node, &pos) {
								goto fail19
							}
							labels[5] = parser.text[pos30:pos]
						}
						// t1:TypeName
						{
							pos31 := pos
							// TypeName
							if !_node(parser, _TypeNameNode, node, &pos) {
								goto fail19
							}
							labels[6] = parser.text[pos31:pos]
						}
						sub := _sub(parser, pos027, pos, node.Kids[nkids26:])
						node.Kids = append(node.Kids[:nkids26], sub)
					}
					continue
				fail19:
					node.Kids = node.Kids[:nkids16]
					pos = pos17
					break
				}
				goto ok4
			fail15:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				goto fail
			ok4:
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[7] = parser.text[pos1:pos]
	}
	// r:Ret?
	{
		pos32 := pos
		// Ret?
		{
			nkids33 := len(node.Kids)
			pos34 := pos
			// Ret
			if !_node(parser, _RetNode, node, &pos) {
				goto fail35
			}
			goto ok36
		fail35:
			node.Kids = node.Kids[:nkids33]
			pos = pos34
		ok36:
		}
		labels[8] = parser.text[pos32:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _FunSigFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [9]string
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
	// ps:(id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+) r:Ret?
	// ps:(id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+)
	{
		pos1 := pos
		// (id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+)
		// id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+
		{
			pos5 := pos
			// action
			// id0:Ident
			{
				pos7 := pos
				// Ident
				if !_fail(parser, _IdentFail, errPos, failure, &pos) {
					goto fail6
				}
				labels[0] = parser.text[pos7:pos]
			}
			goto ok2
		fail6:
			pos = pos5
			// action
			// o:Op id1:Ident t0:TypeName
			// o:Op
			{
				pos10 := pos
				// Op
				if !_fail(parser, _OpFail, errPos, failure, &pos) {
					goto fail8
				}
				labels[1] = parser.text[pos10:pos]
			}
			// id1:Ident
			{
				pos11 := pos
				// Ident
				if !_fail(parser, _IdentFail, errPos, failure, &pos) {
					goto fail8
				}
				labels[2] = parser.text[pos11:pos]
			}
			// t0:TypeName
			{
				pos12 := pos
				// TypeName
				if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
					goto fail8
				}
				labels[3] = parser.text[pos12:pos]
			}
			goto ok2
		fail8:
			pos = pos5
			// (c:IdentC id2:Ident t1:TypeName {…})+
			// (c:IdentC id2:Ident t1:TypeName {…})
			// action
			// c:IdentC id2:Ident t1:TypeName
			// c:IdentC
			{
				pos19 := pos
				// IdentC
				if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
					goto fail13
				}
				labels[4] = parser.text[pos19:pos]
			}
			// id2:Ident
			{
				pos20 := pos
				// Ident
				if !_fail(parser, _IdentFail, errPos, failure, &pos) {
					goto fail13
				}
				labels[5] = parser.text[pos20:pos]
			}
			// t1:TypeName
			{
				pos21 := pos
				// TypeName
				if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
					goto fail13
				}
				labels[6] = parser.text[pos21:pos]
			}
			for {
				pos15 := pos
				// (c:IdentC id2:Ident t1:TypeName {…})
				// action
				// c:IdentC id2:Ident t1:TypeName
				// c:IdentC
				{
					pos23 := pos
					// IdentC
					if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
						goto fail17
					}
					labels[4] = parser.text[pos23:pos]
				}
				// id2:Ident
				{
					pos24 := pos
					// Ident
					if !_fail(parser, _IdentFail, errPos, failure, &pos) {
						goto fail17
					}
					labels[5] = parser.text[pos24:pos]
				}
				// t1:TypeName
				{
					pos25 := pos
					// TypeName
					if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
						goto fail17
					}
					labels[6] = parser.text[pos25:pos]
				}
				continue
			fail17:
				pos = pos15
				break
			}
			goto ok2
		fail13:
			pos = pos5
			goto fail
		ok2:
		}
		labels[7] = parser.text[pos1:pos]
	}
	// r:Ret?
	{
		pos26 := pos
		// Ret?
		{
			pos28 := pos
			// Ret
			if !_fail(parser, _RetFail, errPos, failure, &pos) {
				goto fail29
			}
			goto ok30
		fail29:
			pos = pos28
		ok30:
		}
		labels[8] = parser.text[pos26:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _FunSigAction(parser *_Parser, start int) (int, **Fun) {
	var labels [9]string
	use(labels)
	var label0 Ident
	var label1 Ident
	var label4 Ident
	var label5 Ident
	var label6 TypeName
	var label2 Ident
	var label3 TypeName
	var label7 []parm
	var label8 *TypeName
	dp := parser.deltaPos[start][_FunSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _FunSig}
	n := parser.act[key]
	if n != nil {
		n := n.(*Fun)
		return start + int(dp-1), &n
	}
	var node *Fun
	pos := start
	// action
	{
		start0 := pos
		// ps:(id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+) r:Ret?
		// ps:(id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+)
		{
			pos2 := pos
			// (id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+)
			// id0:Ident {…}/o:Op id1:Ident t0:TypeName {…}/(c:IdentC id2:Ident t1:TypeName {…})+
			{
				pos6 := pos
				var node5 []parm
				// action
				{
					start8 := pos
					// id0:Ident
					{
						pos9 := pos
						// Ident
						if p, n := _IdentAction(parser, pos); n == nil {
							goto fail7
						} else {
							label0 = *n
							pos = p
						}
						labels[0] = parser.text[pos9:pos]
					}
					label7 = func(
						start, end int, id0 Ident) []parm {
						return []parm{{key: id0}}
					}(
						start8, pos, label0)
				}
				goto ok3
			fail7:
				label7 = node5
				pos = pos6
				// action
				{
					start11 := pos
					// o:Op id1:Ident t0:TypeName
					// o:Op
					{
						pos13 := pos
						// Op
						if p, n := _OpAction(parser, pos); n == nil {
							goto fail10
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos13:pos]
					}
					// id1:Ident
					{
						pos14 := pos
						// Ident
						if p, n := _IdentAction(parser, pos); n == nil {
							goto fail10
						} else {
							label2 = *n
							pos = p
						}
						labels[2] = parser.text[pos14:pos]
					}
					// t0:TypeName
					{
						pos15 := pos
						// TypeName
						if p, n := _TypeNameAction(parser, pos); n == nil {
							goto fail10
						} else {
							label3 = *n
							pos = p
						}
						labels[3] = parser.text[pos15:pos]
					}
					label7 = func(
						start, end int, id0 Ident, id1 Ident, o Ident, t0 TypeName) []parm {
						return []parm{{key: o, name: id1, typ: t0}}
					}(
						start11, pos, label0, label2, label1, label3)
				}
				goto ok3
			fail10:
				label7 = node5
				pos = pos6
				// (c:IdentC id2:Ident t1:TypeName {…})+
				{
					var node19 parm
					// (c:IdentC id2:Ident t1:TypeName {…})
					// action
					{
						start21 := pos
						// c:IdentC id2:Ident t1:TypeName
						// c:IdentC
						{
							pos23 := pos
							// IdentC
							if p, n := _IdentCAction(parser, pos); n == nil {
								goto fail16
							} else {
								label4 = *n
								pos = p
							}
							labels[4] = parser.text[pos23:pos]
						}
						// id2:Ident
						{
							pos24 := pos
							// Ident
							if p, n := _IdentAction(parser, pos); n == nil {
								goto fail16
							} else {
								label5 = *n
								pos = p
							}
							labels[5] = parser.text[pos24:pos]
						}
						// t1:TypeName
						{
							pos25 := pos
							// TypeName
							if p, n := _TypeNameAction(parser, pos); n == nil {
								goto fail16
							} else {
								label6 = *n
								pos = p
							}
							labels[6] = parser.text[pos25:pos]
						}
						node19 = func(
							start, end int, c Ident, id0 Ident, id1 Ident, id2 Ident, o Ident, t0 TypeName, t1 TypeName) parm {
							return parm{key: c, name: id2, typ: t1}
						}(
							start21, pos, label4, label0, label2, label5, label1, label3, label6)
					}
					label7 = append(label7, node19)
				}
				for {
					pos18 := pos
					var node19 parm
					// (c:IdentC id2:Ident t1:TypeName {…})
					// action
					{
						start26 := pos
						// c:IdentC id2:Ident t1:TypeName
						// c:IdentC
						{
							pos28 := pos
							// IdentC
							if p, n := _IdentCAction(parser, pos); n == nil {
								goto fail20
							} else {
								label4 = *n
								pos = p
							}
							labels[4] = parser.text[pos28:pos]
						}
						// id2:Ident
						{
							pos29 := pos
							// Ident
							if p, n := _IdentAction(parser, pos); n == nil {
								goto fail20
							} else {
								label5 = *n
								pos = p
							}
							labels[5] = parser.text[pos29:pos]
						}
						// t1:TypeName
						{
							pos30 := pos
							// TypeName
							if p, n := _TypeNameAction(parser, pos); n == nil {
								goto fail20
							} else {
								label6 = *n
								pos = p
							}
							labels[6] = parser.text[pos30:pos]
						}
						node19 = func(
							start, end int, c Ident, id0 Ident, id1 Ident, id2 Ident, o Ident, t0 TypeName, t1 TypeName) parm {
							return parm{key: c, name: id2, typ: t1}
						}(
							start26, pos, label4, label0, label2, label5, label1, label3, label6)
					}
					label7 = append(label7, node19)
					continue
				fail20:
					pos = pos18
					break
				}
				goto ok3
			fail16:
				label7 = node5
				pos = pos6
				goto fail
			ok3:
			}
			labels[7] = parser.text[pos2:pos]
		}
		// r:Ret?
		{
			pos31 := pos
			// Ret?
			{
				pos33 := pos
				label8 = new(TypeName)
				// Ret
				if p, n := _RetAction(parser, pos); n == nil {
					goto fail34
				} else {
					*label8 = *n
					pos = p
				}
				goto ok35
			fail34:
				label8 = nil
				pos = pos33
			ok35:
			}
			labels[8] = parser.text[pos31:pos]
		}
		node = func(
			start, end int, c Ident, id0 Ident, id1 Ident, id2 Ident, o Ident, ps []parm, r *TypeName, t0 TypeName, t1 TypeName) *Fun {
			if len(ps) == 1 && ps[0].name.Text == "" {
				p := ps[0]
				return &Fun{
					location: location{p.key.start, p.typ.end},
					Sel:      p.key.Text,
					Ret:      r,
				}
			}
			var sel string
			var parms []Parm
			for i := range ps {
				p := &ps[i]
				sel += p.key.Text
				parms = append(parms, Parm{
					location: location{p.key.start, p.typ.end},
					Name:     p.name.Text,
					Type:     &p.typ,
				})
			}
			return &Fun{Sel: sel, Parms: parms, Ret: r}
		}(
			start0, pos, label4, label0, label2, label5, label1, label7, label8, label3, label6)
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

func _RetNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [1]string
	use(labels)
	dp := parser.deltaPos[start][_Ret]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ret}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Ret"}
	// action
	// _ "^" t:TypeName
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// "^"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
	pos++
	// t:TypeName
	{
		pos1 := pos
		// TypeName
		if !_node(parser, _TypeNameNode, node, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _VarAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Var, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// n:Ident _ ":=" _ "[" ss:Stmts _ "]"
	// n:Ident
	{
		pos1 := pos
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
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
	// ss:Stmts
	{
		pos2 := pos
		// Stmts
		if !_accept(parser, _StmtsAccepts, &pos, &perr) {
			goto fail
		}
		labels[1] = parser.text[pos2:pos]
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
	return _memoize(parser, _Var, start, pos, perr)
fail:
	return _memoize(parser, _Var, start, -1, perr)
}

func _VarNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_Var]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Var}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Var"}
	// action
	// n:Ident _ ":=" _ "[" ss:Stmts _ "]"
	// n:Ident
	{
		pos1 := pos
		// Ident
		if !_node(parser, _IdentNode, node, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// ":="
	if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
	pos += 2
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// "["
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
	pos++
	// ss:Stmts
	{
		pos2 := pos
		// Stmts
		if !_node(parser, _StmtsNode, node, &pos) {
			goto fail
		}
		labels[1] = parser.text[pos2:pos]
	}
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// "]"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
	pos++
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _VarFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _Var, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Var",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Var}
	// action
	// n:Ident _ ":=" _ "[" ss:Stmts _ "]"
	// n:Ident
	{
		pos1 := pos
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
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
	// ss:Stmts
	{
		pos2 := pos
		// Stmts
		if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
			goto fail
		}
		labels[1] = parser.text[pos2:pos]
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
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _VarAction(parser *_Parser, start int) (int, **Var) {
	var labels [2]string
	use(labels)
	var label0 Ident
	var label1 []Stmt
	dp := parser.deltaPos[start][_Var]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Var}
	n := parser.act[key]
	if n != nil {
		n := n.(*Var)
		return start + int(dp-1), &n
	}
	var node *Var
	pos := start
	// action
	{
		start0 := pos
		// n:Ident _ ":=" _ "[" ss:Stmts _ "]"
		// n:Ident
		{
			pos2 := pos
			// Ident
			if p, n := _IdentAction(parser, pos); n == nil {
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
		// ss:Stmts
		{
			pos3 := pos
			// Stmts
			if p, n := _StmtsAction(parser, pos); n == nil {
				goto fail
			} else {
				label1 = *n
				pos = p
			}
			labels[1] = parser.text[pos3:pos]
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
		node = func(
			start, end int, n Ident, ss []Stmt) *Var {
			return &Var{
				location: location{n.start, loc1(parser, end)},
				Name:     n.Text,
				Val:      ss,
			}
		}(
			start0, pos, label0, label1)
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
	// ps:TypeParms? n:(Ident/Op)
	// ps:TypeParms?
	{
		pos1 := pos
		// TypeParms?
		{
			pos3 := pos
			// TypeParms
			if !_accept(parser, _TypeParmsAccepts, &pos, &perr) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:(Ident/Op)
	{
		pos6 := pos
		// (Ident/Op)
		// Ident/Op
		{
			pos10 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail11
			}
			goto ok7
		fail11:
			pos = pos10
			// Op
			if !_accept(parser, _OpAccepts, &pos, &perr) {
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
	return _memoize(parser, _TypeSig, start, pos, perr)
fail:
	return _memoize(parser, _TypeSig, start, -1, perr)
}

func _TypeSigNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_TypeSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeSig}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "TypeSig"}
	// action
	// ps:TypeParms? n:(Ident/Op)
	// ps:TypeParms?
	{
		pos1 := pos
		// TypeParms?
		{
			nkids2 := len(node.Kids)
			pos3 := pos
			// TypeParms
			if !_node(parser, _TypeParmsNode, node, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			node.Kids = node.Kids[:nkids2]
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:(Ident/Op)
	{
		pos6 := pos
		// (Ident/Op)
		{
			nkids7 := len(node.Kids)
			pos08 := pos
			// Ident/Op
			{
				pos12 := pos
				nkids10 := len(node.Kids)
				// Ident
				if !_node(parser, _IdentNode, node, &pos) {
					goto fail13
				}
				goto ok9
			fail13:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				// Op
				if !_node(parser, _OpNode, node, &pos) {
					goto fail14
				}
				goto ok9
			fail14:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				goto fail
			ok9:
			}
			sub := _sub(parser, pos08, pos, node.Kids[nkids7:])
			node.Kids = append(node.Kids[:nkids7], sub)
		}
		labels[1] = parser.text[pos6:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	// ps:TypeParms? n:(Ident/Op)
	// ps:TypeParms?
	{
		pos1 := pos
		// TypeParms?
		{
			pos3 := pos
			// TypeParms
			if !_fail(parser, _TypeParmsFail, errPos, failure, &pos) {
				goto fail4
			}
			goto ok5
		fail4:
			pos = pos3
		ok5:
		}
		labels[0] = parser.text[pos1:pos]
	}
	// n:(Ident/Op)
	{
		pos6 := pos
		// (Ident/Op)
		// Ident/Op
		{
			pos10 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail11
			}
			goto ok7
		fail11:
			pos = pos10
			// Op
			if !_fail(parser, _OpFail, errPos, failure, &pos) {
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

func _TypeSigAction(parser *_Parser, start int) (int, *TypeSig) {
	var labels [2]string
	use(labels)
	var label1 Ident
	var label0 *[]Parm
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
		// ps:TypeParms? n:(Ident/Op)
		// ps:TypeParms?
		{
			pos2 := pos
			// TypeParms?
			{
				pos4 := pos
				label0 = new([]Parm)
				// TypeParms
				if p, n := _TypeParmsAction(parser, pos); n == nil {
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
		// n:(Ident/Op)
		{
			pos7 := pos
			// (Ident/Op)
			// Ident/Op
			{
				pos11 := pos
				var node10 Ident
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail12
				} else {
					label1 = *n
					pos = p
				}
				goto ok8
			fail12:
				label1 = node10
				pos = pos11
				// Op
				if p, n := _OpAction(parser, pos); n == nil {
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
			start, end int, n Ident, ps *[]Parm) TypeSig {
			if ps == nil {
				return TypeSig{location: n.location, Name: n.Text}
			}
			return TypeSig{
				location: location{(*ps)[0].start, n.end},
				Name:     n.Text,
				Parms:    *ps,
			}
		}(
			start0, pos, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeParmsAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _TypeParms, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// n:TypeVar {…}/_ "(" p0:TypeParm ps:(_ "," p1:TypeParm {…})* (_ ",")? _ ")" {…}
	{
		pos3 := pos
		// action
		// n:TypeVar
		{
			pos5 := pos
			// TypeVar
			if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// _ "(" p0:TypeParm ps:(_ "," p1:TypeParm {…})* (_ ",")? _ ")"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail6
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail6
		}
		pos++
		// p0:TypeParm
		{
			pos8 := pos
			// TypeParm
			if !_accept(parser, _TypeParmAccepts, &pos, &perr) {
				goto fail6
			}
			labels[1] = parser.text[pos8:pos]
		}
		// ps:(_ "," p1:TypeParm {…})*
		{
			pos9 := pos
			// (_ "," p1:TypeParm {…})*
			for {
				pos11 := pos
				// (_ "," p1:TypeParm {…})
				// action
				// _ "," p1:TypeParm
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
				// p1:TypeParm
				{
					pos15 := pos
					// TypeParm
					if !_accept(parser, _TypeParmAccepts, &pos, &perr) {
						goto fail13
					}
					labels[2] = parser.text[pos15:pos]
				}
				continue
			fail13:
				pos = pos11
				break
			}
			labels[3] = parser.text[pos9:pos]
		}
		// (_ ",")?
		{
			pos17 := pos
			// (_ ",")
			// _ ","
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail18
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				perr = _max(perr, pos)
				goto fail18
			}
			pos++
			goto ok20
		fail18:
			pos = pos17
		ok20:
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail6
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
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
	return _memoize(parser, _TypeParms, start, pos, perr)
fail:
	return _memoize(parser, _TypeParms, start, -1, perr)
}

func _TypeParmsNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_TypeParms]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeParms}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "TypeParms"}
	// n:TypeVar {…}/_ "(" p0:TypeParm ps:(_ "," p1:TypeParm {…})* (_ ",")? _ ")" {…}
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// action
		// n:TypeVar
		{
			pos5 := pos
			// TypeVar
			if !_node(parser, _TypeVarNode, node, &pos) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// _ "(" p0:TypeParm ps:(_ "," p1:TypeParm {…})* (_ ",")? _ ")"
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail6
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			goto fail6
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// p0:TypeParm
		{
			pos8 := pos
			// TypeParm
			if !_node(parser, _TypeParmNode, node, &pos) {
				goto fail6
			}
			labels[1] = parser.text[pos8:pos]
		}
		// ps:(_ "," p1:TypeParm {…})*
		{
			pos9 := pos
			// (_ "," p1:TypeParm {…})*
			for {
				nkids10 := len(node.Kids)
				pos11 := pos
				// (_ "," p1:TypeParm {…})
				{
					nkids14 := len(node.Kids)
					pos015 := pos
					// action
					// _ "," p1:TypeParm
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail13
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail13
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					// p1:TypeParm
					{
						pos17 := pos
						// TypeParm
						if !_node(parser, _TypeParmNode, node, &pos) {
							goto fail13
						}
						labels[2] = parser.text[pos17:pos]
					}
					sub := _sub(parser, pos015, pos, node.Kids[nkids14:])
					node.Kids = append(node.Kids[:nkids14], sub)
				}
				continue
			fail13:
				node.Kids = node.Kids[:nkids10]
				pos = pos11
				break
			}
			labels[3] = parser.text[pos9:pos]
		}
		// (_ ",")?
		{
			nkids18 := len(node.Kids)
			pos19 := pos
			// (_ ",")
			{
				nkids21 := len(node.Kids)
				pos022 := pos
				// _ ","
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail20
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					goto fail20
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				sub := _sub(parser, pos022, pos, node.Kids[nkids21:])
				node.Kids = append(node.Kids[:nkids21], sub)
			}
			goto ok24
		fail20:
			node.Kids = node.Kids[:nkids18]
			pos = pos19
		ok24:
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail6
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			goto fail6
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail6:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _TypeParmsFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
	use(labels)
	pos, failure := _failMemo(parser, _TypeParms, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeParms",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeParms}
	// n:TypeVar {…}/_ "(" p0:TypeParm ps:(_ "," p1:TypeParm {…})* (_ ",")? _ ")" {…}
	{
		pos3 := pos
		// action
		// n:TypeVar
		{
			pos5 := pos
			// TypeVar
			if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// _ "(" p0:TypeParm ps:(_ "," p1:TypeParm {…})* (_ ",")? _ ")"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail6
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail6
		}
		pos++
		// p0:TypeParm
		{
			pos8 := pos
			// TypeParm
			if !_fail(parser, _TypeParmFail, errPos, failure, &pos) {
				goto fail6
			}
			labels[1] = parser.text[pos8:pos]
		}
		// ps:(_ "," p1:TypeParm {…})*
		{
			pos9 := pos
			// (_ "," p1:TypeParm {…})*
			for {
				pos11 := pos
				// (_ "," p1:TypeParm {…})
				// action
				// _ "," p1:TypeParm
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
				// p1:TypeParm
				{
					pos15 := pos
					// TypeParm
					if !_fail(parser, _TypeParmFail, errPos, failure, &pos) {
						goto fail13
					}
					labels[2] = parser.text[pos15:pos]
				}
				continue
			fail13:
				pos = pos11
				break
			}
			labels[3] = parser.text[pos9:pos]
		}
		// (_ ",")?
		{
			pos17 := pos
			// (_ ",")
			// _ ","
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail18
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\",\"",
					})
				}
				goto fail18
			}
			pos++
			goto ok20
		fail18:
			pos = pos17
		ok20:
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail6
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
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

func _TypeParmsAction(parser *_Parser, start int) (int, *[]Parm) {
	var labels [4]string
	use(labels)
	var label2 Parm
	var label3 []Parm
	var label0 Ident
	var label1 Parm
	dp := parser.deltaPos[start][_TypeParms]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeParms}
	n := parser.act[key]
	if n != nil {
		n := n.([]Parm)
		return start + int(dp-1), &n
	}
	var node []Parm
	pos := start
	// n:TypeVar {…}/_ "(" p0:TypeParm ps:(_ "," p1:TypeParm {…})* (_ ",")? _ ")" {…}
	{
		pos3 := pos
		var node2 []Parm
		// action
		{
			start5 := pos
			// n:TypeVar
			{
				pos6 := pos
				// TypeVar
				if p, n := _TypeVarAction(parser, pos); n == nil {
					goto fail4
				} else {
					label0 = *n
					pos = p
				}
				labels[0] = parser.text[pos6:pos]
			}
			node = func(
				start, end int, n Ident) []Parm {
				return []Parm{{location: n.location, Name: n.Text}}
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
			// _ "(" p0:TypeParm ps:(_ "," p1:TypeParm {…})* (_ ",")? _ ")"
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail7
			} else {
				pos = p
			}
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				goto fail7
			}
			pos++
			// p0:TypeParm
			{
				pos10 := pos
				// TypeParm
				if p, n := _TypeParmAction(parser, pos); n == nil {
					goto fail7
				} else {
					label1 = *n
					pos = p
				}
				labels[1] = parser.text[pos10:pos]
			}
			// ps:(_ "," p1:TypeParm {…})*
			{
				pos11 := pos
				// (_ "," p1:TypeParm {…})*
				for {
					pos13 := pos
					var node14 Parm
					// (_ "," p1:TypeParm {…})
					// action
					{
						start16 := pos
						// _ "," p1:TypeParm
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
						// p1:TypeParm
						{
							pos18 := pos
							// TypeParm
							if p, n := _TypeParmAction(parser, pos); n == nil {
								goto fail15
							} else {
								label2 = *n
								pos = p
							}
							labels[2] = parser.text[pos18:pos]
						}
						node14 = func(
							start, end int, n Ident, p0 Parm, p1 Parm) Parm {
							return Parm(p1)
						}(
							start16, pos, label0, label1, label2)
					}
					label3 = append(label3, node14)
					continue
				fail15:
					pos = pos13
					break
				}
				labels[3] = parser.text[pos11:pos]
			}
			// (_ ",")?
			{
				pos20 := pos
				// (_ ",")
				// _ ","
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail21
				} else {
					pos = p
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					goto fail21
				}
				pos++
				goto ok23
			fail21:
				pos = pos20
			ok23:
			}
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail7
			} else {
				pos = p
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				goto fail7
			}
			pos++
			node = func(
				start, end int, n Ident, p0 Parm, p1 Parm, ps []Parm) []Parm {
				return []Parm(append([]Parm{p0}, ps...))
			}(
				start8, pos, label0, label1, label2, label3)
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

func _TypeParmAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _TypeParm, start); ok {
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
	return _memoize(parser, _TypeParm, start, pos, perr)
fail:
	return _memoize(parser, _TypeParm, start, -1, perr)
}

func _TypeParmNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_TypeParm]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeParm}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "TypeParm"}
	// action
	// n:TypeVar t1:TypeName?
	// n:TypeVar
	{
		pos1 := pos
		// TypeVar
		if !_node(parser, _TypeVarNode, node, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// t1:TypeName?
	{
		pos2 := pos
		// TypeName?
		{
			nkids3 := len(node.Kids)
			pos4 := pos
			// TypeName
			if !_node(parser, _TypeNameNode, node, &pos) {
				goto fail5
			}
			goto ok6
		fail5:
			node.Kids = node.Kids[:nkids3]
			pos = pos4
		ok6:
		}
		labels[1] = parser.text[pos2:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _TypeParmFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
	use(labels)
	pos, failure := _failMemo(parser, _TypeParm, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeParm",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeParm}
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

func _TypeParmAction(parser *_Parser, start int) (int, *Parm) {
	var labels [2]string
	use(labels)
	var label0 Ident
	var label1 *TypeName
	dp := parser.deltaPos[start][_TypeParm]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeParm}
	n := parser.act[key]
	if n != nil {
		n := n.(Parm)
		return start + int(dp-1), &n
	}
	var node Parm
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
			start, end int, n Ident, t1 *TypeName) Parm {
			e := n.end
			if t1 != nil {
				e = t1.end
			}
			return Parm{
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
	var labels [16]string
	use(labels)
	if dp, de, ok := _memo(parser, _TypeName, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// tv0:TypeVar? ids0:Ident* op:TypeOp {…}/tv1:TypeVar? ids1:Ident+ {…}/tv2:TypeVar {…}/_ blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…}) {…}/_ tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…}) {…}/_ "(" n2:TypeName _ ")" {…}
	{
		pos3 := pos
		// action
		// tv0:TypeVar? ids0:Ident* op:TypeOp
		// tv0:TypeVar?
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
		// ids0:Ident*
		{
			pos11 := pos
			// Ident*
			for {
				pos13 := pos
				// Ident
				if !_accept(parser, _IdentAccepts, &pos, &perr) {
					goto fail15
				}
				continue
			fail15:
				pos = pos13
				break
			}
			labels[1] = parser.text[pos11:pos]
		}
		// op:TypeOp
		{
			pos16 := pos
			// TypeOp
			if !_accept(parser, _TypeOpAccepts, &pos, &perr) {
				goto fail4
			}
			labels[2] = parser.text[pos16:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// tv1:TypeVar? ids1:Ident+
		// tv1:TypeVar?
		{
			pos19 := pos
			// TypeVar?
			{
				pos21 := pos
				// TypeVar
				if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
					goto fail22
				}
				goto ok23
			fail22:
				pos = pos21
			ok23:
			}
			labels[3] = parser.text[pos19:pos]
		}
		// ids1:Ident+
		{
			pos24 := pos
			// Ident+
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail17
			}
			for {
				pos26 := pos
				// Ident
				if !_accept(parser, _IdentAccepts, &pos, &perr) {
					goto fail28
				}
				continue
			fail28:
				pos = pos26
				break
			}
			labels[4] = parser.text[pos24:pos]
		}
		goto ok0
	fail17:
		pos = pos3
		// action
		// tv2:TypeVar
		{
			pos30 := pos
			// TypeVar
			if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
				goto fail29
			}
			labels[5] = parser.text[pos30:pos]
		}
		goto ok0
	fail29:
		pos = pos3
		// action
		// _ blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail31
		}
		// blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
		{
			pos33 := pos
			// ("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
			// action
			// "[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]"
			// "["
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
				perr = _max(perr, pos)
				goto fail31
			}
			pos++
			// ps:TypeName*
			{
				pos35 := pos
				// TypeName*
				for {
					pos37 := pos
					// TypeName
					if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
						goto fail39
					}
					continue
				fail39:
					pos = pos37
					break
				}
				labels[6] = parser.text[pos35:pos]
			}
			// r:(_ "|" r1:TypeName {…})?
			{
				pos40 := pos
				// (_ "|" r1:TypeName {…})?
				{
					pos42 := pos
					// (_ "|" r1:TypeName {…})
					// action
					// _ "|" r1:TypeName
					// _
					if !_accept(parser, __Accepts, &pos, &perr) {
						goto fail43
					}
					// "|"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
						perr = _max(perr, pos)
						goto fail43
					}
					pos++
					// r1:TypeName
					{
						pos45 := pos
						// TypeName
						if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
							goto fail43
						}
						labels[7] = parser.text[pos45:pos]
					}
					goto ok46
				fail43:
					pos = pos42
				ok46:
				}
				labels[8] = parser.text[pos40:pos]
			}
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail31
			}
			// "]"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
				perr = _max(perr, pos)
				goto fail31
			}
			pos++
			labels[9] = parser.text[pos33:pos]
		}
		goto ok0
	fail31:
		pos = pos3
		// action
		// _ tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail47
		}
		// tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
		{
			pos49 := pos
			// ("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
			// action
			// "(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				perr = _max(perr, pos)
				goto fail47
			}
			pos++
			// n0:TypeName
			{
				pos51 := pos
				// TypeName
				if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
					goto fail47
				}
				labels[10] = parser.text[pos51:pos]
			}
			// ns:(_ "," n1:TypeName {…})*
			{
				pos52 := pos
				// (_ "," n1:TypeName {…})*
				for {
					pos54 := pos
					// (_ "," n1:TypeName {…})
					// action
					// _ "," n1:TypeName
					// _
					if !_accept(parser, __Accepts, &pos, &perr) {
						goto fail56
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						perr = _max(perr, pos)
						goto fail56
					}
					pos++
					// n1:TypeName
					{
						pos58 := pos
						// TypeName
						if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
							goto fail56
						}
						labels[11] = parser.text[pos58:pos]
					}
					continue
				fail56:
					pos = pos54
					break
				}
				labels[12] = parser.text[pos52:pos]
			}
			// (_ ",")?
			{
				pos60 := pos
				// (_ ",")
				// _ ","
				// _
				if !_accept(parser, __Accepts, &pos, &perr) {
					goto fail61
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					perr = _max(perr, pos)
					goto fail61
				}
				pos++
				goto ok63
			fail61:
				pos = pos60
			ok63:
			}
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail47
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				perr = _max(perr, pos)
				goto fail47
			}
			pos++
			// ids2:Ident+
			{
				pos64 := pos
				// Ident+
				// Ident
				if !_accept(parser, _IdentAccepts, &pos, &perr) {
					goto fail47
				}
				for {
					pos66 := pos
					// Ident
					if !_accept(parser, _IdentAccepts, &pos, &perr) {
						goto fail68
					}
					continue
				fail68:
					pos = pos66
					break
				}
				labels[13] = parser.text[pos64:pos]
			}
			labels[14] = parser.text[pos49:pos]
		}
		goto ok0
	fail47:
		pos = pos3
		// action
		// _ "(" n2:TypeName _ ")"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail69
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail69
		}
		pos++
		// n2:TypeName
		{
			pos71 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail69
			}
			labels[15] = parser.text[pos71:pos]
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail69
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			perr = _max(perr, pos)
			goto fail69
		}
		pos++
		goto ok0
	fail69:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _TypeName, start, pos, perr)
fail:
	return _memoize(parser, _TypeName, start, -1, perr)
}

func _TypeNameNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [16]string
	use(labels)
	dp := parser.deltaPos[start][_TypeName]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeName}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "TypeName"}
	// tv0:TypeVar? ids0:Ident* op:TypeOp {…}/tv1:TypeVar? ids1:Ident+ {…}/tv2:TypeVar {…}/_ blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…}) {…}/_ tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…}) {…}/_ "(" n2:TypeName _ ")" {…}
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// action
		// tv0:TypeVar? ids0:Ident* op:TypeOp
		// tv0:TypeVar?
		{
			pos6 := pos
			// TypeVar?
			{
				nkids7 := len(node.Kids)
				pos8 := pos
				// TypeVar
				if !_node(parser, _TypeVarNode, node, &pos) {
					goto fail9
				}
				goto ok10
			fail9:
				node.Kids = node.Kids[:nkids7]
				pos = pos8
			ok10:
			}
			labels[0] = parser.text[pos6:pos]
		}
		// ids0:Ident*
		{
			pos11 := pos
			// Ident*
			for {
				nkids12 := len(node.Kids)
				pos13 := pos
				// Ident
				if !_node(parser, _IdentNode, node, &pos) {
					goto fail15
				}
				continue
			fail15:
				node.Kids = node.Kids[:nkids12]
				pos = pos13
				break
			}
			labels[1] = parser.text[pos11:pos]
		}
		// op:TypeOp
		{
			pos16 := pos
			// TypeOp
			if !_node(parser, _TypeOpNode, node, &pos) {
				goto fail4
			}
			labels[2] = parser.text[pos16:pos]
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// tv1:TypeVar? ids1:Ident+
		// tv1:TypeVar?
		{
			pos19 := pos
			// TypeVar?
			{
				nkids20 := len(node.Kids)
				pos21 := pos
				// TypeVar
				if !_node(parser, _TypeVarNode, node, &pos) {
					goto fail22
				}
				goto ok23
			fail22:
				node.Kids = node.Kids[:nkids20]
				pos = pos21
			ok23:
			}
			labels[3] = parser.text[pos19:pos]
		}
		// ids1:Ident+
		{
			pos24 := pos
			// Ident+
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail17
			}
			for {
				nkids25 := len(node.Kids)
				pos26 := pos
				// Ident
				if !_node(parser, _IdentNode, node, &pos) {
					goto fail28
				}
				continue
			fail28:
				node.Kids = node.Kids[:nkids25]
				pos = pos26
				break
			}
			labels[4] = parser.text[pos24:pos]
		}
		goto ok0
	fail17:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// tv2:TypeVar
		{
			pos30 := pos
			// TypeVar
			if !_node(parser, _TypeVarNode, node, &pos) {
				goto fail29
			}
			labels[5] = parser.text[pos30:pos]
		}
		goto ok0
	fail29:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// _ blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail31
		}
		// blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
		{
			pos33 := pos
			// ("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
			{
				nkids34 := len(node.Kids)
				pos035 := pos
				// action
				// "[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]"
				// "["
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
					goto fail31
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// ps:TypeName*
				{
					pos37 := pos
					// TypeName*
					for {
						nkids38 := len(node.Kids)
						pos39 := pos
						// TypeName
						if !_node(parser, _TypeNameNode, node, &pos) {
							goto fail41
						}
						continue
					fail41:
						node.Kids = node.Kids[:nkids38]
						pos = pos39
						break
					}
					labels[6] = parser.text[pos37:pos]
				}
				// r:(_ "|" r1:TypeName {…})?
				{
					pos42 := pos
					// (_ "|" r1:TypeName {…})?
					{
						nkids43 := len(node.Kids)
						pos44 := pos
						// (_ "|" r1:TypeName {…})
						{
							nkids46 := len(node.Kids)
							pos047 := pos
							// action
							// _ "|" r1:TypeName
							// _
							if !_node(parser, __Node, node, &pos) {
								goto fail45
							}
							// "|"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
								goto fail45
							}
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
							pos++
							// r1:TypeName
							{
								pos49 := pos
								// TypeName
								if !_node(parser, _TypeNameNode, node, &pos) {
									goto fail45
								}
								labels[7] = parser.text[pos49:pos]
							}
							sub := _sub(parser, pos047, pos, node.Kids[nkids46:])
							node.Kids = append(node.Kids[:nkids46], sub)
						}
						goto ok50
					fail45:
						node.Kids = node.Kids[:nkids43]
						pos = pos44
					ok50:
					}
					labels[8] = parser.text[pos42:pos]
				}
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail31
				}
				// "]"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
					goto fail31
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				sub := _sub(parser, pos035, pos, node.Kids[nkids34:])
				node.Kids = append(node.Kids[:nkids34], sub)
			}
			labels[9] = parser.text[pos33:pos]
		}
		goto ok0
	fail31:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// _ tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail51
		}
		// tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
		{
			pos53 := pos
			// ("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
			{
				nkids54 := len(node.Kids)
				pos055 := pos
				// action
				// "(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+
				// "("
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
					goto fail51
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// n0:TypeName
				{
					pos57 := pos
					// TypeName
					if !_node(parser, _TypeNameNode, node, &pos) {
						goto fail51
					}
					labels[10] = parser.text[pos57:pos]
				}
				// ns:(_ "," n1:TypeName {…})*
				{
					pos58 := pos
					// (_ "," n1:TypeName {…})*
					for {
						nkids59 := len(node.Kids)
						pos60 := pos
						// (_ "," n1:TypeName {…})
						{
							nkids63 := len(node.Kids)
							pos064 := pos
							// action
							// _ "," n1:TypeName
							// _
							if !_node(parser, __Node, node, &pos) {
								goto fail62
							}
							// ","
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
								goto fail62
							}
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
							pos++
							// n1:TypeName
							{
								pos66 := pos
								// TypeName
								if !_node(parser, _TypeNameNode, node, &pos) {
									goto fail62
								}
								labels[11] = parser.text[pos66:pos]
							}
							sub := _sub(parser, pos064, pos, node.Kids[nkids63:])
							node.Kids = append(node.Kids[:nkids63], sub)
						}
						continue
					fail62:
						node.Kids = node.Kids[:nkids59]
						pos = pos60
						break
					}
					labels[12] = parser.text[pos58:pos]
				}
				// (_ ",")?
				{
					nkids67 := len(node.Kids)
					pos68 := pos
					// (_ ",")
					{
						nkids70 := len(node.Kids)
						pos071 := pos
						// _ ","
						// _
						if !_node(parser, __Node, node, &pos) {
							goto fail69
						}
						// ","
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
							goto fail69
						}
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
						pos++
						sub := _sub(parser, pos071, pos, node.Kids[nkids70:])
						node.Kids = append(node.Kids[:nkids70], sub)
					}
					goto ok73
				fail69:
					node.Kids = node.Kids[:nkids67]
					pos = pos68
				ok73:
				}
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail51
				}
				// ")"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
					goto fail51
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// ids2:Ident+
				{
					pos74 := pos
					// Ident+
					// Ident
					if !_node(parser, _IdentNode, node, &pos) {
						goto fail51
					}
					for {
						nkids75 := len(node.Kids)
						pos76 := pos
						// Ident
						if !_node(parser, _IdentNode, node, &pos) {
							goto fail78
						}
						continue
					fail78:
						node.Kids = node.Kids[:nkids75]
						pos = pos76
						break
					}
					labels[13] = parser.text[pos74:pos]
				}
				sub := _sub(parser, pos055, pos, node.Kids[nkids54:])
				node.Kids = append(node.Kids[:nkids54], sub)
			}
			labels[14] = parser.text[pos53:pos]
		}
		goto ok0
	fail51:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// _ "(" n2:TypeName _ ")"
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail79
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			goto fail79
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// n2:TypeName
		{
			pos81 := pos
			// TypeName
			if !_node(parser, _TypeNameNode, node, &pos) {
				goto fail79
			}
			labels[15] = parser.text[pos81:pos]
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail79
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			goto fail79
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail79:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _TypeNameFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [16]string
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
	// tv0:TypeVar? ids0:Ident* op:TypeOp {…}/tv1:TypeVar? ids1:Ident+ {…}/tv2:TypeVar {…}/_ blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…}) {…}/_ tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…}) {…}/_ "(" n2:TypeName _ ")" {…}
	{
		pos3 := pos
		// action
		// tv0:TypeVar? ids0:Ident* op:TypeOp
		// tv0:TypeVar?
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
		// ids0:Ident*
		{
			pos11 := pos
			// Ident*
			for {
				pos13 := pos
				// Ident
				if !_fail(parser, _IdentFail, errPos, failure, &pos) {
					goto fail15
				}
				continue
			fail15:
				pos = pos13
				break
			}
			labels[1] = parser.text[pos11:pos]
		}
		// op:TypeOp
		{
			pos16 := pos
			// TypeOp
			if !_fail(parser, _TypeOpFail, errPos, failure, &pos) {
				goto fail4
			}
			labels[2] = parser.text[pos16:pos]
		}
		goto ok0
	fail4:
		pos = pos3
		// action
		// tv1:TypeVar? ids1:Ident+
		// tv1:TypeVar?
		{
			pos19 := pos
			// TypeVar?
			{
				pos21 := pos
				// TypeVar
				if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
					goto fail22
				}
				goto ok23
			fail22:
				pos = pos21
			ok23:
			}
			labels[3] = parser.text[pos19:pos]
		}
		// ids1:Ident+
		{
			pos24 := pos
			// Ident+
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail17
			}
			for {
				pos26 := pos
				// Ident
				if !_fail(parser, _IdentFail, errPos, failure, &pos) {
					goto fail28
				}
				continue
			fail28:
				pos = pos26
				break
			}
			labels[4] = parser.text[pos24:pos]
		}
		goto ok0
	fail17:
		pos = pos3
		// action
		// tv2:TypeVar
		{
			pos30 := pos
			// TypeVar
			if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
				goto fail29
			}
			labels[5] = parser.text[pos30:pos]
		}
		goto ok0
	fail29:
		pos = pos3
		// action
		// _ blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail31
		}
		// blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
		{
			pos33 := pos
			// ("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
			// action
			// "[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]"
			// "["
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"[\"",
					})
				}
				goto fail31
			}
			pos++
			// ps:TypeName*
			{
				pos35 := pos
				// TypeName*
				for {
					pos37 := pos
					// TypeName
					if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
						goto fail39
					}
					continue
				fail39:
					pos = pos37
					break
				}
				labels[6] = parser.text[pos35:pos]
			}
			// r:(_ "|" r1:TypeName {…})?
			{
				pos40 := pos
				// (_ "|" r1:TypeName {…})?
				{
					pos42 := pos
					// (_ "|" r1:TypeName {…})
					// action
					// _ "|" r1:TypeName
					// _
					if !_fail(parser, __Fail, errPos, failure, &pos) {
						goto fail43
					}
					// "|"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "\"|\"",
							})
						}
						goto fail43
					}
					pos++
					// r1:TypeName
					{
						pos45 := pos
						// TypeName
						if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
							goto fail43
						}
						labels[7] = parser.text[pos45:pos]
					}
					goto ok46
				fail43:
					pos = pos42
				ok46:
				}
				labels[8] = parser.text[pos40:pos]
			}
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail31
			}
			// "]"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"]\"",
					})
				}
				goto fail31
			}
			pos++
			labels[9] = parser.text[pos33:pos]
		}
		goto ok0
	fail31:
		pos = pos3
		// action
		// _ tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail47
		}
		// tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
		{
			pos49 := pos
			// ("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
			// action
			// "(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"(\"",
					})
				}
				goto fail47
			}
			pos++
			// n0:TypeName
			{
				pos51 := pos
				// TypeName
				if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
					goto fail47
				}
				labels[10] = parser.text[pos51:pos]
			}
			// ns:(_ "," n1:TypeName {…})*
			{
				pos52 := pos
				// (_ "," n1:TypeName {…})*
				for {
					pos54 := pos
					// (_ "," n1:TypeName {…})
					// action
					// _ "," n1:TypeName
					// _
					if !_fail(parser, __Fail, errPos, failure, &pos) {
						goto fail56
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						if pos >= errPos {
							failure.Kids = append(failure.Kids, &peg.Fail{
								Pos:  int(pos),
								Want: "\",\"",
							})
						}
						goto fail56
					}
					pos++
					// n1:TypeName
					{
						pos58 := pos
						// TypeName
						if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
							goto fail56
						}
						labels[11] = parser.text[pos58:pos]
					}
					continue
				fail56:
					pos = pos54
					break
				}
				labels[12] = parser.text[pos52:pos]
			}
			// (_ ",")?
			{
				pos60 := pos
				// (_ ",")
				// _ ","
				// _
				if !_fail(parser, __Fail, errPos, failure, &pos) {
					goto fail61
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\",\"",
						})
					}
					goto fail61
				}
				pos++
				goto ok63
			fail61:
				pos = pos60
			ok63:
			}
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail47
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\")\"",
					})
				}
				goto fail47
			}
			pos++
			// ids2:Ident+
			{
				pos64 := pos
				// Ident+
				// Ident
				if !_fail(parser, _IdentFail, errPos, failure, &pos) {
					goto fail47
				}
				for {
					pos66 := pos
					// Ident
					if !_fail(parser, _IdentFail, errPos, failure, &pos) {
						goto fail68
					}
					continue
				fail68:
					pos = pos66
					break
				}
				labels[13] = parser.text[pos64:pos]
			}
			labels[14] = parser.text[pos49:pos]
		}
		goto ok0
	fail47:
		pos = pos3
		// action
		// _ "(" n2:TypeName _ ")"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail69
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail69
		}
		pos++
		// n2:TypeName
		{
			pos71 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail69
			}
			labels[15] = parser.text[pos71:pos]
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail69
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
				})
			}
			goto fail69
		}
		pos++
		goto ok0
	fail69:
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
	var labels [16]string
	use(labels)
	var label7 TypeName
	var label11 TypeName
	var label1 []Ident
	var label6 []TypeName
	var label5 Ident
	var label8 *TypeName
	var label10 TypeName
	var label2 Ident
	var label4 []Ident
	var label15 TypeName
	var label3 *Ident
	var label14 TypeName
	var label12 []TypeName
	var label13 []Ident
	var label0 *Ident
	var label9 TypeName
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
	// tv0:TypeVar? ids0:Ident* op:TypeOp {…}/tv1:TypeVar? ids1:Ident+ {…}/tv2:TypeVar {…}/_ blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…}) {…}/_ tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…}) {…}/_ "(" n2:TypeName _ ")" {…}
	{
		pos3 := pos
		var node2 TypeName
		// action
		{
			start5 := pos
			// tv0:TypeVar? ids0:Ident* op:TypeOp
			// tv0:TypeVar?
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
			// ids0:Ident*
			{
				pos12 := pos
				// Ident*
				for {
					pos14 := pos
					var node15 Ident
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
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
			// op:TypeOp
			{
				pos17 := pos
				// TypeOp
				if p, n := _TypeOpAction(parser, pos); n == nil {
					goto fail4
				} else {
					label2 = *n
					pos = p
				}
				labels[2] = parser.text[pos17:pos]
			}
			node = func(
				start, end int, ids0 []Ident, op Ident, tv0 *Ident) TypeName {
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
							Name:     id.Text,
							Args:     a,
						}}
					}
				}
				return TypeName{
					location: location{s, op.end},
					Name:     op.Text,
					Args:     a,
				}
			}(
				start5, pos, label1, label2, label0)
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// action
		{
			start19 := pos
			// tv1:TypeVar? ids1:Ident+
			// tv1:TypeVar?
			{
				pos21 := pos
				// TypeVar?
				{
					pos23 := pos
					label3 = new(Ident)
					// TypeVar
					if p, n := _TypeVarAction(parser, pos); n == nil {
						goto fail24
					} else {
						*label3 = *n
						pos = p
					}
					goto ok25
				fail24:
					label3 = nil
					pos = pos23
				ok25:
				}
				labels[3] = parser.text[pos21:pos]
			}
			// ids1:Ident+
			{
				pos26 := pos
				// Ident+
				{
					var node29 Ident
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail18
					} else {
						node29 = *n
						pos = p
					}
					label4 = append(label4, node29)
				}
				for {
					pos28 := pos
					var node29 Ident
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail30
					} else {
						node29 = *n
						pos = p
					}
					label4 = append(label4, node29)
					continue
				fail30:
					pos = pos28
					break
				}
				labels[4] = parser.text[pos26:pos]
			}
			node = func(
				start, end int, ids0 []Ident, ids1 []Ident, op Ident, tv0 *Ident, tv1 *Ident) TypeName {
				s := ids1[0].start
				var a []TypeName
				if tv1 != nil {
					s = tv1.start
					a = []TypeName{{location: tv1.location, Name: tv1.Text}}
				}
				for _, id := range ids1[:len(ids1)-1] {
					a = []TypeName{{
						location: location{s, id.end},
						Name:     id.Text,
						Args:     a,
					}}
				}
				n := ids1[len(ids1)-1]
				return TypeName{
					location: location{s, n.end},
					Name:     n.Text,
					Args:     a,
				}
			}(
				start19, pos, label1, label4, label2, label0, label3)
		}
		goto ok0
	fail18:
		node = node2
		pos = pos3
		// action
		{
			start32 := pos
			// tv2:TypeVar
			{
				pos33 := pos
				// TypeVar
				if p, n := _TypeVarAction(parser, pos); n == nil {
					goto fail31
				} else {
					label5 = *n
					pos = p
				}
				labels[5] = parser.text[pos33:pos]
			}
			node = func(
				start, end int, ids0 []Ident, ids1 []Ident, op Ident, tv0 *Ident, tv1 *Ident, tv2 Ident) TypeName {
				return TypeName{location: tv2.location, Name: tv2.Text}
			}(
				start32, pos, label1, label4, label2, label0, label3, label5)
		}
		goto ok0
	fail31:
		node = node2
		pos = pos3
		// action
		{
			start35 := pos
			// _ blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail34
			} else {
				pos = p
			}
			// blk:("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
			{
				pos37 := pos
				// ("[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]" {…})
				// action
				{
					start38 := pos
					// "[" ps:TypeName* r:(_ "|" r1:TypeName {…})? _ "]"
					// "["
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
						goto fail34
					}
					pos++
					// ps:TypeName*
					{
						pos40 := pos
						// TypeName*
						for {
							pos42 := pos
							var node43 TypeName
							// TypeName
							if p, n := _TypeNameAction(parser, pos); n == nil {
								goto fail44
							} else {
								node43 = *n
								pos = p
							}
							label6 = append(label6, node43)
							continue
						fail44:
							pos = pos42
							break
						}
						labels[6] = parser.text[pos40:pos]
					}
					// r:(_ "|" r1:TypeName {…})?
					{
						pos45 := pos
						// (_ "|" r1:TypeName {…})?
						{
							pos47 := pos
							label8 = new(TypeName)
							// (_ "|" r1:TypeName {…})
							// action
							{
								start49 := pos
								// _ "|" r1:TypeName
								// _
								if p, n := __Action(parser, pos); n == nil {
									goto fail48
								} else {
									pos = p
								}
								// "|"
								if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
									goto fail48
								}
								pos++
								// r1:TypeName
								{
									pos51 := pos
									// TypeName
									if p, n := _TypeNameAction(parser, pos); n == nil {
										goto fail48
									} else {
										label7 = *n
										pos = p
									}
									labels[7] = parser.text[pos51:pos]
								}
								*label8 = func(
									start, end int, ids0 []Ident, ids1 []Ident, op Ident, ps []TypeName, r1 TypeName, tv0 *Ident, tv1 *Ident, tv2 Ident) TypeName {
									return TypeName(r1)
								}(
									start49, pos, label1, label4, label2, label6, label7, label0, label3, label5)
							}
							goto ok52
						fail48:
							label8 = nil
							pos = pos47
						ok52:
						}
						labels[8] = parser.text[pos45:pos]
					}
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail34
					} else {
						pos = p
					}
					// "]"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
						goto fail34
					}
					pos++
					label9 = func(
						start, end int, ids0 []Ident, ids1 []Ident, op Ident, ps []TypeName, r *TypeName, r1 TypeName, tv0 *Ident, tv1 *Ident, tv2 Ident) TypeName {
						name := "[]"
						if r != nil {
							name = "[|]"
							ps = append(ps, *r)
						}
						return TypeName{
							location: loc(parser, start, end),
							Name:     name,
							Args:     ps,
						}
					}(
						start38, pos, label1, label4, label2, label6, label8, label7, label0, label3, label5)
				}
				labels[9] = parser.text[pos37:pos]
			}
			node = func(
				start, end int, blk TypeName, ids0 []Ident, ids1 []Ident, op Ident, ps []TypeName, r *TypeName, r1 TypeName, tv0 *Ident, tv1 *Ident, tv2 Ident) TypeName {
				return TypeName(blk)
			}(
				start35, pos, label9, label1, label4, label2, label6, label8, label7, label0, label3, label5)
		}
		goto ok0
	fail34:
		node = node2
		pos = pos3
		// action
		{
			start54 := pos
			// _ tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail53
			} else {
				pos = p
			}
			// tn0:("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
			{
				pos56 := pos
				// ("(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+ {…})
				// action
				{
					start57 := pos
					// "(" n0:TypeName ns:(_ "," n1:TypeName {…})* (_ ",")? _ ")" ids2:Ident+
					// "("
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
						goto fail53
					}
					pos++
					// n0:TypeName
					{
						pos59 := pos
						// TypeName
						if p, n := _TypeNameAction(parser, pos); n == nil {
							goto fail53
						} else {
							label10 = *n
							pos = p
						}
						labels[10] = parser.text[pos59:pos]
					}
					// ns:(_ "," n1:TypeName {…})*
					{
						pos60 := pos
						// (_ "," n1:TypeName {…})*
						for {
							pos62 := pos
							var node63 TypeName
							// (_ "," n1:TypeName {…})
							// action
							{
								start65 := pos
								// _ "," n1:TypeName
								// _
								if p, n := __Action(parser, pos); n == nil {
									goto fail64
								} else {
									pos = p
								}
								// ","
								if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
									goto fail64
								}
								pos++
								// n1:TypeName
								{
									pos67 := pos
									// TypeName
									if p, n := _TypeNameAction(parser, pos); n == nil {
										goto fail64
									} else {
										label11 = *n
										pos = p
									}
									labels[11] = parser.text[pos67:pos]
								}
								node63 = func(
									start, end int, blk TypeName, ids0 []Ident, ids1 []Ident, n0 TypeName, n1 TypeName, op Ident, ps []TypeName, r *TypeName, r1 TypeName, tv0 *Ident, tv1 *Ident, tv2 Ident) TypeName {
									return TypeName(n1)
								}(
									start65, pos, label9, label1, label4, label10, label11, label2, label6, label8, label7, label0, label3, label5)
							}
							label12 = append(label12, node63)
							continue
						fail64:
							pos = pos62
							break
						}
						labels[12] = parser.text[pos60:pos]
					}
					// (_ ",")?
					{
						pos69 := pos
						// (_ ",")
						// _ ","
						// _
						if p, n := __Action(parser, pos); n == nil {
							goto fail70
						} else {
							pos = p
						}
						// ","
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
							goto fail70
						}
						pos++
						goto ok72
					fail70:
						pos = pos69
					ok72:
					}
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail53
					} else {
						pos = p
					}
					// ")"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
						goto fail53
					}
					pos++
					// ids2:Ident+
					{
						pos73 := pos
						// Ident+
						{
							var node76 Ident
							// Ident
							if p, n := _IdentAction(parser, pos); n == nil {
								goto fail53
							} else {
								node76 = *n
								pos = p
							}
							label13 = append(label13, node76)
						}
						for {
							pos75 := pos
							var node76 Ident
							// Ident
							if p, n := _IdentAction(parser, pos); n == nil {
								goto fail77
							} else {
								node76 = *n
								pos = p
							}
							label13 = append(label13, node76)
							continue
						fail77:
							pos = pos75
							break
						}
						labels[13] = parser.text[pos73:pos]
					}
					label14 = func(
						start, end int, blk TypeName, ids0 []Ident, ids1 []Ident, ids2 []Ident, n0 TypeName, n1 TypeName, ns []TypeName, op Ident, ps []TypeName, r *TypeName, r1 TypeName, tv0 *Ident, tv1 *Ident, tv2 Ident) TypeName {
						s := loc1(parser, start)
						as := append([]TypeName{n0}, ns...)
						for _, id := range ids2[:len(ids2)-1] {
							as = []TypeName{{
								location: location{s, id.end},
								Name:     id.Text,
								Args:     as,
							}}
						}
						return TypeName{
							location: loc(parser, start, end),
							Name:     ids2[len(ids2)-1].Text,
							Args:     as,
						}
					}(
						start57, pos, label9, label1, label4, label13, label10, label11, label12, label2, label6, label8, label7, label0, label3, label5)
				}
				labels[14] = parser.text[pos56:pos]
			}
			node = func(
				start, end int, blk TypeName, ids0 []Ident, ids1 []Ident, ids2 []Ident, n0 TypeName, n1 TypeName, ns []TypeName, op Ident, ps []TypeName, r *TypeName, r1 TypeName, tn0 TypeName, tv0 *Ident, tv1 *Ident, tv2 Ident) TypeName {
				return TypeName(tn0)
			}(
				start54, pos, label9, label1, label4, label13, label10, label11, label12, label2, label6, label8, label7, label14, label0, label3, label5)
		}
		goto ok0
	fail53:
		node = node2
		pos = pos3
		// action
		{
			start79 := pos
			// _ "(" n2:TypeName _ ")"
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail78
			} else {
				pos = p
			}
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				goto fail78
			}
			pos++
			// n2:TypeName
			{
				pos81 := pos
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail78
				} else {
					label15 = *n
					pos = p
				}
				labels[15] = parser.text[pos81:pos]
			}
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail78
			} else {
				pos = p
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				goto fail78
			}
			pos++
			node = func(
				start, end int, blk TypeName, ids0 []Ident, ids1 []Ident, ids2 []Ident, n0 TypeName, n1 TypeName, n2 TypeName, ns []TypeName, op Ident, ps []TypeName, r *TypeName, r1 TypeName, tn0 TypeName, tv0 *Ident, tv1 *Ident, tv2 Ident) TypeName {
				return TypeName(n2)
			}(
				start79, pos, label9, label1, label4, label13, label10, label11, label15, label12, label2, label6, label8, label7, label14, label0, label3, label5)
		}
		goto ok0
	fail78:
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

func _TypeAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Type, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Struct/Enum/Virt
	{
		pos3 := pos
		// Struct
		if !_accept(parser, _StructAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Enum
		if !_accept(parser, _EnumAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Virt
		if !_accept(parser, _VirtAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Type, start, pos, perr)
fail:
	return _memoize(parser, _Type, start, -1, perr)
}

func _TypeNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_Type]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Type}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Type"}
	// Struct/Enum/Virt
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// Struct
		if !_node(parser, _StructNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Enum
		if !_node(parser, _EnumNode, node, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Virt
		if !_node(parser, _VirtNode, node, &pos) {
			goto fail6
		}
		goto ok0
	fail6:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _TypeFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Type, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Type",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Type}
	// Struct/Enum/Virt
	{
		pos3 := pos
		// Struct
		if !_fail(parser, _StructFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Enum
		if !_fail(parser, _EnumFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Virt
		if !_fail(parser, _VirtFail, errPos, failure, &pos) {
			goto fail6
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

func _TypeAction(parser *_Parser, start int) (int, *Def) {
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
	// Struct/Enum/Virt
	{
		pos3 := pos
		var node2 Def
		// Struct
		if p, n := _StructAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Enum
		if p, n := _EnumAction(parser, pos); n == nil {
			goto fail5
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail5:
		node = node2
		pos = pos3
		// Virt
		if p, n := _VirtAction(parser, pos); n == nil {
			goto fail6
		} else {
			node = *n
			pos = p
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

func _StructAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Struct, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ s:("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// s:("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
	{
		pos1 := pos
		// ("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
		// action
		// "{" fs:(n:IdentC t:TypeName {…})* _ "}"
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// fs:(n:IdentC t:TypeName {…})*
		{
			pos3 := pos
			// (n:IdentC t:TypeName {…})*
			for {
				pos5 := pos
				// (n:IdentC t:TypeName {…})
				// action
				// n:IdentC t:TypeName
				// n:IdentC
				{
					pos9 := pos
					// IdentC
					if !_accept(parser, _IdentCAccepts, &pos, &perr) {
						goto fail7
					}
					labels[0] = parser.text[pos9:pos]
				}
				// t:TypeName
				{
					pos10 := pos
					// TypeName
					if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
						goto fail7
					}
					labels[1] = parser.text[pos10:pos]
				}
				continue
			fail7:
				pos = pos5
				break
			}
			labels[2] = parser.text[pos3:pos]
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
		labels[3] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Struct, start, pos, perr)
fail:
	return _memoize(parser, _Struct, start, -1, perr)
}

func _StructNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_Struct]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Struct}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Struct"}
	// action
	// _ s:("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// s:("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
	{
		pos1 := pos
		// ("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// "{" fs:(n:IdentC t:TypeName {…})* _ "}"
			// "{"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// fs:(n:IdentC t:TypeName {…})*
			{
				pos5 := pos
				// (n:IdentC t:TypeName {…})*
				for {
					nkids6 := len(node.Kids)
					pos7 := pos
					// (n:IdentC t:TypeName {…})
					{
						nkids10 := len(node.Kids)
						pos011 := pos
						// action
						// n:IdentC t:TypeName
						// n:IdentC
						{
							pos13 := pos
							// IdentC
							if !_node(parser, _IdentCNode, node, &pos) {
								goto fail9
							}
							labels[0] = parser.text[pos13:pos]
						}
						// t:TypeName
						{
							pos14 := pos
							// TypeName
							if !_node(parser, _TypeNameNode, node, &pos) {
								goto fail9
							}
							labels[1] = parser.text[pos14:pos]
						}
						sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
						node.Kids = append(node.Kids[:nkids10], sub)
					}
					continue
				fail9:
					node.Kids = node.Kids[:nkids6]
					pos = pos7
					break
				}
				labels[2] = parser.text[pos5:pos]
			}
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail
			}
			// "}"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[3] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _StructFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
	use(labels)
	pos, failure := _failMemo(parser, _Struct, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Struct",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Struct}
	// action
	// _ s:("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// s:("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
	{
		pos1 := pos
		// ("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
		// action
		// "{" fs:(n:IdentC t:TypeName {…})* _ "}"
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
		// fs:(n:IdentC t:TypeName {…})*
		{
			pos3 := pos
			// (n:IdentC t:TypeName {…})*
			for {
				pos5 := pos
				// (n:IdentC t:TypeName {…})
				// action
				// n:IdentC t:TypeName
				// n:IdentC
				{
					pos9 := pos
					// IdentC
					if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
						goto fail7
					}
					labels[0] = parser.text[pos9:pos]
				}
				// t:TypeName
				{
					pos10 := pos
					// TypeName
					if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
						goto fail7
					}
					labels[1] = parser.text[pos10:pos]
				}
				continue
			fail7:
				pos = pos5
				break
			}
			labels[2] = parser.text[pos3:pos]
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
		labels[3] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _StructAction(parser *_Parser, start int) (int, *Def) {
	var labels [4]string
	use(labels)
	var label3 *Struct
	var label0 Ident
	var label1 TypeName
	var label2 []Parm
	dp := parser.deltaPos[start][_Struct]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Struct}
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
		// _ s:("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// s:("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
		{
			pos2 := pos
			// ("{" fs:(n:IdentC t:TypeName {…})* _ "}" {…})
			// action
			{
				start3 := pos
				// "{" fs:(n:IdentC t:TypeName {…})* _ "}"
				// "{"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
					goto fail
				}
				pos++
				// fs:(n:IdentC t:TypeName {…})*
				{
					pos5 := pos
					// (n:IdentC t:TypeName {…})*
					for {
						pos7 := pos
						var node8 Parm
						// (n:IdentC t:TypeName {…})
						// action
						{
							start10 := pos
							// n:IdentC t:TypeName
							// n:IdentC
							{
								pos12 := pos
								// IdentC
								if p, n := _IdentCAction(parser, pos); n == nil {
									goto fail9
								} else {
									label0 = *n
									pos = p
								}
								labels[0] = parser.text[pos12:pos]
							}
							// t:TypeName
							{
								pos13 := pos
								// TypeName
								if p, n := _TypeNameAction(parser, pos); n == nil {
									goto fail9
								} else {
									label1 = *n
									pos = p
								}
								labels[1] = parser.text[pos13:pos]
							}
							node8 = func(
								start, end int, n Ident, t TypeName) Parm {
								return Parm{location: n.location, Name: n.Text, Type: &t}
							}(
								start10, pos, label0, label1)
						}
						label2 = append(label2, node8)
						continue
					fail9:
						pos = pos7
						break
					}
					labels[2] = parser.text[pos5:pos]
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
				label3 = func(
					start, end int, fs []Parm, n Ident, t TypeName) *Struct {
					return &Struct{
						location: loc(parser, start, end),
						Fields:   fs,
					}
				}(
					start3, pos, label2, label0, label1)
			}
			labels[3] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, fs []Parm, n Ident, s *Struct, t TypeName) Def {
			return Def(s)
		}(
			start0, pos, label2, label0, label3, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _EnumAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Enum, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ e:("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// e:("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
	{
		pos1 := pos
		// ("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
		// action
		// "{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}"
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// c:Case
		{
			pos3 := pos
			// Case
			if !_accept(parser, _CaseAccepts, &pos, &perr) {
				goto fail
			}
			labels[0] = parser.text[pos3:pos]
		}
		// cs:(_ "," c1:Case {…})*
		{
			pos4 := pos
			// (_ "," c1:Case {…})*
			for {
				pos6 := pos
				// (_ "," c1:Case {…})
				// action
				// _ "," c1:Case
				// _
				if !_accept(parser, __Accepts, &pos, &perr) {
					goto fail8
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					perr = _max(perr, pos)
					goto fail8
				}
				pos++
				// c1:Case
				{
					pos10 := pos
					// Case
					if !_accept(parser, _CaseAccepts, &pos, &perr) {
						goto fail8
					}
					labels[1] = parser.text[pos10:pos]
				}
				continue
			fail8:
				pos = pos6
				break
			}
			labels[2] = parser.text[pos4:pos]
		}
		// (_ ",")?
		{
			pos12 := pos
			// (_ ",")
			// _ ","
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
			goto ok15
		fail13:
			pos = pos12
		ok15:
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
		labels[3] = parser.text[pos1:pos]
	}
	return _memoize(parser, _Enum, start, pos, perr)
fail:
	return _memoize(parser, _Enum, start, -1, perr)
}

func _EnumNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_Enum]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Enum}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Enum"}
	// action
	// _ e:("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// e:("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
	{
		pos1 := pos
		// ("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// "{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}"
			// "{"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// c:Case
			{
				pos5 := pos
				// Case
				if !_node(parser, _CaseNode, node, &pos) {
					goto fail
				}
				labels[0] = parser.text[pos5:pos]
			}
			// cs:(_ "," c1:Case {…})*
			{
				pos6 := pos
				// (_ "," c1:Case {…})*
				for {
					nkids7 := len(node.Kids)
					pos8 := pos
					// (_ "," c1:Case {…})
					{
						nkids11 := len(node.Kids)
						pos012 := pos
						// action
						// _ "," c1:Case
						// _
						if !_node(parser, __Node, node, &pos) {
							goto fail10
						}
						// ","
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
							goto fail10
						}
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
						pos++
						// c1:Case
						{
							pos14 := pos
							// Case
							if !_node(parser, _CaseNode, node, &pos) {
								goto fail10
							}
							labels[1] = parser.text[pos14:pos]
						}
						sub := _sub(parser, pos012, pos, node.Kids[nkids11:])
						node.Kids = append(node.Kids[:nkids11], sub)
					}
					continue
				fail10:
					node.Kids = node.Kids[:nkids7]
					pos = pos8
					break
				}
				labels[2] = parser.text[pos6:pos]
			}
			// (_ ",")?
			{
				nkids15 := len(node.Kids)
				pos16 := pos
				// (_ ",")
				{
					nkids18 := len(node.Kids)
					pos019 := pos
					// _ ","
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail17
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail17
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					sub := _sub(parser, pos019, pos, node.Kids[nkids18:])
					node.Kids = append(node.Kids[:nkids18], sub)
				}
				goto ok21
			fail17:
				node.Kids = node.Kids[:nkids15]
				pos = pos16
			ok21:
			}
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail
			}
			// "}"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[3] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _EnumFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
	use(labels)
	pos, failure := _failMemo(parser, _Enum, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Enum",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Enum}
	// action
	// _ e:("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// e:("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
	{
		pos1 := pos
		// ("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
		// action
		// "{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}"
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
		// c:Case
		{
			pos3 := pos
			// Case
			if !_fail(parser, _CaseFail, errPos, failure, &pos) {
				goto fail
			}
			labels[0] = parser.text[pos3:pos]
		}
		// cs:(_ "," c1:Case {…})*
		{
			pos4 := pos
			// (_ "," c1:Case {…})*
			for {
				pos6 := pos
				// (_ "," c1:Case {…})
				// action
				// _ "," c1:Case
				// _
				if !_fail(parser, __Fail, errPos, failure, &pos) {
					goto fail8
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\",\"",
						})
					}
					goto fail8
				}
				pos++
				// c1:Case
				{
					pos10 := pos
					// Case
					if !_fail(parser, _CaseFail, errPos, failure, &pos) {
						goto fail8
					}
					labels[1] = parser.text[pos10:pos]
				}
				continue
			fail8:
				pos = pos6
				break
			}
			labels[2] = parser.text[pos4:pos]
		}
		// (_ ",")?
		{
			pos12 := pos
			// (_ ",")
			// _ ","
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
			goto ok15
		fail13:
			pos = pos12
		ok15:
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
		labels[3] = parser.text[pos1:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _EnumAction(parser *_Parser, start int) (int, *Def) {
	var labels [4]string
	use(labels)
	var label0 Parm
	var label1 Parm
	var label2 []Parm
	var label3 *Enum
	dp := parser.deltaPos[start][_Enum]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Enum}
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
		// _ e:("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// e:("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
		{
			pos2 := pos
			// ("{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}" {…})
			// action
			{
				start3 := pos
				// "{" c:Case cs:(_ "," c1:Case {…})* (_ ",")? _ "}"
				// "{"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
					goto fail
				}
				pos++
				// c:Case
				{
					pos5 := pos
					// Case
					if p, n := _CaseAction(parser, pos); n == nil {
						goto fail
					} else {
						label0 = *n
						pos = p
					}
					labels[0] = parser.text[pos5:pos]
				}
				// cs:(_ "," c1:Case {…})*
				{
					pos6 := pos
					// (_ "," c1:Case {…})*
					for {
						pos8 := pos
						var node9 Parm
						// (_ "," c1:Case {…})
						// action
						{
							start11 := pos
							// _ "," c1:Case
							// _
							if p, n := __Action(parser, pos); n == nil {
								goto fail10
							} else {
								pos = p
							}
							// ","
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
								goto fail10
							}
							pos++
							// c1:Case
							{
								pos13 := pos
								// Case
								if p, n := _CaseAction(parser, pos); n == nil {
									goto fail10
								} else {
									label1 = *n
									pos = p
								}
								labels[1] = parser.text[pos13:pos]
							}
							node9 = func(
								start, end int, c Parm, c1 Parm) Parm {
								return Parm(c1)
							}(
								start11, pos, label0, label1)
						}
						label2 = append(label2, node9)
						continue
					fail10:
						pos = pos8
						break
					}
					labels[2] = parser.text[pos6:pos]
				}
				// (_ ",")?
				{
					pos15 := pos
					// (_ ",")
					// _ ","
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail16
					} else {
						pos = p
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail16
					}
					pos++
					goto ok18
				fail16:
					pos = pos15
				ok18:
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
				label3 = func(
					start, end int, c Parm, c1 Parm, cs []Parm) *Enum {
					return &Enum{
						location: loc(parser, start, end),
						Cases:    append([]Parm{c}, cs...),
					}
				}(
					start3, pos, label0, label1, label2)
			}
			labels[3] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, c Parm, c1 Parm, cs []Parm, e *Enum) Def {
			return Def(e)
		}(
			start0, pos, label0, label1, label2, label3)
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

func _CaseNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [3]string
	use(labels)
	dp := parser.deltaPos[start][_Case]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Case}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Case"}
	// id0:Ident {…}/id1:IdentC t:TypeName {…}
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// action
		// id0:Ident
		{
			pos5 := pos
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// id1:IdentC t:TypeName
		// id1:IdentC
		{
			pos8 := pos
			// IdentC
			if !_node(parser, _IdentCNode, node, &pos) {
				goto fail6
			}
			labels[1] = parser.text[pos8:pos]
		}
		// t:TypeName
		{
			pos9 := pos
			// TypeName
			if !_node(parser, _TypeNameNode, node, &pos) {
				goto fail6
			}
			labels[2] = parser.text[pos9:pos]
		}
		goto ok0
	fail6:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _CaseAction(parser *_Parser, start int) (int, *Parm) {
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
		n := n.(Parm)
		return start + int(dp-1), &n
	}
	var node Parm
	pos := start
	// id0:Ident {…}/id1:IdentC t:TypeName {…}
	{
		pos3 := pos
		var node2 Parm
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
				start, end int, id0 Ident) Parm {
				return Parm{
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
				start, end int, id0 Ident, id1 Ident, t TypeName) Parm {
				return Parm{
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
	var labels [2]string
	use(labels)
	if dp, de, ok := _memo(parser, _Virt, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ v:("{" ss:MethSig+ _ "}" {…})
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// v:("{" ss:MethSig+ _ "}" {…})
	{
		pos1 := pos
		// ("{" ss:MethSig+ _ "}" {…})
		// action
		// "{" ss:MethSig+ _ "}"
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			perr = _max(perr, pos)
			goto fail
		}
		pos++
		// ss:MethSig+
		{
			pos3 := pos
			// MethSig+
			// MethSig
			if !_accept(parser, _MethSigAccepts, &pos, &perr) {
				goto fail
			}
			for {
				pos5 := pos
				// MethSig
				if !_accept(parser, _MethSigAccepts, &pos, &perr) {
					goto fail7
				}
				continue
			fail7:
				pos = pos5
				break
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
	return _memoize(parser, _Virt, start, pos, perr)
fail:
	return _memoize(parser, _Virt, start, -1, perr)
}

func _VirtNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_Virt]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Virt}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Virt"}
	// action
	// _ v:("{" ss:MethSig+ _ "}" {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// v:("{" ss:MethSig+ _ "}" {…})
	{
		pos1 := pos
		// ("{" ss:MethSig+ _ "}" {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// "{" ss:MethSig+ _ "}"
			// "{"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// ss:MethSig+
			{
				pos5 := pos
				// MethSig+
				// MethSig
				if !_node(parser, _MethSigNode, node, &pos) {
					goto fail
				}
				for {
					nkids6 := len(node.Kids)
					pos7 := pos
					// MethSig
					if !_node(parser, _MethSigNode, node, &pos) {
						goto fail9
					}
					continue
				fail9:
					node.Kids = node.Kids[:nkids6]
					pos = pos7
					break
				}
				labels[0] = parser.text[pos5:pos]
			}
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail
			}
			// "}"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[1] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _VirtFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [2]string
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
	// _ v:("{" ss:MethSig+ _ "}" {…})
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// v:("{" ss:MethSig+ _ "}" {…})
	{
		pos1 := pos
		// ("{" ss:MethSig+ _ "}" {…})
		// action
		// "{" ss:MethSig+ _ "}"
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
		// ss:MethSig+
		{
			pos3 := pos
			// MethSig+
			// MethSig
			if !_fail(parser, _MethSigFail, errPos, failure, &pos) {
				goto fail
			}
			for {
				pos5 := pos
				// MethSig
				if !_fail(parser, _MethSigFail, errPos, failure, &pos) {
					goto fail7
				}
				continue
			fail7:
				pos = pos5
				break
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

func _VirtAction(parser *_Parser, start int) (int, *Def) {
	var labels [2]string
	use(labels)
	var label1 *Virt
	var label0 []MethSig
	dp := parser.deltaPos[start][_Virt]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Virt}
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
		// _ v:("{" ss:MethSig+ _ "}" {…})
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// v:("{" ss:MethSig+ _ "}" {…})
		{
			pos2 := pos
			// ("{" ss:MethSig+ _ "}" {…})
			// action
			{
				start3 := pos
				// "{" ss:MethSig+ _ "}"
				// "{"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
					goto fail
				}
				pos++
				// ss:MethSig+
				{
					pos5 := pos
					// MethSig+
					{
						var node8 MethSig
						// MethSig
						if p, n := _MethSigAction(parser, pos); n == nil {
							goto fail
						} else {
							node8 = *n
							pos = p
						}
						label0 = append(label0, node8)
					}
					for {
						pos7 := pos
						var node8 MethSig
						// MethSig
						if p, n := _MethSigAction(parser, pos); n == nil {
							goto fail9
						} else {
							node8 = *n
							pos = p
						}
						label0 = append(label0, node8)
						continue
					fail9:
						pos = pos7
						break
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
					start, end int, ss []MethSig) *Virt {
					return &Virt{
						location: loc(parser, start, end),
						Meths:    ss,
					}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, ss []MethSig, v *Virt) Def {
			return Def(v)
		}(
			start0, pos, label0, label1)
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

func _MethSigNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [8]string
	use(labels)
	dp := parser.deltaPos[start][_MethSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _MethSig}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "MethSig"}
	// action
	// _ sig:("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// sig:("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
	{
		pos1 := pos
		// ("[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]" {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// "[" ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+) r:Ret? _ "]"
			// "["
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// ps:(id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+)
			{
				pos5 := pos
				// (id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+)
				{
					nkids6 := len(node.Kids)
					pos07 := pos
					// id0:Ident {…}/op:Op t0:TypeName {…}/(id1:IdentC t1:TypeName {…})+
					{
						pos11 := pos
						nkids9 := len(node.Kids)
						// action
						// id0:Ident
						{
							pos13 := pos
							// Ident
							if !_node(parser, _IdentNode, node, &pos) {
								goto fail12
							}
							labels[0] = parser.text[pos13:pos]
						}
						goto ok8
					fail12:
						node.Kids = node.Kids[:nkids9]
						pos = pos11
						// action
						// op:Op t0:TypeName
						// op:Op
						{
							pos16 := pos
							// Op
							if !_node(parser, _OpNode, node, &pos) {
								goto fail14
							}
							labels[1] = parser.text[pos16:pos]
						}
						// t0:TypeName
						{
							pos17 := pos
							// TypeName
							if !_node(parser, _TypeNameNode, node, &pos) {
								goto fail14
							}
							labels[2] = parser.text[pos17:pos]
						}
						goto ok8
					fail14:
						node.Kids = node.Kids[:nkids9]
						pos = pos11
						// (id1:IdentC t1:TypeName {…})+
						// (id1:IdentC t1:TypeName {…})
						{
							nkids23 := len(node.Kids)
							pos024 := pos
							// action
							// id1:IdentC t1:TypeName
							// id1:IdentC
							{
								pos26 := pos
								// IdentC
								if !_node(parser, _IdentCNode, node, &pos) {
									goto fail18
								}
								labels[3] = parser.text[pos26:pos]
							}
							// t1:TypeName
							{
								pos27 := pos
								// TypeName
								if !_node(parser, _TypeNameNode, node, &pos) {
									goto fail18
								}
								labels[4] = parser.text[pos27:pos]
							}
							sub := _sub(parser, pos024, pos, node.Kids[nkids23:])
							node.Kids = append(node.Kids[:nkids23], sub)
						}
						for {
							nkids19 := len(node.Kids)
							pos20 := pos
							// (id1:IdentC t1:TypeName {…})
							{
								nkids28 := len(node.Kids)
								pos029 := pos
								// action
								// id1:IdentC t1:TypeName
								// id1:IdentC
								{
									pos31 := pos
									// IdentC
									if !_node(parser, _IdentCNode, node, &pos) {
										goto fail22
									}
									labels[3] = parser.text[pos31:pos]
								}
								// t1:TypeName
								{
									pos32 := pos
									// TypeName
									if !_node(parser, _TypeNameNode, node, &pos) {
										goto fail22
									}
									labels[4] = parser.text[pos32:pos]
								}
								sub := _sub(parser, pos029, pos, node.Kids[nkids28:])
								node.Kids = append(node.Kids[:nkids28], sub)
							}
							continue
						fail22:
							node.Kids = node.Kids[:nkids19]
							pos = pos20
							break
						}
						goto ok8
					fail18:
						node.Kids = node.Kids[:nkids9]
						pos = pos11
						goto fail
					ok8:
					}
					sub := _sub(parser, pos07, pos, node.Kids[nkids6:])
					node.Kids = append(node.Kids[:nkids6], sub)
				}
				labels[5] = parser.text[pos5:pos]
			}
			// r:Ret?
			{
				pos33 := pos
				// Ret?
				{
					nkids34 := len(node.Kids)
					pos35 := pos
					// Ret
					if !_node(parser, _RetNode, node, &pos) {
						goto fail36
					}
					goto ok37
				fail36:
					node.Kids = node.Kids[:nkids34]
					pos = pos35
				ok37:
				}
				labels[6] = parser.text[pos33:pos]
			}
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail
			}
			// "]"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[7] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _MethSigAction(parser *_Parser, start int) (int, *MethSig) {
	var labels [8]string
	use(labels)
	var label4 TypeName
	var label5 []parm
	var label6 *TypeName
	var label7 MethSig
	var label0 Ident
	var label1 Ident
	var label2 TypeName
	var label3 Ident
	dp := parser.deltaPos[start][_MethSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _MethSig}
	n := parser.act[key]
	if n != nil {
		n := n.(MethSig)
		return start + int(dp-1), &n
	}
	var node MethSig
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
					start, end int, id0 Ident, id1 Ident, op Ident, ps []parm, r *TypeName, t0 TypeName, t1 TypeName) MethSig {
					var s string
					var ts []TypeName
					for _, p := range ps {
						s += p.name.Text
						ts = append(ts, p.typ)
					}
					return MethSig{
						location: loc(parser, start, end),
						Sel:      s,
						Parms:    ts,
						Ret:      r,
					}
				}(
					start3, pos, label0, label3, label1, label5, label6, label2, label4)
			}
			labels[7] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, id0 Ident, id1 Ident, op Ident, ps []parm, r *TypeName, sig MethSig, t0 TypeName, t1 TypeName) MethSig {
			return MethSig(sig)
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

func _StmtsNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_Stmts]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Stmts}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Stmts"}
	// action
	// ss:(s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})?
	{
		pos0 := pos
		// (s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})?
		{
			nkids1 := len(node.Kids)
			pos2 := pos
			// (s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")? {…})
			{
				nkids4 := len(node.Kids)
				pos05 := pos
				// action
				// s0:Stmt s1s:(_ "." s1:Stmt {…})* (_ ".")?
				// s0:Stmt
				{
					pos7 := pos
					// Stmt
					if !_node(parser, _StmtNode, node, &pos) {
						goto fail3
					}
					labels[0] = parser.text[pos7:pos]
				}
				// s1s:(_ "." s1:Stmt {…})*
				{
					pos8 := pos
					// (_ "." s1:Stmt {…})*
					for {
						nkids9 := len(node.Kids)
						pos10 := pos
						// (_ "." s1:Stmt {…})
						{
							nkids13 := len(node.Kids)
							pos014 := pos
							// action
							// _ "." s1:Stmt
							// _
							if !_node(parser, __Node, node, &pos) {
								goto fail12
							}
							// "."
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
								goto fail12
							}
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
							pos++
							// s1:Stmt
							{
								pos16 := pos
								// Stmt
								if !_node(parser, _StmtNode, node, &pos) {
									goto fail12
								}
								labels[1] = parser.text[pos16:pos]
							}
							sub := _sub(parser, pos014, pos, node.Kids[nkids13:])
							node.Kids = append(node.Kids[:nkids13], sub)
						}
						continue
					fail12:
						node.Kids = node.Kids[:nkids9]
						pos = pos10
						break
					}
					labels[2] = parser.text[pos8:pos]
				}
				// (_ ".")?
				{
					nkids17 := len(node.Kids)
					pos18 := pos
					// (_ ".")
					{
						nkids20 := len(node.Kids)
						pos021 := pos
						// _ "."
						// _
						if !_node(parser, __Node, node, &pos) {
							goto fail19
						}
						// "."
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
							goto fail19
						}
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
						pos++
						sub := _sub(parser, pos021, pos, node.Kids[nkids20:])
						node.Kids = append(node.Kids[:nkids20], sub)
					}
					goto ok23
				fail19:
					node.Kids = node.Kids[:nkids17]
					pos = pos18
				ok23:
				}
				sub := _sub(parser, pos05, pos, node.Kids[nkids4:])
				node.Kids = append(node.Kids[:nkids4], sub)
			}
			goto ok24
		fail3:
			node.Kids = node.Kids[:nkids1]
			pos = pos2
		ok24:
		}
		labels[3] = parser.text[pos0:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
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
	var label3 *[]Stmt
	var label0 Stmt
	var label1 Stmt
	var label2 []Stmt
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

func _StmtNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [1]string
	use(labels)
	dp := parser.deltaPos[start][_Stmt]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Stmt}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Stmt"}
	// Return/Assign/e:Expr {…}
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// Return
		if !_node(parser, _ReturnNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Assign
		if !_node(parser, _AssignNode, node, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// e:Expr
		{
			pos7 := pos
			// Expr
			if !_node(parser, _ExprNode, node, &pos) {
				goto fail6
			}
			labels[0] = parser.text[pos7:pos]
		}
		goto ok0
	fail6:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _ReturnNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_Return]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Return}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Return"}
	// action
	// _ r:("^" e:Expr {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// r:("^" e:Expr {…})
	{
		pos1 := pos
		// ("^" e:Expr {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// "^" e:Expr
			// "^"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
				goto fail
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// e:Expr
			{
				pos5 := pos
				// Expr
				if !_node(parser, _ExprNode, node, &pos) {
					goto fail
				}
				labels[0] = parser.text[pos5:pos]
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[1] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	var label1 Ret
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
					start, end int, e Expr) Ret {
					return Ret{start: loc1(parser, start), Val: e}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, e Expr, r Ret) Stmt {
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

func _AssignNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_Assign]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Assign}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Assign"}
	// action
	// l:Lhs _ ":=" r:Expr
	// l:Lhs
	{
		pos1 := pos
		// Lhs
		if !_node(parser, _LhsNode, node, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// ":="
	if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
	pos += 2
	// r:Expr
	{
		pos2 := pos
		// Expr
		if !_node(parser, _ExprNode, node, &pos) {
			goto fail
		}
		labels[1] = parser.text[pos2:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	var label0 []Parm
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
			start, end int, l []Parm, r Expr) Stmt {
			return Stmt(Assign{Var: l, Val: r})
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

func _LhsNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [6]string
	use(labels)
	dp := parser.deltaPos[start][_Lhs]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Lhs}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Lhs"}
	// action
	// id:(i0:Ident t0:TypeName? {…}) is:(_ "," i1:Ident t1:TypeName? {…})*
	// id:(i0:Ident t0:TypeName? {…})
	{
		pos1 := pos
		// (i0:Ident t0:TypeName? {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// i0:Ident t0:TypeName?
			// i0:Ident
			{
				pos5 := pos
				// Ident
				if !_node(parser, _IdentNode, node, &pos) {
					goto fail
				}
				labels[0] = parser.text[pos5:pos]
			}
			// t0:TypeName?
			{
				pos6 := pos
				// TypeName?
				{
					nkids7 := len(node.Kids)
					pos8 := pos
					// TypeName
					if !_node(parser, _TypeNameNode, node, &pos) {
						goto fail9
					}
					goto ok10
				fail9:
					node.Kids = node.Kids[:nkids7]
					pos = pos8
				ok10:
				}
				labels[1] = parser.text[pos6:pos]
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[2] = parser.text[pos1:pos]
	}
	// is:(_ "," i1:Ident t1:TypeName? {…})*
	{
		pos11 := pos
		// (_ "," i1:Ident t1:TypeName? {…})*
		for {
			nkids12 := len(node.Kids)
			pos13 := pos
			// (_ "," i1:Ident t1:TypeName? {…})
			{
				nkids16 := len(node.Kids)
				pos017 := pos
				// action
				// _ "," i1:Ident t1:TypeName?
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail15
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					goto fail15
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// i1:Ident
				{
					pos19 := pos
					// Ident
					if !_node(parser, _IdentNode, node, &pos) {
						goto fail15
					}
					labels[3] = parser.text[pos19:pos]
				}
				// t1:TypeName?
				{
					pos20 := pos
					// TypeName?
					{
						nkids21 := len(node.Kids)
						pos22 := pos
						// TypeName
						if !_node(parser, _TypeNameNode, node, &pos) {
							goto fail23
						}
						goto ok24
					fail23:
						node.Kids = node.Kids[:nkids21]
						pos = pos22
					ok24:
					}
					labels[4] = parser.text[pos20:pos]
				}
				sub := _sub(parser, pos017, pos, node.Kids[nkids16:])
				node.Kids = append(node.Kids[:nkids16], sub)
			}
			continue
		fail15:
			node.Kids = node.Kids[:nkids12]
			pos = pos13
			break
		}
		labels[5] = parser.text[pos11:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _LhsAction(parser *_Parser, start int) (int, *[]Parm) {
	var labels [6]string
	use(labels)
	var label2 Parm
	var label3 Ident
	var label4 *TypeName
	var label5 []Parm
	var label0 Ident
	var label1 *TypeName
	dp := parser.deltaPos[start][_Lhs]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Lhs}
	n := parser.act[key]
	if n != nil {
		n := n.([]Parm)
		return start + int(dp-1), &n
	}
	var node []Parm
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
					start, end int, i0 Ident, t0 *TypeName) Parm {
					e := i0.end
					if t0 != nil {
						e = t0.end
					}
					return Parm{
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
				var node14 Parm
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
						start, end int, i0 Ident, i1 Ident, id Parm, t0 *TypeName, t1 *TypeName) Parm {
						e := i1.end
						if t1 != nil {
							e = t1.end
						}
						return Parm{
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
			start, end int, i0 Ident, i1 Ident, id Parm, is []Parm, t0 *TypeName, t1 *TypeName) []Parm {
			return []Parm(append([]Parm{id}, is...))
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

func _ExprNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_Expr]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Expr}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Expr"}
	// Call/Primary
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// Call
		if !_node(parser, _CallNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Primary
		if !_node(parser, _PrimaryNode, node, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	// c:(Nary/Binary/Unary) cs:(_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
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
	// cs:(_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
	{
		pos9 := pos
		// (_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		for {
			pos11 := pos
			// (_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})
			// action
			// _ ";" m:(UnaryMsg/BinMsg/NaryMsg)
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail13
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
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

func _CallNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [3]string
	use(labels)
	dp := parser.deltaPos[start][_Call]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Call}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Call"}
	// action
	// c:(Nary/Binary/Unary) cs:(_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
	// c:(Nary/Binary/Unary)
	{
		pos1 := pos
		// (Nary/Binary/Unary)
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// Nary/Binary/Unary
			{
				pos7 := pos
				nkids5 := len(node.Kids)
				// Nary
				if !_node(parser, _NaryNode, node, &pos) {
					goto fail8
				}
				goto ok4
			fail8:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				// Binary
				if !_node(parser, _BinaryNode, node, &pos) {
					goto fail9
				}
				goto ok4
			fail9:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				// Unary
				if !_node(parser, _UnaryNode, node, &pos) {
					goto fail10
				}
				goto ok4
			fail10:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				goto fail
			ok4:
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[0] = parser.text[pos1:pos]
	}
	// cs:(_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
	{
		pos11 := pos
		// (_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		for {
			nkids12 := len(node.Kids)
			pos13 := pos
			// (_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})
			{
				nkids16 := len(node.Kids)
				pos017 := pos
				// action
				// _ ";" m:(UnaryMsg/BinMsg/NaryMsg)
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail15
				}
				// ";"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
					goto fail15
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// m:(UnaryMsg/BinMsg/NaryMsg)
				{
					pos19 := pos
					// (UnaryMsg/BinMsg/NaryMsg)
					{
						nkids20 := len(node.Kids)
						pos021 := pos
						// UnaryMsg/BinMsg/NaryMsg
						{
							pos25 := pos
							nkids23 := len(node.Kids)
							// UnaryMsg
							if !_node(parser, _UnaryMsgNode, node, &pos) {
								goto fail26
							}
							goto ok22
						fail26:
							node.Kids = node.Kids[:nkids23]
							pos = pos25
							// BinMsg
							if !_node(parser, _BinMsgNode, node, &pos) {
								goto fail27
							}
							goto ok22
						fail27:
							node.Kids = node.Kids[:nkids23]
							pos = pos25
							// NaryMsg
							if !_node(parser, _NaryMsgNode, node, &pos) {
								goto fail28
							}
							goto ok22
						fail28:
							node.Kids = node.Kids[:nkids23]
							pos = pos25
							goto fail15
						ok22:
						}
						sub := _sub(parser, pos021, pos, node.Kids[nkids20:])
						node.Kids = append(node.Kids[:nkids20], sub)
					}
					labels[1] = parser.text[pos19:pos]
				}
				sub := _sub(parser, pos017, pos, node.Kids[nkids16:])
				node.Kids = append(node.Kids[:nkids16], sub)
			}
			continue
		fail15:
			node.Kids = node.Kids[:nkids12]
			pos = pos13
			break
		}
		labels[2] = parser.text[pos11:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	// c:(Nary/Binary/Unary) cs:(_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
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
	// cs:(_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
	{
		pos9 := pos
		// (_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		for {
			pos11 := pos
			// (_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})
			// action
			// _ ";" m:(UnaryMsg/BinMsg/NaryMsg)
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail13
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\";\"",
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
	var label0 Call
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
		// c:(Nary/Binary/Unary) cs:(_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		// c:(Nary/Binary/Unary)
		{
			pos2 := pos
			// (Nary/Binary/Unary)
			// Nary/Binary/Unary
			{
				pos6 := pos
				var node5 Call
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
		// cs:(_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
		{
			pos10 := pos
			// (_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})*
			for {
				pos12 := pos
				var node13 Msg
				// (_ ";" m:(UnaryMsg/BinMsg/NaryMsg) {…})
				// action
				{
					start15 := pos
					// _ ";" m:(UnaryMsg/BinMsg/NaryMsg)
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail14
					} else {
						pos = p
					}
					// ";"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
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
						start, end int, c Call, m Msg) Msg {
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
			start, end int, c Call, cs []Msg, m Msg) Expr {
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
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Unary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// r:(Primary/n:ModPath {…}) ms:UnaryMsg+
	// r:(Primary/n:ModPath {…})
	{
		pos1 := pos
		// (Primary/n:ModPath {…})
		// Primary/n:ModPath {…}
		{
			pos5 := pos
			// Primary
			if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
				goto fail6
			}
			goto ok2
		fail6:
			pos = pos5
			// action
			// n:ModPath
			{
				pos8 := pos
				// ModPath
				if !_accept(parser, _ModPathAccepts, &pos, &perr) {
					goto fail7
				}
				labels[0] = parser.text[pos8:pos]
			}
			goto ok2
		fail7:
			pos = pos5
			goto fail
		ok2:
		}
		labels[1] = parser.text[pos1:pos]
	}
	// ms:UnaryMsg+
	{
		pos9 := pos
		// UnaryMsg+
		// UnaryMsg
		if !_accept(parser, _UnaryMsgAccepts, &pos, &perr) {
			goto fail
		}
		for {
			pos11 := pos
			// UnaryMsg
			if !_accept(parser, _UnaryMsgAccepts, &pos, &perr) {
				goto fail13
			}
			continue
		fail13:
			pos = pos11
			break
		}
		labels[2] = parser.text[pos9:pos]
	}
	return _memoize(parser, _Unary, start, pos, perr)
fail:
	return _memoize(parser, _Unary, start, -1, perr)
}

func _UnaryNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [3]string
	use(labels)
	dp := parser.deltaPos[start][_Unary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Unary}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Unary"}
	// action
	// r:(Primary/n:ModPath {…}) ms:UnaryMsg+
	// r:(Primary/n:ModPath {…})
	{
		pos1 := pos
		// (Primary/n:ModPath {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// Primary/n:ModPath {…}
			{
				pos7 := pos
				nkids5 := len(node.Kids)
				// Primary
				if !_node(parser, _PrimaryNode, node, &pos) {
					goto fail8
				}
				goto ok4
			fail8:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				// action
				// n:ModPath
				{
					pos10 := pos
					// ModPath
					if !_node(parser, _ModPathNode, node, &pos) {
						goto fail9
					}
					labels[0] = parser.text[pos10:pos]
				}
				goto ok4
			fail9:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				goto fail
			ok4:
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[1] = parser.text[pos1:pos]
	}
	// ms:UnaryMsg+
	{
		pos11 := pos
		// UnaryMsg+
		// UnaryMsg
		if !_node(parser, _UnaryMsgNode, node, &pos) {
			goto fail
		}
		for {
			nkids12 := len(node.Kids)
			pos13 := pos
			// UnaryMsg
			if !_node(parser, _UnaryMsgNode, node, &pos) {
				goto fail15
			}
			continue
		fail15:
			node.Kids = node.Kids[:nkids12]
			pos = pos13
			break
		}
		labels[2] = parser.text[pos11:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _UnaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
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
	// r:(Primary/n:ModPath {…}) ms:UnaryMsg+
	// r:(Primary/n:ModPath {…})
	{
		pos1 := pos
		// (Primary/n:ModPath {…})
		// Primary/n:ModPath {…}
		{
			pos5 := pos
			// Primary
			if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
				goto fail6
			}
			goto ok2
		fail6:
			pos = pos5
			// action
			// n:ModPath
			{
				pos8 := pos
				// ModPath
				if !_fail(parser, _ModPathFail, errPos, failure, &pos) {
					goto fail7
				}
				labels[0] = parser.text[pos8:pos]
			}
			goto ok2
		fail7:
			pos = pos5
			goto fail
		ok2:
		}
		labels[1] = parser.text[pos1:pos]
	}
	// ms:UnaryMsg+
	{
		pos9 := pos
		// UnaryMsg+
		// UnaryMsg
		if !_fail(parser, _UnaryMsgFail, errPos, failure, &pos) {
			goto fail
		}
		for {
			pos11 := pos
			// UnaryMsg
			if !_fail(parser, _UnaryMsgFail, errPos, failure, &pos) {
				goto fail13
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

func _UnaryAction(parser *_Parser, start int) (int, *Call) {
	var labels [3]string
	use(labels)
	var label0 ModPath
	var label1 Expr
	var label2 []Msg
	dp := parser.deltaPos[start][_Unary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Unary}
	n := parser.act[key]
	if n != nil {
		n := n.(Call)
		return start + int(dp-1), &n
	}
	var node Call
	pos := start
	// action
	{
		start0 := pos
		// r:(Primary/n:ModPath {…}) ms:UnaryMsg+
		// r:(Primary/n:ModPath {…})
		{
			pos2 := pos
			// (Primary/n:ModPath {…})
			// Primary/n:ModPath {…}
			{
				pos6 := pos
				var node5 Expr
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail7
				} else {
					label1 = *n
					pos = p
				}
				goto ok3
			fail7:
				label1 = node5
				pos = pos6
				// action
				{
					start9 := pos
					// n:ModPath
					{
						pos10 := pos
						// ModPath
						if p, n := _ModPathAction(parser, pos); n == nil {
							goto fail8
						} else {
							label0 = *n
							pos = p
						}
						labels[0] = parser.text[pos10:pos]
					}
					label1 = func(
						start, end int, n ModPath) Expr {
						return Expr(n)
					}(
						start9, pos, label0)
				}
				goto ok3
			fail8:
				label1 = node5
				pos = pos6
				goto fail
			ok3:
			}
			labels[1] = parser.text[pos2:pos]
		}
		// ms:UnaryMsg+
		{
			pos11 := pos
			// UnaryMsg+
			{
				var node14 Msg
				// UnaryMsg
				if p, n := _UnaryMsgAction(parser, pos); n == nil {
					goto fail
				} else {
					node14 = *n
					pos = p
				}
				label2 = append(label2, node14)
			}
			for {
				pos13 := pos
				var node14 Msg
				// UnaryMsg
				if p, n := _UnaryMsgAction(parser, pos); n == nil {
					goto fail15
				} else {
					node14 = *n
					pos = p
				}
				label2 = append(label2, node14)
				continue
			fail15:
				pos = pos13
				break
			}
			labels[2] = parser.text[pos11:pos]
		}
		node = func(
			start, end int, ms []Msg, n ModPath, r Expr) Call {
			c := Call{
				location: location{r.Start(), ms[0].end},
				Recv:     r,
				Msgs:     []Msg{ms[0]},
			}
			for _, m := range ms[1:] {
				c = Call{
					location: location{r.Start(), m.end},
					Recv:     c,
					Msgs:     []Msg{m},
				}
			}
			return Call(c)
		}(
			start0, pos, label2, label0, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _UnaryMsgAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [1]string
	use(labels)
	if dp, de, ok := _memo(parser, _UnaryMsg, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// i:Ident
	{
		pos0 := pos
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos0:pos]
	}
	return _memoize(parser, _UnaryMsg, start, pos, perr)
fail:
	return _memoize(parser, _UnaryMsg, start, -1, perr)
}

func _UnaryMsgNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [1]string
	use(labels)
	dp := parser.deltaPos[start][_UnaryMsg]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _UnaryMsg}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "UnaryMsg"}
	// action
	// i:Ident
	{
		pos0 := pos
		// Ident
		if !_node(parser, _IdentNode, node, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos0:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _UnaryMsgFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [1]string
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
	// i:Ident
	{
		pos0 := pos
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos0:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _UnaryMsgAction(parser *_Parser, start int) (int, *Msg) {
	var labels [1]string
	use(labels)
	var label0 Ident
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
		// i:Ident
		{
			pos1 := pos
			// Ident
			if p, n := _IdentAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos1:pos]
		}
		node = func(
			start, end int, i Ident) Msg {
			return Msg{location: i.location, Sel: i.Text}
		}(
			start0, pos, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BinaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Binary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// r:(u:Unary {…}/Primary/n:ModPath {…}) m:BinMsg
	// r:(u:Unary {…}/Primary/n:ModPath {…})
	{
		pos1 := pos
		// (u:Unary {…}/Primary/n:ModPath {…})
		// u:Unary {…}/Primary/n:ModPath {…}
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
			// action
			// n:ModPath
			{
				pos10 := pos
				// ModPath
				if !_accept(parser, _ModPathAccepts, &pos, &perr) {
					goto fail9
				}
				labels[1] = parser.text[pos10:pos]
			}
			goto ok2
		fail9:
			pos = pos5
			goto fail
		ok2:
		}
		labels[2] = parser.text[pos1:pos]
	}
	// m:BinMsg
	{
		pos11 := pos
		// BinMsg
		if !_accept(parser, _BinMsgAccepts, &pos, &perr) {
			goto fail
		}
		labels[3] = parser.text[pos11:pos]
	}
	return _memoize(parser, _Binary, start, pos, perr)
fail:
	return _memoize(parser, _Binary, start, -1, perr)
}

func _BinaryNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_Binary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Binary}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Binary"}
	// action
	// r:(u:Unary {…}/Primary/n:ModPath {…}) m:BinMsg
	// r:(u:Unary {…}/Primary/n:ModPath {…})
	{
		pos1 := pos
		// (u:Unary {…}/Primary/n:ModPath {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// u:Unary {…}/Primary/n:ModPath {…}
			{
				pos7 := pos
				nkids5 := len(node.Kids)
				// action
				// u:Unary
				{
					pos9 := pos
					// Unary
					if !_node(parser, _UnaryNode, node, &pos) {
						goto fail8
					}
					labels[0] = parser.text[pos9:pos]
				}
				goto ok4
			fail8:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				// Primary
				if !_node(parser, _PrimaryNode, node, &pos) {
					goto fail10
				}
				goto ok4
			fail10:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				// action
				// n:ModPath
				{
					pos12 := pos
					// ModPath
					if !_node(parser, _ModPathNode, node, &pos) {
						goto fail11
					}
					labels[1] = parser.text[pos12:pos]
				}
				goto ok4
			fail11:
				node.Kids = node.Kids[:nkids5]
				pos = pos7
				goto fail
			ok4:
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[2] = parser.text[pos1:pos]
	}
	// m:BinMsg
	{
		pos13 := pos
		// BinMsg
		if !_node(parser, _BinMsgNode, node, &pos) {
			goto fail
		}
		labels[3] = parser.text[pos13:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _BinaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
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
	// r:(u:Unary {…}/Primary/n:ModPath {…}) m:BinMsg
	// r:(u:Unary {…}/Primary/n:ModPath {…})
	{
		pos1 := pos
		// (u:Unary {…}/Primary/n:ModPath {…})
		// u:Unary {…}/Primary/n:ModPath {…}
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
			// action
			// n:ModPath
			{
				pos10 := pos
				// ModPath
				if !_fail(parser, _ModPathFail, errPos, failure, &pos) {
					goto fail9
				}
				labels[1] = parser.text[pos10:pos]
			}
			goto ok2
		fail9:
			pos = pos5
			goto fail
		ok2:
		}
		labels[2] = parser.text[pos1:pos]
	}
	// m:BinMsg
	{
		pos11 := pos
		// BinMsg
		if !_fail(parser, _BinMsgFail, errPos, failure, &pos) {
			goto fail
		}
		labels[3] = parser.text[pos11:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _BinaryAction(parser *_Parser, start int) (int, *Call) {
	var labels [4]string
	use(labels)
	var label3 Msg
	var label0 Call
	var label1 ModPath
	var label2 Expr
	dp := parser.deltaPos[start][_Binary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Binary}
	n := parser.act[key]
	if n != nil {
		n := n.(Call)
		return start + int(dp-1), &n
	}
	var node Call
	pos := start
	// action
	{
		start0 := pos
		// r:(u:Unary {…}/Primary/n:ModPath {…}) m:BinMsg
		// r:(u:Unary {…}/Primary/n:ModPath {…})
		{
			pos2 := pos
			// (u:Unary {…}/Primary/n:ModPath {…})
			// u:Unary {…}/Primary/n:ModPath {…}
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
					label2 = func(
						start, end int, u Call) Expr {
						return Expr(u)
					}(
						start8, pos, label0)
				}
				goto ok3
			fail7:
				label2 = node5
				pos = pos6
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail10
				} else {
					label2 = *n
					pos = p
				}
				goto ok3
			fail10:
				label2 = node5
				pos = pos6
				// action
				{
					start12 := pos
					// n:ModPath
					{
						pos13 := pos
						// ModPath
						if p, n := _ModPathAction(parser, pos); n == nil {
							goto fail11
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos13:pos]
					}
					label2 = func(
						start, end int, n ModPath, u Call) Expr {
						return Expr(n)
					}(
						start12, pos, label1, label0)
				}
				goto ok3
			fail11:
				label2 = node5
				pos = pos6
				goto fail
			ok3:
			}
			labels[2] = parser.text[pos2:pos]
		}
		// m:BinMsg
		{
			pos14 := pos
			// BinMsg
			if p, n := _BinMsgAction(parser, pos); n == nil {
				goto fail
			} else {
				label3 = *n
				pos = p
			}
			labels[3] = parser.text[pos14:pos]
		}
		node = func(
			start, end int, m Msg, n ModPath, r Expr, u Call) Call {
			return Call{
				location: location{r.Start(), loc1(parser, end)},
				Recv:     r,
				Msgs:     []Msg{m},
			}
		}(
			start0, pos, label3, label1, label2, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BinMsgAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _BinMsg, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// n:Op a:(b:Binary {…}/u:Unary {…}/Primary)
	// n:Op
	{
		pos1 := pos
		// Op
		if !_accept(parser, _OpAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// a:(b:Binary {…}/u:Unary {…}/Primary)
	{
		pos2 := pos
		// (b:Binary {…}/u:Unary {…}/Primary)
		// b:Binary {…}/u:Unary {…}/Primary
		{
			pos6 := pos
			// action
			// b:Binary
			{
				pos8 := pos
				// Binary
				if !_accept(parser, _BinaryAccepts, &pos, &perr) {
					goto fail7
				}
				labels[1] = parser.text[pos8:pos]
			}
			goto ok3
		fail7:
			pos = pos6
			// action
			// u:Unary
			{
				pos10 := pos
				// Unary
				if !_accept(parser, _UnaryAccepts, &pos, &perr) {
					goto fail9
				}
				labels[2] = parser.text[pos10:pos]
			}
			goto ok3
		fail9:
			pos = pos6
			// Primary
			if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
				goto fail11
			}
			goto ok3
		fail11:
			pos = pos6
			goto fail
		ok3:
		}
		labels[3] = parser.text[pos2:pos]
	}
	return _memoize(parser, _BinMsg, start, pos, perr)
fail:
	return _memoize(parser, _BinMsg, start, -1, perr)
}

func _BinMsgNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_BinMsg]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _BinMsg}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "BinMsg"}
	// action
	// n:Op a:(b:Binary {…}/u:Unary {…}/Primary)
	// n:Op
	{
		pos1 := pos
		// Op
		if !_node(parser, _OpNode, node, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// a:(b:Binary {…}/u:Unary {…}/Primary)
	{
		pos2 := pos
		// (b:Binary {…}/u:Unary {…}/Primary)
		{
			nkids3 := len(node.Kids)
			pos04 := pos
			// b:Binary {…}/u:Unary {…}/Primary
			{
				pos8 := pos
				nkids6 := len(node.Kids)
				// action
				// b:Binary
				{
					pos10 := pos
					// Binary
					if !_node(parser, _BinaryNode, node, &pos) {
						goto fail9
					}
					labels[1] = parser.text[pos10:pos]
				}
				goto ok5
			fail9:
				node.Kids = node.Kids[:nkids6]
				pos = pos8
				// action
				// u:Unary
				{
					pos12 := pos
					// Unary
					if !_node(parser, _UnaryNode, node, &pos) {
						goto fail11
					}
					labels[2] = parser.text[pos12:pos]
				}
				goto ok5
			fail11:
				node.Kids = node.Kids[:nkids6]
				pos = pos8
				// Primary
				if !_node(parser, _PrimaryNode, node, &pos) {
					goto fail13
				}
				goto ok5
			fail13:
				node.Kids = node.Kids[:nkids6]
				pos = pos8
				goto fail
			ok5:
			}
			sub := _sub(parser, pos04, pos, node.Kids[nkids3:])
			node.Kids = append(node.Kids[:nkids3], sub)
		}
		labels[3] = parser.text[pos2:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _BinMsgFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
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
	// n:Op a:(b:Binary {…}/u:Unary {…}/Primary)
	// n:Op
	{
		pos1 := pos
		// Op
		if !_fail(parser, _OpFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// a:(b:Binary {…}/u:Unary {…}/Primary)
	{
		pos2 := pos
		// (b:Binary {…}/u:Unary {…}/Primary)
		// b:Binary {…}/u:Unary {…}/Primary
		{
			pos6 := pos
			// action
			// b:Binary
			{
				pos8 := pos
				// Binary
				if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
					goto fail7
				}
				labels[1] = parser.text[pos8:pos]
			}
			goto ok3
		fail7:
			pos = pos6
			// action
			// u:Unary
			{
				pos10 := pos
				// Unary
				if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
					goto fail9
				}
				labels[2] = parser.text[pos10:pos]
			}
			goto ok3
		fail9:
			pos = pos6
			// Primary
			if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
				goto fail11
			}
			goto ok3
		fail11:
			pos = pos6
			goto fail
		ok3:
		}
		labels[3] = parser.text[pos2:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _BinMsgAction(parser *_Parser, start int) (int, *Msg) {
	var labels [4]string
	use(labels)
	var label0 Ident
	var label1 Call
	var label2 Call
	var label3 Expr
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
		// n:Op a:(b:Binary {…}/u:Unary {…}/Primary)
		// n:Op
		{
			pos2 := pos
			// Op
			if p, n := _OpAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos2:pos]
		}
		// a:(b:Binary {…}/u:Unary {…}/Primary)
		{
			pos3 := pos
			// (b:Binary {…}/u:Unary {…}/Primary)
			// b:Binary {…}/u:Unary {…}/Primary
			{
				pos7 := pos
				var node6 Expr
				// action
				{
					start9 := pos
					// b:Binary
					{
						pos10 := pos
						// Binary
						if p, n := _BinaryAction(parser, pos); n == nil {
							goto fail8
						} else {
							label1 = *n
							pos = p
						}
						labels[1] = parser.text[pos10:pos]
					}
					label3 = func(
						start, end int, b Call, n Ident) Expr {
						return Expr(b)
					}(
						start9, pos, label1, label0)
				}
				goto ok4
			fail8:
				label3 = node6
				pos = pos7
				// action
				{
					start12 := pos
					// u:Unary
					{
						pos13 := pos
						// Unary
						if p, n := _UnaryAction(parser, pos); n == nil {
							goto fail11
						} else {
							label2 = *n
							pos = p
						}
						labels[2] = parser.text[pos13:pos]
					}
					label3 = func(
						start, end int, b Call, n Ident, u Call) Expr {
						return Expr(u)
					}(
						start12, pos, label1, label0, label2)
				}
				goto ok4
			fail11:
				label3 = node6
				pos = pos7
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail14
				} else {
					label3 = *n
					pos = p
				}
				goto ok4
			fail14:
				label3 = node6
				pos = pos7
				goto fail
			ok4:
			}
			labels[3] = parser.text[pos3:pos]
		}
		node = func(
			start, end int, a Expr, b Call, n Ident, u Call) Msg {
			return Msg{
				location: location{n.start, loc1(parser, end)},
				Sel:      n.Text,
				Args:     []Expr{a},
			}
		}(
			start0, pos, label3, label1, label0, label2)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _NaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [5]string
	use(labels)
	if dp, de, ok := _memo(parser, _Nary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// r:(b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})? m:NaryMsg
	// r:(b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})?
	{
		pos1 := pos
		// (b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})?
		{
			pos3 := pos
			// (b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})
			// b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…}
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
				// action
				// n:ModPath
				{
					pos15 := pos
					// ModPath
					if !_accept(parser, _ModPathAccepts, &pos, &perr) {
						goto fail14
					}
					labels[2] = parser.text[pos15:pos]
				}
				goto ok5
			fail14:
				pos = pos8
				goto fail4
			ok5:
			}
			goto ok16
		fail4:
			pos = pos3
		ok16:
		}
		labels[3] = parser.text[pos1:pos]
	}
	// m:NaryMsg
	{
		pos17 := pos
		// NaryMsg
		if !_accept(parser, _NaryMsgAccepts, &pos, &perr) {
			goto fail
		}
		labels[4] = parser.text[pos17:pos]
	}
	return _memoize(parser, _Nary, start, pos, perr)
fail:
	return _memoize(parser, _Nary, start, -1, perr)
}

func _NaryNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [5]string
	use(labels)
	dp := parser.deltaPos[start][_Nary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Nary}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Nary"}
	// action
	// r:(b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})? m:NaryMsg
	// r:(b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})?
	{
		pos1 := pos
		// (b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})?
		{
			nkids2 := len(node.Kids)
			pos3 := pos
			// (b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})
			{
				nkids5 := len(node.Kids)
				pos06 := pos
				// b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…}
				{
					pos10 := pos
					nkids8 := len(node.Kids)
					// action
					// b:Binary
					{
						pos12 := pos
						// Binary
						if !_node(parser, _BinaryNode, node, &pos) {
							goto fail11
						}
						labels[0] = parser.text[pos12:pos]
					}
					goto ok7
				fail11:
					node.Kids = node.Kids[:nkids8]
					pos = pos10
					// action
					// u:Unary
					{
						pos14 := pos
						// Unary
						if !_node(parser, _UnaryNode, node, &pos) {
							goto fail13
						}
						labels[1] = parser.text[pos14:pos]
					}
					goto ok7
				fail13:
					node.Kids = node.Kids[:nkids8]
					pos = pos10
					// Primary
					if !_node(parser, _PrimaryNode, node, &pos) {
						goto fail15
					}
					goto ok7
				fail15:
					node.Kids = node.Kids[:nkids8]
					pos = pos10
					// action
					// n:ModPath
					{
						pos17 := pos
						// ModPath
						if !_node(parser, _ModPathNode, node, &pos) {
							goto fail16
						}
						labels[2] = parser.text[pos17:pos]
					}
					goto ok7
				fail16:
					node.Kids = node.Kids[:nkids8]
					pos = pos10
					goto fail4
				ok7:
				}
				sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
				node.Kids = append(node.Kids[:nkids5], sub)
			}
			goto ok18
		fail4:
			node.Kids = node.Kids[:nkids2]
			pos = pos3
		ok18:
		}
		labels[3] = parser.text[pos1:pos]
	}
	// m:NaryMsg
	{
		pos19 := pos
		// NaryMsg
		if !_node(parser, _NaryMsgNode, node, &pos) {
			goto fail
		}
		labels[4] = parser.text[pos19:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _NaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [5]string
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
	// r:(b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})? m:NaryMsg
	// r:(b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})?
	{
		pos1 := pos
		// (b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})?
		{
			pos3 := pos
			// (b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})
			// b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…}
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
				// action
				// n:ModPath
				{
					pos15 := pos
					// ModPath
					if !_fail(parser, _ModPathFail, errPos, failure, &pos) {
						goto fail14
					}
					labels[2] = parser.text[pos15:pos]
				}
				goto ok5
			fail14:
				pos = pos8
				goto fail4
			ok5:
			}
			goto ok16
		fail4:
			pos = pos3
		ok16:
		}
		labels[3] = parser.text[pos1:pos]
	}
	// m:NaryMsg
	{
		pos17 := pos
		// NaryMsg
		if !_fail(parser, _NaryMsgFail, errPos, failure, &pos) {
			goto fail
		}
		labels[4] = parser.text[pos17:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _NaryAction(parser *_Parser, start int) (int, *Call) {
	var labels [5]string
	use(labels)
	var label2 ModPath
	var label3 *Expr
	var label4 Msg
	var label0 Call
	var label1 Call
	dp := parser.deltaPos[start][_Nary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Nary}
	n := parser.act[key]
	if n != nil {
		n := n.(Call)
		return start + int(dp-1), &n
	}
	var node Call
	pos := start
	// action
	{
		start0 := pos
		// r:(b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})? m:NaryMsg
		// r:(b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})?
		{
			pos2 := pos
			// (b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})?
			{
				pos4 := pos
				label3 = new(Expr)
				// (b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…})
				// b:Binary {…}/u:Unary {…}/Primary/n:ModPath {…}
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
						*label3 = func(
							start, end int, b Call) Expr {
							return Expr(b)
						}(
							start11, pos, label0)
					}
					goto ok6
				fail10:
					*label3 = node8
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
						*label3 = func(
							start, end int, b Call, u Call) Expr {
							return Expr(u)
						}(
							start14, pos, label0, label1)
					}
					goto ok6
				fail13:
					*label3 = node8
					pos = pos9
					// Primary
					if p, n := _PrimaryAction(parser, pos); n == nil {
						goto fail16
					} else {
						*label3 = *n
						pos = p
					}
					goto ok6
				fail16:
					*label3 = node8
					pos = pos9
					// action
					{
						start18 := pos
						// n:ModPath
						{
							pos19 := pos
							// ModPath
							if p, n := _ModPathAction(parser, pos); n == nil {
								goto fail17
							} else {
								label2 = *n
								pos = p
							}
							labels[2] = parser.text[pos19:pos]
						}
						*label3 = func(
							start, end int, b Call, n ModPath, u Call) Expr {
							return Expr(n)
						}(
							start18, pos, label0, label2, label1)
					}
					goto ok6
				fail17:
					*label3 = node8
					pos = pos9
					goto fail5
				ok6:
				}
				goto ok20
			fail5:
				label3 = nil
				pos = pos4
			ok20:
			}
			labels[3] = parser.text[pos2:pos]
		}
		// m:NaryMsg
		{
			pos21 := pos
			// NaryMsg
			if p, n := _NaryMsgAction(parser, pos); n == nil {
				goto fail
			} else {
				label4 = *n
				pos = p
			}
			labels[4] = parser.text[pos21:pos]
		}
		node = func(
			start, end int, b Call, m Msg, n ModPath, r *Expr, u Call) Call {
			s := m.start
			var recv Expr
			if r != nil {
				s = (*r).Start()
				recv = *r
			}
			return Call{
				location: location{s, loc1(parser, end)},
				Recv:     recv,
				Msgs:     []Msg{m},
			}
		}(
			start0, pos, label0, label4, label2, label3, label1)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _NaryMsgAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [5]string
	use(labels)
	if dp, de, ok := _memo(parser, _NaryMsg, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
	{
		pos0 := pos
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
		// action
		// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
		// n:IdentC
		{
			pos6 := pos
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail
			}
			labels[0] = parser.text[pos6:pos]
		}
		// v:(b:Binary {…}/u:Unary {…}/Primary)
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
					labels[1] = parser.text[pos13:pos]
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
					labels[2] = parser.text[pos15:pos]
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
			labels[3] = parser.text[pos7:pos]
		}
		for {
			pos2 := pos
			// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
			// action
			// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
			// n:IdentC
			{
				pos18 := pos
				// IdentC
				if !_accept(parser, _IdentCAccepts, &pos, &perr) {
					goto fail4
				}
				labels[0] = parser.text[pos18:pos]
			}
			// v:(b:Binary {…}/u:Unary {…}/Primary)
			{
				pos19 := pos
				// (b:Binary {…}/u:Unary {…}/Primary)
				// b:Binary {…}/u:Unary {…}/Primary
				{
					pos23 := pos
					// action
					// b:Binary
					{
						pos25 := pos
						// Binary
						if !_accept(parser, _BinaryAccepts, &pos, &perr) {
							goto fail24
						}
						labels[1] = parser.text[pos25:pos]
					}
					goto ok20
				fail24:
					pos = pos23
					// action
					// u:Unary
					{
						pos27 := pos
						// Unary
						if !_accept(parser, _UnaryAccepts, &pos, &perr) {
							goto fail26
						}
						labels[2] = parser.text[pos27:pos]
					}
					goto ok20
				fail26:
					pos = pos23
					// Primary
					if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
						goto fail28
					}
					goto ok20
				fail28:
					pos = pos23
					goto fail4
				ok20:
				}
				labels[3] = parser.text[pos19:pos]
			}
			continue
		fail4:
			pos = pos2
			break
		}
		labels[4] = parser.text[pos0:pos]
	}
	return _memoize(parser, _NaryMsg, start, pos, perr)
fail:
	return _memoize(parser, _NaryMsg, start, -1, perr)
}

func _NaryMsgNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [5]string
	use(labels)
	dp := parser.deltaPos[start][_NaryMsg]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _NaryMsg}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "NaryMsg"}
	// action
	// as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
	{
		pos0 := pos
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
		{
			nkids5 := len(node.Kids)
			pos06 := pos
			// action
			// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
			// n:IdentC
			{
				pos8 := pos
				// IdentC
				if !_node(parser, _IdentCNode, node, &pos) {
					goto fail
				}
				labels[0] = parser.text[pos8:pos]
			}
			// v:(b:Binary {…}/u:Unary {…}/Primary)
			{
				pos9 := pos
				// (b:Binary {…}/u:Unary {…}/Primary)
				{
					nkids10 := len(node.Kids)
					pos011 := pos
					// b:Binary {…}/u:Unary {…}/Primary
					{
						pos15 := pos
						nkids13 := len(node.Kids)
						// action
						// b:Binary
						{
							pos17 := pos
							// Binary
							if !_node(parser, _BinaryNode, node, &pos) {
								goto fail16
							}
							labels[1] = parser.text[pos17:pos]
						}
						goto ok12
					fail16:
						node.Kids = node.Kids[:nkids13]
						pos = pos15
						// action
						// u:Unary
						{
							pos19 := pos
							// Unary
							if !_node(parser, _UnaryNode, node, &pos) {
								goto fail18
							}
							labels[2] = parser.text[pos19:pos]
						}
						goto ok12
					fail18:
						node.Kids = node.Kids[:nkids13]
						pos = pos15
						// Primary
						if !_node(parser, _PrimaryNode, node, &pos) {
							goto fail20
						}
						goto ok12
					fail20:
						node.Kids = node.Kids[:nkids13]
						pos = pos15
						goto fail
					ok12:
					}
					sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
					node.Kids = append(node.Kids[:nkids10], sub)
				}
				labels[3] = parser.text[pos9:pos]
			}
			sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
			node.Kids = append(node.Kids[:nkids5], sub)
		}
		for {
			nkids1 := len(node.Kids)
			pos2 := pos
			// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
			{
				nkids21 := len(node.Kids)
				pos022 := pos
				// action
				// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
				// n:IdentC
				{
					pos24 := pos
					// IdentC
					if !_node(parser, _IdentCNode, node, &pos) {
						goto fail4
					}
					labels[0] = parser.text[pos24:pos]
				}
				// v:(b:Binary {…}/u:Unary {…}/Primary)
				{
					pos25 := pos
					// (b:Binary {…}/u:Unary {…}/Primary)
					{
						nkids26 := len(node.Kids)
						pos027 := pos
						// b:Binary {…}/u:Unary {…}/Primary
						{
							pos31 := pos
							nkids29 := len(node.Kids)
							// action
							// b:Binary
							{
								pos33 := pos
								// Binary
								if !_node(parser, _BinaryNode, node, &pos) {
									goto fail32
								}
								labels[1] = parser.text[pos33:pos]
							}
							goto ok28
						fail32:
							node.Kids = node.Kids[:nkids29]
							pos = pos31
							// action
							// u:Unary
							{
								pos35 := pos
								// Unary
								if !_node(parser, _UnaryNode, node, &pos) {
									goto fail34
								}
								labels[2] = parser.text[pos35:pos]
							}
							goto ok28
						fail34:
							node.Kids = node.Kids[:nkids29]
							pos = pos31
							// Primary
							if !_node(parser, _PrimaryNode, node, &pos) {
								goto fail36
							}
							goto ok28
						fail36:
							node.Kids = node.Kids[:nkids29]
							pos = pos31
							goto fail4
						ok28:
						}
						sub := _sub(parser, pos027, pos, node.Kids[nkids26:])
						node.Kids = append(node.Kids[:nkids26], sub)
					}
					labels[3] = parser.text[pos25:pos]
				}
				sub := _sub(parser, pos022, pos, node.Kids[nkids21:])
				node.Kids = append(node.Kids[:nkids21], sub)
			}
			continue
		fail4:
			node.Kids = node.Kids[:nkids1]
			pos = pos2
			break
		}
		labels[4] = parser.text[pos0:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _NaryMsgFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [5]string
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
	// as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
	{
		pos0 := pos
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
		// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
		// action
		// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
		// n:IdentC
		{
			pos6 := pos
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail
			}
			labels[0] = parser.text[pos6:pos]
		}
		// v:(b:Binary {…}/u:Unary {…}/Primary)
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
					labels[1] = parser.text[pos13:pos]
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
					labels[2] = parser.text[pos15:pos]
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
			labels[3] = parser.text[pos7:pos]
		}
		for {
			pos2 := pos
			// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
			// action
			// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
			// n:IdentC
			{
				pos18 := pos
				// IdentC
				if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
					goto fail4
				}
				labels[0] = parser.text[pos18:pos]
			}
			// v:(b:Binary {…}/u:Unary {…}/Primary)
			{
				pos19 := pos
				// (b:Binary {…}/u:Unary {…}/Primary)
				// b:Binary {…}/u:Unary {…}/Primary
				{
					pos23 := pos
					// action
					// b:Binary
					{
						pos25 := pos
						// Binary
						if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
							goto fail24
						}
						labels[1] = parser.text[pos25:pos]
					}
					goto ok20
				fail24:
					pos = pos23
					// action
					// u:Unary
					{
						pos27 := pos
						// Unary
						if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
							goto fail26
						}
						labels[2] = parser.text[pos27:pos]
					}
					goto ok20
				fail26:
					pos = pos23
					// Primary
					if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
						goto fail28
					}
					goto ok20
				fail28:
					pos = pos23
					goto fail4
				ok20:
				}
				labels[3] = parser.text[pos19:pos]
			}
			continue
		fail4:
			pos = pos2
			break
		}
		labels[4] = parser.text[pos0:pos]
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _NaryMsgAction(parser *_Parser, start int) (int, *Msg) {
	var labels [5]string
	use(labels)
	var label0 Ident
	var label1 Call
	var label2 Call
	var label3 Expr
	var label4 []arg
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
		// as:(n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
		{
			pos1 := pos
			// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})+
			{
				var node4 arg
				// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
				// action
				{
					start6 := pos
					// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
					// n:IdentC
					{
						pos8 := pos
						// IdentC
						if p, n := _IdentCAction(parser, pos); n == nil {
							goto fail
						} else {
							label0 = *n
							pos = p
						}
						labels[0] = parser.text[pos8:pos]
					}
					// v:(b:Binary {…}/u:Unary {…}/Primary)
					{
						pos9 := pos
						// (b:Binary {…}/u:Unary {…}/Primary)
						// b:Binary {…}/u:Unary {…}/Primary
						{
							pos13 := pos
							var node12 Expr
							// action
							{
								start15 := pos
								// b:Binary
								{
									pos16 := pos
									// Binary
									if p, n := _BinaryAction(parser, pos); n == nil {
										goto fail14
									} else {
										label1 = *n
										pos = p
									}
									labels[1] = parser.text[pos16:pos]
								}
								label3 = func(
									start, end int, b Call, n Ident) Expr {
									return Expr(b)
								}(
									start15, pos, label1, label0)
							}
							goto ok10
						fail14:
							label3 = node12
							pos = pos13
							// action
							{
								start18 := pos
								// u:Unary
								{
									pos19 := pos
									// Unary
									if p, n := _UnaryAction(parser, pos); n == nil {
										goto fail17
									} else {
										label2 = *n
										pos = p
									}
									labels[2] = parser.text[pos19:pos]
								}
								label3 = func(
									start, end int, b Call, n Ident, u Call) Expr {
									return Expr(u)
								}(
									start18, pos, label1, label0, label2)
							}
							goto ok10
						fail17:
							label3 = node12
							pos = pos13
							// Primary
							if p, n := _PrimaryAction(parser, pos); n == nil {
								goto fail20
							} else {
								label3 = *n
								pos = p
							}
							goto ok10
						fail20:
							label3 = node12
							pos = pos13
							goto fail
						ok10:
						}
						labels[3] = parser.text[pos9:pos]
					}
					node4 = func(
						start, end int, b Call, n Ident, u Call, v Expr) arg {
						return arg{n, v}
					}(
						start6, pos, label1, label0, label2, label3)
				}
				label4 = append(label4, node4)
			}
			for {
				pos3 := pos
				var node4 arg
				// (n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary) {…})
				// action
				{
					start21 := pos
					// n:IdentC v:(b:Binary {…}/u:Unary {…}/Primary)
					// n:IdentC
					{
						pos23 := pos
						// IdentC
						if p, n := _IdentCAction(parser, pos); n == nil {
							goto fail5
						} else {
							label0 = *n
							pos = p
						}
						labels[0] = parser.text[pos23:pos]
					}
					// v:(b:Binary {…}/u:Unary {…}/Primary)
					{
						pos24 := pos
						// (b:Binary {…}/u:Unary {…}/Primary)
						// b:Binary {…}/u:Unary {…}/Primary
						{
							pos28 := pos
							var node27 Expr
							// action
							{
								start30 := pos
								// b:Binary
								{
									pos31 := pos
									// Binary
									if p, n := _BinaryAction(parser, pos); n == nil {
										goto fail29
									} else {
										label1 = *n
										pos = p
									}
									labels[1] = parser.text[pos31:pos]
								}
								label3 = func(
									start, end int, b Call, n Ident) Expr {
									return Expr(b)
								}(
									start30, pos, label1, label0)
							}
							goto ok25
						fail29:
							label3 = node27
							pos = pos28
							// action
							{
								start33 := pos
								// u:Unary
								{
									pos34 := pos
									// Unary
									if p, n := _UnaryAction(parser, pos); n == nil {
										goto fail32
									} else {
										label2 = *n
										pos = p
									}
									labels[2] = parser.text[pos34:pos]
								}
								label3 = func(
									start, end int, b Call, n Ident, u Call) Expr {
									return Expr(u)
								}(
									start33, pos, label1, label0, label2)
							}
							goto ok25
						fail32:
							label3 = node27
							pos = pos28
							// Primary
							if p, n := _PrimaryAction(parser, pos); n == nil {
								goto fail35
							} else {
								label3 = *n
								pos = p
							}
							goto ok25
						fail35:
							label3 = node27
							pos = pos28
							goto fail5
						ok25:
						}
						labels[3] = parser.text[pos24:pos]
					}
					node4 = func(
						start, end int, b Call, n Ident, u Call, v Expr) arg {
						return arg{n, v}
					}(
						start21, pos, label1, label0, label2, label3)
				}
				label4 = append(label4, node4)
				continue
			fail5:
				pos = pos3
				break
			}
			labels[4] = parser.text[pos1:pos]
		}
		node = func(
			start, end int, as []arg, b Call, n Ident, u Call, v Expr) Msg {
			var sel string
			var es []Expr
			for _, a := range as {
				sel += a.name.Text
				es = append(es, a.val)
			}
			return Msg{
				location: location{as[0].name.start, loc1(parser, end)},
				Sel:      sel,
				Args:     es,
			}
		}(
			start0, pos, label4, label1, label0, label2, label3)
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
	// i:Ident {…}/Int/Float/Rune/s:String {…}/Ctor/Block/_ "(" e:Expr _ ")" {…}
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
		// Int
		if !_accept(parser, _IntAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		// Float
		if !_accept(parser, _FloatAccepts, &pos, &perr) {
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

func _PrimaryNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [3]string
	use(labels)
	dp := parser.deltaPos[start][_Primary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Primary}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Primary"}
	// i:Ident {…}/Int/Float/Rune/s:String {…}/Ctor/Block/_ "(" e:Expr _ ")" {…}
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// action
		// i:Ident
		{
			pos5 := pos
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail4
			}
			labels[0] = parser.text[pos5:pos]
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Int
		if !_node(parser, _IntNode, node, &pos) {
			goto fail6
		}
		goto ok0
	fail6:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Float
		if !_node(parser, _FloatNode, node, &pos) {
			goto fail7
		}
		goto ok0
	fail7:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Rune
		if !_node(parser, _RuneNode, node, &pos) {
			goto fail8
		}
		goto ok0
	fail8:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// s:String
		{
			pos10 := pos
			// String
			if !_node(parser, _StringNode, node, &pos) {
				goto fail9
			}
			labels[1] = parser.text[pos10:pos]
		}
		goto ok0
	fail9:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Ctor
		if !_node(parser, _CtorNode, node, &pos) {
			goto fail11
		}
		goto ok0
	fail11:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Block
		if !_node(parser, _BlockNode, node, &pos) {
			goto fail12
		}
		goto ok0
	fail12:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// _ "(" e:Expr _ ")"
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail13
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			goto fail13
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// e:Expr
		{
			pos15 := pos
			// Expr
			if !_node(parser, _ExprNode, node, &pos) {
				goto fail13
			}
			labels[2] = parser.text[pos15:pos]
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail13
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			goto fail13
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail13:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	// i:Ident {…}/Int/Float/Rune/s:String {…}/Ctor/Block/_ "(" e:Expr _ ")" {…}
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
		// Int
		if !_fail(parser, _IntFail, errPos, failure, &pos) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		// Float
		if !_fail(parser, _FloatFail, errPos, failure, &pos) {
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
	// i:Ident {…}/Int/Float/Rune/s:String {…}/Ctor/Block/_ "(" e:Expr _ ")" {…}
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
				return Expr(i)
			}(
				start5, pos, label0)
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Int
		if p, n := _IntAction(parser, pos); n == nil {
			goto fail7
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail7:
		node = node2
		pos = pos3
		// Float
		if p, n := _FloatAction(parser, pos); n == nil {
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
				return Expr(s)
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
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Ctor, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ ("{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}")
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// ("{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}")
	// "{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}"
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	// t:TypeName
	{
		pos2 := pos
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail
		}
		labels[0] = parser.text[pos2:pos]
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
	// as:((IdentC e0:Expr {…})+/a:Ary? {…})
	{
		pos3 := pos
		// ((IdentC e0:Expr {…})+/a:Ary? {…})
		// (IdentC e0:Expr {…})+/a:Ary? {…}
		{
			pos7 := pos
			// (IdentC e0:Expr {…})+
			// (IdentC e0:Expr {…})
			// action
			// IdentC e0:Expr
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail8
			}
			// e0:Expr
			{
				pos14 := pos
				// Expr
				if !_accept(parser, _ExprAccepts, &pos, &perr) {
					goto fail8
				}
				labels[1] = parser.text[pos14:pos]
			}
			for {
				pos10 := pos
				// (IdentC e0:Expr {…})
				// action
				// IdentC e0:Expr
				// IdentC
				if !_accept(parser, _IdentCAccepts, &pos, &perr) {
					goto fail12
				}
				// e0:Expr
				{
					pos16 := pos
					// Expr
					if !_accept(parser, _ExprAccepts, &pos, &perr) {
						goto fail12
					}
					labels[1] = parser.text[pos16:pos]
				}
				continue
			fail12:
				pos = pos10
				break
			}
			goto ok4
		fail8:
			pos = pos7
			// action
			// a:Ary?
			{
				pos18 := pos
				// Ary?
				{
					pos20 := pos
					// Ary
					if !_accept(parser, _AryAccepts, &pos, &perr) {
						goto fail21
					}
					goto ok22
				fail21:
					pos = pos20
				ok22:
				}
				labels[2] = parser.text[pos18:pos]
			}
		ok4:
		}
		labels[3] = parser.text[pos3:pos]
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
	return _memoize(parser, _Ctor, start, pos, perr)
fail:
	return _memoize(parser, _Ctor, start, -1, perr)
}

func _CtorNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_Ctor]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ctor}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Ctor"}
	// action
	// _ ("{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}")
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// ("{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}")
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// "{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}"
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			goto fail
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// t:TypeName
		{
			pos4 := pos
			// TypeName
			if !_node(parser, _TypeNameNode, node, &pos) {
				goto fail
			}
			labels[0] = parser.text[pos4:pos]
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			goto fail
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// as:((IdentC e0:Expr {…})+/a:Ary? {…})
		{
			pos5 := pos
			// ((IdentC e0:Expr {…})+/a:Ary? {…})
			{
				nkids6 := len(node.Kids)
				pos07 := pos
				// (IdentC e0:Expr {…})+/a:Ary? {…}
				{
					pos11 := pos
					nkids9 := len(node.Kids)
					// (IdentC e0:Expr {…})+
					// (IdentC e0:Expr {…})
					{
						nkids17 := len(node.Kids)
						pos018 := pos
						// action
						// IdentC e0:Expr
						// IdentC
						if !_node(parser, _IdentCNode, node, &pos) {
							goto fail12
						}
						// e0:Expr
						{
							pos20 := pos
							// Expr
							if !_node(parser, _ExprNode, node, &pos) {
								goto fail12
							}
							labels[1] = parser.text[pos20:pos]
						}
						sub := _sub(parser, pos018, pos, node.Kids[nkids17:])
						node.Kids = append(node.Kids[:nkids17], sub)
					}
					for {
						nkids13 := len(node.Kids)
						pos14 := pos
						// (IdentC e0:Expr {…})
						{
							nkids21 := len(node.Kids)
							pos022 := pos
							// action
							// IdentC e0:Expr
							// IdentC
							if !_node(parser, _IdentCNode, node, &pos) {
								goto fail16
							}
							// e0:Expr
							{
								pos24 := pos
								// Expr
								if !_node(parser, _ExprNode, node, &pos) {
									goto fail16
								}
								labels[1] = parser.text[pos24:pos]
							}
							sub := _sub(parser, pos022, pos, node.Kids[nkids21:])
							node.Kids = append(node.Kids[:nkids21], sub)
						}
						continue
					fail16:
						node.Kids = node.Kids[:nkids13]
						pos = pos14
						break
					}
					goto ok8
				fail12:
					node.Kids = node.Kids[:nkids9]
					pos = pos11
					// action
					// a:Ary?
					{
						pos26 := pos
						// Ary?
						{
							nkids27 := len(node.Kids)
							pos28 := pos
							// Ary
							if !_node(parser, _AryNode, node, &pos) {
								goto fail29
							}
							goto ok30
						fail29:
							node.Kids = node.Kids[:nkids27]
							pos = pos28
						ok30:
						}
						labels[2] = parser.text[pos26:pos]
					}
				ok8:
				}
				sub := _sub(parser, pos07, pos, node.Kids[nkids6:])
				node.Kids = append(node.Kids[:nkids6], sub)
			}
			labels[3] = parser.text[pos5:pos]
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail
		}
		// "}"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
			goto fail
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _CtorFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
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
	// _ ("{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}")
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// ("{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}")
	// "{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}"
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
	// t:TypeName
	{
		pos2 := pos
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos2:pos]
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
	// as:((IdentC e0:Expr {…})+/a:Ary? {…})
	{
		pos3 := pos
		// ((IdentC e0:Expr {…})+/a:Ary? {…})
		// (IdentC e0:Expr {…})+/a:Ary? {…}
		{
			pos7 := pos
			// (IdentC e0:Expr {…})+
			// (IdentC e0:Expr {…})
			// action
			// IdentC e0:Expr
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail8
			}
			// e0:Expr
			{
				pos14 := pos
				// Expr
				if !_fail(parser, _ExprFail, errPos, failure, &pos) {
					goto fail8
				}
				labels[1] = parser.text[pos14:pos]
			}
			for {
				pos10 := pos
				// (IdentC e0:Expr {…})
				// action
				// IdentC e0:Expr
				// IdentC
				if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
					goto fail12
				}
				// e0:Expr
				{
					pos16 := pos
					// Expr
					if !_fail(parser, _ExprFail, errPos, failure, &pos) {
						goto fail12
					}
					labels[1] = parser.text[pos16:pos]
				}
				continue
			fail12:
				pos = pos10
				break
			}
			goto ok4
		fail8:
			pos = pos7
			// action
			// a:Ary?
			{
				pos18 := pos
				// Ary?
				{
					pos20 := pos
					// Ary
					if !_fail(parser, _AryFail, errPos, failure, &pos) {
						goto fail21
					}
					goto ok22
				fail21:
					pos = pos20
				ok22:
				}
				labels[2] = parser.text[pos18:pos]
			}
		ok4:
		}
		labels[3] = parser.text[pos3:pos]
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

func _CtorAction(parser *_Parser, start int) (int, *Expr) {
	var labels [4]string
	use(labels)
	var label2 *[]Expr
	var label3 []Expr
	var label0 TypeName
	var label1 Expr
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
		// _ ("{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}")
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// ("{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}")
		// "{" t:TypeName _ "|" as:((IdentC e0:Expr {…})+/a:Ary? {…}) _ "}"
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			goto fail
		}
		pos++
		// t:TypeName
		{
			pos3 := pos
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail
			} else {
				label0 = *n
				pos = p
			}
			labels[0] = parser.text[pos3:pos]
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
		// as:((IdentC e0:Expr {…})+/a:Ary? {…})
		{
			pos4 := pos
			// ((IdentC e0:Expr {…})+/a:Ary? {…})
			// (IdentC e0:Expr {…})+/a:Ary? {…}
			{
				pos8 := pos
				var node7 []Expr
				// (IdentC e0:Expr {…})+
				{
					var node12 Expr
					// (IdentC e0:Expr {…})
					// action
					{
						start14 := pos
						// IdentC e0:Expr
						// IdentC
						if p, n := _IdentCAction(parser, pos); n == nil {
							goto fail9
						} else {
							pos = p
						}
						// e0:Expr
						{
							pos16 := pos
							// Expr
							if p, n := _ExprAction(parser, pos); n == nil {
								goto fail9
							} else {
								label1 = *n
								pos = p
							}
							labels[1] = parser.text[pos16:pos]
						}
						node12 = func(
							start, end int, e0 Expr, t TypeName) Expr {
							return Expr(e0)
						}(
							start14, pos, label1, label0)
					}
					label3 = append(label3, node12)
				}
				for {
					pos11 := pos
					var node12 Expr
					// (IdentC e0:Expr {…})
					// action
					{
						start17 := pos
						// IdentC e0:Expr
						// IdentC
						if p, n := _IdentCAction(parser, pos); n == nil {
							goto fail13
						} else {
							pos = p
						}
						// e0:Expr
						{
							pos19 := pos
							// Expr
							if p, n := _ExprAction(parser, pos); n == nil {
								goto fail13
							} else {
								label1 = *n
								pos = p
							}
							labels[1] = parser.text[pos19:pos]
						}
						node12 = func(
							start, end int, e0 Expr, t TypeName) Expr {
							return Expr(e0)
						}(
							start17, pos, label1, label0)
					}
					label3 = append(label3, node12)
					continue
				fail13:
					pos = pos11
					break
				}
				goto ok5
			fail9:
				label3 = node7
				pos = pos8
				// action
				{
					start21 := pos
					// a:Ary?
					{
						pos22 := pos
						// Ary?
						{
							pos24 := pos
							label2 = new([]Expr)
							// Ary
							if p, n := _AryAction(parser, pos); n == nil {
								goto fail25
							} else {
								*label2 = *n
								pos = p
							}
							goto ok26
						fail25:
							label2 = nil
							pos = pos24
						ok26:
						}
						labels[2] = parser.text[pos22:pos]
					}
					label3 = func(
						start, end int, a *[]Expr, e0 Expr, t TypeName) []Expr {
						if a != nil {
							return *a
						}
						return []Expr{}
					}(
						start21, pos, label2, label1, label0)
				}
			ok5:
			}
			labels[3] = parser.text[pos4:pos]
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
			start, end int, a *[]Expr, as []Expr, e0 Expr, t TypeName) Expr {
			return Expr(Ctor{
				location: loc(parser, start, end),
				Type:     t,
				Args:     as,
			})
		}(
			start0, pos, label2, label3, label1, label0)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _AryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [3]string
	use(labels)
	if dp, de, ok := _memo(parser, _Ary, start); ok {
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
	return _memoize(parser, _Ary, start, pos, perr)
fail:
	return _memoize(parser, _Ary, start, -1, perr)
}

func _AryNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [3]string
	use(labels)
	dp := parser.deltaPos[start][_Ary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ary}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Ary"}
	// action
	// e0:Expr es:(_ ";" e:Expr {…})* (_ ";")?
	// e0:Expr
	{
		pos1 := pos
		// Expr
		if !_node(parser, _ExprNode, node, &pos) {
			goto fail
		}
		labels[0] = parser.text[pos1:pos]
	}
	// es:(_ ";" e:Expr {…})*
	{
		pos2 := pos
		// (_ ";" e:Expr {…})*
		for {
			nkids3 := len(node.Kids)
			pos4 := pos
			// (_ ";" e:Expr {…})
			{
				nkids7 := len(node.Kids)
				pos08 := pos
				// action
				// _ ";" e:Expr
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail6
				}
				// ";"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
					goto fail6
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// e:Expr
				{
					pos10 := pos
					// Expr
					if !_node(parser, _ExprNode, node, &pos) {
						goto fail6
					}
					labels[1] = parser.text[pos10:pos]
				}
				sub := _sub(parser, pos08, pos, node.Kids[nkids7:])
				node.Kids = append(node.Kids[:nkids7], sub)
			}
			continue
		fail6:
			node.Kids = node.Kids[:nkids3]
			pos = pos4
			break
		}
		labels[2] = parser.text[pos2:pos]
	}
	// (_ ";")?
	{
		nkids11 := len(node.Kids)
		pos12 := pos
		// (_ ";")
		{
			nkids14 := len(node.Kids)
			pos015 := pos
			// _ ";"
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail13
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
				goto fail13
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			sub := _sub(parser, pos015, pos, node.Kids[nkids14:])
			node.Kids = append(node.Kids[:nkids14], sub)
		}
		goto ok17
	fail13:
		node.Kids = node.Kids[:nkids11]
		pos = pos12
	ok17:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _AryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [3]string
	use(labels)
	pos, failure := _failMemo(parser, _Ary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Ary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Ary}
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

func _AryAction(parser *_Parser, start int) (int, *[]Expr) {
	var labels [3]string
	use(labels)
	var label0 Expr
	var label1 Expr
	var label2 []Expr
	dp := parser.deltaPos[start][_Ary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ary}
	n := parser.act[key]
	if n != nil {
		n := n.([]Expr)
		return start + int(dp-1), &n
	}
	var node []Expr
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
			return []Expr(append([]Expr{e0}, es...))
		}(
			start0, pos, label1, label0, label2)
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BlockAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	var labels [4]string
	use(labels)
	if dp, de, ok := _memo(parser, _Block, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// action
	// _ ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]")
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]")
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
		pos4 := pos
		// (ps:(n:CIdent t:TypeName? {…})+ _ "|")
		// ps:(n:CIdent t:TypeName? {…})+ _ "|"
		// ps:(n:CIdent t:TypeName? {…})+
		{
			pos7 := pos
			// (n:CIdent t:TypeName? {…})+
			// (n:CIdent t:TypeName? {…})
			// action
			// n:CIdent t:TypeName?
			// n:CIdent
			{
				pos13 := pos
				// CIdent
				if !_accept(parser, _CIdentAccepts, &pos, &perr) {
					goto fail5
				}
				labels[0] = parser.text[pos13:pos]
			}
			// t:TypeName?
			{
				pos14 := pos
				// TypeName?
				{
					pos16 := pos
					// TypeName
					if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
						goto fail17
					}
					goto ok18
				fail17:
					pos = pos16
				ok18:
				}
				labels[1] = parser.text[pos14:pos]
			}
			for {
				pos9 := pos
				// (n:CIdent t:TypeName? {…})
				// action
				// n:CIdent t:TypeName?
				// n:CIdent
				{
					pos20 := pos
					// CIdent
					if !_accept(parser, _CIdentAccepts, &pos, &perr) {
						goto fail11
					}
					labels[0] = parser.text[pos20:pos]
				}
				// t:TypeName?
				{
					pos21 := pos
					// TypeName?
					{
						pos23 := pos
						// TypeName
						if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
							goto fail24
						}
						goto ok25
					fail24:
						pos = pos23
					ok25:
					}
					labels[1] = parser.text[pos21:pos]
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
			goto fail5
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			perr = _max(perr, pos)
			goto fail5
		}
		pos++
		goto ok26
	fail5:
		pos = pos4
	ok26:
	}
	// ss:Stmts
	{
		pos27 := pos
		// Stmts
		if !_accept(parser, _StmtsAccepts, &pos, &perr) {
			goto fail
		}
		labels[3] = parser.text[pos27:pos]
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
	return _memoize(parser, _Block, start, pos, perr)
fail:
	return _memoize(parser, _Block, start, -1, perr)
}

func _BlockNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [4]string
	use(labels)
	dp := parser.deltaPos[start][_Block]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Block}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Block"}
	// action
	// _ ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]")
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]")
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// "[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]"
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			goto fail
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts)
		{
			nkids4 := len(node.Kids)
			pos05 := pos
			// (ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts
			// (ps:(n:CIdent t:TypeName? {…})+ _ "|")?
			{
				nkids7 := len(node.Kids)
				pos8 := pos
				// (ps:(n:CIdent t:TypeName? {…})+ _ "|")
				{
					nkids10 := len(node.Kids)
					pos011 := pos
					// ps:(n:CIdent t:TypeName? {…})+ _ "|"
					// ps:(n:CIdent t:TypeName? {…})+
					{
						pos13 := pos
						// (n:CIdent t:TypeName? {…})+
						// (n:CIdent t:TypeName? {…})
						{
							nkids18 := len(node.Kids)
							pos019 := pos
							// action
							// n:CIdent t:TypeName?
							// n:CIdent
							{
								pos21 := pos
								// CIdent
								if !_node(parser, _CIdentNode, node, &pos) {
									goto fail9
								}
								labels[0] = parser.text[pos21:pos]
							}
							// t:TypeName?
							{
								pos22 := pos
								// TypeName?
								{
									nkids23 := len(node.Kids)
									pos24 := pos
									// TypeName
									if !_node(parser, _TypeNameNode, node, &pos) {
										goto fail25
									}
									goto ok26
								fail25:
									node.Kids = node.Kids[:nkids23]
									pos = pos24
								ok26:
								}
								labels[1] = parser.text[pos22:pos]
							}
							sub := _sub(parser, pos019, pos, node.Kids[nkids18:])
							node.Kids = append(node.Kids[:nkids18], sub)
						}
						for {
							nkids14 := len(node.Kids)
							pos15 := pos
							// (n:CIdent t:TypeName? {…})
							{
								nkids27 := len(node.Kids)
								pos028 := pos
								// action
								// n:CIdent t:TypeName?
								// n:CIdent
								{
									pos30 := pos
									// CIdent
									if !_node(parser, _CIdentNode, node, &pos) {
										goto fail17
									}
									labels[0] = parser.text[pos30:pos]
								}
								// t:TypeName?
								{
									pos31 := pos
									// TypeName?
									{
										nkids32 := len(node.Kids)
										pos33 := pos
										// TypeName
										if !_node(parser, _TypeNameNode, node, &pos) {
											goto fail34
										}
										goto ok35
									fail34:
										node.Kids = node.Kids[:nkids32]
										pos = pos33
									ok35:
									}
									labels[1] = parser.text[pos31:pos]
								}
								sub := _sub(parser, pos028, pos, node.Kids[nkids27:])
								node.Kids = append(node.Kids[:nkids27], sub)
							}
							continue
						fail17:
							node.Kids = node.Kids[:nkids14]
							pos = pos15
							break
						}
						labels[2] = parser.text[pos13:pos]
					}
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail9
					}
					// "|"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
						goto fail9
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
					node.Kids = append(node.Kids[:nkids10], sub)
				}
				goto ok36
			fail9:
				node.Kids = node.Kids[:nkids7]
				pos = pos8
			ok36:
			}
			// ss:Stmts
			{
				pos37 := pos
				// Stmts
				if !_node(parser, _StmtsNode, node, &pos) {
					goto fail
				}
				labels[3] = parser.text[pos37:pos]
			}
			sub := _sub(parser, pos05, pos, node.Kids[nkids4:])
			node.Kids = append(node.Kids[:nkids4], sub)
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			goto fail
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _BlockFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	var labels [4]string
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
	// _ ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]")
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]")
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
		pos4 := pos
		// (ps:(n:CIdent t:TypeName? {…})+ _ "|")
		// ps:(n:CIdent t:TypeName? {…})+ _ "|"
		// ps:(n:CIdent t:TypeName? {…})+
		{
			pos7 := pos
			// (n:CIdent t:TypeName? {…})+
			// (n:CIdent t:TypeName? {…})
			// action
			// n:CIdent t:TypeName?
			// n:CIdent
			{
				pos13 := pos
				// CIdent
				if !_fail(parser, _CIdentFail, errPos, failure, &pos) {
					goto fail5
				}
				labels[0] = parser.text[pos13:pos]
			}
			// t:TypeName?
			{
				pos14 := pos
				// TypeName?
				{
					pos16 := pos
					// TypeName
					if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
						goto fail17
					}
					goto ok18
				fail17:
					pos = pos16
				ok18:
				}
				labels[1] = parser.text[pos14:pos]
			}
			for {
				pos9 := pos
				// (n:CIdent t:TypeName? {…})
				// action
				// n:CIdent t:TypeName?
				// n:CIdent
				{
					pos20 := pos
					// CIdent
					if !_fail(parser, _CIdentFail, errPos, failure, &pos) {
						goto fail11
					}
					labels[0] = parser.text[pos20:pos]
				}
				// t:TypeName?
				{
					pos21 := pos
					// TypeName?
					{
						pos23 := pos
						// TypeName
						if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
							goto fail24
						}
						goto ok25
					fail24:
						pos = pos23
					ok25:
					}
					labels[1] = parser.text[pos21:pos]
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
			goto fail5
		}
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"|\"",
				})
			}
			goto fail5
		}
		pos++
		goto ok26
	fail5:
		pos = pos4
	ok26:
	}
	// ss:Stmts
	{
		pos27 := pos
		// Stmts
		if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
			goto fail
		}
		labels[3] = parser.text[pos27:pos]
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
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _BlockAction(parser *_Parser, start int) (int, *Expr) {
	var labels [4]string
	use(labels)
	var label1 *TypeName
	var label2 []Parm
	var label3 []Stmt
	var label0 Ident
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
		// _ ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]")
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			pos = p
		}
		// ("[" ((ps:(n:CIdent t:TypeName? {…})+ _ "|")? ss:Stmts) _ "]")
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
			pos5 := pos
			// (ps:(n:CIdent t:TypeName? {…})+ _ "|")
			// ps:(n:CIdent t:TypeName? {…})+ _ "|"
			// ps:(n:CIdent t:TypeName? {…})+
			{
				pos8 := pos
				// (n:CIdent t:TypeName? {…})+
				{
					var node11 Parm
					// (n:CIdent t:TypeName? {…})
					// action
					{
						start13 := pos
						// n:CIdent t:TypeName?
						// n:CIdent
						{
							pos15 := pos
							// CIdent
							if p, n := _CIdentAction(parser, pos); n == nil {
								goto fail6
							} else {
								label0 = *n
								pos = p
							}
							labels[0] = parser.text[pos15:pos]
						}
						// t:TypeName?
						{
							pos16 := pos
							// TypeName?
							{
								pos18 := pos
								label1 = new(TypeName)
								// TypeName
								if p, n := _TypeNameAction(parser, pos); n == nil {
									goto fail19
								} else {
									*label1 = *n
									pos = p
								}
								goto ok20
							fail19:
								label1 = nil
								pos = pos18
							ok20:
							}
							labels[1] = parser.text[pos16:pos]
						}
						node11 = func(
							start, end int, n Ident, t *TypeName) Parm {
							return Parm{Name: n.Text, Type: t}
						}(
							start13, pos, label0, label1)
					}
					label2 = append(label2, node11)
				}
				for {
					pos10 := pos
					var node11 Parm
					// (n:CIdent t:TypeName? {…})
					// action
					{
						start21 := pos
						// n:CIdent t:TypeName?
						// n:CIdent
						{
							pos23 := pos
							// CIdent
							if p, n := _CIdentAction(parser, pos); n == nil {
								goto fail12
							} else {
								label0 = *n
								pos = p
							}
							labels[0] = parser.text[pos23:pos]
						}
						// t:TypeName?
						{
							pos24 := pos
							// TypeName?
							{
								pos26 := pos
								label1 = new(TypeName)
								// TypeName
								if p, n := _TypeNameAction(parser, pos); n == nil {
									goto fail27
								} else {
									*label1 = *n
									pos = p
								}
								goto ok28
							fail27:
								label1 = nil
								pos = pos26
							ok28:
							}
							labels[1] = parser.text[pos24:pos]
						}
						node11 = func(
							start, end int, n Ident, t *TypeName) Parm {
							return Parm{Name: n.Text, Type: t}
						}(
							start21, pos, label0, label1)
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
				goto fail6
			} else {
				pos = p
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				goto fail6
			}
			pos++
			goto ok29
		fail6:
			pos = pos5
		ok29:
		}
		// ss:Stmts
		{
			pos30 := pos
			// Stmts
			if p, n := _StmtsAction(parser, pos); n == nil {
				goto fail
			} else {
				label3 = *n
				pos = p
			}
			labels[3] = parser.text[pos30:pos]
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
		node = func(
			start, end int, n Ident, ps []Parm, ss []Stmt, t *TypeName) Expr {
			return Expr(Block{
				location: loc(parser, start, end),
				Parms:    ps,
				Stmts:    ss,
			})
		}(
			start0, pos, label0, label2, label3, label1)
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

func _IntNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_Int]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Int}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Int"}
	// action
	// _ tok:(text:([+\-]? [0-9]+) {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// tok:(text:([+\-]? [0-9]+) {…})
	{
		pos1 := pos
		// (text:([+\-]? [0-9]+) {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// text:([+\-]? [0-9]+)
			{
				pos4 := pos
				// ([+\-]? [0-9]+)
				{
					nkids5 := len(node.Kids)
					pos06 := pos
					// [+\-]? [0-9]+
					// [+\-]?
					{
						nkids8 := len(node.Kids)
						pos9 := pos
						// [+\-]
						if r, w := _next(parser, pos); r != '+' && r != '-' {
							goto fail10
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						goto ok11
					fail10:
						node.Kids = node.Kids[:nkids8]
						pos = pos9
					ok11:
					}
					// [0-9]+
					// [0-9]
					if r, w := _next(parser, pos); r < '0' || r > '9' {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					for {
						nkids12 := len(node.Kids)
						pos13 := pos
						// [0-9]
						if r, w := _next(parser, pos); r < '0' || r > '9' {
							goto fail15
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail15:
						node.Kids = node.Kids[:nkids12]
						pos = pos13
						break
					}
					sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
					node.Kids = append(node.Kids[:nkids5], sub)
				}
				labels[0] = parser.text[pos4:pos]
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[1] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	var label1 Int
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
					start, end int, text string) Int {
					return Int{location: loc(parser, start, end), Text: text}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, text string, tok Int) Expr {
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

func _FloatNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_Float]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Float}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Float"}
	// action
	// _ tok:(text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// tok:(text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
	{
		pos1 := pos
		// (text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?) {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// text:([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?)
			{
				pos4 := pos
				// ([+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?)
				{
					nkids5 := len(node.Kids)
					pos06 := pos
					// [+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?
					// [+\-]?
					{
						nkids8 := len(node.Kids)
						pos9 := pos
						// [+\-]
						if r, w := _next(parser, pos); r != '+' && r != '-' {
							goto fail10
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						goto ok11
					fail10:
						node.Kids = node.Kids[:nkids8]
						pos = pos9
					ok11:
					}
					// [0-9]+
					// [0-9]
					if r, w := _next(parser, pos); r < '0' || r > '9' {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					for {
						nkids12 := len(node.Kids)
						pos13 := pos
						// [0-9]
						if r, w := _next(parser, pos); r < '0' || r > '9' {
							goto fail15
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail15:
						node.Kids = node.Kids[:nkids12]
						pos = pos13
						break
					}
					// "."
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
						goto fail
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					// [0-9]+
					// [0-9]
					if r, w := _next(parser, pos); r < '0' || r > '9' {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					for {
						nkids16 := len(node.Kids)
						pos17 := pos
						// [0-9]
						if r, w := _next(parser, pos); r < '0' || r > '9' {
							goto fail19
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail19:
						node.Kids = node.Kids[:nkids16]
						pos = pos17
						break
					}
					// ([eE] [+\-]? [0-9]+)?
					{
						nkids20 := len(node.Kids)
						pos21 := pos
						// ([eE] [+\-]? [0-9]+)
						{
							nkids23 := len(node.Kids)
							pos024 := pos
							// [eE] [+\-]? [0-9]+
							// [eE]
							if r, w := _next(parser, pos); r != 'e' && r != 'E' {
								goto fail22
							} else {
								node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
								pos += w
							}
							// [+\-]?
							{
								nkids26 := len(node.Kids)
								pos27 := pos
								// [+\-]
								if r, w := _next(parser, pos); r != '+' && r != '-' {
									goto fail28
								} else {
									node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
									pos += w
								}
								goto ok29
							fail28:
								node.Kids = node.Kids[:nkids26]
								pos = pos27
							ok29:
							}
							// [0-9]+
							// [0-9]
							if r, w := _next(parser, pos); r < '0' || r > '9' {
								goto fail22
							} else {
								node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
								pos += w
							}
							for {
								nkids30 := len(node.Kids)
								pos31 := pos
								// [0-9]
								if r, w := _next(parser, pos); r < '0' || r > '9' {
									goto fail33
								} else {
									node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
									pos += w
								}
								continue
							fail33:
								node.Kids = node.Kids[:nkids30]
								pos = pos31
								break
							}
							sub := _sub(parser, pos024, pos, node.Kids[nkids23:])
							node.Kids = append(node.Kids[:nkids23], sub)
						}
						goto ok34
					fail22:
						node.Kids = node.Kids[:nkids20]
						pos = pos21
					ok34:
					}
					sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
					node.Kids = append(node.Kids[:nkids5], sub)
				}
				labels[0] = parser.text[pos4:pos]
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[1] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	var label1 Float
	var label0 string
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
					start, end int, text string) Float {
					return Float{location: loc(parser, start, end), Text: text}
				}(
					start3, pos, label0)
			}
			labels[1] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, text string, tok Float) Expr {
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

func _RuneNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [3]string
	use(labels)
	dp := parser.deltaPos[start][_Rune]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Rune}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Rune"}
	// action
	// _ tok:(text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// tok:(text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
	{
		pos1 := pos
		// (text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\']) {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// text:([\'] !"\n" data:(Esc/"\\'"/[^\']) [\'])
			{
				pos4 := pos
				// ([\'] !"\n" data:(Esc/"\\'"/[^\']) [\'])
				{
					nkids5 := len(node.Kids)
					pos06 := pos
					// [\'] !"\n" data:(Esc/"\\'"/[^\']) [\']
					// [\']
					if r, w := _next(parser, pos); r != '\'' {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					// !"\n"
					{
						pos9 := pos
						nkids10 := len(node.Kids)
						// "\n"
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
							goto ok8
						}
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
						pos++
						pos = pos9
						node.Kids = node.Kids[:nkids10]
						goto fail
					ok8:
						pos = pos9
						node.Kids = node.Kids[:nkids10]
					}
					// data:(Esc/"\\'"/[^\'])
					{
						pos12 := pos
						// (Esc/"\\'"/[^\'])
						{
							nkids13 := len(node.Kids)
							pos014 := pos
							// Esc/"\\'"/[^\']
							{
								pos18 := pos
								nkids16 := len(node.Kids)
								// Esc
								if !_node(parser, _EscNode, node, &pos) {
									goto fail19
								}
								goto ok15
							fail19:
								node.Kids = node.Kids[:nkids16]
								pos = pos18
								// "\\'"
								if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\'" {
									goto fail20
								}
								node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
								pos += 2
								goto ok15
							fail20:
								node.Kids = node.Kids[:nkids16]
								pos = pos18
								// [^\']
								if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '\'' {
									goto fail21
								} else {
									node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
									pos += w
								}
								goto ok15
							fail21:
								node.Kids = node.Kids[:nkids16]
								pos = pos18
								goto fail
							ok15:
							}
							sub := _sub(parser, pos014, pos, node.Kids[nkids13:])
							node.Kids = append(node.Kids[:nkids13], sub)
						}
						labels[0] = parser.text[pos12:pos]
					}
					// [\']
					if r, w := _next(parser, pos); r != '\'' {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
					node.Kids = append(node.Kids[:nkids5], sub)
				}
				labels[1] = parser.text[pos4:pos]
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[2] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	var label2 Rune
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
					start, end int, data string, text string) Rune {
					r, w := utf8.DecodeRuneInString(data)
					if w != len(data) {
						panic("impossible")
					}
					return Rune{location: loc(parser, start, end), Text: text, Rune: r}
				}(
					start3, pos, label0, label1)
			}
			labels[2] = parser.text[pos2:pos]
		}
		node = func(
			start, end int, data string, text string, tok Rune) Expr {
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

func _StringNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [6]string
	use(labels)
	dp := parser.deltaPos[start][_String]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _String}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "String"}
	// _ tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…}) {…}/_ tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…}) {…}
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// action
		// _ tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail4
		}
		// tok0:(text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
		{
			pos6 := pos
			// (text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]) {…})
			{
				nkids7 := len(node.Kids)
				pos08 := pos
				// action
				// text0:(["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["])
				{
					pos9 := pos
					// (["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["])
					{
						nkids10 := len(node.Kids)
						pos011 := pos
						// ["] data0:(!"\n" (Esc/"\\\""/[^"]))* ["]
						// ["]
						if r, w := _next(parser, pos); r != '"' {
							goto fail4
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						// data0:(!"\n" (Esc/"\\\""/[^"]))*
						{
							pos13 := pos
							// (!"\n" (Esc/"\\\""/[^"]))*
							for {
								nkids14 := len(node.Kids)
								pos15 := pos
								// (!"\n" (Esc/"\\\""/[^"]))
								{
									nkids18 := len(node.Kids)
									pos019 := pos
									// !"\n" (Esc/"\\\""/[^"])
									// !"\n"
									{
										pos22 := pos
										nkids23 := len(node.Kids)
										// "\n"
										if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
											goto ok21
										}
										node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
										pos++
										pos = pos22
										node.Kids = node.Kids[:nkids23]
										goto fail17
									ok21:
										pos = pos22
										node.Kids = node.Kids[:nkids23]
									}
									// (Esc/"\\\""/[^"])
									{
										nkids25 := len(node.Kids)
										pos026 := pos
										// Esc/"\\\""/[^"]
										{
											pos30 := pos
											nkids28 := len(node.Kids)
											// Esc
											if !_node(parser, _EscNode, node, &pos) {
												goto fail31
											}
											goto ok27
										fail31:
											node.Kids = node.Kids[:nkids28]
											pos = pos30
											// "\\\""
											if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\"" {
												goto fail32
											}
											node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
											pos += 2
											goto ok27
										fail32:
											node.Kids = node.Kids[:nkids28]
											pos = pos30
											// [^"]
											if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '"' {
												goto fail33
											} else {
												node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
												pos += w
											}
											goto ok27
										fail33:
											node.Kids = node.Kids[:nkids28]
											pos = pos30
											goto fail17
										ok27:
										}
										sub := _sub(parser, pos026, pos, node.Kids[nkids25:])
										node.Kids = append(node.Kids[:nkids25], sub)
									}
									sub := _sub(parser, pos019, pos, node.Kids[nkids18:])
									node.Kids = append(node.Kids[:nkids18], sub)
								}
								continue
							fail17:
								node.Kids = node.Kids[:nkids14]
								pos = pos15
								break
							}
							labels[0] = parser.text[pos13:pos]
						}
						// ["]
						if r, w := _next(parser, pos); r != '"' {
							goto fail4
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
						node.Kids = append(node.Kids[:nkids10], sub)
					}
					labels[1] = parser.text[pos9:pos]
				}
				sub := _sub(parser, pos08, pos, node.Kids[nkids7:])
				node.Kids = append(node.Kids[:nkids7], sub)
			}
			labels[2] = parser.text[pos6:pos]
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// _ tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…})
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail34
		}
		// tok1:(text1:([`] data1:("\\`"/[^`])* [`]) {…})
		{
			pos36 := pos
			// (text1:([`] data1:("\\`"/[^`])* [`]) {…})
			{
				nkids37 := len(node.Kids)
				pos038 := pos
				// action
				// text1:([`] data1:("\\`"/[^`])* [`])
				{
					pos39 := pos
					// ([`] data1:("\\`"/[^`])* [`])
					{
						nkids40 := len(node.Kids)
						pos041 := pos
						// [`] data1:("\\`"/[^`])* [`]
						// [`]
						if r, w := _next(parser, pos); r != '`' {
							goto fail34
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						// data1:("\\`"/[^`])*
						{
							pos43 := pos
							// ("\\`"/[^`])*
							for {
								nkids44 := len(node.Kids)
								pos45 := pos
								// ("\\`"/[^`])
								{
									nkids48 := len(node.Kids)
									pos049 := pos
									// "\\`"/[^`]
									{
										pos53 := pos
										nkids51 := len(node.Kids)
										// "\\`"
										if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\`" {
											goto fail54
										}
										node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
										pos += 2
										goto ok50
									fail54:
										node.Kids = node.Kids[:nkids51]
										pos = pos53
										// [^`]
										if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '`' {
											goto fail55
										} else {
											node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
											pos += w
										}
										goto ok50
									fail55:
										node.Kids = node.Kids[:nkids51]
										pos = pos53
										goto fail47
									ok50:
									}
									sub := _sub(parser, pos049, pos, node.Kids[nkids48:])
									node.Kids = append(node.Kids[:nkids48], sub)
								}
								continue
							fail47:
								node.Kids = node.Kids[:nkids44]
								pos = pos45
								break
							}
							labels[3] = parser.text[pos43:pos]
						}
						// [`]
						if r, w := _next(parser, pos); r != '`' {
							goto fail34
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						sub := _sub(parser, pos041, pos, node.Kids[nkids40:])
						node.Kids = append(node.Kids[:nkids40], sub)
					}
					labels[4] = parser.text[pos39:pos]
				}
				sub := _sub(parser, pos038, pos, node.Kids[nkids37:])
				node.Kids = append(node.Kids[:nkids37], sub)
			}
			labels[5] = parser.text[pos36:pos]
		}
		goto ok0
	fail34:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
						return String{location: loc(parser, start, end), Text: text0, Data: data0}
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
						return String{location: loc(parser, start, end), Text: text1, Data: data1}
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

func _EscNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [3]string
	use(labels)
	dp := parser.deltaPos[start][_Esc]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Esc}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Esc"}
	// "\\n" {…}/"\\t" {…}/"\\b" {…}/"\\\\" {…}/"\\" x0:(X X) {…}/"\\x" x1:(X X X X) {…}/"\\X" x2:(X X X X X X X X) {…}
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// action
		// "\\n"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\n" {
			goto fail4
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// "\\t"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\t" {
			goto fail5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// "\\b"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\b" {
			goto fail6
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		goto ok0
	fail6:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// "\\\\"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\\" {
			goto fail7
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		goto ok0
	fail7:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// "\\" x0:(X X)
		// "\\"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\\" {
			goto fail8
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// x0:(X X)
		{
			pos10 := pos
			// (X X)
			{
				nkids11 := len(node.Kids)
				pos012 := pos
				// X X
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail8
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail8
				}
				sub := _sub(parser, pos012, pos, node.Kids[nkids11:])
				node.Kids = append(node.Kids[:nkids11], sub)
			}
			labels[0] = parser.text[pos10:pos]
		}
		goto ok0
	fail8:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// "\\x" x1:(X X X X)
		// "\\x"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\x" {
			goto fail14
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		// x1:(X X X X)
		{
			pos16 := pos
			// (X X X X)
			{
				nkids17 := len(node.Kids)
				pos018 := pos
				// X X X X
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail14
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail14
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail14
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail14
				}
				sub := _sub(parser, pos018, pos, node.Kids[nkids17:])
				node.Kids = append(node.Kids[:nkids17], sub)
			}
			labels[1] = parser.text[pos16:pos]
		}
		goto ok0
	fail14:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// action
		// "\\X" x2:(X X X X X X X X)
		// "\\X"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\X" {
			goto fail20
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		// x2:(X X X X X X X X)
		{
			pos22 := pos
			// (X X X X X X X X)
			{
				nkids23 := len(node.Kids)
				pos024 := pos
				// X X X X X X X X
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail20
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail20
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail20
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail20
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail20
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail20
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail20
				}
				// X
				if !_node(parser, _XNode, node, &pos) {
					goto fail20
				}
				sub := _sub(parser, pos024, pos, node.Kids[nkids23:])
				node.Kids = append(node.Kids[:nkids23], sub)
			}
			labels[2] = parser.text[pos22:pos]
		}
		goto ok0
	fail20:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _XNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_X]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _X}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "X"}
	// [a-fA-F0-9]
	if r, w := _next(parser, pos); (r < 'a' || r > 'f') && (r < 'A' || r > 'F') && (r < '0' || r > '9') {
		goto fail
	} else {
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
		pos += w
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _OpNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_Op]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Op}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Op"}
	// action
	// _ !"//" !"/*" tok:(text:([!%&*+\-/<=>?@\\|~]+) {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// !"//"
	{
		pos2 := pos
		nkids3 := len(node.Kids)
		// "//"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
			goto ok1
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		pos = pos2
		node.Kids = node.Kids[:nkids3]
		goto fail
	ok1:
		pos = pos2
		node.Kids = node.Kids[:nkids3]
	}
	// !"/*"
	{
		pos6 := pos
		nkids7 := len(node.Kids)
		// "/*"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
			goto ok5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		pos = pos6
		node.Kids = node.Kids[:nkids7]
		goto fail
	ok5:
		pos = pos6
		node.Kids = node.Kids[:nkids7]
	}
	// tok:(text:([!%&*+\-/<=>?@\\|~]+) {…})
	{
		pos9 := pos
		// (text:([!%&*+\-/<=>?@\\|~]+) {…})
		{
			nkids10 := len(node.Kids)
			pos011 := pos
			// action
			// text:([!%&*+\-/<=>?@\\|~]+)
			{
				pos12 := pos
				// ([!%&*+\-/<=>?@\\|~]+)
				{
					nkids13 := len(node.Kids)
					pos014 := pos
					// [!%&*+\-/<=>?@\\|~]+
					// [!%&*+\-/<=>?@\\|~]
					if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					for {
						nkids15 := len(node.Kids)
						pos16 := pos
						// [!%&*+\-/<=>?@\\|~]
						if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
							goto fail18
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail18:
						node.Kids = node.Kids[:nkids15]
						pos = pos16
						break
					}
					sub := _sub(parser, pos014, pos, node.Kids[nkids13:])
					node.Kids = append(node.Kids[:nkids13], sub)
				}
				labels[0] = parser.text[pos12:pos]
			}
			sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
			node.Kids = append(node.Kids[:nkids10], sub)
		}
		labels[1] = parser.text[pos9:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _TypeOpNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_TypeOp]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeOp}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "TypeOp"}
	// action
	// _ tok:(text:([!&?]+) {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// tok:(text:([!&?]+) {…})
	{
		pos1 := pos
		// (text:([!&?]+) {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// text:([!&?]+)
			{
				pos4 := pos
				// ([!&?]+)
				{
					nkids5 := len(node.Kids)
					pos06 := pos
					// [!&?]+
					// [!&?]
					if r, w := _next(parser, pos); r != '!' && r != '&' && r != '?' {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					for {
						nkids7 := len(node.Kids)
						pos8 := pos
						// [!&?]
						if r, w := _next(parser, pos); r != '!' && r != '&' && r != '?' {
							goto fail10
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail10:
						node.Kids = node.Kids[:nkids7]
						pos = pos8
						break
					}
					sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
					node.Kids = append(node.Kids[:nkids5], sub)
				}
				labels[0] = parser.text[pos4:pos]
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[1] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _ModNameNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_ModName]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _ModName}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "ModName"}
	// action
	// _ tok:(text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// tok:(text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
	{
		pos1 := pos
		// (text:("#" [_a-zA-Z] [_a-zA-Z0-9]*) {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// text:("#" [_a-zA-Z] [_a-zA-Z0-9]*)
			{
				pos4 := pos
				// ("#" [_a-zA-Z] [_a-zA-Z0-9]*)
				{
					nkids5 := len(node.Kids)
					pos06 := pos
					// "#" [_a-zA-Z] [_a-zA-Z0-9]*
					// "#"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "#" {
						goto fail
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					// [_a-zA-Z]
					if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					// [_a-zA-Z0-9]*
					for {
						nkids8 := len(node.Kids)
						pos9 := pos
						// [_a-zA-Z0-9]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
							goto fail11
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail11:
						node.Kids = node.Kids[:nkids8]
						pos = pos9
						break
					}
					sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
					node.Kids = append(node.Kids[:nkids5], sub)
				}
				labels[0] = parser.text[pos4:pos]
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[1] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _IdentCNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_IdentC]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _IdentC}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "IdentC"}
	// action
	// _ !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// !TypeVar
	{
		pos2 := pos
		nkids3 := len(node.Kids)
		// TypeVar
		if !_node(parser, _TypeVarNode, node, &pos) {
			goto ok1
		}
		pos = pos2
		node.Kids = node.Kids[:nkids3]
		goto fail
	ok1:
		pos = pos2
		node.Kids = node.Kids[:nkids3]
	}
	// !"import"
	{
		pos6 := pos
		nkids7 := len(node.Kids)
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			goto ok5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+6))
		pos += 6
		pos = pos6
		node.Kids = node.Kids[:nkids7]
		goto fail
	ok5:
		pos = pos6
		node.Kids = node.Kids[:nkids7]
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]* ":") {…})
		{
			nkids10 := len(node.Kids)
			pos011 := pos
			// action
			// text:([_a-zA-Z] [_a-zA-Z0-9]* ":")
			{
				pos12 := pos
				// ([_a-zA-Z] [_a-zA-Z0-9]* ":")
				{
					nkids13 := len(node.Kids)
					pos014 := pos
					// [_a-zA-Z] [_a-zA-Z0-9]* ":"
					// [_a-zA-Z]
					if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					// [_a-zA-Z0-9]*
					for {
						nkids16 := len(node.Kids)
						pos17 := pos
						// [_a-zA-Z0-9]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
							goto fail19
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail19:
						node.Kids = node.Kids[:nkids16]
						pos = pos17
						break
					}
					// ":"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
						goto fail
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					sub := _sub(parser, pos014, pos, node.Kids[nkids13:])
					node.Kids = append(node.Kids[:nkids13], sub)
				}
				labels[0] = parser.text[pos12:pos]
			}
			sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
			node.Kids = append(node.Kids[:nkids10], sub)
		}
		labels[1] = parser.text[pos9:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	var label1 Ident
	var label0 string
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

func _CIdentNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_CIdent]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _CIdent}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "CIdent"}
	// action
	// _ ":" !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// ":"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
	pos++
	// !TypeVar
	{
		pos2 := pos
		nkids3 := len(node.Kids)
		// TypeVar
		if !_node(parser, _TypeVarNode, node, &pos) {
			goto ok1
		}
		pos = pos2
		node.Kids = node.Kids[:nkids3]
		goto fail
	ok1:
		pos = pos2
		node.Kids = node.Kids[:nkids3]
	}
	// !"import"
	{
		pos6 := pos
		nkids7 := len(node.Kids)
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			goto ok5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+6))
		pos += 6
		pos = pos6
		node.Kids = node.Kids[:nkids7]
		goto fail
	ok5:
		pos = pos6
		node.Kids = node.Kids[:nkids7]
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]*) {…})
		{
			nkids10 := len(node.Kids)
			pos011 := pos
			// action
			// text:([_a-zA-Z] [_a-zA-Z0-9]*)
			{
				pos12 := pos
				// ([_a-zA-Z] [_a-zA-Z0-9]*)
				{
					nkids13 := len(node.Kids)
					pos014 := pos
					// [_a-zA-Z] [_a-zA-Z0-9]*
					// [_a-zA-Z]
					if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					// [_a-zA-Z0-9]*
					for {
						nkids16 := len(node.Kids)
						pos17 := pos
						// [_a-zA-Z0-9]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
							goto fail19
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail19:
						node.Kids = node.Kids[:nkids16]
						pos = pos17
						break
					}
					sub := _sub(parser, pos014, pos, node.Kids[nkids13:])
					node.Kids = append(node.Kids[:nkids13], sub)
				}
				labels[0] = parser.text[pos12:pos]
			}
			sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
			node.Kids = append(node.Kids[:nkids10], sub)
		}
		labels[1] = parser.text[pos9:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _IdentNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_Ident]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ident}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Ident"}
	// action
	// _ !TypeVar !"import" tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// !TypeVar
	{
		pos2 := pos
		nkids3 := len(node.Kids)
		// TypeVar
		if !_node(parser, _TypeVarNode, node, &pos) {
			goto ok1
		}
		pos = pos2
		node.Kids = node.Kids[:nkids3]
		goto fail
	ok1:
		pos = pos2
		node.Kids = node.Kids[:nkids3]
	}
	// !"import"
	{
		pos6 := pos
		nkids7 := len(node.Kids)
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			goto ok5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+6))
		pos += 6
		pos = pos6
		node.Kids = node.Kids[:nkids7]
		goto fail
	ok5:
		pos = pos6
		node.Kids = node.Kids[:nkids7]
	}
	// tok:(text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
	{
		pos9 := pos
		// (text:([_a-zA-Z] [_a-zA-Z0-9]*) !":" {…})
		{
			nkids10 := len(node.Kids)
			pos011 := pos
			// action
			// text:([_a-zA-Z] [_a-zA-Z0-9]*) !":"
			// text:([_a-zA-Z] [_a-zA-Z0-9]*)
			{
				pos13 := pos
				// ([_a-zA-Z] [_a-zA-Z0-9]*)
				{
					nkids14 := len(node.Kids)
					pos015 := pos
					// [_a-zA-Z] [_a-zA-Z0-9]*
					// [_a-zA-Z]
					if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					// [_a-zA-Z0-9]*
					for {
						nkids17 := len(node.Kids)
						pos18 := pos
						// [_a-zA-Z0-9]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
							goto fail20
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						continue
					fail20:
						node.Kids = node.Kids[:nkids17]
						pos = pos18
						break
					}
					sub := _sub(parser, pos015, pos, node.Kids[nkids14:])
					node.Kids = append(node.Kids[:nkids14], sub)
				}
				labels[0] = parser.text[pos13:pos]
			}
			// !":"
			{
				pos22 := pos
				nkids23 := len(node.Kids)
				// ":"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
					goto ok21
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				pos = pos22
				node.Kids = node.Kids[:nkids23]
				goto fail
			ok21:
				pos = pos22
				node.Kids = node.Kids[:nkids23]
			}
			sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
			node.Kids = append(node.Kids[:nkids10], sub)
		}
		labels[1] = parser.text[pos9:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	var label1 Ident
	var label0 string
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

func _TypeVarNode(parser *_Parser, start int) (int, *peg.Node) {
	var labels [2]string
	use(labels)
	dp := parser.deltaPos[start][_TypeVar]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeVar}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "TypeVar"}
	// action
	// _ tok:(text:([A-Z] ![_a-zA-Z0-9]) {…})
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// tok:(text:([A-Z] ![_a-zA-Z0-9]) {…})
	{
		pos1 := pos
		// (text:([A-Z] ![_a-zA-Z0-9]) {…})
		{
			nkids2 := len(node.Kids)
			pos03 := pos
			// action
			// text:([A-Z] ![_a-zA-Z0-9])
			{
				pos4 := pos
				// ([A-Z] ![_a-zA-Z0-9])
				{
					nkids5 := len(node.Kids)
					pos06 := pos
					// [A-Z] ![_a-zA-Z0-9]
					// [A-Z]
					if r, w := _next(parser, pos); r < 'A' || r > 'Z' {
						goto fail
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					// ![_a-zA-Z0-9]
					{
						pos9 := pos
						nkids10 := len(node.Kids)
						// [_a-zA-Z0-9]
						if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
							goto ok8
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						pos = pos9
						node.Kids = node.Kids[:nkids10]
						goto fail
					ok8:
						pos = pos9
						node.Kids = node.Kids[:nkids10]
					}
					sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
					node.Kids = append(node.Kids[:nkids5], sub)
				}
				labels[0] = parser.text[pos4:pos]
			}
			sub := _sub(parser, pos03, pos, node.Kids[nkids2:])
			node.Kids = append(node.Kids[:nkids2], sub)
		}
		labels[1] = parser.text[pos1:pos]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func __Node(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][__]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: __}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "_"}
	// (Space/Cmnt)*
	for {
		nkids0 := len(node.Kids)
		pos1 := pos
		// (Space/Cmnt)
		{
			nkids4 := len(node.Kids)
			pos05 := pos
			// Space/Cmnt
			{
				pos9 := pos
				nkids7 := len(node.Kids)
				// Space
				if !_node(parser, _SpaceNode, node, &pos) {
					goto fail10
				}
				goto ok6
			fail10:
				node.Kids = node.Kids[:nkids7]
				pos = pos9
				// Cmnt
				if !_node(parser, _CmntNode, node, &pos) {
					goto fail11
				}
				goto ok6
			fail11:
				node.Kids = node.Kids[:nkids7]
				pos = pos9
				goto fail3
			ok6:
			}
			sub := _sub(parser, pos05, pos, node.Kids[nkids4:])
			node.Kids = append(node.Kids[:nkids4], sub)
		}
		continue
	fail3:
		node.Kids = node.Kids[:nkids0]
		pos = pos1
		break
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
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

func _CmntNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_Cmnt]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Cmnt}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Cmnt"}
	// "//" (!"\n" .)*/"/*" (!"*/" .)* "*/"
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// "//" (!"\n" .)*
		// "//"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
			goto fail4
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		// (!"\n" .)*
		for {
			nkids6 := len(node.Kids)
			pos7 := pos
			// (!"\n" .)
			{
				nkids10 := len(node.Kids)
				pos011 := pos
				// !"\n" .
				// !"\n"
				{
					pos14 := pos
					nkids15 := len(node.Kids)
					// "\n"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
						goto ok13
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					pos = pos14
					node.Kids = node.Kids[:nkids15]
					goto fail9
				ok13:
					pos = pos14
					node.Kids = node.Kids[:nkids15]
				}
				// .
				if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
					goto fail9
				} else {
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
					pos += w
				}
				sub := _sub(parser, pos011, pos, node.Kids[nkids10:])
				node.Kids = append(node.Kids[:nkids10], sub)
			}
			continue
		fail9:
			node.Kids = node.Kids[:nkids6]
			pos = pos7
			break
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// "/*" (!"*/" .)* "*/"
		// "/*"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
			goto fail17
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		// (!"*/" .)*
		for {
			nkids19 := len(node.Kids)
			pos20 := pos
			// (!"*/" .)
			{
				nkids23 := len(node.Kids)
				pos024 := pos
				// !"*/" .
				// !"*/"
				{
					pos27 := pos
					nkids28 := len(node.Kids)
					// "*/"
					if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "*/" {
						goto ok26
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
					pos += 2
					pos = pos27
					node.Kids = node.Kids[:nkids28]
					goto fail22
				ok26:
					pos = pos27
					node.Kids = node.Kids[:nkids28]
				}
				// .
				if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
					goto fail22
				} else {
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
					pos += w
				}
				sub := _sub(parser, pos024, pos, node.Kids[nkids23:])
				node.Kids = append(node.Kids[:nkids23], sub)
			}
			continue
		fail22:
			node.Kids = node.Kids[:nkids19]
			pos = pos20
			break
		}
		// "*/"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "*/" {
			goto fail17
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		goto ok0
	fail17:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _SpaceNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_Space]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Space}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Space"}
	// " "/"\t"/"\n"
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// " "
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != " " {
			goto fail4
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// "\t"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\t" {
			goto fail5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// "\n"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
			goto fail6
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail6:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		goto fail
	ok0:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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

func _EOFNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_EOF]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _EOF}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "EOF"}
	// !.
	{
		pos1 := pos
		nkids2 := len(node.Kids)
		// .
		if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' {
			goto ok0
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		pos = pos1
		node.Kids = node.Kids[:nkids2]
		goto fail
	ok0:
		pos = pos1
		node.Kids = node.Kids[:nkids2]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
