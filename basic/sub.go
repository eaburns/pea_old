package basic

type valMap []Val

func makeValMap(n int) valMap {
	return valMap(make([]Val, n))
}

func (s valMap) add(key, val Val) {
	s[key.Num()] = val
}

func (s valMap) get(v Val) Val {
	u := s[v.Num()]
	if u == nil {
		return v
	}
	u = s.get(u)
	s[v.Num()] = u
	return u
}

func subVals(bs []*BBlk, sub valMap) {
	for _, b := range bs {
		for _, s := range b.Stmts {
			s.subVals(sub)
			if c, ok := s.(*Copy); ok && c.Src == c.Dst {
				// Substitution made a self-copy. Delete it.
				c.delete()
			}
		}
	}
}

func (n *Store) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Val)
}

func (n *Copy) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Src)
}

func (n *MakeArray) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	for i := range n.Args {
		sub1(sub, n, &n.Args[i])
	}
}

func (n *MakeSlice) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Ary)
	sub1(sub, n, &n.From)
	sub1(sub, n, &n.To)
}

func (n *MakeString) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
}

func (n *MakeAnd) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	for i := range n.Fields {
		sub1(sub, n, &n.Fields[i])
	}
}

func (n *MakeOr) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	if n.Val != nil {
		sub1(sub, n, &n.Val)
	}
}

func (n *MakeVirt) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Obj)
}

func (n *Call) subVals(sub valMap) {
	for i := range n.Args {
		sub1(sub, n, &n.Args[i])
	}
}

func (n *VirtCall) subVals(sub valMap) {
	sub1(sub, n, &n.Self)
	for i := range n.Args {
		sub1(sub, n, &n.Args[i])
	}
}

func (*stmt) subVals(valMap) {}

func (n *Switch) subVals(sub valMap) {
	sub1(sub, n, &n.Val)
}

func (n *Op) subVals(sub valMap) {
	for i := range n.Args {
		sub1(sub, n, &n.Args[i])
	}
}

func (n *Load) subVals(sub valMap) {
	sub1(sub, n, &n.Src)
}

func (n *Index) subVals(sub valMap) {
	sub1(sub, n, &n.Ary)
	sub1(sub, n, &n.Index)
}

func (n *Field) subVals(sub valMap) {
	sub1(sub, n, &n.Obj)
}

func sub1(sub valMap, s Stmt, v *Val) {
	if u := sub.get(*v); *v != u {
		(*v).value().rmUser(s)
		u.value().addUser(s)
		*v = u
	}
}

type bblkMap []*BBlk

func makeBBlkMap(n int) bblkMap {
	return bblkMap(make([]*BBlk, n))
}

func (s bblkMap) add(key, val *BBlk) {
	s[key.N] = val
}

func (s bblkMap) get(v *BBlk) *BBlk {
	u := s[v.N]
	if u == nil {
		return v
	}
	u = s.get(u)
	s[v.N] = u
	return u
}

func subBBlks(bs []*BBlk, sub bblkMap) {
	for _, b := range bs {
		term := b.Stmts[len(b.Stmts)-1].(Term)
		for _, o := range term.Out() {
			o.rmIn(b)
		}
		term.subBBlk(sub)
		for _, o := range term.Out() {
			o.addIn(b)
		}
	}
}

func (*Ret) subBBlk(bblkMap) {}

func (n *Jmp) subBBlk(sub bblkMap) { n.Dst = sub.get(n.Dst) }

func (n *Switch) subBBlk(sub bblkMap) {
	for i := range n.Dsts {
		n.Dsts[i] = sub.get(n.Dsts[i])
	}
}
