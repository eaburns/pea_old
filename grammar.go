package main

import "github.com/eaburns/peggy/peg"

const (
	_File      int = 0
	_Def       int = 1
	_Fun       int = 2
	_FunSig    int = 3
	_Ret       int = 4
	_Var       int = 5
	_TypeSig   int = 6
	_TypeParms int = 7
	_TypeParm  int = 8
	_TypeName  int = 9
	_Type      int = 10
	_Case      int = 11
	_MethSig   int = 12
	_Stmts     int = 13
	_Stmt      int = 14
	_Return    int = 15
	_Assign    int = 16
	_Expr      int = 17
	_Cascade   int = 18
	_Call      int = 19
	_Unary     int = 20
	_Binary    int = 21
	_BinMsg    int = 22
	_Nary      int = 23
	_NaryMsg   int = 24
	_Primary   int = 25
	_Ctor      int = 26
	_Block     int = 27
	_Int       int = 28
	_Float     int = 29
	_Rune      int = 30
	_String    int = 31
	_Esc       int = 32
	_X         int = 33
	_Op        int = 34
	_ModName   int = 35
	_IdentC    int = 36
	_CIdent    int = 37
	_Ident     int = 38
	_TypeVar   int = 39
	__         int = 40
	_Cmnt      int = 41
	_Space     int = 42
	_EOF       int = 43

	_N int = 44
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
	if dp, de, ok := _memo(parser, _File, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Def* _ EOF
	// Def*
	for {
		pos2 := pos
		// Def
		if !_accept(parser, _DefAccepts, &pos, &perr) {
			goto fail4
		}
		continue
	fail4:
		pos = pos2
		break
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
	// Def* _ EOF
	// Def*
	for {
		nkids1 := len(node.Kids)
		pos2 := pos
		// Def
		if !_node(parser, _DefNode, node, &pos) {
			goto fail4
		}
		continue
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
		break
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
	pos, failure := _failMemo(parser, _File, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "File",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _File}
	// Def* _ EOF
	// Def*
	for {
		pos2 := pos
		// Def
		if !_fail(parser, _DefFail, errPos, failure, &pos) {
			goto fail4
		}
		continue
	fail4:
		pos = pos2
		break
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

func _FileAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_File]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _File}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Def* _ EOF
	{
		var node0 string
		// Def*
		for {
			pos2 := pos
			var node3 string
			// Def
			if p, n := _DefAction(parser, pos); n == nil {
				goto fail4
			} else {
				node3 = *n
				pos = p
			}
			node0 += node3
			continue
		fail4:
			pos = pos2
			break
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// EOF
		if p, n := _EOFAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
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
	// ModName* (_ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")"))
	// ModName*
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
	// (_ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")"))
	// _ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")")
	{
		pos8 := pos
		// _ "import" String
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail9
		}
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			perr = _max(perr, pos)
			goto fail9
		}
		pos += 6
		// String
		if !_accept(parser, _StringAccepts, &pos, &perr) {
			goto fail9
		}
		goto ok5
	fail9:
		pos = pos8
		// _ "(" Def+ _ ")"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail11
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail11
		}
		pos++
		// Def+
		// Def
		if !_accept(parser, _DefAccepts, &pos, &perr) {
			goto fail11
		}
		for {
			pos14 := pos
			// Def
			if !_accept(parser, _DefAccepts, &pos, &perr) {
				goto fail16
			}
			continue
		fail16:
			pos = pos14
			break
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail11
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			perr = _max(perr, pos)
			goto fail11
		}
		pos++
		goto ok5
	fail11:
		pos = pos8
		// Fun
		if !_accept(parser, _FunAccepts, &pos, &perr) {
			goto fail17
		}
		goto ok5
	fail17:
		pos = pos8
		// Var
		if !_accept(parser, _VarAccepts, &pos, &perr) {
			goto fail18
		}
		goto ok5
	fail18:
		pos = pos8
		// TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")")
		// TypeSig
		if !_accept(parser, _TypeSigAccepts, &pos, &perr) {
			goto fail19
		}
		// (Type/Fun/_ "(" (Type/Fun)+ _ ")")
		// Type/Fun/_ "(" (Type/Fun)+ _ ")"
		{
			pos24 := pos
			// Type
			if !_accept(parser, _TypeAccepts, &pos, &perr) {
				goto fail25
			}
			goto ok21
		fail25:
			pos = pos24
			// Fun
			if !_accept(parser, _FunAccepts, &pos, &perr) {
				goto fail26
			}
			goto ok21
		fail26:
			pos = pos24
			// _ "(" (Type/Fun)+ _ ")"
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail27
			}
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				perr = _max(perr, pos)
				goto fail27
			}
			pos++
			// (Type/Fun)+
			// (Type/Fun)
			// Type/Fun
			{
				pos36 := pos
				// Type
				if !_accept(parser, _TypeAccepts, &pos, &perr) {
					goto fail37
				}
				goto ok33
			fail37:
				pos = pos36
				// Fun
				if !_accept(parser, _FunAccepts, &pos, &perr) {
					goto fail38
				}
				goto ok33
			fail38:
				pos = pos36
				goto fail27
			ok33:
			}
			for {
				pos30 := pos
				// (Type/Fun)
				// Type/Fun
				{
					pos42 := pos
					// Type
					if !_accept(parser, _TypeAccepts, &pos, &perr) {
						goto fail43
					}
					goto ok39
				fail43:
					pos = pos42
					// Fun
					if !_accept(parser, _FunAccepts, &pos, &perr) {
						goto fail44
					}
					goto ok39
				fail44:
					pos = pos42
					goto fail32
				ok39:
				}
				continue
			fail32:
				pos = pos30
				break
			}
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail27
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				perr = _max(perr, pos)
				goto fail27
			}
			pos++
			goto ok21
		fail27:
			pos = pos24
			goto fail19
		ok21:
		}
		goto ok5
	fail19:
		pos = pos8
		goto fail
	ok5:
	}
	return _memoize(parser, _Def, start, pos, perr)
fail:
	return _memoize(parser, _Def, start, -1, perr)
}

func _DefNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// ModName* (_ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")"))
	// ModName*
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
	// (_ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")"))
	{
		nkids5 := len(node.Kids)
		pos06 := pos
		// _ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")")
		{
			pos10 := pos
			nkids8 := len(node.Kids)
			// _ "import" String
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail11
			}
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				goto fail11
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+6))
			pos += 6
			// String
			if !_node(parser, _StringNode, node, &pos) {
				goto fail11
			}
			goto ok7
		fail11:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			// _ "(" Def+ _ ")"
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
			// Def+
			// Def
			if !_node(parser, _DefNode, node, &pos) {
				goto fail13
			}
			for {
				nkids15 := len(node.Kids)
				pos16 := pos
				// Def
				if !_node(parser, _DefNode, node, &pos) {
					goto fail18
				}
				continue
			fail18:
				node.Kids = node.Kids[:nkids15]
				pos = pos16
				break
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
			goto ok7
		fail13:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			// Fun
			if !_node(parser, _FunNode, node, &pos) {
				goto fail19
			}
			goto ok7
		fail19:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			// Var
			if !_node(parser, _VarNode, node, &pos) {
				goto fail20
			}
			goto ok7
		fail20:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			// TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")")
			// TypeSig
			if !_node(parser, _TypeSigNode, node, &pos) {
				goto fail21
			}
			// (Type/Fun/_ "(" (Type/Fun)+ _ ")")
			{
				nkids23 := len(node.Kids)
				pos024 := pos
				// Type/Fun/_ "(" (Type/Fun)+ _ ")"
				{
					pos28 := pos
					nkids26 := len(node.Kids)
					// Type
					if !_node(parser, _TypeNode, node, &pos) {
						goto fail29
					}
					goto ok25
				fail29:
					node.Kids = node.Kids[:nkids26]
					pos = pos28
					// Fun
					if !_node(parser, _FunNode, node, &pos) {
						goto fail30
					}
					goto ok25
				fail30:
					node.Kids = node.Kids[:nkids26]
					pos = pos28
					// _ "(" (Type/Fun)+ _ ")"
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail31
					}
					// "("
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
						goto fail31
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					// (Type/Fun)+
					// (Type/Fun)
					{
						nkids37 := len(node.Kids)
						pos038 := pos
						// Type/Fun
						{
							pos42 := pos
							nkids40 := len(node.Kids)
							// Type
							if !_node(parser, _TypeNode, node, &pos) {
								goto fail43
							}
							goto ok39
						fail43:
							node.Kids = node.Kids[:nkids40]
							pos = pos42
							// Fun
							if !_node(parser, _FunNode, node, &pos) {
								goto fail44
							}
							goto ok39
						fail44:
							node.Kids = node.Kids[:nkids40]
							pos = pos42
							goto fail31
						ok39:
						}
						sub := _sub(parser, pos038, pos, node.Kids[nkids37:])
						node.Kids = append(node.Kids[:nkids37], sub)
					}
					for {
						nkids33 := len(node.Kids)
						pos34 := pos
						// (Type/Fun)
						{
							nkids45 := len(node.Kids)
							pos046 := pos
							// Type/Fun
							{
								pos50 := pos
								nkids48 := len(node.Kids)
								// Type
								if !_node(parser, _TypeNode, node, &pos) {
									goto fail51
								}
								goto ok47
							fail51:
								node.Kids = node.Kids[:nkids48]
								pos = pos50
								// Fun
								if !_node(parser, _FunNode, node, &pos) {
									goto fail52
								}
								goto ok47
							fail52:
								node.Kids = node.Kids[:nkids48]
								pos = pos50
								goto fail36
							ok47:
							}
							sub := _sub(parser, pos046, pos, node.Kids[nkids45:])
							node.Kids = append(node.Kids[:nkids45], sub)
						}
						continue
					fail36:
						node.Kids = node.Kids[:nkids33]
						pos = pos34
						break
					}
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail31
					}
					// ")"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
						goto fail31
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					goto ok25
				fail31:
					node.Kids = node.Kids[:nkids26]
					pos = pos28
					goto fail21
				ok25:
				}
				sub := _sub(parser, pos024, pos, node.Kids[nkids23:])
				node.Kids = append(node.Kids[:nkids23], sub)
			}
			goto ok7
		fail21:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			goto fail
		ok7:
		}
		sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
		node.Kids = append(node.Kids[:nkids5], sub)
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
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
	// ModName* (_ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")"))
	// ModName*
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
	// (_ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")"))
	// _ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")")
	{
		pos8 := pos
		// _ "import" String
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail9
		}
		// "import"
		if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"import\"",
				})
			}
			goto fail9
		}
		pos += 6
		// String
		if !_fail(parser, _StringFail, errPos, failure, &pos) {
			goto fail9
		}
		goto ok5
	fail9:
		pos = pos8
		// _ "(" Def+ _ ")"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail11
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail11
		}
		pos++
		// Def+
		// Def
		if !_fail(parser, _DefFail, errPos, failure, &pos) {
			goto fail11
		}
		for {
			pos14 := pos
			// Def
			if !_fail(parser, _DefFail, errPos, failure, &pos) {
				goto fail16
			}
			continue
		fail16:
			pos = pos14
			break
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail11
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
				})
			}
			goto fail11
		}
		pos++
		goto ok5
	fail11:
		pos = pos8
		// Fun
		if !_fail(parser, _FunFail, errPos, failure, &pos) {
			goto fail17
		}
		goto ok5
	fail17:
		pos = pos8
		// Var
		if !_fail(parser, _VarFail, errPos, failure, &pos) {
			goto fail18
		}
		goto ok5
	fail18:
		pos = pos8
		// TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")")
		// TypeSig
		if !_fail(parser, _TypeSigFail, errPos, failure, &pos) {
			goto fail19
		}
		// (Type/Fun/_ "(" (Type/Fun)+ _ ")")
		// Type/Fun/_ "(" (Type/Fun)+ _ ")"
		{
			pos24 := pos
			// Type
			if !_fail(parser, _TypeFail, errPos, failure, &pos) {
				goto fail25
			}
			goto ok21
		fail25:
			pos = pos24
			// Fun
			if !_fail(parser, _FunFail, errPos, failure, &pos) {
				goto fail26
			}
			goto ok21
		fail26:
			pos = pos24
			// _ "(" (Type/Fun)+ _ ")"
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail27
			}
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"(\"",
					})
				}
				goto fail27
			}
			pos++
			// (Type/Fun)+
			// (Type/Fun)
			// Type/Fun
			{
				pos36 := pos
				// Type
				if !_fail(parser, _TypeFail, errPos, failure, &pos) {
					goto fail37
				}
				goto ok33
			fail37:
				pos = pos36
				// Fun
				if !_fail(parser, _FunFail, errPos, failure, &pos) {
					goto fail38
				}
				goto ok33
			fail38:
				pos = pos36
				goto fail27
			ok33:
			}
			for {
				pos30 := pos
				// (Type/Fun)
				// Type/Fun
				{
					pos42 := pos
					// Type
					if !_fail(parser, _TypeFail, errPos, failure, &pos) {
						goto fail43
					}
					goto ok39
				fail43:
					pos = pos42
					// Fun
					if !_fail(parser, _FunFail, errPos, failure, &pos) {
						goto fail44
					}
					goto ok39
				fail44:
					pos = pos42
					goto fail32
				ok39:
				}
				continue
			fail32:
				pos = pos30
				break
			}
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail27
			}
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\")\"",
					})
				}
				goto fail27
			}
			pos++
			goto ok21
		fail27:
			pos = pos24
			goto fail19
		ok21:
		}
		goto ok5
	fail19:
		pos = pos8
		goto fail
	ok5:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _DefAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Def]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Def}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// ModName* (_ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")"))
	{
		var node0 string
		// ModName*
		for {
			pos2 := pos
			var node3 string
			// ModName
			if p, n := _ModNameAction(parser, pos); n == nil {
				goto fail4
			} else {
				node3 = *n
				pos = p
			}
			node0 += node3
			continue
		fail4:
			pos = pos2
			break
		}
		node, node0 = node+node0, ""
		// (_ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")"))
		// _ "import" String/_ "(" Def+ _ ")"/Fun/Var/TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")")
		{
			pos8 := pos
			var node7 string
			// _ "import" String
			{
				var node10 string
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail9
				} else {
					node10 = *n
					pos = p
				}
				node0, node10 = node0+node10, ""
				// "import"
				if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
					goto fail9
				}
				node10 = parser.text[pos : pos+6]
				pos += 6
				node0, node10 = node0+node10, ""
				// String
				if p, n := _StringAction(parser, pos); n == nil {
					goto fail9
				} else {
					node10 = *n
					pos = p
				}
				node0, node10 = node0+node10, ""
			}
			goto ok5
		fail9:
			node0 = node7
			pos = pos8
			// _ "(" Def+ _ ")"
			{
				var node12 string
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail11
				} else {
					node12 = *n
					pos = p
				}
				node0, node12 = node0+node12, ""
				// "("
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
					goto fail11
				}
				node12 = parser.text[pos : pos+1]
				pos++
				node0, node12 = node0+node12, ""
				// Def+
				{
					var node15 string
					// Def
					if p, n := _DefAction(parser, pos); n == nil {
						goto fail11
					} else {
						node15 = *n
						pos = p
					}
					node12 += node15
				}
				for {
					pos14 := pos
					var node15 string
					// Def
					if p, n := _DefAction(parser, pos); n == nil {
						goto fail16
					} else {
						node15 = *n
						pos = p
					}
					node12 += node15
					continue
				fail16:
					pos = pos14
					break
				}
				node0, node12 = node0+node12, ""
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail11
				} else {
					node12 = *n
					pos = p
				}
				node0, node12 = node0+node12, ""
				// ")"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
					goto fail11
				}
				node12 = parser.text[pos : pos+1]
				pos++
				node0, node12 = node0+node12, ""
			}
			goto ok5
		fail11:
			node0 = node7
			pos = pos8
			// Fun
			if p, n := _FunAction(parser, pos); n == nil {
				goto fail17
			} else {
				node0 = *n
				pos = p
			}
			goto ok5
		fail17:
			node0 = node7
			pos = pos8
			// Var
			if p, n := _VarAction(parser, pos); n == nil {
				goto fail18
			} else {
				node0 = *n
				pos = p
			}
			goto ok5
		fail18:
			node0 = node7
			pos = pos8
			// TypeSig (Type/Fun/_ "(" (Type/Fun)+ _ ")")
			{
				var node20 string
				// TypeSig
				if p, n := _TypeSigAction(parser, pos); n == nil {
					goto fail19
				} else {
					node20 = *n
					pos = p
				}
				node0, node20 = node0+node20, ""
				// (Type/Fun/_ "(" (Type/Fun)+ _ ")")
				// Type/Fun/_ "(" (Type/Fun)+ _ ")"
				{
					pos24 := pos
					var node23 string
					// Type
					if p, n := _TypeAction(parser, pos); n == nil {
						goto fail25
					} else {
						node20 = *n
						pos = p
					}
					goto ok21
				fail25:
					node20 = node23
					pos = pos24
					// Fun
					if p, n := _FunAction(parser, pos); n == nil {
						goto fail26
					} else {
						node20 = *n
						pos = p
					}
					goto ok21
				fail26:
					node20 = node23
					pos = pos24
					// _ "(" (Type/Fun)+ _ ")"
					{
						var node28 string
						// _
						if p, n := __Action(parser, pos); n == nil {
							goto fail27
						} else {
							node28 = *n
							pos = p
						}
						node20, node28 = node20+node28, ""
						// "("
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
							goto fail27
						}
						node28 = parser.text[pos : pos+1]
						pos++
						node20, node28 = node20+node28, ""
						// (Type/Fun)+
						{
							var node31 string
							// (Type/Fun)
							// Type/Fun
							{
								pos36 := pos
								var node35 string
								// Type
								if p, n := _TypeAction(parser, pos); n == nil {
									goto fail37
								} else {
									node31 = *n
									pos = p
								}
								goto ok33
							fail37:
								node31 = node35
								pos = pos36
								// Fun
								if p, n := _FunAction(parser, pos); n == nil {
									goto fail38
								} else {
									node31 = *n
									pos = p
								}
								goto ok33
							fail38:
								node31 = node35
								pos = pos36
								goto fail27
							ok33:
							}
							node28 += node31
						}
						for {
							pos30 := pos
							var node31 string
							// (Type/Fun)
							// Type/Fun
							{
								pos42 := pos
								var node41 string
								// Type
								if p, n := _TypeAction(parser, pos); n == nil {
									goto fail43
								} else {
									node31 = *n
									pos = p
								}
								goto ok39
							fail43:
								node31 = node41
								pos = pos42
								// Fun
								if p, n := _FunAction(parser, pos); n == nil {
									goto fail44
								} else {
									node31 = *n
									pos = p
								}
								goto ok39
							fail44:
								node31 = node41
								pos = pos42
								goto fail32
							ok39:
							}
							node28 += node31
							continue
						fail32:
							pos = pos30
							break
						}
						node20, node28 = node20+node28, ""
						// _
						if p, n := __Action(parser, pos); n == nil {
							goto fail27
						} else {
							node28 = *n
							pos = p
						}
						node20, node28 = node20+node28, ""
						// ")"
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
							goto fail27
						}
						node28 = parser.text[pos : pos+1]
						pos++
						node20, node28 = node20+node28, ""
					}
					goto ok21
				fail27:
					node20 = node23
					pos = pos24
					goto fail19
				ok21:
				}
				node0, node20 = node0+node20, ""
			}
			goto ok5
		fail19:
			node0 = node7
			pos = pos8
			goto fail
		ok5:
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FunAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Fun, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ "[" FunSig _ "|" Stmts _ "]"
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
	// FunSig
	if !_accept(parser, _FunSigAccepts, &pos, &perr) {
		goto fail
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
	// Stmts
	if !_accept(parser, _StmtsAccepts, &pos, &perr) {
		goto fail
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
	return _memoize(parser, _Fun, start, pos, perr)
fail:
	return _memoize(parser, _Fun, start, -1, perr)
}

func _FunNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ "[" FunSig _ "|" Stmts _ "]"
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
	// FunSig
	if !_node(parser, _FunSigNode, node, &pos) {
		goto fail
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
	// Stmts
	if !_node(parser, _StmtsNode, node, &pos) {
		goto fail
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

func _FunFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Fun, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Fun",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Fun}
	// _ "[" FunSig _ "|" Stmts _ "]"
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
	// FunSig
	if !_fail(parser, _FunSigFail, errPos, failure, &pos) {
		goto fail
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
	// Stmts
	if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
		goto fail
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

func _FunAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Fun]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Fun}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ "[" FunSig _ "|" Stmts _ "]"
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// FunSig
		if p, n := _FunSigAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// Stmts
		if p, n := _StmtsAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FunSigAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _FunSig, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// (Ident/Op Ident TypeName/(IdentC Ident TypeName)+) Ret?
	// (Ident/Op Ident TypeName/(IdentC Ident TypeName)+)
	// Ident/Op Ident TypeName/(IdentC Ident TypeName)+
	{
		pos4 := pos
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// Op Ident TypeName
		// Op
		if !_accept(parser, _OpAccepts, &pos, &perr) {
			goto fail6
		}
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail6
		}
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok1
	fail6:
		pos = pos4
		// (IdentC Ident TypeName)+
		// (IdentC Ident TypeName)
		// IdentC Ident TypeName
		// IdentC
		if !_accept(parser, _IdentCAccepts, &pos, &perr) {
			goto fail8
		}
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail8
		}
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail8
		}
		for {
			pos10 := pos
			// (IdentC Ident TypeName)
			// IdentC Ident TypeName
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail12
			}
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail12
			}
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail12
			}
			continue
		fail12:
			pos = pos10
			break
		}
		goto ok1
	fail8:
		pos = pos4
		goto fail
	ok1:
	}
	// Ret?
	{
		pos16 := pos
		// Ret
		if !_accept(parser, _RetAccepts, &pos, &perr) {
			goto fail17
		}
		goto ok18
	fail17:
		pos = pos16
	ok18:
	}
	return _memoize(parser, _FunSig, start, pos, perr)
fail:
	return _memoize(parser, _FunSig, start, -1, perr)
}

func _FunSigNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// (Ident/Op Ident TypeName/(IdentC Ident TypeName)+) Ret?
	// (Ident/Op Ident TypeName/(IdentC Ident TypeName)+)
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// Ident/Op Ident TypeName/(IdentC Ident TypeName)+
		{
			pos6 := pos
			nkids4 := len(node.Kids)
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail7
			}
			goto ok3
		fail7:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// Op Ident TypeName
			// Op
			if !_node(parser, _OpNode, node, &pos) {
				goto fail8
			}
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail8
			}
			// TypeName
			if !_node(parser, _TypeNameNode, node, &pos) {
				goto fail8
			}
			goto ok3
		fail8:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// (IdentC Ident TypeName)+
			// (IdentC Ident TypeName)
			{
				nkids15 := len(node.Kids)
				pos016 := pos
				// IdentC Ident TypeName
				// IdentC
				if !_node(parser, _IdentCNode, node, &pos) {
					goto fail10
				}
				// Ident
				if !_node(parser, _IdentNode, node, &pos) {
					goto fail10
				}
				// TypeName
				if !_node(parser, _TypeNameNode, node, &pos) {
					goto fail10
				}
				sub := _sub(parser, pos016, pos, node.Kids[nkids15:])
				node.Kids = append(node.Kids[:nkids15], sub)
			}
			for {
				nkids11 := len(node.Kids)
				pos12 := pos
				// (IdentC Ident TypeName)
				{
					nkids18 := len(node.Kids)
					pos019 := pos
					// IdentC Ident TypeName
					// IdentC
					if !_node(parser, _IdentCNode, node, &pos) {
						goto fail14
					}
					// Ident
					if !_node(parser, _IdentNode, node, &pos) {
						goto fail14
					}
					// TypeName
					if !_node(parser, _TypeNameNode, node, &pos) {
						goto fail14
					}
					sub := _sub(parser, pos019, pos, node.Kids[nkids18:])
					node.Kids = append(node.Kids[:nkids18], sub)
				}
				continue
			fail14:
				node.Kids = node.Kids[:nkids11]
				pos = pos12
				break
			}
			goto ok3
		fail10:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			goto fail
		ok3:
		}
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
	}
	// Ret?
	{
		nkids21 := len(node.Kids)
		pos22 := pos
		// Ret
		if !_node(parser, _RetNode, node, &pos) {
			goto fail23
		}
		goto ok24
	fail23:
		node.Kids = node.Kids[:nkids21]
		pos = pos22
	ok24:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _FunSigFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _FunSig, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "FunSig",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _FunSig}
	// (Ident/Op Ident TypeName/(IdentC Ident TypeName)+) Ret?
	// (Ident/Op Ident TypeName/(IdentC Ident TypeName)+)
	// Ident/Op Ident TypeName/(IdentC Ident TypeName)+
	{
		pos4 := pos
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// Op Ident TypeName
		// Op
		if !_fail(parser, _OpFail, errPos, failure, &pos) {
			goto fail6
		}
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail6
		}
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail6
		}
		goto ok1
	fail6:
		pos = pos4
		// (IdentC Ident TypeName)+
		// (IdentC Ident TypeName)
		// IdentC Ident TypeName
		// IdentC
		if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
			goto fail8
		}
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail8
		}
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail8
		}
		for {
			pos10 := pos
			// (IdentC Ident TypeName)
			// IdentC Ident TypeName
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail12
			}
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail12
			}
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail12
			}
			continue
		fail12:
			pos = pos10
			break
		}
		goto ok1
	fail8:
		pos = pos4
		goto fail
	ok1:
	}
	// Ret?
	{
		pos16 := pos
		// Ret
		if !_fail(parser, _RetFail, errPos, failure, &pos) {
			goto fail17
		}
		goto ok18
	fail17:
		pos = pos16
	ok18:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _FunSigAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_FunSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _FunSig}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// (Ident/Op Ident TypeName/(IdentC Ident TypeName)+) Ret?
	{
		var node0 string
		// (Ident/Op Ident TypeName/(IdentC Ident TypeName)+)
		// Ident/Op Ident TypeName/(IdentC Ident TypeName)+
		{
			pos4 := pos
			var node3 string
			// Ident
			if p, n := _IdentAction(parser, pos); n == nil {
				goto fail5
			} else {
				node0 = *n
				pos = p
			}
			goto ok1
		fail5:
			node0 = node3
			pos = pos4
			// Op Ident TypeName
			{
				var node7 string
				// Op
				if p, n := _OpAction(parser, pos); n == nil {
					goto fail6
				} else {
					node7 = *n
					pos = p
				}
				node0, node7 = node0+node7, ""
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail6
				} else {
					node7 = *n
					pos = p
				}
				node0, node7 = node0+node7, ""
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail6
				} else {
					node7 = *n
					pos = p
				}
				node0, node7 = node0+node7, ""
			}
			goto ok1
		fail6:
			node0 = node3
			pos = pos4
			// (IdentC Ident TypeName)+
			{
				var node11 string
				// (IdentC Ident TypeName)
				// IdentC Ident TypeName
				{
					var node13 string
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail8
					} else {
						node13 = *n
						pos = p
					}
					node11, node13 = node11+node13, ""
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail8
					} else {
						node13 = *n
						pos = p
					}
					node11, node13 = node11+node13, ""
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail8
					} else {
						node13 = *n
						pos = p
					}
					node11, node13 = node11+node13, ""
				}
				node0 += node11
			}
			for {
				pos10 := pos
				var node11 string
				// (IdentC Ident TypeName)
				// IdentC Ident TypeName
				{
					var node14 string
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail12
					} else {
						node14 = *n
						pos = p
					}
					node11, node14 = node11+node14, ""
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail12
					} else {
						node14 = *n
						pos = p
					}
					node11, node14 = node11+node14, ""
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail12
					} else {
						node14 = *n
						pos = p
					}
					node11, node14 = node11+node14, ""
				}
				node0 += node11
				continue
			fail12:
				pos = pos10
				break
			}
			goto ok1
		fail8:
			node0 = node3
			pos = pos4
			goto fail
		ok1:
		}
		node, node0 = node+node0, ""
		// Ret?
		{
			pos16 := pos
			// Ret
			if p, n := _RetAction(parser, pos); n == nil {
				goto fail17
			} else {
				node0 = *n
				pos = p
			}
			goto ok18
		fail17:
			node0 = ""
			pos = pos16
		ok18:
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _RetAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Ret, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ "^" TypeName
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
	// TypeName
	if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
		goto fail
	}
	return _memoize(parser, _Ret, start, pos, perr)
fail:
	return _memoize(parser, _Ret, start, -1, perr)
}

func _RetNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ "^" TypeName
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
	// TypeName
	if !_node(parser, _TypeNameNode, node, &pos) {
		goto fail
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _RetFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Ret, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Ret",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Ret}
	// _ "^" TypeName
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
	// TypeName
	if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
		goto fail
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _RetAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Ret]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ret}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ "^" TypeName
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "^"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// TypeName
		if p, n := _TypeNameAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _VarAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Var, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Ident _ ":=" _ "[" Stmts _ "]"
	// Ident
	if !_accept(parser, _IdentAccepts, &pos, &perr) {
		goto fail
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
	// Stmts
	if !_accept(parser, _StmtsAccepts, &pos, &perr) {
		goto fail
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
	// Ident _ ":=" _ "[" Stmts _ "]"
	// Ident
	if !_node(parser, _IdentNode, node, &pos) {
		goto fail
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
	// Stmts
	if !_node(parser, _StmtsNode, node, &pos) {
		goto fail
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
	pos, failure := _failMemo(parser, _Var, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Var",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Var}
	// Ident _ ":=" _ "[" Stmts _ "]"
	// Ident
	if !_fail(parser, _IdentFail, errPos, failure, &pos) {
		goto fail
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
	// Stmts
	if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
		goto fail
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

func _VarAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Var]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Var}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Ident _ ":=" _ "[" Stmts _ "]"
	{
		var node0 string
		// Ident
		if p, n := _IdentAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// ":="
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
			goto fail
		}
		node0 = parser.text[pos : pos+2]
		pos += 2
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// Stmts
		if p, n := _StmtsAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeSigAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _TypeSig, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// !TypeVar TypeName/TypeParms? (Ident/Op)
	{
		pos3 := pos
		// !TypeVar TypeName
		// !TypeVar
		{
			pos7 := pos
			perr9 := perr
			// TypeVar
			if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
				goto ok6
			}
			pos = pos7
			perr = _max(perr9, pos)
			goto fail4
		ok6:
			pos = pos7
			perr = perr9
		}
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// TypeParms? (Ident/Op)
		// TypeParms?
		{
			pos13 := pos
			// TypeParms
			if !_accept(parser, _TypeParmsAccepts, &pos, &perr) {
				goto fail14
			}
			goto ok15
		fail14:
			pos = pos13
		ok15:
		}
		// (Ident/Op)
		// Ident/Op
		{
			pos19 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail20
			}
			goto ok16
		fail20:
			pos = pos19
			// Op
			if !_accept(parser, _OpAccepts, &pos, &perr) {
				goto fail21
			}
			goto ok16
		fail21:
			pos = pos19
			goto fail10
		ok16:
		}
		goto ok0
	fail10:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _TypeSig, start, pos, perr)
fail:
	return _memoize(parser, _TypeSig, start, -1, perr)
}

func _TypeSigNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// !TypeVar TypeName/TypeParms? (Ident/Op)
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// !TypeVar TypeName
		// !TypeVar
		{
			pos7 := pos
			nkids8 := len(node.Kids)
			// TypeVar
			if !_node(parser, _TypeVarNode, node, &pos) {
				goto ok6
			}
			pos = pos7
			node.Kids = node.Kids[:nkids8]
			goto fail4
		ok6:
			pos = pos7
			node.Kids = node.Kids[:nkids8]
		}
		// TypeName
		if !_node(parser, _TypeNameNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// TypeParms? (Ident/Op)
		// TypeParms?
		{
			nkids12 := len(node.Kids)
			pos13 := pos
			// TypeParms
			if !_node(parser, _TypeParmsNode, node, &pos) {
				goto fail14
			}
			goto ok15
		fail14:
			node.Kids = node.Kids[:nkids12]
			pos = pos13
		ok15:
		}
		// (Ident/Op)
		{
			nkids16 := len(node.Kids)
			pos017 := pos
			// Ident/Op
			{
				pos21 := pos
				nkids19 := len(node.Kids)
				// Ident
				if !_node(parser, _IdentNode, node, &pos) {
					goto fail22
				}
				goto ok18
			fail22:
				node.Kids = node.Kids[:nkids19]
				pos = pos21
				// Op
				if !_node(parser, _OpNode, node, &pos) {
					goto fail23
				}
				goto ok18
			fail23:
				node.Kids = node.Kids[:nkids19]
				pos = pos21
				goto fail10
			ok18:
			}
			sub := _sub(parser, pos017, pos, node.Kids[nkids16:])
			node.Kids = append(node.Kids[:nkids16], sub)
		}
		goto ok0
	fail10:
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

func _TypeSigFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _TypeSig, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeSig",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeSig}
	// !TypeVar TypeName/TypeParms? (Ident/Op)
	{
		pos3 := pos
		// !TypeVar TypeName
		// !TypeVar
		{
			pos7 := pos
			nkids8 := len(failure.Kids)
			// TypeVar
			if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
				goto ok6
			}
			pos = pos7
			failure.Kids = failure.Kids[:nkids8]
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "!TypeVar",
				})
			}
			goto fail4
		ok6:
			pos = pos7
			failure.Kids = failure.Kids[:nkids8]
		}
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// TypeParms? (Ident/Op)
		// TypeParms?
		{
			pos13 := pos
			// TypeParms
			if !_fail(parser, _TypeParmsFail, errPos, failure, &pos) {
				goto fail14
			}
			goto ok15
		fail14:
			pos = pos13
		ok15:
		}
		// (Ident/Op)
		// Ident/Op
		{
			pos19 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail20
			}
			goto ok16
		fail20:
			pos = pos19
			// Op
			if !_fail(parser, _OpFail, errPos, failure, &pos) {
				goto fail21
			}
			goto ok16
		fail21:
			pos = pos19
			goto fail10
		ok16:
		}
		goto ok0
	fail10:
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

func _TypeSigAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_TypeSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeSig}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// !TypeVar TypeName/TypeParms? (Ident/Op)
	{
		pos3 := pos
		var node2 string
		// !TypeVar TypeName
		{
			var node5 string
			// !TypeVar
			{
				pos7 := pos
				// TypeVar
				if p, n := _TypeVarAction(parser, pos); n == nil {
					goto ok6
				} else {
					pos = p
				}
				pos = pos7
				goto fail4
			ok6:
				pos = pos7
				node5 = ""
			}
			node, node5 = node+node5, ""
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail4
			} else {
				node5 = *n
				pos = p
			}
			node, node5 = node+node5, ""
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// TypeParms? (Ident/Op)
		{
			var node11 string
			// TypeParms?
			{
				pos13 := pos
				// TypeParms
				if p, n := _TypeParmsAction(parser, pos); n == nil {
					goto fail14
				} else {
					node11 = *n
					pos = p
				}
				goto ok15
			fail14:
				node11 = ""
				pos = pos13
			ok15:
			}
			node, node11 = node+node11, ""
			// (Ident/Op)
			// Ident/Op
			{
				pos19 := pos
				var node18 string
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail20
				} else {
					node11 = *n
					pos = p
				}
				goto ok16
			fail20:
				node11 = node18
				pos = pos19
				// Op
				if p, n := _OpAction(parser, pos); n == nil {
					goto fail21
				} else {
					node11 = *n
					pos = p
				}
				goto ok16
			fail21:
				node11 = node18
				pos = pos19
				goto fail10
			ok16:
			}
			node, node11 = node+node11, ""
		}
		goto ok0
	fail10:
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

func _TypeParmsAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _TypeParms, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// TypeVar/_ "(" TypeParm (_ "," TypeParm)* (_ ",")? _ ")"
	{
		pos3 := pos
		// TypeVar
		if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// _ "(" TypeParm (_ "," TypeParm)* (_ ",")? _ ")"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail5
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail5
		}
		pos++
		// TypeParm
		if !_accept(parser, _TypeParmAccepts, &pos, &perr) {
			goto fail5
		}
		// (_ "," TypeParm)*
		for {
			pos8 := pos
			// (_ "," TypeParm)
			// _ "," TypeParm
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail10
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				perr = _max(perr, pos)
				goto fail10
			}
			pos++
			// TypeParm
			if !_accept(parser, _TypeParmAccepts, &pos, &perr) {
				goto fail10
			}
			continue
		fail10:
			pos = pos8
			break
		}
		// (_ ",")?
		{
			pos13 := pos
			// (_ ",")
			// _ ","
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail14
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				perr = _max(perr, pos)
				goto fail14
			}
			pos++
			goto ok16
		fail14:
			pos = pos13
		ok16:
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail5
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			perr = _max(perr, pos)
			goto fail5
		}
		pos++
		goto ok0
	fail5:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _TypeParms, start, pos, perr)
fail:
	return _memoize(parser, _TypeParms, start, -1, perr)
}

func _TypeParmsNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// TypeVar/_ "(" TypeParm (_ "," TypeParm)* (_ ",")? _ ")"
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// TypeVar
		if !_node(parser, _TypeVarNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// _ "(" TypeParm (_ "," TypeParm)* (_ ",")? _ ")"
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail5
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			goto fail5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// TypeParm
		if !_node(parser, _TypeParmNode, node, &pos) {
			goto fail5
		}
		// (_ "," TypeParm)*
		for {
			nkids7 := len(node.Kids)
			pos8 := pos
			// (_ "," TypeParm)
			{
				nkids11 := len(node.Kids)
				pos012 := pos
				// _ "," TypeParm
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
				// TypeParm
				if !_node(parser, _TypeParmNode, node, &pos) {
					goto fail10
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
		// (_ ",")?
		{
			nkids14 := len(node.Kids)
			pos15 := pos
			// (_ ",")
			{
				nkids17 := len(node.Kids)
				pos018 := pos
				// _ ","
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail16
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					goto fail16
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				sub := _sub(parser, pos018, pos, node.Kids[nkids17:])
				node.Kids = append(node.Kids[:nkids17], sub)
			}
			goto ok20
		fail16:
			node.Kids = node.Kids[:nkids14]
			pos = pos15
		ok20:
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail5
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			goto fail5
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
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

func _TypeParmsFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _TypeParms, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeParms",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeParms}
	// TypeVar/_ "(" TypeParm (_ "," TypeParm)* (_ ",")? _ ")"
	{
		pos3 := pos
		// TypeVar
		if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// _ "(" TypeParm (_ "," TypeParm)* (_ ",")? _ ")"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail5
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail5
		}
		pos++
		// TypeParm
		if !_fail(parser, _TypeParmFail, errPos, failure, &pos) {
			goto fail5
		}
		// (_ "," TypeParm)*
		for {
			pos8 := pos
			// (_ "," TypeParm)
			// _ "," TypeParm
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail10
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\",\"",
					})
				}
				goto fail10
			}
			pos++
			// TypeParm
			if !_fail(parser, _TypeParmFail, errPos, failure, &pos) {
				goto fail10
			}
			continue
		fail10:
			pos = pos8
			break
		}
		// (_ ",")?
		{
			pos13 := pos
			// (_ ",")
			// _ ","
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail14
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\",\"",
					})
				}
				goto fail14
			}
			pos++
			goto ok16
		fail14:
			pos = pos13
		ok16:
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail5
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
				})
			}
			goto fail5
		}
		pos++
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

func _TypeParmsAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_TypeParms]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeParms}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// TypeVar/_ "(" TypeParm (_ "," TypeParm)* (_ ",")? _ ")"
	{
		pos3 := pos
		var node2 string
		// TypeVar
		if p, n := _TypeVarAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// _ "(" TypeParm (_ "," TypeParm)* (_ ",")? _ ")"
		{
			var node6 string
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail5
			} else {
				node6 = *n
				pos = p
			}
			node, node6 = node+node6, ""
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				goto fail5
			}
			node6 = parser.text[pos : pos+1]
			pos++
			node, node6 = node+node6, ""
			// TypeParm
			if p, n := _TypeParmAction(parser, pos); n == nil {
				goto fail5
			} else {
				node6 = *n
				pos = p
			}
			node, node6 = node+node6, ""
			// (_ "," TypeParm)*
			for {
				pos8 := pos
				var node9 string
				// (_ "," TypeParm)
				// _ "," TypeParm
				{
					var node11 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail10
					} else {
						node11 = *n
						pos = p
					}
					node9, node11 = node9+node11, ""
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail10
					}
					node11 = parser.text[pos : pos+1]
					pos++
					node9, node11 = node9+node11, ""
					// TypeParm
					if p, n := _TypeParmAction(parser, pos); n == nil {
						goto fail10
					} else {
						node11 = *n
						pos = p
					}
					node9, node11 = node9+node11, ""
				}
				node6 += node9
				continue
			fail10:
				pos = pos8
				break
			}
			node, node6 = node+node6, ""
			// (_ ",")?
			{
				pos13 := pos
				// (_ ",")
				// _ ","
				{
					var node15 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail14
					} else {
						node15 = *n
						pos = p
					}
					node6, node15 = node6+node15, ""
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail14
					}
					node15 = parser.text[pos : pos+1]
					pos++
					node6, node15 = node6+node15, ""
				}
				goto ok16
			fail14:
				node6 = ""
				pos = pos13
			ok16:
			}
			node, node6 = node+node6, ""
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail5
			} else {
				node6 = *n
				pos = p
			}
			node, node6 = node+node6, ""
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				goto fail5
			}
			node6 = parser.text[pos : pos+1]
			pos++
			node, node6 = node+node6, ""
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

func _TypeParmAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _TypeParm, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// TypeName/TypeVar TypeName?
	{
		pos3 := pos
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// TypeVar TypeName?
		// TypeVar
		if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
			goto fail5
		}
		// TypeName?
		{
			pos8 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail9
			}
			goto ok10
		fail9:
			pos = pos8
		ok10:
		}
		goto ok0
	fail5:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _TypeParm, start, pos, perr)
fail:
	return _memoize(parser, _TypeParm, start, -1, perr)
}

func _TypeParmNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// TypeName/TypeVar TypeName?
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// TypeName
		if !_node(parser, _TypeNameNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// TypeVar TypeName?
		// TypeVar
		if !_node(parser, _TypeVarNode, node, &pos) {
			goto fail5
		}
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

func _TypeParmFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _TypeParm, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeParm",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeParm}
	// TypeName/TypeVar TypeName?
	{
		pos3 := pos
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// TypeVar TypeName?
		// TypeVar
		if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
			goto fail5
		}
		// TypeName?
		{
			pos8 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail9
			}
			goto ok10
		fail9:
			pos = pos8
		ok10:
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

func _TypeParmAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_TypeParm]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeParm}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// TypeName/TypeVar TypeName?
	{
		pos3 := pos
		var node2 string
		// TypeName
		if p, n := _TypeNameAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// TypeVar TypeName?
		{
			var node6 string
			// TypeVar
			if p, n := _TypeVarAction(parser, pos); n == nil {
				goto fail5
			} else {
				node6 = *n
				pos = p
			}
			node, node6 = node+node6, ""
			// TypeName?
			{
				pos8 := pos
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail9
				} else {
					node6 = *n
					pos = p
				}
				goto ok10
			fail9:
				node6 = ""
				pos = pos8
			ok10:
			}
			node, node6 = node+node6, ""
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

func _TypeNameAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _TypeName, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// TypeVar? Ident* [?&]/TypeVar? Ident+/TypeVar/_ "[" TypeName* (_ "|" TypeName)? _ "]"/_ "(" TypeName (_ "," TypeName)* (_ ",")? _ ")" Ident+/_ "(" TypeName _ ")"
	{
		pos3 := pos
		// TypeVar? Ident* [?&]
		// TypeVar?
		{
			pos7 := pos
			// TypeVar
			if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
				goto fail8
			}
			goto ok9
		fail8:
			pos = pos7
		ok9:
		}
		// Ident*
		for {
			pos11 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail13
			}
			continue
		fail13:
			pos = pos11
			break
		}
		// [?&]
		if r, w := _next(parser, pos); r != '?' && r != '&' {
			perr = _max(perr, pos)
			goto fail4
		} else {
			pos += w
		}
		goto ok0
	fail4:
		pos = pos3
		// TypeVar? Ident+
		// TypeVar?
		{
			pos17 := pos
			// TypeVar
			if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
				goto fail18
			}
			goto ok19
		fail18:
			pos = pos17
		ok19:
		}
		// Ident+
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail14
		}
		for {
			pos21 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail23
			}
			continue
		fail23:
			pos = pos21
			break
		}
		goto ok0
	fail14:
		pos = pos3
		// TypeVar
		if !_accept(parser, _TypeVarAccepts, &pos, &perr) {
			goto fail24
		}
		goto ok0
	fail24:
		pos = pos3
		// _ "[" TypeName* (_ "|" TypeName)? _ "]"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail25
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			perr = _max(perr, pos)
			goto fail25
		}
		pos++
		// TypeName*
		for {
			pos28 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail30
			}
			continue
		fail30:
			pos = pos28
			break
		}
		// (_ "|" TypeName)?
		{
			pos32 := pos
			// (_ "|" TypeName)
			// _ "|" TypeName
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail33
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				perr = _max(perr, pos)
				goto fail33
			}
			pos++
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail33
			}
			goto ok35
		fail33:
			pos = pos32
		ok35:
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail25
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			perr = _max(perr, pos)
			goto fail25
		}
		pos++
		goto ok0
	fail25:
		pos = pos3
		// _ "(" TypeName (_ "," TypeName)* (_ ",")? _ ")" Ident+
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail36
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail36
		}
		pos++
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail36
		}
		// (_ "," TypeName)*
		for {
			pos39 := pos
			// (_ "," TypeName)
			// _ "," TypeName
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail41
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				perr = _max(perr, pos)
				goto fail41
			}
			pos++
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail41
			}
			continue
		fail41:
			pos = pos39
			break
		}
		// (_ ",")?
		{
			pos44 := pos
			// (_ ",")
			// _ ","
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail45
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				perr = _max(perr, pos)
				goto fail45
			}
			pos++
			goto ok47
		fail45:
			pos = pos44
		ok47:
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail36
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			perr = _max(perr, pos)
			goto fail36
		}
		pos++
		// Ident+
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail36
		}
		for {
			pos49 := pos
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail51
			}
			continue
		fail51:
			pos = pos49
			break
		}
		goto ok0
	fail36:
		pos = pos3
		// _ "(" TypeName _ ")"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail52
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail52
		}
		pos++
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail52
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail52
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			perr = _max(perr, pos)
			goto fail52
		}
		pos++
		goto ok0
	fail52:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _TypeName, start, pos, perr)
fail:
	return _memoize(parser, _TypeName, start, -1, perr)
}

func _TypeNameNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// TypeVar? Ident* [?&]/TypeVar? Ident+/TypeVar/_ "[" TypeName* (_ "|" TypeName)? _ "]"/_ "(" TypeName (_ "," TypeName)* (_ ",")? _ ")" Ident+/_ "(" TypeName _ ")"
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// TypeVar? Ident* [?&]
		// TypeVar?
		{
			nkids6 := len(node.Kids)
			pos7 := pos
			// TypeVar
			if !_node(parser, _TypeVarNode, node, &pos) {
				goto fail8
			}
			goto ok9
		fail8:
			node.Kids = node.Kids[:nkids6]
			pos = pos7
		ok9:
		}
		// Ident*
		for {
			nkids10 := len(node.Kids)
			pos11 := pos
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail13
			}
			continue
		fail13:
			node.Kids = node.Kids[:nkids10]
			pos = pos11
			break
		}
		// [?&]
		if r, w := _next(parser, pos); r != '?' && r != '&' {
			goto fail4
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// TypeVar? Ident+
		// TypeVar?
		{
			nkids16 := len(node.Kids)
			pos17 := pos
			// TypeVar
			if !_node(parser, _TypeVarNode, node, &pos) {
				goto fail18
			}
			goto ok19
		fail18:
			node.Kids = node.Kids[:nkids16]
			pos = pos17
		ok19:
		}
		// Ident+
		// Ident
		if !_node(parser, _IdentNode, node, &pos) {
			goto fail14
		}
		for {
			nkids20 := len(node.Kids)
			pos21 := pos
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail23
			}
			continue
		fail23:
			node.Kids = node.Kids[:nkids20]
			pos = pos21
			break
		}
		goto ok0
	fail14:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// TypeVar
		if !_node(parser, _TypeVarNode, node, &pos) {
			goto fail24
		}
		goto ok0
	fail24:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// _ "[" TypeName* (_ "|" TypeName)? _ "]"
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail25
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			goto fail25
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// TypeName*
		for {
			nkids27 := len(node.Kids)
			pos28 := pos
			// TypeName
			if !_node(parser, _TypeNameNode, node, &pos) {
				goto fail30
			}
			continue
		fail30:
			node.Kids = node.Kids[:nkids27]
			pos = pos28
			break
		}
		// (_ "|" TypeName)?
		{
			nkids31 := len(node.Kids)
			pos32 := pos
			// (_ "|" TypeName)
			{
				nkids34 := len(node.Kids)
				pos035 := pos
				// _ "|" TypeName
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail33
				}
				// "|"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
					goto fail33
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// TypeName
				if !_node(parser, _TypeNameNode, node, &pos) {
					goto fail33
				}
				sub := _sub(parser, pos035, pos, node.Kids[nkids34:])
				node.Kids = append(node.Kids[:nkids34], sub)
			}
			goto ok37
		fail33:
			node.Kids = node.Kids[:nkids31]
			pos = pos32
		ok37:
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail25
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			goto fail25
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail25:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// _ "(" TypeName (_ "," TypeName)* (_ ",")? _ ")" Ident+
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail38
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			goto fail38
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// TypeName
		if !_node(parser, _TypeNameNode, node, &pos) {
			goto fail38
		}
		// (_ "," TypeName)*
		for {
			nkids40 := len(node.Kids)
			pos41 := pos
			// (_ "," TypeName)
			{
				nkids44 := len(node.Kids)
				pos045 := pos
				// _ "," TypeName
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail43
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					goto fail43
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// TypeName
				if !_node(parser, _TypeNameNode, node, &pos) {
					goto fail43
				}
				sub := _sub(parser, pos045, pos, node.Kids[nkids44:])
				node.Kids = append(node.Kids[:nkids44], sub)
			}
			continue
		fail43:
			node.Kids = node.Kids[:nkids40]
			pos = pos41
			break
		}
		// (_ ",")?
		{
			nkids47 := len(node.Kids)
			pos48 := pos
			// (_ ",")
			{
				nkids50 := len(node.Kids)
				pos051 := pos
				// _ ","
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail49
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					goto fail49
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				sub := _sub(parser, pos051, pos, node.Kids[nkids50:])
				node.Kids = append(node.Kids[:nkids50], sub)
			}
			goto ok53
		fail49:
			node.Kids = node.Kids[:nkids47]
			pos = pos48
		ok53:
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail38
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			goto fail38
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// Ident+
		// Ident
		if !_node(parser, _IdentNode, node, &pos) {
			goto fail38
		}
		for {
			nkids54 := len(node.Kids)
			pos55 := pos
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail57
			}
			continue
		fail57:
			node.Kids = node.Kids[:nkids54]
			pos = pos55
			break
		}
		goto ok0
	fail38:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// _ "(" TypeName _ ")"
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail58
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			goto fail58
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// TypeName
		if !_node(parser, _TypeNameNode, node, &pos) {
			goto fail58
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail58
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			goto fail58
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail58:
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
	pos, failure := _failMemo(parser, _TypeName, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeName",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeName}
	// TypeVar? Ident* [?&]/TypeVar? Ident+/TypeVar/_ "[" TypeName* (_ "|" TypeName)? _ "]"/_ "(" TypeName (_ "," TypeName)* (_ ",")? _ ")" Ident+/_ "(" TypeName _ ")"
	{
		pos3 := pos
		// TypeVar? Ident* [?&]
		// TypeVar?
		{
			pos7 := pos
			// TypeVar
			if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
				goto fail8
			}
			goto ok9
		fail8:
			pos = pos7
		ok9:
		}
		// Ident*
		for {
			pos11 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail13
			}
			continue
		fail13:
			pos = pos11
			break
		}
		// [?&]
		if r, w := _next(parser, pos); r != '?' && r != '&' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[?&]",
				})
			}
			goto fail4
		} else {
			pos += w
		}
		goto ok0
	fail4:
		pos = pos3
		// TypeVar? Ident+
		// TypeVar?
		{
			pos17 := pos
			// TypeVar
			if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
				goto fail18
			}
			goto ok19
		fail18:
			pos = pos17
		ok19:
		}
		// Ident+
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail14
		}
		for {
			pos21 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail23
			}
			continue
		fail23:
			pos = pos21
			break
		}
		goto ok0
	fail14:
		pos = pos3
		// TypeVar
		if !_fail(parser, _TypeVarFail, errPos, failure, &pos) {
			goto fail24
		}
		goto ok0
	fail24:
		pos = pos3
		// _ "[" TypeName* (_ "|" TypeName)? _ "]"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail25
		}
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"[\"",
				})
			}
			goto fail25
		}
		pos++
		// TypeName*
		for {
			pos28 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail30
			}
			continue
		fail30:
			pos = pos28
			break
		}
		// (_ "|" TypeName)?
		{
			pos32 := pos
			// (_ "|" TypeName)
			// _ "|" TypeName
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail33
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\"|\"",
					})
				}
				goto fail33
			}
			pos++
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail33
			}
			goto ok35
		fail33:
			pos = pos32
		ok35:
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail25
		}
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"]\"",
				})
			}
			goto fail25
		}
		pos++
		goto ok0
	fail25:
		pos = pos3
		// _ "(" TypeName (_ "," TypeName)* (_ ",")? _ ")" Ident+
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail36
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail36
		}
		pos++
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail36
		}
		// (_ "," TypeName)*
		for {
			pos39 := pos
			// (_ "," TypeName)
			// _ "," TypeName
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail41
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\",\"",
					})
				}
				goto fail41
			}
			pos++
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail41
			}
			continue
		fail41:
			pos = pos39
			break
		}
		// (_ ",")?
		{
			pos44 := pos
			// (_ ",")
			// _ ","
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail45
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\",\"",
					})
				}
				goto fail45
			}
			pos++
			goto ok47
		fail45:
			pos = pos44
		ok47:
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail36
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
				})
			}
			goto fail36
		}
		pos++
		// Ident+
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail36
		}
		for {
			pos49 := pos
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail51
			}
			continue
		fail51:
			pos = pos49
			break
		}
		goto ok0
	fail36:
		pos = pos3
		// _ "(" TypeName _ ")"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail52
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail52
		}
		pos++
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail52
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail52
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
				})
			}
			goto fail52
		}
		pos++
		goto ok0
	fail52:
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

func _TypeNameAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_TypeName]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeName}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// TypeVar? Ident* [?&]/TypeVar? Ident+/TypeVar/_ "[" TypeName* (_ "|" TypeName)? _ "]"/_ "(" TypeName (_ "," TypeName)* (_ ",")? _ ")" Ident+/_ "(" TypeName _ ")"
	{
		pos3 := pos
		var node2 string
		// TypeVar? Ident* [?&]
		{
			var node5 string
			// TypeVar?
			{
				pos7 := pos
				// TypeVar
				if p, n := _TypeVarAction(parser, pos); n == nil {
					goto fail8
				} else {
					node5 = *n
					pos = p
				}
				goto ok9
			fail8:
				node5 = ""
				pos = pos7
			ok9:
			}
			node, node5 = node+node5, ""
			// Ident*
			for {
				pos11 := pos
				var node12 string
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail13
				} else {
					node12 = *n
					pos = p
				}
				node5 += node12
				continue
			fail13:
				pos = pos11
				break
			}
			node, node5 = node+node5, ""
			// [?&]
			if r, w := _next(parser, pos); r != '?' && r != '&' {
				goto fail4
			} else {
				node5 = parser.text[pos : pos+w]
				pos += w
			}
			node, node5 = node+node5, ""
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// TypeVar? Ident+
		{
			var node15 string
			// TypeVar?
			{
				pos17 := pos
				// TypeVar
				if p, n := _TypeVarAction(parser, pos); n == nil {
					goto fail18
				} else {
					node15 = *n
					pos = p
				}
				goto ok19
			fail18:
				node15 = ""
				pos = pos17
			ok19:
			}
			node, node15 = node+node15, ""
			// Ident+
			{
				var node22 string
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail14
				} else {
					node22 = *n
					pos = p
				}
				node15 += node22
			}
			for {
				pos21 := pos
				var node22 string
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail23
				} else {
					node22 = *n
					pos = p
				}
				node15 += node22
				continue
			fail23:
				pos = pos21
				break
			}
			node, node15 = node+node15, ""
		}
		goto ok0
	fail14:
		node = node2
		pos = pos3
		// TypeVar
		if p, n := _TypeVarAction(parser, pos); n == nil {
			goto fail24
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail24:
		node = node2
		pos = pos3
		// _ "[" TypeName* (_ "|" TypeName)? _ "]"
		{
			var node26 string
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail25
			} else {
				node26 = *n
				pos = p
			}
			node, node26 = node+node26, ""
			// "["
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
				goto fail25
			}
			node26 = parser.text[pos : pos+1]
			pos++
			node, node26 = node+node26, ""
			// TypeName*
			for {
				pos28 := pos
				var node29 string
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail30
				} else {
					node29 = *n
					pos = p
				}
				node26 += node29
				continue
			fail30:
				pos = pos28
				break
			}
			node, node26 = node+node26, ""
			// (_ "|" TypeName)?
			{
				pos32 := pos
				// (_ "|" TypeName)
				// _ "|" TypeName
				{
					var node34 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail33
					} else {
						node34 = *n
						pos = p
					}
					node26, node34 = node26+node34, ""
					// "|"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
						goto fail33
					}
					node34 = parser.text[pos : pos+1]
					pos++
					node26, node34 = node26+node34, ""
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail33
					} else {
						node34 = *n
						pos = p
					}
					node26, node34 = node26+node34, ""
				}
				goto ok35
			fail33:
				node26 = ""
				pos = pos32
			ok35:
			}
			node, node26 = node+node26, ""
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail25
			} else {
				node26 = *n
				pos = p
			}
			node, node26 = node+node26, ""
			// "]"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
				goto fail25
			}
			node26 = parser.text[pos : pos+1]
			pos++
			node, node26 = node+node26, ""
		}
		goto ok0
	fail25:
		node = node2
		pos = pos3
		// _ "(" TypeName (_ "," TypeName)* (_ ",")? _ ")" Ident+
		{
			var node37 string
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail36
			} else {
				node37 = *n
				pos = p
			}
			node, node37 = node+node37, ""
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				goto fail36
			}
			node37 = parser.text[pos : pos+1]
			pos++
			node, node37 = node+node37, ""
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail36
			} else {
				node37 = *n
				pos = p
			}
			node, node37 = node+node37, ""
			// (_ "," TypeName)*
			for {
				pos39 := pos
				var node40 string
				// (_ "," TypeName)
				// _ "," TypeName
				{
					var node42 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail41
					} else {
						node42 = *n
						pos = p
					}
					node40, node42 = node40+node42, ""
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail41
					}
					node42 = parser.text[pos : pos+1]
					pos++
					node40, node42 = node40+node42, ""
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail41
					} else {
						node42 = *n
						pos = p
					}
					node40, node42 = node40+node42, ""
				}
				node37 += node40
				continue
			fail41:
				pos = pos39
				break
			}
			node, node37 = node+node37, ""
			// (_ ",")?
			{
				pos44 := pos
				// (_ ",")
				// _ ","
				{
					var node46 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail45
					} else {
						node46 = *n
						pos = p
					}
					node37, node46 = node37+node46, ""
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail45
					}
					node46 = parser.text[pos : pos+1]
					pos++
					node37, node46 = node37+node46, ""
				}
				goto ok47
			fail45:
				node37 = ""
				pos = pos44
			ok47:
			}
			node, node37 = node+node37, ""
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail36
			} else {
				node37 = *n
				pos = p
			}
			node, node37 = node+node37, ""
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				goto fail36
			}
			node37 = parser.text[pos : pos+1]
			pos++
			node, node37 = node+node37, ""
			// Ident+
			{
				var node50 string
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail36
				} else {
					node50 = *n
					pos = p
				}
				node37 += node50
			}
			for {
				pos49 := pos
				var node50 string
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail51
				} else {
					node50 = *n
					pos = p
				}
				node37 += node50
				continue
			fail51:
				pos = pos49
				break
			}
			node, node37 = node+node37, ""
		}
		goto ok0
	fail36:
		node = node2
		pos = pos3
		// _ "(" TypeName _ ")"
		{
			var node53 string
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail52
			} else {
				node53 = *n
				pos = p
			}
			node, node53 = node+node53, ""
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				goto fail52
			}
			node53 = parser.text[pos : pos+1]
			pos++
			node, node53 = node+node53, ""
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail52
			} else {
				node53 = *n
				pos = p
			}
			node, node53 = node+node53, ""
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail52
			} else {
				node53 = *n
				pos = p
			}
			node, node53 = node+node53, ""
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				goto fail52
			}
			node53 = parser.text[pos : pos+1]
			pos++
			node, node53 = node+node53, ""
		}
		goto ok0
	fail52:
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
	// _ "{" ((IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+) _ "}"
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
	// ((IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+)
	// (IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+
	{
		pos4 := pos
		// (IdentC TypeName)+
		// (IdentC TypeName)
		// IdentC TypeName
		// IdentC
		if !_accept(parser, _IdentCAccepts, &pos, &perr) {
			goto fail5
		}
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail5
		}
		for {
			pos7 := pos
			// (IdentC TypeName)
			// IdentC TypeName
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail9
			}
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail9
			}
			continue
		fail9:
			pos = pos7
			break
		}
		goto ok1
	fail5:
		pos = pos4
		// Case (_ "," Case) (_ ",")?
		// Case
		if !_accept(parser, _CaseAccepts, &pos, &perr) {
			goto fail12
		}
		// (_ "," Case)
		// _ "," Case
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail12
		}
		// ","
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
			perr = _max(perr, pos)
			goto fail12
		}
		pos++
		// Case
		if !_accept(parser, _CaseAccepts, &pos, &perr) {
			goto fail12
		}
		// (_ ",")?
		{
			pos16 := pos
			// (_ ",")
			// _ ","
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
			goto ok19
		fail17:
			pos = pos16
		ok19:
		}
		goto ok1
	fail12:
		pos = pos4
		// MethSig+
		// MethSig
		if !_accept(parser, _MethSigAccepts, &pos, &perr) {
			goto fail20
		}
		for {
			pos22 := pos
			// MethSig
			if !_accept(parser, _MethSigAccepts, &pos, &perr) {
				goto fail24
			}
			continue
		fail24:
			pos = pos22
			break
		}
		goto ok1
	fail20:
		pos = pos4
		goto fail
	ok1:
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
	// _ "{" ((IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+) _ "}"
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
	pos++
	// ((IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+)
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// (IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+
		{
			pos6 := pos
			nkids4 := len(node.Kids)
			// (IdentC TypeName)+
			// (IdentC TypeName)
			{
				nkids12 := len(node.Kids)
				pos013 := pos
				// IdentC TypeName
				// IdentC
				if !_node(parser, _IdentCNode, node, &pos) {
					goto fail7
				}
				// TypeName
				if !_node(parser, _TypeNameNode, node, &pos) {
					goto fail7
				}
				sub := _sub(parser, pos013, pos, node.Kids[nkids12:])
				node.Kids = append(node.Kids[:nkids12], sub)
			}
			for {
				nkids8 := len(node.Kids)
				pos9 := pos
				// (IdentC TypeName)
				{
					nkids15 := len(node.Kids)
					pos016 := pos
					// IdentC TypeName
					// IdentC
					if !_node(parser, _IdentCNode, node, &pos) {
						goto fail11
					}
					// TypeName
					if !_node(parser, _TypeNameNode, node, &pos) {
						goto fail11
					}
					sub := _sub(parser, pos016, pos, node.Kids[nkids15:])
					node.Kids = append(node.Kids[:nkids15], sub)
				}
				continue
			fail11:
				node.Kids = node.Kids[:nkids8]
				pos = pos9
				break
			}
			goto ok3
		fail7:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// Case (_ "," Case) (_ ",")?
			// Case
			if !_node(parser, _CaseNode, node, &pos) {
				goto fail18
			}
			// (_ "," Case)
			{
				nkids20 := len(node.Kids)
				pos021 := pos
				// _ "," Case
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail18
				}
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					goto fail18
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// Case
				if !_node(parser, _CaseNode, node, &pos) {
					goto fail18
				}
				sub := _sub(parser, pos021, pos, node.Kids[nkids20:])
				node.Kids = append(node.Kids[:nkids20], sub)
			}
			// (_ ",")?
			{
				nkids23 := len(node.Kids)
				pos24 := pos
				// (_ ",")
				{
					nkids26 := len(node.Kids)
					pos027 := pos
					// _ ","
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail25
					}
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail25
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					sub := _sub(parser, pos027, pos, node.Kids[nkids26:])
					node.Kids = append(node.Kids[:nkids26], sub)
				}
				goto ok29
			fail25:
				node.Kids = node.Kids[:nkids23]
				pos = pos24
			ok29:
			}
			goto ok3
		fail18:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// MethSig+
			// MethSig
			if !_node(parser, _MethSigNode, node, &pos) {
				goto fail30
			}
			for {
				nkids31 := len(node.Kids)
				pos32 := pos
				// MethSig
				if !_node(parser, _MethSigNode, node, &pos) {
					goto fail34
				}
				continue
			fail34:
				node.Kids = node.Kids[:nkids31]
				pos = pos32
				break
			}
			goto ok3
		fail30:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			goto fail
		ok3:
		}
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
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
	// _ "{" ((IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+) _ "}"
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
	// ((IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+)
	// (IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+
	{
		pos4 := pos
		// (IdentC TypeName)+
		// (IdentC TypeName)
		// IdentC TypeName
		// IdentC
		if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
			goto fail5
		}
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail5
		}
		for {
			pos7 := pos
			// (IdentC TypeName)
			// IdentC TypeName
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail9
			}
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail9
			}
			continue
		fail9:
			pos = pos7
			break
		}
		goto ok1
	fail5:
		pos = pos4
		// Case (_ "," Case) (_ ",")?
		// Case
		if !_fail(parser, _CaseFail, errPos, failure, &pos) {
			goto fail12
		}
		// (_ "," Case)
		// _ "," Case
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail12
		}
		// ","
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\",\"",
				})
			}
			goto fail12
		}
		pos++
		// Case
		if !_fail(parser, _CaseFail, errPos, failure, &pos) {
			goto fail12
		}
		// (_ ",")?
		{
			pos16 := pos
			// (_ ",")
			// _ ","
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
			goto ok19
		fail17:
			pos = pos16
		ok19:
		}
		goto ok1
	fail12:
		pos = pos4
		// MethSig+
		// MethSig
		if !_fail(parser, _MethSigFail, errPos, failure, &pos) {
			goto fail20
		}
		for {
			pos22 := pos
			// MethSig
			if !_fail(parser, _MethSigFail, errPos, failure, &pos) {
				goto fail24
			}
			continue
		fail24:
			pos = pos22
			break
		}
		goto ok1
	fail20:
		pos = pos4
		goto fail
	ok1:
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

func _TypeAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Type]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Type}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ "{" ((IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+) _ "}"
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// ((IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+)
		// (IdentC TypeName)+/Case (_ "," Case) (_ ",")?/MethSig+
		{
			pos4 := pos
			var node3 string
			// (IdentC TypeName)+
			{
				var node8 string
				// (IdentC TypeName)
				// IdentC TypeName
				{
					var node10 string
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail5
					} else {
						node10 = *n
						pos = p
					}
					node8, node10 = node8+node10, ""
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail5
					} else {
						node10 = *n
						pos = p
					}
					node8, node10 = node8+node10, ""
				}
				node0 += node8
			}
			for {
				pos7 := pos
				var node8 string
				// (IdentC TypeName)
				// IdentC TypeName
				{
					var node11 string
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail9
					} else {
						node11 = *n
						pos = p
					}
					node8, node11 = node8+node11, ""
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail9
					} else {
						node11 = *n
						pos = p
					}
					node8, node11 = node8+node11, ""
				}
				node0 += node8
				continue
			fail9:
				pos = pos7
				break
			}
			goto ok1
		fail5:
			node0 = node3
			pos = pos4
			// Case (_ "," Case) (_ ",")?
			{
				var node13 string
				// Case
				if p, n := _CaseAction(parser, pos); n == nil {
					goto fail12
				} else {
					node13 = *n
					pos = p
				}
				node0, node13 = node0+node13, ""
				// (_ "," Case)
				// _ "," Case
				{
					var node14 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail12
					} else {
						node14 = *n
						pos = p
					}
					node13, node14 = node13+node14, ""
					// ","
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
						goto fail12
					}
					node14 = parser.text[pos : pos+1]
					pos++
					node13, node14 = node13+node14, ""
					// Case
					if p, n := _CaseAction(parser, pos); n == nil {
						goto fail12
					} else {
						node14 = *n
						pos = p
					}
					node13, node14 = node13+node14, ""
				}
				node0, node13 = node0+node13, ""
				// (_ ",")?
				{
					pos16 := pos
					// (_ ",")
					// _ ","
					{
						var node18 string
						// _
						if p, n := __Action(parser, pos); n == nil {
							goto fail17
						} else {
							node18 = *n
							pos = p
						}
						node13, node18 = node13+node18, ""
						// ","
						if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
							goto fail17
						}
						node18 = parser.text[pos : pos+1]
						pos++
						node13, node18 = node13+node18, ""
					}
					goto ok19
				fail17:
					node13 = ""
					pos = pos16
				ok19:
				}
				node0, node13 = node0+node13, ""
			}
			goto ok1
		fail12:
			node0 = node3
			pos = pos4
			// MethSig+
			{
				var node23 string
				// MethSig
				if p, n := _MethSigAction(parser, pos); n == nil {
					goto fail20
				} else {
					node23 = *n
					pos = p
				}
				node0 += node23
			}
			for {
				pos22 := pos
				var node23 string
				// MethSig
				if p, n := _MethSigAction(parser, pos); n == nil {
					goto fail24
				} else {
					node23 = *n
					pos = p
				}
				node0 += node23
				continue
			fail24:
				pos = pos22
				break
			}
			goto ok1
		fail20:
			node0 = node3
			pos = pos4
			goto fail
		ok1:
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "}"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _CaseAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Case, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Ident/IdentC TypeName
	{
		pos3 := pos
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// IdentC TypeName
		// IdentC
		if !_accept(parser, _IdentCAccepts, &pos, &perr) {
			goto fail5
		}
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Case, start, pos, perr)
fail:
	return _memoize(parser, _Case, start, -1, perr)
}

func _CaseNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// Ident/IdentC TypeName
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// Ident
		if !_node(parser, _IdentNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// IdentC TypeName
		// IdentC
		if !_node(parser, _IdentCNode, node, &pos) {
			goto fail5
		}
		// TypeName
		if !_node(parser, _TypeNameNode, node, &pos) {
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

func _CaseFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Case, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Case",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Case}
	// Ident/IdentC TypeName
	{
		pos3 := pos
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// IdentC TypeName
		// IdentC
		if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
			goto fail5
		}
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
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

func _CaseAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Case]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Case}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Ident/IdentC TypeName
	{
		pos3 := pos
		var node2 string
		// Ident
		if p, n := _IdentAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// IdentC TypeName
		{
			var node6 string
			// IdentC
			if p, n := _IdentCAction(parser, pos); n == nil {
				goto fail5
			} else {
				node6 = *n
				pos = p
			}
			node, node6 = node+node6, ""
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail5
			} else {
				node6 = *n
				pos = p
			}
			node, node6 = node+node6, ""
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

func _MethSigAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _MethSig, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ "[" (Ident/Op TypeName/(IdentC TypeName)+) Ret? _ "]"
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
	// (Ident/Op TypeName/(IdentC TypeName)+)
	// Ident/Op TypeName/(IdentC TypeName)+
	{
		pos4 := pos
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// Op TypeName
		// Op
		if !_accept(parser, _OpAccepts, &pos, &perr) {
			goto fail6
		}
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok1
	fail6:
		pos = pos4
		// (IdentC TypeName)+
		// (IdentC TypeName)
		// IdentC TypeName
		// IdentC
		if !_accept(parser, _IdentCAccepts, &pos, &perr) {
			goto fail8
		}
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail8
		}
		for {
			pos10 := pos
			// (IdentC TypeName)
			// IdentC TypeName
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail12
			}
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail12
			}
			continue
		fail12:
			pos = pos10
			break
		}
		goto ok1
	fail8:
		pos = pos4
		goto fail
	ok1:
	}
	// Ret?
	{
		pos16 := pos
		// Ret
		if !_accept(parser, _RetAccepts, &pos, &perr) {
			goto fail17
		}
		goto ok18
	fail17:
		pos = pos16
	ok18:
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
	return _memoize(parser, _MethSig, start, pos, perr)
fail:
	return _memoize(parser, _MethSig, start, -1, perr)
}

func _MethSigNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ "[" (Ident/Op TypeName/(IdentC TypeName)+) Ret? _ "]"
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
	// (Ident/Op TypeName/(IdentC TypeName)+)
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// Ident/Op TypeName/(IdentC TypeName)+
		{
			pos6 := pos
			nkids4 := len(node.Kids)
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail7
			}
			goto ok3
		fail7:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// Op TypeName
			// Op
			if !_node(parser, _OpNode, node, &pos) {
				goto fail8
			}
			// TypeName
			if !_node(parser, _TypeNameNode, node, &pos) {
				goto fail8
			}
			goto ok3
		fail8:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// (IdentC TypeName)+
			// (IdentC TypeName)
			{
				nkids15 := len(node.Kids)
				pos016 := pos
				// IdentC TypeName
				// IdentC
				if !_node(parser, _IdentCNode, node, &pos) {
					goto fail10
				}
				// TypeName
				if !_node(parser, _TypeNameNode, node, &pos) {
					goto fail10
				}
				sub := _sub(parser, pos016, pos, node.Kids[nkids15:])
				node.Kids = append(node.Kids[:nkids15], sub)
			}
			for {
				nkids11 := len(node.Kids)
				pos12 := pos
				// (IdentC TypeName)
				{
					nkids18 := len(node.Kids)
					pos019 := pos
					// IdentC TypeName
					// IdentC
					if !_node(parser, _IdentCNode, node, &pos) {
						goto fail14
					}
					// TypeName
					if !_node(parser, _TypeNameNode, node, &pos) {
						goto fail14
					}
					sub := _sub(parser, pos019, pos, node.Kids[nkids18:])
					node.Kids = append(node.Kids[:nkids18], sub)
				}
				continue
			fail14:
				node.Kids = node.Kids[:nkids11]
				pos = pos12
				break
			}
			goto ok3
		fail10:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			goto fail
		ok3:
		}
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
	}
	// Ret?
	{
		nkids21 := len(node.Kids)
		pos22 := pos
		// Ret
		if !_node(parser, _RetNode, node, &pos) {
			goto fail23
		}
		goto ok24
	fail23:
		node.Kids = node.Kids[:nkids21]
		pos = pos22
	ok24:
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

func _MethSigFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _MethSig, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "MethSig",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _MethSig}
	// _ "[" (Ident/Op TypeName/(IdentC TypeName)+) Ret? _ "]"
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
	// (Ident/Op TypeName/(IdentC TypeName)+)
	// Ident/Op TypeName/(IdentC TypeName)+
	{
		pos4 := pos
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// Op TypeName
		// Op
		if !_fail(parser, _OpFail, errPos, failure, &pos) {
			goto fail6
		}
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail6
		}
		goto ok1
	fail6:
		pos = pos4
		// (IdentC TypeName)+
		// (IdentC TypeName)
		// IdentC TypeName
		// IdentC
		if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
			goto fail8
		}
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail8
		}
		for {
			pos10 := pos
			// (IdentC TypeName)
			// IdentC TypeName
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail12
			}
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail12
			}
			continue
		fail12:
			pos = pos10
			break
		}
		goto ok1
	fail8:
		pos = pos4
		goto fail
	ok1:
	}
	// Ret?
	{
		pos16 := pos
		// Ret
		if !_fail(parser, _RetFail, errPos, failure, &pos) {
			goto fail17
		}
		goto ok18
	fail17:
		pos = pos16
	ok18:
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

func _MethSigAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_MethSig]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _MethSig}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ "[" (Ident/Op TypeName/(IdentC TypeName)+) Ret? _ "]"
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// (Ident/Op TypeName/(IdentC TypeName)+)
		// Ident/Op TypeName/(IdentC TypeName)+
		{
			pos4 := pos
			var node3 string
			// Ident
			if p, n := _IdentAction(parser, pos); n == nil {
				goto fail5
			} else {
				node0 = *n
				pos = p
			}
			goto ok1
		fail5:
			node0 = node3
			pos = pos4
			// Op TypeName
			{
				var node7 string
				// Op
				if p, n := _OpAction(parser, pos); n == nil {
					goto fail6
				} else {
					node7 = *n
					pos = p
				}
				node0, node7 = node0+node7, ""
				// TypeName
				if p, n := _TypeNameAction(parser, pos); n == nil {
					goto fail6
				} else {
					node7 = *n
					pos = p
				}
				node0, node7 = node0+node7, ""
			}
			goto ok1
		fail6:
			node0 = node3
			pos = pos4
			// (IdentC TypeName)+
			{
				var node11 string
				// (IdentC TypeName)
				// IdentC TypeName
				{
					var node13 string
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail8
					} else {
						node13 = *n
						pos = p
					}
					node11, node13 = node11+node13, ""
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail8
					} else {
						node13 = *n
						pos = p
					}
					node11, node13 = node11+node13, ""
				}
				node0 += node11
			}
			for {
				pos10 := pos
				var node11 string
				// (IdentC TypeName)
				// IdentC TypeName
				{
					var node14 string
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail12
					} else {
						node14 = *n
						pos = p
					}
					node11, node14 = node11+node14, ""
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail12
					} else {
						node14 = *n
						pos = p
					}
					node11, node14 = node11+node14, ""
				}
				node0 += node11
				continue
			fail12:
				pos = pos10
				break
			}
			goto ok1
		fail8:
			node0 = node3
			pos = pos4
			goto fail
		ok1:
		}
		node, node0 = node+node0, ""
		// Ret?
		{
			pos16 := pos
			// Ret
			if p, n := _RetAction(parser, pos); n == nil {
				goto fail17
			} else {
				node0 = *n
				pos = p
			}
			goto ok18
		fail17:
			node0 = ""
			pos = pos16
		ok18:
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _StmtsAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Stmts, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// (Stmt (_ "." Stmt)* (_ ".")?)?
	{
		pos1 := pos
		// (Stmt (_ "." Stmt)* (_ ".")?)
		// Stmt (_ "." Stmt)* (_ ".")?
		// Stmt
		if !_accept(parser, _StmtAccepts, &pos, &perr) {
			goto fail2
		}
		// (_ "." Stmt)*
		for {
			pos5 := pos
			// (_ "." Stmt)
			// _ "." Stmt
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail7
			}
			// "."
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
				perr = _max(perr, pos)
				goto fail7
			}
			pos++
			// Stmt
			if !_accept(parser, _StmtAccepts, &pos, &perr) {
				goto fail7
			}
			continue
		fail7:
			pos = pos5
			break
		}
		// (_ ".")?
		{
			pos10 := pos
			// (_ ".")
			// _ "."
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail11
			}
			// "."
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
				perr = _max(perr, pos)
				goto fail11
			}
			pos++
			goto ok13
		fail11:
			pos = pos10
		ok13:
		}
		goto ok14
	fail2:
		pos = pos1
	ok14:
	}
	return _memoize(parser, _Stmts, start, pos, perr)
}

func _StmtsNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// (Stmt (_ "." Stmt)* (_ ".")?)?
	{
		nkids0 := len(node.Kids)
		pos1 := pos
		// (Stmt (_ "." Stmt)* (_ ".")?)
		{
			nkids3 := len(node.Kids)
			pos04 := pos
			// Stmt (_ "." Stmt)* (_ ".")?
			// Stmt
			if !_node(parser, _StmtNode, node, &pos) {
				goto fail2
			}
			// (_ "." Stmt)*
			for {
				nkids6 := len(node.Kids)
				pos7 := pos
				// (_ "." Stmt)
				{
					nkids10 := len(node.Kids)
					pos011 := pos
					// _ "." Stmt
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail9
					}
					// "."
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
						goto fail9
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					// Stmt
					if !_node(parser, _StmtNode, node, &pos) {
						goto fail9
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
			// (_ ".")?
			{
				nkids13 := len(node.Kids)
				pos14 := pos
				// (_ ".")
				{
					nkids16 := len(node.Kids)
					pos017 := pos
					// _ "."
					// _
					if !_node(parser, __Node, node, &pos) {
						goto fail15
					}
					// "."
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
						goto fail15
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
					pos++
					sub := _sub(parser, pos017, pos, node.Kids[nkids16:])
					node.Kids = append(node.Kids[:nkids16], sub)
				}
				goto ok19
			fail15:
				node.Kids = node.Kids[:nkids13]
				pos = pos14
			ok19:
			}
			sub := _sub(parser, pos04, pos, node.Kids[nkids3:])
			node.Kids = append(node.Kids[:nkids3], sub)
		}
		goto ok20
	fail2:
		node.Kids = node.Kids[:nkids0]
		pos = pos1
	ok20:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
}

func _StmtsFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Stmts, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Stmts",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Stmts}
	// (Stmt (_ "." Stmt)* (_ ".")?)?
	{
		pos1 := pos
		// (Stmt (_ "." Stmt)* (_ ".")?)
		// Stmt (_ "." Stmt)* (_ ".")?
		// Stmt
		if !_fail(parser, _StmtFail, errPos, failure, &pos) {
			goto fail2
		}
		// (_ "." Stmt)*
		for {
			pos5 := pos
			// (_ "." Stmt)
			// _ "." Stmt
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail7
			}
			// "."
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\".\"",
					})
				}
				goto fail7
			}
			pos++
			// Stmt
			if !_fail(parser, _StmtFail, errPos, failure, &pos) {
				goto fail7
			}
			continue
		fail7:
			pos = pos5
			break
		}
		// (_ ".")?
		{
			pos10 := pos
			// (_ ".")
			// _ "."
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail11
			}
			// "."
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\".\"",
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
		goto ok14
	fail2:
		pos = pos1
	ok14:
	}
	parser.fail[key] = failure
	return pos, failure
}

func _StmtsAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Stmts]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Stmts}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// (Stmt (_ "." Stmt)* (_ ".")?)?
	{
		pos1 := pos
		// (Stmt (_ "." Stmt)* (_ ".")?)
		// Stmt (_ "." Stmt)* (_ ".")?
		{
			var node3 string
			// Stmt
			if p, n := _StmtAction(parser, pos); n == nil {
				goto fail2
			} else {
				node3 = *n
				pos = p
			}
			node, node3 = node+node3, ""
			// (_ "." Stmt)*
			for {
				pos5 := pos
				var node6 string
				// (_ "." Stmt)
				// _ "." Stmt
				{
					var node8 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail7
					} else {
						node8 = *n
						pos = p
					}
					node6, node8 = node6+node8, ""
					// "."
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
						goto fail7
					}
					node8 = parser.text[pos : pos+1]
					pos++
					node6, node8 = node6+node8, ""
					// Stmt
					if p, n := _StmtAction(parser, pos); n == nil {
						goto fail7
					} else {
						node8 = *n
						pos = p
					}
					node6, node8 = node6+node8, ""
				}
				node3 += node6
				continue
			fail7:
				pos = pos5
				break
			}
			node, node3 = node+node3, ""
			// (_ ".")?
			{
				pos10 := pos
				// (_ ".")
				// _ "."
				{
					var node12 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail11
					} else {
						node12 = *n
						pos = p
					}
					node3, node12 = node3+node12, ""
					// "."
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
						goto fail11
					}
					node12 = parser.text[pos : pos+1]
					pos++
					node3, node12 = node3+node12, ""
				}
				goto ok13
			fail11:
				node3 = ""
				pos = pos10
			ok13:
			}
			node, node3 = node+node3, ""
		}
		goto ok14
	fail2:
		node = ""
		pos = pos1
	ok14:
	}
	parser.act[key] = node
	return pos, &node
}

func _StmtAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Stmt, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Return/Assign/Expr
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
		// Expr
		if !_accept(parser, _ExprAccepts, &pos, &perr) {
			goto fail6
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
	// Return/Assign/Expr
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
		// Expr
		if !_node(parser, _ExprNode, node, &pos) {
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

func _StmtFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Stmt, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Stmt",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Stmt}
	// Return/Assign/Expr
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
		// Expr
		if !_fail(parser, _ExprFail, errPos, failure, &pos) {
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

func _StmtAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Stmt]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Stmt}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Return/Assign/Expr
	{
		pos3 := pos
		var node2 string
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
		// Expr
		if p, n := _ExprAction(parser, pos); n == nil {
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

func _ReturnAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Return, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ "^" Expr
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
	// Expr
	if !_accept(parser, _ExprAccepts, &pos, &perr) {
		goto fail
	}
	return _memoize(parser, _Return, start, pos, perr)
fail:
	return _memoize(parser, _Return, start, -1, perr)
}

func _ReturnNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ "^" Expr
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
	// Expr
	if !_node(parser, _ExprNode, node, &pos) {
		goto fail
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _ReturnFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Return, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Return",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Return}
	// _ "^" Expr
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
	// Expr
	if !_fail(parser, _ExprFail, errPos, failure, &pos) {
		goto fail
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _ReturnAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Return]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Return}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ "^" Expr
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "^"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "^" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// Expr
		if p, n := _ExprAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _AssignAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Assign, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Ident TypeName? (_ "," Ident TypeName?)* _ ":=" Expr
	// Ident
	if !_accept(parser, _IdentAccepts, &pos, &perr) {
		goto fail
	}
	// TypeName?
	{
		pos2 := pos
		// TypeName
		if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
			goto fail3
		}
		goto ok4
	fail3:
		pos = pos2
	ok4:
	}
	// (_ "," Ident TypeName?)*
	for {
		pos6 := pos
		// (_ "," Ident TypeName?)
		// _ "," Ident TypeName?
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
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail8
		}
		// TypeName?
		{
			pos11 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail12
			}
			goto ok13
		fail12:
			pos = pos11
		ok13:
		}
		continue
	fail8:
		pos = pos6
		break
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
	// Expr
	if !_accept(parser, _ExprAccepts, &pos, &perr) {
		goto fail
	}
	return _memoize(parser, _Assign, start, pos, perr)
fail:
	return _memoize(parser, _Assign, start, -1, perr)
}

func _AssignNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// Ident TypeName? (_ "," Ident TypeName?)* _ ":=" Expr
	// Ident
	if !_node(parser, _IdentNode, node, &pos) {
		goto fail
	}
	// TypeName?
	{
		nkids1 := len(node.Kids)
		pos2 := pos
		// TypeName
		if !_node(parser, _TypeNameNode, node, &pos) {
			goto fail3
		}
		goto ok4
	fail3:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
	ok4:
	}
	// (_ "," Ident TypeName?)*
	for {
		nkids5 := len(node.Kids)
		pos6 := pos
		// (_ "," Ident TypeName?)
		{
			nkids9 := len(node.Kids)
			pos010 := pos
			// _ "," Ident TypeName?
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail8
			}
			// ","
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
				goto fail8
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail8
			}
			// TypeName?
			{
				nkids12 := len(node.Kids)
				pos13 := pos
				// TypeName
				if !_node(parser, _TypeNameNode, node, &pos) {
					goto fail14
				}
				goto ok15
			fail14:
				node.Kids = node.Kids[:nkids12]
				pos = pos13
			ok15:
			}
			sub := _sub(parser, pos010, pos, node.Kids[nkids9:])
			node.Kids = append(node.Kids[:nkids9], sub)
		}
		continue
	fail8:
		node.Kids = node.Kids[:nkids5]
		pos = pos6
		break
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
	// Expr
	if !_node(parser, _ExprNode, node, &pos) {
		goto fail
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _AssignFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Assign, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Assign",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Assign}
	// Ident TypeName? (_ "," Ident TypeName?)* _ ":=" Expr
	// Ident
	if !_fail(parser, _IdentFail, errPos, failure, &pos) {
		goto fail
	}
	// TypeName?
	{
		pos2 := pos
		// TypeName
		if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
			goto fail3
		}
		goto ok4
	fail3:
		pos = pos2
	ok4:
	}
	// (_ "," Ident TypeName?)*
	for {
		pos6 := pos
		// (_ "," Ident TypeName?)
		// _ "," Ident TypeName?
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
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail8
		}
		// TypeName?
		{
			pos11 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail12
			}
			goto ok13
		fail12:
			pos = pos11
		ok13:
		}
		continue
	fail8:
		pos = pos6
		break
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
	// Expr
	if !_fail(parser, _ExprFail, errPos, failure, &pos) {
		goto fail
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _AssignAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Assign]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Assign}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Ident TypeName? (_ "," Ident TypeName?)* _ ":=" Expr
	{
		var node0 string
		// Ident
		if p, n := _IdentAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// TypeName?
		{
			pos2 := pos
			// TypeName
			if p, n := _TypeNameAction(parser, pos); n == nil {
				goto fail3
			} else {
				node0 = *n
				pos = p
			}
			goto ok4
		fail3:
			node0 = ""
			pos = pos2
		ok4:
		}
		node, node0 = node+node0, ""
		// (_ "," Ident TypeName?)*
		for {
			pos6 := pos
			var node7 string
			// (_ "," Ident TypeName?)
			// _ "," Ident TypeName?
			{
				var node9 string
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail8
				} else {
					node9 = *n
					pos = p
				}
				node7, node9 = node7+node9, ""
				// ","
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "," {
					goto fail8
				}
				node9 = parser.text[pos : pos+1]
				pos++
				node7, node9 = node7+node9, ""
				// Ident
				if p, n := _IdentAction(parser, pos); n == nil {
					goto fail8
				} else {
					node9 = *n
					pos = p
				}
				node7, node9 = node7+node9, ""
				// TypeName?
				{
					pos11 := pos
					// TypeName
					if p, n := _TypeNameAction(parser, pos); n == nil {
						goto fail12
					} else {
						node9 = *n
						pos = p
					}
					goto ok13
				fail12:
					node9 = ""
					pos = pos11
				ok13:
				}
				node7, node9 = node7+node9, ""
			}
			node0 += node7
			continue
		fail8:
			pos = pos6
			break
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// ":="
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != ":=" {
			goto fail
		}
		node0 = parser.text[pos : pos+2]
		pos += 2
		node, node0 = node+node0, ""
		// Expr
		if p, n := _ExprAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
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
	// Cascade/Call/Primary
	{
		pos3 := pos
		// Cascade
		if !_accept(parser, _CascadeAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Call
		if !_accept(parser, _CallAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Primary
		if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok0
	fail6:
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
	// Cascade/Call/Primary
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// Cascade
		if !_node(parser, _CascadeNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Call
		if !_node(parser, _CallNode, node, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Primary
		if !_node(parser, _PrimaryNode, node, &pos) {
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
	// Cascade/Call/Primary
	{
		pos3 := pos
		// Cascade
		if !_fail(parser, _CascadeFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Call
		if !_fail(parser, _CallFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Primary
		if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
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

func _ExprAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Expr]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Expr}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Cascade/Call/Primary
	{
		pos3 := pos
		var node2 string
		// Cascade
		if p, n := _CascadeAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Call
		if p, n := _CallAction(parser, pos); n == nil {
			goto fail5
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail5:
		node = node2
		pos = pos3
		// Primary
		if p, n := _PrimaryAction(parser, pos); n == nil {
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

func _CascadeAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Cascade, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Call (_ ";" Ident/BinMsg/NaryMsg)+
	// Call
	if !_accept(parser, _CallAccepts, &pos, &perr) {
		goto fail
	}
	// (_ ";" Ident/BinMsg/NaryMsg)+
	// (_ ";" Ident/BinMsg/NaryMsg)
	// _ ";" Ident/BinMsg/NaryMsg
	{
		pos8 := pos
		// _ ";" Ident
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail9
		}
		// ";"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
			perr = _max(perr, pos)
			goto fail9
		}
		pos++
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail9
		}
		goto ok5
	fail9:
		pos = pos8
		// BinMsg
		if !_accept(parser, _BinMsgAccepts, &pos, &perr) {
			goto fail11
		}
		goto ok5
	fail11:
		pos = pos8
		// NaryMsg
		if !_accept(parser, _NaryMsgAccepts, &pos, &perr) {
			goto fail12
		}
		goto ok5
	fail12:
		pos = pos8
		goto fail
	ok5:
	}
	for {
		pos2 := pos
		// (_ ";" Ident/BinMsg/NaryMsg)
		// _ ";" Ident/BinMsg/NaryMsg
		{
			pos16 := pos
			// _ ";" Ident
			// _
			if !_accept(parser, __Accepts, &pos, &perr) {
				goto fail17
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
				perr = _max(perr, pos)
				goto fail17
			}
			pos++
			// Ident
			if !_accept(parser, _IdentAccepts, &pos, &perr) {
				goto fail17
			}
			goto ok13
		fail17:
			pos = pos16
			// BinMsg
			if !_accept(parser, _BinMsgAccepts, &pos, &perr) {
				goto fail19
			}
			goto ok13
		fail19:
			pos = pos16
			// NaryMsg
			if !_accept(parser, _NaryMsgAccepts, &pos, &perr) {
				goto fail20
			}
			goto ok13
		fail20:
			pos = pos16
			goto fail4
		ok13:
		}
		continue
	fail4:
		pos = pos2
		break
	}
	return _memoize(parser, _Cascade, start, pos, perr)
fail:
	return _memoize(parser, _Cascade, start, -1, perr)
}

func _CascadeNode(parser *_Parser, start int) (int, *peg.Node) {
	dp := parser.deltaPos[start][_Cascade]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Cascade}
	node := parser.node[key]
	if node != nil {
		return start + int(dp-1), node
	}
	pos := start
	node = &peg.Node{Name: "Cascade"}
	// Call (_ ";" Ident/BinMsg/NaryMsg)+
	// Call
	if !_node(parser, _CallNode, node, &pos) {
		goto fail
	}
	// (_ ";" Ident/BinMsg/NaryMsg)+
	// (_ ";" Ident/BinMsg/NaryMsg)
	{
		nkids5 := len(node.Kids)
		pos06 := pos
		// _ ";" Ident/BinMsg/NaryMsg
		{
			pos10 := pos
			nkids8 := len(node.Kids)
			// _ ";" Ident
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail11
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
				goto fail11
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			// Ident
			if !_node(parser, _IdentNode, node, &pos) {
				goto fail11
			}
			goto ok7
		fail11:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			// BinMsg
			if !_node(parser, _BinMsgNode, node, &pos) {
				goto fail13
			}
			goto ok7
		fail13:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			// NaryMsg
			if !_node(parser, _NaryMsgNode, node, &pos) {
				goto fail14
			}
			goto ok7
		fail14:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			goto fail
		ok7:
		}
		sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
		node.Kids = append(node.Kids[:nkids5], sub)
	}
	for {
		nkids1 := len(node.Kids)
		pos2 := pos
		// (_ ";" Ident/BinMsg/NaryMsg)
		{
			nkids15 := len(node.Kids)
			pos016 := pos
			// _ ";" Ident/BinMsg/NaryMsg
			{
				pos20 := pos
				nkids18 := len(node.Kids)
				// _ ";" Ident
				// _
				if !_node(parser, __Node, node, &pos) {
					goto fail21
				}
				// ";"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
					goto fail21
				}
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
				pos++
				// Ident
				if !_node(parser, _IdentNode, node, &pos) {
					goto fail21
				}
				goto ok17
			fail21:
				node.Kids = node.Kids[:nkids18]
				pos = pos20
				// BinMsg
				if !_node(parser, _BinMsgNode, node, &pos) {
					goto fail23
				}
				goto ok17
			fail23:
				node.Kids = node.Kids[:nkids18]
				pos = pos20
				// NaryMsg
				if !_node(parser, _NaryMsgNode, node, &pos) {
					goto fail24
				}
				goto ok17
			fail24:
				node.Kids = node.Kids[:nkids18]
				pos = pos20
				goto fail4
			ok17:
			}
			sub := _sub(parser, pos016, pos, node.Kids[nkids15:])
			node.Kids = append(node.Kids[:nkids15], sub)
		}
		continue
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
		break
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _CascadeFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Cascade, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Cascade",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Cascade}
	// Call (_ ";" Ident/BinMsg/NaryMsg)+
	// Call
	if !_fail(parser, _CallFail, errPos, failure, &pos) {
		goto fail
	}
	// (_ ";" Ident/BinMsg/NaryMsg)+
	// (_ ";" Ident/BinMsg/NaryMsg)
	// _ ";" Ident/BinMsg/NaryMsg
	{
		pos8 := pos
		// _ ";" Ident
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail9
		}
		// ";"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\";\"",
				})
			}
			goto fail9
		}
		pos++
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail9
		}
		goto ok5
	fail9:
		pos = pos8
		// BinMsg
		if !_fail(parser, _BinMsgFail, errPos, failure, &pos) {
			goto fail11
		}
		goto ok5
	fail11:
		pos = pos8
		// NaryMsg
		if !_fail(parser, _NaryMsgFail, errPos, failure, &pos) {
			goto fail12
		}
		goto ok5
	fail12:
		pos = pos8
		goto fail
	ok5:
	}
	for {
		pos2 := pos
		// (_ ";" Ident/BinMsg/NaryMsg)
		// _ ";" Ident/BinMsg/NaryMsg
		{
			pos16 := pos
			// _ ";" Ident
			// _
			if !_fail(parser, __Fail, errPos, failure, &pos) {
				goto fail17
			}
			// ";"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "\";\"",
					})
				}
				goto fail17
			}
			pos++
			// Ident
			if !_fail(parser, _IdentFail, errPos, failure, &pos) {
				goto fail17
			}
			goto ok13
		fail17:
			pos = pos16
			// BinMsg
			if !_fail(parser, _BinMsgFail, errPos, failure, &pos) {
				goto fail19
			}
			goto ok13
		fail19:
			pos = pos16
			// NaryMsg
			if !_fail(parser, _NaryMsgFail, errPos, failure, &pos) {
				goto fail20
			}
			goto ok13
		fail20:
			pos = pos16
			goto fail4
		ok13:
		}
		continue
	fail4:
		pos = pos2
		break
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _CascadeAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Cascade]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Cascade}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Call (_ ";" Ident/BinMsg/NaryMsg)+
	{
		var node0 string
		// Call
		if p, n := _CallAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// (_ ";" Ident/BinMsg/NaryMsg)+
		{
			var node3 string
			// (_ ";" Ident/BinMsg/NaryMsg)
			// _ ";" Ident/BinMsg/NaryMsg
			{
				pos8 := pos
				var node7 string
				// _ ";" Ident
				{
					var node10 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail9
					} else {
						node10 = *n
						pos = p
					}
					node3, node10 = node3+node10, ""
					// ";"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
						goto fail9
					}
					node10 = parser.text[pos : pos+1]
					pos++
					node3, node10 = node3+node10, ""
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail9
					} else {
						node10 = *n
						pos = p
					}
					node3, node10 = node3+node10, ""
				}
				goto ok5
			fail9:
				node3 = node7
				pos = pos8
				// BinMsg
				if p, n := _BinMsgAction(parser, pos); n == nil {
					goto fail11
				} else {
					node3 = *n
					pos = p
				}
				goto ok5
			fail11:
				node3 = node7
				pos = pos8
				// NaryMsg
				if p, n := _NaryMsgAction(parser, pos); n == nil {
					goto fail12
				} else {
					node3 = *n
					pos = p
				}
				goto ok5
			fail12:
				node3 = node7
				pos = pos8
				goto fail
			ok5:
			}
			node0 += node3
		}
		for {
			pos2 := pos
			var node3 string
			// (_ ";" Ident/BinMsg/NaryMsg)
			// _ ";" Ident/BinMsg/NaryMsg
			{
				pos16 := pos
				var node15 string
				// _ ";" Ident
				{
					var node18 string
					// _
					if p, n := __Action(parser, pos); n == nil {
						goto fail17
					} else {
						node18 = *n
						pos = p
					}
					node3, node18 = node3+node18, ""
					// ";"
					if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
						goto fail17
					}
					node18 = parser.text[pos : pos+1]
					pos++
					node3, node18 = node3+node18, ""
					// Ident
					if p, n := _IdentAction(parser, pos); n == nil {
						goto fail17
					} else {
						node18 = *n
						pos = p
					}
					node3, node18 = node3+node18, ""
				}
				goto ok13
			fail17:
				node3 = node15
				pos = pos16
				// BinMsg
				if p, n := _BinMsgAction(parser, pos); n == nil {
					goto fail19
				} else {
					node3 = *n
					pos = p
				}
				goto ok13
			fail19:
				node3 = node15
				pos = pos16
				// NaryMsg
				if p, n := _NaryMsgAction(parser, pos); n == nil {
					goto fail20
				} else {
					node3 = *n
					pos = p
				}
				goto ok13
			fail20:
				node3 = node15
				pos = pos16
				goto fail4
			ok13:
			}
			node0 += node3
			continue
		fail4:
			pos = pos2
			break
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _CallAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Call, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Nary/Binary/Unary
	{
		pos3 := pos
		// Nary
		if !_accept(parser, _NaryAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Binary
		if !_accept(parser, _BinaryAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Unary
		if !_accept(parser, _UnaryAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Call, start, pos, perr)
fail:
	return _memoize(parser, _Call, start, -1, perr)
}

func _CallNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// Nary/Binary/Unary
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// Nary
		if !_node(parser, _NaryNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Binary
		if !_node(parser, _BinaryNode, node, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Unary
		if !_node(parser, _UnaryNode, node, &pos) {
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

func _CallFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Call, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Call",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Call}
	// Nary/Binary/Unary
	{
		pos3 := pos
		// Nary
		if !_fail(parser, _NaryFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Binary
		if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Unary
		if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
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

func _CallAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Call]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Call}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Nary/Binary/Unary
	{
		pos3 := pos
		var node2 string
		// Nary
		if p, n := _NaryAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Binary
		if p, n := _BinaryAction(parser, pos); n == nil {
			goto fail5
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail5:
		node = node2
		pos = pos3
		// Unary
		if p, n := _UnaryAction(parser, pos); n == nil {
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

func _UnaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Unary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// (Primary/ModName+) Ident+
	// (Primary/ModName+)
	// Primary/ModName+
	{
		pos4 := pos
		// Primary
		if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// ModName+
		// ModName
		if !_accept(parser, _ModNameAccepts, &pos, &perr) {
			goto fail6
		}
		for {
			pos8 := pos
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail10
			}
			continue
		fail10:
			pos = pos8
			break
		}
		goto ok1
	fail6:
		pos = pos4
		goto fail
	ok1:
	}
	// Ident+
	// Ident
	if !_accept(parser, _IdentAccepts, &pos, &perr) {
		goto fail
	}
	for {
		pos12 := pos
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail14
		}
		continue
	fail14:
		pos = pos12
		break
	}
	return _memoize(parser, _Unary, start, pos, perr)
fail:
	return _memoize(parser, _Unary, start, -1, perr)
}

func _UnaryNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// (Primary/ModName+) Ident+
	// (Primary/ModName+)
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// Primary/ModName+
		{
			pos6 := pos
			nkids4 := len(node.Kids)
			// Primary
			if !_node(parser, _PrimaryNode, node, &pos) {
				goto fail7
			}
			goto ok3
		fail7:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// ModName+
			// ModName
			if !_node(parser, _ModNameNode, node, &pos) {
				goto fail8
			}
			for {
				nkids9 := len(node.Kids)
				pos10 := pos
				// ModName
				if !_node(parser, _ModNameNode, node, &pos) {
					goto fail12
				}
				continue
			fail12:
				node.Kids = node.Kids[:nkids9]
				pos = pos10
				break
			}
			goto ok3
		fail8:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			goto fail
		ok3:
		}
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
	}
	// Ident+
	// Ident
	if !_node(parser, _IdentNode, node, &pos) {
		goto fail
	}
	for {
		nkids13 := len(node.Kids)
		pos14 := pos
		// Ident
		if !_node(parser, _IdentNode, node, &pos) {
			goto fail16
		}
		continue
	fail16:
		node.Kids = node.Kids[:nkids13]
		pos = pos14
		break
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _UnaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Unary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Unary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Unary}
	// (Primary/ModName+) Ident+
	// (Primary/ModName+)
	// Primary/ModName+
	{
		pos4 := pos
		// Primary
		if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// ModName+
		// ModName
		if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
			goto fail6
		}
		for {
			pos8 := pos
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail10
			}
			continue
		fail10:
			pos = pos8
			break
		}
		goto ok1
	fail6:
		pos = pos4
		goto fail
	ok1:
	}
	// Ident+
	// Ident
	if !_fail(parser, _IdentFail, errPos, failure, &pos) {
		goto fail
	}
	for {
		pos12 := pos
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail14
		}
		continue
	fail14:
		pos = pos12
		break
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _UnaryAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Unary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Unary}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// (Primary/ModName+) Ident+
	{
		var node0 string
		// (Primary/ModName+)
		// Primary/ModName+
		{
			pos4 := pos
			var node3 string
			// Primary
			if p, n := _PrimaryAction(parser, pos); n == nil {
				goto fail5
			} else {
				node0 = *n
				pos = p
			}
			goto ok1
		fail5:
			node0 = node3
			pos = pos4
			// ModName+
			{
				var node9 string
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail6
				} else {
					node9 = *n
					pos = p
				}
				node0 += node9
			}
			for {
				pos8 := pos
				var node9 string
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail10
				} else {
					node9 = *n
					pos = p
				}
				node0 += node9
				continue
			fail10:
				pos = pos8
				break
			}
			goto ok1
		fail6:
			node0 = node3
			pos = pos4
			goto fail
		ok1:
		}
		node, node0 = node+node0, ""
		// Ident+
		{
			var node13 string
			// Ident
			if p, n := _IdentAction(parser, pos); n == nil {
				goto fail
			} else {
				node13 = *n
				pos = p
			}
			node0 += node13
		}
		for {
			pos12 := pos
			var node13 string
			// Ident
			if p, n := _IdentAction(parser, pos); n == nil {
				goto fail14
			} else {
				node13 = *n
				pos = p
			}
			node0 += node13
			continue
		fail14:
			pos = pos12
			break
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BinaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Binary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// (Unary/Primary/ModName+) BinMsg
	// (Unary/Primary/ModName+)
	// Unary/Primary/ModName+
	{
		pos4 := pos
		// Unary
		if !_accept(parser, _UnaryAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// Primary
		if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok1
	fail6:
		pos = pos4
		// ModName+
		// ModName
		if !_accept(parser, _ModNameAccepts, &pos, &perr) {
			goto fail7
		}
		for {
			pos9 := pos
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail11
			}
			continue
		fail11:
			pos = pos9
			break
		}
		goto ok1
	fail7:
		pos = pos4
		goto fail
	ok1:
	}
	// BinMsg
	if !_accept(parser, _BinMsgAccepts, &pos, &perr) {
		goto fail
	}
	return _memoize(parser, _Binary, start, pos, perr)
fail:
	return _memoize(parser, _Binary, start, -1, perr)
}

func _BinaryNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// (Unary/Primary/ModName+) BinMsg
	// (Unary/Primary/ModName+)
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// Unary/Primary/ModName+
		{
			pos6 := pos
			nkids4 := len(node.Kids)
			// Unary
			if !_node(parser, _UnaryNode, node, &pos) {
				goto fail7
			}
			goto ok3
		fail7:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// Primary
			if !_node(parser, _PrimaryNode, node, &pos) {
				goto fail8
			}
			goto ok3
		fail8:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// ModName+
			// ModName
			if !_node(parser, _ModNameNode, node, &pos) {
				goto fail9
			}
			for {
				nkids10 := len(node.Kids)
				pos11 := pos
				// ModName
				if !_node(parser, _ModNameNode, node, &pos) {
					goto fail13
				}
				continue
			fail13:
				node.Kids = node.Kids[:nkids10]
				pos = pos11
				break
			}
			goto ok3
		fail9:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			goto fail
		ok3:
		}
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
	}
	// BinMsg
	if !_node(parser, _BinMsgNode, node, &pos) {
		goto fail
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _BinaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Binary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Binary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Binary}
	// (Unary/Primary/ModName+) BinMsg
	// (Unary/Primary/ModName+)
	// Unary/Primary/ModName+
	{
		pos4 := pos
		// Unary
		if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// Primary
		if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
			goto fail6
		}
		goto ok1
	fail6:
		pos = pos4
		// ModName+
		// ModName
		if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
			goto fail7
		}
		for {
			pos9 := pos
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail11
			}
			continue
		fail11:
			pos = pos9
			break
		}
		goto ok1
	fail7:
		pos = pos4
		goto fail
	ok1:
	}
	// BinMsg
	if !_fail(parser, _BinMsgFail, errPos, failure, &pos) {
		goto fail
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _BinaryAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Binary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Binary}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// (Unary/Primary/ModName+) BinMsg
	{
		var node0 string
		// (Unary/Primary/ModName+)
		// Unary/Primary/ModName+
		{
			pos4 := pos
			var node3 string
			// Unary
			if p, n := _UnaryAction(parser, pos); n == nil {
				goto fail5
			} else {
				node0 = *n
				pos = p
			}
			goto ok1
		fail5:
			node0 = node3
			pos = pos4
			// Primary
			if p, n := _PrimaryAction(parser, pos); n == nil {
				goto fail6
			} else {
				node0 = *n
				pos = p
			}
			goto ok1
		fail6:
			node0 = node3
			pos = pos4
			// ModName+
			{
				var node10 string
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail7
				} else {
					node10 = *n
					pos = p
				}
				node0 += node10
			}
			for {
				pos9 := pos
				var node10 string
				// ModName
				if p, n := _ModNameAction(parser, pos); n == nil {
					goto fail11
				} else {
					node10 = *n
					pos = p
				}
				node0 += node10
				continue
			fail11:
				pos = pos9
				break
			}
			goto ok1
		fail7:
			node0 = node3
			pos = pos4
			goto fail
		ok1:
		}
		node, node0 = node+node0, ""
		// BinMsg
		if p, n := _BinMsgAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BinMsgAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _BinMsg, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Op (Binary/Unary/Primary)
	// Op
	if !_accept(parser, _OpAccepts, &pos, &perr) {
		goto fail
	}
	// (Binary/Unary/Primary)
	// Binary/Unary/Primary
	{
		pos4 := pos
		// Binary
		if !_accept(parser, _BinaryAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// Unary
		if !_accept(parser, _UnaryAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok1
	fail6:
		pos = pos4
		// Primary
		if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
			goto fail7
		}
		goto ok1
	fail7:
		pos = pos4
		goto fail
	ok1:
	}
	return _memoize(parser, _BinMsg, start, pos, perr)
fail:
	return _memoize(parser, _BinMsg, start, -1, perr)
}

func _BinMsgNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// Op (Binary/Unary/Primary)
	// Op
	if !_node(parser, _OpNode, node, &pos) {
		goto fail
	}
	// (Binary/Unary/Primary)
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// Binary/Unary/Primary
		{
			pos6 := pos
			nkids4 := len(node.Kids)
			// Binary
			if !_node(parser, _BinaryNode, node, &pos) {
				goto fail7
			}
			goto ok3
		fail7:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// Unary
			if !_node(parser, _UnaryNode, node, &pos) {
				goto fail8
			}
			goto ok3
		fail8:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// Primary
			if !_node(parser, _PrimaryNode, node, &pos) {
				goto fail9
			}
			goto ok3
		fail9:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			goto fail
		ok3:
		}
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _BinMsgFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _BinMsg, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "BinMsg",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _BinMsg}
	// Op (Binary/Unary/Primary)
	// Op
	if !_fail(parser, _OpFail, errPos, failure, &pos) {
		goto fail
	}
	// (Binary/Unary/Primary)
	// Binary/Unary/Primary
	{
		pos4 := pos
		// Binary
		if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok1
	fail5:
		pos = pos4
		// Unary
		if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
			goto fail6
		}
		goto ok1
	fail6:
		pos = pos4
		// Primary
		if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
			goto fail7
		}
		goto ok1
	fail7:
		pos = pos4
		goto fail
	ok1:
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _BinMsgAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_BinMsg]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _BinMsg}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Op (Binary/Unary/Primary)
	{
		var node0 string
		// Op
		if p, n := _OpAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// (Binary/Unary/Primary)
		// Binary/Unary/Primary
		{
			pos4 := pos
			var node3 string
			// Binary
			if p, n := _BinaryAction(parser, pos); n == nil {
				goto fail5
			} else {
				node0 = *n
				pos = p
			}
			goto ok1
		fail5:
			node0 = node3
			pos = pos4
			// Unary
			if p, n := _UnaryAction(parser, pos); n == nil {
				goto fail6
			} else {
				node0 = *n
				pos = p
			}
			goto ok1
		fail6:
			node0 = node3
			pos = pos4
			// Primary
			if p, n := _PrimaryAction(parser, pos); n == nil {
				goto fail7
			} else {
				node0 = *n
				pos = p
			}
			goto ok1
		fail7:
			node0 = node3
			pos = pos4
			goto fail
		ok1:
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _NaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Nary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// (Binary/Unary/Primary/ModName+)? NaryMsg
	// (Binary/Unary/Primary/ModName+)?
	{
		pos2 := pos
		// (Binary/Unary/Primary/ModName+)
		// Binary/Unary/Primary/ModName+
		{
			pos7 := pos
			// Binary
			if !_accept(parser, _BinaryAccepts, &pos, &perr) {
				goto fail8
			}
			goto ok4
		fail8:
			pos = pos7
			// Unary
			if !_accept(parser, _UnaryAccepts, &pos, &perr) {
				goto fail9
			}
			goto ok4
		fail9:
			pos = pos7
			// Primary
			if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
				goto fail10
			}
			goto ok4
		fail10:
			pos = pos7
			// ModName+
			// ModName
			if !_accept(parser, _ModNameAccepts, &pos, &perr) {
				goto fail11
			}
			for {
				pos13 := pos
				// ModName
				if !_accept(parser, _ModNameAccepts, &pos, &perr) {
					goto fail15
				}
				continue
			fail15:
				pos = pos13
				break
			}
			goto ok4
		fail11:
			pos = pos7
			goto fail3
		ok4:
		}
		goto ok16
	fail3:
		pos = pos2
	ok16:
	}
	// NaryMsg
	if !_accept(parser, _NaryMsgAccepts, &pos, &perr) {
		goto fail
	}
	return _memoize(parser, _Nary, start, pos, perr)
fail:
	return _memoize(parser, _Nary, start, -1, perr)
}

func _NaryNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// (Binary/Unary/Primary/ModName+)? NaryMsg
	// (Binary/Unary/Primary/ModName+)?
	{
		nkids1 := len(node.Kids)
		pos2 := pos
		// (Binary/Unary/Primary/ModName+)
		{
			nkids4 := len(node.Kids)
			pos05 := pos
			// Binary/Unary/Primary/ModName+
			{
				pos9 := pos
				nkids7 := len(node.Kids)
				// Binary
				if !_node(parser, _BinaryNode, node, &pos) {
					goto fail10
				}
				goto ok6
			fail10:
				node.Kids = node.Kids[:nkids7]
				pos = pos9
				// Unary
				if !_node(parser, _UnaryNode, node, &pos) {
					goto fail11
				}
				goto ok6
			fail11:
				node.Kids = node.Kids[:nkids7]
				pos = pos9
				// Primary
				if !_node(parser, _PrimaryNode, node, &pos) {
					goto fail12
				}
				goto ok6
			fail12:
				node.Kids = node.Kids[:nkids7]
				pos = pos9
				// ModName+
				// ModName
				if !_node(parser, _ModNameNode, node, &pos) {
					goto fail13
				}
				for {
					nkids14 := len(node.Kids)
					pos15 := pos
					// ModName
					if !_node(parser, _ModNameNode, node, &pos) {
						goto fail17
					}
					continue
				fail17:
					node.Kids = node.Kids[:nkids14]
					pos = pos15
					break
				}
				goto ok6
			fail13:
				node.Kids = node.Kids[:nkids7]
				pos = pos9
				goto fail3
			ok6:
			}
			sub := _sub(parser, pos05, pos, node.Kids[nkids4:])
			node.Kids = append(node.Kids[:nkids4], sub)
		}
		goto ok18
	fail3:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
	ok18:
	}
	// NaryMsg
	if !_node(parser, _NaryMsgNode, node, &pos) {
		goto fail
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _NaryFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Nary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Nary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Nary}
	// (Binary/Unary/Primary/ModName+)? NaryMsg
	// (Binary/Unary/Primary/ModName+)?
	{
		pos2 := pos
		// (Binary/Unary/Primary/ModName+)
		// Binary/Unary/Primary/ModName+
		{
			pos7 := pos
			// Binary
			if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
				goto fail8
			}
			goto ok4
		fail8:
			pos = pos7
			// Unary
			if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
				goto fail9
			}
			goto ok4
		fail9:
			pos = pos7
			// Primary
			if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
				goto fail10
			}
			goto ok4
		fail10:
			pos = pos7
			// ModName+
			// ModName
			if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
				goto fail11
			}
			for {
				pos13 := pos
				// ModName
				if !_fail(parser, _ModNameFail, errPos, failure, &pos) {
					goto fail15
				}
				continue
			fail15:
				pos = pos13
				break
			}
			goto ok4
		fail11:
			pos = pos7
			goto fail3
		ok4:
		}
		goto ok16
	fail3:
		pos = pos2
	ok16:
	}
	// NaryMsg
	if !_fail(parser, _NaryMsgFail, errPos, failure, &pos) {
		goto fail
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _NaryAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Nary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Nary}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// (Binary/Unary/Primary/ModName+)? NaryMsg
	{
		var node0 string
		// (Binary/Unary/Primary/ModName+)?
		{
			pos2 := pos
			// (Binary/Unary/Primary/ModName+)
			// Binary/Unary/Primary/ModName+
			{
				pos7 := pos
				var node6 string
				// Binary
				if p, n := _BinaryAction(parser, pos); n == nil {
					goto fail8
				} else {
					node0 = *n
					pos = p
				}
				goto ok4
			fail8:
				node0 = node6
				pos = pos7
				// Unary
				if p, n := _UnaryAction(parser, pos); n == nil {
					goto fail9
				} else {
					node0 = *n
					pos = p
				}
				goto ok4
			fail9:
				node0 = node6
				pos = pos7
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail10
				} else {
					node0 = *n
					pos = p
				}
				goto ok4
			fail10:
				node0 = node6
				pos = pos7
				// ModName+
				{
					var node14 string
					// ModName
					if p, n := _ModNameAction(parser, pos); n == nil {
						goto fail11
					} else {
						node14 = *n
						pos = p
					}
					node0 += node14
				}
				for {
					pos13 := pos
					var node14 string
					// ModName
					if p, n := _ModNameAction(parser, pos); n == nil {
						goto fail15
					} else {
						node14 = *n
						pos = p
					}
					node0 += node14
					continue
				fail15:
					pos = pos13
					break
				}
				goto ok4
			fail11:
				node0 = node6
				pos = pos7
				goto fail3
			ok4:
			}
			goto ok16
		fail3:
			node0 = ""
			pos = pos2
		ok16:
		}
		node, node0 = node+node0, ""
		// NaryMsg
		if p, n := _NaryMsgAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _NaryMsgAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _NaryMsg, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// (IdentC (Binary/Unary/Primary))+
	// (IdentC (Binary/Unary/Primary))
	// IdentC (Binary/Unary/Primary)
	// IdentC
	if !_accept(parser, _IdentCAccepts, &pos, &perr) {
		goto fail
	}
	// (Binary/Unary/Primary)
	// Binary/Unary/Primary
	{
		pos8 := pos
		// Binary
		if !_accept(parser, _BinaryAccepts, &pos, &perr) {
			goto fail9
		}
		goto ok5
	fail9:
		pos = pos8
		// Unary
		if !_accept(parser, _UnaryAccepts, &pos, &perr) {
			goto fail10
		}
		goto ok5
	fail10:
		pos = pos8
		// Primary
		if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
			goto fail11
		}
		goto ok5
	fail11:
		pos = pos8
		goto fail
	ok5:
	}
	for {
		pos1 := pos
		// (IdentC (Binary/Unary/Primary))
		// IdentC (Binary/Unary/Primary)
		// IdentC
		if !_accept(parser, _IdentCAccepts, &pos, &perr) {
			goto fail3
		}
		// (Binary/Unary/Primary)
		// Binary/Unary/Primary
		{
			pos16 := pos
			// Binary
			if !_accept(parser, _BinaryAccepts, &pos, &perr) {
				goto fail17
			}
			goto ok13
		fail17:
			pos = pos16
			// Unary
			if !_accept(parser, _UnaryAccepts, &pos, &perr) {
				goto fail18
			}
			goto ok13
		fail18:
			pos = pos16
			// Primary
			if !_accept(parser, _PrimaryAccepts, &pos, &perr) {
				goto fail19
			}
			goto ok13
		fail19:
			pos = pos16
			goto fail3
		ok13:
		}
		continue
	fail3:
		pos = pos1
		break
	}
	return _memoize(parser, _NaryMsg, start, pos, perr)
fail:
	return _memoize(parser, _NaryMsg, start, -1, perr)
}

func _NaryMsgNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// (IdentC (Binary/Unary/Primary))+
	// (IdentC (Binary/Unary/Primary))
	{
		nkids4 := len(node.Kids)
		pos05 := pos
		// IdentC (Binary/Unary/Primary)
		// IdentC
		if !_node(parser, _IdentCNode, node, &pos) {
			goto fail
		}
		// (Binary/Unary/Primary)
		{
			nkids7 := len(node.Kids)
			pos08 := pos
			// Binary/Unary/Primary
			{
				pos12 := pos
				nkids10 := len(node.Kids)
				// Binary
				if !_node(parser, _BinaryNode, node, &pos) {
					goto fail13
				}
				goto ok9
			fail13:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				// Unary
				if !_node(parser, _UnaryNode, node, &pos) {
					goto fail14
				}
				goto ok9
			fail14:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				// Primary
				if !_node(parser, _PrimaryNode, node, &pos) {
					goto fail15
				}
				goto ok9
			fail15:
				node.Kids = node.Kids[:nkids10]
				pos = pos12
				goto fail
			ok9:
			}
			sub := _sub(parser, pos08, pos, node.Kids[nkids7:])
			node.Kids = append(node.Kids[:nkids7], sub)
		}
		sub := _sub(parser, pos05, pos, node.Kids[nkids4:])
		node.Kids = append(node.Kids[:nkids4], sub)
	}
	for {
		nkids0 := len(node.Kids)
		pos1 := pos
		// (IdentC (Binary/Unary/Primary))
		{
			nkids16 := len(node.Kids)
			pos017 := pos
			// IdentC (Binary/Unary/Primary)
			// IdentC
			if !_node(parser, _IdentCNode, node, &pos) {
				goto fail3
			}
			// (Binary/Unary/Primary)
			{
				nkids19 := len(node.Kids)
				pos020 := pos
				// Binary/Unary/Primary
				{
					pos24 := pos
					nkids22 := len(node.Kids)
					// Binary
					if !_node(parser, _BinaryNode, node, &pos) {
						goto fail25
					}
					goto ok21
				fail25:
					node.Kids = node.Kids[:nkids22]
					pos = pos24
					// Unary
					if !_node(parser, _UnaryNode, node, &pos) {
						goto fail26
					}
					goto ok21
				fail26:
					node.Kids = node.Kids[:nkids22]
					pos = pos24
					// Primary
					if !_node(parser, _PrimaryNode, node, &pos) {
						goto fail27
					}
					goto ok21
				fail27:
					node.Kids = node.Kids[:nkids22]
					pos = pos24
					goto fail3
				ok21:
				}
				sub := _sub(parser, pos020, pos, node.Kids[nkids19:])
				node.Kids = append(node.Kids[:nkids19], sub)
			}
			sub := _sub(parser, pos017, pos, node.Kids[nkids16:])
			node.Kids = append(node.Kids[:nkids16], sub)
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
fail:
	return -1, nil
}

func _NaryMsgFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _NaryMsg, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "NaryMsg",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _NaryMsg}
	// (IdentC (Binary/Unary/Primary))+
	// (IdentC (Binary/Unary/Primary))
	// IdentC (Binary/Unary/Primary)
	// IdentC
	if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
		goto fail
	}
	// (Binary/Unary/Primary)
	// Binary/Unary/Primary
	{
		pos8 := pos
		// Binary
		if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
			goto fail9
		}
		goto ok5
	fail9:
		pos = pos8
		// Unary
		if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
			goto fail10
		}
		goto ok5
	fail10:
		pos = pos8
		// Primary
		if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
			goto fail11
		}
		goto ok5
	fail11:
		pos = pos8
		goto fail
	ok5:
	}
	for {
		pos1 := pos
		// (IdentC (Binary/Unary/Primary))
		// IdentC (Binary/Unary/Primary)
		// IdentC
		if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
			goto fail3
		}
		// (Binary/Unary/Primary)
		// Binary/Unary/Primary
		{
			pos16 := pos
			// Binary
			if !_fail(parser, _BinaryFail, errPos, failure, &pos) {
				goto fail17
			}
			goto ok13
		fail17:
			pos = pos16
			// Unary
			if !_fail(parser, _UnaryFail, errPos, failure, &pos) {
				goto fail18
			}
			goto ok13
		fail18:
			pos = pos16
			// Primary
			if !_fail(parser, _PrimaryFail, errPos, failure, &pos) {
				goto fail19
			}
			goto ok13
		fail19:
			pos = pos16
			goto fail3
		ok13:
		}
		continue
	fail3:
		pos = pos1
		break
	}
	parser.fail[key] = failure
	return pos, failure
fail:
	parser.fail[key] = failure
	return -1, failure
}

func _NaryMsgAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_NaryMsg]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _NaryMsg}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// (IdentC (Binary/Unary/Primary))+
	{
		var node2 string
		// (IdentC (Binary/Unary/Primary))
		// IdentC (Binary/Unary/Primary)
		{
			var node4 string
			// IdentC
			if p, n := _IdentCAction(parser, pos); n == nil {
				goto fail
			} else {
				node4 = *n
				pos = p
			}
			node2, node4 = node2+node4, ""
			// (Binary/Unary/Primary)
			// Binary/Unary/Primary
			{
				pos8 := pos
				var node7 string
				// Binary
				if p, n := _BinaryAction(parser, pos); n == nil {
					goto fail9
				} else {
					node4 = *n
					pos = p
				}
				goto ok5
			fail9:
				node4 = node7
				pos = pos8
				// Unary
				if p, n := _UnaryAction(parser, pos); n == nil {
					goto fail10
				} else {
					node4 = *n
					pos = p
				}
				goto ok5
			fail10:
				node4 = node7
				pos = pos8
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail11
				} else {
					node4 = *n
					pos = p
				}
				goto ok5
			fail11:
				node4 = node7
				pos = pos8
				goto fail
			ok5:
			}
			node2, node4 = node2+node4, ""
		}
		node += node2
	}
	for {
		pos1 := pos
		var node2 string
		// (IdentC (Binary/Unary/Primary))
		// IdentC (Binary/Unary/Primary)
		{
			var node12 string
			// IdentC
			if p, n := _IdentCAction(parser, pos); n == nil {
				goto fail3
			} else {
				node12 = *n
				pos = p
			}
			node2, node12 = node2+node12, ""
			// (Binary/Unary/Primary)
			// Binary/Unary/Primary
			{
				pos16 := pos
				var node15 string
				// Binary
				if p, n := _BinaryAction(parser, pos); n == nil {
					goto fail17
				} else {
					node12 = *n
					pos = p
				}
				goto ok13
			fail17:
				node12 = node15
				pos = pos16
				// Unary
				if p, n := _UnaryAction(parser, pos); n == nil {
					goto fail18
				} else {
					node12 = *n
					pos = p
				}
				goto ok13
			fail18:
				node12 = node15
				pos = pos16
				// Primary
				if p, n := _PrimaryAction(parser, pos); n == nil {
					goto fail19
				} else {
					node12 = *n
					pos = p
				}
				goto ok13
			fail19:
				node12 = node15
				pos = pos16
				goto fail3
			ok13:
			}
			node2, node12 = node2+node12, ""
		}
		node += node2
		continue
	fail3:
		pos = pos1
		break
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _PrimaryAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Primary, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// Ident/Int/Float/Rune/String/Ctor/Block/_ "(" Expr _ ")"
	{
		pos3 := pos
		// Ident
		if !_accept(parser, _IdentAccepts, &pos, &perr) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Int
		if !_accept(parser, _IntAccepts, &pos, &perr) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Float
		if !_accept(parser, _FloatAccepts, &pos, &perr) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		// Rune
		if !_accept(parser, _RuneAccepts, &pos, &perr) {
			goto fail7
		}
		goto ok0
	fail7:
		pos = pos3
		// String
		if !_accept(parser, _StringAccepts, &pos, &perr) {
			goto fail8
		}
		goto ok0
	fail8:
		pos = pos3
		// Ctor
		if !_accept(parser, _CtorAccepts, &pos, &perr) {
			goto fail9
		}
		goto ok0
	fail9:
		pos = pos3
		// Block
		if !_accept(parser, _BlockAccepts, &pos, &perr) {
			goto fail10
		}
		goto ok0
	fail10:
		pos = pos3
		// _ "(" Expr _ ")"
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail11
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			perr = _max(perr, pos)
			goto fail11
		}
		pos++
		// Expr
		if !_accept(parser, _ExprAccepts, &pos, &perr) {
			goto fail11
		}
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail11
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			perr = _max(perr, pos)
			goto fail11
		}
		pos++
		goto ok0
	fail11:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Primary, start, pos, perr)
fail:
	return _memoize(parser, _Primary, start, -1, perr)
}

func _PrimaryNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// Ident/Int/Float/Rune/String/Ctor/Block/_ "(" Expr _ ")"
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// Ident
		if !_node(parser, _IdentNode, node, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Int
		if !_node(parser, _IntNode, node, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Float
		if !_node(parser, _FloatNode, node, &pos) {
			goto fail6
		}
		goto ok0
	fail6:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Rune
		if !_node(parser, _RuneNode, node, &pos) {
			goto fail7
		}
		goto ok0
	fail7:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// String
		if !_node(parser, _StringNode, node, &pos) {
			goto fail8
		}
		goto ok0
	fail8:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Ctor
		if !_node(parser, _CtorNode, node, &pos) {
			goto fail9
		}
		goto ok0
	fail9:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// Block
		if !_node(parser, _BlockNode, node, &pos) {
			goto fail10
		}
		goto ok0
	fail10:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// _ "(" Expr _ ")"
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail11
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			goto fail11
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// Expr
		if !_node(parser, _ExprNode, node, &pos) {
			goto fail11
		}
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail11
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			goto fail11
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		goto ok0
	fail11:
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
	pos, failure := _failMemo(parser, _Primary, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Primary",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Primary}
	// Ident/Int/Float/Rune/String/Ctor/Block/_ "(" Expr _ ")"
	{
		pos3 := pos
		// Ident
		if !_fail(parser, _IdentFail, errPos, failure, &pos) {
			goto fail4
		}
		goto ok0
	fail4:
		pos = pos3
		// Int
		if !_fail(parser, _IntFail, errPos, failure, &pos) {
			goto fail5
		}
		goto ok0
	fail5:
		pos = pos3
		// Float
		if !_fail(parser, _FloatFail, errPos, failure, &pos) {
			goto fail6
		}
		goto ok0
	fail6:
		pos = pos3
		// Rune
		if !_fail(parser, _RuneFail, errPos, failure, &pos) {
			goto fail7
		}
		goto ok0
	fail7:
		pos = pos3
		// String
		if !_fail(parser, _StringFail, errPos, failure, &pos) {
			goto fail8
		}
		goto ok0
	fail8:
		pos = pos3
		// Ctor
		if !_fail(parser, _CtorFail, errPos, failure, &pos) {
			goto fail9
		}
		goto ok0
	fail9:
		pos = pos3
		// Block
		if !_fail(parser, _BlockFail, errPos, failure, &pos) {
			goto fail10
		}
		goto ok0
	fail10:
		pos = pos3
		// _ "(" Expr _ ")"
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail11
		}
		// "("
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"(\"",
				})
			}
			goto fail11
		}
		pos++
		// Expr
		if !_fail(parser, _ExprFail, errPos, failure, &pos) {
			goto fail11
		}
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail11
		}
		// ")"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\")\"",
				})
			}
			goto fail11
		}
		pos++
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

func _PrimaryAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Primary]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Primary}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// Ident/Int/Float/Rune/String/Ctor/Block/_ "(" Expr _ ")"
	{
		pos3 := pos
		var node2 string
		// Ident
		if p, n := _IdentAction(parser, pos); n == nil {
			goto fail4
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// Int
		if p, n := _IntAction(parser, pos); n == nil {
			goto fail5
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail5:
		node = node2
		pos = pos3
		// Float
		if p, n := _FloatAction(parser, pos); n == nil {
			goto fail6
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail6:
		node = node2
		pos = pos3
		// Rune
		if p, n := _RuneAction(parser, pos); n == nil {
			goto fail7
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail7:
		node = node2
		pos = pos3
		// String
		if p, n := _StringAction(parser, pos); n == nil {
			goto fail8
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail8:
		node = node2
		pos = pos3
		// Ctor
		if p, n := _CtorAction(parser, pos); n == nil {
			goto fail9
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail9:
		node = node2
		pos = pos3
		// Block
		if p, n := _BlockAction(parser, pos); n == nil {
			goto fail10
		} else {
			node = *n
			pos = p
		}
		goto ok0
	fail10:
		node = node2
		pos = pos3
		// _ "(" Expr _ ")"
		{
			var node12 string
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail11
			} else {
				node12 = *n
				pos = p
			}
			node, node12 = node+node12, ""
			// "("
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "(" {
				goto fail11
			}
			node12 = parser.text[pos : pos+1]
			pos++
			node, node12 = node+node12, ""
			// Expr
			if p, n := _ExprAction(parser, pos); n == nil {
				goto fail11
			} else {
				node12 = *n
				pos = p
			}
			node, node12 = node+node12, ""
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail11
			} else {
				node12 = *n
				pos = p
			}
			node, node12 = node+node12, ""
			// ")"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ")" {
				goto fail11
			}
			node12 = parser.text[pos : pos+1]
			pos++
			node, node12 = node+node12, ""
		}
		goto ok0
	fail11:
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
	if dp, de, ok := _memo(parser, _Ctor, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ "{" TypeName _ "|" ((IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?) _ "}"
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
	// TypeName
	if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
		goto fail
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
	// ((IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?)
	// (IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?
	{
		pos4 := pos
		// (IdentC Expr)+
		// (IdentC Expr)
		// IdentC Expr
		// IdentC
		if !_accept(parser, _IdentCAccepts, &pos, &perr) {
			goto fail5
		}
		// Expr
		if !_accept(parser, _ExprAccepts, &pos, &perr) {
			goto fail5
		}
		for {
			pos7 := pos
			// (IdentC Expr)
			// IdentC Expr
			// IdentC
			if !_accept(parser, _IdentCAccepts, &pos, &perr) {
				goto fail9
			}
			// Expr
			if !_accept(parser, _ExprAccepts, &pos, &perr) {
				goto fail9
			}
			continue
		fail9:
			pos = pos7
			break
		}
		goto ok1
	fail5:
		pos = pos4
		// (Expr (_ ";" Expr)* (_ ";")?)?
		{
			pos14 := pos
			// (Expr (_ ";" Expr)* (_ ";")?)
			// Expr (_ ";" Expr)* (_ ";")?
			// Expr
			if !_accept(parser, _ExprAccepts, &pos, &perr) {
				goto fail15
			}
			// (_ ";" Expr)*
			for {
				pos18 := pos
				// (_ ";" Expr)
				// _ ";" Expr
				// _
				if !_accept(parser, __Accepts, &pos, &perr) {
					goto fail20
				}
				// ";"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
					perr = _max(perr, pos)
					goto fail20
				}
				pos++
				// Expr
				if !_accept(parser, _ExprAccepts, &pos, &perr) {
					goto fail20
				}
				continue
			fail20:
				pos = pos18
				break
			}
			// (_ ";")?
			{
				pos23 := pos
				// (_ ";")
				// _ ";"
				// _
				if !_accept(parser, __Accepts, &pos, &perr) {
					goto fail24
				}
				// ";"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
					perr = _max(perr, pos)
					goto fail24
				}
				pos++
				goto ok26
			fail24:
				pos = pos23
			ok26:
			}
			goto ok27
		fail15:
			pos = pos14
		ok27:
		}
	ok1:
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
	// _ "{" TypeName _ "|" ((IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?) _ "}"
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// "{"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
		goto fail
	}
	node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
	pos++
	// TypeName
	if !_node(parser, _TypeNameNode, node, &pos) {
		goto fail
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
	// ((IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?)
	{
		nkids1 := len(node.Kids)
		pos02 := pos
		// (IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?
		{
			pos6 := pos
			nkids4 := len(node.Kids)
			// (IdentC Expr)+
			// (IdentC Expr)
			{
				nkids12 := len(node.Kids)
				pos013 := pos
				// IdentC Expr
				// IdentC
				if !_node(parser, _IdentCNode, node, &pos) {
					goto fail7
				}
				// Expr
				if !_node(parser, _ExprNode, node, &pos) {
					goto fail7
				}
				sub := _sub(parser, pos013, pos, node.Kids[nkids12:])
				node.Kids = append(node.Kids[:nkids12], sub)
			}
			for {
				nkids8 := len(node.Kids)
				pos9 := pos
				// (IdentC Expr)
				{
					nkids15 := len(node.Kids)
					pos016 := pos
					// IdentC Expr
					// IdentC
					if !_node(parser, _IdentCNode, node, &pos) {
						goto fail11
					}
					// Expr
					if !_node(parser, _ExprNode, node, &pos) {
						goto fail11
					}
					sub := _sub(parser, pos016, pos, node.Kids[nkids15:])
					node.Kids = append(node.Kids[:nkids15], sub)
				}
				continue
			fail11:
				node.Kids = node.Kids[:nkids8]
				pos = pos9
				break
			}
			goto ok3
		fail7:
			node.Kids = node.Kids[:nkids4]
			pos = pos6
			// (Expr (_ ";" Expr)* (_ ";")?)?
			{
				nkids19 := len(node.Kids)
				pos20 := pos
				// (Expr (_ ";" Expr)* (_ ";")?)
				{
					nkids22 := len(node.Kids)
					pos023 := pos
					// Expr (_ ";" Expr)* (_ ";")?
					// Expr
					if !_node(parser, _ExprNode, node, &pos) {
						goto fail21
					}
					// (_ ";" Expr)*
					for {
						nkids25 := len(node.Kids)
						pos26 := pos
						// (_ ";" Expr)
						{
							nkids29 := len(node.Kids)
							pos030 := pos
							// _ ";" Expr
							// _
							if !_node(parser, __Node, node, &pos) {
								goto fail28
							}
							// ";"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
								goto fail28
							}
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
							pos++
							// Expr
							if !_node(parser, _ExprNode, node, &pos) {
								goto fail28
							}
							sub := _sub(parser, pos030, pos, node.Kids[nkids29:])
							node.Kids = append(node.Kids[:nkids29], sub)
						}
						continue
					fail28:
						node.Kids = node.Kids[:nkids25]
						pos = pos26
						break
					}
					// (_ ";")?
					{
						nkids32 := len(node.Kids)
						pos33 := pos
						// (_ ";")
						{
							nkids35 := len(node.Kids)
							pos036 := pos
							// _ ";"
							// _
							if !_node(parser, __Node, node, &pos) {
								goto fail34
							}
							// ";"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
								goto fail34
							}
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
							pos++
							sub := _sub(parser, pos036, pos, node.Kids[nkids35:])
							node.Kids = append(node.Kids[:nkids35], sub)
						}
						goto ok38
					fail34:
						node.Kids = node.Kids[:nkids32]
						pos = pos33
					ok38:
					}
					sub := _sub(parser, pos023, pos, node.Kids[nkids22:])
					node.Kids = append(node.Kids[:nkids22], sub)
				}
				goto ok39
			fail21:
				node.Kids = node.Kids[:nkids19]
				pos = pos20
			ok39:
			}
		ok3:
		}
		sub := _sub(parser, pos02, pos, node.Kids[nkids1:])
		node.Kids = append(node.Kids[:nkids1], sub)
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
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _CtorFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Ctor, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Ctor",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Ctor}
	// _ "{" TypeName _ "|" ((IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?) _ "}"
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
	// TypeName
	if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
		goto fail
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
	// ((IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?)
	// (IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?
	{
		pos4 := pos
		// (IdentC Expr)+
		// (IdentC Expr)
		// IdentC Expr
		// IdentC
		if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
			goto fail5
		}
		// Expr
		if !_fail(parser, _ExprFail, errPos, failure, &pos) {
			goto fail5
		}
		for {
			pos7 := pos
			// (IdentC Expr)
			// IdentC Expr
			// IdentC
			if !_fail(parser, _IdentCFail, errPos, failure, &pos) {
				goto fail9
			}
			// Expr
			if !_fail(parser, _ExprFail, errPos, failure, &pos) {
				goto fail9
			}
			continue
		fail9:
			pos = pos7
			break
		}
		goto ok1
	fail5:
		pos = pos4
		// (Expr (_ ";" Expr)* (_ ";")?)?
		{
			pos14 := pos
			// (Expr (_ ";" Expr)* (_ ";")?)
			// Expr (_ ";" Expr)* (_ ";")?
			// Expr
			if !_fail(parser, _ExprFail, errPos, failure, &pos) {
				goto fail15
			}
			// (_ ";" Expr)*
			for {
				pos18 := pos
				// (_ ";" Expr)
				// _ ";" Expr
				// _
				if !_fail(parser, __Fail, errPos, failure, &pos) {
					goto fail20
				}
				// ";"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\";\"",
						})
					}
					goto fail20
				}
				pos++
				// Expr
				if !_fail(parser, _ExprFail, errPos, failure, &pos) {
					goto fail20
				}
				continue
			fail20:
				pos = pos18
				break
			}
			// (_ ";")?
			{
				pos23 := pos
				// (_ ";")
				// _ ";"
				// _
				if !_fail(parser, __Fail, errPos, failure, &pos) {
					goto fail24
				}
				// ";"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\";\"",
						})
					}
					goto fail24
				}
				pos++
				goto ok26
			fail24:
				pos = pos23
			ok26:
			}
			goto ok27
		fail15:
			pos = pos14
		ok27:
		}
	ok1:
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

func _CtorAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Ctor]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ctor}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ "{" TypeName _ "|" ((IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?) _ "}"
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "{"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "{" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// TypeName
		if p, n := _TypeNameAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "|"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// ((IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?)
		// (IdentC Expr)+/(Expr (_ ";" Expr)* (_ ";")?)?
		{
			pos4 := pos
			var node3 string
			// (IdentC Expr)+
			{
				var node8 string
				// (IdentC Expr)
				// IdentC Expr
				{
					var node10 string
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail5
					} else {
						node10 = *n
						pos = p
					}
					node8, node10 = node8+node10, ""
					// Expr
					if p, n := _ExprAction(parser, pos); n == nil {
						goto fail5
					} else {
						node10 = *n
						pos = p
					}
					node8, node10 = node8+node10, ""
				}
				node0 += node8
			}
			for {
				pos7 := pos
				var node8 string
				// (IdentC Expr)
				// IdentC Expr
				{
					var node11 string
					// IdentC
					if p, n := _IdentCAction(parser, pos); n == nil {
						goto fail9
					} else {
						node11 = *n
						pos = p
					}
					node8, node11 = node8+node11, ""
					// Expr
					if p, n := _ExprAction(parser, pos); n == nil {
						goto fail9
					} else {
						node11 = *n
						pos = p
					}
					node8, node11 = node8+node11, ""
				}
				node0 += node8
				continue
			fail9:
				pos = pos7
				break
			}
			goto ok1
		fail5:
			node0 = node3
			pos = pos4
			// (Expr (_ ";" Expr)* (_ ";")?)?
			{
				pos14 := pos
				// (Expr (_ ";" Expr)* (_ ";")?)
				// Expr (_ ";" Expr)* (_ ";")?
				{
					var node16 string
					// Expr
					if p, n := _ExprAction(parser, pos); n == nil {
						goto fail15
					} else {
						node16 = *n
						pos = p
					}
					node0, node16 = node0+node16, ""
					// (_ ";" Expr)*
					for {
						pos18 := pos
						var node19 string
						// (_ ";" Expr)
						// _ ";" Expr
						{
							var node21 string
							// _
							if p, n := __Action(parser, pos); n == nil {
								goto fail20
							} else {
								node21 = *n
								pos = p
							}
							node19, node21 = node19+node21, ""
							// ";"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
								goto fail20
							}
							node21 = parser.text[pos : pos+1]
							pos++
							node19, node21 = node19+node21, ""
							// Expr
							if p, n := _ExprAction(parser, pos); n == nil {
								goto fail20
							} else {
								node21 = *n
								pos = p
							}
							node19, node21 = node19+node21, ""
						}
						node16 += node19
						continue
					fail20:
						pos = pos18
						break
					}
					node0, node16 = node0+node16, ""
					// (_ ";")?
					{
						pos23 := pos
						// (_ ";")
						// _ ";"
						{
							var node25 string
							// _
							if p, n := __Action(parser, pos); n == nil {
								goto fail24
							} else {
								node25 = *n
								pos = p
							}
							node16, node25 = node16+node25, ""
							// ";"
							if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ";" {
								goto fail24
							}
							node25 = parser.text[pos : pos+1]
							pos++
							node16, node25 = node16+node25, ""
						}
						goto ok26
					fail24:
						node16 = ""
						pos = pos23
					ok26:
					}
					node0, node16 = node0+node16, ""
				}
				goto ok27
			fail15:
				node0 = ""
				pos = pos14
			ok27:
			}
		ok1:
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "}"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "}" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _BlockAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Block, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ "[" ((CIdent TypeName?)+ _ "|")? Stmts _ "]"
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
	// ((CIdent TypeName?)+ _ "|")?
	{
		pos2 := pos
		// ((CIdent TypeName?)+ _ "|")
		// (CIdent TypeName?)+ _ "|"
		// (CIdent TypeName?)+
		// (CIdent TypeName?)
		// CIdent TypeName?
		// CIdent
		if !_accept(parser, _CIdentAccepts, &pos, &perr) {
			goto fail3
		}
		// TypeName?
		{
			pos11 := pos
			// TypeName
			if !_accept(parser, _TypeNameAccepts, &pos, &perr) {
				goto fail12
			}
			goto ok13
		fail12:
			pos = pos11
		ok13:
		}
		for {
			pos6 := pos
			// (CIdent TypeName?)
			// CIdent TypeName?
			// CIdent
			if !_accept(parser, _CIdentAccepts, &pos, &perr) {
				goto fail8
			}
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
			continue
		fail8:
			pos = pos6
			break
		}
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
		goto ok19
	fail3:
		pos = pos2
	ok19:
	}
	// Stmts
	if !_accept(parser, _StmtsAccepts, &pos, &perr) {
		goto fail
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
	// _ "[" ((CIdent TypeName?)+ _ "|")? Stmts _ "]"
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
	// ((CIdent TypeName?)+ _ "|")?
	{
		nkids1 := len(node.Kids)
		pos2 := pos
		// ((CIdent TypeName?)+ _ "|")
		{
			nkids4 := len(node.Kids)
			pos05 := pos
			// (CIdent TypeName?)+ _ "|"
			// (CIdent TypeName?)+
			// (CIdent TypeName?)
			{
				nkids11 := len(node.Kids)
				pos012 := pos
				// CIdent TypeName?
				// CIdent
				if !_node(parser, _CIdentNode, node, &pos) {
					goto fail3
				}
				// TypeName?
				{
					nkids14 := len(node.Kids)
					pos15 := pos
					// TypeName
					if !_node(parser, _TypeNameNode, node, &pos) {
						goto fail16
					}
					goto ok17
				fail16:
					node.Kids = node.Kids[:nkids14]
					pos = pos15
				ok17:
				}
				sub := _sub(parser, pos012, pos, node.Kids[nkids11:])
				node.Kids = append(node.Kids[:nkids11], sub)
			}
			for {
				nkids7 := len(node.Kids)
				pos8 := pos
				// (CIdent TypeName?)
				{
					nkids18 := len(node.Kids)
					pos019 := pos
					// CIdent TypeName?
					// CIdent
					if !_node(parser, _CIdentNode, node, &pos) {
						goto fail10
					}
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
					sub := _sub(parser, pos019, pos, node.Kids[nkids18:])
					node.Kids = append(node.Kids[:nkids18], sub)
				}
				continue
			fail10:
				node.Kids = node.Kids[:nkids7]
				pos = pos8
				break
			}
			// _
			if !_node(parser, __Node, node, &pos) {
				goto fail3
			}
			// "|"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
				goto fail3
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
			pos++
			sub := _sub(parser, pos05, pos, node.Kids[nkids4:])
			node.Kids = append(node.Kids[:nkids4], sub)
		}
		goto ok25
	fail3:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
	ok25:
	}
	// Stmts
	if !_node(parser, _StmtsNode, node, &pos) {
		goto fail
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

func _BlockFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Block, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Block",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Block}
	// _ "[" ((CIdent TypeName?)+ _ "|")? Stmts _ "]"
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
	// ((CIdent TypeName?)+ _ "|")?
	{
		pos2 := pos
		// ((CIdent TypeName?)+ _ "|")
		// (CIdent TypeName?)+ _ "|"
		// (CIdent TypeName?)+
		// (CIdent TypeName?)
		// CIdent TypeName?
		// CIdent
		if !_fail(parser, _CIdentFail, errPos, failure, &pos) {
			goto fail3
		}
		// TypeName?
		{
			pos11 := pos
			// TypeName
			if !_fail(parser, _TypeNameFail, errPos, failure, &pos) {
				goto fail12
			}
			goto ok13
		fail12:
			pos = pos11
		ok13:
		}
		for {
			pos6 := pos
			// (CIdent TypeName?)
			// CIdent TypeName?
			// CIdent
			if !_fail(parser, _CIdentFail, errPos, failure, &pos) {
				goto fail8
			}
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
			continue
		fail8:
			pos = pos6
			break
		}
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
		goto ok19
	fail3:
		pos = pos2
	ok19:
	}
	// Stmts
	if !_fail(parser, _StmtsFail, errPos, failure, &pos) {
		goto fail
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

func _BlockAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Block]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Block}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ "[" ((CIdent TypeName?)+ _ "|")? Stmts _ "]"
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "["
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "[" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// ((CIdent TypeName?)+ _ "|")?
		{
			pos2 := pos
			// ((CIdent TypeName?)+ _ "|")
			// (CIdent TypeName?)+ _ "|"
			{
				var node4 string
				// (CIdent TypeName?)+
				{
					var node7 string
					// (CIdent TypeName?)
					// CIdent TypeName?
					{
						var node9 string
						// CIdent
						if p, n := _CIdentAction(parser, pos); n == nil {
							goto fail3
						} else {
							node9 = *n
							pos = p
						}
						node7, node9 = node7+node9, ""
						// TypeName?
						{
							pos11 := pos
							// TypeName
							if p, n := _TypeNameAction(parser, pos); n == nil {
								goto fail12
							} else {
								node9 = *n
								pos = p
							}
							goto ok13
						fail12:
							node9 = ""
							pos = pos11
						ok13:
						}
						node7, node9 = node7+node9, ""
					}
					node4 += node7
				}
				for {
					pos6 := pos
					var node7 string
					// (CIdent TypeName?)
					// CIdent TypeName?
					{
						var node14 string
						// CIdent
						if p, n := _CIdentAction(parser, pos); n == nil {
							goto fail8
						} else {
							node14 = *n
							pos = p
						}
						node7, node14 = node7+node14, ""
						// TypeName?
						{
							pos16 := pos
							// TypeName
							if p, n := _TypeNameAction(parser, pos); n == nil {
								goto fail17
							} else {
								node14 = *n
								pos = p
							}
							goto ok18
						fail17:
							node14 = ""
							pos = pos16
						ok18:
						}
						node7, node14 = node7+node14, ""
					}
					node4 += node7
					continue
				fail8:
					pos = pos6
					break
				}
				node0, node4 = node0+node4, ""
				// _
				if p, n := __Action(parser, pos); n == nil {
					goto fail3
				} else {
					node4 = *n
					pos = p
				}
				node0, node4 = node0+node4, ""
				// "|"
				if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "|" {
					goto fail3
				}
				node4 = parser.text[pos : pos+1]
				pos++
				node0, node4 = node0+node4, ""
			}
			goto ok19
		fail3:
			node0 = ""
			pos = pos2
		ok19:
		}
		node, node0 = node+node0, ""
		// Stmts
		if p, n := _StmtsAction(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "]"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "]" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _IntAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Int, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ [+\-]? [0-9]+
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// [+\-]?
	{
		pos2 := pos
		// [+\-]
		if r, w := _next(parser, pos); r != '+' && r != '-' {
			perr = _max(perr, pos)
			goto fail3
		} else {
			pos += w
		}
		goto ok4
	fail3:
		pos = pos2
	ok4:
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
		pos6 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			perr = _max(perr, pos)
			goto fail8
		} else {
			pos += w
		}
		continue
	fail8:
		pos = pos6
		break
	}
	perr = start
	return _memoize(parser, _Int, start, pos, perr)
fail:
	return _memoize(parser, _Int, start, -1, perr)
}

func _IntNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ [+\-]? [0-9]+
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// [+\-]?
	{
		nkids1 := len(node.Kids)
		pos2 := pos
		// [+\-]
		if r, w := _next(parser, pos); r != '+' && r != '-' {
			goto fail3
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		goto ok4
	fail3:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
	ok4:
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
		nkids5 := len(node.Kids)
		pos6 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			goto fail8
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		continue
	fail8:
		node.Kids = node.Kids[:nkids5]
		pos = pos6
		break
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _IntFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Int, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Int",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Int}
	// _ [+\-]? [0-9]+
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// [+\-]?
	{
		pos2 := pos
		// [+\-]
		if r, w := _next(parser, pos); r != '+' && r != '-' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[+\\-]",
				})
			}
			goto fail3
		} else {
			pos += w
		}
		goto ok4
	fail3:
		pos = pos2
	ok4:
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
		pos6 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[0-9]",
				})
			}
			goto fail8
		} else {
			pos += w
		}
		continue
	fail8:
		pos = pos6
		break
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

func _IntAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Int]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Int}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ [+\-]? [0-9]+
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// [+\-]?
		{
			pos2 := pos
			// [+\-]
			if r, w := _next(parser, pos); r != '+' && r != '-' {
				goto fail3
			} else {
				node0 = parser.text[pos : pos+w]
				pos += w
			}
			goto ok4
		fail3:
			node0 = ""
			pos = pos2
		ok4:
		}
		node, node0 = node+node0, ""
		// [0-9]+
		{
			var node7 string
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				goto fail
			} else {
				node7 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node7
		}
		for {
			pos6 := pos
			var node7 string
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				goto fail8
			} else {
				node7 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node7
			continue
		fail8:
			pos = pos6
			break
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _FloatAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Float, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ [+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// [+\-]?
	{
		pos2 := pos
		// [+\-]
		if r, w := _next(parser, pos); r != '+' && r != '-' {
			perr = _max(perr, pos)
			goto fail3
		} else {
			pos += w
		}
		goto ok4
	fail3:
		pos = pos2
	ok4:
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
		pos6 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			perr = _max(perr, pos)
			goto fail8
		} else {
			pos += w
		}
		continue
	fail8:
		pos = pos6
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
		pos10 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			perr = _max(perr, pos)
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
	}
	// ([eE] [+\-]? [0-9]+)?
	{
		pos14 := pos
		// ([eE] [+\-]? [0-9]+)
		// [eE] [+\-]? [0-9]+
		// [eE]
		if r, w := _next(parser, pos); r != 'e' && r != 'E' {
			perr = _max(perr, pos)
			goto fail15
		} else {
			pos += w
		}
		// [+\-]?
		{
			pos18 := pos
			// [+\-]
			if r, w := _next(parser, pos); r != '+' && r != '-' {
				perr = _max(perr, pos)
				goto fail19
			} else {
				pos += w
			}
			goto ok20
		fail19:
			pos = pos18
		ok20:
		}
		// [0-9]+
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			perr = _max(perr, pos)
			goto fail15
		} else {
			pos += w
		}
		for {
			pos22 := pos
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				perr = _max(perr, pos)
				goto fail24
			} else {
				pos += w
			}
			continue
		fail24:
			pos = pos22
			break
		}
		goto ok25
	fail15:
		pos = pos14
	ok25:
	}
	perr = start
	return _memoize(parser, _Float, start, pos, perr)
fail:
	return _memoize(parser, _Float, start, -1, perr)
}

func _FloatNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ [+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// [+\-]?
	{
		nkids1 := len(node.Kids)
		pos2 := pos
		// [+\-]
		if r, w := _next(parser, pos); r != '+' && r != '-' {
			goto fail3
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		goto ok4
	fail3:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
	ok4:
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
		nkids5 := len(node.Kids)
		pos6 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			goto fail8
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		continue
	fail8:
		node.Kids = node.Kids[:nkids5]
		pos = pos6
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
		nkids9 := len(node.Kids)
		pos10 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			goto fail12
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		continue
	fail12:
		node.Kids = node.Kids[:nkids9]
		pos = pos10
		break
	}
	// ([eE] [+\-]? [0-9]+)?
	{
		nkids13 := len(node.Kids)
		pos14 := pos
		// ([eE] [+\-]? [0-9]+)
		{
			nkids16 := len(node.Kids)
			pos017 := pos
			// [eE] [+\-]? [0-9]+
			// [eE]
			if r, w := _next(parser, pos); r != 'e' && r != 'E' {
				goto fail15
			} else {
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
				pos += w
			}
			// [+\-]?
			{
				nkids19 := len(node.Kids)
				pos20 := pos
				// [+\-]
				if r, w := _next(parser, pos); r != '+' && r != '-' {
					goto fail21
				} else {
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
					pos += w
				}
				goto ok22
			fail21:
				node.Kids = node.Kids[:nkids19]
				pos = pos20
			ok22:
			}
			// [0-9]+
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				goto fail15
			} else {
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
				pos += w
			}
			for {
				nkids23 := len(node.Kids)
				pos24 := pos
				// [0-9]
				if r, w := _next(parser, pos); r < '0' || r > '9' {
					goto fail26
				} else {
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
					pos += w
				}
				continue
			fail26:
				node.Kids = node.Kids[:nkids23]
				pos = pos24
				break
			}
			sub := _sub(parser, pos017, pos, node.Kids[nkids16:])
			node.Kids = append(node.Kids[:nkids16], sub)
		}
		goto ok27
	fail15:
		node.Kids = node.Kids[:nkids13]
		pos = pos14
	ok27:
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _FloatFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Float, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Float",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Float}
	// _ [+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
	// [+\-]?
	{
		pos2 := pos
		// [+\-]
		if r, w := _next(parser, pos); r != '+' && r != '-' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[+\\-]",
				})
			}
			goto fail3
		} else {
			pos += w
		}
		goto ok4
	fail3:
		pos = pos2
	ok4:
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
		pos6 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[0-9]",
				})
			}
			goto fail8
		} else {
			pos += w
		}
		continue
	fail8:
		pos = pos6
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
		pos10 := pos
		// [0-9]
		if r, w := _next(parser, pos); r < '0' || r > '9' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[0-9]",
				})
			}
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
	}
	// ([eE] [+\-]? [0-9]+)?
	{
		pos14 := pos
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
			goto fail15
		} else {
			pos += w
		}
		// [+\-]?
		{
			pos18 := pos
			// [+\-]
			if r, w := _next(parser, pos); r != '+' && r != '-' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[+\\-]",
					})
				}
				goto fail19
			} else {
				pos += w
			}
			goto ok20
		fail19:
			pos = pos18
		ok20:
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
			goto fail15
		} else {
			pos += w
		}
		for {
			pos22 := pos
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				if pos >= errPos {
					failure.Kids = append(failure.Kids, &peg.Fail{
						Pos:  int(pos),
						Want: "[0-9]",
					})
				}
				goto fail24
			} else {
				pos += w
			}
			continue
		fail24:
			pos = pos22
			break
		}
		goto ok25
	fail15:
		pos = pos14
	ok25:
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

func _FloatAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Float]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Float}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ [+\-]? [0-9]+ "." [0-9]+ ([eE] [+\-]? [0-9]+)?
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// [+\-]?
		{
			pos2 := pos
			// [+\-]
			if r, w := _next(parser, pos); r != '+' && r != '-' {
				goto fail3
			} else {
				node0 = parser.text[pos : pos+w]
				pos += w
			}
			goto ok4
		fail3:
			node0 = ""
			pos = pos2
		ok4:
		}
		node, node0 = node+node0, ""
		// [0-9]+
		{
			var node7 string
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				goto fail
			} else {
				node7 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node7
		}
		for {
			pos6 := pos
			var node7 string
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				goto fail8
			} else {
				node7 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node7
			continue
		fail8:
			pos = pos6
			break
		}
		node, node0 = node+node0, ""
		// "."
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "." {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// [0-9]+
		{
			var node11 string
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				goto fail
			} else {
				node11 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node11
		}
		for {
			pos10 := pos
			var node11 string
			// [0-9]
			if r, w := _next(parser, pos); r < '0' || r > '9' {
				goto fail12
			} else {
				node11 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node11
			continue
		fail12:
			pos = pos10
			break
		}
		node, node0 = node+node0, ""
		// ([eE] [+\-]? [0-9]+)?
		{
			pos14 := pos
			// ([eE] [+\-]? [0-9]+)
			// [eE] [+\-]? [0-9]+
			{
				var node16 string
				// [eE]
				if r, w := _next(parser, pos); r != 'e' && r != 'E' {
					goto fail15
				} else {
					node16 = parser.text[pos : pos+w]
					pos += w
				}
				node0, node16 = node0+node16, ""
				// [+\-]?
				{
					pos18 := pos
					// [+\-]
					if r, w := _next(parser, pos); r != '+' && r != '-' {
						goto fail19
					} else {
						node16 = parser.text[pos : pos+w]
						pos += w
					}
					goto ok20
				fail19:
					node16 = ""
					pos = pos18
				ok20:
				}
				node0, node16 = node0+node16, ""
				// [0-9]+
				{
					var node23 string
					// [0-9]
					if r, w := _next(parser, pos); r < '0' || r > '9' {
						goto fail15
					} else {
						node23 = parser.text[pos : pos+w]
						pos += w
					}
					node16 += node23
				}
				for {
					pos22 := pos
					var node23 string
					// [0-9]
					if r, w := _next(parser, pos); r < '0' || r > '9' {
						goto fail24
					} else {
						node23 = parser.text[pos : pos+w]
						pos += w
					}
					node16 += node23
					continue
				fail24:
					pos = pos22
					break
				}
				node0, node16 = node0+node16, ""
			}
			goto ok25
		fail15:
			node0 = ""
			pos = pos14
		ok25:
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _RuneAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Rune, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ [\'] !"\n" (Esc/"\\'"/[^\']) [\']
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// [\']
	if r, w := _next(parser, pos); r != '\'' {
		perr = _max(perr, pos)
		goto fail
	} else {
		pos += w
	}
	// !"\n"
	{
		pos2 := pos
		perr4 := perr
		// "\n"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
			perr = _max(perr, pos)
			goto ok1
		}
		pos++
		pos = pos2
		perr = _max(perr4, pos)
		goto fail
	ok1:
		pos = pos2
		perr = perr4
	}
	// (Esc/"\\'"/[^\'])
	// Esc/"\\'"/[^\']
	{
		pos8 := pos
		// Esc
		if !_accept(parser, _EscAccepts, &pos, &perr) {
			goto fail9
		}
		goto ok5
	fail9:
		pos = pos8
		// "\\'"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\'" {
			perr = _max(perr, pos)
			goto fail10
		}
		pos += 2
		goto ok5
	fail10:
		pos = pos8
		// [^\']
		if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '\'' {
			perr = _max(perr, pos)
			goto fail11
		} else {
			pos += w
		}
		goto ok5
	fail11:
		pos = pos8
		goto fail
	ok5:
	}
	// [\']
	if r, w := _next(parser, pos); r != '\'' {
		perr = _max(perr, pos)
		goto fail
	} else {
		pos += w
	}
	perr = start
	return _memoize(parser, _Rune, start, pos, perr)
fail:
	return _memoize(parser, _Rune, start, -1, perr)
}

func _RuneNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ [\'] !"\n" (Esc/"\\'"/[^\']) [\']
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// [\']
	if r, w := _next(parser, pos); r != '\'' {
		goto fail
	} else {
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
		pos += w
	}
	// !"\n"
	{
		pos2 := pos
		nkids3 := len(node.Kids)
		// "\n"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
			goto ok1
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		pos = pos2
		node.Kids = node.Kids[:nkids3]
		goto fail
	ok1:
		pos = pos2
		node.Kids = node.Kids[:nkids3]
	}
	// (Esc/"\\'"/[^\'])
	{
		nkids5 := len(node.Kids)
		pos06 := pos
		// Esc/"\\'"/[^\']
		{
			pos10 := pos
			nkids8 := len(node.Kids)
			// Esc
			if !_node(parser, _EscNode, node, &pos) {
				goto fail11
			}
			goto ok7
		fail11:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			// "\\'"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\'" {
				goto fail12
			}
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
			pos += 2
			goto ok7
		fail12:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			// [^\']
			if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '\'' {
				goto fail13
			} else {
				node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
				pos += w
			}
			goto ok7
		fail13:
			node.Kids = node.Kids[:nkids8]
			pos = pos10
			goto fail
		ok7:
		}
		sub := _sub(parser, pos06, pos, node.Kids[nkids5:])
		node.Kids = append(node.Kids[:nkids5], sub)
	}
	// [\']
	if r, w := _next(parser, pos); r != '\'' {
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

func _RuneFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Rune, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Rune",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Rune}
	// _ [\'] !"\n" (Esc/"\\'"/[^\']) [\']
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
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
	// !"\n"
	{
		pos2 := pos
		nkids3 := len(failure.Kids)
		// "\n"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\n\"",
				})
			}
			goto ok1
		}
		pos++
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!\"\\n\"",
			})
		}
		goto fail
	ok1:
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
	}
	// (Esc/"\\'"/[^\'])
	// Esc/"\\'"/[^\']
	{
		pos8 := pos
		// Esc
		if !_fail(parser, _EscFail, errPos, failure, &pos) {
			goto fail9
		}
		goto ok5
	fail9:
		pos = pos8
		// "\\'"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\'" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\'\"",
				})
			}
			goto fail10
		}
		pos += 2
		goto ok5
	fail10:
		pos = pos8
		// [^\']
		if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '\'' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[^\\']",
				})
			}
			goto fail11
		} else {
			pos += w
		}
		goto ok5
	fail11:
		pos = pos8
		goto fail
	ok5:
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
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "rune"
	parser.fail[key] = failure
	return -1, failure
}

func _RuneAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Rune]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Rune}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ [\'] !"\n" (Esc/"\\'"/[^\']) [\']
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// [\']
		if r, w := _next(parser, pos); r != '\'' {
			goto fail
		} else {
			node0 = parser.text[pos : pos+w]
			pos += w
		}
		node, node0 = node+node0, ""
		// !"\n"
		{
			pos2 := pos
			// "\n"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\n" {
				goto ok1
			}
			pos++
			pos = pos2
			goto fail
		ok1:
			pos = pos2
			node0 = ""
		}
		node, node0 = node+node0, ""
		// (Esc/"\\'"/[^\'])
		// Esc/"\\'"/[^\']
		{
			pos8 := pos
			var node7 string
			// Esc
			if p, n := _EscAction(parser, pos); n == nil {
				goto fail9
			} else {
				node0 = *n
				pos = p
			}
			goto ok5
		fail9:
			node0 = node7
			pos = pos8
			// "\\'"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\'" {
				goto fail10
			}
			node0 = parser.text[pos : pos+2]
			pos += 2
			goto ok5
		fail10:
			node0 = node7
			pos = pos8
			// [^\']
			if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '\'' {
				goto fail11
			} else {
				node0 = parser.text[pos : pos+w]
				pos += w
			}
			goto ok5
		fail11:
			node0 = node7
			pos = pos8
			goto fail
		ok5:
		}
		node, node0 = node+node0, ""
		// [\']
		if r, w := _next(parser, pos); r != '\'' {
			goto fail
		} else {
			node0 = parser.text[pos : pos+w]
			pos += w
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _StringAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _String, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ ["] (!"\n" (Esc/"\\\""/[^"]))* ["]/_ [`] ("\\`"/[^`])* [`]
	{
		pos3 := pos
		// _ ["] (!"\n" (Esc/"\\\""/[^"]))* ["]
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail4
		}
		// ["]
		if r, w := _next(parser, pos); r != '"' {
			perr = _max(perr, pos)
			goto fail4
		} else {
			pos += w
		}
		// (!"\n" (Esc/"\\\""/[^"]))*
		for {
			pos7 := pos
			// (!"\n" (Esc/"\\\""/[^"]))
			// !"\n" (Esc/"\\\""/[^"])
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
			// (Esc/"\\\""/[^"])
			// Esc/"\\\""/[^"]
			{
				pos18 := pos
				// Esc
				if !_accept(parser, _EscAccepts, &pos, &perr) {
					goto fail19
				}
				goto ok15
			fail19:
				pos = pos18
				// "\\\""
				if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\"" {
					perr = _max(perr, pos)
					goto fail20
				}
				pos += 2
				goto ok15
			fail20:
				pos = pos18
				// [^"]
				if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '"' {
					perr = _max(perr, pos)
					goto fail21
				} else {
					pos += w
				}
				goto ok15
			fail21:
				pos = pos18
				goto fail9
			ok15:
			}
			continue
		fail9:
			pos = pos7
			break
		}
		// ["]
		if r, w := _next(parser, pos); r != '"' {
			perr = _max(perr, pos)
			goto fail4
		} else {
			pos += w
		}
		goto ok0
	fail4:
		pos = pos3
		// _ [`] ("\\`"/[^`])* [`]
		// _
		if !_accept(parser, __Accepts, &pos, &perr) {
			goto fail22
		}
		// [`]
		if r, w := _next(parser, pos); r != '`' {
			perr = _max(perr, pos)
			goto fail22
		} else {
			pos += w
		}
		// ("\\`"/[^`])*
		for {
			pos25 := pos
			// ("\\`"/[^`])
			// "\\`"/[^`]
			{
				pos31 := pos
				// "\\`"
				if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\`" {
					perr = _max(perr, pos)
					goto fail32
				}
				pos += 2
				goto ok28
			fail32:
				pos = pos31
				// [^`]
				if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '`' {
					perr = _max(perr, pos)
					goto fail33
				} else {
					pos += w
				}
				goto ok28
			fail33:
				pos = pos31
				goto fail27
			ok28:
			}
			continue
		fail27:
			pos = pos25
			break
		}
		// [`]
		if r, w := _next(parser, pos); r != '`' {
			perr = _max(perr, pos)
			goto fail22
		} else {
			pos += w
		}
		goto ok0
	fail22:
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
	// _ ["] (!"\n" (Esc/"\\\""/[^"]))* ["]/_ [`] ("\\`"/[^`])* [`]
	{
		pos3 := pos
		nkids1 := len(node.Kids)
		// _ ["] (!"\n" (Esc/"\\\""/[^"]))* ["]
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail4
		}
		// ["]
		if r, w := _next(parser, pos); r != '"' {
			goto fail4
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		// (!"\n" (Esc/"\\\""/[^"]))*
		for {
			nkids6 := len(node.Kids)
			pos7 := pos
			// (!"\n" (Esc/"\\\""/[^"]))
			{
				nkids10 := len(node.Kids)
				pos011 := pos
				// !"\n" (Esc/"\\\""/[^"])
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
				// (Esc/"\\\""/[^"])
				{
					nkids17 := len(node.Kids)
					pos018 := pos
					// Esc/"\\\""/[^"]
					{
						pos22 := pos
						nkids20 := len(node.Kids)
						// Esc
						if !_node(parser, _EscNode, node, &pos) {
							goto fail23
						}
						goto ok19
					fail23:
						node.Kids = node.Kids[:nkids20]
						pos = pos22
						// "\\\""
						if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\"" {
							goto fail24
						}
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
						pos += 2
						goto ok19
					fail24:
						node.Kids = node.Kids[:nkids20]
						pos = pos22
						// [^"]
						if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '"' {
							goto fail25
						} else {
							node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
							pos += w
						}
						goto ok19
					fail25:
						node.Kids = node.Kids[:nkids20]
						pos = pos22
						goto fail9
					ok19:
					}
					sub := _sub(parser, pos018, pos, node.Kids[nkids17:])
					node.Kids = append(node.Kids[:nkids17], sub)
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
		// ["]
		if r, w := _next(parser, pos); r != '"' {
			goto fail4
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		goto ok0
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// _ [`] ("\\`"/[^`])* [`]
		// _
		if !_node(parser, __Node, node, &pos) {
			goto fail26
		}
		// [`]
		if r, w := _next(parser, pos); r != '`' {
			goto fail26
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		// ("\\`"/[^`])*
		for {
			nkids28 := len(node.Kids)
			pos29 := pos
			// ("\\`"/[^`])
			{
				nkids32 := len(node.Kids)
				pos033 := pos
				// "\\`"/[^`]
				{
					pos37 := pos
					nkids35 := len(node.Kids)
					// "\\`"
					if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\`" {
						goto fail38
					}
					node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
					pos += 2
					goto ok34
				fail38:
					node.Kids = node.Kids[:nkids35]
					pos = pos37
					// [^`]
					if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '`' {
						goto fail39
					} else {
						node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
						pos += w
					}
					goto ok34
				fail39:
					node.Kids = node.Kids[:nkids35]
					pos = pos37
					goto fail31
				ok34:
				}
				sub := _sub(parser, pos033, pos, node.Kids[nkids32:])
				node.Kids = append(node.Kids[:nkids32], sub)
			}
			continue
		fail31:
			node.Kids = node.Kids[:nkids28]
			pos = pos29
			break
		}
		// [`]
		if r, w := _next(parser, pos); r != '`' {
			goto fail26
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		goto ok0
	fail26:
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
	pos, failure := _failMemo(parser, _String, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "String",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _String}
	// _ ["] (!"\n" (Esc/"\\\""/[^"]))* ["]/_ [`] ("\\`"/[^`])* [`]
	{
		pos3 := pos
		// _ ["] (!"\n" (Esc/"\\\""/[^"]))* ["]
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail4
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
		// (!"\n" (Esc/"\\\""/[^"]))*
		for {
			pos7 := pos
			// (!"\n" (Esc/"\\\""/[^"]))
			// !"\n" (Esc/"\\\""/[^"])
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
			// (Esc/"\\\""/[^"])
			// Esc/"\\\""/[^"]
			{
				pos18 := pos
				// Esc
				if !_fail(parser, _EscFail, errPos, failure, &pos) {
					goto fail19
				}
				goto ok15
			fail19:
				pos = pos18
				// "\\\""
				if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\"" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"\\\\\\\"\"",
						})
					}
					goto fail20
				}
				pos += 2
				goto ok15
			fail20:
				pos = pos18
				// [^"]
				if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '"' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[^\"]",
						})
					}
					goto fail21
				} else {
					pos += w
				}
				goto ok15
			fail21:
				pos = pos18
				goto fail9
			ok15:
			}
			continue
		fail9:
			pos = pos7
			break
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
		goto ok0
	fail4:
		pos = pos3
		// _ [`] ("\\`"/[^`])* [`]
		// _
		if !_fail(parser, __Fail, errPos, failure, &pos) {
			goto fail22
		}
		// [`]
		if r, w := _next(parser, pos); r != '`' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[`]",
				})
			}
			goto fail22
		} else {
			pos += w
		}
		// ("\\`"/[^`])*
		for {
			pos25 := pos
			// ("\\`"/[^`])
			// "\\`"/[^`]
			{
				pos31 := pos
				// "\\`"
				if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\`" {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "\"\\\\`\"",
						})
					}
					goto fail32
				}
				pos += 2
				goto ok28
			fail32:
				pos = pos31
				// [^`]
				if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '`' {
					if pos >= errPos {
						failure.Kids = append(failure.Kids, &peg.Fail{
							Pos:  int(pos),
							Want: "[^`]",
						})
					}
					goto fail33
				} else {
					pos += w
				}
				goto ok28
			fail33:
				pos = pos31
				goto fail27
			ok28:
			}
			continue
		fail27:
			pos = pos25
			break
		}
		// [`]
		if r, w := _next(parser, pos); r != '`' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[`]",
				})
			}
			goto fail22
		} else {
			pos += w
		}
		goto ok0
	fail22:
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

func _StringAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_String]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _String}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ ["] (!"\n" (Esc/"\\\""/[^"]))* ["]/_ [`] ("\\`"/[^`])* [`]
	{
		pos3 := pos
		var node2 string
		// _ ["] (!"\n" (Esc/"\\\""/[^"]))* ["]
		{
			var node5 string
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail4
			} else {
				node5 = *n
				pos = p
			}
			node, node5 = node+node5, ""
			// ["]
			if r, w := _next(parser, pos); r != '"' {
				goto fail4
			} else {
				node5 = parser.text[pos : pos+w]
				pos += w
			}
			node, node5 = node+node5, ""
			// (!"\n" (Esc/"\\\""/[^"]))*
			for {
				pos7 := pos
				var node8 string
				// (!"\n" (Esc/"\\\""/[^"]))
				// !"\n" (Esc/"\\\""/[^"])
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
					// (Esc/"\\\""/[^"])
					// Esc/"\\\""/[^"]
					{
						pos18 := pos
						var node17 string
						// Esc
						if p, n := _EscAction(parser, pos); n == nil {
							goto fail19
						} else {
							node10 = *n
							pos = p
						}
						goto ok15
					fail19:
						node10 = node17
						pos = pos18
						// "\\\""
						if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\"" {
							goto fail20
						}
						node10 = parser.text[pos : pos+2]
						pos += 2
						goto ok15
					fail20:
						node10 = node17
						pos = pos18
						// [^"]
						if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '"' {
							goto fail21
						} else {
							node10 = parser.text[pos : pos+w]
							pos += w
						}
						goto ok15
					fail21:
						node10 = node17
						pos = pos18
						goto fail9
					ok15:
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
			// ["]
			if r, w := _next(parser, pos); r != '"' {
				goto fail4
			} else {
				node5 = parser.text[pos : pos+w]
				pos += w
			}
			node, node5 = node+node5, ""
		}
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// _ [`] ("\\`"/[^`])* [`]
		{
			var node23 string
			// _
			if p, n := __Action(parser, pos); n == nil {
				goto fail22
			} else {
				node23 = *n
				pos = p
			}
			node, node23 = node+node23, ""
			// [`]
			if r, w := _next(parser, pos); r != '`' {
				goto fail22
			} else {
				node23 = parser.text[pos : pos+w]
				pos += w
			}
			node, node23 = node+node23, ""
			// ("\\`"/[^`])*
			for {
				pos25 := pos
				var node26 string
				// ("\\`"/[^`])
				// "\\`"/[^`]
				{
					pos31 := pos
					var node30 string
					// "\\`"
					if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\`" {
						goto fail32
					}
					node26 = parser.text[pos : pos+2]
					pos += 2
					goto ok28
				fail32:
					node26 = node30
					pos = pos31
					// [^`]
					if r, w := _next(parser, pos); w == 0 || r == '\uFFFD' || r == '`' {
						goto fail33
					} else {
						node26 = parser.text[pos : pos+w]
						pos += w
					}
					goto ok28
				fail33:
					node26 = node30
					pos = pos31
					goto fail27
				ok28:
				}
				node23 += node26
				continue
			fail27:
				pos = pos25
				break
			}
			node, node23 = node+node23, ""
			// [`]
			if r, w := _next(parser, pos); r != '`' {
				goto fail22
			} else {
				node23 = parser.text[pos : pos+w]
				pos += w
			}
			node, node23 = node+node23, ""
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

func _EscAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Esc, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// "\\n"/"\\t"/"\\b"/"\\\\"/"\\" X X/"\\x" X X X X/"\\X" X X X X X X X X
	{
		pos3 := pos
		// "\\n"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\n" {
			perr = _max(perr, pos)
			goto fail4
		}
		pos += 2
		goto ok0
	fail4:
		pos = pos3
		// "\\t"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\t" {
			perr = _max(perr, pos)
			goto fail5
		}
		pos += 2
		goto ok0
	fail5:
		pos = pos3
		// "\\b"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\b" {
			perr = _max(perr, pos)
			goto fail6
		}
		pos += 2
		goto ok0
	fail6:
		pos = pos3
		// "\\\\"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\\" {
			perr = _max(perr, pos)
			goto fail7
		}
		pos += 2
		goto ok0
	fail7:
		pos = pos3
		// "\\" X X
		// "\\"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\\" {
			perr = _max(perr, pos)
			goto fail8
		}
		pos++
		// X
		if !_accept(parser, _XAccepts, &pos, &perr) {
			goto fail8
		}
		// X
		if !_accept(parser, _XAccepts, &pos, &perr) {
			goto fail8
		}
		goto ok0
	fail8:
		pos = pos3
		// "\\x" X X X X
		// "\\x"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\x" {
			perr = _max(perr, pos)
			goto fail10
		}
		pos += 2
		// X
		if !_accept(parser, _XAccepts, &pos, &perr) {
			goto fail10
		}
		// X
		if !_accept(parser, _XAccepts, &pos, &perr) {
			goto fail10
		}
		// X
		if !_accept(parser, _XAccepts, &pos, &perr) {
			goto fail10
		}
		// X
		if !_accept(parser, _XAccepts, &pos, &perr) {
			goto fail10
		}
		goto ok0
	fail10:
		pos = pos3
		// "\\X" X X X X X X X X
		// "\\X"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\X" {
			perr = _max(perr, pos)
			goto fail12
		}
		pos += 2
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
		goto ok0
	fail12:
		pos = pos3
		goto fail
	ok0:
	}
	return _memoize(parser, _Esc, start, pos, perr)
fail:
	return _memoize(parser, _Esc, start, -1, perr)
}

func _EscNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// "\\n"/"\\t"/"\\b"/"\\\\"/"\\" X X/"\\x" X X X X/"\\X" X X X X X X X X
	{
		pos3 := pos
		nkids1 := len(node.Kids)
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
		// "\\" X X
		// "\\"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\\" {
			goto fail8
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail8
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail8
		}
		goto ok0
	fail8:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// "\\x" X X X X
		// "\\x"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\x" {
			goto fail10
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail10
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail10
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail10
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail10
		}
		goto ok0
	fail10:
		node.Kids = node.Kids[:nkids1]
		pos = pos3
		// "\\X" X X X X X X X X
		// "\\X"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\X" {
			goto fail12
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+2))
		pos += 2
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail12
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail12
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail12
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail12
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail12
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail12
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail12
		}
		// X
		if !_node(parser, _XNode, node, &pos) {
			goto fail12
		}
		goto ok0
	fail12:
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
	pos, failure := _failMemo(parser, _Esc, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Esc",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Esc}
	// "\\n"/"\\t"/"\\b"/"\\\\"/"\\" X X/"\\x" X X X X/"\\X" X X X X X X X X
	{
		pos3 := pos
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
		// "\\" X X
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
		// X
		if !_fail(parser, _XFail, errPos, failure, &pos) {
			goto fail8
		}
		// X
		if !_fail(parser, _XFail, errPos, failure, &pos) {
			goto fail8
		}
		goto ok0
	fail8:
		pos = pos3
		// "\\x" X X X X
		// "\\x"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\x" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\x\"",
				})
			}
			goto fail10
		}
		pos += 2
		// X
		if !_fail(parser, _XFail, errPos, failure, &pos) {
			goto fail10
		}
		// X
		if !_fail(parser, _XFail, errPos, failure, &pos) {
			goto fail10
		}
		// X
		if !_fail(parser, _XFail, errPos, failure, &pos) {
			goto fail10
		}
		// X
		if !_fail(parser, _XFail, errPos, failure, &pos) {
			goto fail10
		}
		goto ok0
	fail10:
		pos = pos3
		// "\\X" X X X X X X X X
		// "\\X"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\X" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\"\\\\X\"",
				})
			}
			goto fail12
		}
		pos += 2
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
		goto ok0
	fail12:
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
	// "\\n"/"\\t"/"\\b"/"\\\\"/"\\" X X/"\\x" X X X X/"\\X" X X X X X X X X
	{
		pos3 := pos
		var node2 string
		// "\\n"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\n" {
			goto fail4
		}
		node = parser.text[pos : pos+2]
		pos += 2
		goto ok0
	fail4:
		node = node2
		pos = pos3
		// "\\t"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\t" {
			goto fail5
		}
		node = parser.text[pos : pos+2]
		pos += 2
		goto ok0
	fail5:
		node = node2
		pos = pos3
		// "\\b"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\b" {
			goto fail6
		}
		node = parser.text[pos : pos+2]
		pos += 2
		goto ok0
	fail6:
		node = node2
		pos = pos3
		// "\\\\"
		if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\\\" {
			goto fail7
		}
		node = parser.text[pos : pos+2]
		pos += 2
		goto ok0
	fail7:
		node = node2
		pos = pos3
		// "\\" X X
		{
			var node9 string
			// "\\"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "\\" {
				goto fail8
			}
			node9 = parser.text[pos : pos+1]
			pos++
			node, node9 = node+node9, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail8
			} else {
				node9 = *n
				pos = p
			}
			node, node9 = node+node9, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail8
			} else {
				node9 = *n
				pos = p
			}
			node, node9 = node+node9, ""
		}
		goto ok0
	fail8:
		node = node2
		pos = pos3
		// "\\x" X X X X
		{
			var node11 string
			// "\\x"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\x" {
				goto fail10
			}
			node11 = parser.text[pos : pos+2]
			pos += 2
			node, node11 = node+node11, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail10
			} else {
				node11 = *n
				pos = p
			}
			node, node11 = node+node11, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail10
			} else {
				node11 = *n
				pos = p
			}
			node, node11 = node+node11, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail10
			} else {
				node11 = *n
				pos = p
			}
			node, node11 = node+node11, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail10
			} else {
				node11 = *n
				pos = p
			}
			node, node11 = node+node11, ""
		}
		goto ok0
	fail10:
		node = node2
		pos = pos3
		// "\\X" X X X X X X X X
		{
			var node13 string
			// "\\X"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "\\X" {
				goto fail12
			}
			node13 = parser.text[pos : pos+2]
			pos += 2
			node, node13 = node+node13, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail12
			} else {
				node13 = *n
				pos = p
			}
			node, node13 = node+node13, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail12
			} else {
				node13 = *n
				pos = p
			}
			node, node13 = node+node13, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail12
			} else {
				node13 = *n
				pos = p
			}
			node, node13 = node+node13, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail12
			} else {
				node13 = *n
				pos = p
			}
			node, node13 = node+node13, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail12
			} else {
				node13 = *n
				pos = p
			}
			node, node13 = node+node13, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail12
			} else {
				node13 = *n
				pos = p
			}
			node, node13 = node+node13, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail12
			} else {
				node13 = *n
				pos = p
			}
			node, node13 = node+node13, ""
			// X
			if p, n := _XAction(parser, pos); n == nil {
				goto fail12
			} else {
				node13 = *n
				pos = p
			}
			node, node13 = node+node13, ""
		}
		goto ok0
	fail12:
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
	if dp, de, ok := _memo(parser, _Op, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ !"//" !"/*" [!%&*+\-/<=>?@\\|~]+
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
	// [!%&*+\-/<=>?@\\|~]+
	// [!%&*+\-/<=>?@\\|~]
	if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
		perr = _max(perr, pos)
		goto fail
	} else {
		pos += w
	}
	for {
		pos10 := pos
		// [!%&*+\-/<=>?@\\|~]
		if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
			perr = _max(perr, pos)
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
	}
	perr = start
	return _memoize(parser, _Op, start, pos, perr)
fail:
	return _memoize(parser, _Op, start, -1, perr)
}

func _OpNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ !"//" !"/*" [!%&*+\-/<=>?@\\|~]+
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
	// [!%&*+\-/<=>?@\\|~]+
	// [!%&*+\-/<=>?@\\|~]
	if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
		goto fail
	} else {
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
		pos += w
	}
	for {
		nkids9 := len(node.Kids)
		pos10 := pos
		// [!%&*+\-/<=>?@\\|~]
		if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
			goto fail12
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		continue
	fail12:
		node.Kids = node.Kids[:nkids9]
		pos = pos10
		break
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _OpFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Op, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Op",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Op}
	// _ !"//" !"/*" [!%&*+\-/<=>?@\\|~]+
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
		pos10 := pos
		// [!%&*+\-/<=>?@\\|~]
		if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[!%&*+\\-/<=>?@\\\\|~]",
				})
			}
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
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

func _OpAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Op]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Op}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ !"//" !"/*" [!%&*+\-/<=>?@\\|~]+
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// !"//"
		{
			pos2 := pos
			// "//"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "//" {
				goto ok1
			}
			pos += 2
			pos = pos2
			goto fail
		ok1:
			pos = pos2
			node0 = ""
		}
		node, node0 = node+node0, ""
		// !"/*"
		{
			pos6 := pos
			// "/*"
			if len(parser.text[pos:]) < 2 || parser.text[pos:pos+2] != "/*" {
				goto ok5
			}
			pos += 2
			pos = pos6
			goto fail
		ok5:
			pos = pos6
			node0 = ""
		}
		node, node0 = node+node0, ""
		// [!%&*+\-/<=>?@\\|~]+
		{
			var node11 string
			// [!%&*+\-/<=>?@\\|~]
			if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
				goto fail
			} else {
				node11 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node11
		}
		for {
			pos10 := pos
			var node11 string
			// [!%&*+\-/<=>?@\\|~]
			if r, w := _next(parser, pos); r != '!' && r != '%' && r != '&' && r != '*' && r != '+' && r != '-' && r != '/' && r != '<' && r != '=' && r != '>' && r != '?' && r != '@' && r != '\\' && r != '|' && r != '~' {
				goto fail12
			} else {
				node11 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node11
			continue
		fail12:
			pos = pos10
			break
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _ModNameAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _ModName, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ "#" [_a-zA-Z] [_a-zA-Z0-9]*
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
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
		pos2 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			perr = _max(perr, pos)
			goto fail4
		} else {
			pos += w
		}
		continue
	fail4:
		pos = pos2
		break
	}
	perr = start
	return _memoize(parser, _ModName, start, pos, perr)
fail:
	return _memoize(parser, _ModName, start, -1, perr)
}

func _ModNameNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ "#" [_a-zA-Z] [_a-zA-Z0-9]*
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
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
		nkids1 := len(node.Kids)
		pos2 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			goto fail4
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		continue
	fail4:
		node.Kids = node.Kids[:nkids1]
		pos = pos2
		break
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _ModNameFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _ModName, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "ModName",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _ModName}
	// _ "#" [_a-zA-Z] [_a-zA-Z0-9]*
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
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
		pos2 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[_a-zA-Z0-9]",
				})
			}
			goto fail4
		} else {
			pos += w
		}
		continue
	fail4:
		pos = pos2
		break
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

func _ModNameAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_ModName]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _ModName}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ "#" [_a-zA-Z] [_a-zA-Z0-9]*
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// "#"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != "#" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// [_a-zA-Z]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			goto fail
		} else {
			node0 = parser.text[pos : pos+w]
			pos += w
		}
		node, node0 = node+node0, ""
		// [_a-zA-Z0-9]*
		for {
			pos2 := pos
			var node3 string
			// [_a-zA-Z0-9]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				goto fail4
			} else {
				node3 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node3
			continue
		fail4:
			pos = pos2
			break
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _IdentCAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _IdentC, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]* ":"
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
	// [_a-zA-Z]
	if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
		perr = _max(perr, pos)
		goto fail
	} else {
		pos += w
	}
	// [_a-zA-Z0-9]*
	for {
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			perr = _max(perr, pos)
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
	}
	// ":"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
		perr = _max(perr, pos)
		goto fail
	}
	pos++
	perr = start
	return _memoize(parser, _IdentC, start, pos, perr)
fail:
	return _memoize(parser, _IdentC, start, -1, perr)
}

func _IdentCNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]* ":"
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
	// [_a-zA-Z]
	if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
		goto fail
	} else {
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
		pos += w
	}
	// [_a-zA-Z0-9]*
	for {
		nkids9 := len(node.Kids)
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			goto fail12
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		continue
	fail12:
		node.Kids = node.Kids[:nkids9]
		pos = pos10
		break
	}
	// ":"
	if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
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

func _IdentCFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _IdentC, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "IdentC",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _IdentC}
	// _ !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]* ":"
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
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[_a-zA-Z0-9]",
				})
			}
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
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
	failure.Kids = nil
	parser.fail[key] = failure
	return pos, failure
fail:
	failure.Kids = nil
	failure.Want = "identifier:"
	parser.fail[key] = failure
	return -1, failure
}

func _IdentCAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_IdentC]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _IdentC}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]* ":"
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// !TypeVar
		{
			pos2 := pos
			// TypeVar
			if p, n := _TypeVarAction(parser, pos); n == nil {
				goto ok1
			} else {
				pos = p
			}
			pos = pos2
			goto fail
		ok1:
			pos = pos2
			node0 = ""
		}
		node, node0 = node+node0, ""
		// !"import"
		{
			pos6 := pos
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				goto ok5
			}
			pos += 6
			pos = pos6
			goto fail
		ok5:
			pos = pos6
			node0 = ""
		}
		node, node0 = node+node0, ""
		// [_a-zA-Z]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			goto fail
		} else {
			node0 = parser.text[pos : pos+w]
			pos += w
		}
		node, node0 = node+node0, ""
		// [_a-zA-Z0-9]*
		for {
			pos10 := pos
			var node11 string
			// [_a-zA-Z0-9]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				goto fail12
			} else {
				node11 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node11
			continue
		fail12:
			pos = pos10
			break
		}
		node, node0 = node+node0, ""
		// ":"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _CIdentAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _CIdent, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ ":" !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]*
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
	// [_a-zA-Z]
	if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
		perr = _max(perr, pos)
		goto fail
	} else {
		pos += w
	}
	// [_a-zA-Z0-9]*
	for {
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			perr = _max(perr, pos)
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
	}
	perr = start
	return _memoize(parser, _CIdent, start, pos, perr)
fail:
	return _memoize(parser, _CIdent, start, -1, perr)
}

func _CIdentNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ ":" !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]*
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
	// [_a-zA-Z]
	if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
		goto fail
	} else {
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
		pos += w
	}
	// [_a-zA-Z0-9]*
	for {
		nkids9 := len(node.Kids)
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			goto fail12
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		continue
	fail12:
		node.Kids = node.Kids[:nkids9]
		pos = pos10
		break
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _CIdentFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _CIdent, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "CIdent",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _CIdent}
	// _ ":" !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]*
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
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[_a-zA-Z0-9]",
				})
			}
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
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

func _CIdentAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_CIdent]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _CIdent}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ ":" !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]*
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// ":"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
			goto fail
		}
		node0 = parser.text[pos : pos+1]
		pos++
		node, node0 = node+node0, ""
		// !TypeVar
		{
			pos2 := pos
			// TypeVar
			if p, n := _TypeVarAction(parser, pos); n == nil {
				goto ok1
			} else {
				pos = p
			}
			pos = pos2
			goto fail
		ok1:
			pos = pos2
			node0 = ""
		}
		node, node0 = node+node0, ""
		// !"import"
		{
			pos6 := pos
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				goto ok5
			}
			pos += 6
			pos = pos6
			goto fail
		ok5:
			pos = pos6
			node0 = ""
		}
		node, node0 = node+node0, ""
		// [_a-zA-Z]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			goto fail
		} else {
			node0 = parser.text[pos : pos+w]
			pos += w
		}
		node, node0 = node+node0, ""
		// [_a-zA-Z0-9]*
		for {
			pos10 := pos
			var node11 string
			// [_a-zA-Z0-9]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				goto fail12
			} else {
				node11 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node11
			continue
		fail12:
			pos = pos10
			break
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _IdentAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _Ident, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]* !":"
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
	// [_a-zA-Z]
	if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
		perr = _max(perr, pos)
		goto fail
	} else {
		pos += w
	}
	// [_a-zA-Z0-9]*
	for {
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			perr = _max(perr, pos)
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
	}
	// !":"
	{
		pos14 := pos
		perr16 := perr
		// ":"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
			perr = _max(perr, pos)
			goto ok13
		}
		pos++
		pos = pos14
		perr = _max(perr16, pos)
		goto fail
	ok13:
		pos = pos14
		perr = perr16
	}
	perr = start
	return _memoize(parser, _Ident, start, pos, perr)
fail:
	return _memoize(parser, _Ident, start, -1, perr)
}

func _IdentNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]* !":"
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
	// [_a-zA-Z]
	if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
		goto fail
	} else {
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
		pos += w
	}
	// [_a-zA-Z0-9]*
	for {
		nkids9 := len(node.Kids)
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			goto fail12
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		continue
	fail12:
		node.Kids = node.Kids[:nkids9]
		pos = pos10
		break
	}
	// !":"
	{
		pos14 := pos
		nkids15 := len(node.Kids)
		// ":"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
			goto ok13
		}
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+1))
		pos++
		pos = pos14
		node.Kids = node.Kids[:nkids15]
		goto fail
	ok13:
		pos = pos14
		node.Kids = node.Kids[:nkids15]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _IdentFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _Ident, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "Ident",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _Ident}
	// _ !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]* !":"
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
		pos10 := pos
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[_a-zA-Z0-9]",
				})
			}
			goto fail12
		} else {
			pos += w
		}
		continue
	fail12:
		pos = pos10
		break
	}
	// !":"
	{
		pos14 := pos
		nkids15 := len(failure.Kids)
		// ":"
		if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "\":\"",
				})
			}
			goto ok13
		}
		pos++
		pos = pos14
		failure.Kids = failure.Kids[:nkids15]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "!\":\"",
			})
		}
		goto fail
	ok13:
		pos = pos14
		failure.Kids = failure.Kids[:nkids15]
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

func _IdentAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_Ident]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _Ident}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ !TypeVar !"import" [_a-zA-Z] [_a-zA-Z0-9]* !":"
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// !TypeVar
		{
			pos2 := pos
			// TypeVar
			if p, n := _TypeVarAction(parser, pos); n == nil {
				goto ok1
			} else {
				pos = p
			}
			pos = pos2
			goto fail
		ok1:
			pos = pos2
			node0 = ""
		}
		node, node0 = node+node0, ""
		// !"import"
		{
			pos6 := pos
			// "import"
			if len(parser.text[pos:]) < 6 || parser.text[pos:pos+6] != "import" {
				goto ok5
			}
			pos += 6
			pos = pos6
			goto fail
		ok5:
			pos = pos6
			node0 = ""
		}
		node, node0 = node+node0, ""
		// [_a-zA-Z]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			goto fail
		} else {
			node0 = parser.text[pos : pos+w]
			pos += w
		}
		node, node0 = node+node0, ""
		// [_a-zA-Z0-9]*
		for {
			pos10 := pos
			var node11 string
			// [_a-zA-Z0-9]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				goto fail12
			} else {
				node11 = parser.text[pos : pos+w]
				pos += w
			}
			node0 += node11
			continue
		fail12:
			pos = pos10
			break
		}
		node, node0 = node+node0, ""
		// !":"
		{
			pos14 := pos
			// ":"
			if len(parser.text[pos:]) < 1 || parser.text[pos:pos+1] != ":" {
				goto ok13
			}
			pos++
			pos = pos14
			goto fail
		ok13:
			pos = pos14
			node0 = ""
		}
		node, node0 = node+node0, ""
	}
	parser.act[key] = node
	return pos, &node
fail:
	return -1, nil
}

func _TypeVarAccepts(parser *_Parser, start int) (deltaPos, deltaErr int) {
	if dp, de, ok := _memo(parser, _TypeVar, start); ok {
		return dp, de
	}
	pos, perr := start, -1
	// _ [A-Z] ![_a-zA-Z0-9]
	// _
	if !_accept(parser, __Accepts, &pos, &perr) {
		goto fail
	}
	// [A-Z]
	if r, w := _next(parser, pos); r < 'A' || r > 'Z' {
		perr = _max(perr, pos)
		goto fail
	} else {
		pos += w
	}
	// ![_a-zA-Z0-9]
	{
		pos2 := pos
		perr4 := perr
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			perr = _max(perr, pos)
			goto ok1
		} else {
			pos += w
		}
		pos = pos2
		perr = _max(perr4, pos)
		goto fail
	ok1:
		pos = pos2
		perr = perr4
	}
	perr = start
	return _memoize(parser, _TypeVar, start, pos, perr)
fail:
	return _memoize(parser, _TypeVar, start, -1, perr)
}

func _TypeVarNode(parser *_Parser, start int) (int, *peg.Node) {
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
	// _ [A-Z] ![_a-zA-Z0-9]
	// _
	if !_node(parser, __Node, node, &pos) {
		goto fail
	}
	// [A-Z]
	if r, w := _next(parser, pos); r < 'A' || r > 'Z' {
		goto fail
	} else {
		node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
		pos += w
	}
	// ![_a-zA-Z0-9]
	{
		pos2 := pos
		nkids3 := len(node.Kids)
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			goto ok1
		} else {
			node.Kids = append(node.Kids, _leaf(parser, pos, pos+w))
			pos += w
		}
		pos = pos2
		node.Kids = node.Kids[:nkids3]
		goto fail
	ok1:
		pos = pos2
		node.Kids = node.Kids[:nkids3]
	}
	node.Text = parser.text[start:pos]
	parser.node[key] = node
	return pos, node
fail:
	return -1, nil
}

func _TypeVarFail(parser *_Parser, start, errPos int) (int, *peg.Fail) {
	pos, failure := _failMemo(parser, _TypeVar, start, errPos)
	if failure != nil {
		return pos, failure
	}
	failure = &peg.Fail{
		Name: "TypeVar",
		Pos:  int(start),
	}
	key := _key{start: start, rule: _TypeVar}
	// _ [A-Z] ![_a-zA-Z0-9]
	// _
	if !_fail(parser, __Fail, errPos, failure, &pos) {
		goto fail
	}
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
		pos2 := pos
		nkids3 := len(failure.Kids)
		// [_a-zA-Z0-9]
		if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			if pos >= errPos {
				failure.Kids = append(failure.Kids, &peg.Fail{
					Pos:  int(pos),
					Want: "[_a-zA-Z0-9]",
				})
			}
			goto ok1
		} else {
			pos += w
		}
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
		if pos >= errPos {
			failure.Kids = append(failure.Kids, &peg.Fail{
				Pos:  int(pos),
				Want: "![_a-zA-Z0-9]",
			})
		}
		goto fail
	ok1:
		pos = pos2
		failure.Kids = failure.Kids[:nkids3]
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

func _TypeVarAction(parser *_Parser, start int) (int, *string) {
	dp := parser.deltaPos[start][_TypeVar]
	if dp < 0 {
		return -1, nil
	}
	key := _key{start: start, rule: _TypeVar}
	n := parser.act[key]
	if n != nil {
		n := n.(string)
		return start + int(dp-1), &n
	}
	var node string
	pos := start
	// _ [A-Z] ![_a-zA-Z0-9]
	{
		var node0 string
		// _
		if p, n := __Action(parser, pos); n == nil {
			goto fail
		} else {
			node0 = *n
			pos = p
		}
		node, node0 = node+node0, ""
		// [A-Z]
		if r, w := _next(parser, pos); r < 'A' || r > 'Z' {
			goto fail
		} else {
			node0 = parser.text[pos : pos+w]
			pos += w
		}
		node, node0 = node+node0, ""
		// ![_a-zA-Z0-9]
		{
			pos2 := pos
			// [_a-zA-Z0-9]
			if r, w := _next(parser, pos); r != '_' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				goto ok1
			} else {
				pos += w
			}
			pos = pos2
			goto fail
		ok1:
			pos = pos2
			node0 = ""
		}
		node, node0 = node+node0, ""
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
