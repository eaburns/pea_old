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

func subValues(bs []*BBlk, sub valMap) {
	for _, b := range bs {
		for _, s := range b.Stmts {
			s.sub(sub)
		}
	}
}

func (*Comment) sub(valMap) {}

func (n *Store) sub(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Val)
}

func (n *Copy) sub(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Src)
}

func (n *MakeArray) sub(sub valMap) {
	sub1(sub, n, &n.Dst)
	for i := range n.Args {
		sub1(sub, n, &n.Args[i])
	}
}

func (n *MakeSlice) sub(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Ary)
	sub1(sub, n, &n.From)
	sub1(sub, n, &n.To)
}

func (n *MakeString) sub(sub valMap) {
	sub1(sub, n, &n.Dst)
}

func (n *MakeAnd) sub(sub valMap) {
	sub1(sub, n, &n.Dst)
	for i := range n.Fields {
		sub1(sub, n, &n.Fields[i])
	}
}

func (n *MakeOr) sub(sub valMap) {
	sub1(sub, n, &n.Dst)
	if n.Val != nil {
		sub1(sub, n, &n.Val)
	}
}

func (n *MakeVirt) sub(sub valMap) {
	sub1(sub, n, &n.Dst)
	sub1(sub, n, &n.Obj)
}

func (n *Call) sub(sub valMap) {
	for i := range n.Args {
		sub1(sub, n, &n.Args[i])
	}
}

func (n *VirtCall) sub(sub valMap) {
	sub1(sub, n, &n.Self)
	for i := range n.Args {
		sub1(sub, n, &n.Args[i])
	}
}

func (*Ret) sub(valMap) {}

func (*Jmp) sub(valMap) {}

func (n *Switch) sub(sub valMap) {
	sub1(sub, n, &n.Val)
}

func (val) sub(valMap) {}

func (n *Op) sub(sub valMap) {
	for i := range n.Args {
		sub1(sub, n, &n.Args[i])
	}
}

func (n *Load) sub(sub valMap) {
	sub1(sub, n, &n.Src)
}

func (n *Index) sub(sub valMap) {
	sub1(sub, n, &n.Ary)
	sub1(sub, n, &n.Index)
}

func (n *Field) sub(sub valMap) {
	sub1(sub, n, &n.Obj)
}

func sub1(sub valMap, s Stmt, v *Val) {
	if u := sub.get(*v); *v != u {
		(*v).value().rmUser(s)
		u.value().addUser(s)
		*v = u
	}
}
