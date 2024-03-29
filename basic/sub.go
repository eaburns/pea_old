// Copyright © 2020 The Pea Authors under an MIT-style license.

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

func (*stmt) subVals(valMap) {}

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

func (n *NewArray) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Size)
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

func (n *NewString) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Data)
}

func (n *MakeAnd) subVals(sub valMap) {
	sub1(sub, n, &n.Dst)
	for i := range n.Fields {
		if n.Fields[i] != nil {
			sub1(sub, n, &n.Fields[i])
		}
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
	if n.Obj != nil {
		sub1(sub, n, &n.Obj)
	}
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

func (n *Panic) subVals(sub valMap) {
	sub1(sub, n, &n.Arg)
}

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
		if len(b.Stmts) == 0 {
			// The BBlk can have 0 statements during cleanup
			// after deleted statements have been removed.
			continue
		}
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

func (*Ret) subBBlk(bblkMap)   {}
func (*Panic) subBBlk(bblkMap) {}

func (n *Jmp) subBBlk(sub bblkMap) { n.Dst = sub.get(n.Dst) }

func (n *Switch) subBBlk(sub bblkMap) {
	for i := range n.Dsts {
		n.Dsts[i] = sub.get(n.Dsts[i])
	}
}
